from __future__ import annotations

import sqlite3
import threading
from datetime import datetime
from pathlib import Path


class DownlinkCalCalibrationRepository:
    def __init__(self, db_path: str, table_name: str = "DownlinkCalCalibrationData") -> None:
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
                    CalId TEXT NOT NULL,
                    Code TEXT NOT NULL,
                    Port TEXT NOT NULL,
                    Frequency REAL NOT NULL,
                    FrequencyLabel TEXT NOT NULL DEFAULT '',
                    Value REAL NOT NULL,
                    DateTime TEXT NOT NULL,
                    PRIMARY KEY (CalId, Code, Port, Frequency)
                )
                """
            )
            self._migrate_schema_if_needed()
            self._conn.commit()

    def _migrate_schema_if_needed(self) -> None:
        info_rows = self._conn.execute(f"PRAGMA table_info({self._table_name})").fetchall()
        column_names = {str(row[1]) for row in info_rows}
        if "FrequencyLabel" not in column_names:
            self._conn.execute(
                f"ALTER TABLE {self._table_name} ADD COLUMN FrequencyLabel TEXT NOT NULL DEFAULT ''"
            )

    def upsert(
        self,
        cal_id: str,
        code: str,
        port: str,
        frequency: float,
        frequency_label: str,
        value: float,
        date_time: datetime,
    ) -> None:
        with self._lock:
            self._conn.execute(
                f"""
                INSERT INTO {self._table_name} (CalId, Code, Port, Frequency, FrequencyLabel, Value, DateTime)
                VALUES (?, ?, ?, ?, ?, ?, ?)
                ON CONFLICT(CalId, Code, Port, Frequency) DO UPDATE SET
                    FrequencyLabel = excluded.FrequencyLabel,
                    Value = excluded.Value,
                    DateTime = excluded.DateTime
                """,
                (
                    str(cal_id or "").strip(),
                    str(code or "").strip(),
                    str(port or "").strip(),
                    float(frequency),
                    str(frequency_label or "").strip(),
                    float(value),
                    date_time.isoformat(),
                ),
            )
            self._conn.commit()

    def list_rows(self, cal_id: str | None = None) -> list[dict[str, str | float]]:
        with self._lock:
            if cal_id is None:
                cursor = self._conn.execute(
                    f"SELECT CalId, Code, Port, Frequency, FrequencyLabel, Value, DateTime FROM {self._table_name} "
                    f"ORDER BY Code, Port, Frequency"
                )
            else:
                cursor = self._conn.execute(
                    f"SELECT CalId, Code, Port, Frequency, FrequencyLabel, Value, DateTime FROM {self._table_name} "
                    f"WHERE CalId = ? ORDER BY Code, Port, Frequency",
                    (str(cal_id or "").strip(),),
                )
            rows = cursor.fetchall()

        return [
            {
                "cal_id": str(row["CalId"]),
                "code": str(row["Code"]),
                "port": str(row["Port"]),
                "frequency": float(row["Frequency"]),
                "frequency_label": str(row["FrequencyLabel"]),
                "value": float(row["Value"]),
                "datetime": str(row["DateTime"]),
            }
            for row in rows
        ]

    def list_calibrated_keys(self, cal_id: str) -> list[dict[str, str | float]]:
        """Return (code, port, frequency) tuples already calibrated for the given cal_id."""
        with self._lock:
            cursor = self._conn.execute(
                f"SELECT Code, Port, Frequency FROM {self._table_name} WHERE CalId = ?",
                (str(cal_id or "").strip(),),
            )
            rows = cursor.fetchall()
        return [
            {"code": str(r["Code"]), "port": str(r["Port"]), "frequency": float(r["Frequency"])}
            for r in rows
        ]

    def list_cal_ids(self) -> list[str]:
        """Return distinct CalId values ordered by latest DateTime descending."""
        with self._lock:
            cursor = self._conn.execute(
                f"""
                SELECT CalId, MAX(DateTime) AS LatestDateTime
                FROM {self._table_name}
                GROUP BY CalId
                ORDER BY LatestDateTime DESC, CalId DESC
                """
            )
            rows = cursor.fetchall()
        return [str(r["CalId"]) for r in rows if str(r["CalId"] or "").strip() != ""]
