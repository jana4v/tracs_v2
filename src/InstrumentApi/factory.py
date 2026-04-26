import os, importlib
from .Models import InstrumentAddress


class Factory:
    def __init__(self):
        self._creators = {}

    def register_component(self, type, model, class_ref):
        if self._creators.get(type):
            self._creators[type][model] = class_ref
        else:
            self._creators[type] = {model: class_ref}
        print(f"""Instrument type:{type} with Model Number:{model} got Registered.""")

    def get_component(self, type, model):
        #print(f"Getting component Command Format:{command_format}")
        try:
            type = self._creators.get(type)
            if type:
                return type.get(model)
            return None
        except AttributeError as ae:
            return None


factory = Factory()

destination_dir = os.path.dirname(__file__)
root_package = "src.InstrumentApi"
for root, sub_folders, files in os.walk(destination_dir):
    if "__pycache__" in root:
        continue
    for file in files:
        if file.startswith("INS_") and file.endswith(".py"):
            module_name = file[:-3]  # Remove the .py extension
            relative_path = os.path.relpath(root, destination_dir)
            # Form the package name dynamically
            if relative_path == '.':
                package_name = root_package
            else:
                package_name = ".".join([root_package] + relative_path.split(os.sep))

            #print(f"Importing module {module_name} from package {package_name}")
            try:
                importlib.import_module(f".{module_name}", package=package_name)
            except ModuleNotFoundError as e:
                print(f"Error importing module {module_name} from package {package_name}: {e}")


def get_instrument(instrument_type, model_number, address: InstrumentAddress):
    cls = factory.get_component(instrument_type, model_number)
    ins = cls()
    ins.address = address
    return ins
