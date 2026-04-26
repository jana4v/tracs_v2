from __future__ import annotations

import asyncio
from datetime import datetime, timezone
from typing import Any, Iterable, cast

from src.InstrumentApi.Models import InstrumentTypes
from src.InstrumentApi.PowerMeter.BaseClass import PowerMeter
from src.InstrumentApi.SignalGenerator.BaseClass import SignalGenerator
from src.InstrumentApi.factory import get_instrument
from src.services.switch_driver_paths import apply_switch_driver_paths, build_address, normalize_model

from .base import CalibrationDependencies, CalibrationProcedure


class SgCalibrationProcedure(CalibrationProcedure):
    cal_type = "cal_sg"

    async def execute(self, service, runtime, payload, deps: CalibrationDependencies) -> None:
        power_meter_rows = deps.test_systems_repo.get_project_power_meter_rows()
        powermeter_channel = self._get_power_meter_channel(power_meter_rows)
        prompt_channel = runtime.channels[0] if runtime.channels else None

        await service.prompt_operator(
            runtime,
            self._get_operator_prompt_message(powermeter_channel),
            prompt_channel,
        )
        if runtime.abort_requested:
            await service.abort_runtime(runtime, "Calibration aborted before instrument setup")
            return

        signal_generator: SignalGenerator | None = None
        try:
            instruments = deps.test_systems_repo.get_project_instruments_rows()
            if self._should_apply_switch_paths():
                switch_path_rows = self._get_switch_path_rows(deps, runtime, payload)
                await service.push_status(runtime, "Applying SDU switch path commands")
                applied_switch_commands = await apply_switch_driver_paths(switch_path_rows, instruments)
                if self._require_switch_commands() and len(applied_switch_commands) == 0:
                    raise ValueError("No SDU switch commands were parsed for this calibration")
                if len(applied_switch_commands) > 0:
                    formatted_commands = ", ".join(
                        f"{driver}: {', '.join(commands)}"
                        for driver, commands in sorted(applied_switch_commands.items())
                    )
                    await service.push_status(runtime, f"Applied SDU switch path commands: {formatted_commands}")

            sg_config = self._get_instrument_config(instruments, "CalSignalGenerator")
            pm_config = self._get_instrument_config(instruments, "CalPowerMeter")

            await service.push_status(
                runtime,
                f"Using CalSignalGenerator model={sg_config.get('model', '')} address={sg_config.get('address', '')}",
            )
            await service.push_status(
                runtime,
                f"Using CalPowerMeter model={pm_config.get('model', '')} address={pm_config.get('address', '')}",
            )

            signal_generator = self._build_signal_generator(sg_config)
            power_meter = self._build_power_meter(pm_config)

            power_meter_channel_number = 2 if powermeter_channel == "B" else 1

            await signal_generator.preset_instrument()
            await signal_generator.set_power_level(10)
            await signal_generator.set_rf_on()

            await power_meter.preset_instrument()
            await power_meter.set_channel_limits(power_meter_channel_number, -50, 30)

            total_measurements = self._estimate_total_measurements(payload.channels, deps, payload.include_spurious_bands)
            completed = 0

            await service.set_running(runtime, "Calibration run started")
            for channel in payload.channels:
                if runtime.abort_requested:
                    await service.abort_runtime(runtime, "Calibration aborted by user")
                    return

                frequencies = self._build_frequency_list(channel, deps, payload.include_spurious_bands)
                await service.push_status(
                    runtime,
                    f"Prepared {len(frequencies)} frequencies for {channel.frequency_label} ({channel.frequency})",
                )

                for frequency in frequencies:
                    if runtime.abort_requested:
                        await service.abort_runtime(runtime, "Calibration aborted by user")
                        return

                    await signal_generator.set_frequency(frequency)
                    await power_meter.set_channel_frequency(frequency, power_meter_channel_number)
                    await asyncio.sleep(1.0)
                    measured_value = await power_meter.get_channel_power(power_meter_channel_number)
                    measured_value_rounded = round(float(measured_value), 1)

                    now = datetime.now(timezone.utc)
                    deps.cal_sg_repo.upsert(runtime.cal_id, frequency, measured_value_rounded, now)
                    completed += 1
                    progress = 10.0 + ((completed / max(total_measurements, 1)) * 88.0)
                    await service.record_measurement(
                        runtime,
                        channel,
                        frequency,
                        value=measured_value_rounded,
                        timestamp=now,
                        progress=min(progress, 98.0),
                    )

            await service.complete_runtime(runtime, "Calibration run completed")
        except Exception as exc:
            raise ValueError(f"Cal SG calibration failed: {exc}") from exc
        finally:
            try:
                if signal_generator is not None:
                    await signal_generator.set_rf_off()
            except Exception:
                pass

    def _get_operator_prompt_message(self, power_meter_channel: str) -> str:
        return f"Connect powermeter channel {power_meter_channel} to CAL Signal Generator"

    def _get_switch_path_rows(self, deps: CalibrationDependencies, runtime, payload) -> list[dict[str, Any]]:
        return []

    def _should_apply_switch_paths(self) -> bool:
        return False

    def _require_switch_commands(self) -> bool:
        return False

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
        if channel_code != "":
            tx = deps.transmitter_repo.get_by_code(channel_code)
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

                # If a specific port is provided, prefer that exact row.
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

    def _build_signal_generator(self, config: dict[str, str]) -> SignalGenerator:
        model = normalize_model(config.get("model", ""))
        address = build_address(config.get("address", ""))
        instrument = cast(SignalGenerator | None, get_instrument(InstrumentTypes.SignalGenerator, model, address))
        if instrument is None:
            raise ValueError(f"Unsupported CalSignalGenerator model: {config.get('model', '')}")
        return instrument

    def _build_power_meter(self, config: dict[str, str]) -> PowerMeter:
        model = normalize_model(config.get("model", ""))
        address = build_address(config.get("address", ""))
        instrument = cast(PowerMeter | None, get_instrument(InstrumentTypes.PowerMeter, model, address))
        if instrument is None:
            raise ValueError(f"Unsupported CalPowerMeter model: {config.get('model', '')}")
        return instrument

    def _get_power_meter_channel(self, rows: list[dict[str, Any]]) -> str:
        for row in rows:
            if str(row.get("powerMeter") or "") == "CalPowerMeter":
                return str(row.get("channel") or "A").upper()
        raise ValueError("CalPowerMeter channel mapping not found")

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

    def _to_float(self, value: Any) -> float | None:
        try:
            text = str(value).strip()
            if text == "":
                return None
            return float(text)
        except Exception:
            return None


PROCEDURE_CLASS = SgCalibrationProcedure