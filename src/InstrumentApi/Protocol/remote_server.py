"""
Remote Instrument Bridge Server
Run this on the dual-LAN PC (checknet + intranet) to bridge instrument commands

This server receives HTTP requests from intranet PCs and executes SCPI commands
on instruments connected to checknet using GPIB/LAN/VISA protocols.

Usage:
    python remote_server.py --host 0.0.0.0 --port 5000

Requirements:
    pip install flask pyvisa pyvisa-py
"""

from flask import Flask, request, jsonify
import pyvisa
from pydantic import BaseModel
import json
import logging
from logging.handlers import RotatingFileHandler
import base64
from typing import Optional
import sys
import os

# Add parent directory to path to import protocol modules
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from Protocol.visa import visa, VisaRequest, InstResponse
from Protocol.gpib import gpib_command, GpibRequest

# Initialize Flask app
app = Flask(__name__)
app.config['JSON_SORT_KEYS'] = False


# Setup rotating file handler for logging
log_formatter = logging.Formatter('%(asctime)s - %(name)s - %(levelname)s - %(message)s')
file_handler = RotatingFileHandler(
    'instrument_bridge.log',
    maxBytes=10 * 1024 * 1024,  # 10MB
    backupCount=10
)
file_handler.setFormatter(log_formatter)
file_handler.setLevel(logging.INFO)

stream_handler = logging.StreamHandler()
stream_handler.setFormatter(log_formatter)
stream_handler.setLevel(logging.INFO)

logger = logging.getLogger(__name__)
logger.setLevel(logging.INFO)
logger.addHandler(file_handler)
logger.addHandler(stream_handler)

# PyVISA resource manager
rm = pyvisa.ResourceManager()


class CommandRequest(BaseModel):
    """Request model for instrument command"""
    instrument_address: str
    command_string: str
    read_operation: bool = False
    protocol: str = "ANY"  # GPIB, LAN, VISA, ANY
    buffer_length: int = 200
    get_raw_data: bool = False
    release_all_devices: bool = False
    timeout_seconds:int = 5


@app.route('/health', methods=['GET'])
def health_check():
    """Health check endpoint"""
    logger.info(f"Request: /health {request.method} {request.remote_addr}")
    try:
        resources = rm.list_resources()
        response = {
            "status": "online",
            "server": "Instrument Bridge Server",
            "version": "1.0.0",
            "available_resources": list(resources),
            "protocols": ["GPIB", "LAN", "VISA"],
            "message": "Bridge server is operational"
        }
        logger.info(f"Response: /health {response}")
        return jsonify(response)
    except Exception as e:
        response = {
            "status": "degraded",
            "error": str(e),
            "message": "Bridge server online but resource detection failed"
        }
        logger.error(f"Response: /health {response}")
        return jsonify(response), 500



@app.route('/instruments', methods=['GET'])
def list_instruments():
    """List available instruments"""
    logger.info(f"Request: /instruments {request.method} {request.remote_addr}")
    try:
        resources = rm.list_resources()
        instruments = []
        for resource in resources:
            try:
                inst = rm.open_resource(resource)
                idn = inst.query("*IDN?").strip()
                inst.close()
                instruments.append({
                    "address": resource,
                    "idn": idn,
                    "status": "available"
                })
            except:
                instruments.append({
                    "address": resource,
                    "idn": "Unknown",
                    "status": "detected"
                })
        response = {
            "count": len(instruments),
            "instruments": instruments
        }
        logger.info(f"Response: /instruments {response}")
        return jsonify(response)
    except Exception as e:
        response = {
            "error": str(e),
            "instruments": []
        }
        logger.error(f"Response: /instruments {response}")
        return jsonify(response), 500



@app.route('/execute_command', methods=['POST'])
def execute_command():
    """
    Execute SCPI command on instrument
    """
    logger.info(f"Request: /execute_command {request.method} {request.remote_addr} Payload: {request.get_json()}")
    try:
        data = request.get_json()
        cmd_req = CommandRequest(**data)
        logger.info(f"Executing: {cmd_req.command_string} on {cmd_req.instrument_address} via {cmd_req.protocol}")
        if cmd_req.protocol.upper() == "VISA" or cmd_req.protocol.upper() == "ANY":
            result = execute_via_visa(cmd_req)
        elif cmd_req.protocol.upper() == "GPIB":
            result = execute_via_gpib(cmd_req)
        else:
            response_data = {
                "error": f"Unsupported protocol: {cmd_req.protocol}",
                "data": None
            }
            logger.error(f"Response: /execute_command {response_data}")
            return jsonify(response_data), 400
        response_data = {
            "error": result.error,
            "data": None
        }
        if result.data is not None:
            if cmd_req.get_raw_data and isinstance(result.data, bytes):
                response_data["data"] = base64.b64encode(result.data).decode('utf-8')
            else:
                response_data["data"] = str(result.data)
        logger.info(f"Response: /execute_command {response_data}")
        if result.error:
            return jsonify(response_data), 500
        else:
            return jsonify(response_data)
    except Exception as e:
        response_data = {
            "error": f"Server error: {str(e)}",
            "data": None
        }
        logger.error(f"Response: /execute_command {response_data}")
        return jsonify(response_data), 500


