from .Models import Limits, InstrumentAddress, InterfaceProtocols
from .Protocol.gpib import gpib_command, GpibRequest
from .Protocol.remote import remote_command, RemoteRequest, remote_config
import asyncio
import functools
import inspect
import logging
import re
import pyvisa
import sqlite3
import threading
import time
from pathlib import Path

from src.config import settings

logger = logging.getLogger(__name__)

_LOW_LEVEL_GPIB_LOCK = threading.Lock()
_LOW_LEVEL_GPIB_CACHE = {"ts": 0.0, "enabled": False}


def _is_low_level_gpib_enabled() -> bool:
    now = time.monotonic()
    with _LOW_LEVEL_GPIB_LOCK:
        if now - float(_LOW_LEVEL_GPIB_CACHE["ts"]) < 2.0:
            return bool(_LOW_LEVEL_GPIB_CACHE["enabled"])

    db_path = Path(__file__).resolve().parents[2] / settings.SQLITE_DB_PATH
    enabled = False

    try:
        conn = sqlite3.connect(str(db_path))
        try:
            row = conn.execute(
                "SELECT Value FROM Configuration WHERE Parameter = ?",
                ("USE_LOW_LEVEL_GPIB",),
            ).fetchone()
            text = "" if row is None else str(row[0] or "").strip().lower()
            enabled = text in {"1", "true", "yes", "on"}
        finally:
            conn.close()
    except Exception:
        enabled = False

    with _LOW_LEVEL_GPIB_LOCK:
        _LOW_LEVEL_GPIB_CACHE["ts"] = now
        _LOW_LEVEL_GPIB_CACHE["enabled"] = enabled
    return enabled


class InstrumentBaseClass:
    def __init__(self):
        self.address: InstrumentAddress = None
        # Delay VISA backend initialization until local VISA command path is used.
        # This allows remote-bridge mode to run on machines without NI-VISA/pyvisa-py.
        self.rm = None
        self.ins_ref = None

    def __del__(self):
        try:
            self.ins_ref.close()
        except:
            pass

    def connect(self):
        """Open the VISA resource explicitly. Can also be used as a context manager."""
        if self.rm is None:
            try:
                self.rm = pyvisa.ResourceManager("@ni")
            except Exception:
                self.rm = pyvisa.ResourceManager()
        if self.ins_ref is None:
            if self.is_lan_based_instrument():
                self.ins_ref = self.rm.open_resource(
                    f'TCPIP0::{self.address.ip_or_gpib_address}::inst0::INSTR'
                )
            else:
                self.ins_ref = self.rm.open_resource(
                    f'GPIB{self.address.port_or_gpib_bus}::{self.address.ip_or_gpib_address}::INSTR'
                )

    def disconnect(self):
        """Close the VISA resource and reset connection state."""
        if self.ins_ref is not None:
            try:
                self.ins_ref.close()
            except Exception:
                pass
            self.ins_ref = None
        if self.rm is not None:
            try:
                self.rm.close()
            except Exception:
                pass
            self.rm = None

    def __enter__(self):
        self.connect()
        return self

    def __exit__(self, *_):
        self.disconnect()

    def is_lan_based_instrument(self):
        addr = self.address.ip_or_gpib_address
        if not isinstance(addr, str):
            return False
        a = addr.strip()
        # IPv4 address: four dot-separated octets (e.g. "192.168.1.10")
        if re.match(r'^\d{1,3}(?:\.\d{1,3}){3}$', a):
            return True
        # Hostname with at least one dot (e.g. "myinstrument.local")
        if '.' in a and not a.replace('.', '').isdigit():
            return True
        return False

    def command(self, command: str, read_operation: bool = False, buffer_length=200, get_raw_data=False, release_all_devices=False):
        if remote_config.enabled:
            return self._remote(command, read_operation, buffer_length, get_raw_data, release_all_devices)
        if self.supported_protocol == InterfaceProtocols.ANY.value:
            if not self.is_lan_based_instrument() and _is_low_level_gpib_enabled():
                return self.gpib(command, read_operation, buffer_length, get_raw_data, release_all_devices)
            try:
                return self.visa(command, read_operation, get_raw_data)
            except Exception:
                # Fallback for local GPIB setups where VISA backend is unavailable/misconfigured.
                if not self.is_lan_based_instrument():
                    return self.gpib(command, read_operation, buffer_length, get_raw_data, release_all_devices)
                raise
        elif self.supported_protocol == InterfaceProtocols.GPIB.value:
            return self.gpib(command, read_operation, buffer_length, get_raw_data, release_all_devices)
        elif self.supported_protocol == InterfaceProtocols.LAN.value:
            return self.visa(command, read_operation, get_raw_data)
        else:
            raise NotImplementedError(f"Unsupported protocol: {self.supported_protocol}")

    def _remote(self, command: str, read_operation: bool, buffer_length: int, get_raw_data: bool, release_all_devices: bool):
        # When Low Level GPIB is enabled and the instrument is not LAN-based,
        # tell the bridge server to use the GPIB protocol instead of VISA/ANY.
        effective_protocol = self.supported_protocol
        if (
            effective_protocol == InterfaceProtocols.ANY.value
            and not self.is_lan_based_instrument()
            and _is_low_level_gpib_enabled()
        ):
            effective_protocol = InterfaceProtocols.GPIB.value

        req = RemoteRequest(
            instrument_address=str(self.address.ip_or_gpib_address),
            command_string=command,
            read_operation=read_operation,
            protocol=effective_protocol,
            buffer_length=buffer_length,
            get_raw_data=get_raw_data,
            release_all_devices=release_all_devices,
        )
        response = remote_command(req)
        if response.error:
            raise Exception(response.error)
        return response.data
    def visa(self,command: str, read_operation: bool = False,get_raw_data:bool = False):
        if self.rm is None:
            try:
                # Prefer NI-VISA for reliable GPIB support.
                self.rm = pyvisa.ResourceManager("@ni")
            except Exception:
                self.rm = pyvisa.ResourceManager()
        if self.ins_ref is None:
            if self.is_lan_based_instrument():
                ins: pyvisa.resources.tcpip.TCPIPInstrument = self.rm.open_resource(f'TCPIP0::{self.address.ip_or_gpib_address}::inst0::INSTR')
            else:
                ins: pyvisa.resources.gpib.GPIBInstrument = self.rm.open_resource(f'GPIB{self.address.port_or_gpib_bus}::{self.address.ip_or_gpib_address}::INSTR')
            self.ins_ref = ins
        if read_operation and get_raw_data:
            return self.ins_ref.query_binary_values(command, datatype='B', container=bytes)
        elif read_operation:
            return self.ins_ref.query(command)
        else:
            return self.ins_ref.write(command)

    def lan(self):
        pass

    def gpib(self,command: str, read_operation: bool = False,buffer_length=200, get_raw_data=False,release_all_devices=False):
        req = GpibRequest()
        req.commandString = command
        req.gpibRead = read_operation
        req.boardIndex = self.address.port_or_gpib_bus
        req.primaryAddress = int(self.address.ip_or_gpib_address)
        req.bufferLength = buffer_length
        req.getRawData = get_raw_data
        req.release_all_devices = release_all_devices
        return gpib_command(req)

    def check_limit(self, limits: Limits, value: float, param_name: str = "value"):
        if not (limits.lower_limit <= value <= limits.upper_limit):
            from .exceptions import LimitViolationError
            raise LimitViolationError(param_name, value, limits.lower_limit, limits.upper_limit)


