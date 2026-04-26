# Comparator Service

## Overview

The Comparator performs cross-chain consistency checking. It reads mnemonic values from two redundant telemetry chains, compares them, and records any confirmed mismatches. This enables detection of sensor faults, data corruption, or chain divergence during spacecraft checkout.

---

## Architecture

```
  MongoDB  (tm_mnemonics collection)
      │  load mnemonics where enable_comparison=true
      │
  ┌───┴──────────────────────────────────────────┐
  │              Comparator                       │
  │  ┌─────────────────────────────────────────┐  │
  │  │  MnemonicLoader                          │  │
  │  │  (reloads on TM_LIMIT_CHANGED pub/sub)  │  │
  │  └─────────────────────────────────────────┘  │
  │  ┌──────────────────────────────────────────┐  │
  │  │  Compare loop  (configurable interval)   │  │
  │  │  Mode A: Frame ID Sync                   │  │
  │  │  Mode B: Time Delta                      │  │
  │  └──────────────────────────────────────────┘  │
  └───────────────────────┬──────────────────────┘
                          │ HGETALL / HSET / HDEL
                          ▼
  Redis
  ┌──────────────────────────────────────────┐
  │  TM1_MAP, TM2_MAP          (read)        │
  │  TM_SOFTWARE_CFG_MAP       (read mode)   │
  │  TM_CHAIN_MISMATCHES_MAP   (write)       │
  └──────────────────────────────────────────┘
```

---

## How It Works

### 1. Mnemonic loading

On startup (and on every `TM_LIMIT_CHANGED` pub/sub event), the service loads all mnemonics from the `tm_mnemonics` MongoDB collection where `enable_comparison: true`. This allows live reconfiguration without a restart.

### 2. Comparison loop

A ticker fires at the configured interval. For each configured chain pair (e.g. `TM1` vs `TM2`):

1. Reads the full `<CHAIN1>_MAP` and `<CHAIN2>_MAP` hashes from Redis.
2. For each enabled mnemonic, compares the values from both chains.
3. If values match → clears any existing mismatch entry for that mnemonic.
4. If values differ → passes to the mode-specific confirmation handler.

### 3. Value comparison logic

| Mnemonic type | Match condition |
|---------------|----------------|
| `ANALOG` | `abs(value1 - value2) <= 2 × tolerance` |
| `BINARY` | Exact string equality |

### 4. Comparison modes

The mode is read from `TM_SOFTWARE_CFG_MAP[CHAIN_COMPARE_MODE]` on each cycle:

| Mode | Key | Description |
|------|-----|-------------|
| **A** — Frame ID Sync | `"A"` (default) | A mismatch is confirmed only when both chains report data for the same frame ID |
| **B** — Time Delta | `"B"` | A mismatch is confirmed when both values arrive within a configured time window |

### 5. Mismatch recording

A confirmed mismatch is written to `TM_CHAIN_MISMATCHES_MAP` as a JSON object:

```json
{
  "mnemonic":  "ACM05521",
  "chain1":    "TM1",
  "chain2":    "TM2",
  "value1":    "3.14",
  "value2":    "3.20",
  "type":      "ANALOG",
  "timestamp": "2026-02-25T10:00:00Z"
}
```

When a previously mismatched mnemonic returns to agreement, its entry is removed from `TM_CHAIN_MISMATCHES_MAP` automatically (`HDEL`).

---

## Endpoints

No HTTP endpoints — background service only. Query results via the **gateway** (`GET /chain-mismatches`).

---

## Redis Keys Used

| Key | Operation | Description |
|-----|-----------|-------------|
| `<CHAIN>_MAP` | HGETALL | Per-chain telemetry values |
| `TM_SOFTWARE_CFG_MAP` | HGET | Reads `CHAIN_COMPARE_MODE` and `CHAIN_COMPARE_DELAY_SECONDS` |
| `TM_CHAIN_MISMATCHES_MAP` | HSET / HDEL | Write confirmed mismatches; clear resolved ones |
| `TM_LIMIT_CHANGED` | SUBSCRIBE | Trigger mnemonic reload |

---

## Configuration (`config.yaml`)

```yaml
service:
  name: "comparator"
  log_level: "info"

redis:
  addr: "localhost:6379"
  password: ""
  db: 0

mongodb:
  uri: "mongodb://localhost:27017"
  database: "mainframe"

comparator:
  interval_seconds: 1
  pairs:
    - chain1: "TM1"
      chain2: "TM2"
```

| Field | Description |
|-------|-------------|
| `interval_seconds` | How often to run the comparison loop |
| `pairs` | List of chain pairs to compare |

---

## How to Run

```bash
# From the comparator directory
go run ./cmd -config config.yaml
```

The service is also started automatically by the **launcher** when `comparator.enabled: true` in `launcher/config.yaml`.
