# UMACS TC Emulator

## Overview

The UMACS TC Emulator is a lightweight mock server that faithfully reproduces the UMACS hardware TC REST interface (port 21003). It allows the full **umacs-tc → UMACS TC** integration to be tested on a development machine without physical spacecraft checkout hardware. When Redis integration is enabled, it also writes status transitions to `TC_FILES_STATUS` so that the `umacs-tc` service's polling loop resolves automatically.

---

## Architecture

```
  umacs-tc service  (port 21002)
           │  HTTP POST
           ▼
  ┌──────────────────────────────────────────────┐
  │           UMACS TC Emulator  (port 21003)      │
  │                                               │
  │  ┌─────────────────────────────────────────┐  │
  │  │  ProcedureStore  (in-memory state)       │  │
  │  │  • created procedures                    │  │
  │  │  • validated flag                        │  │
  │  │  • exe_status state machine              │  │
  │  └──────────────┬──────────────────────────┘  │
  │                 │  async goroutine per proc    │
  │  ┌──────────────▼──────────────────────────┐  │
  │  │  Execution simulation                    │  │
  │  │  queued → in-progress → success|failure  │  │
  │  └──────────────┬──────────────────────────┘  │
  └─────────────────┼────────────────────────────┘
                    │  HSET (optional)
                    ▼
                 Redis
           TC_FILES_STATUS
           (satisfies umacs-tc's polling loop)
```

---

## How It Works

### 1. In-memory procedure registry

`ProcedureStore` holds all procedure records in a thread-safe map. Each record tracks:

| Field | Description |
|-------|-------------|
| `Name` | Procedure name |
| `Content` | Full procedure text |
| `Validated` | Whether `validateProcedure` was called |
| `ExeStatus` | Current execution status string |
| `Mode` / `Priority` | Values from the last `loadProcedure` call |

### 2. Execution state machine

When `loadProcedure` is accepted, an async goroutine drives the state machine:

```
queued  ──(queued-delay)──►  in-progress  ──(inprogress-duration)──►  success | failure
```

Timing and outcome are configurable via CLI flags. If Redis is configured, each state transition is also written to `TC_FILES_STATUS[proc_name]` so that `umacs-tc`'s `triggerFileWaitForExecutionComplete` poll loop resolves without any changes to that service.

### 3. Validation guard

By default (`--no-validate-required` is **not** set), `loadProcedure` returns an error if `validateProcedure` was not called first for that procedure. This mirrors the real UMACS behaviour and catches integration bugs early.

---

## API Endpoints

All endpoints are `POST` (matching the real UMACS TC interface) and return:

```json
{ "ack": true|false, "error_msg": "...", "exe_status": "..." }
```

### `POST /createProcedure`

Stores a procedure's content in the emulator's in-memory registry.

**Request**
```json
{
  "proc_name": "test1.tst",
  "procedure": "001 SEND ACM_CMD VALUE=1\n002 end"
}
```

**Success response**
```json
{ "ack": true, "error_msg": "procedure created successfully", "exe_status": "" }
```

---

### `POST /validateProcedure`

Marks a previously created procedure as validated.

**Request**
```json
{
  "proc_name": "test1.tst",
  "proc_src":  "integration-test",
  "subsystem": "PAYLOAD"
}
```

> Both `proc_src` (spec) and `proc_source` (umacs-tc compat) are accepted.

**Success response**
```json
{ "ack": true, "error_msg": "procedure validated successfully", "exe_status": "" }
```

**Error** (procedure not found):
```json
{ "ack": false, "error_msg": "procedure 'test1.tst' not found — call createProcedure first", "exe_status": "" }
```

---

### `POST /loadProcedure`

Queues a validated procedure for simulated execution. Returns immediately; status transitions happen asynchronously.

**Request**
```json
{
  "action":       "execute",
  "proc_name":    "test1.tst",
  "proc_src":     "integration-test",
  "proc_mode":    1,
  "proc_priority": 1
}
```

