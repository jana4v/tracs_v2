"""
System Catalog: normalized storage for systems (transmitters/receivers/transponders),
their ports, frequency labels, and dataset row tables (specs, losses, calibration).

This module is additive. It coexists with the legacy JSON-in-transmitter storage in
`transmitters` and is intended to become the source of truth for spec/loss/profile
rows once the GUI is migrated.

Design summary
--------------
- Identity catalogs use surrogate INTEGER ids so that port names and frequency labels
  can be renamed without invalidating any dependent row.
- Per-system catalogs: each port/frequency belongs to one (system_kind, system_code).
- Row tables reference (system_kind, system_code, port_id, frequency_id) and carry a
  small JSON `payload_json` for irregular fields (e.g. spurious fbt matrices).
- A single `transmitter_spec_rows` table with a `parameter_type` discriminator covers
  power, frequency, modulation_index, and spurious specs.
"""

from __future__ import annotations

import json
import sqlite3
import threading
from pathlib import Path
from typing import Any, Iterable, Optional


SYSTEM_KIND_TRANSMITTER = "transmitter"
SYSTEM_KIND_RECEIVER = "receiver"
SYSTEM_KIND_TRANSPONDER = "transponder"

PARAMETER_TYPE_POWER = "power"
PARAMETER_TYPE_FREQUENCY = "frequency"
PARAMETER_TYPE_MODULATION_INDEX = "modulation_index"
PARAMETER_TYPE_SPURIOUS = "spurious"
PARAMETER_TYPE_COMMAND_THRESHOLD = "command_threshold"
PARAMETER_TYPE_RANGING_THRESHOLD = "ranging_threshold"

ALL_PARAMETER_TYPES = (
    PARAMETER_TYPE_POWER,
    PARAMETER_TYPE_FREQUENCY,
    PARAMETER_TYPE_MODULATION_INDEX,
    PARAMETER_TYPE_SPURIOUS,
    PARAMETER_TYPE_COMMAND_THRESHOLD,
    PARAMETER_TYPE_RANGING_THRESHOLD,
)

ALL_SYSTEM_KINDS = (
    SYSTEM_KIND_TRANSMITTER,
    SYSTEM_KIND_RECEIVER,
    SYSTEM_KIND_TRANSPONDER,
)


