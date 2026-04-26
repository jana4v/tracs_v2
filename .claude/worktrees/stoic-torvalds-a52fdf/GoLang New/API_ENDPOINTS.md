# API Endpoints By Service

This document lists HTTP API endpoints discovered from service route registration code.

## Kong Single-Port Access

Kong proxy port: `8000`

Service prefixes configured in `kong/kong.yml`:
- Gateway via Kong: `/gateway/*` -> upstream `http://host.docker.internal:21000/*`
- Simulator via Kong: `/simulator/*` -> upstream `http://host.docker.internal:21001/*`
- UMACS-TC via Kong: `/umacs-tc/*` -> upstream `http://host.docker.internal:21002/*`
- UMACS-TC Emulator via Kong: `/umacs-emulator/*` -> upstream `http://host.docker.internal:21003/*`
- EMQX MQTT WebSocket via Kong: `/mqtt*` -> upstream `http://host.docker.internal:8083/*`

Example URLs through Kong:
- `http://localhost:8000/gateway/api/go/v1/chain-status`
- `http://localhost:8000/simulator/api/go/v1/simulator-status`
- `http://localhost:8000/umacs-tc/api/go/v1/umacs_tc_clear_command_queue`
- `http://localhost:8000/umacs-emulator/api/go/v1/health`
- `ws://localhost:8000/mqtt`
- `ws://localhost:8000/mqtt/`

## HTTP Services And Ports

| Service | Port | Host | Port source |
|---|---:|---|---|
| gateway | 21000 | all interfaces (`:21000`) | `gateway/config.yaml` (`http.port`) |
| simulator | 21001 | all interfaces (`:21001`) | `simulator/config.yaml` (`http.port`), default fallback in code |
| umacs-tc | 21002 | configurable (`server.host`) | `umacs-tc/config.yaml` (`server.port`) |
| umacs-tc-emulator | 21003 | configurable (`--host`, default `0.0.0.0`) | CLI default `--port 21003` (config file is reference only) |

## gateway (`:21000`)

### Core telemetry and control
- `POST /api/go/v1/get-telemetry`
- `GET /api/go/v1/chain-status`
- `GET /api/go/v1/chain-mismatches`
- `GET /api/go/v1/limit-failures`
- `GET /api/go/v1/simulator-status`
- `PUT /api/go/v1/udtm/values`
- `PUT /api/go/v1/dtm/values`

### Mnemonics and catalog
- `GET /api/go/v1/mnemonics/tm`
- `GET /api/go/v1/mnemonics/tm/{subsystem}`
- `GET /api/go/v1/get/mnemonics/tm`
- `GET /api/go/v1/get/mnemonics/tm/{subsystem}`
- `GET /api/go/v1/get/mnemonics/tm/{subsystem}/{mnemonic}/range`
- `GET /api/go/v1/mnemonics/tc`
- `GET /api/go/v1/mnemonics/sco`
- `GET /api/go/v1/mnemonics/all`
- `GET /api/go/v1/telemetry/subsystems`
- `GET /api/go/v1/tm/mnemonics`

### Upload APIs
- `POST /api/go/v1/telemetry/upload`
- `POST /api/go/v1/telecommand/upload`

### Telemetry configuration update APIs
- `PUT /api/go/v1/telemetry/limits`
- `PUT /api/go/v1/telemetry/ignore-limit-check`
- `PUT /api/go/v1/telemetry/ignore-change-detection`
- `PUT /api/go/v1/telemetry/ignore-chain-comparision`
- `PUT /api/go/v1/telemetry/available-chains`

### UD-TM CRUD
- `GET /api/go/v1/ud-tm`
- `POST /api/go/v1/ud-tm`
- `GET /api/go/v1/ud-tm/versions`
- `GET /api/go/v1/ud-tm/versions/{version}`

### DTM procedure CRUD
- `GET /api/go/v1/dtm/procedures`
- `POST /api/go/v1/dtm/procedures`

