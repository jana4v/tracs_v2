# TM System — Go Services

A modern, microservices-based telemetry management system built in Go 1.24. The system ingests real-time telemetry from multiple chains via WebSocket, stores it in Redis for live access, persists to InfluxDB 3 for historical queries, provides REST APIs for frontend integration, and bridges test-procedure execution to the UMACS TC hardware interface.

Implements **TM System SRS v1.1** — a clean rewrite of the legacy monolithic `TM_TC` application into independent services with a shared library.

---

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Directory Structure](#directory-structure)
- [Service Summary](#service-summary)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Building](#building)
- [Running the Services](#running-the-services)
- [Service Details](#service-details)
  - [1. Internal (Shared Library)](#1-internal-shared-library)
  - [2. Simulator](#2-simulator)
  - [3. Gateway](#3-gateway)
  - [4. Ingest](#4-ingest)
  - [5. Chain Monitor](#5-chain-monitor)
  - [6. Limiter](#6-limiter)
  - [7. Comparator](#7-comparator)
  - [8. Storage](#8-storage)
  - [9. UMACS TC](#9-umacs-tc)
  - [10. UMACS TC Emulator](#10-umacs-tc-emulator)
  - [11. Launcher](#11-launcher)
- [Redis Data Model](#redis-data-model)
- [MongoDB Schema](#mongodb-schema)
- [InfluxDB Schema](#influxdb-schema)
- [API Reference](#api-reference)
- [Key Patterns](#key-patterns)
- [Verification & Testing](#verification--testing)
- [Dependencies](#dependencies)

---

## Architecture Overview

```
  Live Hardware / Simulators
  (TM1, TM2, SMON1, ADC1 …)
           │  WebSocket
           ▼
  ┌─────────────┐     ┌──────────┐
  │   ingest    │     │ simulator│  synthetic TM for testing
  └──────┬──────┘     └────┬─────┘
         │ TM_MAP          │ SIMULATED_TM_MAP
         ▼                 ▼
  ┌──────────────────────────────────┐
  │              Redis               │
  │  TM_MAP   TM_LIMIT_FAILURES_MAP  │
  │  TM#_MAP  TM_CHAIN_MISMATCHES    │
  │  heartbeats  TC_FILES_STATUS     │
  └──┬────────────────────────────┬──┘
     │                            │
  ┌──▼──────────────┐   ┌────────▼───────┐
  │  limiter        │   │  comparator    │
  │  chainmon       │   │  storage→InfluxDB
  └──┬──────────────┘   └────────────────┘
     │
  ┌──▼──────────────┐
  │    gateway      │  REST API
  └──┬──────────────┘
     │ HTTP
     ▼
  Frontend / Test scripts / Julia

  ┌──────────────┐   HTTP POST      ┌───────────────────────┐
  │  umacs-tc   │ ──────────────►  │  UMACS TC Server :21003│
  │  :21002      │                  │  (or umacs-tc-emulator)│
  └──────────────┘                  └───────────────────────┘

  ┌──────────────┐
  │   launcher  │  starts all services from one config file
  └──────────────┘
```

**Data Flow:**
1. **ingest** — connects to telemetry sources via WebSocket, parses TM/SCOS packets, writes to Redis chain maps and `TM_MAP`
2. **chainmon** — monitors heartbeat status keys, publishes heartbeat events on Redis pub/sub
3. **limiter** — polls `TM_MAP`, checks values against MongoDB-configured limits, records violations
4. **comparator** — reads chain pairs, detects cross-chain value mismatches, records confirmed ones
5. **storage** — reads `TM_MAP`, applies write-on-change rules, persists qualifying values to InfluxDB
6. **gateway** — REST façade over Redis; consumed by the frontend and Julia test procedures
7. **simulator** — generates synthetic telemetry in FIXED/RANDOM mode for testing without hardware
8. **umacs-tc** — bridges Julia test procedures to the UMACS TC REST API via a priority command queue
9. **umacs-tc-emulator** — mock UMACS TC server for integration testing without physical hardware
10. **launcher** — orchestrates all services from a single config file

---

## Service Summary

| Directory | Port | Type | Description |
|-----------|------|------|-------------|
| [gateway/](gateway/) | 8080 | HTTP server | REST API for frontend and scripts |
| [ingest/](ingest/) | — | Background | WebSocket→Redis telemetry ingest |
| [chainmon/](chainmon/) | — | Background | Chain heartbeat monitor + publisher |
| [comparator/](comparator/) | — | Background | Cross-chain mismatch detection |
| [limiter/](limiter/) | — | Background | Telemetry limit checking |
| [simulator/](simulator/) | 8083 | HTTP server | Synthetic telemetry generator |
| [storage/](storage/) | — | Background | Smart InfluxDB persistence |
| [umacs-tc/](umacs-tc/) | 21002 | HTTP server | UMACS TC procedure bridge + queue |
| [umacs-tc-emulator/](umacs-tc-emulator/) | 21003 | HTTP server | Mock UMACS TC server for testing |
| [launcher/](launcher/) | — | Orchestrator | Starts all services from one config |
| [internal/](internal/) | — | Shared lib | Models, Redis keys, clients, logging |

---

## Directory Structure

```
GoLang New/
├── go.work                        # Go workspace linking all 11 modules
│
├── internal/                      # Shared library (used by all services)
│   ├── go.mod
│   ├── config/config.go           # YAML config loader (viper)
│   ├── models/
│   │   ├── mnemonic.go            # TmMnemonic struct (MongoDB schema)
│   │   ├── telemetry.go           # TmPacket, ScosPkt, HeartbeatPayload
│   │   └── rediskeys.go           # Redis key constants & dynamic builders
│   ├── clients/
│   │   ├── redis.go               # Redis client with retry
│   │   ├── mongo.go               # MongoDB client with retry
│   │   ├── influx.go              # InfluxDB 3 client with retry
│   │   └── websocket.go           # WebSocket with auto-reconnect
│   └── logging/logging.go         # slog JSON structured logger
│
├── simulator/                     # Service 1: TM Simulator
│   ├── go.mod
│   ├── config.yaml
│   ├── cmd/main.go
│   └── internal/
│       ├── simulator.go           # Core simulation loop
│       ├── generator.go           # RANDOM / FIXED value generators
│       └── handler.go             # HTTP API handlers
│
├── gateway/                       # Service 2: API Gateway
│   ├── go.mod
│   ├── config.yaml
│   ├── cmd/main.go
│   └── internal/
│       ├── router.go              # Route registration (chi)
│       ├── telemetry.go           # POST /get-telemetry
│       ├── chains.go              # GET /chain-status, /chain-mismatches
│       ├── limits.go              # GET /limit-failures
│       ├── simulator.go           # GET /simulator-status
│       ├── udtm.go                # PUT /udtm/values
│       └── dtm.go                 # PUT /dtm/values
│
├── ingest/                        # Service 3: Telemetry Ingest
│   ├── go.mod
│   ├── config.yaml
│   ├── cmd/main.go
│   └── internal/
│       ├── subscriber.go          # Core chain subscriber (WS → Redis)
│       ├── tmsubscriber.go        # TM packet parser
│       ├── scossubscriber.go      # SCOS/SMON/ADC packet parser
│       └── unifiedmap.go          # TM_MAP merge with fallback rules
│
├── chainmon/                      # Service 4: Chain Monitor
│   ├── go.mod
│   ├── config.yaml
│   ├── cmd/main.go
│   └── internal/
│       ├── monitor.go             # Per-chain heartbeat monitor
│       └── heartbeat.go           # Redis pub/sub heartbeat publisher
│
├── limiter/                       # Service 5: Limit Monitor
│   ├── go.mod
│   ├── config.yaml
│   ├── cmd/main.go
│   └── internal/
│       ├── monitor.go             # Core limit checking loop
│       ├── analog.go              # ANALOG range check
│       ├── digital.go             # BINARY state check
│       └── mnemonics.go           # MongoDB loader + event reload
│
├── comparator/                    # Service 6: Chain Comparator
│   ├── go.mod
│   ├── config.yaml
│   ├── cmd/main.go
│   └── internal/
│       ├── comparator.go          # Core comparison engine
│       ├── modeA.go               # Frame ID Sync mode
│       ├── modeB.go               # Time Delta mode
│       └── mnemonics.go           # MongoDB loader + event reload
│
├── storage/                       # Service 7: InfluxDB Storage
│   ├── go.mod
│   ├── config.yaml
│   ├── cmd/main.go
│   └── internal/
│       ├── writer.go              # InfluxDB point writer
│       ├── rules.go               # Storage filter rules
│       ├── cache.go               # Last-written value cache
│       └── mnemonics.go           # MongoDB loader + event reload
│
├── umacs-tc/                      # Service 8: UMACS TC Bridge
│   ├── go.mod
│   ├── config.yaml
│   ├── cmd/main.go
│   └── internal/
│       ├── handler.go             # HTTP endpoints + UMACS REST calls
│       ├── queue_consumer.go      # Priority queue consumer
│       └── config.go              # UMACS environment from Redis
│
├── umacs-tc-emulator/             # Service 9: Mock UMACS TC Server
│   ├── go.mod
│   ├── config.yaml
│   ├── cmd/main.go
│   └── internal/
│       ├── handler.go             # 4 UMACS endpoints + admin/health
│       └── store.go               # In-memory state machine + Redis push
│
└── launcher/                      # Service 10: Service Orchestrator
    ├── go.mod
    ├── config.yaml
    └── cmd/main.go                # Starts all services as goroutines
```

---

## Prerequisites

| Dependency   | Version      | Purpose                          |
|-------------|-------------|----------------------------------|
| **Go**       | 1.24+       | Language runtime                 |
| **Redis**    | 7.x+        | Real-time telemetry maps, pub/sub |
| **MongoDB**  | 7.x+        | Mnemonic definitions (`tm_mnemonics`) |
| **InfluxDB** | 3.x OSS     | Time-series telemetry storage    |

Ensure all three databases are running and accessible before starting the services.

---

## Configuration

Each service has its own `config.yaml` file. All configs share a common structure via `BaseConfig`:

```yaml
# Common to all services
service:
  name: "service-name"       # Appears in structured log output
  log_level: "info"          # debug | info | warn | error

redis:
  addr: "localhost:6379"
  password: ""
  db: 0

mongo:
  uri: "mongodb://localhost:27017"
  database: "astra"
```

Service-specific config fields are documented in each service section below.

Pass a custom config path via CLI flag:

```bash
./service -config /path/to/config.yaml
```

---

## Building

### Build All Services

```bash
cd "GoLang New"

# Sync workspace dependencies
go work sync

# Build each module
for mod in internal simulator gateway ingest chainmon limiter comparator storage umacs-tc umacs-tc-emulator launcher; do
  go build ./$mod/...
done
```

### Build a Single Service

```bash
cd "GoLang New"
go build -o bin/simulator         ./simulator/cmd/
go build -o bin/gateway           ./gateway/cmd/
go build -o bin/ingest            ./ingest/cmd/
go build -o bin/chainmon          ./chainmon/cmd/
go build -o bin/limiter           ./limiter/cmd/
go build -o bin/comparator        ./comparator/cmd/
go build -o bin/storage           ./storage/cmd/
go build -o bin/umacs-tc          ./umacs-tc/cmd/
go build -o bin/umacs-tc-emulator ./umacs-tc-emulator/cmd/
go build -o bin/launcher          ./launcher/cmd/
```

---

## Running the Services

### Recommended Startup Order

Start services in this order to satisfy data flow dependencies:

| Order | Service          | Command                                                       |
|------:|------------------|---------------------------------------------------------------|
| 1     | Simulator        | `./simulator/cmd/main -config simulator/config.yaml`          |
| 2     | Gateway          | `./gateway/cmd/main -config gateway/config.yaml`              |
| 3     | Ingest           | `./ingest/cmd/main -config ingest/config.yaml`                |
| 4     | Chain Monitor    | `./chainmon/cmd/main -config chainmon/config.yaml`            |
| 5     | Limiter          | `./limiter/cmd/main -config limiter/config.yaml`              |
| 6     | Comparator       | `./comparator/cmd/main -config comparator/config.yaml`        |
| 7     | Storage          | `./storage/cmd/main -config storage/config.yaml`              |
| 8     | UMACS TC Emulator| `./umacs-tc-emulator/cmd/main --port 21003 --redis-addr localhost:6379` |
| 9     | UMACS TC         | `./umacs-tc/cmd/main -config umacs-tc/config.yaml`            |

**Or start everything at once with the launcher:**
```bash
cd "GoLang New/launcher"
go run ./cmd -config config.yaml
```

### Graceful Shutdown

All services handle `SIGINT` (Ctrl+C) and `SIGTERM` gracefully:
- Cancel all in-flight operations via `context.Context`
- Close Redis, MongoDB, and InfluxDB connections
- Shut down HTTP servers cleanly

---

## Service Details

### 1. Internal (Shared Library)

Not a runnable service. Provides shared types, clients, and utilities imported by all services.

#### Models

**`models.TmMnemonic`** — Represents a telemetry mnemonic from MongoDB `tm_mnemonics`:

| Field              | Type       | Description                                    |
|--------------------|------------|------------------------------------------------|
| `ID`               | `string`   | Mnemonic identifier (e.g., `"ACM05521"`)       |
| `Subsystem`        | `string`   | Subsystem name                                 |
| `Type`             | `string`   | `"BINARY"` or `"ANALOG"`                       |
| `ProcessingType`   | `string`   | e.g., `"STATUS"`, `"EUCN-16B"`                 |
| `Range`            | `[]string` | ANALOG: `[min, max]`; BINARY: `[state1, ...]`  |
| `Tolerance`        | `float64`  | Delta threshold for comparison/storage         |
| `Unit`             | `string`   | e.g., `"V"`, `"INT"`                           |
| `EnableComparison` | `bool`     | Include in chain comparison                    |
| `EnableLimit`      | `bool`     | Include in limit checking                      |
| `EnableStorage`    | `bool`     | Include in InfluxDB storage                    |

**`models.TmPacket`** — TM telemetry packet from WebSocket:

| Field        | Type     | Description                |
|-------------|----------|----------------------------|
| `ParamID`   | `string` | Parameter identifier       |
| `Param`     | `string` | Mnemonic name              |
| `ProcValue` | `string` | Processed telemetry value  |
| `TimeStamp` | `string` | ISO timestamp              |
| `ErrDesc`   | `string` | Error description (contains `"break"` on data breaks) |

**`models.ScosPkt`** — SCOS/SMON/ADC packet from WebSocket:

| Field       | Type              | Description                |
|------------|-------------------|----------------------------|
| `ParamList` | `[]ScosParamInfo` | Array of parameter entries |
| `Stream`    | `string`          | Stream identifier          |
| `Error`     | `string`          | Error description          |

#### Client Factories

All clients use **exponential backoff retry** (10 attempts, 1s initial, 30s max, 2x multiplier):

| Client                  | Function                                | Returns              |
|------------------------|-----------------------------------------|----------------------|
| `clients.NewRedisClient`  | `(ctx, addr, password, db, logger)`   | `*redis.Client`      |
| `clients.NewMongoClient`  | `(ctx, uri, logger)`                  | `*mongo.Client`      |
| `clients.NewInfluxClient` | `(ctx, url, token, database, logger)` | `*influxdb3.Client`  |

#### WebSocket Subscriber

`clients.WSSubscriber` handles persistent WebSocket connections with auto-reconnect:

```go
ws := &clients.WSSubscriber{
    URL:          "ws://172.20.26.1:9050/ws",
    ChainName:    "TM1",
    OnMessage:    func(ctx context.Context, msg []byte) error { ... },
    OnConnect:    func(ctx context.Context) { ... },
    OnDisconnect: func(ctx context.Context, err error) { ... },
    Logger:       logger,
}
go ws.Run(ctx)  // runs until context is cancelled
```

On connect, it automatically sends a subscribe message:
```json
{"action": "subscribe", "param_list": [""]}
```

---

### 2. Simulator

Generates synthetic telemetry data for testing without live hardware.

**Config (`simulator/config.yaml`):**
```yaml
service:
  name: "tm-simulator"
  log_level: "info"
redis:
  addr: "localhost:6379"
  db: 0
mongo:
  uri: "mongodb://localhost:27017"
  database: "astra"
http:
  port: 21001
```

**How it works:**
1. Loads all mnemonics from MongoDB `tm_mnemonics` collection
2. Reads runtime config from Redis `TM_SIMULATOR_CFG_MAP` every cycle:
   - `ENABLE` — `1` to activate, `0` to pause
   - `MODE` — `"RANDOM"` or `"FIXED"`
   - `SAMPLE_DELAY` — milliseconds between cycles (default: 1000)
3. Generates values per mnemonic and writes to `SIMULATED_TM_MAP` via Redis pipeline
4. Publishes heartbeat to `TM_SIMULATOR_CHANNEL`

**Value Generation:**

| Mode     | ANALOG                                  | BINARY                          |
|----------|------------------------------------------|---------------------------------|
| `RANDOM` | Random float in `[Range[0], Range[1]]`  | Random pick from `Range` array  |
| `FIXED`  | Always `Range[0]`                       | Always `Range[0]`               |

**HTTP Endpoints (port 21001):**

| Method | Path                  | Description                           |
|--------|-----------------------|---------------------------------------|
| PUT    | `/simulator/values`   | Override specific mnemonic values     |
| POST   | `/simulator/reset`    | Reset all values to `Range[0]`        |
| GET    | `/simulator-status`   | Get current config and mnemonic count |

**Example — Override a value:**
```bash
curl -X PUT http://localhost:21001/simulator/values \
  -H "Content-Type: application/json" \
  -d '[{"mnemonic": "ACM05521", "value": "ABSENT"}]'
```

**Example — Check status:**
```bash
curl http://localhost:21001/simulator-status
# {"config":{"ENABLE":"1","MODE":"RANDOM","SAMPLE_DELAY":"1000"},"mnemonic_count":500}
```

---

### 3. Gateway

REST API gateway for frontend and external clients. Reads from Redis.

**Config (`gateway/config.yaml`):**
```yaml
service:
  name: "tm-gateway"
  log_level: "info"
redis:
  addr: "localhost:6379"
  db: 0
http:
  port: 21000
```

**Endpoints:**

| Method | Path                 | Description                                         |
|--------|----------------------|-----------------------------------------------------|
| POST   | `/get-telemetry`     | Batch-read mnemonic values from Redis               |
| GET    | `/chain-status`      | Heartbeat status of all configured chains           |
| GET    | `/chain-mismatches`  | Current comparison mismatches between chain pairs   |
| GET    | `/limit-failures`    | Current limit violations                            |
| GET    | `/simulator-status`  | Simulator configuration and status                  |
| PUT    | `/udtm/values`       | Write user-defined telemetry values                 |
| PUT    | `/dtm/values`        | Write derived telemetry values                      |

See [API Reference](#api-reference) for request/response formats.

---

### 4. Ingest

Connects to UMACS telemetry sources via WebSocket and populates Redis.

**Config (`ingest/config.yaml`):**
```yaml
service:
  name: "tm-ingest"
  log_level: "info"
redis:
  addr: "localhost:6379"
  db: 0
websocket:
  retry_initial: "1s"
  retry_max: "30s"
  retry_multiplier: 2
chains:
  - name: TM1
    type: TM            # "TM" or "SCOS"
    host: "172.20.26.1"
    port: 9050
  - name: TM2
    type: TM
    host: "172.20.26.1"
    port: 9051
  - name: SMON1
    type: SCOS
    host: "172.20.26.1"
    port: 9060
  - name: ADC1
    type: SCOS
    host: "172.20.26.1"
    port: 9011
```

**How it works:**
1. Launches one goroutine per configured chain
2. Each goroutine connects via WebSocket to `ws://<host>:<port>/ws`
3. Parses incoming packets (TM or SCOS format)
4. Writes to chain-specific Redis map (e.g., `TM1_MAP`) via pipeline
5. Merges into unified `TM_MAP` using fallback rules
6. Sets heartbeat status key with 2-second TTL
7. Detects data breaks via `"break"` keyword in `err_desc`

**Unified Map Fallback Rules (SRS 9.2):**

| Chain Type | Write Behavior                                    |
|------------|---------------------------------------------------|
| TM1, TM2   | Direct write to `TM_MAP` (last writer wins)      |
| SMON1      | Direct write to `TM_MAP` (primary SCOS)          |
| SMON2+     | Write with `_SMON<N>` suffix (redundant)         |
| ADC1       | Direct write to `TM_MAP` (primary ADC)           |
| ADC2+      | Write with `_ADC<N>` suffix (redundant)          |

---

### 5. Chain Monitor

Monitors chain health and publishes heartbeat messages.

**Config (`chainmon/config.yaml`):**
```yaml
service:
  name: "tm-chainmon"
  log_level: "info"
redis:
  addr: "localhost:6379"
  db: 0
chains:
  - name: TM1
    type: TM
    timeout_seconds: 1
  - name: SMON1
    type: SMON
    timeout_seconds: 5
```

**How it works:**
1. Checks each chain's heartbeat status key in Redis every 5 seconds
2. If status is `"OK"`, publishes heartbeat to `<CHAIN>_HEARTBEAT_CHANNEL`
3. Heartbeat payload includes chain name, status, and timestamps

**Heartbeat Status State Machine:**
```
CONNECTION_FAILED → CONNECTED → OK → DATA_BREAK
                                ↑         │
                                └─────────┘
```

---

### 6. Limiter

Checks telemetry values against configured limits.

**Config (`limiter/config.yaml`):**
```yaml
service:
  name: "tm-limiter"
  log_level: "info"
redis:
  addr: "localhost:6379"
  db: 0
mongo:
  uri: "mongodb://localhost:27017"
  database: "astra"
poll_interval_ms: 500
```

**How it works:**
1. Loads mnemonics with `enable_limit: true` from MongoDB
2. Subscribes to `TM_LIMIT_CHANGED` for event-driven mnemonic reload
3. Every `poll_interval_ms`, reads all values from `TM_MAP`
4. For each mnemonic, checks limits:
   - **ANALOG**: `Range[0] <= value <= Range[1]`
   - **BINARY**: `value == expected` (from `TM_EXPECTED_DIGITAL_STATES_MAP`)
5. Writes violations to `TM_LIMIT_FAILURES_MAP` as JSON
6. Clears resolved violations automatically

**Limit Violation JSON:**
```json
{
  "mnemonic": "ACM05521",
  "type": "ANALOG",
  "value": "42.5",
  "min": "0.0",
  "max": "40.0",
  "timestamp": "2026-02-20T12:34:56Z"
}
```

---

### 7. Comparator

Compares telemetry values between redundant chain pairs.

**Config (`comparator/config.yaml`):**
```yaml
service:
  name: "tm-comparator"
  log_level: "info"
redis:
  addr: "localhost:6379"
  db: 0
mongo:
  uri: "mongodb://localhost:27017"
  database: "astra"
compare_pairs:
  - chain1: TM1
    chain2: TM2
poll_interval_ms: 500
```

**How it works:**
1. Loads mnemonics with `enable_comparison: true` from MongoDB
2. Every `poll_interval_ms`, reads values from both chains in each pair
3. Compares values using type-appropriate logic:
   - **ANALOG**: `|value1 - value2| <= 2 * tolerance`
   - **BINARY**: exact string match
4. On mismatch, delegates to the active comparison mode for confirmation

**Comparison Modes (from `TM_SOFTWARE_CFG_MAP`):**

| Mode | Name            | Behavior                                            |
|------|-----------------|-----------------------------------------------------|
| `A`  | Frame ID Sync   | Waits for matching frame IDs, then re-checks        |
| `B`  | Time Delta      | Waits `CHAIN_COMPARE_DELAY_SECONDS`, then re-checks |

Confirmed mismatches are written to `TM_CHAIN_MISMATCHES_MAP`. Transient mismatches (resolved after delay/sync) are discarded.

---

### 8. Storage

Persists telemetry to InfluxDB 3 with intelligent filtering.

**Config (`storage/config.yaml`):**
```yaml
service:
  name: "tm-storage"
  log_level: "info"
redis:
  addr: "localhost:6379"
  db: 0
mongo:
  uri: "mongodb://localhost:27017"
  database: "astra"
influx:
  url: "http://localhost:8086"
  token: ""
  org: "mainframe"
  database: "telemetry"
poll_interval_ms: 500
```

**How it works:**
1. Loads mnemonics with `enable_storage: true` from MongoDB
2. Every `poll_interval_ms`, reads values from `TM_MAP`
3. Applies storage rules (write-on-change) to reduce InfluxDB writes
4. Writes qualifying points to InfluxDB

**Storage Rules:**

| Rule                   | Applies To | Condition                                 |
|------------------------|------------|-------------------------------------------|
| First sample           | All        | Always store (new or after break)         |
| Value change           | BINARY     | Store only when value differs from cached |
| Delta threshold        | ANALOG     | Store when `|current - cached| > tolerance` |
| Data break (last)      | All        | Mark last sample before break             |
| Data break (first)     | All        | Always store first sample after break     |
| Global enable          | All        | Skip if `TM_STORAGE_ENABLE` is disabled   |

---

### 9. UMACS TC

Bridges Julia test procedures to the physical UMACS TC hardware interface via a priority command queue. Ensures only one SEND command reaches the hardware at a time.

**Config (`umacs-tc/config.yaml`):**
```yaml
service:
  name: "umacs-tc"
  log_level: "info"
redis:
  addr: "localhost:6379"
  db: 0
umacs:
  tc_ip: "172.20.xx.xx"
  tc_port: 21003
http:
  port: 21002
```

UMACS TC environment variables (IP, port, credentials) can also be read from the `ENV_VARIABLES_UMACS` Redis hash at startup, overriding `config.yaml` values.

**How it works:**
1. Exposes an HTTP API on port 21002 for Julia procedures
2. Maintains a Redis sorted-set priority queue (`TC_COMMAND_QUEUE`) to serialise SEND commands
3. `queue_consumer` pops the highest-priority command with `BZPOPMIN`, calls the UMACS TC hardware, and publishes the result to `TC_COMMAND_COMPLETED:{request_id}`
4. For load/send operations, polls `TC_FILES_STATUS` until execution completes or times out
5. Supports EXPECTED suppression: before issuing SEND, writes suppression entries to `TM_LIMIT_SUPPRESSION_MAP` so the limiter skips transient violations

**HTTP Endpoints (port 21002):**

| Method | Path                        | Description                                         |
|--------|-----------------------------|-----------------------------------------------------|
| POST   | `/create-procedure`         | Create and validate a procedure on UMACS TC         |
| POST   | `/validate-procedure`       | Re-validate a procedure (syntax check only)         |
| POST   | `/load-procedure`           | Load procedure and wait for execution completion    |
| GET    | `/get-exe-status`           | Poll current execution status for a procedure       |
| POST   | `/send`                     | Queue a SEND command at specified priority          |
| GET    | `/queue-status`             | Current queue length and pending commands           |
| POST   | `/clear-queue`              | Remove all pending commands from the queue          |

**`POST /create-procedure`**

Request:
```json
{
  "proc_name": "PROC_INIT",
  "proc_src": "BEGIN\n  SEND CMD1;\nEND",
  "proc_mode": "1",
  "proc_priority": "0"
}
```

Response:
```json
{
  "proc_name": "PROC_INIT",
  "proc_id": "proc-abc-123",
  "status": "created",
  "error_msg": ""
}
```

**`POST /load-procedure`**

Request:
```json
{
  "proc_name": "PROC_INIT",
  "proc_id":   "proc-abc-123",
  "request_id": "req-xyz-456"
}
```

Response (after polling `TC_FILES_STATUS` until done):
```json
{
  "proc_name": "PROC_INIT",
  "exe_status": "SUCCESS",
  "error_msg": ""
}
```

**`POST /send`**

Request:
```json
{
  "proc_name":  "PROC_INIT",
  "proc_id":    "proc-abc-123",
  "request_id": "req-xyz-456",
  "priority":   1
}
```

The command is enqueued at `priority` score in `TC_COMMAND_QUEUE`. Lower score = higher priority.

**Redis Keys Used:**

| Key | Operation | Description |
|-----|-----------|-------------|
| `TC_COMMAND_QUEUE` | BZPOPMIN / ZADD | Priority sorted-set of SEND commands |
| `TC_FILES_STATUS` | HGET | Per-procedure execution status (set by UMACS TC or emulator) |
| `TC_COMMAND_COMPLETED:{request_id}` | PUBLISH | Queue consumer publishes result |
| `TM_LIMIT_SUPPRESSION_MAP` | HSET | Pre-declared EXPECTED suppressions |
| `ENV_VARIABLES_UMACS` | HGETALL | UMACS environment config override |

---

### 10. UMACS TC Emulator

A lightweight mock UMACS TC HTTP server for integration testing without physical hardware. Implements the same four-endpoint UMACS interface and optionally pushes status updates to Redis so `umacs-tc` can poll `TC_FILES_STATUS` normally.

**How to run:**
```bash
cd "GoLang New/umacs-tc-emulator"
go run ./cmd \
  --port 21003 \
  --inprogress-duration 4000 \
  --success-rate 100 \
  --redis-addr localhost:6379
```

**CLI Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--host` | `""` | Bind address (empty = all interfaces) |
| `--port` | `21003` | HTTP listen port (matches UMACS TC default) |
| `--queued-delay` | `500` | ms in QUEUED state before moving to IN-PROGRESS |
| `--inprogress-duration` | `4000` | ms in IN-PROGRESS before completing |
| `--success-rate` | `100` | % chance of SUCCESS (remainder → FAILURE) |
| `--no-validate-required` | false | If set, load succeeds without prior validate |
| `--redis-addr` | `""` | Redis address; if set, writes TC_FILES_STATUS |
| `--redis-password` | `""` | Redis password |
| `--redis-db` | `0` | Redis DB index |

**Procedure State Machine:**
```
create  → [CREATED]
validate → [VALIDATED]
load    → [QUEUED] → (queued-delay ms) → [IN-PROGRESS] → (inprogress-duration ms) → [SUCCESS | FAILURE]
```

**HTTP Endpoints (port 21003):**

| Method | Path                 | Description |
|--------|----------------------|-------------|
| POST   | `/createProcedure`   | Create a procedure record |
| POST   | `/validateProcedure` | Mark procedure as validated |
| POST   | `/loadProcedure`     | Start execution (async state machine) |
| POST   | `/getExeStatus`      | Poll current execution status |
| GET    | `/admin/procedures`  | List all in-memory procedures (debug) |
| GET    | `/health`            | Liveness check |

**`POST /createProcedure`**

Request:
```json
{
  "proc_name": "PROC_INIT",
  "proc_src":  "BEGIN\n  SEND CMD1;\nEND",
  "proc_mode": 1,
  "proc_priority": 0
}
```

Response:
```json
{
  "proc_name": "PROC_INIT",
  "status":    "created",
  "error_msg": ""
}
```

**`POST /validateProcedure`**

Request:
```json
{
  "proc_name": "PROC_INIT",
  "proc_src":  "BEGIN\n  SEND CMD1;\nEND"
}
```

Response:
```json
{
  "proc_name": "PROC_INIT",
  "status":    "validated",
  "error_msg": ""
}
```

**`POST /loadProcedure`**

Request:
```json
{
  "proc_name": "PROC_INIT"
}
```

Response (immediate, before execution completes):
```json
{
  "proc_name":  "PROC_INIT",
  "exe_status": "QUEUED",
  "error_msg":  ""
}
```

**`POST /getExeStatus`**

Request:
```json
{
  "proc_name": "PROC_INIT"
}
```

Response (poll until SUCCESS or FAILURE):
```json
{
  "proc_name":  "PROC_INIT",
  "exe_status": "SUCCESS",
  "error_msg":  ""
}
```

Possible `exe_status` values: `QUEUED`, `IN-PROGRESS`, `SUCCESS`, `FAILURE`, `NOT FOUND`

**Redis Integration:**

When `--redis-addr` is supplied, the emulator writes status transitions to `TC_FILES_STATUS`:
```
TC_FILES_STATUS[PROC_INIT] = "QUEUED"      → "IN-PROGRESS" → "SUCCESS" / "FAILURE"
```
This allows `umacs-tc` to poll `TC_FILES_STATUS` normally, just as it does against real hardware.

**GET /admin/procedures**

Returns all in-memory procedure records — useful for debugging test runs:
```json
[
  {
    "name": "PROC_INIT",
    "validated": true,
    "exe_status": "SUCCESS",
    "mode": 1,
    "priority": 0
  }
]
```

---

### 11. Launcher

Orchestrates all services as goroutines within a single process. One config file controls which services are enabled and where their individual configs live.

**Config (`launcher/config.yaml`):**
```yaml
services:
  simulator:
    enabled: true
    config: "../simulator/config.yaml"
  gateway:
    enabled: true
    config: "../gateway/config.yaml"
  ingest:
    enabled: true
    config: "../ingest/config.yaml"
  chainmon:
    enabled: true
    config: "../chainmon/config.yaml"
  limiter:
    enabled: true
    config: "../limiter/config.yaml"
  comparator:
    enabled: true
    config: "../comparator/config.yaml"
  storage:
    enabled: true
    config: "../storage/config.yaml"
  umacs_tc:
    enabled: false
    config: "../umacs-tc/config.yaml"
```

**How it works:**
1. Reads `launcher/config.yaml`
2. For each service where `enabled: true`, calls the service's `service.Run(ctx, configPath)` function in a goroutine
3. Each goroutine is wrapped in a `recover()` so a panicking service does not crash others
4. On `SIGINT`/`SIGTERM`, cancels the shared context — all services shut down gracefully within 30 seconds

**How to run:**
```bash
cd "GoLang New/launcher"
go run ./cmd -config config.yaml
```

**Startup order:**

| Order | Service | Notes |
|------:|---------|-------|
| 1 | Simulator | No dependencies |
| 2 | Gateway | Reads from Redis (may start before data flows) |
| 3 | Ingest | Connects to WebSocket sources |
| 4 | Chain Monitor | Depends on Ingest writing heartbeat keys |
| 5 | Limiter | Depends on Ingest writing TM_MAP |
| 6 | Comparator | Depends on Ingest writing chain maps |
| 7 | Storage | Depends on Ingest writing TM_MAP |
| 8 | UMACS TC | Started separately, requires UMACS hardware or emulator |

All services start concurrently but are designed to handle the case where their data sources are not yet available (they retry connections and skip empty data gracefully).

---

## Redis Data Model

### Hash Maps

| Key                              | Description                                        |
|----------------------------------|----------------------------------------------------|
| `TM_MAP`                        | Unified telemetry values (merged from all chains)  |
| `TM1_MAP`, `TM2_MAP`, ...       | Chain-specific telemetry values                    |
| `SMON1_MAP`, `ADC1_MAP`, ...    | SCOS chain-specific values                         |
| `SIMULATED_TM_MAP`              | Simulator-generated values                         |
| `UDTM_MAP`                      | User-defined telemetry values                      |
| `DTM_MAP`                       | Derived telemetry values                           |
| `TM_CHAIN_MISMATCHES_MAP`       | Confirmed comparison mismatches (JSON)             |
| `TM_LIMIT_FAILURES_MAP`         | Active limit violations (JSON)                     |
| `TM_EXPECTED_DIGITAL_STATES_MAP`| Expected values for BINARY mnemonics               |
| `TM_LIMIT_SUPPRESSION_MAP`      | EXPECTED suppression entries (with TTL)            |
| `TM_SIMULATOR_CFG_MAP`          | Simulator runtime config (ENABLE, MODE, DELAY)     |
| `TM_SOFTWARE_CFG_MAP`           | Global system config (compare mode, timeouts)      |
| `TC_FILES_STATUS`               | Per-procedure execution status from UMACS TC       |
| `ENV_VARIABLES_UMACS`           | UMACS TC environment config (IP, port, credentials)|

### Pub/Sub Channels

| Channel                          | Publisher     | Subscriber(s)        | Payload       |
|----------------------------------|---------------|----------------------|---------------|
| `TM_SIMULATOR_CHANNEL`          | Simulator     | —                    | `"heartbeat"` |
| `TM_LIMIT_CHANGED`              | External/API  | Limiter, Comparator  | Any (trigger) |
| `TM1_HEARTBEAT_CHANNEL`, ...    | Chain Monitor | Frontend             | JSON payload  |
| `TC_COMMAND_COMPLETED:{id}`     | Queue consumer| UMACS TC handler     | Result JSON   |

### Sorted Sets (Priority Queues)

| Key | Operations | Description |
|-----|-----------|-------------|
| `TC_COMMAND_QUEUE` | ZADD / BZPOPMIN | SEND commands keyed by priority score (lower = higher priority) |

### Key-Value Keys (with TTL)

| Key Pattern                | Set By  | TTL | Description              |
|---------------------------|---------|-----|--------------------------|
| `<CHAIN>_HEART_BEAT`      | Ingest  | 2s  | Chain heartbeat status   |
| `<CHAIN>_LAST_DATA_TIME`  | Ingest  | —   | Last data reception time |

### Simulator Config Fields (`TM_SIMULATOR_CFG_MAP`)

| Field          | Values           | Default   |
|----------------|------------------|-----------|
| `ENABLE`       | `"0"` / `"1"`   | `"0"`     |
| `MODE`         | `"RANDOM"` / `"FIXED"` | `"RANDOM"` |
| `SAMPLE_DELAY` | Milliseconds     | `"1000"`  |

### Software Config Fields (`TM_SOFTWARE_CFG_MAP`)

| Field                           | Values      | Default |
|---------------------------------|-------------|---------|
| `CHAIN_COMPARE_MODE`            | `"A"` / `"B"` | `"B"` |
| `CHAIN_COMPARE_DELAY_SECONDS`   | Seconds     | `"1"`   |
| `MASTER_FRAME_WAIT_COUNT`       | Integer     | `"3"`   |
| `TM_DATA_TIMEOUT_SECONDS`       | Seconds     | `"1"`   |
| `SMON_DATA_TIMEOUT_SECONDS`     | Seconds     | `"5"`   |
| `ADC_DATA_TIMEOUT_SECONDS`      | Seconds     | `"1"`   |
| `TM_STORAGE_ENABLE`             | `"0"` / `"1"` | `"1"` |

---

## MongoDB Schema

### Collection: `tm_mnemonics`

```json
{
  "_id": "ACM05521",
  "subsystem": "ACS",
  "type": "BINARY",
  "processing_type": "STATUS",
  "range": ["PRESENT", "ABSENT"],
  "tolerance": 0.0,
  "unit": "",
  "cdb_mnemonic": "ACM05521 - ACS Mode Status",
  "source_file": "tm_definitions.xlsx",
  "enable_comparison": true,
  "enable_limit": true,
  "enable_storage": true
}
```

**ANALOG example:**
```json
{
  "_id": "PWR00101",
  "subsystem": "PWR",
  "type": "ANALOG",
  "processing_type": "EUCN-16B",
  "range": ["0.0", "42.5"],
  "tolerance": 0.1,
  "unit": "V",
  "enable_comparison": true,
  "enable_limit": true,
  "enable_storage": true
}
```

---

## InfluxDB Schema

### Measurement: `telemetry`

| Component | Name        | Type     | Description                     |
|-----------|-------------|----------|---------------------------------|
| Tag       | `mnemonic`  | string   | Mnemonic ID                     |
| Tag       | `subsystem` | string   | Subsystem name                  |
| Tag       | `type`      | string   | `"BINARY"` or `"ANALOG"`       |
| Field     | `value`     | float/string | ANALOG as float64, BINARY as string |
| Timestamp | —           | RFC3339  | Sample timestamp                |

---

## API Reference

### POST /get-telemetry

Batch-read telemetry values from Redis.

**Request:**
```json
{
  "mnemonics": ["ACM05521", "PWR00101"],
  "source": "TM1"
}
```

- `source` is optional. Defaults to `TM_MAP`. Use `"TM1"`, `"SMON1"`, etc. to read from a specific chain.

**Response:**
```json
{
  "results": {
    "ACM05521": "PRESENT",
    "PWR00101": "28.340000"
  }
}
```

### GET /chain-status

**Response:**
```json
{
  "chains": [
    {"chain": "TM1", "status": "active"},
    {"chain": "TM2", "status": "inactive"},
    {"chain": "SMON1", "status": "active"}
  ]
}
```

### GET /chain-mismatches

**Response:**
```json
{
  "mismatches": {
    "ACM05521": "{\"mnemonic\":\"ACM05521\",\"chain1\":\"TM1\",\"chain2\":\"TM2\",\"value1\":\"PRESENT\",\"value2\":\"ABSENT\",\"type\":\"BINARY\",\"timestamp\":\"2026-02-20T12:34:56Z\"}"
  }
}
```

### GET /limit-failures

**Response:**
```json
{
  "failures": {
    "PWR00101": "{\"mnemonic\":\"PWR00101\",\"type\":\"ANALOG\",\"value\":\"42.5\",\"min\":\"0.0\",\"max\":\"40.0\",\"timestamp\":\"2026-02-20T12:34:56Z\"}"
  }
}
```

### GET /simulator-status

**Response:**
```json
{
  "ENABLE": "1",
  "MODE": "RANDOM",
  "SAMPLE_DELAY": "1000"
}
```

### PUT /udtm/values

Write user-defined telemetry to both `UDTM_MAP` and `TM_MAP`.

**Request:**
```json
[
  {"mnemonic": "PARAM1", "value": "VALUE1"},
  {"mnemonic": "PARAM2", "value": "VALUE2"}
]
```

**Response:**
```json
{
  "success": true,
  "updated": 2
}
```

### PUT /dtm/values

Write derived telemetry to `DTM_MAP`.

**Request/Response:** Same format as `/udtm/values`.

---

## Key Patterns

### 1. Exponential Backoff Retry

All external client connections (Redis, MongoDB, InfluxDB) use a consistent retry pattern:

```
Attempt 1: wait 1s
Attempt 2: wait 2s
Attempt 3: wait 4s
...
Attempt N: wait min(2^N, 30s)
Max attempts: 10
```

### 2. Redis Pipeline Writes

All batch writes use Redis pipelines for performance — a single round-trip instead of N individual HSET calls:

```go
pipe := rdb.Pipeline()
for _, m := range mnemonics {
    pipe.HSet(ctx, "TM_MAP", m.ID, value)
}
pipe.Exec(ctx)
```

### 3. Write-on-Change (Storage)

InfluxDB writes are filtered to reduce storage volume:
- BINARY values: only store when the value changes
- ANALOG values: only store when delta exceeds tolerance
- Data breaks: always store boundary samples

### 4. Event-Driven Config Reload

Limiter and Comparator subscribe to `TM_LIMIT_CHANGED` via Redis pub/sub. When mnemonics are updated in MongoDB, publishing any message to this channel triggers an immediate reload — no restart required.

### 5. Context-Based Graceful Shutdown

All services use `signal.NotifyContext` to propagate cancellation:

```go
ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer cancel()
// All goroutines receive ctx and check ctx.Done()
```

### 6. Dynamic Chain Configuration

Chains are defined as config arrays, not hardcoded. Adding a new chain (e.g., TM3, SMON2) requires only a config change — no code modifications.

---

## Verification & Testing

### 1. Build Verification

```bash
cd "GoLang New"
for mod in internal simulator gateway ingest chainmon limiter comparator storage umacs-tc umacs-tc-emulator launcher; do
  echo "=== $mod ===" && go build ./$mod/... && echo "OK"
done
```

### 2. Simulator Smoke Test

```bash
# Start simulator
./simulator/cmd/main -config simulator/config.yaml

# Enable simulator via Redis
redis-cli HSET TM_SIMULATOR_CFG_MAP ENABLE 1 MODE RANDOM SAMPLE_DELAY 1000

# Verify data flowing
redis-cli HGETALL SIMULATED_TM_MAP
```

### 3. Gateway Smoke Test

```bash
# Start gateway
./gateway/cmd/main -config gateway/config.yaml

# Query telemetry
curl -X POST http://localhost:21000/get-telemetry \
  -H "Content-Type: application/json" \
  -d '{"mnemonics": ["ACM05521"]}'

# Check chain status
curl http://localhost:21000/chain-status
```

### 4. Ingest Verification

```bash
# Start ingest (requires UMACS WebSocket server or mock)
./ingest/cmd/main -config ingest/config.yaml

# Verify chain maps
redis-cli HGETALL TM1_MAP
redis-cli HGETALL TM_MAP
```

### 5. Limiter Verification

```bash
# Inject out-of-range value
redis-cli HSET TM_MAP PWR00101 999.0

# Start limiter, then check
redis-cli HGETALL TM_LIMIT_FAILURES_MAP
```

### 6. Comparator Verification

```bash
# Set different values on paired chains
redis-cli HSET TM1_MAP ACM05521 PRESENT
redis-cli HSET TM2_MAP ACM05521 ABSENT

# Start comparator, then check
redis-cli HGETALL TM_CHAIN_MISMATCHES_MAP
```

### 7. Storage Verification

```bash
# Start storage (requires InfluxDB 3)
./storage/cmd/main -config storage/config.yaml

# Query InfluxDB for written points
influx3 query "SELECT * FROM telemetry WHERE mnemonic = 'ACM05521' LIMIT 10"
```

---

## Dependencies

| Package                                    | Version    | Used By                      | Purpose                      |
|--------------------------------------------|-----------|------------------------------|------------------------------|
| `github.com/redis/go-redis/v9`            | v9.7.0    | All services                 | Redis client                 |
| `go.mongodb.org/mongo-driver/v2`          | v2.0.1    | All except gateway/emulator  | MongoDB driver               |
| `github.com/gorilla/websocket`            | v1.5.3    | Ingest                       | WebSocket client             |
| `github.com/InfluxCommunity/influxdb3-go/v2` | v2.13.0 | Storage                     | InfluxDB 3 OSS client        |
| `github.com/go-chi/chi/v5`               | v5.2.1    | Gateway                      | HTTP router                  |
| `github.com/spf13/viper`                 | v1.19.0   | All services                 | YAML config loading          |
| `log/slog` (stdlib)                       | Go 1.24   | All services                 | Structured JSON logging      |

### Key Changes from Legacy Code

| Aspect              | Old (`TM_TC`)                  | New (`GoLang New`)                   |
|---------------------|--------------------------------|--------------------------------------|
| Architecture        | 1 monolith                     | 10 independent microservices         |
| Messaging           | WAMP (gammazero/nexus)         | Redis pub/sub                        |
| Chain Configuration | Hard-coded TM1/TM2/SMON1/SMON2| Config-driven array (dynamic)        |
| Logging             | `log.New(os.Stdout, "", 0)`    | `slog` structured JSON              |
| Shutdown            | None (kill process)            | Graceful via context + signals       |
| Redis Client        | `go-redis/v8`                  | `go-redis/v9` (context-first)       |
| MongoDB Driver      | `mongo-driver` v1              | `mongo-driver/v2`                    |
| InfluxDB Client     | `influxdb-client-go/v2`        | `influxdb3-go` (InfluxDB 3 OSS)     |
| HTTP Router         | `gorilla/mux`                  | `go-chi/chi/v5`                      |
| Config Reload       | 5-second polling               | Event-driven (Redis pub/sub)         |