class SystemCatalogStore:
    """SQLite-backed store for the system catalog and dataset row tables."""

    def __init__(self, db_path: str) -> None:
        self._path = Path(db_path)
        if not self._path.is_absolute():
            self._path = Path.cwd() / self._path
        self._path.parent.mkdir(parents=True, exist_ok=True)
        self._lock = threading.Lock()
        self._conn = sqlite3.connect(str(self._path), check_same_thread=False)
        self._conn.row_factory = sqlite3.Row
        self._conn.execute("PRAGMA foreign_keys = ON")
        self._ensure_schema()

    # ---------- schema ----------
    def _ensure_schema(self) -> None:
        with self._lock:
            cur = self._conn.cursor()
            cur.executescript(
                """
                CREATE TABLE IF NOT EXISTS system_ports (
                    port_id      INTEGER PRIMARY KEY AUTOINCREMENT,
                    system_kind  TEXT    NOT NULL,
                    system_code  TEXT    NOT NULL,
                    port_name    TEXT    NOT NULL,
                    sort_order   INTEGER NOT NULL DEFAULT 0,
                    UNIQUE(system_kind, system_code, port_name)
                );

                CREATE TABLE IF NOT EXISTS system_frequencies (
                    frequency_id    INTEGER PRIMARY KEY AUTOINCREMENT,
                    system_kind     TEXT    NOT NULL,
                    system_code     TEXT    NOT NULL,
                    frequency_label TEXT    NOT NULL,
                    frequency_hz    TEXT    NOT NULL DEFAULT '',
                    sort_order      INTEGER NOT NULL DEFAULT 0,
                    UNIQUE(system_kind, system_code, frequency_label)
                );

                CREATE INDEX IF NOT EXISTS ix_system_ports_system
                    ON system_ports(system_kind, system_code);
                CREATE INDEX IF NOT EXISTS ix_system_frequencies_system
                    ON system_frequencies(system_kind, system_code);

                CREATE TABLE IF NOT EXISTS transmitter_spec_rows (
                    id              INTEGER PRIMARY KEY AUTOINCREMENT,
                    system_code     TEXT    NOT NULL,
                    parameter_type  TEXT    NOT NULL,
                    port_id         INTEGER NOT NULL,
                    frequency_id    INTEGER NOT NULL,
                    payload_json    TEXT    NOT NULL DEFAULT '{}',
                    sort_order      INTEGER NOT NULL DEFAULT 0,
                    UNIQUE(system_code, parameter_type, port_id, frequency_id),
                    FOREIGN KEY(port_id)      REFERENCES system_ports(port_id)
                        ON DELETE CASCADE,
                    FOREIGN KEY(frequency_id) REFERENCES system_frequencies(frequency_id)
                        ON DELETE CASCADE
                );
                CREATE INDEX IF NOT EXISTS ix_tx_spec_rows_lookup
                    ON transmitter_spec_rows(system_code, parameter_type);

                CREATE TABLE IF NOT EXISTS transmitter_onboard_loss_rows (
                    id            INTEGER PRIMARY KEY AUTOINCREMENT,
                    system_code   TEXT    NOT NULL,
                    port_id       INTEGER NOT NULL,
                    frequency_id  INTEGER NOT NULL,
                    loss_db       REAL,
                    payload_json  TEXT    NOT NULL DEFAULT '{}',
                    sort_order    INTEGER NOT NULL DEFAULT 0,
                    UNIQUE(system_code, port_id, frequency_id),
                    FOREIGN KEY(port_id)      REFERENCES system_ports(port_id)
                        ON DELETE CASCADE,
                    FOREIGN KEY(frequency_id) REFERENCES system_frequencies(frequency_id)
                        ON DELETE CASCADE
                );
                CREATE INDEX IF NOT EXISTS ix_tx_loss_rows_lookup
                    ON transmitter_onboard_loss_rows(system_code);

                CREATE TABLE IF NOT EXISTS transmitter_calibration_rows (
                    id            INTEGER PRIMARY KEY AUTOINCREMENT,
                    system_code   TEXT    NOT NULL,
                    port_id       INTEGER NOT NULL,
                    frequency_id  INTEGER NOT NULL,
                    payload_json  TEXT    NOT NULL DEFAULT '{}',
                    sort_order    INTEGER NOT NULL DEFAULT 0,
                    UNIQUE(system_code, port_id, frequency_id),
                    FOREIGN KEY(port_id)      REFERENCES system_ports(port_id)
                        ON DELETE CASCADE,
                    FOREIGN KEY(frequency_id) REFERENCES system_frequencies(frequency_id)
                        ON DELETE CASCADE
                );
                CREATE INDEX IF NOT EXISTS ix_tx_cal_rows_lookup
                    ON transmitter_calibration_rows(system_code);

                CREATE TABLE IF NOT EXISTS system_test_plan_rows (
                    id            INTEGER PRIMARY KEY AUTOINCREMENT,
                    system_kind   TEXT    NOT NULL,
                    system_code   TEXT    NOT NULL,
                    port_id       INTEGER,
                    frequency_id  INTEGER,
                    plan_name     TEXT    NOT NULL DEFAULT '',
                    payload_json  TEXT    NOT NULL DEFAULT '{}',
                    sort_order    INTEGER NOT NULL DEFAULT 0,
                    FOREIGN KEY(port_id)      REFERENCES system_ports(port_id)
                        ON DELETE SET NULL,
                    FOREIGN KEY(frequency_id) REFERENCES system_frequencies(frequency_id)
                        ON DELETE SET NULL
                );
                CREATE INDEX IF NOT EXISTS ix_system_test_plan_rows_lookup
                    ON system_test_plan_rows(system_kind, system_code, plan_name);

                CREATE TABLE IF NOT EXISTS system_profile_rows (
                    id            INTEGER PRIMARY KEY AUTOINCREMENT,
                    system_kind   TEXT    NOT NULL,
                    system_code   TEXT    NOT NULL,
                    profile_name  TEXT    NOT NULL,
                    payload_json  TEXT    NOT NULL DEFAULT '{}',
                    sort_order    INTEGER NOT NULL DEFAULT 0,
                    UNIQUE(system_kind, system_code, profile_name)
                );
                CREATE INDEX IF NOT EXISTS ix_system_profile_rows_lookup
                    ON system_profile_rows(system_kind, system_code);

                CREATE TABLE IF NOT EXISTS ranging_tones (
                    id         INTEGER PRIMARY KEY AUTOINCREMENT,
                    tone_khz   TEXT    NOT NULL UNIQUE,
                    sort_order INTEGER NOT NULL DEFAULT 0
                );

                CREATE TABLE IF NOT EXISTS ranging_threshold_rows (
                    id               INTEGER PRIMARY KEY AUTOINCREMENT,
                    transponder_code TEXT    NOT NULL,
                    tone_id          INTEGER NOT NULL,
                    uplink           TEXT    NOT NULL DEFAULT '',
                    downlink         TEXT    NOT NULL DEFAULT '',
                    max_input_power  REAL    DEFAULT -60,
                    specification    REAL,
                    tolerance        REAL,
                    fbt_json         TEXT    NOT NULL DEFAULT 'null',
                    fbt_hot_json     TEXT    NOT NULL DEFAULT 'null',
                    fbt_cold_json    TEXT    NOT NULL DEFAULT 'null',
                    sort_order       INTEGER NOT NULL DEFAULT 0,
                    UNIQUE(transponder_code, tone_id),
                    FOREIGN KEY(tone_id) REFERENCES ranging_tones(id) ON DELETE CASCADE
                );

                CREATE TABLE IF NOT EXISTS catalog_migrations (
                    name        TEXT PRIMARY KEY,
                    executed_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
                );

                CREATE TABLE IF NOT EXISTS onboard_losses (
                    id          INTEGER PRIMARY KEY AUTOINCREMENT,
                    source_type TEXT    NOT NULL,
                    code        TEXT    NOT NULL DEFAULT '',
                    port        TEXT    NOT NULL DEFAULT '',
                    frequency   TEXT    NOT NULL DEFAULT '',
                    freq_label  TEXT    NOT NULL DEFAULT '',
                    loss_db     REAL,
                    sort_order  INTEGER NOT NULL DEFAULT 0,
                    UNIQUE(source_type, code, port, freq_label)
                );
                CREATE INDEX IF NOT EXISTS ix_onboard_losses_source
                    ON onboard_losses(source_type, code);
                """
            )
            self._ensure_compat_columns()
            self._seed_ranging_tones()
            self._conn.commit()

    def _ensure_compat_columns(self) -> None:
        """Backfill additive columns on existing DBs created in earlier phases."""
        self._ensure_profile_table_no_fk()
        self._ensure_column_exists(
            table_name="transmitter_spec_rows",
            column_name="system_kind",
            column_def="TEXT NOT NULL DEFAULT 'transmitter'",
        )
        self._ensure_column_exists(
            table_name="transmitter_onboard_loss_rows",
            column_name="system_kind",
            column_def="TEXT NOT NULL DEFAULT 'transmitter'",
        )
        self._ensure_column_exists(
            table_name="transmitter_calibration_rows",
            column_name="system_kind",
            column_def="TEXT NOT NULL DEFAULT 'transmitter'",
        )

    def _ensure_column_exists(
        self,
        table_name: str,
        column_name: str,
        column_def: str,
    ) -> None:
        columns = self._conn.execute(
            f"PRAGMA table_info({table_name})"
        ).fetchall()
        if any(str(row["name"]) == column_name for row in columns):
            return
        self._conn.execute(
            f"ALTER TABLE {table_name} ADD COLUMN {column_name} {column_def}"
        )

    def _ensure_profile_table_no_fk(self) -> None:
        fk_rows = self._conn.execute(
            "PRAGMA foreign_key_list(system_profile_rows)"
        ).fetchall()
        if len(fk_rows) == 0:
            return

        # Rebuild table in place to drop invalid foreign keys from older schema.
        self._conn.execute(
            """
            CREATE TABLE IF NOT EXISTS system_profile_rows_v2 (
                id            INTEGER PRIMARY KEY AUTOINCREMENT,
                system_kind   TEXT    NOT NULL,
                system_code   TEXT    NOT NULL,
                profile_name  TEXT    NOT NULL,
                payload_json  TEXT    NOT NULL DEFAULT '{}',
                sort_order    INTEGER NOT NULL DEFAULT 0,
                UNIQUE(system_kind, system_code, profile_name)
            )
            """
        )
        self._conn.execute(
            """
            INSERT OR REPLACE INTO system_profile_rows_v2
                (id, system_kind, system_code, profile_name, payload_json, sort_order)
            SELECT id, system_kind, system_code, profile_name, payload_json, sort_order
            FROM system_profile_rows
            """
        )
        self._conn.execute("DROP TABLE system_profile_rows")
        self._conn.execute("ALTER TABLE system_profile_rows_v2 RENAME TO system_profile_rows")

    def _seed_ranging_tones(self) -> None:
        """Insert default ranging tones if not already present."""
        defaults = [("22", 0), ("27.777", 1)]
        for tone_khz, sort_order in defaults:
            self._conn.execute(
                """
                INSERT OR IGNORE INTO ranging_tones (tone_khz, sort_order)
                VALUES (?, ?)
                """,
                (tone_khz, sort_order),
            )

    # ---------- low-level helpers ----------
    def conn(self) -> sqlite3.Connection:
        return self._conn

    def lock(self) -> threading.Lock:
        return self._lock

    # ---------- ports ----------
    def list_ports(self, system_kind: str, system_code: str) -> list[dict[str, Any]]:
        rows = self._conn.execute(
            """
            SELECT port_id, system_kind, system_code, port_name, sort_order
            FROM system_ports
            WHERE system_kind = ? AND system_code = ?
            ORDER BY sort_order ASC, port_name ASC
            """,
            (system_kind, system_code),
        ).fetchall()
        return [dict(r) for r in rows]

    def upsert_port(
        self,
        system_kind: str,
        system_code: str,
        port_name: str,
        sort_order: int = 0,
    ) -> int:
        """Insert if missing, return port_id. Idempotent on (kind, code, name)."""
        with self._lock:
            row = self._conn.execute(
                """
                SELECT port_id FROM system_ports
                WHERE system_kind = ? AND system_code = ? AND port_name = ?
                """,
                (system_kind, system_code, port_name),
            ).fetchone()
            if row is not None:
                self._conn.execute(
                    "UPDATE system_ports SET sort_order = ? WHERE port_id = ?",
                    (sort_order, row["port_id"]),
                )
                self._conn.commit()
                return int(row["port_id"])
            cur = self._conn.execute(
                """
                INSERT INTO system_ports
                    (system_kind, system_code, port_name, sort_order)
                VALUES (?, ?, ?, ?)
                """,
                (system_kind, system_code, port_name, sort_order),
            )
            self._conn.commit()
            return int(cur.lastrowid)

    def rename_port(self, port_id: int, new_name: str) -> None:
        with self._lock:
            self._conn.execute(
                "UPDATE system_ports SET port_name = ? WHERE port_id = ?",
                (new_name, port_id),
            )
            self._conn.commit()

    def update_port_sort_order(self, port_id: int, sort_order: int) -> None:
        with self._lock:
            self._conn.execute(
                "UPDATE system_ports SET sort_order = ? WHERE port_id = ?",
                (sort_order, port_id),
            )
            self._conn.commit()

    def delete_port(self, port_id: int) -> None:
        """Delete a port. Cascades to dependent row tables via FK."""
        with self._lock:
            self._conn.execute(
                "DELETE FROM system_ports WHERE port_id = ?", (port_id,)
            )
            self._conn.commit()

    def port_dependent_count(self, port_id: int) -> int:
        row = self._conn.execute(
            """
            SELECT
              (SELECT COUNT(1) FROM transmitter_spec_rows         WHERE port_id = ?)
            + (SELECT COUNT(1) FROM transmitter_onboard_loss_rows WHERE port_id = ?)
            + (SELECT COUNT(1) FROM transmitter_calibration_rows  WHERE port_id = ?) AS c
            """,
            (port_id, port_id, port_id),
        ).fetchone()
        return int(row["c"] or 0)

    # ---------- frequencies ----------
    def list_frequencies(self, system_kind: str, system_code: str) -> list[dict[str, Any]]:
        rows = self._conn.execute(
            """
            SELECT frequency_id, system_kind, system_code,
                   frequency_label, frequency_hz, sort_order
            FROM system_frequencies
            WHERE system_kind = ? AND system_code = ?
            ORDER BY sort_order ASC, frequency_label ASC
            """,
            (system_kind, system_code),
        ).fetchall()
        return [dict(r) for r in rows]

    def upsert_frequency(
        self,
        system_kind: str,
        system_code: str,
        frequency_label: str,
        frequency_hz: str = "",
        sort_order: int = 0,
    ) -> int:
        with self._lock:
            row = self._conn.execute(
                """
                SELECT frequency_id FROM system_frequencies
                WHERE system_kind = ? AND system_code = ? AND frequency_label = ?
                """,
                (system_kind, system_code, frequency_label),
            ).fetchone()
            if row is not None:
                self._conn.execute(
                    """
                    UPDATE system_frequencies
                    SET frequency_hz = ?, sort_order = ?
                    WHERE frequency_id = ?
                    """,
                    (frequency_hz, sort_order, row["frequency_id"]),
                )
                self._conn.commit()
                return int(row["frequency_id"])
            cur = self._conn.execute(
                """
                INSERT INTO system_frequencies
                    (system_kind, system_code, frequency_label, frequency_hz, sort_order)
                VALUES (?, ?, ?, ?, ?)
                """,
                (system_kind, system_code, frequency_label, frequency_hz, sort_order),
            )
            self._conn.commit()
            return int(cur.lastrowid)

    def rename_frequency(
        self,
        frequency_id: int,
        new_label: Optional[str] = None,
        new_hz: Optional[str] = None,
    ) -> None:
        with self._lock:
            sets: list[str] = []
            params: list[Any] = []
            if new_label is not None:
                sets.append("frequency_label = ?")
                params.append(new_label)
            if new_hz is not None:
                sets.append("frequency_hz = ?")
                params.append(new_hz)
            if not sets:
                return
            params.append(frequency_id)
            self._conn.execute(
                f"UPDATE system_frequencies SET {', '.join(sets)} WHERE frequency_id = ?",
                params,
            )
            self._conn.commit()

    def update_frequency_sort_order(self, frequency_id: int, sort_order: int) -> None:
        with self._lock:
            self._conn.execute(
                "UPDATE system_frequencies SET sort_order = ? WHERE frequency_id = ?",
                (sort_order, frequency_id),
            )
            self._conn.commit()

    def delete_frequency(self, frequency_id: int) -> None:
        with self._lock:
            self._conn.execute(
                "DELETE FROM system_frequencies WHERE frequency_id = ?",
                (frequency_id,),
            )
            self._conn.commit()

    def frequency_dependent_count(self, frequency_id: int) -> int:
        row = self._conn.execute(
            """
            SELECT
              (SELECT COUNT(1) FROM transmitter_spec_rows         WHERE frequency_id = ?)
            + (SELECT COUNT(1) FROM transmitter_onboard_loss_rows WHERE frequency_id = ?)
            + (SELECT COUNT(1) FROM transmitter_calibration_rows  WHERE frequency_id = ?) AS c
            """,
            (frequency_id, frequency_id, frequency_id),
        ).fetchone()
        return int(row["c"] or 0)

    # ---------- system rename / delete ----------
    def rename_system_code(
        self, system_kind: str, old_code: str, new_code: str
    ) -> None:
        """Cascade-rename system_code across catalog and row tables."""
        with self._lock:
            with self._conn:
                self._conn.execute(
                    "UPDATE system_ports SET system_code = ? "
                    "WHERE system_kind = ? AND system_code = ?",
                    (new_code, system_kind, old_code),
                )
                self._conn.execute(
                    "UPDATE system_frequencies SET system_code = ? "
                    "WHERE system_kind = ? AND system_code = ?",
                    (new_code, system_kind, old_code),
                )
                if system_kind == SYSTEM_KIND_TRANSMITTER:
                    self._conn.execute(
                        "UPDATE transmitter_spec_rows SET system_code = ? "
                        "WHERE system_kind = ? AND system_code = ?",
                        (new_code, system_kind, old_code),
                    )
                    self._conn.execute(
                        "UPDATE transmitter_onboard_loss_rows SET system_code = ? "
                        "WHERE system_kind = ? AND system_code = ?",
                        (new_code, system_kind, old_code),
                    )
                    self._conn.execute(
                        "UPDATE transmitter_calibration_rows SET system_code = ? "
                        "WHERE system_kind = ? AND system_code = ?",
                        (new_code, system_kind, old_code),
                    )
                self._conn.execute(
                    "UPDATE system_test_plan_rows SET system_code = ? "
                    "WHERE system_kind = ? AND system_code = ?",
                    (new_code, system_kind, old_code),
                )
                self._conn.execute(
                    "UPDATE system_profile_rows SET system_code = ? "
                    "WHERE system_kind = ? AND system_code = ?",
                    (new_code, system_kind, old_code),
                )

    def delete_system(self, system_kind: str, system_code: str) -> None:
        """Cascade-delete catalog + dependent rows for one system."""
        with self._lock:
            with self._conn:
                if system_kind == SYSTEM_KIND_TRANSMITTER:
                    self._conn.execute(
                        "DELETE FROM transmitter_spec_rows WHERE system_kind = ? AND system_code = ?",
                        (system_kind, system_code),
                    )
                    self._conn.execute(
                        "DELETE FROM transmitter_onboard_loss_rows WHERE system_kind = ? AND system_code = ?",
                        (system_kind, system_code),
                    )
                    self._conn.execute(
                        "DELETE FROM transmitter_calibration_rows WHERE system_kind = ? AND system_code = ?",
                        (system_kind, system_code),
                    )
                self._conn.execute(
                    "DELETE FROM system_test_plan_rows WHERE system_kind = ? AND system_code = ?",
                    (system_kind, system_code),
                )
                self._conn.execute(
                    "DELETE FROM system_profile_rows WHERE system_kind = ? AND system_code = ?",
                    (system_kind, system_code),
                )
                self._conn.execute(
                    "DELETE FROM system_ports "
                    "WHERE system_kind = ? AND system_code = ?",
                    (system_kind, system_code),
                )
                self._conn.execute(
                    "DELETE FROM system_frequencies "
                    "WHERE system_kind = ? AND system_code = ?",
                    (system_kind, system_code),
                )

    # ---------- spec rows ----------
    def list_spec_rows(
        self,
        system_kind: str = SYSTEM_KIND_TRANSMITTER,
        system_code: Optional[str] = None,
        parameter_type: Optional[str] = None,
    ) -> list[dict[str, Any]]:
        sql = """
            SELECT r.id, r.system_kind, r.system_code, r.parameter_type,
                   r.port_id, p.port_name,
                   r.frequency_id, f.frequency_label, f.frequency_hz,
                   r.sort_order, r.payload_json
            FROM transmitter_spec_rows r
            JOIN system_ports        p ON p.port_id = r.port_id
            JOIN system_frequencies  f ON f.frequency_id = r.frequency_id
        """
        clauses: list[str] = ["r.system_kind = ?"]
        params: list[Any] = [system_kind]
        if system_code is not None:
            clauses.append("r.system_code = ?")
            params.append(system_code)
        if parameter_type is not None:
            clauses.append("r.parameter_type = ?")
            params.append(parameter_type)
        if clauses:
            sql += " WHERE " + " AND ".join(clauses)
        sql += " ORDER BY r.system_code, r.parameter_type, r.sort_order, p.port_name, f.frequency_label"

        rows = self._conn.execute(sql, params).fetchall()
        out: list[dict[str, Any]] = []
        for r in rows:
            d = dict(r)
            try:
                d["payload"] = json.loads(d.pop("payload_json") or "{}")
            except Exception:
                d["payload"] = {}
            out.append(d)
        return out

    def upsert_spec_row(
        self,
        system_kind: str,
        system_code: str,
        parameter_type: str,
        port_id: int,
        frequency_id: int,
        payload: dict[str, Any],
        sort_order: int = 0,
    ) -> int:
        payload_json = json.dumps(payload, ensure_ascii=True, default=str)
        with self._lock:
            row = self._conn.execute(
                """
                SELECT id FROM transmitter_spec_rows
                                WHERE system_kind = ? AND system_code = ? AND parameter_type = ?
                  AND port_id = ? AND frequency_id = ?
                """,
                                (system_kind, system_code, parameter_type, port_id, frequency_id),
            ).fetchone()
            if row is not None:
                self._conn.execute(
                    """
                    UPDATE transmitter_spec_rows
                    SET payload_json = ?, sort_order = ?
                    WHERE id = ?
                    """,
                    (payload_json, sort_order, row["id"]),
                )
                self._conn.commit()
                return int(row["id"])
            cur = self._conn.execute(
                """
                INSERT INTO transmitter_spec_rows
                    (system_kind, system_code, parameter_type, port_id, frequency_id,
                     payload_json, sort_order)
                VALUES (?, ?, ?, ?, ?, ?, ?)
                """,
                (
                    system_kind,
                    system_code,
                    parameter_type,
                    port_id,
                    frequency_id,
                    payload_json,
                    sort_order,
                ),
            )
            self._conn.commit()
            return int(cur.lastrowid)

    def delete_spec_row(self, row_id: int) -> None:
        with self._lock:
            self._conn.execute(
                "DELETE FROM transmitter_spec_rows WHERE id = ?", (row_id,)
            )
            self._conn.commit()

    # ---------- loss rows ----------
    def list_loss_rows(self, system_code: Optional[str] = None) -> list[dict[str, Any]]:
        return self.list_loss_rows_for_kind(SYSTEM_KIND_TRANSMITTER, system_code)

    def list_loss_rows_for_kind(
        self,
        system_kind: str,
        system_code: Optional[str] = None,
    ) -> list[dict[str, Any]]:
        sql = """
            SELECT r.id, r.system_kind, r.system_code,
                   r.port_id, p.port_name,
                   r.frequency_id, f.frequency_label, f.frequency_hz,
                   r.loss_db, r.sort_order, r.payload_json
            FROM transmitter_onboard_loss_rows r
            JOIN system_ports       p ON p.port_id = r.port_id
            JOIN system_frequencies f ON f.frequency_id = r.frequency_id
        """
        params: list[Any] = [system_kind]
        sql += " WHERE r.system_kind = ?"
        if system_code is not None:
            sql += " AND r.system_code = ?"
            params.append(system_code)
        sql += " ORDER BY r.system_code, r.sort_order, p.port_name, f.frequency_label"
        rows = self._conn.execute(sql, params).fetchall()
        out: list[dict[str, Any]] = []
        for r in rows:
            d = dict(r)
            try:
                d["payload"] = json.loads(d.pop("payload_json") or "{}")
            except Exception:
                d["payload"] = {}
            out.append(d)
        return out

    def upsert_loss_row(
        self,
        system_kind: str,
        system_code: str,
        port_id: int,
        frequency_id: int,
        loss_db: Optional[float],
        payload: Optional[dict[str, Any]] = None,
        sort_order: int = 0,
    ) -> int:
        payload_json = json.dumps(payload or {}, ensure_ascii=True, default=str)
        with self._lock:
            row = self._conn.execute(
                """
                SELECT id FROM transmitter_onboard_loss_rows
                WHERE system_kind = ? AND system_code = ? AND port_id = ? AND frequency_id = ?
                """,
                (system_kind, system_code, port_id, frequency_id),
            ).fetchone()
            if row is not None:
                self._conn.execute(
                    """
                    UPDATE transmitter_onboard_loss_rows
                    SET loss_db = ?, payload_json = ?, sort_order = ?
                    WHERE id = ?
                    """,
                    (loss_db, payload_json, sort_order, row["id"]),
                )
                self._conn.commit()
                return int(row["id"])
            cur = self._conn.execute(
                """
                INSERT INTO transmitter_onboard_loss_rows
                    (system_kind, system_code, port_id, frequency_id, loss_db, payload_json, sort_order)
                VALUES (?, ?, ?, ?, ?, ?, ?)
                """,
                (system_kind, system_code, port_id, frequency_id, loss_db, payload_json, sort_order),
            )
            self._conn.commit()
            return int(cur.lastrowid)

    # ---------- calibration rows ----------
    def list_calibration_rows(
        self, system_code: Optional[str] = None
    ) -> list[dict[str, Any]]:
        return self.list_calibration_rows_for_kind(SYSTEM_KIND_TRANSMITTER, system_code)

    def list_calibration_rows_for_kind(
        self,
        system_kind: str,
        system_code: Optional[str] = None,
    ) -> list[dict[str, Any]]:
        sql = """
            SELECT r.id, r.system_kind, r.system_code,
                   r.port_id, p.port_name,
                   r.frequency_id, f.frequency_label, f.frequency_hz,
                   r.sort_order, r.payload_json
            FROM transmitter_calibration_rows r
            JOIN system_ports       p ON p.port_id = r.port_id
            JOIN system_frequencies f ON f.frequency_id = r.frequency_id
        """
        params: list[Any] = [system_kind]
        sql += " WHERE r.system_kind = ?"
        if system_code is not None:
            sql += " AND r.system_code = ?"
            params.append(system_code)
        sql += " ORDER BY r.system_code, r.sort_order, p.port_name, f.frequency_label"
        rows = self._conn.execute(sql, params).fetchall()
        out: list[dict[str, Any]] = []
        for r in rows:
            d = dict(r)
            try:
                d["payload"] = json.loads(d.pop("payload_json") or "{}")
            except Exception:
                d["payload"] = {}
            out.append(d)
        return out

    def upsert_calibration_row(
        self,
        system_kind: str,
        system_code: str,
        port_id: int,
        frequency_id: int,
        payload: dict[str, Any],
        sort_order: int = 0,
    ) -> int:
        payload_json = json.dumps(payload, ensure_ascii=True, default=str)
        with self._lock:
            row = self._conn.execute(
                """
                SELECT id FROM transmitter_calibration_rows
                WHERE system_kind = ? AND system_code = ? AND port_id = ? AND frequency_id = ?
                """,
                (system_kind, system_code, port_id, frequency_id),
            ).fetchone()
            if row is not None:
                self._conn.execute(
                    """
                    UPDATE transmitter_calibration_rows
                    SET payload_json = ?, sort_order = ?
                    WHERE id = ?
                    """,
                    (payload_json, sort_order, row["id"]),
                )
                self._conn.commit()
                return int(row["id"])
            cur = self._conn.execute(
                """
                INSERT INTO transmitter_calibration_rows
                    (system_kind, system_code, port_id, frequency_id, payload_json, sort_order)
                VALUES (?, ?, ?, ?, ?, ?)
                """,
                (system_kind, system_code, port_id, frequency_id, payload_json, sort_order),
            )
            self._conn.commit()
            return int(cur.lastrowid)

    def list_test_plan_rows(
        self,
        system_kind: str,
        system_code: Optional[str] = None,
    ) -> list[dict[str, Any]]:
        sql = """
            SELECT r.id, r.system_kind, r.system_code,
                   r.port_id, p.port_name,
                   r.frequency_id, f.frequency_label, f.frequency_hz,
                   r.plan_name, r.sort_order, r.payload_json
            FROM system_test_plan_rows r
            LEFT JOIN system_ports       p ON p.port_id = r.port_id
            LEFT JOIN system_frequencies f ON f.frequency_id = r.frequency_id
            WHERE r.system_kind = ?
        """
        params: list[Any] = [system_kind]
        if system_code is not None:
            sql += " AND r.system_code = ?"
            params.append(system_code)
        sql += " ORDER BY r.system_code, r.plan_name, r.sort_order, p.port_name, f.frequency_label"

        rows = self._conn.execute(sql, params).fetchall()
        out: list[dict[str, Any]] = []
        for r in rows:
            d = dict(r)
            try:
                d["payload"] = json.loads(d.pop("payload_json") or "{}")
            except Exception:
                d["payload"] = {}
            out.append(d)
        return out

    def upsert_test_plan_row(
        self,
        system_kind: str,
        system_code: str,
        plan_name: str,
        payload: dict[str, Any],
        port_id: Optional[int] = None,
        frequency_id: Optional[int] = None,
        sort_order: int = 0,
    ) -> int:
        payload_json = json.dumps(payload, ensure_ascii=True, default=str)
        with self._lock:
            row = self._conn.execute(
                """
                SELECT id FROM system_test_plan_rows
                WHERE system_kind = ? AND system_code = ? AND plan_name = ?
                  AND COALESCE(port_id, -1) = COALESCE(?, -1)
                  AND COALESCE(frequency_id, -1) = COALESCE(?, -1)
                """,
                (system_kind, system_code, plan_name, port_id, frequency_id),
            ).fetchone()
            if row is not None:
                self._conn.execute(
                    """
                    UPDATE system_test_plan_rows
                    SET payload_json = ?, sort_order = ?
                    WHERE id = ?
                    """,
                    (payload_json, sort_order, row["id"]),
                )
                self._conn.commit()
                return int(row["id"])

            cur = self._conn.execute(
                """
                INSERT INTO system_test_plan_rows
                    (system_kind, system_code, port_id, frequency_id,
                     plan_name, payload_json, sort_order)
                VALUES (?, ?, ?, ?, ?, ?, ?)
                """,
                (
                    system_kind,
                    system_code,
                    port_id,
                    frequency_id,
                    plan_name,
                    payload_json,
                    sort_order,
                ),
            )
            self._conn.commit()
            return int(cur.lastrowid)

    def list_profile_rows(
        self,
        system_kind: str,
        system_code: Optional[str] = None,
    ) -> list[dict[str, Any]]:
        sql = """
            SELECT id, system_kind, system_code, profile_name, sort_order, payload_json
            FROM system_profile_rows
            WHERE system_kind = ?
        """
        params: list[Any] = [system_kind]
        if system_code is not None:
            sql += " AND system_code = ?"
            params.append(system_code)
        sql += " ORDER BY system_code, sort_order, profile_name"

        rows = self._conn.execute(sql, params).fetchall()
        out: list[dict[str, Any]] = []
        for r in rows:
            d = dict(r)
            try:
                d["payload"] = json.loads(d.pop("payload_json") or "{}")
            except Exception:
                d["payload"] = {}
            out.append(d)
        return out

    def upsert_profile_row(
        self,
        system_kind: str,
        system_code: str,
        profile_name: str,
        payload: dict[str, Any],
        sort_order: int = 0,
    ) -> int:
        payload_json = json.dumps(payload, ensure_ascii=True, default=str)
        with self._lock:
            row = self._conn.execute(
                """
                SELECT id FROM system_profile_rows
                WHERE system_kind = ? AND system_code = ? AND profile_name = ?
                """,
                (system_kind, system_code, profile_name),
            ).fetchone()
            if row is not None:
                self._conn.execute(
                    """
                    UPDATE system_profile_rows
                    SET payload_json = ?, sort_order = ?
                    WHERE id = ?
                    """,
                    (payload_json, sort_order, row["id"]),
                )
                self._conn.commit()
                return int(row["id"])

            cur = self._conn.execute(
                """
                INSERT INTO system_profile_rows
                    (system_kind, system_code, profile_name, payload_json, sort_order)
                VALUES (?, ?, ?, ?, ?)
                """,
                (system_kind, system_code, profile_name, payload_json, sort_order),
            )
            self._conn.commit()
            return int(cur.lastrowid)

    # ---------- migrations ----------
    def has_migration(self, name: str) -> bool:
        row = self._conn.execute(
            "SELECT 1 FROM catalog_migrations WHERE name = ?", (name,)
        ).fetchone()
        return row is not None

    def mark_migration(self, name: str) -> None:
        with self._lock:
            self._conn.execute(
                "INSERT OR IGNORE INTO catalog_migrations(name) VALUES (?)", (name,)
            )
            self._conn.commit()


    # ---------- ranging tones ----------
    def list_ranging_tones(self) -> list[dict[str, Any]]:
        rows = self._conn.execute(
            "SELECT id, tone_khz, sort_order FROM ranging_tones ORDER BY sort_order ASC, tone_khz ASC"
        ).fetchall()
        return [dict(r) for r in rows]

    # ---------- ranging threshold rows ----------
    def list_ranging_threshold_rows(
        self,
        transponder_code: Optional[str] = None,
    ) -> list[dict[str, Any]]:
        sql = """
            SELECT r.id, r.transponder_code, r.tone_id, t.tone_khz,
                   r.uplink, r.downlink,
                   r.max_input_power, r.specification, r.tolerance,
                   r.fbt_json, r.fbt_hot_json, r.fbt_cold_json, r.sort_order
            FROM ranging_threshold_rows r
            JOIN ranging_tones t ON t.id = r.tone_id
        """
        params: list[Any] = []
        if transponder_code is not None:
            sql += " WHERE r.transponder_code = ?"
            params.append(transponder_code)
        sql += " ORDER BY r.transponder_code, t.sort_order, r.sort_order"
        rows = self._conn.execute(sql, params).fetchall()
        out: list[dict[str, Any]] = []
        for r in rows:
            d = dict(r)
            for field in ("fbt_json", "fbt_hot_json", "fbt_cold_json"):
                key = field.replace("_json", "")
                try:
                    d[key] = json.loads(d.pop(field) or "null")
                except Exception:
                    d[key] = None
            out.append(d)
        return out

    def upsert_ranging_threshold_row(
        self,
        transponder_code: str,
        tone_id: int,
        uplink: str,
        downlink: str,
        max_input_power: Optional[float],
        specification: Optional[float],
        tolerance: Optional[float],
        fbt: Any,
        fbt_hot: Any,
        fbt_cold: Any,
        sort_order: int = 0,
    ) -> int:
        fbt_json = json.dumps(fbt, ensure_ascii=True, default=str)
        fbt_hot_json = json.dumps(fbt_hot, ensure_ascii=True, default=str)
        fbt_cold_json = json.dumps(fbt_cold, ensure_ascii=True, default=str)
        with self._lock:
            row = self._conn.execute(
                """
                SELECT id FROM ranging_threshold_rows
                WHERE transponder_code = ? AND tone_id = ?
                """,
                (transponder_code, tone_id),
            ).fetchone()
            if row is not None:
                self._conn.execute(
                    """
                    UPDATE ranging_threshold_rows
                    SET uplink = ?, downlink = ?, max_input_power = ?,
                        specification = ?, tolerance = ?,
                        fbt_json = ?, fbt_hot_json = ?, fbt_cold_json = ?,
                        sort_order = ?
                    WHERE id = ?
                    """,
                    (
                        uplink, downlink, max_input_power,
                        specification, tolerance,
                        fbt_json, fbt_hot_json, fbt_cold_json,
                        sort_order, row["id"],
                    ),
                )
                self._conn.commit()
                return int(row["id"])
            cur = self._conn.execute(
                """
                INSERT INTO ranging_threshold_rows
                    (transponder_code, tone_id, uplink, downlink,
                     max_input_power, specification, tolerance,
                     fbt_json, fbt_hot_json, fbt_cold_json, sort_order)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                """,
                (
                    transponder_code, tone_id, uplink, downlink,
                    max_input_power, specification, tolerance,
                    fbt_json, fbt_hot_json, fbt_cold_json, sort_order,
                ),
            )
            self._conn.commit()
            return int(cur.lastrowid)

    def delete_ranging_threshold_row(self, row_id: int) -> None:
        with self._lock:
            self._conn.execute(
                "DELETE FROM ranging_threshold_rows WHERE id = ?", (row_id,)
            )
            self._conn.commit()

    # ---------- onboard losses ----------
    def list_onboard_losses(self, source_type: str) -> list[dict[str, Any]]:
        rows = self._conn.execute(
            """
            SELECT id, source_type, code, port, frequency, freq_label, loss_db, sort_order
            FROM onboard_losses
            WHERE source_type = ?
            ORDER BY code, sort_order, port, freq_label
            """,
            (source_type,),
        ).fetchall()
        return [dict(r) for r in rows]

    def upsert_onboard_loss(
        self,
        source_type: str,
        code: str,
        port: str,
        frequency: str,
        freq_label: str,
        loss_db: Optional[float] = None,
        sort_order: int = 0,
        preserve_existing_loss: bool = True,
    ) -> int:
        with self._lock:
            row = self._conn.execute(
                """
                SELECT id FROM onboard_losses
                WHERE source_type = ? AND code = ? AND port = ? AND freq_label = ?
                """,
                (source_type, code, port, freq_label),
            ).fetchone()
            if row is not None:
                if preserve_existing_loss:
                    # Only update frequency and sort_order; keep existing loss_db
                    self._conn.execute(
                        """
                        UPDATE onboard_losses
                        SET frequency = ?, sort_order = ?
                        WHERE id = ?
                        """,
                        (frequency, sort_order, row["id"]),
                    )
                else:
                    self._conn.execute(
                        """
                        UPDATE onboard_losses
                        SET frequency = ?, loss_db = ?, sort_order = ?
                        WHERE id = ?
                        """,
                        (frequency, loss_db, sort_order, row["id"]),
                    )
                self._conn.commit()
                return int(row["id"])
            cur = self._conn.execute(
                """
                INSERT INTO onboard_losses
                    (source_type, code, port, frequency, freq_label, loss_db, sort_order)
                VALUES (?, ?, ?, ?, ?, ?, ?)
                """,
                (source_type, code, port, frequency, freq_label, loss_db, sort_order),
            )
            self._conn.commit()
            return int(cur.lastrowid)

    def update_onboard_loss_db(self, row_id: int, loss_db: Optional[float]) -> None:
        with self._lock:
            self._conn.execute(
                "UPDATE onboard_losses SET loss_db = ? WHERE id = ?",
                (loss_db, row_id),
            )
            self._conn.commit()

    def delete_onboard_loss(self, row_id: int) -> None:
        with self._lock:
            self._conn.execute(
                "DELETE FROM onboard_losses WHERE id = ?", (row_id,)
            )
            self._conn.commit()

    def delete_onboard_losses_for_code(self, source_type: str, code: str) -> None:
        with self._lock:
            self._conn.execute(
                "DELETE FROM onboard_losses WHERE source_type = ? AND code = ?",
                (source_type, code),
            )
            self._conn.commit()


_store_singleton: Optional[SystemCatalogStore] = None
_store_lock = threading.Lock()


def get_catalog_store() -> SystemCatalogStore:
    """Module-level singleton bound to the project SQLite database."""
    global _store_singleton
    if _store_singleton is None:
        with _store_lock:
            if _store_singleton is None:
                # Imported lazily to avoid a circular import.
                from src.database.connection import Database

                _store_singleton = SystemCatalogStore(Database._db_path)
    return _store_singleton


def reset_catalog_store_for_tests() -> None:
    """Test helper to drop the singleton (does not delete data)."""
    global _store_singleton
    with _store_lock:
        _store_singleton = None
