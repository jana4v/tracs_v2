from abc import ABC,abstractmethod
from typing import Optional
from ..InstrumentBaseClass import InstrumentBaseClass
from ..Models import Limits, InterfaceProtocols
from pydantic import BaseModel



class Specification(BaseModel):
    frequency:Limits|None = None
    tone_generator_frequency:Limits|None = None
    tone_generator_amplitude:Limits|None = None
    cariier_Level:Limits|None = None
    PMmod_index:Limits|None = None
    FM_deviation:Limits|None = None
    

class TtcTransmitter(ABC, InstrumentBaseClass):
    def __init_subclass__(cls, model_key: str = "", **kwargs):
        super().__init_subclass__(**kwargs)
        if model_key:
            from ..factory import factory
            from ..Models import InstrumentTypes
            factory.register_component(InstrumentTypes.TtcTransmitter, model_key, cls)

    def __init__(self):
        super().__init__()
        self.supported_protocol = InterfaceProtocols.ANY.value

    @abstractmethod
    def preset_instrument(self):
        pass

    @abstractmethod    
    def set_power_level(self,power):
        pass
        
    @abstractmethod
    def set_frequency(self,frequency):
        pass
        
    @abstractmethod
    def set_rf_on(self,onoff=1):
        pass

    @abstractmethod
    def set_rf_off(self,onoff=1):
        pass

    @abstractmethod
    def set_tone_generator_frequency(self,frequency):
        pass

    @abstractmethod
    def set_tone_generator_amplitude(self,Amplitude):
        pass

    @abstractmethod
    def set_tone_generator_state_on_off(self,onoff=1):
        pass

    @abstractmethod
    def set_modulation_type(self,modulationType='PM',source=1):
        pass

    @abstractmethod
    def set_pm_mod_index(self,modIndex):
        pass

    @abstractmethod
    def set_fm_mod_deviation(self,deviation):
        pass

    @abstractmethod
    def set_modulation_on(self):
        pass

    @abstractmethod
    def set_modulation_off(self):
        pass

    @abstractmethod
    def set_fm_modulation(self,deviation,fmRate=32,fmSource=1):
        pass

