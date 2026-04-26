# Launcher

## Overview

The Launcher is a single-binary orchestrator that starts all TM-system services in parallel goroutines from one config file. It eliminates the need to manage multiple terminal windows or process managers during development and integration testing. Each service runs independently — a panic or failure in one service is caught and logged without affecting the others.

---

## Architecture

```
  launcher  (single process)
  ┌────────────────────────────────────────────────────────────┐
  │                                                            │
  │  goroutine: chainmon   (config: ../chainmon/config.yaml)   │
  │  goroutine: comparator (config: ../comparator/config.yaml) │
  │  goroutine: gateway    (config: ../gateway/config.yaml)    │
  │  goroutine: ingest     (config: ../ingest/config.yaml)     │
  │  goroutine: limiter    (config: ../limiter/config.yaml)    │
  │  goroutine: simulator  (config: ../simulator/config.yaml)  │
  │  goroutine: storage    (config: ../storage/config.yaml)    │
  │  goroutine: umacs_tc   (config: ../umacs-tc/config.yaml)   │
  │                                                            │
  │  shared context — cancelled on SIGINT / SIGTERM            │
  │  WaitGroup — 30 s hard shutdown timeout                    │
  └────────────────────────────────────────────────────────────┘
```

---

## How It Works

1. **Config loading** — reads `launcher/config.yaml` (or the path specified via `--config`). Each service entry has an `enabled` flag and a path to that service's own config file.

2. **Service startup** — for each enabled service, `startService()` launches a goroutine that:
   - Calls the service's `Run(ctx, configPath)` function.
   - Recovers from panics, logging the error without crashing the launcher.
   - Logs start and stop events.

3. **Shared context** — all goroutines receive the same `context.Context`. On `SIGINT`/`SIGTERM`, the context is cancelled, triggering graceful shutdown in every service simultaneously.

4. **Shutdown** — after the context is cancelled, the launcher waits for all goroutines to finish via `WaitGroup`. If any goroutine does not return within **30 seconds**, the process force-exits with code 1.

---

## Configuration (`config.yaml`)

```yaml
services:
  chainmon:
    enabled: true
    config: ../chainmon/config.yaml

  comparator:
    enabled: true
    config: ../comparator/config.yaml

  gateway:
    enabled: true
    config: ../gateway/config.yaml

  ingest:
    enabled: true
    config: ../ingest/config.yaml

  limiter:
    enabled: true
    config: ../limiter/config.yaml

  simulator:
    enabled: true
    config: ../simulator/config.yaml

  storage:
    enabled: true
    config: ../storage/config.yaml

  umacs_tc:
    enabled: true
    config: ../umacs-tc/config.yaml
```

| Field | Description |
|-------|-------------|
| `services.<name>.enabled` | Set to `false` to skip launching this service |
| `services.<name>.config` | Path to the service's own config file (relative to launcher working directory) |

To disable a service without removing it from the config, set `enabled: false`.

---

## How to Run

```bash
# From the launcher directory using defaults (config.yaml in current dir)
go run ./cmd

# Specify a custom launcher config path
go run ./cmd --config /etc/mainframe/launcher.yaml

# Build and run
go build -o launcher ./cmd
./launcher --config config.yaml
```

Shutdown: press `Ctrl+C` or send `SIGTERM`. The launcher will propagate the cancellation to all services and wait up to 30 seconds for clean shutdown.

---

## Startup Order

Services start in the following fixed order (defined in `main.go`):

1. `chainmon`
2. `comparator`
3. `gateway`
4. `ingest`
5. `limiter`
6. `simulator`
7. `storage`
8. `umacs_tc`

All start near-simultaneously in separate goroutines; the order matters only for log readability.

---

## Notes

- The launcher does **not** restart a failed service; failed goroutines log the error and exit. Use a process supervisor (e.g. systemd, supervisor) if auto-restart is needed in production.
- Each service connects to Redis independently with its own retry logic.
- The `umacs-tc-emulator` is **not** included in the launcher — it is started separately as a testing aid.
