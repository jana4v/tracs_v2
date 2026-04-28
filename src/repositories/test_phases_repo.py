from __future__ import annotations

import sqlite3
import threading
from pathlib import Path


class TestPhasesRepository:
    def __init__(self, db_path: str, table_name: str = "TestPhases") -> None:
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
                    TEST_PHASE TEXT NOT NULL UNIQUE,
                    TEST_PHASE_NO INTEGER NOT NULL PRIMARY KEY,
                    STD_TEST_PHASE TEXT NOT NULL
                )
                """
            )
            self._seed_defaults()
            self._conn.commit()

    def _seed_defaults(self) -> None:
        rows = [
            ("Verification", 100, "Quick IST"),
            ("Pre T&E", 150, "Assembled IST"),
            ("Test & Evaluation", 200, "Quick IST"),
            ("Dis-Assembled IST", 250, "Dis-Assembled IST"),
            ("Assembled IST", 300, "Assembled IST"),
            ("Pre-TVAC IST", 350, "Pre-TVAC IST"),
            ("Minor Cold-1", 400, "TVAC Minor"),
            ("Minor Hot-1", 450, "TVAC Minor"),
            ("Minor Cold-2", 500, "TVAC Minor"),
            ("Minor Hot-2", 550, "TVAC Minor"),
            ("Cold Soak", 600, "TVAC Soaks"),
            ("Hot Soak", 650, "TVAC Soaks"),
            ("Terminating Minor Cold", 700, "TVAC Minor"),
            ("Terminating Minor Hot", 750, "TVAC Minor"),
            ("Vacuum Ambient", 800, "TVAC Vacuum Ambient"),
            ("Post TVAC", 850, "POST TVAC"),
            ("Clean Room IST", 900, "Quick IST"),
            ("EMI-EMC", 950, "Quick IST"),
            ("Vibration", 1000, "Quick IST"),
            ("Accoustics", 1050, "Quick IST"),
            ("Clean Room Radiation IST", 1100, "Quick IST"),
            ("CATF", 1150, "CATF"),
            ("Pre-Launch IST", 1200, "Quick IST"),
            ("In-Orbit Testing", 1250, "Quick IST"),
            ("Post-TVAC IST CAL VERFI", 1300, "Quick IST"),
            ("Clean Room Post TVAC", 1350, "Quick IST"),
            ("Integration", 1450, "Quick IST"),
            ("PreDyn IST", 1500, "Quick IST"),
            ("POST-REWORK-CLEANROOM", 1550, "Quick IST"),
            ("Deployment", 1600, "Quick IST"),
            ("Post Dynamic", 1650, "Quick IST"),
        ]

        self._conn.executemany(
            f"""
            INSERT INTO {self._table_name} (TEST_PHASE, TEST_PHASE_NO, STD_TEST_PHASE)
            VALUES (?, ?, ?)
            ON CONFLICT(TEST_PHASE_NO) DO UPDATE SET
                TEST_PHASE = excluded.TEST_PHASE,
                STD_TEST_PHASE = excluded.STD_TEST_PHASE
            """,
            rows,
        )

    def list_rows(self) -> list[dict[str, str | int]]:
        with self._lock:
            cursor = self._conn.execute(
                f"SELECT TEST_PHASE, TEST_PHASE_NO, STD_TEST_PHASE FROM {self._table_name} ORDER BY TEST_PHASE_NO"
            )
            rows = cursor.fetchall()

        return [
            {
                "TEST_PHASE": str(row["TEST_PHASE"]),
                "TEST_PHASE_NO": int(row["TEST_PHASE_NO"]),
                "STD_TEST_PHASE": str(row["STD_TEST_PHASE"]),
            }
            for row in rows
        ]
