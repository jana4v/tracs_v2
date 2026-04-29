from __future__ import annotations

from dataclasses import dataclass

from src.calibration.base import CalibrationDependencies
from src.schemas.calibration_data import MeasureRunResultRow, MeasureRunStartRequest, MeasureTableRow


@dataclass(slots=True)
class MeasureSelectedRow:
    system_kind: str
    row: MeasureTableRow


class MeasureProcedure:
    parameter_name: str = ""

    async def execute(
        self,
        payload: MeasureRunStartRequest,
        deps: CalibrationDependencies,
        selected_rows: list[MeasureSelectedRow],
    ) -> list[MeasureRunResultRow]:
        raise NotImplementedError
