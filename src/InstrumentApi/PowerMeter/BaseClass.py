from abc import ABC,abstractmethod
from typing import Optional
from ..InstrumentBaseClass import InstrumentBaseClass
from ..Models import Limits,InterfaceProtocols
from pydantic import BaseModel
 

class Specification(BaseModel):
    model_number: Optional[str] = None
    frequency: Optional[Limits] = None
    channel_number: Optional[Limits] = None
    channel_limits: Optional[Limits] = None
    display_offset: Optional[Limits] = None
    

class PowerMeter(ABC, InstrumentBaseClass):
    def __init__(self):
        super().__init__()
        self.supported_protocol = InterfaceProtocols.ANY.value
    @abstractmethod    
    def preset_instrument(self):
        pass

    @abstractmethod    
    def set_channel_frequency(self, channel_frequency, channel_number=1):
        pass
        
    @abstractmethod
    def set_channel_limits(self, channel_number=1, lower_limit=-50, upper_limit=50):
        pass
        
    @abstractmethod
    def set_channel_display_offset_value(self, offset_value, channel_number=1):
        pass

    @abstractmethod
    def get_channel_power(self, channel_number=1):
        pass

    @abstractmethod
    def get_channel_frequency(self, channel_number=1):
        pass

    @abstractmethod
    def get_channel_display_offset(self, channel_number=1):
        pass



