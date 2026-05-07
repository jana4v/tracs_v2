from __future__ import annotations

import asyncio
import math
from datetime import datetime, timezone
from typing import Any

import numpy as np
import scipy.special as bessel

from src.calibration._downlink import DownlinkCalibrationProcedure
from src.calibration.base import CalibrationDependencies
from src.config import settings
from src.database.connection import Database
from src.repositories.mod_index_measurement_repo import ModIndexMeasurementRepository
from src.schemas.calibration_data import MeasureRunResultRow, MeasureRunStartRequest
from src.services.switch_driver_paths import apply_switch_driver_paths

from .base import MeasureProcedure, MeasureSelectedRow


_NO_ITERATIONS = 5
_SPAN_MHZ = 1.0
_RBW_KHZ = 5.0
_VBW_KHZ = 5.0
_MAXHOLD_DWELL_SECONDS = 2.0
_SIDEBAND_WINDOW_HZ = 10_000.0  # +/- 5 kHz around the expected sideband bin


class ModulationIndexMeasureProcedure(MeasureProcedure):
    """Measure modulation index per (transmitter, port, frequency, tone) using SA trace data.

    Procedure:
      * Set downlink switch path for the selected (code, port).
      * Configure SA: span 1 MHz, RBW/VBW 5 kHz, MAX HOLD trace mode.
      * Wait 2 seconds for the trace to settle.
      * Read trace and compute mod-index for each tone (from transmitter sub-carriers).
      * Repeat 5 times and average.
      * Persist results in ``ModIndexMeasurement`` table.
    """

    parameter_name = "modulation_index"

    def __init__(self) -> None:
        self._downlink = DownlinkCalibrationProcedure()
        self._repo = ModIndexMeasurementRepository(
            Database._db_path,
            settings.MOD_INDEX_MEASUREMENT_TABLE,
        )

    async def execute(
        self,
        payload: MeasureRunStartRequest,
        deps: CalibrationDependencies,
        selected_rows: list[MeasureSelectedRow],
    ) -> list[MeasureRunResultRow]:
        mod_rows = [item for item in selected_rows if bool(item.row.modulation_index_selected)]
        if len(mod_rows) == 0:
            return []

        instruments = deps.test_systems_repo.get_project_instruments_rows()
        sa_config = self._downlink._get_instrument_config(instruments, "SpectrumAnalyser")
        spectrum_analyzer = self._downlink._build_spectrum_analyzer(sa_config)

        run_id = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
        results: list[MeasureRunResultRow] = []

        active_code: str | None = None
        active_port: str | None = None
        active_switch_signature: tuple[tuple[str, tuple[str, ...]], ...] | None = None

        try:
            await spectrum_analyzer.preset_instrument()
            await spectrum_analyzer.set_resolution_bandwidth_auto_on_off("OFF")
            await spectrum_analyzer.set_video_bandwidth_auto_on_off("OFF")
            await spectrum_analyzer.set_resolution_bandwidth(_RBW_KHZ)
            await spectrum_analyzer.set_video_bandwidth(_VBW_KHZ)
            await spectrum_analyzer.set_span(_SPAN_MHZ)

            for selected in mod_rows:
                row = selected.row
                code = str(row.code or "").strip()
                port = str(row.port or "").strip()
                freq_label = str(row.frequency_label or "").strip()

                try:
                    frequency = float(str(row.frequency or "").strip())
                except Exception:
                    results.append(
                        self._make_result(
                            selected, code, port, freq_label, 0.0, 0.0, 0.0,
                            "failed", "Invalid frequency in selected row",
                        )
                    )
                    continue

                tones_khz = self._get_tones_khz_for_code(deps, code)
                if len(tones_khz) == 0:
                    results.append(
                        self._make_result(
                            selected, code, port, freq_label, frequency, 0.0, 0.0,
                            "skipped",
                            f"No sub-carrier tones configured for transmitter '{code}'",
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
                    active_code = code
                    active_port = port
                    active_switch_signature = current_switch_signature

                await spectrum_analyzer.set_center_frequency(frequency)
                try:
                    await spectrum_analyzer.set_marker_on_off(0)
                except Exception:
                    pass

                per_tone_index: dict[float, list[float]] = {tone: [] for tone in tones_khz}
                per_tone_sb_upper: dict[float, list[float]] = {tone: [] for tone in tones_khz}
                per_tone_sb_lower: dict[float, list[float]] = {tone: [] for tone in tones_khz}

                for _ in range(_NO_ITERATIONS):
                    await spectrum_analyzer.set_trace_mode_to_normal(1)
                    await asyncio.sleep(0.3)
                    await spectrum_analyzer.set_trace_mode_to_maxhold(1)
                    await asyncio.sleep(_MAXHOLD_DWELL_SECONDS)

                    trace_raw = await spectrum_analyzer.get_trace_data(1)
                    trace_array = np.asarray(trace_raw, dtype=float)
                    if trace_array.size == 0:
                        continue

                    for tone_khz in tones_khz:
                        sample = self._compute_mod_index(
                            trace_array,
                            span_mhz=_SPAN_MHZ,
                            tone_hz=float(tone_khz) * 1000.0,
                        )
                        if sample is None:
                            continue
                        sb_upper, sb_lower, mi = sample
                        per_tone_index[tone_khz].append(mi)
                        per_tone_sb_upper[tone_khz].append(sb_upper)
                        per_tone_sb_lower[tone_khz].append(sb_lower)

                try:
                    await spectrum_analyzer.set_trace_mode_to_normal(1)
                except Exception:
                    pass

                run_time = datetime.now(timezone.utc)
                for tone_khz in tones_khz:
                    samples = per_tone_index[tone_khz]
                    if len(samples) > 0:
                        avg_mi = round(sum(samples) / len(samples), 3)
                        avg_sb_upper = round(
                            sum(per_tone_sb_upper[tone_khz]) / len(per_tone_sb_upper[tone_khz]),
                            2,
                        )
                        avg_sb_lower = round(
                            sum(per_tone_sb_lower[tone_khz]) / len(per_tone_sb_lower[tone_khz]),
                            2,
                        )
                        status = "completed"
                        message = (
                            f"Tone {tone_khz:g} kHz: mod_index={avg_mi:.3f} "
                            f"({len(samples)}/{_NO_ITERATIONS} valid samples)"
                        )
                    else:
                        avg_mi = 0.0
                        avg_sb_upper = 0.0
                        avg_sb_lower = 0.0
                        status = "failed"
                        message = (
                            f"Tone {tone_khz:g} kHz: no valid sidebands detected over "
                            f"{_NO_ITERATIONS} samples"
                        )

                    self._repo.upsert(
                        run_id=run_id,
                        system_kind=str(selected.system_kind or ""),
                        code=code,
                        port=port,
                        frequency=frequency,
                        frequency_label=freq_label,
                        tone_khz=float(tone_khz),
                        mod_index=avg_mi,
                        sideband_upper=avg_sb_upper,
                        sideband_lower=avg_sb_lower,
                        samples=len(samples),
                        status=status,
                        date_time=run_time,
                    )

                    results.append(
                        MeasureRunResultRow(
                            system_kind=selected.system_kind,  # type: ignore[arg-type]
                            code=code,
                            port=port,
                            frequency_label=freq_label,
                            frequency=frequency,
                            parameter="modulation_index",
                            measured_value=avg_mi,
                            applied_loss=0.0,
                            final_value=avg_mi,
                            status=status,
                            message=message,
                            timestamp=run_time,
                        )
                    )
        finally:
            try:
                await spectrum_analyzer.set_trace_mode_to_normal(1)
            except Exception:
                pass

        return results

    # ── helpers ──────────────────────────────────────────────────────────────

    def _get_tones_khz_for_code(self, deps: CalibrationDependencies, code: str) -> list[float]:
        try:
            tx = deps.transmitter_repo.get_by_code(code)
        except Exception:
            tx = None
        if tx is None:
            return []
        try:
            tx_dict: dict[str, Any] = tx.model_dump()  # type: ignore[attr-defined]
        except Exception:
            return []
        details = tx_dict.get("modulation_details") or {}
        sub_carriers = details.get("sub_carriers") or []
        tones: list[float] = []
        for entry in sub_carriers:
            if isinstance(entry, list) and len(entry) > 0:
                value = entry[0]
            else:
                value = entry
            try:
                tone = float(value)
            except Exception:
                continue
            if tone > 0:
                tones.append(tone)
        return tones

    @staticmethod
    def _sideband_level_to_mod_index(sideband_level_wrt_carrier: float) -> float | None:
        """Map a sideband-level (dB w.r.t. carrier) to a mod-index using a Bessel J0/J1 chart."""
        try:
            chart: dict[float, float] = {}
            for mi in np.arange(0.01, 2.0, 0.01):
                j0 = float(bessel.j0(mi))
                j1 = float(bessel.j1(mi))
                if j0 <= 0.0 or j1 <= 0.0:
                    continue
                key = round(20.0 * math.log10(j0) - 20.0 * math.log10(j1), 2)
                chart[key] = round(float(mi), 2)
            if len(chart) == 0:
                return None
            target = float(sideband_level_wrt_carrier)
            differences = {round(abs(k + target), 2): k for k in chart.keys()}
            best_key = chart.get(differences.get(min(differences.keys())))  # type: ignore[arg-type]
            return float(best_key) if best_key is not None else None
        except Exception:
            return None

    def _compute_mod_index(
        self,
        trace_data: np.ndarray,
        span_mhz: float,
        tone_hz: float,
    ) -> tuple[float, float, float] | None:
        """Locate sidebands at +/- tone offset and compute averaged mod index."""
        if tone_hz <= 0:
            return None
        n = int(trace_data.size)
        if n < 8:
            return None
        center_idx = n // 2
        frsp_bin_mhz = float(span_mhz) / float(n)  # MHz per bin
        if frsp_bin_mhz <= 0:
            return None
        window_bins = max(1, int((_SIDEBAND_WINDOW_HZ / 1e6) / frsp_bin_mhz))
        half_window = max(1, window_bins // 2)
        offset_bins = int((tone_hz / 1e6) / frsp_bin_mhz)
        if offset_bins <= 0:
            return None
        upper_center = center_idx + offset_bins
        lower_center = center_idx - offset_bins
        if upper_center + half_window >= n or lower_center - half_window < 0:
            return None
        cf_peak = float(np.max(trace_data))
        sb_upper = float(np.max(trace_data[upper_center - half_window: upper_center + half_window]))
        if abs(abs(cf_peak) - abs(sb_upper)) > 20.0:
            return None
        sb_lower = float(np.max(trace_data[lower_center - half_window: lower_center + half_window]))
        upper_rel = sb_upper - cf_peak
        lower_rel = sb_lower - cf_peak
        mi_upper = self._sideband_level_to_mod_index(upper_rel)
        mi_lower = self._sideband_level_to_mod_index(lower_rel)
        if mi_upper is None or mi_lower is None:
            return None
        return round(upper_rel, 2), round(lower_rel, 2), (mi_upper + mi_lower) / 2.0

    def _make_result(
        self,
        selected: MeasureSelectedRow,
        code: str,
        port: str,
        frequency_label: str,
        frequency: float,
        measured_value: float,
        final_value: float,
        status: str,
        message: str,
    ) -> MeasureRunResultRow:
        return MeasureRunResultRow(
            system_kind=selected.system_kind,  # type: ignore[arg-type]
            code=code,
            port=port,
            frequency_label=frequency_label,
            frequency=frequency,
            parameter="modulation_index",
            measured_value=measured_value,
            applied_loss=0.0,
            final_value=final_value,
            status=status,
            message=message,
            timestamp=datetime.now(timezone.utc),
        )