def execute_via_visa(cmd_req: CommandRequest) -> InstResponse:
    """Execute command using VISA protocol"""
    visa_req = VisaRequest(
        address=cmd_req.instrument_address,
        commandString=cmd_req.command_string,
        read=cmd_req.read_operation,
        getRawData=cmd_req.get_raw_data,
        release_all_devices=cmd_req.release_all_devices,
        timeOutValue=cmd_req.timeout_seconds
    )
    
    return visa(visa_req)


def execute_via_gpib(cmd_req: CommandRequest) -> InstResponse:
    """Execute command using GPIB protocol"""
    # Parse GPIB address (format: GPIB0::18::INSTR or just 18)
    try:
        if "::" in cmd_req.instrument_address:
            parts = cmd_req.instrument_address.split("::")
            board_index = int(parts[0].replace("GPIB", ""))
            primary_address = int(parts[1])
        else:
            board_index = 0
            primary_address = int(cmd_req.instrument_address)
        
        gpib_req = GpibRequest(
            boardIndex=board_index,
            primaryAddress=primary_address,
            commandString=cmd_req.command_string,
            gpibRead=cmd_req.read_operation,
            bufferLength=cmd_req.buffer_length,
            getRawData=cmd_req.get_raw_data,
            release_all_devices=cmd_req.release_all_devices
        )
        
        return gpib_command(gpib_req)
        
    except Exception as e:
        result = InstResponse()
        result.error = f"GPIB address parsing error: {str(e)}"
        return result



@app.route('/close_all', methods=['POST'])
def close_all_resources():
    """Close all open instrument connections"""
    logger.info(f"Request: /close_all {request.method} {request.remote_addr}")
    try:
        from Protocol.visa import close_all_instruments
        errors = close_all_instruments()
        if errors:
            response = {"status": "partial", "message": "Some resources failed to close", "errors": errors}
            logger.warning(f"Response: /close_all {response}")
            return jsonify(response), 207
        response = {"status": "success", "message": "All resources closed"}
        logger.info(f"Response: /close_all {response}")
        return jsonify(response)
    except Exception as e:
        response = {"error": str(e)}
        logger.error(f"Response: /close_all {response}")
        return jsonify(response), 500


def main():
    """Main entry point for bridge server"""
    import argparse
    
    parser = argparse.ArgumentParser(description='Instrument Bridge Server')
    parser.add_argument('--host', default='0.0.0.0', help='Host address (default: 0.0.0.0)')
    parser.add_argument('--port', type=int, default=5000, help='Port number (default: 5000)')
    parser.add_argument('--debug', action='store_true', help='Enable debug mode')
    
    args = parser.parse_args()
    
    logger.info("=" * 60)
    logger.info("Instrument Bridge Server Starting")
    logger.info("=" * 60)
    logger.info(f"Host: {args.host}")
    logger.info(f"Port: {args.port}")
    logger.info(f"Protocols: GPIB, LAN, VISA")
    logger.info("=" * 60)
    
    try:
        # Test VISA availability
        resources = rm.list_resources()
        logger.info(f"Available VISA resources: {len(resources)}")
        for res in resources:
            logger.info(f"  - {res}")
    except Exception as e:
        logger.warning(f"VISA resource detection failed: {str(e)}")
    
    logger.info("\nServer ready to accept connections...")
    logger.info(f"Health check: http://{args.host}:{args.port}/health")
    logger.info(f"Instruments: http://{args.host}:{args.port}/instruments")
    logger.info(f"Execute: POST http://{args.host}:{args.port}/execute_command")
    logger.info("=" * 60)
    
    # Start Flask server
    app.run(
        host=args.host,
        port=args.port,
        debug=args.debug,
        threaded=True
    )


if __name__ == '__main__':
    main()


"""
Example usage:

1. Start the server on dual-LAN PC (checknet bridge):
   python remote_server.py --host 0.0.0.0 --port 5000

2. From intranet PC, send HTTP request:
   curl -X POST http://192.168.1.100:5000/execute_command \
        -H "Content-Type: application/json" \
        -d '{
            "instrument_address": "GPIB0::18::INSTR",
            "command_string": "*IDN?",
            "read_operation": true,
            "protocol": "VISA"
        }'

3. Or use the remote.py client module from Python:
   from InstrumentApi.Protocol.remote import configure_remote_bridge, remote_command, RemoteRequest
   
   configure_remote_bridge("http://192.168.1.100:5000")
   
   req = RemoteRequest(
       instrument_address="GPIB0::18::INSTR",
       command_string="*IDN?",
       read_operation=True
   )
   
   response = remote_command(req)
   print(response.data)
"""
