from abc import ABC,abstractmethod
from typing import Optional
from ..InstrumentBaseClass import InstrumentBaseClass
from ..Models import Limits,InterfaceProtocols
from pydantic import BaseModel


class Specification(BaseModel):
    pass


class SwitchDriver(ABC, InstrumentBaseClass):
    def __init__(self):
        super().__init__()
        self.supported_protocol = InterfaceProtocols.ANY.value

    @abstractmethod
    def get_switch_positions(self):
        pass

    @abstractmethod
    def set_switch_position(self, switch_position: str):
        pass

    @abstractmethod
    def validate_switch_positions(self):
        pass

    