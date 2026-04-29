class InstrumentError(Exception):
    pass


class CommunicationError(InstrumentError):
    pass


class LimitViolationError(InstrumentError):
    def __init__(self, param: str, value: float, lower: float, upper: float):
        self.param = param
        self.value = value
        self.lower = lower
        self.upper = upper
        super().__init__(
            f"Limit violated for '{param}': {value} is outside [{lower}, {upper}]"
        )


class DriverNotImplementedError(InstrumentError):
    pass


class ConfigurationError(InstrumentError):
    pass
