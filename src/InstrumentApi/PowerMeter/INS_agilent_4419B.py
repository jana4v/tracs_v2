import logging,time
from .BaseClass import PowerMeter,Specification
from ..Models import InstrumentTypes,Limits,Units
from ..factory import factory
from ..InstrumentBaseClass import apply_exceptions_and_traceback_to_all_methods,catch_exceptions_and_traceback


@apply_exceptions_and_traceback_to_all_methods(catch_exceptions_and_traceback)
class AG_4419B(PowerMeter):
    def __init__(self) -> None:
        super().__init__()
        self.specification: Specification = self.initialize_specifications()

    def initialize_specifications(self) -> Specification:
        specification = Specification()
        specification.frequency = Limits(lower_limit=0, upper_limit=22000, units=Units.MHz)
        specification.channel_number = Limits(lower_limit=1, upper_limit=2, units=Units.count)
        specification.channel_limits = Limits(lower_limit=-90, upper_limit=90, units=Units.dBm)
        specification.display_offset = Limits(lower_limit=-100, upper_limit=100, units=Units.count)
        return specification

    def get_instrument_name(self):
        data = self.command("*IDN?", read_operation=True)
        if len(data) < 2:
            raise Exception('GPIB ERROR, Failed to Communicate Power Meter AG_4419B')

    def preset_instrument(self):
        upper_limit = 30
        lower_limit = -50

        code = ''
        for channel_number in [1, 2]:
            code += f':SENS{channel_number}:LIM:STAT ON;'
            code += f':SENS{channel_number}:LIM:UPP:DATA {upper_limit} DBM;'
            code += f':SENS{channel_number}:LIM:LOW:DATA {lower_limit} DBM;'
            code += f':CALC{channel_number}:GAIN 0;'

        code = "SYST:PRES;" + code
        self.command(code)
        return self

    def set_channel_frequency(self, channel_frequency, channel_number=1):
        self.check_limit(self.specification.frequency, channel_frequency)
        self.check_limit(self.specification.channel_number, channel_number)
        code = f':SENS{channel_number}:FREQ {channel_frequency} MHz'
        self.command(code)
        return self

    def set_channel_limits(self, channel_number=1, lower_limit=-50, upper_limit=50):
        self.check_limit(self.specification.channel_limits, lower_limit)
        self.check_limit(self.specification.channel_limits, upper_limit)
        self.check_limit(self.specification.channel_number, channel_number)
        code = f':SENS{channel_number}:LIM:STAT ON;'
        code += f':SENS{channel_number}:LIM:UPP:DATA {upper_limit} DBM;'
        code += f':SENS{channel_number}:LIM:LOW:DATA {lower_limit} DBM'
        self.command(code)
        return self

    def set_channel_display_offset_value(self, offset_value, channel_number=1):
        self.check_limit(self.specification.display_offset, offset_value)
        self.check_limit(self.specification.channel_number, channel_number)
        code = f':CALC{channel_number}:GAIN {offset_value}'
        self.command(code)
        return self

    def get_channel_power(self, channel_number=1):
        self.check_limit(self.specification.channel_number, channel_number)
        code = f':FETCh{channel_number}?'
        value1 = float(self.command(code, read_operation=True))
        previous_value = 0.0
        count = 0
        while abs(value1 - previous_value) > 0.1 and count < 4:
            previous_value = value1
            time.sleep(0.5)
            value1 = float(self.command(code, read_operation=True))
            count += 1
        return value1

    def get_channel_frequency(self, channel_number=1):
        self.check_limit(self.specification.channel_number, channel_number)
        code = f':SENS{channel_number}:FREQ?'
        return float(self.command(code, read_operation=True))

    def get_channel_display_offset(self, channel_number=1):
        self.check_limit(self.specification.channel_number, channel_number)
        code = f':CALC{channel_number}:GAIN?'
        return float(self.command(code, read_operation=True))
    
factory.register_component(InstrumentTypes.PowerMeter,"AG_4419B",AG_4419B)
factory.register_component(InstrumentTypes.PowerMeter,"E4419B",AG_4419B)