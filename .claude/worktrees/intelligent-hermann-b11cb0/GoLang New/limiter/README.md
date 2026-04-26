# Limiter Service

## Overview

The Limiter performs continuous limit checking against the live telemetry stream. It reads mnemonic values from `TM_MAP`, checks each value against its configured limits, and records violations in `TM_LIMIT_FAILURES_MAP`. It also handles **EXPECTED suppression** — when a Julia procedure pre-declares that a mnemonic will temporarily violate its limits due to a commanded action, the limiter skips those mnemonics for the declared suppression window.

---

## Architecture

```
  MongoDB  (tm_mnemonics collection)
      │  load mnemonics where enable_limit=true
      │
  ┌───┴────────────────────────────────────────────┐
  │                  Limiter                        │
  │  ┌───────────────────────────────────────────┐  │
  │  │  MnemonicLoader                            │  │
  │  │  (reloads on TM_LIMIT_CHANGED pub/sub)    │  │
  │  └───────────────────────────────────────────┘  │
  │  ┌───────────────────────────────────────────┐  │
  │  │  Check loop  (configurable interval)       │  │
  │  │  AnalogCheck + DigitalCheck + Suppression  │  │
  │  └───────────────────────────────────────────┘  │
  └───────────────────────┬────────────────────────┘
                          │
                          ▼
  Redis
  ┌──────────────────────────────────────────────────────┐
  │  TM_MAP                         (read all values)    │
  │  TM_EXPECTED_DIGITAL_STATES_MAP (read expected vals) │
  │  TM_LIMIT_SUPPRESSION_MAP       (read suppressions)  │
  │  TM_LIMIT_FAILURES_MAP          (write violations)   │
  └──────────────────────────────────────────────────────┘
```

---

## How It Works

### 1. Mnemonic loading

On startup (and on every `TM_LIMIT_CHANGED` pub/sub event), all mnemonics from MongoDB with `enable_limit: true` are loaded. This enables live limit reconfiguration without a restart.

### 2. Check loop

A ticker fires at the configured interval (typically 1 s). Each cycle:

1. Reads all values from `TM_MAP` (`HGETALL`).
2. Reads expected digital states from `TM_EXPECTED_DIGITAL_STATES_MAP`.
3. Loads active **EXPECTED suppressions** from `TM_LIMIT_SUPPRESSION_MAP`.
4. For each mnemonic in the loaded list:
   - Skips if a suppression entry is active for this mnemonic.
   - Runs the appropriate check (analog or digital).
   - If violation → writes to `TM_LIMIT_FAILURES_MAP`.
   - If resolved → clears the entry from `TM_LIMIT_FAILURES_MAP`.
5. Verifies expectations for just-expired suppressions.
6. Cleans up expired suppression entries from Redis (atomic Lua script).

### 3. Analog limit check

```
violation if:  value < range[0]  OR  value > range[1]
```

If the mnemonic has an active suppression, the check is skipped entirely.

### 4. Digital expected-state check

A binary mnemonic has a violation when its current value does not match the value in `TM_EXPECTED_DIGITAL_STATES_MAP[mnemonic]`. If the mnemonic is suppressed (pre-declared via EXPECTED), the check is skipped.

### 5. EXPECTED suppression mechanism

Before issuing a `SEND` command, Julia writes a `SuppressionEntry` to `TM_LIMIT_SUPPRESSION_MAP` for each mnemonic expected to transiently change state:

```json
{
  "request_id":    "req-abc-123",
  "procedure_id":  "PROC_NAME",
  "operator":      ">=",
  "expected_value":"3.0",
  "mnemonic_type": "ANALOG",
  "expires_at_unix": 1740000060
}
```

The limiter reads these on every cycle. An active suppression means the mnemonic is skipped. When the TTL expires, the limiter verifies that the expected outcome was actually achieved; if not, a violation is recorded.

### 6. LimitViolation object

Written to `TM_LIMIT_FAILURES_MAP` as JSON:

```json
{
  "mnemonic":  "ACM05521",
  "type":      "ANALOG",
  "value":     "150.5",
  "min":       "0",
  "max":       "100",
  "timestamp": "2026-02-25T10:00:00Z"
}
```

For digital violations: `"type": "DIGITAL"`, `"expected": "expected: PRESENT"` (no `min`/`max`).

---

## Endpoints

No HTTP endpoints — background service only. Query results via the **gateway** (`GET /limit-failures`).

---

## Redis Keys Used

| Key | Operation | Description |
|-----|-----------|-------------|
| `TM_MAP` | HGETALL | All current telemetry values |
| `TM_EXPECTED_DIGITAL_STATES_MAP` | HGETALL | Expected state per binary mnemonic |
| `TM_LIMIT_SUPPRESSION_MAP` | HGETALL | EXPECTED pre-declarations (with TTL) |
| `TM_LIMIT_FAILURES_MAP` | HSET / HDEL | Active violations; cleared when resolved |
| `TM_LIMIT_CHANGED` | SUBSCRIBE | Trigger mnemonic reload |
| `TM_SOFTWARE_CFG_MAP` | HGET | Reads `MASTER_FRAME_WAIT_COUNT`, suppression window config |

---

## Configuration (`config.yaml`)

```yaml
service:
  name: "limiter"
  log_level: "info"

redis:
  addr: "localhost:6379"
  password: ""
  db: 0

mongodb:
  uri: "mongodb://localhost:27017"
  database: "mainframe"

limiter:
  interval_ms: 1000
```

| Field | Description |
|-------|-------------|
| `interval_ms` | Check loop interval in milliseconds |

---

## How to Run

```bash
# From the limiter directory
go run ./cmd -config config.yaml
```

The service is also started automatically by the **launcher** when `limiter.enabled: true` in `launcher/config.yaml`.
