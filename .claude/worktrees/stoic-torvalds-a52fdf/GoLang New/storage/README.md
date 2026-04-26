# Storage Service

## Overview

The Storage service provides intelligent persistence of spacecraft telemetry to InfluxDB. Rather than writing every value on every tick, it applies storage rules to filter out noise: binary mnemonics are stored only on state change, analog mnemonics are stored only when the value has changed by more than the configured tolerance. Data-break boundaries are always stored regardless of rules.

---

## Architecture

```
  MongoDB  (tm_mnemonics collection)
      │  load mnemonics where enable_storage=true
      │
  ┌───┴────────────────────────────────────────────┐
  │                  Storage                        │
  │  ┌────────────────────────────────────────────┐ │
  │  │  MnemonicLoader                             │ │
  │  │  (reloads on TM_LIMIT_CHANGED pub/sub)     │ │
  │  └────────────────────────────────────────────┘ │
  │  ┌────────────────────────────────────────────┐ │
  │  │  Poll loop  (configurable interval)         │ │
  │  │  RuleEngine: change-detect + tolerance      │ │
  │  │  In-memory Cache: last stored value         │ │
  │  └────────────────────────────────────────────┘ │
  └───────────────────────┬────────────────────────┘
         │ HGETALL        │ WritePoints
         ▼                ▼
      Redis             InfluxDB
   TM_MAP             "telemetry" measurement
   TM_SOFTWARE_CFG_MAP
```

---

## How It Works

### 1. Mnemonic loading

On startup (and on every `TM_LIMIT_CHANGED` pub/sub event), all mnemonics from MongoDB with `enable_storage: true` are loaded. Dynamic reload allows adding or removing mnemonics without restarting the service.

### 2. Poll loop

A ticker fires at the configured interval. Each cycle:

1. Checks `TM_SOFTWARE_CFG_MAP[TM_STORAGE_ENABLE]`; skips the cycle if storage is globally disabled.
2. Reads all values from `TM_MAP` (`HGETALL`).
3. For each mnemonic in the loaded list:
   - Runs the rule engine to decide whether to store.
   - If yes → writes a point to InfluxDB and updates the in-memory cache.
   - If no → skips.

### 3. Storage rules (per mnemonic)

| Condition | Action |
|-----------|--------|
| `enable_storage = false` | Never store |
| First sample (no cache entry) | Always store |
| First sample after data break | Always store |
| Last sample before data break | Always store |
| `BINARY`: value changed | Store |
| `BINARY`: value unchanged | Skip |
| `ANALOG`: `abs(current - last_stored) > tolerance` | Store |
| `ANALOG`: within tolerance | Skip |

### 4. InfluxDB data model

Each stored value is written as a point to the `telemetry` measurement:

| Tag | Value |
|-----|-------|
| `mnemonic` | Mnemonic ID (e.g. `ACM05521`) |
| `subsystem` | Subsystem name (e.g. `PAYLOAD`) |
| `type` | `ANALOG` or `BINARY` |

| Field | Value |
|-------|-------|
| `value` | `float64` for ANALOG; `string` for BINARY |

Timestamp is set to the time the value was read from Redis (wall clock).

### 5. In-memory cache

The `Cache` struct holds the last-stored value for each mnemonic in memory. It is the reference point for change detection and tolerance comparison. Cache entries are marked as `IsBreak = true` when a data break is detected, ensuring the first post-break sample is always stored.

---

## Endpoints

No HTTP endpoints — background service only.

---

## Redis Keys Used

| Key | Operation | Description |
|-----|-----------|-------------|
| `TM_MAP` | HGETALL | All current telemetry values |
| `TM_SOFTWARE_CFG_MAP` | HGET | Reads `TM_STORAGE_ENABLE` global flag |
| `TM_LIMIT_CHANGED` | SUBSCRIBE | Trigger mnemonic reload |

---

## InfluxDB Output

| Measurement | Tags | Fields |
|-------------|------|--------|
| `telemetry` | `mnemonic`, `subsystem`, `type` | `value` (float64 or string) |

---

## Configuration (`config.yaml`)

```yaml
service:
  name: "storage"
  log_level: "info"

redis:
  addr: "localhost:6379"
  password: ""
  db: 0

mongodb:
  uri: "mongodb://localhost:27017"
  database: "mainframe"

influxdb:
  host: "http://localhost:8086"
  token: "your-influxdb-token"
  database: "telemetry"

storage:
  interval_ms: 1000
```

| Field | Description |
|-------|-------------|
| `influxdb.host` | InfluxDB v3 server URL |
| `influxdb.token` | Auth token |
| `influxdb.database` | InfluxDB database (bucket) name |
| `storage.interval_ms` | Poll interval in milliseconds |

---

## How to Run

```bash
# From the storage directory
go run ./cmd -config config.yaml
```

The service is also started automatically by the **launcher** when `storage.enabled: true` in `launcher/config.yaml`.
