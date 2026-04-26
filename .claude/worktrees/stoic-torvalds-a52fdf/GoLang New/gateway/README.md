# Gateway Service

## Overview

The Gateway is the single HTTP entry-point for all external clients (frontend dashboard, test scripts, Julia procedures). It translates REST requests into Redis reads/writes and returns results as JSON. It does not process or transform telemetry — that is left to the specialised background services (ingest, limiter, comparator, simulator).

---

## Architecture

```
  Frontend / Clients
          │
          ▼  HTTP REST  (Chi router + CORS)
  ┌───────────────────┐
  │      Gateway      │
  └───────────────────┘
          │  HMGET / HGETALL / HSET
          ▼
       Redis
  ┌─────────────────────────────────────────────┐
  │  TM_MAP            TM_CHAIN_MISMATCHES_MAP  │
  │  TM1_MAP …         TM_LIMIT_FAILURES_MAP    │
  │  UDTM_MAP          TM_SIMULATOR_CFG_MAP     │
  │  DTM_MAP           TM1_HEART_BEAT …         │
  └─────────────────────────────────────────────┘
```

---

## How It Works

1. **Startup** — loads `config.yaml`, connects to Redis with exponential back-off retry.
2. **Router** — mounts a `chi.Router` with global middleware: request logging, panic recovery, and CORS (all origins allowed).
3. **Handlers** — each handler is a thin Redis façade:
   - Reads use `HMGET` (batch mnemonic reads) or `HGETALL` (full map reads).
   - Writes use `HSET` (batch mnemonic updates).
4. **Graceful shutdown** — listens for `SIGINT`/`SIGTERM`; drains in-flight requests within 10 s.

---

## API Endpoints

Base URL: `http://localhost:21000`

### Quick Reference

| # | Method | Path | Source | Description |
|---|--------|------|--------|-------------|
| 1 | `POST` | `/get-telemetry` | Redis | Selected params by mnemonic list |
| 2 | `GET` | `/maps/{name}` | Redis | Full map as `[{param,value}]` array |
| 3 | `GET` | `/chain-status` | Redis | Chain heartbeat status |
| 4 | `GET` | `/chain-mismatches` | Redis | TM1 vs TM2 mismatches |
| 5 | `GET` | `/limit-failures` | Redis | Out-of-limit parameters |
| 6 | `GET` | `/simulator-status` | Redis | Simulator config |
| 7 | `PUT` | `/udtm/values` | Redis | Inject UD-TM values |
| 8 | `PUT` | `/dtm/values` | Redis | Inject DTM values |
| 9 | `GET` | `/mnemonics/tm` | MongoDB | TM catalog (`?subsystem=` optional) |
| 10 | `GET` | `/mnemonics/tm/{subsystem}` | MongoDB | TM catalog filtered by subsystem |
| 11 | `GET` | `/mnemonics/tc` | MongoDB | TC command catalog |
| 12 | `GET` | `/mnemonics/sco` | MongoDB | SCO command catalog |
| 13 | `GET` | `/mnemonics/all` | MongoDB | Combined Monaco autocomplete catalog |
| 14 | `GET` | `/telemetry/subsystems` | MongoDB | Distinct subsystem names |
| 15 | `GET` | `/tm/mnemonics` | Redis | Live TM_MAP key list |
| 16 | `GET` | `/ud-tm` | MongoDB | Current UD-TM table |
| 17 | `POST` | `/ud-tm` | MongoDB | Save UD-TM + version snapshot |
| 18 | `GET` | `/ud-tm/versions` | MongoDB | Version list (summaries) |
| 19 | `GET` | `/ud-tm/versions/{version}` | MongoDB | Full version snapshot |

---

### `POST /get-telemetry`

Returns current values for a list of mnemonics using a single Redis `HMGET` call.

