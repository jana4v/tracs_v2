from __future__ import annotations

import asyncio
from datetime import datetime, timezone
from typing import Any, Iterable, cast

from src.InstrumentApi.Models import InstrumentTypes
from src.InstrumentApi.PowerMeter.BaseClass import PowerMeter
from src.InstrumentApi.SignalGenerator.BaseClass import SignalGenerator
from src.InstrumentApi.SpectrumAnalyzer.BaseClass import SpectrumAnalyzer
from src.InstrumentApi.factory import get_instrument
from src.services.switch_driver_paths import apply_switch_driver_paths, build_address, normalize_model

from .base import CalibrationDependencies, CalibrationProcedure


class DownlinkCalibrationProcedure(CalibrationProcedure):
    cal_type = "downlink"

    async def execute(self, service, runtime, payload, deps: CalibrationDependencies) -> None:
        cal_signal_generator: SignalGenerator | None = None
        inject_signal_generator: SignalGenerator | None = None
        try:
            grouped = self._build_grouped_frequency_map(payload.channels, deps, payload.include_spurious_bands)
            if len(grouped) == 0:
                await service.abort_runtime(runtime, "No valid frequencies found for Downlink calibration")
                return

            required_frequency_keys = {
                self._frequency_key(freq)
                for frequencies in grouped.values()
                for freq in frequencies
            }

            cal_id = str(runtime.cal_id or "").strip()
            cal_sg_rows = deps.cal_sg_repo.list_rows(cal_id)
            inject_rows = deps.inject_cal_repo.list_rows(cal_id)
            cal_sg_by_key = {self._frequency_key(float(row["frequency"])): float(row["value"]) for row in cal_sg_rows}
            inject_by_key = {
                self._frequency_key(float(row["frequency"])): {
                    "sa_loss": float(row["sa_loss"]),
                    "dl_pm_loss": float(row["dl_pm_loss"]),
                }
                for row in inject_rows
            }

            missing_cal_sg = sorted(required_frequency_keys - set(cal_sg_by_key.keys()))
            if missing_cal_sg:
                preview = ", ".join(missing_cal_sg[:5])
                suffix = "..." if len(missing_cal_sg) > 5 else ""
                message = f"Cal SG calibration to be completed before starting Downlink cal. Missing frequencies: {preview}{suffix}"
                await service.prompt_operator(
                    runtime,
                    message,
                    payload.channels[0] if payload.channels else None,
                )
                await service.abort_runtime(runtime, message)
                return

            missing_inject = sorted(required_frequency_keys - set(inject_by_key.keys()))
            if missing_inject:
                preview = ", ".join(missing_inject[:5])
                suffix = "..." if len(missing_inject) > 5 else ""
                message = f"Cal Inject calibration to be completed before starting Downlink cal. Missing frequencies: {preview}{suffix}"
                await service.prompt_operator(
                    runtime,
                    message,
                    payload.channels[0] if payload.channels else None,
                )
                await service.abort_runtime(runtime, message)
                return

            instruments = deps.test_systems_repo.get_project_instruments_rows()

            cal_sg_config = self._get_instrument_config(instruments, "CalSignalGenerator")
            inject_sg_config = self._get_instrument_config(instruments, "InjectSignalGenerator")
            sa_config = self._get_instrument_config(instruments, "SpectrumAnalyser")
            dl_pm_config = self._get_instrument_config(instruments, "DownlinkPowerMeter")

            power_meter_rows = deps.test_systems_repo.get_project_power_meter_rows()
            dl_pm_channel = self._get_power_meter_channel(power_meter_rows, "DownlinkPowerMeter")
            dl_pm_channel_number = 2 if dl_pm_channel == "B" else 1

            await service.push_status(
                runtime,
                f"Using CalSignalGenerator model={cal_sg_config.get('model', '')} address={cal_sg_config.get('address', '')}",
            )
            await service.push_status(
                runtime,
                f"Using InjectSignalGenerator model={inject_sg_config.get('model', '')} address={inject_sg_config.get('address', '')}",
            )
            await service.push_status(
                runtime,
                f"Using SpectrumAnalyser model={sa_config.get('model', '')} address={sa_config.get('address', '')}",
            )
            await service.push_status(
                runtime,
                f"Using DownlinkPowerMeter model={dl_pm_config.get('model', '')} address={dl_pm_config.get('address', '')}",
            )

            cal_signal_generator = self._build_signal_generator(cal_sg_config)
            inject_signal_generator = self._build_signal_generator(inject_sg_config)
            spectrum_analyzer = self._build_spectrum_analyzer(sa_config)
            downlink_power_meter = self._build_power_meter(dl_pm_config)

            await cal_signal_generator.preset_instrument()
            await cal_signal_generator.set_rf_on()

            await inject_signal_generator.preset_instrument()
            await inject_signal_generator.set_power_level(-80.0)
            await inject_signal_generator.set_rf_on()

            await downlink_power_meter.preset_instrument()
            await downlink_power_meter.set_channel_limits(dl_pm_channel_number, -50, 30)

            await spectrum_analyzer.preset_instrument()
            await spectrum_analyzer.set_resolution_bandwidth_auto_on_off("OFF")
            await spectrum_analyzer.set_video_bandwidth_auto_on_off("OFF")
            await spectrum_analyzer.set_resolution_bandwidth(1)
            await spectrum_analyzer.set_video_bandwidth(1)
            await spectrum_analyzer.set_span(5)

            await service.set_running(runtime, "Calibration run started")

            total_measurements = max(1, sum(len(v) for v in grouped.values()))
            completed = 0

            ordered_groups = sorted(
                grouped.items(),
                key=lambda item: (item[0][0], item[0][1]),
            )

            for (code, port), frequencies in ordered_groups:
                if runtime.abort_requested:
                    await service.abort_runtime(runtime, "Calibration aborted by user")
                    return

                prompt_channel = self._build_prompt_channel(payload.channels, code, port, frequencies)
                cable_name = f"{code}/{port}" if port else code
                if cable_name.strip() == "":
                    cable_name = "selected cable"
                await service.prompt_operator(
                    runtime,
                    f"Connect {cable_name} cable to CalSignalGenerator",
                    prompt_channel,
                )
                if runtime.abort_requested:
                    await service.abort_runtime(runtime, "Calibration aborted by user")
                    return

                tsm_rows = self._get_group_switch_rows(deps, code, port)
                await service.push_status(runtime, f"Applying SDU switch path commands for {cable_name}")
                applied = await apply_switch_driver_paths(tsm_rows, instruments)
                if len(applied) == 0:
                    raise ValueError(f"No SDU switch commands were parsed for {cable_name}")

                inject_tsm_rows = self._get_inject_switch_rows(deps)

                for frequency in frequencies:
                    if runtime.abort_requested:
                        await service.abort_runtime(runtime, "Calibration aborted by user")
                        return

                    key = self._frequency_key(frequency)
                    sg_cal_value = cal_sg_by_key[key]
                    inject_loss = inject_by_key[key]
                    inject_sa_loss = float(inject_loss["sa_loss"])
                    inject_pm_loss = float(inject_loss["dl_pm_loss"])

                    cal_sg_setpoint = 10.0 + (10.0 - sg_cal_value)
                    await cal_signal_generator.set_frequency(frequency)
                    await cal_signal_generator.set_power_level(cal_sg_setpoint)
                    await spectrum_analyzer.set_center_frequency(frequency)
                    await spectrum_analyzer.set_peak_search(1)
                    await asyncio.sleep(1.0)
                    measured_sa_peak = float(await spectrum_analyzer.get_delta_marker_delta_y_value(1))

                    await service.push_status(
                        runtime,
                        f"{cable_name} @ {frequency} MHz: SG set={cal_sg_setpoint:.2f} dBm, SA peak={measured_sa_peak:.2f} dBm",
                    )

                    applied_inject = await apply_switch_driver_paths(inject_tsm_rows, instruments)
                    if len(applied_inject) == 0:
                        raise ValueError("No SDU switch commands were parsed for INJECT_CAL")

                    inject_frequency = frequency + 1.0
                    desired_power_meter_level = measured_sa_peak - (inject_sa_loss - inject_pm_loss)
                    estimated_sg_level = desired_power_meter_level - inject_pm_loss

                    await inject_signal_generator.set_frequency(inject_frequency)
                    current_inject_sg_level = estimated_sg_level
                    await inject_signal_generator.set_power_level(current_inject_sg_level)
                    await downlink_power_meter.set_channel_frequency(inject_frequency, dl_pm_channel_number)

                    measured_pm = float(await downlink_power_meter.get_channel_power(dl_pm_channel_number))
                    for _ in range(10):
                        delta = desired_power_meter_level - measured_pm
                        if abs(delta) <= 0.1:
                            break
                        current_inject_sg_level += delta
                        await inject_signal_generator.set_power_level(current_inject_sg_level)
                        await asyncio.sleep(0.6)
                        measured_pm = float(await downlink_power_meter.get_channel_power(dl_pm_channel_number))

                    await spectrum_analyzer.set_center_frequency(frequency)
                    await spectrum_analyzer.set_peak_search(1)
                    await spectrum_analyzer.set_normal_marker(2)
                    await spectrum_analyzer.set_delta_marker_on(2)
                    await spectrum_analyzer.set_delta_marker_peak(2)
                    await spectrum_analyzer.set_delta_marker_maximum_next(2)
                    await asyncio.sleep(0.5)

                    downlink_peak_value = float(await spectrum_analyzer.get_delta_marker_delta_y_value(1))
                    delta_uncertainty = abs(float(await spectrum_analyzer.get_delta_marker_delta_y_value(2)))
                    measured_value = round(downlink_peak_value - delta_uncertainty, 1)

                    completed += 1
                    progress = 10.0 + ((completed / total_measurements) * 88.0)
                    now = datetime.now(timezone.utc)

                    await service.record_measurement(
                        runtime,
                        self._resolve_runtime_channel(payload.channels, code, port, frequency),
                        frequency,
                        value=measured_value,
                        timestamp=now,
                        progress=min(progress, 98.0),
                    )
                    await service.push_status(
                        runtime,
                        (
                            f"Downlink cal @ {frequency} MHz ({cable_name}): desired PM={desired_power_meter_level:.2f} dBm, "
                            f"measured PM={measured_pm:.2f} dBm, inject SG={current_inject_sg_level:.2f} dBm, "
                            f"delta={delta_uncertainty:.2f} dB, value={measured_value:.1f} dBm"
                        ),
                    )

                    # Restore downlink path before next frequency for this cable.
                    await apply_switch_driver_paths(tsm_rows, instruments)

            await service.complete_runtime(runtime, "Calibration run completed")
        except Exception as exc:
            raise ValueError(f"Downlink calibration failed: {exc}") from exc
        finally:
            try:
                if cal_signal_generator is not None:
                    await cal_signal_generator.set_rf_off()
            except Exception:
                pass
            try:
                if inject_signal_generator is not None:
                    await inject_signal_generator.set_rf_off()
            except Exception:
                pass

    def _build_grouped_frequency_map(
        self,
        channels: Iterable[Any],
        deps: CalibrationDependencies,
        include_spurious_bands: bool | None,
    ) -> dict[tuple[str, str], list[float]]:
        grouped: dict[tuple[str, str], set[float]] = {}

        for channel in channels:
            frequencies = self._build_frequency_list(channel, deps, include_spurious_bands)
            if len(frequencies) == 0:
                continue

            codes = self._extract_codes(getattr(channel, "code", ""))
            port = str(getattr(channel, "port", "") or "").strip()
            if len(codes) == 0:
                codes = [""]

            for code in codes:
                key = (code, port)
                bucket = grouped.setdefault(key, set())
                for frequency in frequencies:
                    bucket.add(round(float(frequency), 6))

        return {key: sorted(values) for key, values in grouped.items()}

    def _extract_codes(self, value: Any) -> list[str]:
        text = str(value or "").strip()
        if text == "":
            return []

        out: list[str] = []
        for item in text.split(","):
            code = item.strip()
            if code.lower().endswith("_spurious"):
                code = code[: -len("_spurious")].strip()
            if code != "" and code not in out:
                out.append(code)
        return out

    def _build_prompt_channel(self, channels: Iterable[Any], code: str, port: str, frequencies: list[float]):
        for channel in channels:
            channel_port = str(getattr(channel, "port", "") or "").strip()
            if port != channel_port:
                continue

            channel_codes = self._extract_codes(getattr(channel, "code", ""))
            if code != "" and code not in channel_codes:
                continue

            frequency = self._to_float(getattr(channel, "frequency", None))
            if frequency is None:
                continue
            if self._frequency_key(frequency) in {self._frequency_key(f) for f in frequencies}:
                return channel

        for channel in channels:
            return channel
        return None

    def _resolve_runtime_channel(self, channels: Iterable[Any], code: str, port: str, frequency: float):
        for channel in channels:
            channel_port = str(getattr(channel, "port", "") or "").strip()
            if port != channel_port:
                continue

            channel_codes = self._extract_codes(getattr(channel, "code", ""))
            if code != "" and code not in channel_codes:
                continue

            channel_frequency = self._to_float(getattr(channel, "frequency", None))
            if channel_frequency is None:
                continue
            if self._frequency_key(channel_frequency) != self._frequency_key(frequency):
                continue
            return channel

        for channel in channels:
            return channel
        raise ValueError("No channels available for recording measurement")

    def _frequency_key(self, value: float) -> str:
        return f"{float(value):.6f}"

    def _get_group_switch_rows(self, deps: CalibrationDependencies, code: str, port: str) -> list[dict[str, Any]]:
        candidate_codes = [code] if code.strip() != "" else []
        if "" not in candidate_codes:
            candidate_codes.append("")

        for candidate in candidate_codes:
            if candidate.strip() == "":
                continue

            rows = deps.test_systems_repo.get_project_tsm_path_rows_for_code(candidate, port if port != "" else None)
            if len(rows) > 0:
                return rows

            if port != "":
                rows = deps.test_systems_repo.get_project_tsm_path_rows_for_code(candidate)
                if len(rows) > 0:
                    return rows

        raise ValueError(
            f"No TSM path rows found for code/port '{code or '[blank]'}'/'{port or '[blank]'}' in Test Systems > TSM Paths"
        )

    def _get_inject_switch_rows(self, deps: CalibrationDependencies) -> list[dict[str, Any]]:
        rows = deps.test_systems_repo.get_project_tsm_path_rows_for_code("INJECT_CAL")
        if len(rows) == 0:
            raise ValueError("No TSM path rows found for code INJECT_CAL in Test Systems > TSM Paths")
        return rows

    def _build_signal_generator(self, config: dict[str, str]) -> SignalGenerator:
        model = normalize_model(config.get("model", ""))
        address = build_address(config.get("address", ""))
        instrument = cast(SignalGenerator | None, get_instrument(InstrumentTypes.SignalGenerator, model, address))
        if instrument is None:
            raise ValueError(f"Unsupported signal generator model: {config.get('model', '')}")
        return instrument

    def _build_power_meter(self, config: dict[str, str]) -> PowerMeter:
        model = normalize_model(config.get("model", ""))
        address = build_address(config.get("address", ""))
        instrument = cast(PowerMeter | None, get_instrument(InstrumentTypes.PowerMeter, model, address))
        if instrument is None:
            raise ValueError(f"Unsupported power meter model: {config.get('model', '')}")
        return instrument

    def _build_spectrum_analyzer(self, config: dict[str, str]) -> SpectrumAnalyzer:
        model = normalize_model(config.get("model", ""))
        address = build_address(config.get("address", ""))
        instrument = cast(SpectrumAnalyzer | None, get_instrument(InstrumentTypes.SpectrumAnalyzer, model, address))
        if instrument is None:
            raise ValueError(f"Unsupported spectrum analyzer model: {config.get('model', '')}")
        return instrument

    def _get_power_meter_channel(self, rows: list[dict[str, Any]], power_meter_name: str) -> str:
        for row in rows:
            if str(row.get("powerMeter") or "") == power_meter_name:
                return str(row.get("channel") or "A").upper()
        raise ValueError(f"{power_meter_name} channel mapping not found")

    def _get_instrument_config(self, rows: list[dict[str, Any]], instrument_name: str) -> dict[str, str]:
        for row in rows:
            if str(row.get("instrument_name") or "") == instrument_name:
                address_main = str(row.get("address_main") or "").strip()
                address_redt = str(row.get("address_redt") or "").strip()
                use_redt = self._to_bool(row.get("use_redt", False))
                selected_address = address_redt if use_redt and address_redt != "" else address_main
                return {
                    "model": str(row.get("model") or ""),
                    "address_main": address_main,
                    "address_redt": address_redt,
                    "address": selected_address,
                }
        raise ValueError(f"Instrument configuration not found: {instrument_name}")

    def _to_bool(self, value: Any) -> bool:
        if isinstance(value, bool):
            return value
        if isinstance(value, (int, float)):
            return int(value) != 0
        return str(value or "").strip().lower() in {"1", "true", "yes", "on"}

    def _estimate_total_measurements(self, channels: Iterable[Any], deps: CalibrationDependencies, include_spurious_bands: bool | None) -> int:
        total = 0
        for channel in channels:
            total += len(self._build_frequency_list(channel, deps, include_spurious_bands))
        return max(total, 1)

    def _build_frequency_list(self, channel, deps: CalibrationDependencies, include_spurious_bands: bool | None) -> list[float]:
        base_frequency = self._to_float(channel.frequency)
        if base_frequency is None:
            return []

        frequencies: set[float] = {base_frequency}
        if not include_spurious_bands:
            return sorted(frequencies)

        spurious_row = self._get_spurious_row_for_channel(channel, deps)

        if spurious_row:
            for field_name in ("fbt", "fbt_hot", "fbt_cold"):
                for offset in self._extract_offsets(spurious_row.get(field_name)):
                    frequencies.add(round(base_frequency + offset, 6))

            profile_name = str(spurious_row.get("profile_name") or "").strip()
            if profile_name:
                for start_frequency, stop_frequency in self._get_band_ranges(profile_name, deps):
                    for frequency in self._expand_range(start_frequency, stop_frequency, 100.0):
                        frequencies.add(round(frequency, 6))

        return sorted(frequencies)

    def _get_spurious_row_for_channel(self, channel, deps: CalibrationDependencies) -> dict[str, Any] | None:
        channel_code = str(channel.code or "").strip()
        channel_port = str(channel.port or "").strip()
        channel_label = str(channel.frequency_label or "").strip()
        channel_frequency = str(channel.frequency or "").strip()

        candidates = []
        for code in self._extract_codes(channel_code):
            tx = deps.transmitter_repo.get_by_code(code)
            if tx is not None:
                candidates.append(tx)
        if not candidates:
            candidates = deps.transmitter_repo.get_all()

        for tx in candidates:
            details = getattr(tx, "modulation_details", None)
            rows = getattr(details, "spurious_specs", []) or []
            for row in rows:
                row_label = str(getattr(row, "frequency_label", "")).strip()
                row_frequency = str(getattr(row, "frequency", "")).strip()
                if row_label != channel_label or row_frequency != channel_frequency:
                    continue

                row_port = str(getattr(row, "port", "")).strip()
                if channel_port != "" and row_port != channel_port:
                    continue
                return row.model_dump()
        return None

    def _get_band_ranges(self, profile_name: str, deps: CalibrationDependencies) -> list[tuple[float, float]]:
        ranges: list[tuple[float, float]] = []
        for row in deps.transmitter_repo.get_spurious_band_configs():
            if not bool(row.get("enable", True)):
                continue
            if str(row.get("profile_name") or "").strip() != profile_name:
                continue
            start_frequency = self._to_float(row.get("start_frequency"))
            stop_frequency = self._to_float(row.get("stop_frequency"))
            if start_frequency is None or stop_frequency is None:
                continue
            ranges.append((start_frequency, stop_frequency))
        return ranges

    def _expand_range(self, start_frequency: float, stop_frequency: float, step: float) -> list[float]:
        low = min(start_frequency, stop_frequency)
        high = max(start_frequency, stop_frequency)
        values: list[float] = []
        current = low
        while current <= high + 1e-9:
            values.append(round(current, 6))
            current += step
        if round(high, 6) not in values:
            values.append(round(high, 6))
        return values

    def _extract_offsets(self, matrix: Any) -> list[float]:
        offsets: list[float] = []
        if not isinstance(matrix, list):
            return offsets
        for row in matrix:
            if not isinstance(row, list) or len(row) == 0:
                continue
            offset = self._to_float(row[0])
            if offset is None:
                continue
            offsets.append(offset)
        return offsets

    def _to_float(self, value: Any) -> float | None:
        try:
            text = str(value).strip()
            if text == "":
                return None
            return float(text)
        except Exception:
            return None


PROCEDURE_CLASS = DownlinkCalibrationProcedure