### Redis map read
- `GET /api/go/v1/maps/{name}`

### Spasdacs diagrams
- `GET /api/go/v1/diagrams`
- `GET /api/go/v1/diagrams/{id}`
- `POST /api/go/v1/diagrams`
- `DELETE /api/go/v1/diagrams/{id}`

## simulator (`:21001`)

### Simulator API
- `PUT /api/go/v1/simulator/values`
- `GET /api/go/v1/simulator/values`
- `GET /api/go/v1/simulator/subsystems`
- `POST /api/go/v1/simulator/reset`
- `GET /api/go/v1/simulator-status`
- `GET /api/go/v1/simulator/mnemonics`
- `GET /api/go/v1/simulator/mnemonic/range`
- `GET /api/go/v1/simulator/mode`
- `PUT /api/go/v1/simulator/mode`
- `POST /api/go/v1/simulator/start`
- `POST /api/go/v1/simulator/stop`

### Explicit CORS preflight routes
- `OPTIONS /api/go/v1/simulator/values`
- `OPTIONS /api/go/v1/simulator/subsystems`
- `OPTIONS /api/go/v1/simulator/reset`
- `OPTIONS /api/go/v1/simulator-status`
- `OPTIONS /api/go/v1/simulator/mnemonics`
- `OPTIONS /api/go/v1/simulator/mnemonic/range`
- `OPTIONS /api/go/v1/simulator/mode`
- `OPTIONS /api/go/v1/simulator/start`
- `OPTIONS /api/go/v1/simulator/stop`

## umacs-tc (`0.0.0.0:21002` by current config)

### UMACS bridge API
- `POST /api/go/v1/umacs_tc_trigger_file_execution`
- `POST /api/go/v1/umacs_tc_trigger_file_execution_and_wait_for_completion`
- `POST /api/go/v1/umacs_tc_transfer_file_and_trigger_execution`
- `POST /api/go/v1/umacs_tc_transfer_file_trigger_execution_and_wait_for_completion`
- `POST /api/go/v1/umacs_tc_handle_split_file_execution`
- `POST /api/go/v1/umacs_tc_enqueue_command`
- `POST /api/go/v1/umacs_tc_clear_command_queue`

### Explicit CORS preflight routes
- `OPTIONS /api/go/v1/umacs_tc_trigger_file_execution`
- `OPTIONS /api/go/v1/umacs_tc_trigger_file_execution_and_wait_for_completion`
- `OPTIONS /api/go/v1/umacs_tc_transfer_file_and_trigger_execution`
- `OPTIONS /api/go/v1/umacs_tc_transfer_file_trigger_execution_and_wait_for_completion`
- `OPTIONS /api/go/v1/umacs_tc_handle_split_file_execution`
- `OPTIONS /api/go/v1/umacs_tc_enqueue_command`
- `OPTIONS /api/go/v1/umacs_tc_clear_command_queue`

## umacs-tc-emulator (default `0.0.0.0:21003`)

### Emulated UMACS endpoints
- `POST /api/go/v1/createProcedure`
- `POST /api/go/v1/validateProcedure`
- `POST /api/go/v1/loadProcedure`
- `POST /api/go/v1/getExeStatus`

### Admin/health
- `GET /api/go/v1/admin/procedures`
- `GET /api/go/v1/health`

## Services Without HTTP API Endpoints

These services run background workers and do not register HTTP routes in current code:
- `ingest`
- `chainmon`
- `comparator`
- `limiter`
- `storage`
- `launcher`

## Route Sources

- `gateway/internal/router.go`
- `simulator/internal/handler.go`
- `umacs-tc/internal/handler.go`
- `umacs-tc-emulator/internal/handler.go`
- Port configs: `gateway/config.yaml`, `simulator/config.yaml`, `umacs-tc/config.yaml`
- Emulator port default: `umacs-tc-emulator/cmd/main.go` (`--port` default)