**Request**
```json
{
  "mnemonics": ["ACM05521", "ACM05522"],
  "source": "TM1"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `mnemonics` | `[]string` | Yes | Mnemonic IDs to fetch |
| `source` | `string` | No | If set (e.g. `"TM1"`), reads from `TM1_MAP`; omit to read from unified `TM_MAP` |

**Response**
```json
{
  "results": {
    "ACM05521": "3.142",
    "ACM05522": null
  }
}
```

Values are `null` when the mnemonic is absent from the map.

---

### `GET /maps/{name}`

Returns the **full contents** of a Redis hash map as an array of `{param, value}` objects. Useful for bulk reads, debugging, or populating external tools.

| `name` | Redis key | Contents |
|--------|-----------|----------|
| `tm` | `TM_MAP` | Unified map — all chains merged (~5000 params) |
| `tm1` | `TM1_MAP` | TM chain 1 only |
| `tm2` | `TM2_MAP` | TM chain 2 only |
| `smon1` | `SMON1_MAP` | SMON chain 1 |
| `smon2` | `SMON2_MAP` | SMON chain 2 |
| `adc1` | `ADC1_MAP` | ADC chain 1 |
| `adc2` | `ADC2_MAP` | ADC chain 2 |
| `udtm` | `UDTM_MAP` | User-defined TM |
| `dtm` | `DTM_MAP` | Derived TM |

**Response**
```json
[
  {"param": "acm05521", "value": "3.142"},
  {"param": "bus_voltage", "value": "28.1"}
]
```

**Examples**
```bash
curl http://localhost:21000/maps/tm        # full unified map
curl http://localhost:21000/maps/tm1       # TM1 chain only
curl http://localhost:21000/maps/smon1     # SMON1 chain
curl http://localhost:21000/maps/udtm      # user-defined TM
```

**Error** (unknown name) → HTTP 400:
```json
{"error": "unknown map: xyz. Valid: tm, tm1, tm2, smon1, smon2, adc1, adc2, udtm, dtm"}
```

---

### `GET /chain-status`

Returns active/inactive status for all configured chains. A chain is **active** if its `<CHAIN>_HEART_BEAT` Redis key exists (keys carry a 2-second TTL, set by the ingest service on every received packet).

**Response**
```json
{
  "chains": [
    { "chain": "TM1",   "status": "active" },
    { "chain": "TM2",   "status": "inactive" },
    { "chain": "SMON1", "status": "active" },
    { "chain": "ADC1",  "status": "inactive" }
  ]
}
```

Monitored chains: `TM1`, `TM2`, `TM3`, `TM4`, `SMON1`, `ADC1`.

---

### `GET /chain-mismatches`

Returns all cross-chain value mismatches from `TM_CHAIN_MISMATCHES_MAP`. Populated by the **comparator** service. Returns `{}` when no mismatches are active.

**Response**
```json
{
  "ACM05521": "{\"mnemonic\":\"ACM05521\",\"chain1\":\"TM1\",\"chain2\":\"TM2\",\"value1\":\"3.14\",\"value2\":\"3.20\",\"type\":\"ANALOG\",\"timestamp\":\"2026-02-25T10:00:00Z\"}"
}
```

Each value is a JSON-encoded `Mismatch` object (see comparator README).

---

### `GET /limit-failures`

Returns all currently active limit violations from `TM_LIMIT_FAILURES_MAP`. Populated by the **limiter** service. Returns `{}` when no violations are active.

**Response**
```json
{
  "ACM05521": "{\"mnemonic\":\"ACM05521\",\"type\":\"ANALOG\",\"value\":\"150.5\",\"min\":\"0\",\"max\":\"100\",\"timestamp\":\"2026-02-25T10:00:00Z\"}"
}
```

Each value is a JSON-encoded `LimitViolation` object (see limiter README).

---

### `GET /simulator-status`

Returns the current simulator configuration hash from `TM_SIMULATOR_CFG_MAP`.

**Response**
```json
{
  "ENABLE": "1",
  "MODE": "FIXED",
  "SAMPLE_DELAY": "1000"
}
```

---

### `PUT /udtm/values`

Writes user-defined telemetry (UDTM) values to both `UDTM_MAP` and the unified `TM_MAP` simultaneously.

**Request**
```json
[
  { "mnemonic": "ACM05521", "value": "42.0" },
  { "mnemonic": "ACM05522", "value": "PRESENT" }
]
```

**Response**
```json
{ "updated": 2 }
```

---

### `PUT /dtm/values`

Writes derived telemetry (DTM) values to both `DTM_MAP` and `TM_MAP`. Same interface as UDTM.

**Request**
```json
[
  { "mnemonic": "ACM05599", "value": "1.23" }
]
```

**Response**
```json
{ "updated": 1 }
```

---

### `GET /mnemonics/tm`

Returns TM mnemonic catalog from MongoDB `tm_mnemonics` collection.

**Query parameters:** `?subsystem=PAYLOAD` (optional — filters by subsystem)

**Response**
```json
[
  {
    "subsystem": "PAYLOAD",
    "type": "ANALOG",
    "unit": "degC",
    "cdbMnemonic": "ACM05521"
  }
]
```

```bash
curl http://localhost:21000/mnemonics/tm
curl http://localhost:21000/mnemonics/tm?subsystem=PAYLOAD
```

---

### `GET /mnemonics/tm/{subsystem}`

Same as above but subsystem is a path parameter instead of query string.

```bash
curl http://localhost:21000/mnemonics/tm/PAYLOAD
```

---

### `GET /mnemonics/tc`

Returns TC command catalog from MongoDB `tc_mnemonics`. Adds `full_ref: "TC.<command>"` to each entry.

**Response**
```json
[
  {
    "command": "PAYLOAD_ON",
    "full_ref": "TC.PAYLOAD_ON",
    "description": "Power on payload",
    "parameters": [],
    "subsystem": "PAYLOAD",
    "category": "POWER"
  }
]
```

---

### `GET /mnemonics/sco`

Returns SCO command catalog from MongoDB `sco_commands`. Adds `full_ref: "SCO.<command>"` to each entry.

**Response**
```json
[
  {
    "command": "SCO_RESET",
    "full_ref": "SCO.SCO_RESET",
    "description": "Reset SCO",
    "subsystem": "SCOS",
    "category": "CONTROL"
  }
]
```

---

### `GET /mnemonics/all`

Combined catalog for Monaco editor autocomplete — fetches all collections in parallel.

**Response**
```json
{
  "tm_mnemonics":    [ ... ],
  "tc_mnemonics":    [ ... ],
  "sco_commands":    [ ... ],
  "ud_tm_mnemonics": [ ... ]
}
```

`ud_tm_mnemonics` are TM mnemonics with `subsystem="UDTM"` from `tm_mnemonics`.

---

### `GET /telemetry/subsystems`

Returns distinct subsystem names from `tm_mnemonics`, sorted alphabetically.

**Response**
```json
{
  "subsystems": ["ADC", "PAYLOAD", "POWER", "SMON", "TTC"]
}
```

---

### `GET /tm/mnemonics`

Returns the live key list from Redis `TM_MAP` — parameters currently being received by the ingest service.

**Response**
```json
{
  "mnemonics": ["acm05521", "bus_voltage", "payload_temp", ...]
}
```

---

### `GET /ud-tm`

Retrieves the current UD-TM table for a project from MongoDB `user_telemetry`.

**Query parameters:** `?project=default` (optional, defaults to `"default"`)

**Response**
```json
{
  "rows": [
    {
      "row_index": 0,
      "mnemonic": "ud_temp",
      "description": "User-defined temperature",
      "type": "ANALOG",
      "unit": "degC",
      "value": "",
      "last_updated": "2026-02-25T10:00:00Z"
    }
  ],
  "latest_version": 3
}
```

Returns `{"rows": [], "latest_version": 0}` if no document exists for the project yet.

---

### `POST /ud-tm`

Saves UD-TM rows. Performs three operations:
1. Upserts document in `user_telemetry` (bumps version)
2. Inserts snapshot in `user_telemetry_versions`
3. Syncs non-empty rows → `tm_mnemonics` with `subsystem="UDTM"`, then publishes `MDB_TM_MNEMONICS_UPDATED` so limiter/comparator/storage reload

**Request**
```json
{
  "rows": [
    {
      "row_index": 0,
      "mnemonic": "ud_temp",
      "description": "User-defined temperature",
      "type": "ANALOG",
      "unit": "degC"
    }
  ],
  "project": "default",
  "created_by": "operator1",
  "change_message": "Added temperature parameter"
}
```

**Response**
```json
{
  "success": true,
  "version": 4,
  "synced_mnemonics": 1,
  "message": "saved"
}
```

---

### `GET /ud-tm/versions`

Lists all saved UD-TM versions (summary only — no rows included), sorted descending.

**Query parameters:** `?project=default`

**Response**
```json
{
  "versions": [
    {
      "version": 4,
      "created_by": "operator1",
      "created_at": "2026-02-25T10:00:00Z",
      "change_message": "Added temperature parameter",
      "row_count": 1
    }
  ]
}
```

---

### `GET /ud-tm/versions/{version}`

Returns the full snapshot of a specific version including all rows.

**Query parameters:** `?project=default`

**Response**
```json
{
  "version": {
    "project": "default",
    "version": 4,
    "rows": [ ... ],
    "created_by": "operator1",
    "created_at": "2026-02-25T10:00:00Z",
    "change_message": "Added temperature parameter"
  }
}
```

Returns HTTP 404 if version not found.

---

## Redis Keys Used

| Key | Operation | Endpoint(s) |
|-----|-----------|-------------|
| `TM_MAP` | HMGET / HGETALL | `/get-telemetry`, `/maps/tm`, `/tm/mnemonics` |
| `TM1_MAP` … `ADC2_MAP` | HGETALL | `/maps/tm1` … `/maps/adc2` |
| `<CHAIN>_HEART_BEAT` | GET (TTL check) | `/chain-status` |
| `TM_CHAIN_MISMATCHES_MAP` | HGETALL | `/chain-mismatches` |
| `TM_LIMIT_FAILURES_MAP` | HGETALL | `/limit-failures` |
| `TM_SIMULATOR_CFG_MAP` | HGETALL | `/simulator-status` |
| `UDTM_MAP` | HSET / HGETALL | `/udtm/values`, `/maps/udtm` |
| `DTM_MAP` | HSET / HGETALL | `/dtm/values`, `/maps/dtm` |

## MongoDB Collections Used

| Collection | Operation | Endpoint(s) |
|------------|-----------|-------------|
| `tm_mnemonics` | find | `/mnemonics/tm`, `/mnemonics/tm/{subsystem}`, `/mnemonics/all` |
| `tc_mnemonics` | find | `/mnemonics/tc`, `/mnemonics/all` |
| `sco_commands` | find | `/mnemonics/sco`, `/mnemonics/all` |
| `user_telemetry` | findOne / upsert | `/ud-tm` (GET + POST) |
| `user_telemetry_versions` | find / insertOne | `/ud-tm/versions`, `/ud-tm/versions/{version}`, `/ud-tm` (POST) |

---

## Configuration (`config.yaml`)

```yaml
service:
  name: "gateway"
  log_level: "info"

redis:
  addr: "localhost:6379"
  password: ""
  db: 0

http:
  port: 8080
```

---

## How to Run

```bash
# From the gateway directory
go run ./cmd config.yaml

# Build then run
go build -o gateway ./cmd
./gateway config.yaml
```

The service is also started automatically by the **launcher** when `gateway.enabled: true` in `launcher/config.yaml`.
