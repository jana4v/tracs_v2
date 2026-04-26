# Simulator Service

## Overview

The Simulator generates synthetic spacecraft telemetry for testing and development without live hardware. It loads mnemonic definitions from MongoDB, produces values in either FIXED (static initial values) or RANDOM (values within defined ranges) mode, and writes them to `SIMULATED_TM_MAP` in Redis. It exposes a full REST API for control and inspection.

---

## Architecture

```
  MongoDB  (tm_mnemonics collection)
      │  load all mnemonics (with range/type metadata)
      │
  ┌───┴───────────────────────────────────────────────┐
  │                  Simulator                         │
  │  ┌──────────────────────────────────────────────┐  │
  │  │  Simulation loop                              │  │
  │  │  (start/stop via TM_SIMULATOR_CTRL_CHANNEL)  │  │
  │  │  FIXED mode: write initial values once        │  │
  │  │  RANDOM mode: write new values each tick      │  │
  │  └──────────────────────────────────────────────┘  │
  │  ┌──────────────────────────────────────────────┐  │
  │  │  HTTP API (SimulatorHandler)                  │  │
  │  └──────────────────────────────────────────────┘  │
  └───────────────────────────────────────────────────┘
          │  HSET / PUBLISH
          ▼
  Redis
  ┌────────────────────────────────────────────────────┐
  │  SIMULATED_TM_MAP      (write simulated values)    │
  │  TM_SIMULATOR_CFG_MAP  (read/write config)         │
  │  TM_SIMULATOR_CTRL_CHANNEL  (subscribe: start/stop)│
  │  TM_SIMULATOR_CHANNEL       (publish: heartbeat)   │
  └────────────────────────────────────────────────────┘
```

---

## How It Works

### 1. Mnemonic loading

On startup, all mnemonic definitions are loaded from the `tm_mnemonics` MongoDB collection. Each definition includes type (`ANALOG` or `BINARY`), `range`, `cdbMnemonic`, and `subsystem`.

### 2. Configuration initialisation

If `TM_SIMULATOR_CFG_MAP` does not already contain `ENABLE`, `MODE`, or `SAMPLE_DELAY` keys, the service writes defaults (`1`, `FIXED`, `1000`) so that the Redis config is always consistent.

### 3. Simulation loop

The loop subscribes to `TM_SIMULATOR_CTRL_CHANNEL` for `start`/`stop` commands:

- **`start`** — begins writing values to `SIMULATED_TM_MAP` and publishing heartbeats to `TM_SIMULATOR_CHANNEL`.
- **`stop`** — pauses generation and deletes `SIMULATED_TM_MAP`.

**FIXED mode** — writes each mnemonic's `range[0]` (minimum value) once and idles, re-checking mode every second.

**RANDOM mode** — on each tick (period = `SAMPLE_DELAY` ms):
- ANALOG: generates a random float64 within `[range[0], range[1]]`.
- BINARY: randomly selects one of the configured digital states.

### 4. Value generation rules

| Type | FIXED value | RANDOM value |
|------|------------|-------------|
| `ANALOG` | `range[0]` | Random float in `[range[0], range[1]]` |
| `BINARY` | First digital state | Random digital state from defined set |

---

## API Endpoints

### `PUT /simulator/values`

Manually set specific mnemonic values in `SIMULATED_TM_MAP`.

**Request**
```json
[
  { "mnemonic": "ACM05521", "value": "42.5" },
  { "mnemonic": "ACM05522", "value": "PRESENT" }
]
```

**Response**
```json
{ "success": true, "updated": 2 }
```

---

### `GET /simulator/values`

Returns current values from `SIMULATED_TM_MAP`, optionally filtered by subsystem.

**Query params**
- `?subsystem=PAYLOAD` — comma-separated list of subsystems to filter by

**Response**
```json
[
  { "mnemonic": "ACM05521", "value": "42.5" },
  { "mnemonic": "ACM05522", "value": "PRESENT" }
]
```

---

### `GET /simulator/subsystems`

Returns a sorted list of all distinct subsystem names from the loaded mnemonics.

**Response**
```json
["ADC", "PAYLOAD", "POWER", "TCS"]
```

---

### `POST /simulator/reset`

Resets all mnemonic values in `SIMULATED_TM_MAP` to their initial values (`range[0]`).

**Response**
```json
{ "success": true, "message": "all values reset to initial state" }
```

---

### `GET /simulator-status`

Returns the current contents of `TM_SIMULATOR_CFG_MAP` plus the number of loaded mnemonics.

**Response**
```json
{
  "config": {
    "ENABLE": "1",
    "MODE": "RANDOM",
    "SAMPLE_DELAY": "500"
  },
  "mnemonic_count": 312
}
```

---

### `GET /simulator/mnemonics`

Returns the full list of mnemonic definitions loaded from MongoDB.

**Response**
```json
[
  {
    "_id": "ACM05521",
    "subsystem": "PAYLOAD",
    "type": "ANALOG",
    "range": ["0", "100"],
    "cdbMnemonic": "ACM05521",
    "tolerance": 0.5,
    "enable_limit": true,
    "enable_comparison": true,
    "enable_storage": true
  }
]
```

---

### `GET /simulator/mnemonic/range`

Returns the range configuration for a single mnemonic.

**Query params**
- `?mnemonic=ACM05521` — required

**Response**
```json
{
  "mnemonic": "ACM05521",
  "type": "ANALOG",
  "range": ["0", "100"]
}
```

---

### `GET /simulator/mode`

Returns the current simulation mode.

**Response**
```json
{ "mode": "RANDOM" }
```

---

### `PUT /simulator/mode`

Sets the simulation mode. Takes effect on the next tick.

**Request**
```json
{ "mode": "FIXED" }
```

Valid values: `"FIXED"` or `"RANDOM"`.

**Response**
```json
{ "success": true, "mode": "FIXED" }
```

---

### `POST /simulator/start`

Publishes a `start` command to `TM_SIMULATOR_CTRL_CHANNEL` and sets `ENABLE = 1`.

**Response**
```json
{ "success": true, "running": true }
```

---

### `POST /simulator/stop`

Publishes a `stop` command, sets `ENABLE = 0`, and deletes `SIMULATED_TM_MAP`.

**Response**
```json
{ "success": true, "running": false }
```

---

## Redis Keys Used

| Key | Operation | Description |
|-----|-----------|-------------|
| `SIMULATED_TM_MAP` | HSET / HGETALL / DEL | Simulated mnemonic values |
| `TM_SIMULATOR_CFG_MAP` | HSET / HGETALL / HGET | Simulator configuration (`ENABLE`, `MODE`, `SAMPLE_DELAY`) |
| `TM_SIMULATOR_CTRL_CHANNEL` | SUBSCRIBE / PUBLISH | Control commands: `start`, `stop` |
| `TM_SIMULATOR_CHANNEL` | PUBLISH | Heartbeat events (value: `"heartbeat"`) |

---

## Configuration (`config.yaml`)

```yaml
service:
  name: "simulator"
  log_level: "info"

redis:
  addr: "localhost:6379"
  password: ""
  db: 0

mongodb:
  uri: "mongodb://localhost:27017"
  database: "mainframe"

http:
  port: 8083
```

---

## How to Run

```bash
# From the simulator directory
go run ./cmd -config config.yaml
```

The service is also started automatically by the **launcher** when `simulator.enabled: true` in `launcher/config.yaml`.
