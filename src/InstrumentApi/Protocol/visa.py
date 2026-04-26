import pyvisa
from pyvisa import ResourceManager
from pydantic import BaseModel
import logging
from collections import defaultdict





# Define Request Models
class SCPIRequest(BaseModel):
    request_id: str | None = None
    instrument_name: str | None = None
    address: str
    command: str
    read: bool = False



# Define Response Model
class InstResponse(BaseModel):
    data: str | None = None
    error: str | None = None


class VisaRequest(BaseModel):
    address: str = ""
    commandString: str = ""
    read: bool = False
    getRawData: bool = False
    release_all_devices: bool = False
    timeOutValue: int = 13
    close_all_resources:bool = False


# Initialize PyVISA ResourceManager
rm = ResourceManager()

# Shared resources
instrument_pool = {}

def visa(req: VisaRequest):
    try:
        res = InstResponse()
        if req.close_all_resources:
            res.error = close_all_instruments()
            return res
        
        if req.address not in instrument_pool:
            inst:pyvisa.resources = rm.open_resource(req.address)
            # Set timeout (timeOutValue is in seconds, PyVISA uses milliseconds)
            inst.timeout = req.timeOutValue * 1000
            instrument_pool[req.address] = inst
        else:
            inst = instrument_pool[req.address]
            # Update timeout if different
            inst.timeout = req.timeOutValue * 1000
            
        if req.read and req.getRawData:
                res.data=inst.query_binary_values(req.commandString, datatype='B', container=bytes)
        elif req.read:
            res.data = inst.query(req.commandString)
        else:
            inst.write(req.commandString)
        return res
    except Exception as e:
        res.error = str(e)
        return res
        
            
def close_all_instruments():
    error = ""
    for address, inst in list(instrument_pool.items()):
        try:
            inst.close()
            del instrument_pool[address]
        except Exception as e:
            error += f"{address}: {e}\n"
    return error