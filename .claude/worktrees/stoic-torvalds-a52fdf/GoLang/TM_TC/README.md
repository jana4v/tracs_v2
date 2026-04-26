# GoLang Codebase Documentation

## Overview

The `GoLang` folder contains **satellite ground station software** built in Go. It provides telemetry (TM) reception, telecommand (TC) transmission, and test procedure execution capabilities for spacecraft communication.

---

## Folder Structure

```
GoLang/
├── TM_TC/                           # Main application
│   ├── main.go                      # Application entry point
│   ├── umacs_websocket_server.go    # Mock UMACS server for testing
│   ├── config.toml                  # Configuration file
│   ├── bin/                         # Compiled binaries
│   ├── tm/                          # Telemetry receivers
│   ├── tm_simulator/                # TM data simulator
│   ├── restapi/                     # REST API server
│   ├── wampApi/                    # WAMP real-time messaging
│   ├── shared/                     # Shared utilities & data types
│   ├── connectionManager/          # Connection management
│   ├── delete/                     # Cleanup utilities
│   ├── logs/                       # Logging utilities
│   ├── templates/                  # HTML templates for API docs
│   └── vendor/                    # Go dependencies
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        TM_TC Application                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────────┐    │
│  │ REST API     │   │ WAMP API     │   │ TM Simulator    │    │
│  │ Server       │   │ (Pub/Sub)    │   │ (Testing)       │    │
│  │ (Port 11000) │   │              │   │                 │    │
│  └──────┬───────┘   └──────┬───────┘   └────────┬─────────┘    │
│         │                   │                     │              │
│         └───────────────────┼─────────────────────┘              │
│                             │                                    │
│                    ┌────────┴────────┐                          │
│                    │   Redis Cache  │                          │
│                    │   (Real-time)  │                          │
│                    └────────┬────────┘                          │
│                             │                                    │
│         ┌───────────────────┼───────────────────┐              │
│         │                   │                   │              │
│  ┌──────┴──────┐   ┌───────┴───────┐   ┌──────┴──────┐     │
│  │ MongoDB     │   │ InfluxDB      │   │ ArangoDB    │     │
│  │ (Documents) │   │ (Time-series) │   │ (Graph)     │     │
│  └─────────────┘   └────────────────┘   └─────────────┘     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
         │                              │
         │ WebSocket                   │ HTTP/WebSocket
         ▼                              ▼
┌─────────────────┐           ┌─────────────────────┐
│ UMACS Data     │           │ Frontend GUI        │
│ Server         │           │ (Web Applications)  │
│ (Spacecraft)   │           │                     │
└─────────────────┘           └─────────────────────┘
```

---

## Component Details

### 1. Main Application Entry Point

