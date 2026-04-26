from __future__ import annotations

import sqlite3
import threading
from pathlib import Path


REQUIRED_PARAMETERS = ["SATELLITE_NAME", "RESULTS_DIRECTORY"]


class EnvDataRepository:
    def __init__(self, db_path: str, table_name: str = "EnvData") -> None:
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
                    Parameter TEXT NOT NULL PRIMARY KEY,
                    Value TEXT NOT NULL
                )
                """
            )

            cols = [
                str(row["name"]).lower()
                for row in self._conn.execute(f"PRAGMA table_info({self._table_name})").fetchall()
            ]

            if "id" in cols:
                tmp_table = f"{self._table_name}_tmp"
                self._conn.execute(
                    f"""
                    CREATE TABLE IF NOT EXISTS {tmp_table} (
                        Parameter TEXT NOT NULL PRIMARY KEY,
                        Value TEXT NOT NULL
                    )
                    """
                )

                legacy_rows = self._conn.execute(
                    f"SELECT Parameter, Value FROM {self._table_name} ORDER BY Id"
                ).fetchall()
                for row in legacy_rows:
                    parameter_text = str(row["Parameter"] or "").strip()
                    if parameter_text == "":
                        continue
                    self._conn.execute(
                        f"INSERT OR REPLACE INTO {tmp_table} (Parameter, Value) VALUES (?, ?)",
                        (parameter_text, str(row["Value"] or "")),
                    )

                self._conn.execute(f"DROP TABLE {self._table_name}")
                self._conn.execute(f"ALTER TABLE {tmp_table} RENAME TO {self._table_name}")

            for parameter in REQUIRED_PARAMETERS:
                self._conn.execute(
                    f"INSERT OR IGNORE INTO {self._table_name} (Parameter, Value) VALUES (?, ?)",
                    (parameter, ""),
                )

            self._conn.commit()

    def list_rows(self) -> list[dict[str, str]]:
        with self._lock:
            cursor = self._conn.execute(
                f"SELECT Parameter, Value FROM {self._table_name}"
            )
            rows = cursor.fetchall()

        values_by_parameter: dict[str, str] = {
            str(row["Parameter"]): str(row["Value"])
            for row in rows
        }

        ordered = REQUIRED_PARAMETERS + sorted(
            [p for p in values_by_parameter.keys() if p not in REQUIRED_PARAMETERS],
            key=str.lower,
        )
        return [
            {
                "parameter": parameter,
                "value": values_by_parameter.get(parameter, ""),
            }
            for parameter in ordered
        ]

    def upsert_row(self, parameter: str, value: str) -> dict[str, str]:
        parameter_text = str(parameter).strip()
        value_text = str(value)
        if parameter_text == "":
            raise ValueError("Parameter is required")

        with self._lock:
            self._conn.execute(
                f"INSERT OR REPLACE INTO {self._table_name} (Parameter, Value) VALUES (?, ?)",
                (parameter_text, value_text),
            )
            self._conn.commit()

        return {"parameter": parameter_text, "value": value_text}

    def delete_row(self, parameter: str) -> bool:
        parameter_text = str(parameter).strip()
        if parameter_text in REQUIRED_PARAMETERS:
            return False
        with self._lock:
            cursor = self._conn.execute(
                f"DELETE FROM {self._table_name} WHERE Parameter = ?",
                (parameter_text,),
            )
            self._conn.commit()
            return cursor.rowcount > 0

    def replace_rows(self, rows: list[dict[str, str]]) -> int:
        values_by_parameter: dict[str, str] = {}
        for row in rows:
            parameter_text = str(row.get("parameter", "")).strip()
            if parameter_text == "":
                continue
            values_by_parameter[parameter_text] = str(row.get("value", ""))

        for parameter in REQUIRED_PARAMETERS:
            values_by_parameter.setdefault(parameter, "")

        with self._lock:
            self._conn.execute(f"DELETE FROM {self._table_name}")

            for parameter, value in values_by_parameter.items():
                self._conn.execute(
                    f"INSERT INTO {self._table_name} (Parameter, Value) VALUES (?, ?)",
                    (parameter, value),
                )

            self._conn.commit()

        return len(values_by_parameter)
