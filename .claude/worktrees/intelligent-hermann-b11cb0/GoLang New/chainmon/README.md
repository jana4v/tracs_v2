# ChainMon Service

## Overview

ChainMon (Chain Monitor) watches the heartbeat status of every configured telemetry chain and publishes structured heartbeat events on Redis pub/sub channels. Other services (and the frontend via WebSocket) subscribe to these channels to receive real-time chain health updates.

---

## Architecture

```
  Redis
  ┌──────────────────────────────────┐
  │  TM1_HEART_BEAT (string, TTL 2s) │  ← written by ingest
  │  TM2_HEART_BEAT                  │
  │  SMON1_HEART_BEAT                │
  │  TM1_LAST_DATA_TIME              │
  └──────────┬───────────────────────┘
             │  GET every 5 s (per chain goroutine)
  ┌──────────┴───────────────────────┐
  │        ChainMon                  │
  │  (one goroutine per chain)       │
  └──────────┬───────────────────────┘
             │  PUBLISH
             ▼
  Redis pub/sub channels
  ┌────────────────────────────────────┐
  │  TM1_HEARTBEAT_CHANNEL             │
  │  TM2_HEARTBEAT_CHANNEL             │
  │  SMON1_HEARTBEAT_CHANNEL           │
  └────────────────────────────────────┘
```

---

## How It Works

### 1. Per-chain monitor goroutines

One `ChainMonitor` goroutine is started for each chain in `config.yaml`. Goroutines run completely independently.

### 2. Polling loop (every 5 seconds per chain)

On each tick, the monitor reads `<CHAIN>_HEART_BEAT` from Redis:

| Status value | Action |
|-------------|--------|
| `OK` | Reads `<CHAIN>_LAST_DATA_TIME`, publishes an ACTIVE heartbeat payload to `<CHAIN>_HEARTBEAT_CHANNEL` |
| Anything else (or key missing) | No publish — silence signals inactivity to subscribers |

The 2-second TTL on `_HEART_BEAT` keys (set by ingest) means a chain that stops sending data will appear inactive within ~2 seconds even if ChainMon polls on a 5-second interval.

### 3. Heartbeat payload

The JSON payload published on each active heartbeat:

```json
{
  "chain":       "TM1",
  "status":      "ACTIVE",
  "last_data_ts":"2026-02-25T10:00:00Z",
  "timestamp":   "2026-02-25T10:00:05Z"
}
```

| Field | Description |
|-------|-------------|
| `chain` | Chain name (e.g. `TM1`) |
| `status` | Always `ACTIVE` when published |
| `last_data_ts` | Timestamp of the last data point received from the chain |
| `timestamp` | Time this heartbeat was published |

---

## Endpoints

No HTTP endpoints — background pub/sub service only.

---

## Redis Keys Used

| Key | Operation | Description |
|-----|-----------|-------------|
| `<CHAIN>_HEART_BEAT` | GET | Heartbeat status set by ingest (2 s TTL) |
| `<CHAIN>_LAST_DATA_TIME` | GET | Timestamp of last data received |
| `<CHAIN>_HEARTBEAT_CHANNEL` | PUBLISH | Output pub/sub channel for heartbeat events |

---

## Configuration (`config.yaml`)

```yaml
service:
  name: "chainmon"
  log_level: "info"

redis:
  addr: "localhost:6379"
  password: ""
  db: 0

chains:
  - name: "TM1"
    type: "TM"
    timeout_seconds: 10
  - name: "TM2"
    type: "TM"
    timeout_seconds: 10
  - name: "SMON1"
    type: "SCOS"
    timeout_seconds: 15
  - name: "ADC1"
    type: "ADC"
    timeout_seconds: 15
```

| Field | Description |
|-------|-------------|
| `chains[].name` | Chain identifier — must match the names used by ingest |
| `chains[].type` | Chain type for logging/metadata (`TM`, `SCOS`, `ADC`) |
| `chains[].timeout_seconds` | Informational; used for logging context |

---

## How to Run

```bash
# From the chainmon directory
go run ./cmd -config config.yaml
```

The service is also started automatically by the **launcher** when `chainmon.enabled: true` in `launcher/config.yaml`.
