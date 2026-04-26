"""
Remote HTTP-based Instrument Protocol
This module enables instrument control from intranet PC through a bridge PC (dual-LAN)

Architecture:
- Intranet PC: Runs this client code to send SCPI commands via HTTP
- Bridge PC (dual-LAN): Runs remote_server.py to execute SCPI on actual instruments
- Instruments: Connected to checknet, accessed by bridge PC

Flow:
1. Application on intranet PC calls remote_command()
2. HTTP POST request sent to bridge PC
3. Bridge PC executes SCPI command using GPIB/LAN/VISA protocol
4. Response returned via HTTP to intranet PC
"""

import requests
import json
import base64
import logging
import os
from pathlib import Path
from pydantic import BaseModel
from typing import Optional

logger = logging.getLogger(__name__)

# Resolve config file: look for instrument_config.toml walking up from this file's location
def _find_config_file() -> Path | None:
    search = Path(__file__).resolve().parent
    for _ in range(6):  # walk up at most 6 levels
        candidate = search / "instrument_config.toml"
        if candidate.exists():
            return candidate
        search = search.parent
    return None


def _load_toml_config() -> dict:
    config_path = _find_config_file()
    if config_path is None:
        logger.warning("instrument_config.toml not found — using defaults (local mode)")
        return {}
    try:
        # tomllib is stdlib in Python 3.11+; fall back to tomli for older versions
        try:
            import tomllib
        except ImportError:
            import tomli as tomllib  # pip install tomli
        with open(config_path, "rb") as f:
            data = tomllib.load(f)
        logger.info("Loaded instrument config from: %s", config_path)
        return data
    except Exception as e:
        logger.error("Failed to read instrument_config.toml: %s — using defaults", e)
        return {}


class RemoteRequest(BaseModel):
    """Request model for remote SCPI command execution"""
    instrument_address: str  # GPIB address or IP:port
    command_string: str  # SCPI command to execute
    read_operation: bool = False  # Whether to read response
    protocol: str = "ANY"  # Protocol to use on bridge PC: GPIB, LAN, VISA, ANY
    buffer_length: int = 200  # Read buffer length
    get_raw_data: bool = False  # Get binary/raw data
    release_all_devices: bool = False  # Release GPIB devices
    timeout_seconds: int = 30  # HTTP request timeout


class InstResponse(BaseModel):
    """Response model for instrument commands"""
    data: str | bytes | None = None
    error: str | None = None


class RemoteConfig(BaseModel):
    """Configuration for remote instrument server — populated from instrument_config.toml"""
    bridge_pc_url: str = "http://10.22.14.168:5000"
    timeout: int = 30
    retry_attempts: int = 3
    enabled: bool = False  # default: local mode


def _build_config() -> RemoteConfig:
    raw = _load_toml_config().get("remote", {})
    return RemoteConfig(
        enabled=raw.get("enabled", False),
        bridge_pc_url=raw.get("bridge_url", "http://10.22.14.168:5000"),
        timeout=raw.get("timeout", 30),
        retry_attempts=raw.get("retry_attempts", 3),
    )


# Loaded once at import time from instrument_config.toml
remote_config = _build_config()


def reload_config():
    """Re-read instrument_config.toml at runtime without restarting the app."""
    global remote_config
    remote_config = _build_config()
    logger.info("Remote config reloaded — enabled=%s, url=%s", remote_config.enabled, remote_config.bridge_pc_url)


def configure_remote_bridge(bridge_url: str = None, timeout: int = None, retry_attempts: int = None, enabled: bool = None):
    """
    Override remote config at runtime (does not write to instrument_config.toml).
    Only the provided arguments are updated; omitted ones keep their current value.
    To persist changes, edit instrument_config.toml instead.
    """
    global remote_config
    if bridge_url is not None:
        remote_config.bridge_pc_url = bridge_url.rstrip('/')
    if timeout is not None:
        remote_config.timeout = timeout
    if retry_attempts is not None:
        remote_config.retry_attempts = retry_attempts
    if enabled is not None:
        remote_config.enabled = enabled


