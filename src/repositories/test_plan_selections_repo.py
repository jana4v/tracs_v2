from __future__ import annotations

import json
import sqlite3
import threading
from pathlib import Path


class TestPlanSelectionsRepository:
    """
    Stores per-Test-Plan-Type checkbox selections for transmitter / receiver /
    transponder rows. Each entry is keyed by (system_kind, test_plan_name) and
    stores a JSON list of selection rows.

    Selection row shape (transmitter / receiver):
        { "code": str, "port": str, "frequency_label": str,
          "params": { "<param_name>": bool, ... } }

    Selection row shape (transponder):
        { "transponder_code": str, "uplink": str, "downlink": str,
          "params": { "ranging_threshold": bool } }
    """

    VALID_KINDS = ("transmitter", "receiver", "transponder")

    def __init__(self, db_path: str, table_name: str = "TestPlanSelections") -> None:
        self._path = Path(db_path)
        if not self._path.is_absolute():
            self._path = Path.cwd() / self._path
        self._path.parent.mkdir(parents=True, exist_ok=True)
        self._table_name = table_name
        self._lock = threading.Lock()
        self._conn = sqlite3.connect(str(self._path), check_same_thread=False)
        self._conn.row_factory = sqlite3.Row
        self._ensure_schema()

    def _ensure_schema(self) -> None:
        with self._lock:
            self._conn.execute(
                f"""
                CREATE TABLE IF NOT EXISTS {self._table_name} (
                    system_kind TEXT NOT NULL,
                    test_plan_name TEXT NOT NULL,
                    data TEXT NOT NULL DEFAULT '[]',
                    PRIMARY KEY (system_kind, test_plan_name)
                )
                """
            )
            self._conn.commit()

    @staticmethod
    def _validate_kind(system_kind: str) -> str:
        kind = (system_kind or "").strip().lower()
        if kind not in TestPlanSelectionsRepository.VALID_KINDS:
            raise ValueError(f"Invalid system_kind: {system_kind}")
        return kind

    def get_selections(self, system_kind: str, test_plan_name: str) -> list[dict]:
        kind = self._validate_kind(system_kind)
        name = (test_plan_name or "").strip()
        with self._lock:
            cursor = self._conn.execute(
                f"SELECT data FROM {self._table_name} WHERE system_kind = ? AND test_plan_name = ?",
                (kind, name),
            )
            row = cursor.fetchone()
        if row is None:
            return []
        try:
            data = json.loads(row["data"])
            return data if isinstance(data, list) else []
        except (json.JSONDecodeError, TypeError):
            return []

    def save_selections(self, system_kind: str, test_plan_name: str, rows: list[dict]) -> None:
        kind = self._validate_kind(system_kind)
        name = (test_plan_name or "").strip()
        if not name:
            raise ValueError("test_plan_name is required")
        payload = json.dumps(rows or [])
        with self._lock:
            self._conn.execute(
                f"""
                INSERT INTO {self._table_name} (system_kind, test_plan_name, data)
                VALUES (?, ?, ?)
                ON CONFLICT(system_kind, test_plan_name) DO UPDATE SET data = excluded.data
                """,
                (kind, name, payload),
            )
            self._conn.commit()
