from __future__ import annotations

import sqlite3
import threading
from pathlib import Path


class TestPlanTypesRepository:
    def __init__(self, db_path: str, table_name: str = "TestPlanType") -> None:
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
                    TEST_PLAN_TYPE_NO INTEGER NOT NULL PRIMARY KEY,
                    TEST_PLAN_TYPE TEXT NOT NULL UNIQUE
                )
                """
            )
            self._seed_defaults()
            self._conn.commit()

    def _seed_defaults(self) -> None:
        rows = [
            (100, "Detailed"),
            (200, "Short"),
            (300, "Go/No-Go"),
        ]

        self._conn.executemany(
            f"""
            INSERT INTO {self._table_name} (TEST_PLAN_TYPE_NO, TEST_PLAN_TYPE)
            VALUES (?, ?)
            ON CONFLICT(TEST_PLAN_TYPE_NO) DO UPDATE SET
                TEST_PLAN_TYPE = excluded.TEST_PLAN_TYPE
            """,
            rows,
        )

    def list_rows(self) -> list[dict[str, str | int]]:
        with self._lock:
            cursor = self._conn.execute(
                f"SELECT TEST_PLAN_TYPE_NO, TEST_PLAN_TYPE FROM {self._table_name} ORDER BY TEST_PLAN_TYPE_NO"
            )
            rows = cursor.fetchall()

        return [
            {
                "TEST_PLAN_TYPE_NO": int(row["TEST_PLAN_TYPE_NO"]),
                "TEST_PLAN_TYPE": str(row["TEST_PLAN_TYPE"]),
            }
            for row in rows
        ]

    def list_types(self) -> list[str]:
        return [
            str(row.get("TEST_PLAN_TYPE") or "").strip()
            for row in self.list_rows()
            if str(row.get("TEST_PLAN_TYPE") or "").strip() != ""
        ]