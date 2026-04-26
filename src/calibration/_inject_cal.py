from __future__ import annotations

import asyncio
from datetime import datetime, timezone
from typing import Any

from src.InstrumentApi.Models import InstrumentTypes
from src.InstrumentApi.PowerMeter.BaseClass import PowerMeter
from src.InstrumentApi.SignalGenerator.BaseClass import SignalGenerator
from src.InstrumentApi.factory import get_instrument
from src.services.switch_driver_paths import apply_switch_driver_paths, build_address, normalize_model

from .base import CalibrationDependencies, CalibrationProcedure


class InjectCalProcedure(CalibrationProcedure):
    cal_type = "inject_cal"

    async def execute(self, service, runtime, payload, deps: CalibrationDependencies) -> None:
        power_meter_rows = deps.test_systems_repo.get_project_power_meter_rows()
        sa_channel = self._get_power_meter_channel(power_meter_rows, "UplinkPowerMeter")
        dl_pm_channel = self._get_power_meter_channel(power_meter_rows, "DownlinkPowerMeter")
        prompt_channel = runtime.channels[0] if runtime.channels else None

        await service.prompt_operator(
            runtime,
            "Connect Uplink power sensor to Spectrum analyser Cable end",
            prompt_channel,
        )
        if runtime.abort_requested:
            await service.abort_runtime(runtime, "Calibration aborted before instrument setup")
            return

        instruments = deps.test_systems_repo.get_project_instruments_rows()
        switch_path_rows = self._get_switch_path_rows(deps)
        await service.push_status(runtime, "Applying SDU switch path commands")
        applied_switch_commands = await apply_switch_driver_paths(switch_path_rows, instruments)
        if len(applied_switch_commands) == 0:
            raise ValueError("No SDU switch commands were parsed for this calibration")
        formatted_commands = ", ".join(
            f"{driver}: {', '.join(commands)}"
            for driver, commands in sorted(applied_switch_commands.items())
        )
        await service.push_status(runtime, f"Applied SDU switch path commands: {formatted_commands}")

        sg_config = self._get_instrument_config(instruments, "InjectSignalGenerator")
        sa_pm_config = self._get_instrument_config(instruments, "UplinkPowerMeter")
        dl_pm_config = self._get_instrument_config(instruments, "DownlinkPowerMeter")

        await service.push_status(
            runtime,
            f"Using InjectSignalGenerator model={sg_config.get('model', '')} address={sg_config.get('address', '')}",
        )
        await service.push_status(
            runtime,
            f"Using UplinkPowerMeter model={sa_pm_config.get('model', '')} address={sa_pm_config.get('address', '')}",
        )
        await service.push_status(
            runtime,
            f"Using DownlinkPowerMeter model={dl_pm_config.get('model', '')} address={dl_pm_config.get('address', '')}",
        )

        signal_generator: SignalGenerator | None = None
        try:
            signal_generator = self._build_signal_generator(sg_config)
            sa_power_meter = self._build_power_meter(sa_pm_config)
            dl_power_meter = self._build_power_meter(dl_pm_config)

            sa_channel_number = 2 if sa_channel == "B" else 1
            dl_channel_number = 2 if dl_pm_channel == "B" else 1

            await signal_generator.preset_instrument()
            await signal_generator.set_power_level(0)
            await signal_generator.set_rf_on()

            await sa_power_meter.preset_instrument()
            await sa_power_meter.set_channel_limits(sa_channel_number, -50, 30)

            await dl_power_meter.preset_instrument()
            await dl_power_meter.set_channel_limits(dl_channel_number, -50, 30)

            total_measurements = self._estimate_total_measurements(payload.channels)
            completed = 0

            await service.set_running(runtime, "Calibration run started")
            for channel in payload.channels:
                if runtime.abort_requested:
                    await service.abort_runtime(runtime, "Calibration aborted by user")
                    return

                frequency = self._to_float(channel.frequency)
                if frequency is None:
                    continue

                await signal_generator.set_frequency(frequency)
                await sa_power_meter.set_channel_frequency(frequency, sa_channel_number)
                await dl_power_meter.set_channel_frequency(frequency, dl_channel_number)
                await asyncio.sleep(1.0)

                sa_value = await sa_power_meter.get_channel_power(sa_channel_number)
                dl_pm_value = await dl_power_meter.get_channel_power(dl_channel_number)
                sa_value_rounded = round(float(sa_value), 1)
                dl_pm_value_rounded = round(float(dl_pm_value), 1)

                now = datetime.now(timezone.utc)
                deps.inject_cal_repo.upsert(runtime.cal_id, frequency, sa_value_rounded, dl_pm_value_rounded, now)
                completed += 1
                progress = 10.0 + ((completed / max(total_measurements, 1)) * 88.0)
                await service.record_measurement(
                    runtime,
                    channel,
                    frequency,
                    value=None,
                    sa_loss=sa_value_rounded,
                    dl_pm_loss=dl_pm_value_rounded,
                    timestamp=now,
                    progress=min(progress, 98.0),
                )
                await service.push_status(
                    runtime,
                    f"Inject cal @ {frequency} MHz: SA Loss={sa_value_rounded} dB, DL_PM Loss={dl_pm_value_rounded} dB",
                )

            await service.complete_runtime(runtime, "Calibration run completed")
        except Exception as exc:
            raise ValueError(f"Inject calibration failed: {exc}") from exc
        finally:
            try:
                if signal_generator is not None:
                    await signal_generator.set_rf_off()
            except Exception:
                pass

    def _get_switch_path_rows(self, deps: CalibrationDependencies) -> list[dict[str, Any]]:
        rows = deps.test_systems_repo.get_project_tsm_path_rows_for_code("INJECT_CAL")
        if len(rows) == 0:
            raise ValueError("No TSM path rows found for code INJECT_CAL in Test Systems > TSM Paths")
        return rows

    def _estimate_total_measurements(self, channels: list[Any]) -> int:
        total = 0
        for channel in channels:
            if self._to_float(getattr(channel, "frequency", None)) is not None:
                total += 1
        return max(total, 1)

    def _build_signal_generator(self, config: dict[str, str]) -> SignalGenerator:
        model = normalize_model(config.get("model", ""))
        address = build_address(config.get("address", ""))
        instrument = get_instrument(InstrumentTypes.SignalGenerator, model, address)
        if instrument is None:
            raise ValueError(f"Unsupported InjectSignalGenerator model: {config.get('model', '')}")
        return instrument

    def _build_power_meter(self, config: dict[str, str]) -> PowerMeter:
        model = normalize_model(config.get("model", ""))
        address = build_address(config.get("address", ""))
        instrument = get_instrument(InstrumentTypes.PowerMeter, model, address)
        if instrument is None:
            raise ValueError(f"Unsupported UplinkPowerMeter model: {config.get('model', '')}")
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

    def _to_float(self, value: Any) -> float | None:
        try:
            text = str(value).strip()
            if text == "":
                return None
            return float(text)
        except Exception:
            return None


PROCEDURE_CLASS = InjectCalProcedure
