from abc import ABC,abstractmethod
from ..InstrumentBaseClass import InstrumentBaseClass
from ..Models import Limits,InterfaceProtocols
from pydantic import BaseModel

class Specification(BaseModel):
    frequency: Limits | None = None
    level: Limits | None = None
    sweep_span: Limits | None = None 
    sweep_start: Limits | None = None 
    sweep_stop: Limits | None = None
    sweep_time: Limits | None = None
    sweep_dwell:Limits | None = None
    sweep_points :Limits | None = None
    pm_deviation :Limits | None = None
    fm_deviation :Limits | None = None
    pm_rate :Limits | None = None
    fm_rate :Limits | None = None

class SignalGenerator(ABC, InstrumentBaseClass):
    def __init__(self):
        super().__init__()
        self.supported_protocol = InterfaceProtocols.ANY.value

    @abstractmethod
    def preset_instrument(self):
        pass

    @abstractmethod
    def get_instrument_name(self):
        pass

    def set_frequency(self, frequency: float):
        pass

    def get_frequency(self):
        pass

    def set_power_level(self, level: float):
        pass

    def set_rf_on(self):
        pass
        
    def set_rf_off(self):
        pass

    def set_sweep_ramp_params(self, freq_start: float, freq_stop: float, sweep_time: float):
        pass

    def start_freq_sweep(self):
        pass

    def set_sweep_step_params(self, freq_start: float, freq_stop: float, sweep_dwell: float, sweep_points: int):
        pass

    def set_sweep_list_params(self, freq_params, pow_params, dwell_params):
        pass

    def start_list_sweep(self):
        pass

    def set_phase_modulation_tone_one(self, pm_deviation: float, pm_rate: float, pm_source: int = 1):
        pass

    def set_phase_modulation_tone_two(self, pm_deviation: float, pm_source: int, pm_rate: float):
        pass

    def set_freq_modulation_tone_one(self, fm_deviation: float, fm_rate: float = 32, fm_source: int = 1):
        pass

    def set_freq_modulation_tone_two(self, fm_deviation: float, fm_source: int, fm_rate: float):
        pass

    def set_modulation_on(self):
        pass

    def set_modulation_off(self):
        pass
        

