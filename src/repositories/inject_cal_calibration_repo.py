from __future__ import annotations

import sqlite3
import threading
from datetime import datetime
from pathlib import Path


class InjectCalCalibrationRepository:
    def __init__(self, db_path: str, table_name: str = "InjectCalCalibrationData") -> None:
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
                    Frequency REAL NOT NULL,
                    SA_Loss REAL NOT NULL,
                    DL_PM_Loss REAL NOT NULL,
                    DateTime TEXT NOT NULL,
                    PRIMARY KEY (CalId, Frequency)
                )
                """
            )
            self._migrate_schema_if_needed()
            self._conn.commit()

    def _migrate_schema_if_needed(self) -> None:
        info_rows = self._conn.execute(f"PRAGMA table_info({self._table_name})").fetchall()
        column_names = {str(row[1]) for row in info_rows}
        composite_pk = {str(row[1]) for row in info_rows if int(row[5] or 0) > 0}

        has_new_columns = {"CalId", "Frequency", "SA_Loss", "DL_PM_Loss", "DateTime"}.issubset(column_names)
        if has_new_columns and composite_pk == {"CalId", "Frequency"}:
            return

        has_old_value = "Value" in column_names

        temp_table = f"{self._table_name}__new"
        self._conn.execute(
            f"""
            CREATE TABLE IF NOT EXISTS {temp_table} (
                CalId TEXT NOT NULL,
                Frequency REAL NOT NULL,
                SA_Loss REAL NOT NULL,
                DL_PM_Loss REAL NOT NULL,
                DateTime TEXT NOT NULL,
                PRIMARY KEY (CalId, Frequency)
            )
            """
        )

        if has_old_value:
            self._conn.execute(
                f"""
                INSERT OR REPLACE INTO {temp_table} (CalId, Frequency, SA_Loss, DL_PM_Loss, DateTime)
                SELECT COALESCE(CalId, ''), Frequency, Value, Value, DateTime FROM {self._table_name}
                """
            )
        else:
            self._conn.execute(
                f"""
                INSERT OR REPLACE INTO {temp_table} (CalId, Frequency, SA_Loss, DL_PM_Loss, DateTime)
                SELECT COALESCE(CalId, ''), Frequency,
                       COALESCE(SA_Loss, 0), COALESCE(DL_PM_Loss, 0), DateTime
                FROM {self._table_name}
                """
            )

        self._conn.execute(f"DROP TABLE {self._table_name}")
        self._conn.execute(f"ALTER TABLE {temp_table} RENAME TO {self._table_name}")

    def upsert(self, cal_id: str, frequency: float, sa_loss: float, dl_pm_loss: float, date_time: datetime) -> None:
        with self._lock:
            self._conn.execute(
                f"""
                INSERT INTO {self._table_name} (CalId, Frequency, SA_Loss, DL_PM_Loss, DateTime)
                VALUES (?, ?, ?, ?, ?)
                ON CONFLICT(CalId, Frequency) DO UPDATE SET
                    SA_Loss = excluded.SA_Loss,
                    DL_PM_Loss = excluded.DL_PM_Loss,
                    DateTime = excluded.DateTime
                """,
                (
                    str(cal_id or "").strip(),
                    float(frequency),
                    abs(float(sa_loss)),
                    abs(float(dl_pm_loss)),
                    date_time.isoformat(),
                ),
            )
            self._conn.commit()

    def list_rows(self, cal_id: str | None = None) -> list[dict[str, str | float]]:
        with self._lock:
            if cal_id is None:
                cursor = self._conn.execute(
                    f"SELECT CalId, Frequency, SA_Loss, DL_PM_Loss, DateTime FROM {self._table_name} ORDER BY Frequency"
                )
            else:
                cursor = self._conn.execute(
                    f"SELECT CalId, Frequency, SA_Loss, DL_PM_Loss, DateTime FROM {self._table_name} WHERE CalId = ? ORDER BY Frequency",
                    (str(cal_id or "").strip(),),
                )
            rows = cursor.fetchall()

        return [
            {
                "cal_id": str(row["CalId"]),
                "frequency": float(row["Frequency"]),
                "sa_loss": abs(float(row["SA_Loss"])),
                "dl_pm_loss": abs(float(row["DL_PM_Loss"])),
                # Keep value for backward-compatible consumers expecting a single loss field.
                "value": abs(float(row["SA_Loss"])),
                "datetime": str(row["DateTime"]),
            }
            for row in rows
        ]

    def list_frequencies(self, cal_id: str) -> list[float]:
        with self._lock:
            cursor = self._conn.execute(
                f"SELECT Frequency FROM {self._table_name} WHERE CalId = ? ORDER BY Frequency",
                (str(cal_id or "").strip(),),
            )
            rows = cursor.fetchall()
        return [float(row["Frequency"]) for row in rows]
