# Internal Shared Library

**Purpose**
- Shared utilities and models used by all GoLang New services.

**How it works**
- Provides configuration loading, logging, and database clients with retry logic.
- Defines shared data models for telemetry, mnemonics, heartbeat payloads, and Redis keys.
- Exposes reusable WebSocket subscriber helpers for ingest services.

**Key modules**
- `config/` — YAML config loader and shared config structs.
- `clients/` — Redis, MongoDB, InfluxDB, and WebSocket clients.
- `models/` — telemetry and mnemonic models, Redis key builders.
- `logging/` — structured slog logger wrapper.

**Endpoints**
- No HTTP endpoints (library only).
