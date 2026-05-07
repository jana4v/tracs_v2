from __future__ import annotations

import asyncio
from datetime import datetime, timezone
from typing import Any

from src.calibration._downlink import DownlinkCalibrationProcedure
from src.calibration.base import CalibrationDependencies
from src.schemas.calibration_data import MeasureMissingChannel
from src.schemas.calibration_data import MeasureRunResultRow, MeasureRunStartRequest
from src.services.switch_driver_paths import apply_switch_driver_paths
from src.utils.fspl import compute_fspl

from .base import MeasureProcedure, MeasureSelectedRow


def _to_float(value: Any) -> float:
    try:
        return float(value)
    except Exception:
        return 0.0


class PowerMeasureProcedure(MeasureProcedure):
    parameter_name = "power"

    def __init__(self) -> None:
        self._downlink = DownlinkCalibrationProcedure()

    async def execute(
        self,
        payload: MeasureRunStartRequest,
        deps: CalibrationDependencies,
        selected_rows: list[MeasureSelectedRow],
    ) -> list[MeasureRunResultRow]:
        cal_id = str(payload.cal_id or "").strip()
        if cal_id == "":
            raise ValueError("cal_id is required to run power measurements")

        power_rows = [item for item in selected_rows if bool(item.row.power_selected)]
        if len(power_rows) == 0:
            return []

        inject_rows = deps.inject_cal_repo.list_rows(cal_id)
        inject_by_key = {
            self._downlink._frequency_key(float(row["frequency"])): {
                "sa_loss": float(row["sa_loss"]),
                "dl_pm_loss": float(row["dl_pm_loss"]),
            }
            for row in inject_rows
        }

        downlink_rows = deps.downlink_cal_repo.list_rows(cal_id)
        cable_loss_exact: dict[tuple[str, str, str], float] = {}
        for row in downlink_rows:
            code = str(row.get("code") or "").strip()
            port = str(row.get("port") or "").strip()
            freq = float(row.get("frequency") or 0.0)
            value = float(row.get("value") or 0.0)
            key = (code, port, self._downlink._frequency_key(freq))
            cable_loss_exact[key] = value

        # Build a per-channel loss map from the Calibration / Transmitter table
        # (modulation_details.calibration_specs) and the On Board Losses /
        # Transmitter table (modulation_details.on_board_loss_specs).
        calibration_loss_map: dict[tuple[str, str, str], dict[str, float]] = {}
        try:
            calibration_entries = deps.transmitter_repo.get_calibration_rows()
        except Exception:
            calibration_entries = []
        for entry in calibration_entries:
            row = entry.get("row") if isinstance(entry, dict) else None
            if not isinstance(row, dict):
                continue
            cal_code = str(row.get("code") or "").strip()
            cal_port = str(row.get("port") or "").strip()
            try:
                cal_freq = float(str(row.get("frequency") or "").strip())
            except Exception:
                continue
            calibration_loss_map[(cal_code, cal_port, self._downlink._frequency_key(cal_freq))] = {
                "system_loss": _to_float(row.get("system_loss")),
                "fixed_pad_loss": _to_float(row.get("fixed_pad_loss")),
                "antenna_gain": _to_float(row.get("antenna_gain")),
                "ground_antenna_gain": _to_float(row.get("ground_antenna_gain")),
                "distance": _to_float(row.get("distance")),
            }

        on_board_loss_map: dict[tuple[str, str, str], float] = {}
        try:
            on_board_entries = deps.transmitter_repo.get_on_board_loss_rows()
        except Exception:
            on_board_entries = []
        for entry in on_board_entries:
            row = entry.get("row") if isinstance(entry, dict) else None
            if not isinstance(row, dict):
                continue
            ob_code = str(row.get("code") or "").strip()
            ob_port = str(row.get("port") or "").strip()
            try:
                ob_freq = float(str(row.get("frequency") or "").strip())
            except Exception:
                continue
            on_board_loss_map[(ob_code, ob_port, self._downlink._frequency_key(ob_freq))] = (
                _to_float(row.get("loss_db"))
            )

        instruments = deps.test_systems_repo.get_project_instruments_rows()
        inject_sg_config = self._downlink._get_instrument_config(instruments, "InjectSignalGenerator")
        sa_config = self._downlink._get_instrument_config(instruments, "SpectrumAnalyser")
        dl_pm_config = self._downlink._get_instrument_config(instruments, "DownlinkPowerMeter")

        power_meter_rows = deps.test_systems_repo.get_project_power_meter_rows()
        dl_pm_channel = self._downlink._get_power_meter_channel(power_meter_rows, "DownlinkPowerMeter")
        dl_pm_channel_number = 2 if dl_pm_channel == "B" else 1

        inject_signal_generator = self._downlink._build_signal_generator(inject_sg_config)
        spectrum_analyzer = self._downlink._build_spectrum_analyzer(sa_config)
        downlink_power_meter = self._downlink._build_power_meter(dl_pm_config)

        results: list[MeasureRunResultRow] = []

        inject_tsm_rows = self._downlink._get_inject_switch_rows(deps)
        active_code: str | None = None
        active_port: str | None = None
        active_switch_signature: tuple[tuple[str, tuple[str, ...]], ...] | None = None

        try:
            await inject_signal_generator.preset_instrument()
            await inject_signal_generator.set_power_level(-80.0)
            await inject_signal_generator.set_rf_off()

            await downlink_power_meter.preset_instrument()
            await downlink_power_meter.set_channel_limits(dl_pm_channel_number, -50, 30)

            await spectrum_analyzer.preset_instrument()
            await spectrum_analyzer.set_resolution_bandwidth_auto_on_off("OFF")
            await spectrum_analyzer.set_video_bandwidth_auto_on_off("OFF")
            await spectrum_analyzer.set_resolution_bandwidth(30)
            await spectrum_analyzer.set_video_bandwidth(3)
            await spectrum_analyzer.set_span(5)
            await spectrum_analyzer.set_sweep_points(1001)

            for selected in power_rows:
                row = selected.row
                code = str(row.code or "").strip()
                port = str(row.port or "").strip()
                freq_label = str(row.frequency_label or "").strip()
                try:
                    frequency = float(str(row.frequency or "").strip())
                except Exception:
                    results.append(
                        MeasureRunResultRow(
                            system_kind=selected.system_kind,  # type: ignore[arg-type]
                            code=code,
                            port=port,
                            frequency_label=freq_label,
                            frequency=0.0,
                            parameter="power",
                            measured_value=0.0,
                            applied_loss=0.0,
                            final_value=0.0,
                            status="failed",
                            message="Invalid frequency in selected row",
                            timestamp=datetime.now(timezone.utc),
                        )
                    )
                    continue

                inject_key = self._downlink._frequency_key(frequency)
                inject_loss = inject_by_key.get(inject_key)
                if inject_loss is None:
                    results.append(
                        MeasureRunResultRow(
                            system_kind=selected.system_kind,  # type: ignore[arg-type]
                            code=code,
                            port=port,
                            frequency_label=freq_label,
                            frequency=frequency,
                            parameter="power",
                            measured_value=0.0,
                            applied_loss=0.0,
                            final_value=0.0,
                            status="failed",
                            message=(
                                f"Inject calibration missing for frequency {frequency:.6f} MHz "
                                f"in cal_id {cal_id}"
                            ),
                            timestamp=datetime.now(timezone.utc),
                        )
                    )
                    continue

                tsm_rows = self._downlink._get_group_switch_rows(deps, code, port)
                current_switch_signature = self._downlink._build_switch_signature(tsm_rows)
                needs_switch_update = (
                    code != active_code
                    or port != active_port
                    or current_switch_signature != active_switch_signature
                )
                if needs_switch_update:
                    await apply_switch_driver_paths(tsm_rows, instruments)
                    await apply_switch_driver_paths(inject_tsm_rows, instruments)
                    active_code = code
                    active_port = port
                    active_switch_signature = current_switch_signature

                await spectrum_analyzer.set_center_frequency(frequency)
                sweep_time = float(await spectrum_analyzer.get_sweep_time())
                await asyncio.sleep(0.5 + (2 * sweep_time))
                await spectrum_analyzer.set_peak_search(1)
                await asyncio.sleep(sweep_time)
                measured_sa_peak = float(await spectrum_analyzer.get_marker_value_y_data(1))
                await spectrum_analyzer.set_reference_level(measured_sa_peak + 10)
                await asyncio.sleep(0.5 + (2 * sweep_time))
                channel_power_1mhz = float(
                    await spectrum_analyzer.measure_channel_power(
                        center_frequency_mhz=frequency,
                        channel_bandwidth_mhz=1.0,
                        trace_number=1,
                        detector_mode="average",
                    )
                )

                inject_frequency = frequency + 1.0
                inject_sa_loss = float(inject_loss["sa_loss"])
                inject_pm_loss = float(inject_loss["dl_pm_loss"])
                desired_power_meter_level = channel_power_1mhz + (inject_sa_loss - inject_pm_loss)
                current_inject_sg_level = desired_power_meter_level + inject_pm_loss

                await inject_signal_generator.set_frequency(inject_frequency)
                await inject_signal_generator.set_power_level(current_inject_sg_level)
                await inject_signal_generator.set_rf_on()

                await downlink_power_meter.set_channel_frequency(inject_frequency, dl_pm_channel_number)
                measured_pm = float(await downlink_power_meter.get_channel_power(dl_pm_channel_number))
   
                # Read injected-carrier peak directly and compute uncertainty
                # against the expected injected level derived from channel power.

                await spectrum_analyzer.set_peak_search(1)
                await asyncio.sleep(sweep_time)
                expected_inject_freq_hz = inject_frequency * 1_000_000.0
                freq_tolerance_hz = 200_000.0
                marker_freq_hz = float(await spectrum_analyzer.get_marker_value_x_data(1))
                for _ in range(20):
                    if abs(marker_freq_hz - expected_inject_freq_hz) <= freq_tolerance_hz:
                        break
                    if marker_freq_hz < expected_inject_freq_hz:
                        await spectrum_analyzer.set_marker_find_next_right_peak(1)
                    else:
                        await spectrum_analyzer.set_marker_find_next_left_peak(1)
                    await asyncio.sleep(max(0.1, sweep_time))
                    marker_freq_hz = float(await spectrum_analyzer.get_marker_value_x_data(1))
                measured_inject_peak = float(await spectrum_analyzer.get_marker_value_y_data(1))
                uncertainty = channel_power_1mhz - (measured_inject_peak + (desired_power_meter_level-measured_pm))
                await spectrum_analyzer.set_markers_off()
                await inject_signal_generator.set_rf_off()

                
                corrected_measured_value = round(channel_power_1mhz + uncertainty, 1)

                cal_key = (code, port, self._downlink._frequency_key(frequency))
                cal_components = calibration_loss_map.get(cal_key, {})
                # The GUI Calibration / Transmitter panel sources `system_loss`
                # from DownlinkCalCalibrationData for the selected Cal ID
                # (calibrationDataApi.getDownlinkCalData) and overlays it on
                # the persisted calibration row. Mirror that exactly here:
                # always prefer the downlink-cal value for the active cal_id,
                # and fall back to the persisted `calibration_specs.system_loss`
                # only if no downlink-cal entry exists.
                downlink_loss = cable_loss_exact.get(cal_key)
                if downlink_loss is not None:
                    system_loss = _to_float(downlink_loss)
                else:
                    system_loss = _to_float(cal_components.get("system_loss", 0.0))
                fixed_pad_loss = _to_float(cal_components.get("fixed_pad_loss", 0.0))
                antenna_gain = _to_float(cal_components.get("antenna_gain", 0.0))
                ground_antenna_gain = _to_float(cal_components.get("ground_antenna_gain", 0.0))
                distance = _to_float(cal_components.get("distance", 0.0))
                fspl = compute_fspl(distance, frequency)
                total_loss_calibration = (
                    abs(antenna_gain)
                    + abs(ground_antenna_gain)
                    - abs(fspl)
                    - abs(system_loss)
                    - abs(fixed_pad_loss)
                )
                on_board_loss = _to_float(on_board_loss_map.get(cal_key, 0.0))
                applied_loss = round(total_loss_calibration + on_board_loss, 2)
                final_value = round(corrected_measured_value - applied_loss, 1)

                results.append(
                    MeasureRunResultRow(
                        system_kind=selected.system_kind,  # type: ignore[arg-type]
                        code=code,
                        port=port,
                        frequency_label=freq_label,
                        frequency=frequency,
                        parameter="power",
                        measured_value=corrected_measured_value,
                        applied_loss=applied_loss,
                        final_value=final_value,
                        raw_value=round(channel_power_1mhz, 1),
                        system_loss=round(abs(system_loss), 2),
                        fixed_pad_loss=round(abs(fixed_pad_loss), 2),
                        antenna_gain=round(antenna_gain, 2),
                        ground_antenna_gain=round(ground_antenna_gain, 2),
                        distance=round(distance, 3),
                        fspl=round(abs(fspl), 2),
                        total_loss_calibration=round(total_loss_calibration, 2),
                        on_board_loss=round(on_board_loss, 2),
                        status="completed",
                        message=(
                            f"Power measured (1MHz channel power). raw={channel_power_1mhz:.1f} dBm, "
                            f"corrected={corrected_measured_value:.1f} dBm, "
                            f"cal_loss={total_loss_calibration:.2f} dB, on_board_loss={on_board_loss:.2f} dB, "
                            f"applied_loss={applied_loss:.2f} dB, final={final_value:.1f} dBm, "
                            f"inject uncertainty={uncertainty:.2f} dB"
                        ),
                        timestamp=datetime.now(timezone.utc),
                    )
                )
        finally:
            try:
                await inject_signal_generator.set_rf_off()
            except Exception:
                pass

        return results

    def get_missing_downlink_channels(
        self,
        payload: MeasureRunStartRequest,
        deps: CalibrationDependencies,
        selected_rows: list[MeasureSelectedRow],
    ) -> list[MeasureMissingChannel]:
        # Power applied loss is now sourced from the per-transmitter
        # Calibration / Transmitter table plus On Board Losses, so the
        # legacy DownlinkCalCalibrationData gating is no longer required.
        _ = (payload, deps, selected_rows)
        return []
