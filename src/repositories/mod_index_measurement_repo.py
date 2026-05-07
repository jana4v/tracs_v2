from __future__ import annotations

import sqlite3
import threading
from datetime import datetime
from pathlib import Path


class ModIndexMeasurementRepository:
    """Stores measured modulation-index values per (run, transmitter, port, frequency, tone).

    A "run id" is provided by the caller (typically a UTC timestamp string) so that
    repeated measurements at the same channel/tone are preserved as separate rows.
    """

    def __init__(self, db_path: str, table_name: str = "ModIndexMeasurement") -> None:
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
                    RunId TEXT NOT NULL,
                    SystemKind TEXT NOT NULL,
                    Code TEXT NOT NULL,
                    Port TEXT NOT NULL,
                    Frequency REAL NOT NULL,
                    FrequencyLabel TEXT NOT NULL DEFAULT '',
                    ToneKHz REAL NOT NULL,
                    ModIndex REAL NOT NULL,
                    SidebandUpper REAL NOT NULL DEFAULT 0,
                    SidebandLower REAL NOT NULL DEFAULT 0,
                    Samples INTEGER NOT NULL DEFAULT 0,
                    Status TEXT NOT NULL DEFAULT '',
                    DateTime TEXT NOT NULL,
                    PRIMARY KEY (RunId, Code, Port, Frequency, ToneKHz)
                )
                """
            )
            self._conn.commit()

    def upsert(
        self,
        run_id: str,
        system_kind: str,
        code: str,
        port: str,
        frequency: float,
        frequency_label: str,
        tone_khz: float,
        mod_index: float,
        sideband_upper: float,
        sideband_lower: float,
        samples: int,
        status: str,
        date_time: datetime,
    ) -> None:
        with self._lock:
            self._conn.execute(
                f"""
                INSERT INTO {self._table_name}
                    (RunId, SystemKind, Code, Port, Frequency, FrequencyLabel,
                     ToneKHz, ModIndex, SidebandUpper, SidebandLower,
                     Samples, Status, DateTime)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                ON CONFLICT(RunId, Code, Port, Frequency, ToneKHz) DO UPDATE SET
                    SystemKind = excluded.SystemKind,
                    FrequencyLabel = excluded.FrequencyLabel,
                    ModIndex = excluded.ModIndex,
                    SidebandUpper = excluded.SidebandUpper,
                    SidebandLower = excluded.SidebandLower,
                    Samples = excluded.Samples,
                    Status = excluded.Status,
                    DateTime = excluded.DateTime
                """,
                (
                    str(run_id or "").strip(),
                    str(system_kind or "").strip(),
                    str(code or "").strip(),
                    str(port or "").strip(),
                    float(frequency),
                    str(frequency_label or "").strip(),
                    float(tone_khz),
                    float(mod_index),
                    float(sideband_upper),
                    float(sideband_lower),
                    int(samples),
                    str(status or "").strip(),
                    date_time.isoformat(),
                ),
            )
            self._conn.commit()

    def list_rows(
        self,
        run_id: str | None = None,
        code: str | None = None,
        port: str | None = None,
    ) -> list[dict[str, str | float | int]]:
        clauses: list[str] = []
        params: list[object] = []
        if run_id is not None:
            clauses.append("RunId = ?")
            params.append(str(run_id).strip())
        if code is not None:
            clauses.append("Code = ?")
            params.append(str(code).strip())
        if port is not None:
            clauses.append("Port = ?")
            params.append(str(port).strip())
        where = ("WHERE " + " AND ".join(clauses)) if clauses else ""
        with self._lock:
            cursor = self._conn.execute(
                f"""
                SELECT RunId, SystemKind, Code, Port, Frequency, FrequencyLabel,
                       ToneKHz, ModIndex, SidebandUpper, SidebandLower,
                       Samples, Status, DateTime
                FROM {self._table_name}
                {where}
                ORDER BY DateTime DESC, Code, Port, Frequency, ToneKHz
                """,
                params,
            )
            rows = cursor.fetchall()
        return [
            {
                "run_id": str(r["RunId"]),
                "system_kind": str(r["SystemKind"]),
                "code": str(r["Code"]),
                "port": str(r["Port"]),
                "frequency": float(r["Frequency"]),
                "frequency_label": str(r["FrequencyLabel"]),
                "tone_khz": float(r["ToneKHz"]),
                "mod_index": float(r["ModIndex"]),
                "sideband_upper": float(r["SidebandUpper"]),
                "sideband_lower": float(r["SidebandLower"]),
                "samples": int(r["Samples"]),
                "status": str(r["Status"]),
                "datetime": str(r["DateTime"]),
            }
            for r in rows
        ]

    def delete_for_code(self, code: str, system_kind: str | None = None) -> int:
        """Delete every measurement row tied to the given system code.

        Used to cascade transmitter/receiver/transponder deletions so
        ModIndexMeasurement rows do not become orphans (the system document
        is the source of truth for code/port/frequency).

        If `system_kind` is provided (e.g. "transmitter", "receiver",
        "transponder"), the delete is restricted to that kind so codes that
        happen to coincide across kinds are not over-deleted.
        """
        clauses = ["Code = ?"]
        params: list[object] = [str(code or "").strip()]
        if system_kind is not None:
            clauses.append("SystemKind = ?")
            params.append(str(system_kind or "").strip().lower())
        with self._lock:
            cursor = self._conn.execute(
                f"DELETE FROM {self._table_name} WHERE {' AND '.join(clauses)}",
                params,
            )
            self._conn.commit()
            return cursor.rowcount or 0
