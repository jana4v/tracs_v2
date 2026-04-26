from __future__ import annotations

from abc import ABC, abstractmethod
from dataclasses import dataclass

from src.repositories.cal_sg_calibration_repo import CalSgCalibrationRepository
from src.repositories.inject_cal_calibration_repo import InjectCalCalibrationRepository
from src.repositories.test_systems_repo import TestSystemsRepository
from src.repositories.transmitter_repo import TransmitterRepository


@dataclass(slots=True)
class CalibrationDependencies:
    transmitter_repo: TransmitterRepository
    test_systems_repo: TestSystemsRepository
    cal_sg_repo: CalSgCalibrationRepository
    inject_cal_repo: InjectCalCalibrationRepository


class CalibrationProcedure(ABC):
    cal_type: str = ""

    @abstractmethod
    async def execute(self, service, runtime, payload, deps: CalibrationDependencies) -> None:
        raise NotImplementedError