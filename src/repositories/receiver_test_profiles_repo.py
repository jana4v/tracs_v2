from __future__ import annotations

import json
import sqlite3
import threading
from pathlib import Path


class ReceiverTestProfilesRepository:
    def __init__(self, db_path: str, table_name: str = "ReceiverTestProfiles") -> None:
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
                    name TEXT NOT NULL PRIMARY KEY,
                    data TEXT NOT NULL DEFAULT '[]'
                )
                """
            )
            self._conn.commit()

    def get_profile(self, name: str) -> list[dict]:
        with self._lock:
            cursor = self._conn.execute(
                f"SELECT data FROM {self._table_name} WHERE name = ?",
                (name,),
            )
            row = cursor.fetchone()

        if row is None:
            return []
        try:
            return json.loads(row["data"]) or []
        except (json.JSONDecodeError, TypeError):
            return []

    def save_profile(self, name: str, rows: list[dict]) -> None:
        with self._lock:
            self._conn.execute(
                f"""
                INSERT INTO {self._table_name} (name, data)
                VALUES (?, ?)
                ON CONFLICT(name) DO UPDATE SET data = excluded.data
                """,
                (name, json.dumps(rows)),
            )
            self._conn.commit()

    def list_names(self) -> list[str]:
        with self._lock:
            cursor = self._conn.execute(
                f"SELECT name FROM {self._table_name} ORDER BY name"
            )
            return [str(row["name"]) for row in cursor.fetchall()]
