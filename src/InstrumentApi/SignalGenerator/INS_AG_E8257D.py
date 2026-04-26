
from ..factory import factory
from ..Models import InstrumentTypes,Limits,Units
from .BaseClass import SignalGenerator, Specification
from ..InstrumentBaseClass import apply_exceptions_and_traceback_to_all_methods,catch_exceptions_and_traceback


@apply_exceptions_and_traceback_to_all_methods(catch_exceptions_and_traceback)
class E8257D(SignalGenerator):
    def __init__(self) -> None:
        super().__init__()
        self.specification: Specification = self.initialize_specifications()

    def initialize_specifications(self) -> Specification:
        specification = Specification()
        specification.frequency = Limits(lower_limit=0.25, upper_limit=40000, units=Units.MHz)
        specification.sweep_span = Limits(lower_limit=0.0001, upper_limit=39990, units=Units.MHz)
        specification.sweep_start = Limits(lower_limit=10, upper_limit=40000, units=Units.MHz)
        specification.sweep_stop = Limits(lower_limit=10.0001, upper_limit=40000, units=Units.MHz)
        specification.level = Limits(lower_limit=-130, upper_limit=25, units=Units.dBm)
        specification.sweep_time = Limits(lower_limit=1, upper_limit=200, units=Units.seconds)
        specification.sweep_dwell = Limits(lower_limit=1, upper_limit=60000, units=Units.milli_seconds)
        specification.sweep_points = Limits(lower_limit=2, upper_limit=65535, units=Units.count)
        specification.pm_deviation = Limits(lower_limit=0, upper_limit=320, units=Units.radians)
        specification.fm_deviation = Limits(lower_limit=0, upper_limit=32000, units=Units.kHz)
        specification.pm_rate = Limits(lower_limit=0, upper_limit=100, units=Units.kHz)
        specification.fm_rate = Limits(lower_limit=0, upper_limit=100, units=Units.kHz)
        return specification
    
    def preset_instrument(self):
        code = 'SYST:PRES;:OUTP OFF;:OUTP:MOD OFF;:FREQ:MODE CW;:FREQ 3GHZ;:POW:ATT:AUTO ON;:SWE:POIN 1003;:POW -80DBM'
        self.command(code)
        return self

    def get_instrument_name(self):
        data = ''
        data = self.command("*IDN?", read_operation=True)
        if len(data) < 2:
            raise Exception('GPIB ERROR, Failed to Communicate Signal generator 8257D')
         
    def set_frequency(self, frequency_MHz: float):
        self.check_limit(self.specification.frequency, frequency_MHz)
        code = f':POW:MODE FIX;:FREQ:MODE FIX;:FREQ:MODE CW;:FREQ {frequency_MHz} MHz'
        self.command(code)
        return self
    
    def get_frequency(self):
        code = ':FREQ:CW?'
        return self.command(code, read_operation=True)

    def set_power_level(self, power_level_dbm: float):
        self.check_limit(self.specification.level, power_level_dbm)
        code = f':POW:ALC:SOUR INT;:POW:ATT:AUTO ON;:POW {power_level_dbm} DBM;'
        self.command(code)
        return self      
        
    def set_rf_on(self):
        code = 'OUTP ON'
        self.command(code)
        return self   
        
    def set_rf_off(self):
        code = 'OUTP OFF'
        self.command(code)
        return self   
    
    def set_sweep_ramp_params(self, freq_start, freq_stop, sweep_time):
        self.check_limit(self.specification.sweep_start, freq_start)
        self.check_limit(self.specification.sweep_stop, freq_stop)
        self.check_limit(self.specification.sweep_time, sweep_time)
        code = f""":FREQ:MODE FIX;\
        :SWE:GEN ANAL;\
        :FREQ:START {str(freq_start)} MHz;\
        :FREQ:STOP {str(freq_stop)} MHz;\
        :SWEEP:TIME {str(sweep_time)} s;\
        :SWEEP:MODE AUTO;"""
        self.command(code)
        self.start_freq_sweep()
        return self

    def start_freq_sweep(self):
        self.command(':FREQ:MODE SWE')
        return self

    def set_sweep_step_params(self, freq_start, freq_stop, sweep_dwell, sweep_points):
        self.check_limit(self.specification.sweep_start, freq_start)
        self.check_limit(self.specification.sweep_stop, freq_stop)
        self.check_limit(self.specification.sweep_dwell, sweep_dwell)
        self.check_limit(self.specification.sweep_points, sweep_points)
        sweep_dwell = float(sweep_dwell) / 1000
        
        code = f""":FREQ:MODE CW;\
        :SWE:GEN STEP;\
        :FREQ:STAR {str(freq_start)} MHz;\
        :FREQ:STOP {str(freq_stop)} MHz;\
        :SWE:DWEL {str(sweep_dwell)} s;\
        :SWE:POIN {str(sweep_points)} ;\
        :SWE:MODE AUTO;"""
        self.command(code)
        self.start_freq_sweep()
        return self
        
    def set_sweep_list_params(self, freq_params, pow_params, dwell_params):
        freq_lst = ''
        pow_lst = ''
        dwell_lst = ''
        
        for i in freq_params:
            self.check_limit(self.specification.sweep_start, i)
            self.check_limit(self.specification.sweep_time, i)
            freq_lst = freq_lst + str(i) + 'MHz' + ','
                     
        for i in pow_params:
            self.check_limit(self.specification.level, i)
            pow_lst = pow_lst + str(i) + ','
            
        for i in dwell_params:
            self.check_limit(self.specification.sweep_dwell, i)
            dwell_lst = dwell_lst + str(float(i) / 1000) + ','
            
        freq_lst = freq_lst.rstrip(',')
        pow_lst = pow_lst.rstrip(',')
        dwell_lst = dwell_lst.rstrip(',')
        
        # Presets the previous list
        code = ':LIST:TYPE:LIST:INIT:PRES' 
        code = code + f';:LIST:FREQ {freq_lst}'
        code = code + f';:LIST:POW {pow_lst}'
        code = code + f';:LIST:DWEL {dwell_lst}'

        self.command(code)
        self.start_list_sweep()
        return self
        
    def start_list_sweep(self):
        code = ':FREQ:MODE LIST;:POW:MODE LIST;:LIST TYPE LIST'
        self.command(code)
        return self

    def set_phase_modulation_tone_one(self, pm_deviation, pm_rate, pm_source=1):
        self.check_limit(self.specification.pm_deviation, pm_deviation)
        self.check_limit(self.specification.pm_rate, pm_rate)
        
        code = ':PM1:BAND NORM'

        pm_source_map = {1: 'INT1', 2: 'INT2', 3: 'EXT1', 4: 'EXT2'}
        selected_source = pm_source_map.get(pm_source)
        if selected_source is None:
            raise Exception("Incorrect pm_source specified")

        if pm_source > 2:
            code = f';:PM1:{selected_source}:COUP AC'

        code += f';:PM1:SOUR {selected_source}'
        code += f';:PM1 {str(pm_deviation)} RAD'

        if pm_source < 3:
            code += f';:PM1:{selected_source}:FREQ {str(pm_rate)} KHZ'
            code += f';:PM1:{selected_source}:FUNC:SHAP SINE'

        code += ';:PM1:STAT 1'

        self.command(code)
        self.set_modulation_on()
        return self

    def set_phase_modulation_tone_two(self, pm_deviation, pm_source, pm_rate):
        self.check_limit(self.specification.pm_deviation, pm_deviation)
        self.check_limit(self.specification.pm_rate, pm_rate)
        
        code = ':PM2:BAND NORM'

        pm_source_map = {1: 'INT1', 2: 'INT2', 3: 'EXT1', 4: 'EXT2'}
        selected_source = pm_source_map.get(pm_source)
        if selected_source is None:
             raise Exception("Incorrect pm_source specified")

        if pm_source > 2:
            code = f':PM2:{selected_source}:COUP AC'

        code += f';:PM2:SOUR {selected_source}'
        code += f';:PM2 {str(pm_deviation)} RAD'

        if pm_source < 3:
            code += f';:PM2:{selected_source}:FREQ {str(pm_rate)} KHZ'
            code += f';:PM2:{selected_source}:FUNC:SHAP SINE'

        code += ';:PM2:STAT 1'

        self.command(code)
        self.set_modulation_on()
        return self
    
    def set_freq_modulation_tone_one(self, fm_deviation, fm_rate=32, fm_source=1):
        self.check_limit(self.specification.fm_deviation, fm_deviation)
        self.check_limit(self.specification.fm_rate, fm_rate)
        
        code = f':FM1 {str(fm_deviation)} KHZ'
        
        fm_source_map = {1: 'INT1', 2: 'INT2', 3: 'EXT1', 4: 'EXT2'}
        selected_source = fm_source_map.get(fm_source)
        if selected_source is None:
            raise Exception  ("Incorrect fm_source specified")

        if fm_source > 2:
            code += f';:FM1:{selected_source}:COUP AC'

        code += f';:FM1:SOUR {selected_source}'

        if fm_source < 3:
            code += f';:FM1:{selected_source}:FREQ {str(fm_rate)} KHZ'
            code += f';:FM1:{selected_source}:FUNC:SHAP SINE'

        code += ';:FM1:STAT 1'

        self.command(code)
        self.set_modulation_on()
        return self
    
    def set_freq_modulation_tone_two(self, fm_deviation, fm_source, fm_rate):
        self.check_limit(self.specification.fm_deviation, fm_deviation)
        self.check_limit(self.specification.fm_rate, fm_rate)
        
        code = f':FM2 {str(fm_deviation)} RAD'
        
        fm_source_map = {1: 'INT1', 2: 'INT2', 3: 'EXT1', 4: 'EXT2'}
        selected_source = fm_source_map.get(fm_source)
        if selected_source is None:
            raise Exception("Incorrect fm_source specified")

        if fm_source > 2:
            code = f';:FM2:{selected_source}:COUP AC'

        code += f';:FM2:SOUR {selected_source}'

        if fm_source < 3:
            code += f';:FM2:{selected_source}:FREQ {str(fm_rate)} KHZ'
            code += f';:FM2:{selected_source}:FUNC:SHAP SINE'

        code += ';:FM2:STAT 1'

        self.command(code)
        self.set_modulation_on()
        return self

    def set_modulation_on(self):
        code = ':OUTP:MOD 1'
        self.command(code)
        return self

    def set_modulation_off(self):
        code = ':OUTP:MOD 0'
        self.command(code)
        return self


factory.register_component(InstrumentTypes.SignalGenerator,"E8257D",E8257D)