# UMACS TC Service

## Overview

The UMACS TC service bridges Julia test procedures to the UMACS hardware TC (Telecommand) REST interface. It accepts procedure files over HTTP, drives the full create → validate → load → poll lifecycle against the UMACS server, and manages a Redis-backed **priority command queue** so that multiple concurrent Julia procedures can submit commands without overwhelming the inherently sequential UMACS API.

---

## Architecture

```
  Julia Procedures / Test Scripts
           │  HTTP POST
           ▼
  ┌──────────────────────────────────────────┐
  │              UMACS TC Service            │
  │                                          │
  │  HTTP endpoints (port 21002)              │
  │  ┌────────────────────────────────────┐  │
  │  │  trigger / transfer / split        │  │
  │  │  enqueue / clear                   │  │
  │  └───────────────┬────────────────────┘  │
  │                  │ enqueue               │
  │  ┌───────────────▼────────────────────┐  │
  │  │  QueueConsumer  (one goroutine)     │  │
  │  │  BZPopMin from TC_COMMAND_QUEUE     │  │
  │  │  → create → validate → load → poll │  │
  │  └───────────────┬────────────────────┘  │
  └──────────────────┼───────────────────────┘
                     │  HTTP POST
                     ▼
          UMACS TC Server  (port 21003)
          /createProcedure
          /validateProcedure
          /loadProcedure
          /getExeStatus (via TC_FILES_STATUS poll)

  Redis
  ┌──────────────────────────────────────────────┐
  │  ENV_VARIABLES_UMACS  (UMACS connection cfg)  │
  │  TC_FILES_STATUS      (per-file exe status)   │
  │  TC_COMMAND_QUEUE     (sorted set, priority)  │
  │  TC_COMMAND_COMPLETED:{request_id}  (pub/sub) │
  └──────────────────────────────────────────────┘
```

---

## How It Works

### 1. UMACS environment

On startup the service reads UMACS connection parameters from `ENV_VARIABLES_UMACS` in Redis (or falls back to `config.yaml` defaults and writes them to Redis). Parameters stored:

| Redis field | Description |
|------------|-------------|
| `UMACS_TC_IP` | IP address of the UMACS TC server |
| `UMACS_TC_PORT` | Port (default `21003`) |
| `UMACS_DATA_SERVER_IP` | Data server IP |
| `TC_API_REQ_SOURCE` | Proc source tag (e.g. `"pcc"`) |
| `TC_API_REQ_PRIORITY` | Default priority (e.g. `"normal"`) |
| `TC_API_REQ_EXECUTION_MODE` | Default mode (e.g. `"auto"`) |
| `TC_API_REQ_SUBSYSTEM` | Default subsystem (e.g. `"payload"`) |

### 2. Procedure lifecycle

Every endpoint that executes a procedure drives the same UMACS API sequence:

```
createProcedure  →  validateProcedure  →  loadProcedure  →  poll TC_FILES_STATUS
```

Status polling reads `TC_FILES_STATUS[proc_name]` from Redis (written by ingest or the UMACS system) until the status is no longer `queued` or `in-progress`.

### 3. Priority command queue

The queue consumer (`QueueConsumer`) blocks on `TC_COMMAND_QUEUE` (a Redis sorted set) using `BZPopMin`. Items are popped in priority order (lower score = higher priority) and dispatched **one at a time** — this serialises all SEND commands from all concurrent Julia procedures into a single sequential stream, as the UMACS TC API requires.

On completion, the result is published to `TC_COMMAND_COMPLETED:{request_id}` so the Julia caller can unblock.

### 4. Split file execution

Large procedures with long `wait HH:MM:SS:mmm` markers are split into sub-procedures at each wait point ≥ 1 minute. Each sub-procedure is:
1. Created and validated on the UMACS server.
2. Executed to completion (with the appropriate wait between parts).

Sub-procedure names follow the pattern `<base_name>_part1`, `<base_name>_part2`, etc., with command numbers renumbered from `001` within each part.

---

## API Endpoints

### `POST /umacs_tc_trigger_file_execution`

Load and execute a procedure that already exists on the UMACS server. Returns immediately after the load request is accepted.

**Request**
```json
{ "proc_name": "test1.tst" }
```

**Response**
```json
{ "ack": true, "error_msg": "procedure loaded successfully", "exe_status": "" }
```

---

