from abc import ABC,abstractmethod
from typing import List, Optional
from ..InstrumentBaseClass import InstrumentBaseClass
from ..Models import Limits,InterfaceProtocols
from pydantic import BaseModel


class Specification(BaseModel):
    name: Optional[str] = None
    valid_positions: Optional[List[str]] = None
    port_count: Optional[int] = None
    topology: Optional[str] = None


class SwitchDriver(ABC, InstrumentBaseClass):
    def __init_subclass__(cls, model_key: str = "", **kwargs):
        super().__init_subclass__(**kwargs)
        if model_key:
            from ..factory import factory
            from ..Models import InstrumentTypes
            factory.register_component(InstrumentTypes.SwitchDriver, model_key, cls)

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

    