from __future__ import annotations

import importlib
import pkgutil
from typing import Dict

from .base import CalibrationProcedure


class CalibrationProcedureFactory:
    def __init__(self) -> None:
        self._procedures: Dict[str, CalibrationProcedure] = {}
        self._loaded = False

    def register(self, procedure: CalibrationProcedure) -> None:
        key = self.normalize_cal_type(procedure.cal_type)
        if not key:
            raise ValueError("Calibration procedure must define a cal_type")
        self._procedures[key] = procedure

    def ensure_loaded(self) -> None:
        if self._loaded:
            return

        package_name = "src.calibration"
        package = importlib.import_module(package_name)
        for module_info in pkgutil.iter_modules(package.__path__):
            if not module_info.name.startswith("_"):
                continue
            module = importlib.import_module(f"{package_name}.{module_info.name}")
            procedure_class = getattr(module, "PROCEDURE_CLASS", None)
            if procedure_class is None:
                continue
            self.register(procedure_class())

        self._loaded = True

    def get(self, cal_type: str) -> CalibrationProcedure | None:
        self.ensure_loaded()
        return self._procedures.get(self.normalize_cal_type(cal_type))

    @staticmethod
    def normalize_cal_type(cal_type: str) -> str:
        text = str(cal_type or "").strip().lower()
        aliases = {
            "calsg": "cal_sg",
            "calsg": "cal_sg",
            "sgcal": "cal_sg",
            "cal_sg": "cal_sg",
            "injectcal": "inject_cal",
            "inject_cal": "inject_cal",
            "fixedpad": "fixed_pad",
            "tvacref": "tvac_ref",
        }
        return aliases.get(text, text)


procedure_factory = CalibrationProcedureFactory()