def remote_command(req: RemoteRequest) -> InstResponse:
    """
    Send SCPI command to instrument via HTTP bridge
    
    Args:
        req: RemoteRequest object with command details
        
    Returns:
        InstResponse object with data or error
    """
    response = InstResponse()
    
    # Prepare request payload
    payload = {
        "instrument_address": req.instrument_address,
        "command_string": req.command_string,
        "read_operation": req.read_operation,
        "protocol": req.protocol,
        "buffer_length": req.buffer_length,
        "get_raw_data": req.get_raw_data,
        "release_all_devices": req.release_all_devices
    }
    
    # Attempt command execution with retries
    for attempt in range(remote_config.retry_attempts):
        try:
            # Send HTTP POST request to bridge PC
            http_response = requests.post(
                f"{remote_config.bridge_pc_url}/execute_command",
                json=payload,
                timeout=req.timeout_seconds or remote_config.timeout
            )
            
            # Parse response JSON first (even for error responses)
            try:
                result = http_response.json()
            except ValueError:
                # If JSON parsing fails, use status code error
                http_response.raise_for_status()
                result = {}
            
            # Check for errors in response
            if result.get("error"):
                response.error = result["error"]
            elif http_response.status_code >= 400:
                # HTTP error but no error message in JSON
                response.error = f"HTTP {http_response.status_code}: {http_response.reason}"
            else:
                response.data = result.get("data")
                
                # Handle binary data if requested
                if req.get_raw_data and response.data:
                    # Binary data is base64 encoded in JSON response
                    response.data = base64.b64decode(response.data)
            
            return response
            
        except requests.exceptions.Timeout:
            response.error = f"Timeout: Bridge PC did not respond within {req.timeout_seconds}s (attempt {attempt + 1}/{remote_config.retry_attempts})"
            if attempt == remote_config.retry_attempts - 1:
                break
                
        except requests.exceptions.ConnectionError:
            response.error = f"Connection Error: Cannot connect to bridge PC at {remote_config.bridge_pc_url} (attempt {attempt + 1}/{remote_config.retry_attempts})"
            if attempt == remote_config.retry_attempts - 1:
                break
                
        except requests.exceptions.RequestException as e:
            response.error = f"HTTP Request Error: {str(e)}"
            break
            
        except Exception as e:
            response.error = f"Remote Command Error: {str(e)}"
            break
    
    return response


def test_bridge_connection() -> dict:
    """
    Test connection to bridge PC
    
    Returns:
        dict with status, message, and bridge_info
    """
    try:
        http_response = requests.get(
            f"{remote_config.bridge_pc_url}/health",
            timeout=5
        )
        http_response.raise_for_status()
        
        result = http_response.json()
        return {
            "status": "connected",
            "message": "Bridge PC is reachable",
            "bridge_info": result
        }
        
    except requests.exceptions.ConnectionError:
        return {
            "status": "error",
            "message": f"Cannot connect to bridge PC at {remote_config.bridge_pc_url}",
            "bridge_info": None
        }
        
    except Exception as e:
        return {
            "status": "error",
            "message": f"Connection test failed: {str(e)}",
            "bridge_info": None
        }


def get_bridge_instruments() -> dict:
    """
    Get list of instruments available on bridge PC
    
    Returns:
        dict with available instruments and their addresses
    """
    try:
        http_response = requests.get(
            f"{remote_config.bridge_pc_url}/instruments",
            timeout=10
        )
        http_response.raise_for_status()
        
        return http_response.json()
        
    except Exception as e:
        return {
            "error": f"Failed to get instruments list: {str(e)}",
            "instruments": []
        }


# Example usage:
"""
from InstrumentApi.Protocol.remote import configure_remote_bridge, remote_command, RemoteRequest

# Configure bridge PC
configure_remote_bridge(bridge_url="http://192.168.1.100:5000", timeout=30)

# Send command to instrument via bridge
req = RemoteRequest(
    instrument_address="GPIB0::18::INSTR",
    command_string="*IDN?",
    read_operation=True,
    protocol="VISA"
)

response = remote_command(req)
if response.error:
    print(f"Error: {response.error}")
else:
    print(f"Response: {response.data}")
"""
