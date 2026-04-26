from ipaddress import IPv4Address
#from .SignalGenerator.BaseClass import SignalGenerator
#from .PowerMeter.BaseClass import PowerMeter
#from .SpectrumAnalyzer.BaseClass import SpectrumAnalyzer
from pydantic import BaseModel
from enum import Enum

class InstrumentTypes(str, Enum):
    SignalGenerator = "SG"
    SpectrumAnalyzer = "SA"
    PowerMeter = "PM"
    TtcTransmitter = "TTC"
    SwitchDriver = "SDU"

# class InsIntelisense:
#     SignalGenerator = SignalGenerator
#     PowerMeter = PowerMeter
#     SpectrumAnalyzer = SpectrumAnalyzer

class Units(Enum):
    MHz = "MHz"
    kHz = "kHz"
    dBm = "dBm"
    dB = "dB"
    radians = "radians"
    seconds = "seconds"
    milli_seconds = "milli_seconds"
    count = "count"
    volts = "volts"

class Limits(BaseModel):
    lower_limit: float
    upper_limit: float 
    units: Units

    class config:
        use_enums = True


class InterfaceProtocols(Enum):
    GPIB = "GPIB"
    LAN = "LAN"
    VXI = "VXI"
    ANY = "ANY"


class InstrumentAddress(BaseModel):
    ip_or_gpib_address: str | int 
    port_or_gpib_bus: int
    def __init__(self, ip_or_gpib_address, port_or_gpib_bus, **kwargs):
        super().__init__(ip_or_gpib_address=ip_or_gpib_address, port_or_gpib_bus=port_or_gpib_bus, **kwargs)