import traceback


def catch_exceptions_and_traceback(method):
    def wrapper(*args, **kwargs):
        try:
            return method(*args, **kwargs)
        except Exception as e:
            tb = traceback.extract_tb(e.__traceback__)  # Extract the traceback details
            logger.error(f"Exception in {method.__name__}: {e}")
            for frame in tb:
                logger.error(f"File: {frame.filename}, Line: {frame.lineno}, Function: {frame.name}, Code: {frame.line}")
            raise  # Optionally re-raise the exception if you want to propagate it

    return wrapper


def apply_exceptions_and_traceback_to_all_methods(decorator):
    def decorate(cls):
        do_not_async_wrap = {"initialize_specifications"}

        # Collect public sync methods before renaming
        sync_methods_to_wrap = {}
        for attr in dir(cls):
            if attr in do_not_async_wrap:
                continue
            if callable(getattr(cls, attr)) and not attr.startswith("__"):
                original_method = getattr(cls, attr)
                decorated_method = decorator(original_method)
                setattr(cls, attr, decorated_method)
                sync_methods_to_wrap[attr] = original_method

        # Rename sync methods to _sync and create async versions with clean names
        for original_name, sync_method in sync_methods_to_wrap.items():
            if original_name.endswith("_sync") or original_name.endswith("_async"):
                continue
            if original_name in do_not_async_wrap:
                continue

            # Rename the sync method to method_sync
            sync_name = f"{original_name}_sync"
            setattr(cls, sync_name, getattr(cls, original_name))

            # Create async wrapper with clean name (no suffix)
            def _make_async_wrapper(sync_method):
                @functools.wraps(sync_method)
                def _async_wrapper(self, *args, **kwargs):
                    try:
                        asyncio.get_running_loop()
                    except RuntimeError:
                        # Called from sync context (including worker threads): run directly.
                        return sync_method(self, *args, **kwargs)

                    # Called from async context: run sync instrument I/O off the event loop.
                    return asyncio.to_thread(sync_method, self, *args, **kwargs)

                return _async_wrapper

            setattr(cls, original_name, _make_async_wrapper(sync_method))

        return cls

    return decorate
