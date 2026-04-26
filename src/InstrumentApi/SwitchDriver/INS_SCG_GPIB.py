from .BaseClass import SwitchDriver,Specification
from ..Models import InstrumentTypes,Limits,Units,InterfaceProtocols
from ..factory import factory
from ..InstrumentBaseClass import apply_exceptions_and_traceback_to_all_methods,catch_exceptions_and_traceback


@apply_exceptions_and_traceback_to_all_methods(catch_exceptions_and_traceback)
class ScgSduGpib(SwitchDriver):
    def __init__(self) -> None:
        super().__init__()
        self.supported_protocol = InterfaceProtocols.GPIB.value

    def get_instrument_name(self):
        data = self.command("*IDN?", read_operation=True)
        if len(data) < 2:
            raise Exception('GPIB ERROR, Failed to Communicate Power Meter N1914A')

    def get_switch_positions(self):
        return self.command("D?\r",read_operation=True)

    def set_switch_position(self, switch_position: str):
        if switch_position:
            switch_position +='\r'
            self.command(switch_position)
            position =self.command("",read_operation=True)  # Query back to ensure command was processed
        return self

    def validate_switch_positions(self):
        raise NotImplementedError


factory.register_component(InstrumentTypes.SwitchDriver,"SDU_SCG_GPIB",ScgSduGpib)