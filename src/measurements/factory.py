from __future__ import annotations

from ._power import PowerMeasureProcedure
from .base import MeasureProcedure


class MeasureProcedureFactory:
    def __init__(self) -> None:
        self._procedures: dict[str, MeasureProcedure] = {
            "power": PowerMeasureProcedure(),
        }

    def get(self, parameter_name: str) -> MeasureProcedure | None:
        return self._procedures.get(str(parameter_name or "").strip().lower())
