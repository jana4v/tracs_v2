#BHE C-Band Only
from .BaseClass import TtcTransmitter, Specification
from ..factory import factory
from ..Models import InstrumentTypes, Limits, Units, InterfaceProtocols
import math
import logging
from ..InstrumentBaseClass import apply_exceptions_and_traceback_to_all_methods, catch_exceptions_and_traceback

logger = logging.getLogger("BHE")


@apply_exceptions_and_traceback_to_all_methods(catch_exceptions_and_traceback)
# default port number 11111
# call preset_instrument before calling any method
class BHE(TtcTransmitter):
    def __init__(self) -> None:
        super().__init__()
        self.supported_protocol = InterfaceProtocols.LAN.value
        self.specification: Specification = self.initialize_specifications()

    def initialize_specifications(self) -> Specification:
        specification = Specification()
        specification.frequency = Limits(lower_limit=6300, upper_limit=6600, units=Units.MHz)
        specification.tone_generator_frequency = Limits(lower_limit=0.0001, upper_limit=0.1, units=Units.MHz)
        specification.tone_generator_amplitude = Limits(lower_limit=0, upper_limit=5, units=Units.volts)
        specification.cariier_Level = Limits(lower_limit=-70, upper_limit=5, units=Units.dBm)
        specification.PMmod_index = Limits(lower_limit=0, upper_limit=2.5, units=Units.count)
        specification.FM_deviation = Limits(lower_limit=100, upper_limit=600, units=Units.kHz)
        return specification

    def _statusupdate(self, status):
        logger.info(status)

    def decode_to_hex(self, value, number_of_bytes=1):
        code = ''
        for i in reversed(range(number_of_bytes)):
            temp = hex(math.trunc((value / pow(256, i)) % 256)).upper()
            temp = temp[2:]
            if len(temp) == 1:
                temp = '0' + temp
            code = code + temp
        code = code + '\r'
        return code

    def preset_instrument(self):
        self.set_remote_mode()

    def set_power_level(self, power):
        self.check_limit(self.specification.cariier_Level, power)
        attenuation = 2 * (5 - power)
        attenuation = hex(int(round(attenuation, 0)))
        attenuation = attenuation[2:]
        attenuation = attenuation.upper()
        if len(attenuation) == 1:
            attenuation = '0' + attenuation
        code = '*ATTN=' + attenuation + '\r'
        result = self.command(code, read_operation=True, buffer_length=100)
        self._statusupdate(result)
        return self

    def set_frequency(self, frequency):
        self.check_limit(self.specification.frequency, frequency)
        frequency = frequency * 1000
        code = '*FREQ=' + self.decode_to_hex(frequency, 3)
        result = self.command(code, read_operation=True, buffer_length=100)
        self._statusupdate(result)
        return self

    def set_rf_on(self, onoff=1):
        if onoff == 1:
            status = 'Setting RF ON state'
            code = '*RFON=01\r'
        else:
            status = 'Setting RF OFF state'
            code = '*RFON=00\r'
        self._statusupdate(status)
        result = self.command(code, read_operation=True, buffer_length=100)
        self._statusupdate(result)
        return self

    def set_rf_off(self, onoff=1):
        self._statusupdate('Setting RF OFF state')
        result = self.command('*RFON=00\r', read_operation=True, buffer_length=100)
        self._statusupdate(result)
        return self

    def set_tone_generator_frequency(self, frequency):
        self.check_limit(self.specification.tone_generator_frequency, frequency / 1000.0)
        frequency = frequency * 100
        code = '*FGEN=' + self.decode_to_hex(frequency, 2)
        result = self.command(code, read_operation=True, buffer_length=100)
        self._statusupdate(result)
        return self

    def set_tone_generator_amplitude(self, Amplitude):
        self.check_limit(self.specification.tone_generator_amplitude, Amplitude)
        Amplitude = hex(int(Amplitude * (100 / 5)))
        Amplitude = Amplitude[2:]
        if len(Amplitude) == 1:
            Amplitude = '0' + Amplitude
        code = '*AGEN=' + str(Amplitude) + '\r'
        result = self.command(code, read_operation=True, buffer_length=100)
        self._statusupdate(result)
        return self

    def set_tone_generator_state_on_off(self, onoff=1):
        code = '*EGEN=01\r' if onoff == 1 else '*EGEN=00\r'
        result = self.command(code, read_operation=True, buffer_length=100)
        self._statusupdate(result)
        return self

    def set_modulation_type(self, modulation_type='PM', source=1):
        if modulation_type == 'PM':
            code = '*MODT=02\r'
        elif modulation_type == 'FM':
            code = '*MODT=01\r'
        else:
            code = '*MODT=00\r'
        result = self.command(code, read_operation=True, buffer_length=100)
        self._statusupdate(result)
        return self

    def set_pm_mod_index(self, mod_index):
        self.check_limit(self.specification.PMmod_index, mod_index)
        modIndex = math.floor(mod_index * 20)
        code = '*PMOD=' + str(modIndex) + '\r'
        result = self.command(code, read_operation=True, buffer_length=100)
        self._statusupdate(result)
        return self

    def set_fm_mod_deviation(self, deviation):
        self.check_limit(self.specification.FM_deviation, deviation)
        code = '*FMOD=' + self.decode_to_hex(deviation, 2)
        result = self.command(code, read_operation=True, buffer_length=100)
        self._statusupdate(result)
        return self

    def set_remote_mode(self):
        result = self.command('*REMO=\r', read_operation=True, buffer_length=100)
        self._statusupdate(result)
        return self

    def set_modulation_on(self):
        self.set_tone_generator_state_on_off(onoff=1)
        return self

    def set_modulation_off(self):
        self.set_tone_generator_state_on_off(onoff=0)
        return self

    def set_fm_modulation(self, deviation, fmRate=32, fmSource=1):
        self.set_modulation_type(modulation_type='FM')
        self.set_fm_mod_deviation(deviation)
        self.set_tone_generator_frequency(fmRate)
        self.set_tone_generator_amplitude(1)
        self.set_tone_generator_state_on_off(1)
        return self


factory.register_component(InstrumentTypes.TtcTransmitter, "BHE_C", BHE)
