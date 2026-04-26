# Ingest Service

## Overview

The Ingest service is the live telemetry entry point. It connects to every configured telemetry chain over WebSocket, parses incoming packets, and writes decoded parameter values into Redis. All downstream services (limiter, comparator, storage, gateway) consume the Redis maps that Ingest populates.

---

## Architecture

```
  SCOS / TM WebSocket Servers
  (TM1, TM2, SMON1, ADC1 …)
           │  ws://host:port/ws
           │
  ┌────────┴─────────────────────────────┐
  │   Ingest  (one goroutine per chain)  │
  │  ┌───────────────────────────────┐   │
  │  │  WSSubscriber                 │   │
  │  │  auto-reconnect + backoff     │   │
  │  └───────────────────────────────┘   │
  └────────┬─────────────────────────────┘
           │  Redis Pipeline  (HSET, 4 keys per message)
           ▼
  ┌──────────────────────────────────────────────────┐
  │  TM1_MAP     TM1_PKT     TM_MAP (unified)        │
  │  TM2_MAP     TM2_PKT     TM1_HEART_BEAT (TTL 2s) │
  │  SMON1_MAP   SMON1_PKT   …                        │
  └──────────────────────────────────────────────────┘
```

---

## How It Works

### 1. Per-chain goroutines

One `ChainSubscriber` goroutine is launched per chain entry in `config.yaml`. A failure in one chain's connection does not affect any other chain.

### 2. WebSocket lifecycle

The `WSSubscriber` maintains a persistent WebSocket connection with automatic reconnect and exponential back-off. Three lifecycle hooks drive Redis state:

| Event | Redis action |
|-------|-------------|
| `OnConnect` | Sets `<CHAIN>_HEART_BEAT = CONNECTED` (no TTL) |
| `OnDisconnect` | Sets `<CHAIN>_HEART_BEAT = CONNECTION_FAILED`; clears `<CHAIN>_MAP` and `<CHAIN>_PKT` |
| `OnMessage` | Parses packet; writes per-chain and unified maps; sets heartbeat to OK with 2 s TTL |

### 3. Packet parsing

Two chain types are supported, selected via the `type` field in config:

| Chain Type | Parser | Used for |
|-----------|--------|---------|
| `TM` | `ParseTMPacket` | Telemetry chains (TM1, TM2, …) |
| `SCOS` | `ParseSCOSPacket` | SCOS/SMON and ADC chains |

A **data break** (detected by `"break"` in the `err_desc` field) resets the heartbeat to `DATA_BREAK` and clears the chain's Redis data, ready for fresh telemetry.

### 4. Redis writes per message (single pipeline)

| Key | Type | Written value |
|-----|------|---------------|
| `<CHAIN>_MAP` | Hash | `param → decoded value` |
| `<CHAIN>_PKT` | Hash | `param → raw JSON packet` |
| `TM_MAP` | Hash | `param → decoded value` (with suffix rules) |
| `<CHAIN>_HEART_BEAT` | String (TTL 2 s) | `OK` |

### 5. Unified TM_MAP suffix rules

`TM_MAP` is the single source of truth for all downstream consumers. Writing rules by chain:

| Chain | TM_MAP field name |
|-------|-------------------|
| `TM1`, `TM2`, `TM3`, `TM4` | `param` — no suffix; last writer wins (failover) |
| `SMON1` | `param` — primary SCOS source |
| `SMON2`, `SMON3`, … | `param_SMON2`, `param_SMON3`, … |
| `ADC1` | `param` — primary ADC source |
| `ADC2`, `ADC3`, … | `param_ADC2`, `param_ADC3`, … |

---

## Endpoints

No HTTP endpoints — background service only.

---

## Redis Keys Written

| Key | Type | Description |
|-----|------|-------------|
| `<CHAIN>_MAP` | Hash | Per-chain mnemonic → value |
| `<CHAIN>_PKT` | Hash | Per-chain mnemonic → raw packet JSON |
| `TM_MAP` | Hash | Unified mnemonic → value (all chains merged) |
| `<CHAIN>_HEART_BEAT` | String | Heartbeat status: `OK` (TTL 2 s), `CONNECTED`, `CONNECTION_FAILED`, `DATA_BREAK` |

---

## MQTT Publishing

When MQTT is enabled, Ingest publishes TM and status updates with the configured topic prefix.

| Topic | Type | Delivery | Payload |
|-------|------|----------|---------|
| `<prefix>/tm_map` | Delta (changed mnemonics only) | QoS from config, retain `false` | JSON array of single-key objects, e.g. `[ {"TEMP":"123"}, {"PRESS":"45"} ]` |
| `<prefix>/tm_map/full` | Full TM_MAP snapshot | QoS `1`, retain `true` | Same as `<prefix>/tm_map` (JSON array of single-key objects) |
| `<prefix>/heartbeat` | Publisher health | QoS `0`, retain `false` | `OK` or `NO_DATA` |
| `<prefix>/limit-failures` | Status | QoS `0`, retain `false` | `{"timestamp":"...","data":{...}}` |
| `<prefix>/chain-mismatches` | Status | QoS `0`, retain `false` | `{"timestamp":"...","data":{...}}` |
| `<prefix>/chain-status` | Status | QoS `0`, retain `false` | `{"timestamp":"...","chains":[{"chain":"TM1","status":"active"}]}` |

---

## Configuration (`config.yaml`)

```yaml
service:
  name: "ingest"
  log_level: "info"

redis:
  addr: "localhost:6379"
  password: ""
  db: 0

websocket:
  reconnect_delay_ms: 1000
  max_reconnect_delay_ms: 30000

chains:
  - name: "TM1"
    type: "TM"
    host: "192.168.1.10"
    port: 9001
  - name: "TM2"
    type: "TM"
    host: "192.168.1.11"
    port: 9001
  - name: "SMON1"
    type: "SCOS"
    host: "192.168.1.20"
    port: 9002
  - name: "ADC1"
    type: "SCOS"
    host: "192.168.1.30"
    port: 9003
```

| Field | Description |
|-------|-------------|
| `chains[].name` | Chain identifier used as Redis key prefix (e.g. `TM1` → `TM1_MAP`) |
| `chains[].type` | `TM` or `SCOS` — selects the packet parser |
| `chains[].host` | WebSocket server hostname or IP |
| `chains[].port` | WebSocket server port |

---

## How to Run

```bash
# From the ingest directory
go run ./cmd -config config.yaml

# Custom config path
go run ./cmd -config /etc/mainframe/ingest.yaml
```

The service is also started automatically by the **launcher** when `ingest.enabled: true` in `launcher/config.yaml`.
