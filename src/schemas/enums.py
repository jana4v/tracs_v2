from enum import Enum


class SystemType(str, Enum):
    Transmitter = "Transmitter"
    Receiver = "Receiver"
    Transponder = "Transponder"


class ModulationType(str, Enum):
    PSK_PM = "PSK_PM"
    PSK_FM = "PSK_FM"
    QPSK_CDMA = "QPSK_CDMA"


class Port(str, Enum):
    EV = "EV"
    AEV = "AEV"
    GLOBAL = "GLOBAL"
    PAA = "PAA"
    PATCH_ARRAY = "PATCH_ARRAY"