`proc_mode` and `proc_priority` accept both JSON integers and quoted strings:

| `proc_mode` value | Interpretation |
|------------------|----------------|
| `1` or `"auto"` | Auto mode |
| `0` or `"manual"` | Manual mode |

| `proc_priority` value | Interpretation |
|----------------------|----------------|
| `0` or `"normal"` | Normal |
| `1` or `"high"` | High |
| `2` or `"critical"` | Critical |
| `3` or `"emergency"` | Emergency |

**Success response** (returned immediately)
```json
{ "ack": true, "error_msg": "procedure loaded successfully", "exe_status": "" }
```

---

### `POST /getExeStatus`

Returns the current execution status of a procedure.

**Request**
```json
{
  "action":    "exestatus",
  "proc_name": "test1.tst",
  "proc_src":  "integration-test",
  "proc_mode": 0,
  "proc_priority": 0
}
```

(`proc_mode` and `proc_priority` are ignored for status queries.)

**Response**
```json
{ "ack": true, "error_msg": "", "exe_status": "in-progress" }
```

Possible `exe_status` values:

| Value | Meaning |
|-------|---------|
| `queued` | Accepted, waiting to start |
| `in-progress` | Execution running |
| `success` | Completed successfully |
| `failure` | Execution failed |
| `aborted` | Procedure was aborted |
| `suspended` | Execution is suspended |
| `not-available` | Procedure not found |

---

### `GET /admin/procedures`

Debug endpoint — returns a JSON snapshot of all stored procedures and their current state. Not part of the UMACS spec.

**Response**
```json
{
  "test1.tst": {
    "name": "test1.tst",
    "content": "001 SEND ...",
    "validated": true,
    "exe_status": "success",
    "loaded_at": "2026-02-25T10:00:00Z",
    "mode": 1,
    "priority": 1
  }
}
```

---

### `GET /health`

Liveness probe. Returns `{"status":"ok"}`.

---

## Redis Integration (Optional)

When `--redis-addr` is provided, the emulator writes status transitions to `TC_FILES_STATUS` in Redis:

```
loadProcedure accepted  →  TC_FILES_STATUS[proc_name] = "in-progress"
simulation complete     →  TC_FILES_STATUS[proc_name] = "success" | "failure"
```

This satisfies the `triggerFileWaitForExecutionComplete` polling loop in `umacs-tc` without any changes to that service.

---

## How to Run

```bash
# Standalone (no Redis, works for REST-level testing)
go run ./cmd

# Full integration with umacs-tc (Redis mirrors TC_FILES_STATUS)
go run ./cmd --redis-addr localhost:6379

# Faster simulation timing (useful for CI)
go run ./cmd --queued-delay 100 --inprogress-duration 500

# Simulate 20% failure rate
go run ./cmd --success-rate 80

# Skip validation requirement
go run ./cmd --no-validate-required
```

### All flags

| Flag | Default | Description |
|------|---------|-------------|
| `--host` | `0.0.0.0` | Listen host |
| `--port` | `21003` | Listen port (matches real UMACS TC port) |
| `--queued-delay` | `500` | ms before queued → in-progress |
| `--inprogress-duration` | `4000` | ms for in-progress phase |
| `--success-rate` | `100` | % of procedures that succeed (0–100) |
| `--no-validate-required` | `false` | Accept loadProcedure without prior validate |
| `--redis-addr` | `""` | Redis address; empty = no Redis integration |
| `--redis-password` | `""` | Redis password |
| `--redis-db` | `0` | Redis database index |

### End-to-end test setup

1. Start Redis: `redis-server`
2. Start the emulator: `go run ./cmd --redis-addr localhost:6379`
3. In `umacs-tc/config.yaml` set `tc_ip: localhost`, `tc_port: 21003`
4. Start umacs-tc: `go run ./cmd`
5. Submit a procedure via umacs-tc's HTTP endpoints

The emulator logs all received requests in JSON format with `slog`.
