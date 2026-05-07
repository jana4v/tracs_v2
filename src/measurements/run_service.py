from __future__ import annotations

from datetime import datetime, timezone

from src.calibration.base import CalibrationDependencies
from src.schemas.calibration_data import MeasureRunResultRow, MeasureRunStartRequest, MeasureRunStartResponse

from .base import MeasureSelectedRow
from .factory import MeasureProcedureFactory
from ._power import PowerMeasureProcedure


class MeasureRunService:
    def __init__(self) -> None:
        self._factory = MeasureProcedureFactory()

    async def start_run(
        self,
        payload: MeasureRunStartRequest,
        deps: CalibrationDependencies,
    ) -> MeasureRunStartResponse:
        selected_rows = self._collect_selected_rows(payload)
        requested_parameters = self._collect_requested_parameters(selected_rows)

        results: list[MeasureRunResultRow] = []
        for parameter in requested_parameters:
            procedure = self._factory.get(parameter)
            if parameter == "power" and isinstance(procedure, PowerMeasureProcedure):
                missing_downlink = procedure.get_missing_downlink_channels(payload, deps, selected_rows)
                if len(missing_downlink) > 0 and not bool(payload.continue_on_missing_downlink_cal):
                    return MeasureRunStartResponse(
                        message="Downlink cal not present for one or more selected channels.",
                        processed_rows=0,
                        requires_confirmation=True,
                        missing_downlink_channels=missing_downlink,
                        requested_parameters=requested_parameters,
                        results=[],
                    )

                if len(missing_downlink) > 0 and bool(payload.continue_on_missing_downlink_cal):
                    skip_keys = {
                        (
                            str(item.code or "").strip(),
                            str(item.port or "").strip(),
                            f"{float(item.frequency):.6f}",
                        )
                        for item in missing_downlink
                    }
                    def _row_key(selected: MeasureSelectedRow) -> tuple[str, str, str]:
                        row = selected.row
                        return (
                            str(row.code or "").strip(),
                            str(row.port or "").strip(),
                            f"{self._safe_frequency(str(row.frequency or '') ):.6f}",
                        )

                    selected_rows = [
                        selected
                        for selected in selected_rows
                        if not (
                            bool(selected.row.power_selected)
                            and _row_key(selected) in skip_keys
                        )
                    ]

            if procedure is None:
                for selected in selected_rows:
                    row = selected.row
                    is_selected = {
                        "frequency": bool(row.frequency_selected),
                        "modulation_index": bool(row.modulation_index_selected),
                        "spurious": bool(row.spurious_selected),
                    }.get(parameter, False)
                    if not is_selected:
                        continue
                    results.append(
                        MeasureRunResultRow(
                            system_kind=selected.system_kind,  # type: ignore[arg-type]
                            code=str(row.code or "").strip(),
                            port=str(row.port or "").strip(),
                            frequency_label=str(row.frequency_label or "").strip(),
                            frequency=self._safe_frequency(row.frequency),
                            parameter=parameter,
                            measured_value=0.0,
                            applied_loss=0.0,
                            final_value=0.0,
                            status="skipped",
                            message=f"Procedure not implemented yet for parameter '{parameter}'",
                            timestamp=datetime.now(timezone.utc),
                        )
                    )
                continue

            proc_results = await procedure.execute(payload, deps, selected_rows)
            results.extend(proc_results)

        return MeasureRunStartResponse(
            message="Measure run executed",
            processed_rows=len(results),
            requires_confirmation=False,
            missing_downlink_channels=[],
            requested_parameters=requested_parameters,
            results=results,
        )

    def _collect_selected_rows(self, payload: MeasureRunStartRequest) -> list[MeasureSelectedRow]:
        selected: list[MeasureSelectedRow] = []
        for row in payload.transmitter_rows:
            selected.append(MeasureSelectedRow(system_kind="transmitter", row=row))
        for row in payload.receiver_rows:
            selected.append(MeasureSelectedRow(system_kind="receiver", row=row))
        for row in payload.transponder_rows:
            selected.append(MeasureSelectedRow(system_kind="transponder", row=row))
        return selected

    def _collect_requested_parameters(self, selected_rows: list[MeasureSelectedRow]) -> list[str]:
        order = ["power", "frequency", "modulation_index", "spurious", "command_threshold", "ranging_threshold"]
        requested: list[str] = []
        for parameter in order:
            if parameter == "power" and any(bool(s.row.power_selected) for s in selected_rows):
                requested.append(parameter)
            if parameter == "frequency" and any(bool(s.row.frequency_selected) for s in selected_rows):
                requested.append(parameter)
            if parameter == "modulation_index" and any(bool(s.row.modulation_index_selected) for s in selected_rows):
                requested.append(parameter)
            if parameter == "spurious" and any(bool(s.row.spurious_selected) for s in selected_rows):
                requested.append(parameter)
            if parameter == "command_threshold" and any(bool(getattr(s.row, "command_threshold_selected", False)) for s in selected_rows):
                requested.append(parameter)
            if parameter == "ranging_threshold" and any(bool(getattr(s.row, "ranging_threshold_selected", False)) for s in selected_rows):
                requested.append(parameter)
        return requested

    def _safe_frequency(self, value: str) -> float:
        try:
            return float(str(value or "").strip())
        except Exception:
            return 0.0