**[main.go](file:///e:\Code\Mainframe\MainframeAutomation\GoLang\TM_TC\main.go)**

Starts three concurrent services:

```go
func main() {
    shared.ReadTomlConfigFile()           // Load configuration
    go api.RestApiServer()                  // HTTP API (port 11000)
    go tm.TmApp(umacsDataServerIP)         // TM receivers
    go wampApi.InitWampAPI()               // WAMP pub/sub
    wg.Wait()
}
```

---

### 2. Telemetry Application (`tm/`)

Receives real-time telemetry from the spacecraft data server via WebSocket.

#### Supported Data Streams

| Stream  | Description                    | File              |
|---------|--------------------------------|-------------------|
| TM1     | Primary telemetry              | ReceiveScTm.go   |
| TM2     | Secondary telemetry             | ReceiveScTm.go   |
| SMON1/2 | Status monitoring              | ReceiveSmon.go   |
| ADC1/2  | Analog digital converter       | ReceiveAdc.go    |
| PTM1/2  | Processed telemetry             | ReceiveScPtm.go  |
| TC      | Telecommand sent confirmation  | ReceiveTcSent.go |

#### Key Functions ([tm/main.go](file:///e:\Code\Mainframe\MainframeAutomation\GoLang\TM_TC\tm\main.go))

```go
func TmApp(umacsDataServerIP string) {
    // Subscribes to configured data streams
    if contains(streams, "TM1") { go TmSubscriber("TM1", ...) }
    if contains(streams, "SMON1") { go SmonSubscriber("SMON1", ...) }
    if contains(streams, "ADC1") { go ADCSubscriber("ADC1", ...) }
    if contains(streams, "PTM1") { go PtmSubscriber("PTM1", ...) }
    if contains(streams, "TC") { go TcSent(...) }
    
    // Background processors
    go UserDefinedTm()                    // Custom TM injection
    go InjectTm()                         // TM injection from file
    go RedisToWampMessagePublisher()       // Real-time updates
}
```

#### Data Processing ([ReceiveScTm.go](file:///e:\Code\Mainframe\MainframeAutomation\GoLang\TM_TC\tm\ReceiveScTm.go))

- Establishes WebSocket connection to UMACS data server
- Parses incoming TM packets
- Validates against limits (upper/lower)
- Stores in Redis with timestamps
- Optionally writes to InfluxDB for historical data
- Publishes via WAMP for real-time UI updates

---

### 3. REST API Server (`restapi/`)

HTTP server running on **port 11000** providing endpoints for ground station control.

#### API Endpoints

| Handler | File | Endpoints | Description |
|---------|------|-----------|-------------|
| TM | [tm.go](file:///e:\Code\Mainframe\MainframeAutomation\GoLang\TM_TC\restapi\tm.go) | `/api/tm/*` | Get TM data, mnemonics, limits |
| Priority Queue | [PQueue.go](file:///e:\Code\Mainframe\MainframeAutomation\GoLang\TM_TC\restapi\PQueue.go) | `/api/queue/*` | Test procedure queue management |
| UMACS TC | [umacs_tc_interface.go](file:///e:\Code\Mainframe\MainframeAutomation\GoLang\TM_TC\restapi\umacs_tc_interface.go) | `/api/tc/*` | Send telecommands to spacecraft |
| File Transfer | [fileTransferToUmacs.go](file:///e:\Code\Mainframe\MainframeAutomation\GoLang\TM_TC\restapi\fileTransferToUmacs.go) | `/api/file/*` | SFTP upload to UMACS |
| MongoDB | [MongoDb.go](file:///e:\Code\Mainframe\MainframeAutomation\GoLang\TM_TC\restapi\MongoDb.go) | `/api/mongo/*` | MongoDB CRUD operations |
| ArangoDB | [ArangoDb.go](file:///e:\Code\Mainframe\MainframeAutomation\GoLang\TM_TC\restapi\ArangoDb.go) | `/api/arangodb/*` | ArangoDB operations |
| Payload State | [get_payload_state.go](file:///e:\Code\Mainframe\MainframeAutomation\GoLang\TM_TC\restapi\get_payload_state.go) | `/api/payload/*` | Spacecraft payload status |
| WAMP Publisher | [wamp_publisher.go](file:///e:\Code\Mainframe\MainframeAutomation\GoLang\TM_TC\restapi\wamp_publisher.go) | `/api/wamp/*` | Publish messages to WAMP |
| Redis Channel | [wait_for_redis_channel_value.go](file:///e:\Code\Mainframe\MainframeAutomation\GoLang\TM_TC\restapi\wait_for_redis_channel_value.go) | `/api/redis/*` | Poll Redis channels |
| Logical Expression | [logicalExpressionValidator.go](file:///e:\Code\Mainframe\MainframeAutomation\GoLang\TM_TC\restapi\logicalExpressionValidator.go) | `/api/expr/*` | Evaluate logical expressions |

#### Example: Telemetry Routes ([tm.go](file:///e:\Code\Mainframe\MainframeAutomation\GoLang\TM_TC\restapi\tm.go))

```go
func registerRoutesForTm(r *mux.Router) {
    r.HandleFunc("/api/tm/mnemonics", getTmMnemonics).Methods("GET")
    r.HandleFunc("/api/tm/data", getTmData).Methods("GET")
    r.HandleFunc("/api/tm/limits", getTmLimits).Methods("GET")
    r.HandleFunc("/api/tm/heartbeat", getHeartBeat).Methods("GET")
}
```

---

### 4. WAMP API (`wampApi/`)

Real-time pub/sub messaging for live UI updates.

#### Registered Procedures

| Procedure | Description |
|-----------|-------------|
| `scg.tm.inject_dtm` | Inject derived TM data |
| `scg.tm.get_tm_data` | Get telemetry data |
| `scg.tm.get_tm_mnemonics` | Get available TM parameters |
| `scg.db.*` | Database operations (NoSQL) |

---

### 5. TM Simulator (`tm_simulator/`)

Testing tool that simulates spacecraft telemetry data without connecting to real hardware.

**Features:**
- Generates random TM values for testing
- WebSocket server for TM stream publishing
- Supports digital and analog parameter types
- Can inject custom TC responses

**Usage:**
```bash
./tm_simulator.exe -wstm1 9050 -random
```

---

### 6. UMACS WebSocket Server (Mock)

**[umacs_websocket_server.go](file:///e:\Code\Mainframe\MainframeAutomation\GoLang\TM_TC\umacs_websocket_server.go)**

Mock server for testing procedure execution without real spacecraft.

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/createProcedure` | POST | Create new test procedure |
| `/validateProcedure` | POST | Validate procedure syntax |
| `/loadProcedure` | POST | Load procedure for execution |
| `/getExeStatus` | POST | Get execution status |

---

### 7. Shared Components (`shared/`)

#### Data Types ([datatypes.go](file:///e:\Code\Mainframe\MainframeAutomation\GoLang\TM_TC\shared\datatypes.go))

```go
// Telemetry Packet
type TmPacket struct {
    Paramid    string  `json:"param_id"`
    Param      string  `json:"param"`
    SourceInfo string  `json:"source_info"`
    RawCount   int64   `json:"raw_count"`
    ProcValue  string  `json:"proc_value"`
    TimeStamp  string  `json:"time_stamp"`
    UpperLimit float64 `json:"upper_limit"`
    LowerLimit float64 `json:"lower_limit"`
    ErrDesc    string  `json:"err_desc"`
}

// Telecommand Packet
type TcSentPkt struct {
    Cmd      string `json:"cmd"`
    Code     string `json:"code"`
    FullCode string `json:"full_code"`
    DataPart string `json:"data_part"`
    Status   string `json:"status"`
    Time     string `json:"time"`
}
```

#### Configuration ([readConfigToml.go](file:///e:\Code\Mainframe\MainframeAutomation\GoLang\TM_TC\shared\readConfigToml.go))

Loads configuration from `config.toml`:
```toml
umacs_data_server_ip = ""

[paths]
main_dist = "../../GUI/GUI/.output/public"
spasdacs_dist = "../../deployment/lib/spasdacs/generated_webapp/dist"
```

---

## Data Flow

### Telemetry Reception

```
Spacecraft UMACS
      │
      │ WebSocket (ws://IP:PORT/ws)
      │
      ▼
┌─────────────────┐
│ TmSubscriber() │  ◄── Receives TM1, TM2, SMON, ADC, PTM, TC
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
    ▼         ▼
 Redis    InfluxDB
 (real-   (history)
 time)
    │
    ▼
┌─────────────────┐
│WAMP Publisher() │  ◄── Publishes to subscribers
└────────┬────────┘
         │
         ▼
   Frontend GUI
```

### Telecommand Execution

```
Frontend GUI
      │
      │ HTTP POST /api/tc/send
      ▼
┌─────────────────┐
│ REST API Server │
└────────┬────────┘
         │
         │ HTTP/SFTP
         ▼
┌─────────────────┐
│ UMACS Interface│  ◄── Sends TC to spacecraft
└─────────────────┘
```

---

## Configuration

### config.toml

```toml
# UMACS Data Server IP (empty = use Redis config)
umacs_data_server_ip = ""

[paths]
# Frontend static files
main_dist = "../../GUI/GUI/.output/public"
# SPAS DACS webapp
spasdacs_dist = "../../deployment/lib/spasdacs/generated_webapp/dist"

[redis]
# Redis connection (hardcoded or via environment)

[database]
# MongoDB/InfluxDB/ArangoDB settings
```

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/gorilla/mux` | HTTP routing |
| `github.com/gorilla/websocket` | WebSocket client/server |
| `github.com/go-redis/redis/v8` | Redis client |
| `github.com/gammazero/nexus/v3` | WAMP protocol |
| `github.com/mongodb/mongo-driver` | MongoDB driver |
| `github.com/arangodb/go-driver` | ArangoDB driver |
| `github.com/influxdata/influxdb-client-go` | InfluxDB client |
| `github.com/spf13/viper` | Configuration management |

---

## Running the Application

### Build
```bash
cd GoLang/TM_TC
go build -o bin/tm .
```

### Run
```bash
# With default config
./bin/tm

# With custom UMACS IP
./bin/tm -umacs_data_server_ip 172.20.5.100
```

### Run Simulator (for testing)
```bash
cd tm_simulator
go build -o tm_simulator.exe .
./tm_simulator.exe -wstm1 9050 -random
```

---

## Ports

| Service | Port | Description |
|---------|------|-------------|
| REST API | 11000 | Main HTTP API |
| TM1 WS | 9050 | Primary telemetry |
| TM2 WS | 9051 | Secondary telemetry |
| Mock UMACS | 8787 | Test server |

---

## Summary

This is a **comprehensive satellite ground station system** that:

1. **Receives** real-time telemetry from spacecraft (TM data)
2. **Monitors** status parameters (SMON, ADC)
3. **Processes** telecommands and file transfers
4. **Stores** data in multiple databases (Redis, InfluxDB, MongoDB, ArangoDB)
5. **Serves** data via REST API and WAMP for real-time frontend updates
6. **Simulates** TM data for testing without hardware

The system is modular with clear separation between telemetry reception, data processing, API serving, and real-time messaging components.