### `POST /umacs_tc_trigger_file_execution_and_wait_for_completion`

Load an existing procedure and block until execution completes (or fails).

**Request**
```json
{ "proc_name": "test1.tst" }
```

**Response** — same shape as above, returned only when execution finishes.

---

### `POST /umacs_tc_transfer_file_and_trigger_execution`

Upload a new procedure to the UMACS server, validate it, and execute it. Returns after load (does not wait for completion).

**Request**
```json
{
  "proc_name": "test1.tst",
  "procedure": "001 SEND ACM_CMD VALUE=1\n002 end"
}
```

**Response**
```json
{ "ack": true, "error_msg": "procedure loaded successfully", "exe_status": "" }
```

---

### `POST /umacs_tc_transfer_file_trigger_execution_and_wait_for_completion`

Upload, validate, execute, and **wait for completion** before returning. Full synchronous flow.

**Request**
```json
{
  "proc_name": "test1.tst",
  "procedure": "001 SEND ACM_CMD VALUE=1\n002 end"
}
```

**Response** — returned only when execution is complete.

---

### `POST /umacs_tc_handle_split_file_execution`

Upload a large procedure, split it at `wait` markers ≥ 1 minute, execute each part sequentially with the corresponding inter-part delays.

**Request**
```json
{
  "proc_name": "longtest.tst",
  "procedure": "001 SEND CMD1\n002 wait 00:02:00:000\n003 SEND CMD2\n004 end"
}
```

**Response**
```json
{ "ack": true, "exe_status": "success" }
```

---

### `POST /umacs_tc_enqueue_command`

Enqueue a procedure command into the priority queue (`TC_COMMAND_QUEUE` sorted set). The `QueueConsumer` will dequeue and execute it when the queue is free.

**Request**
```json
{
  "request_id":   "req-abc-123",
  "procedure_id": "test1.tst",
  "procedure":    "001 SEND ACM_CMD VALUE=1\n002 end",
  "priority":     1,
  "timestamp":    "2026-02-25T10:00:00Z"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `request_id` | `string` | Unique request identifier; used for completion pub/sub channel |
| `procedure_id` | `string` | Procedure name |
| `procedure` | `string` | Full procedure text (alias: `command`) |
| `priority` | `int` | Queue priority score — lower = dequeued first |
| `timestamp` | `string` | ISO 8601 timestamp; auto-set if omitted |

**Response**
```json
{ "ack": true }
```

HTTP 201 Created.

Completion is signalled via `TC_COMMAND_COMPLETED:{request_id}` Redis pub/sub:
```json
{
  "request_id":   "req-abc-123",
  "procedure_id": "test1.tst",
  "status":       "completed",
  "timestamp":    "2026-02-25T10:00:05Z"
}
```

---

### `POST /umacs_tc_clear_command_queue`

Deletes the entire `TC_COMMAND_QUEUE` sorted set.

**Response**
```json
{ "ack": true }
```

---

## Redis Keys Used

| Key | Operation | Description |
|-----|-----------|-------------|
| `ENV_VARIABLES_UMACS` | HGET / HSET | UMACS connection configuration |
| `TC_FILES_STATUS` | HGET / HSET | Per-procedure execution status |
| `TC_COMMAND_QUEUE` | BZPopMin / ZADD / DEL | Priority queue of pending TC commands |
| `TC_COMMAND_COMPLETED:{req_id}` | PUBLISH | Completion notification per request |

---

## Configuration (`config.yaml`)

```yaml
server:
  host: "0.0.0.0"
  port: 21002

redis:
  addr: "localhost:6379"
  password: ""
  db: 0

umacs:
  tc_ip: "172.20.xx.xx"
  tc_port: "21003"
  data_server_ip: "172.20.xx.xx"
  api_req_source: "pcc"
  api_req_priority: "normal"
  api_req_execution_mode: "auto"
  api_req_subsystem: "payload"
```

Config values are read into Redis on startup if not already present, allowing runtime override via Redis without a restart.

---

## How to Run

```bash
# From the umacs-tc directory
go run ./cmd

# Custom config path
go run ./cmd  # reads config.yaml in working dir
```

For testing without real UMACS hardware, point `tc_ip`/`tc_port` to the **umacs-tc-emulator** running on `localhost:21003`.

The service is also started automatically by the **launcher** when `umacs_tc.enabled: true` in `launcher/config.yaml`.
