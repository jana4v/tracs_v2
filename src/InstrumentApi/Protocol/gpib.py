import sys
import logging
import ctypes.util
from ctypes import cdll, c_long, c_char_p, c_int
import ctypes, json
from pydantic import BaseModel

logger = logging.getLogger(__name__)


class GpibRequest(BaseModel):
    boardIndex: int = 0
    primaryAddress: int = 0
    bufferLength: int = 200
    commandString: str = ""
    gpibRead: bool = False
    getRawData: bool = False
    release_all_devices: bool = False
    secondaryAddress: int = 0
    timeOutValue: int = 13
    EOImode: int = 1
    EOSmode: int = 0


class InstResponse(BaseModel):
    data: str | None = None
    error: str | None = None


isError = True
gpib_error = None
_gpib_instance = None


class NI488Error():
    def __init__(self):
        self.ERR = 1 << 15
        self.IbStatusCodes = ['DCAS', 'DTAS', 'LACS', 'TACS', 'ATN', 'CIC', 'REM',
                              'LOK', 'CMPL', 'EVENT', 'SPOLL', 'RQS', 'SRQI', 'END', 'TIMO', 'ERR']
        self.IbStatusCode = dict([(1 << i, self.IbStatusCodes[i])
                                  for i in range(len(self.IbStatusCodes))])

        self.IbErrorCodes = {0: 'EDVR', 1: 'ECIC', 2: 'ENOL', 3: 'EADR', 4: 'EARG', 5: 'ESAC', 6: 'EABO', 7: 'ENEB',
                             8: 'EDMA', 10: 'EOIP', 11: 'ECAP',
                             12: 'EFSO', 14: 'EBUS', 16: 'ESRQ', 20: 'ETAB', 21: 'ELCK', 22: 'EARM', 23: 'EHDL',
                             26: 'EWIP', 27: 'ERST', 28: 'EPWR'}

        self.IbErrorDescription = {'EDVR': 'System error',
                                   'ECIC': 'Function requires GPIB interface to be CIC',
                                   'ENOL': 'No Listeners on the GPIB',
                                   'EADR': 'GPIB interface not addressed correctly',
                                   'EARG': 'Invalid argument to function call',
                                   'ESAC': 'GPIB interface not System Controller as required',
                                   'EABO': 'I/O operation aborted (timeout)',
                                   'ENEB': 'Nonexistent GPIB interface',
                                   'EDMA': 'DMA error',
                                   'EOIP': 'Asynchronous I/O in progress',
                                   'ECAP': 'No capability for operation',
                                   'EFSO': 'File system error',
                                   'EBUS': 'GPIB bus error',
                                   'ESRQ': 'SRQ stuck in ON position',
                                   'ETAB': 'Table problem',
                                   'ELCK': 'Interface is locked',
                                   'EARM': 'ibnotify callback failed to rearm',
                                   'EHDL': 'Input handle is invalid',
                                   'EWIP': 'Wait in progress on specified input handle',
                                   'ERST': 'The event notification was cancelled due to a reset of the interface',
                                   'EPWR': 'The interface lost power'}

        self.timeOutNames = ['TNONE', 'T10us', 'T30us', 'T100us', 'T300us', 'T1ms',
                             'T3ms', 'T10ms', 'T30ms', 'T100ms', 'T300ms', 'T1s', 'T3s', 'T10s',
                             'T30s', 'T100s', 'T300s', 'T1000s']
        self.timeOuts = dict([(self.timeOutNames[i], i) for i in
                              range(len(self.timeOutNames))])


NI488Error = NI488Error()


class GPIBException(Exception):
    pass


class NI488(object):
    """
    This Class Handles NI488.2(GPIB) communication.
    Functions:
        gpibCommand: This is high level function of this class to handle GPIB communication.
        remaining functions are low level.

    """

    def __init__(self, errorHandlinfFunctionRefernce=None):
        """
        This Constructor loads NI488.2 library
        and instance variables are initilized with function references from NI488.2 library
        """
        self._lib = None
        self._GPIBboardIndex = 0
        self._GPIBprimaryAddress = -1
        self._GPIBsecondaryAddress = 0
        self._GPIB_EOImode = 1
        self._GPIB_EOSmode = 0
        self._GPIBtimeOutValue = 13
        self._GPIBcommandString = None
        self._GPIBreadResult = None
        self._deviceDescriptor = -1
        self._bufferLength = 200
        self._parameter = None
        self._value = None
        self._handleError = errorHandlinfFunctionRefernce
        self._deviceDescriptorDict = {}
        self._loadLibrary()
        self._isGPIBReadOperation = False

    def __del__(self):
        logger.info("Releasing all gpib instruments....")
        self.releaseAllInstruments()
    def _loadLibrary(self):
        try:
            path_to_NI488_Library = ctypes.util.find_library("NI4882")
            if path_to_NI488_Library != None:
                # On Windows load NI488.2 dll as windll
                if sys.platform.startswith('win'):
                    self._lib = ctypes.windll.LoadLibrary(
                        path_to_NI488_Library)
                else:  # On other OS load NI488.2 library  as cdll
                    self._lib = cdll.LoadLibrary(path_to_NI488_Library)
            else:
                error = "NI488.2 Not Found in System Path, please Install NI488.2 Drivers"
                if self._handleError == None:
                    raise GPIBException(error)
                else:
                    self._handleError(error, self._loadLibrary)

            # get NI488.2 library function references
            self._Ibsta = getattr(self._lib, 'ThreadIbsta')
            self._Ibcnt = getattr(self._lib, 'ThreadIbcnt')
            self._Ibcnt1 = getattr(self._lib, 'Ibcnt')
            self._Iberr = getattr(self._lib, 'ThreadIberr')

        except Exception as e:
            error = "Error from NI488 Class,Problem in Loading Functions From GPIB Library : " + \
                    str(e)
            if self._handleError == None:
                raise GPIBException(error)
            else:
                self._handleError(error, self._loadLibrary)

    def _del_(self):
        if self._deviceDescriptor == -1:
            self._Ibonl()  # release device descriptor

    def _Ibdev(self, req: GpibRequest):
        """
        This function gets device handle for a GPIB instrument

        Function Arguments
        :boardIndex       : 0 or 1 or 2 .. (NI GPIB to Lan Card Index) defaults to o
        :primaryAddress   : GPIB primaryAddress of instrument.
        :secondaryAddress : GPIB secondaryAddress of instrument
        :timeOutValue     : GPIB time out value
        :EOImode          : if Zero: The GPIB EOI line will not be asserted at the end of a write operation.
                            if non Zero: EOI will be asserted at the end of a write.
        :EOSmode          : GPIB data transfers are terminated either when the GPIB EOI line is asserted
                            with the last byte of a transfer or when a preconfigured end-of-string (EOS)
                            character is transmitted. By default, EOI is asserted with the last byte of
                            writes and the EOS modes are disabled

        Return Value:
            On Sucess returns device descriptor(an integer value)
            On failues returns -1

        """
        try:
            # int ibdev (int BdIndx, int pad, int sad, int tmo, int eot, int eos) function declaration in NI488 library
            #self._lib.ibdev.argtypes = [c_int,c_int,c_int,c_int,c_int,c_int]
            logger.debug("Opening GPIB device at primary address %s", self._GPIBprimaryAddress)
            deviceDescriptor = self._lib.ibdev(req.boardIndex,
                                               req.primaryAddress,
                                               req.secondaryAddress,
                                               req.timeOutValue,
                                               req.EOImode,
                                               req.EOSmode)
            if deviceDescriptor == -1:  # if failed to get device descriptor
                # check iberror registor status
                self._CheckStatus(caller=self._Ibdev)
            else:
                self._deviceDescriptor = deviceDescriptor
                key = str(req.boardIndex) + '_' + \
                      str(req.primaryAddress)
                self._deviceDescriptorDict[key] = self._deviceDescriptor
        except Exception as e:
            error = 'Error From NI488 Class in Ibdev funtion: ' + str(e)
            if self._handleError == None:
                raise GPIBException(error)
            else:
                self._handleError(error, self._Ibdev)

    def _Ibrd(self, deviceDescriptor=None, bufferLength=None):
        """
        This function reads data from a GPIB instrument

        Function Arguments
        :deviceDescriptor : GPIB device descripot of an instrument(Use Ibdev to get deviceDescriptor )
        :bufferLength     : Number of bytes to be read from the GPIB

        Return Value:
            The data from GPIB instrument

        """
        try:
            # copy arguments to instance variables
            if deviceDescriptor != None:
                self._deviceDescriptor = deviceDescriptor
            if bufferLength != None:
                self._bufferLength = bufferLength
            self._lib.ibrd.argtypes = [c_int, c_char_p, c_long]
            result = ctypes.create_string_buffer(self._bufferLength)
            if not result:
                error = 'can\'t allocate memory for string buffer for Ibrd'
                if self._handleError == None:
                    raise GPIBException(error)
                else:
                    self._handleError(error, self._Ibrd)

            ibstatus = self._lib.ibrd(
                self._deviceDescriptor, result, self._bufferLength)
            # check iberror registor status
            self._CheckStatus(ibsta=ibstatus, caller=self._Ibrd)
            self._GPIBreadResult = result.raw
            return result.raw
        except Exception as e:
            error = 'Error From NI488 Class in Ibrd funtion: ' + str(e)
            if self._handleError == None:
                raise GPIBException(error)
            else:
                self._handleError(error, self._Ibrd)

    def _Ibwrt(self, deviceDescriptor=None, commandString=None):
        """
        This function sends commands to GPIB instrument

        Function Arguments
        :deviceDescriptor : GPIB device descripot of an instrument(Use Ibdev to get deviceDescriptor )
        :commandString    : Command to be sent to GPIB instrument

        Return Value:
            The value of Ibsta
        """
        try:
            # copy arguments to instance variables
            if deviceDescriptor != None:
                self._deviceDescriptor = deviceDescriptor
            if commandString != None:
                self._GPIBcommandString = commandString
            command = ctypes.create_string_buffer(
                self._GPIBcommandString.encode('utf-8'))
            self._lib.ibwrt.argtypes = [c_int, c_char_p, c_long]
            ibstatus = self._lib.ibwrt(self._deviceDescriptor,
                                       command,
                                       len(self._GPIBcommandString))

            # check iberror registor status
            self._CheckStatus(ibsta=ibstatus, caller=self._Ibwrt)
        except Exception as e:
            error = 'Error From NI488 Class in Ibwrt funtion: ' + str(e)
            if self._handleError == None:
                raise GPIBException(error)
            else:
                self._handleError(error, self._Ibwrt)

    def _Ibonl(self, deviceDescriptor=None, mode=0):
        """
        This function releses GPIB handle of a  GPIB instrument

        Function Arguments
        :deviceDescriptor : GPIB device descriptor of an instrument(Use Ibdev to get deviceDescriptor )
        :mode    : Indicates whether the board or device is to be placed online or offline

        Return Value:
            The value of Ibsta

        Description:
        ibonl resets the board or device and places all its software configuration parameters in
        their pre-configured state. In addition, if v is zero, the device or interface is placed offline.
        If v is non-zero, the device or interface is left operational, or online.

        If a device or an interface is taken offline, the board or device descriptor (ud) is no longer valid.
        You must execute an ibdev or ibfind to access the board or device again.
        """
        try:
            # copy arguments to instance variables
            if deviceDescriptor != None:
                self._deviceDescriptor = deviceDescriptor
            self._lib.ibonl.argtypes = [c_int, c_int]
            ibstatus = self._lib.ibonl(self._deviceDescriptor, mode)
            # check iberror registor status
            self._CheckStatus(ibsta=ibstatus, caller=self._Ibonl)
            self._deviceDescriptor = -1
        except Exception as e:
            error = 'Error From NI488 Class in Ibonl function: ' + str(e)
            if self._handleError == None:
                raise GPIBException(error)
            else:
                self._handleError(error, self._Ibonl)

    def _Ibconfig(self, deviceDescriptor=None, parameter=None, value=None):
        # copy arguments to instance variables
        """
        This function releses GPIB handle of a  GPIB instrument

        Function Arguments
        :deviceDescriptor : GPIB device descriptor of an instrument(Use Ibdev to get deviceDescriptor )
        :parameter        : A parameter that selects the software configuration item
        :value            : The value to which the selected configuration item is to be changed
        Return Value:
            The value of Ibsta

        Description:
        ibconfig changes a configuration item to the specified value for the selected board or device.
        option can be any of the defined options (see ibconfig Board Configuration Parameter Options
        or ibconfig Device Configuration Parameter Options). value must be valid for the parameter
        that you are configuring. The previous setting of the configured item is returned in Iberr.

        """
        try:
            if deviceDescriptor != None:
                self._deviceDescriptor = deviceDescriptor
            if parameter != None:
                self._parameter = parameter
            if value != None:
                self._value = value
            # C function :unsigned long ibconfig (int ud, int option, int value)
            self._lib.ibconfig.argtypes = [c_int, c_int, c_int]
            ibstatus = self._lib.ibconfig(
                self._deviceDescriptor, self._parameter, self._value)
            # check iberror registor status
            self._CheckStatus(ibsta=ibstatus, caller=self._Ibconfig)
        except Exception as e:
            error = 'Error From NI488 Class in Ibconfig function: ' + str(e)
            if self._handleError == None:
                raise GPIBException(error)
            else:
                self._handleError(error, self._Ibconfig)

    def _CheckStatus(self, caller=None, ibsta=None):
        """
        This function checks Ibstatus registor and raises an Exception in case of error

        Function Arguments
        :ibsta            : ibsta register value

        Description:
        This function is implimented to check status of any GPIB transaction.
        """
        if ibsta == None:
            ibsta = self._Ibsta()
        ERR = 1 << 15
        if (ibsta & ERR):
            errorIn = ' GPIB BoardIndex:' + \
                      str(self._GPIBboardIndex) + ' GPIB Address:' + \
                      str(self._GPIBprimaryAddress)
            error = NI488Error.IbErrorDescription[NI488Error.IbErrorCodes[self._Iberr(
            )]] + errorIn
            if self._handleError == None:
                raise GPIBException(error)
            else:
                self._handleError(error, caller)
        else:
            return None

    def gpibCommand(self, req: GpibRequest):
        """
        This is high level funtion of NI488 Class.

        Function Arguments
        :commandString    : GPIB command to be sent to GPIB instrument
        :gpibRead         : gpib Read or Write Operation
        :boardIndex       : 0 or 1 or 2 .. (NI GPIB to Lan Card Index) defaults to o
        :primaryAddress   : GPIB primaryAddress of instrument.
        :secondaryAddress : GPIB secondaryAddress of instrument
        :timeOutValue     : GPIB time out value
        :EOImode          : if Zero: The GPIB EOI line will not be asserted at the end of a write operation.
                            if non Zero: EOI will be asserted at the end of a write.
        :EOSmode          : GPIB data transfers are terminated either when the GPIB EOI line is asserted
                            with the last byte of a transfer or when a preconfigured end-of-string (EOS)
                            character is transmitted. By default, EOI is asserted with the last byte of
                            writes and the EOS modes are disabled


        :bufferLength     : Number of bytes to be read from the GPIB(For Trace read bufferLength=18*numberOfPoints )


        Return Value:
            None: in case of GPIB write
            data from instrument in case of GPIB read

        Description:
        This function is implimented for easy use of NI488.2 communication.
        Use this function to communicate with any GPIB instrument.
        """
        self._isGPIBReadOperation = req.gpibRead
        result = None
        boardIndex = int(req.boardIndex)
        key = str(boardIndex) + "_" + str(req.primaryAddress)
        self._deviceDescriptor = self._deviceDescriptorDict.get(key)
        self._GPIBboardIndex = boardIndex
        self._GPIBprimaryAddress = req.primaryAddress
        if not self._deviceDescriptor:  # Check wether command is for the same instrument from previous call
            self._Ibdev(req)  # if command is not for same instrument then get devhandle
        if req.commandString != None and req.commandString != '':
            self._Ibwrt(self._deviceDescriptor, req.commandString)
        if req.gpibRead and not req.getRawData:
            result = self._Ibrd(self._deviceDescriptor, req.bufferLength)
            if result != None:
                result = result.decode()
                result = result.strip('\x00')
                result = result.strip()
        elif req.gpibRead and req.getRawData:
            result = self._Ibrd(self._deviceDescriptor, req.bufferLength)
        return result

    def releaseAllInstruments(self):
        for deviceDescriptor in self._deviceDescriptorDict.values():
            self._Ibonl(deviceDescriptor)
        self._deviceDescriptorDict = {}


def errorHandler(error, functionHandler):
    global isError
    global gpib_error
    error_msg = json.dumps('Error:' + str(error))
    logger.error(error_msg)
    isError = True
    gpib_error = error_msg


def _get_gpib() -> NI488:
    global _gpib_instance
    if _gpib_instance is None:
        _gpib_instance = NI488(errorHandler)
        _gpib_instance.releaseAllInstruments()
    return _gpib_instance


def gpib_command(gpib_request: GpibRequest):
    global gpib_error
    gpib_error = None  # reset before each call to avoid stale errors
    gpib_response = InstResponse()
    gpib = _get_gpib()
    if gpib_request.release_all_devices:
        gpib.releaseAllInstruments()
        return gpib_response
    response = gpib.gpibCommand(gpib_request)
    gpib_response.error = gpib_error
    gpib_response.data = response
    return gpib_response
