import copy
import json
import sqlite3
import threading
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Optional
from uuid import uuid4


@dataclass
class UpdateResult:
    modified_count: int = 0
    upserted_id: Optional[str] = None


@dataclass
class DeleteResult:
    deleted_count: int = 0


class SQLiteJsonCollection:
    def __init__(self, db_path: str, name: str) -> None:
        self._name = name
        self._is_transmitters = name == "transmitters"
        self._is_instrument_models = name == "Instruments"
        self._is_project_instruments = name == "ProjectInstruments"
        self._is_project_power_meters = name == "ProjectPowerMeters"
        self._is_project_tsm_paths = name == "ProjectTSMPaths"
        self._is_configuration = name == "Configuration"
        self._path = Path(db_path)
        if not self._path.is_absolute():
            self._path = Path.cwd() / self._path
        self._path.parent.mkdir(parents=True, exist_ok=True)
        self._lock = threading.Lock()
        self._conn = sqlite3.connect(str(self._path), check_same_thread=False)
        self._conn.row_factory = sqlite3.Row
        self._ensure_schema()

    def _ensure_schema(self) -> None:
        with self._lock:
            self._conn.execute(
                """
                CREATE TABLE IF NOT EXISTS json_documents (
                    collection TEXT NOT NULL,
                    doc_id TEXT NOT NULL,
                    data TEXT NOT NULL,
                    PRIMARY KEY (collection, doc_id)
                )
                """
            )
            if self._is_transmitters:
                self._conn.execute(
                    """
                    CREATE TABLE IF NOT EXISTS transmitters (
                        id TEXT NOT NULL PRIMARY KEY,
                        data TEXT NOT NULL
                    )
                    """
                )
                self._migrate_transmitters_collection()
            if self._is_instrument_models:
                self._conn.execute(
                    """
                    CREATE TABLE IF NOT EXISTS InstrumentModels (
                        id TEXT NOT NULL PRIMARY KEY,
                        models TEXT NOT NULL
                    )
                    """
                )
                self._migrate_instruments_collection()
                self._seed_default_instrument_models()
            if self._is_project_instruments:
                self._conn.execute(
                    """
                    CREATE TABLE IF NOT EXISTS instruments (
                        instrument_name TEXT NOT NULL PRIMARY KEY,
                        model TEXT NOT NULL,
                        address_main TEXT NOT NULL,
                        address_redt TEXT NOT NULL,
                        UseRedt INTEGER NOT NULL DEFAULT 0
                    )
                    """
                )
                self._ensure_project_instruments_schema()
                self._migrate_project_instruments_collection()
                self._seed_default_project_instruments()
            if self._is_project_power_meters:
                self._conn.execute(
                    """
                    CREATE TABLE IF NOT EXISTS PowerMeter (
                        PowerMeter TEXT NOT NULL PRIMARY KEY,
                        Channel TEXT NOT NULL
                    )
                    """
                )
                self._migrate_project_power_meters_collection()
                self._seed_default_project_power_meters()
            if self._is_project_tsm_paths:
                self._conn.execute(
                    """
                    CREATE TABLE IF NOT EXISTS TSMPaths (
                        Code TEXT NOT NULL,
                        Port TEXT NOT NULL,
                        Path1 TEXT,
                        Path2 TEXT,
                        Path3 TEXT,
                        Path4 TEXT,
                        Path5 TEXT,
                        Path6 TEXT,
                        PRIMARY KEY (Code, Port)
                    )
                    """
                )
                self._ensure_tsm_paths_schema()
                self._migrate_project_tsm_paths_collection()
            if self._is_configuration:
                self._conn.execute(
                    """
                    CREATE TABLE IF NOT EXISTS Configuration (
                        Parameter TEXT NOT NULL PRIMARY KEY,
                        Value TEXT NOT NULL
                    )
                    """
                )
                self._migrate_configuration_collection()
            self._conn.commit()

    def _migrate_transmitters_collection(self) -> None:
        rows = self._conn.execute(
            "SELECT data FROM json_documents WHERE collection = ?",
            (self._name,),
        ).fetchall()
        for row in rows:
            try:
                doc = json.loads(row["data"])
            except Exception:
                continue

            code = str(doc.get("code") or "").strip()
            if not code:
                continue

            payload = json.dumps(doc, ensure_ascii=True)
            self._conn.execute(
                """
                INSERT INTO transmitters (id, data)
                VALUES (?, ?)
                ON CONFLICT(id) DO UPDATE SET data = excluded.data
                """,
                (code, payload),
            )

    def _default_instrument_models(self) -> dict[str, list[str]]:
        return {
            "CalPowerMeter": ["E4419B"],
            "CalSignalGenerator": ["E8257D"],
            "DownConvertorLo1": ["BOHN"],
            "DownConvertorLo2": ["BOHN"],
            "DownlinkPowerMeter": ["E4419B"],
            "SpectrumAnalyser": ["N9030A", "N9030B"],
            "TSM1": ["SDU_SCG_GPIB", "SDU_SCG_LAN", "SCG_SDU", "DUCOM"],
            "TSM2": ["SDU_SCG_GPIB", "SDU_SCG_LAN", "SCG_SDU", "DUCOM"],
            "UpConvertorLo1": ["BOHN"],
            "UpConvertorLo2": ["BOHN"],
            "InjectSignalGenerator": ["E8257D"],
            "UplinkPowerMeter": ["E4419B"],
            "UplinkSignalGenerator": ["E8257D"],
        }

    def _upsert_instrument_models(self, instrument_type: str, models: list[Any]) -> None:
        key = str(instrument_type).strip()
        if not key:
            return

        clean_models = [str(model) for model in models if model is not None]
        self._conn.execute(
            """
            INSERT INTO InstrumentModels (id, models)
            VALUES (?, ?)
            ON CONFLICT(id) DO UPDATE SET models = excluded.models
            """,
            (key, json.dumps(clean_models, ensure_ascii=True)),
        )

    def _migrate_instruments_collection(self) -> None:
        existing_rows = self._conn.execute("SELECT COUNT(1) AS c FROM InstrumentModels").fetchone()
        if existing_rows and int(existing_rows["c"] or 0) > 0:
            return

        rows = self._conn.execute(
            "SELECT data FROM json_documents WHERE collection = ?",
            (self._name,),
        ).fetchall()
        for row in rows:
            try:
                doc = json.loads(row["data"])
            except Exception:
                continue

            if not isinstance(doc, dict):
                continue

            for key, value in doc.items():
                if key == "_id" or not isinstance(value, list):
                    continue
                self._upsert_instrument_models(str(key), value)

    def _seed_default_instrument_models(self) -> None:
        row = self._conn.execute("SELECT COUNT(1) AS c FROM InstrumentModels").fetchone()
        if row and int(row["c"] or 0) > 0:
            self._migrate_tsm_models()
            return

        for instrument_type, models in self._default_instrument_models().items():
            self._upsert_instrument_models(instrument_type, models)

    def _migrate_tsm_models(self) -> None:
        """Ensure TSM entries in existing databases include the GPIB/LAN switch driver models."""
        required = {"SDU_SCG_GPIB", "SDU_SCG_LAN"}
        for tsm_key in ("TSM1", "TSM2"):
            row = self._conn.execute(
                "SELECT models FROM InstrumentModels WHERE id = ?", (tsm_key,)
            ).fetchone()
            if row is None:
                self._upsert_instrument_models(tsm_key, self._default_instrument_models()[tsm_key])
                continue
            try:
                existing: list[str] = json.loads(row["models"])
            except Exception:
                existing = []
            if not isinstance(existing, list):
                existing = []
            existing_set = set(existing)
            missing = required - existing_set
            if missing:
                updated = list(missing) + existing
                self._upsert_instrument_models(tsm_key, updated)

    def _default_project_instruments(self) -> list[dict[str, str]]:
        return [
            {
                "instrument_name": "UplinkSignalGenerator",
                "model": "E8257D",
                "address_main": "10.22.14.1",
                "address_redt": "10.22.14.2",
                "use_redt": False,
            },
            {
                "instrument_name": "CalSignalGenerator",
                "model": "E8257D",
                "address_main": "10.22.14.1",
                "address_redt": "10.22.14.2",
                "use_redt": False,
            },
            {
                "instrument_name": "InjectSignalGenerator",
                "model": "E8257D",
                "address_main": "10.22.14.1",
                "address_redt": "10.22.14.2",
                "use_redt": False,
            },
            {
                "instrument_name": "UplinkPowerMeter",
                "model": "E4419B",
                "address_main": "10.22.14.1",
                "address_redt": "10.22.14.2",
                "use_redt": False,
            },
            {
                "instrument_name": "DownlinkPowerMeter",
                "model": "E4419B",
                "address_main": "10.22.14.1",
                "address_redt": "10.22.14.2",
                "use_redt": False,
            },
            {
                "instrument_name": "CalPowerMeter",
                "model": "E4419B",
                "address_main": "10.22.14.1",
                "address_redt": "10.22.14.2",
                "use_redt": False,
            },
            {
                "instrument_name": "SpectrumAnalyser",
                "model": "N9030A",
                "address_main": "10.22.14.1",
                "address_redt": "10.22.14.2",
                "use_redt": False,
            },
            {
                "instrument_name": "TSM1",
                "model": "SCG_SDU",
                "address_main": "10.22.14.1",
                "address_redt": "10.22.14.2",
                "use_redt": False,
            },
            {
                "instrument_name": "TSM2",
                "model": "SCG_SDU",
                "address_main": "10.22.14.1",
                "address_redt": "10.22.14.2",
                "use_redt": False,
            },
            {
                "instrument_name": "UpConvertorLo1",
                "model": "BOHN",
                "address_main": "10.22.14.1",
                "address_redt": "10.22.14.2",
                "use_redt": False,
            },
            {
                "instrument_name": "UpConvertorLo2",
                "model": "BOHN",
                "address_main": "10.22.14.1",
                "address_redt": "10.22.14.2",
                "use_redt": False,
            },
            {
                "instrument_name": "DownConvertorLo1",
                "model": "BOHN",
                "address_main": "10.22.14.1",
                "address_redt": "10.22.14.2",
                "use_redt": False,
            },
            {
                "instrument_name": "DownConvertorLo2",
                "model": "BOHN",
                "address_main": "10.22.14.1",
                "address_redt": "10.22.14.2",
                "use_redt": False,
            },
        ]

    def _ensure_project_instruments_schema(self) -> None:
        rows = self._conn.execute("PRAGMA table_info(instruments)").fetchall()
        column_names = {str(row[1]) for row in rows}
        if "UseRedt" not in column_names:
            self._conn.execute("ALTER TABLE instruments ADD COLUMN UseRedt INTEGER NOT NULL DEFAULT 0")

    def _normalize_project_instrument_row(self, row: dict[str, Any]) -> Optional[dict[str, str]]:
        instrument_name = str(row.get("instrument_name") or "").strip()
        if not instrument_name:
            return None

        use_redt_value = row.get("use_redt", row.get("UseRedt", False))
        use_redt = False
        if isinstance(use_redt_value, bool):
            use_redt = use_redt_value
        elif isinstance(use_redt_value, (int, float)):
            use_redt = int(use_redt_value) != 0
        else:
            use_redt = str(use_redt_value).strip().lower() in {"1", "true", "yes", "on"}

        return {
            "instrument_name": instrument_name,
            "model": str(row.get("model") or "").strip(),
            "address_main": str(row.get("address_main") or "").strip(),
            "address_redt": str(row.get("address_redt") or "").strip(),
            "use_redt": use_redt,
        }

    def _upsert_project_instrument(self, row: dict[str, Any]) -> None:
        normalized = self._normalize_project_instrument_row(row)
        if normalized is None:
            return
        self._conn.execute(
            """
            INSERT INTO instruments (instrument_name, model, address_main, address_redt, UseRedt)
            VALUES (?, ?, ?, ?, ?)
            ON CONFLICT(instrument_name) DO UPDATE SET
                model = excluded.model,
                address_main = excluded.address_main,
                address_redt = excluded.address_redt,
                UseRedt = excluded.UseRedt
            """,
            (
                normalized["instrument_name"],
                normalized["model"],
                normalized["address_main"],
                normalized["address_redt"],
                1 if bool(normalized["use_redt"]) else 0,
            ),
        )

    def _migrate_project_instruments_collection(self) -> None:
        existing_rows = self._conn.execute("SELECT COUNT(1) AS c FROM instruments").fetchone()
        if existing_rows and int(existing_rows["c"] or 0) > 0:
            return

        rows = self._conn.execute(
            "SELECT data FROM json_documents WHERE collection = ?",
            (self._name,),
        ).fetchall()
        for row in rows:
            try:
                doc = json.loads(row["data"])
            except Exception:
                continue

            if not isinstance(doc, dict):
                continue

            payload_rows = doc.get("rows")
            if isinstance(payload_rows, list):
                for item in payload_rows:
                    if isinstance(item, dict):
                        self._upsert_project_instrument(item)

    def _seed_default_project_instruments(self) -> None:
        row = self._conn.execute("SELECT COUNT(1) AS c FROM instruments").fetchone()
        if row and int(row["c"] or 0) > 0:
            return

        for item in self._default_project_instruments():
            self._upsert_project_instrument(item)

    def _default_project_power_meters(self) -> list[dict[str, str]]:
        return [
            {"PowerMeter": "CalPowerMeter", "Channel": "A"},
            {"PowerMeter": "DownlinkPowerMeter", "Channel": "B"},
            {"PowerMeter": "UplinkPowerMeter", "Channel": "A"},
        ]

    def _normalize_project_power_meter_row(self, row: dict[str, Any]) -> Optional[dict[str, str]]:
        power_meter = str(row.get("PowerMeter") or row.get("powerMeter") or "").strip()
        if power_meter == "":
            return None

        channel = str(row.get("Channel") or row.get("channel") or "A").strip().upper()
        if channel != "B":
            channel = "A"

        return {"PowerMeter": power_meter, "Channel": channel}

    def _upsert_project_power_meter(self, row: dict[str, Any]) -> None:
        normalized = self._normalize_project_power_meter_row(row)
        if normalized is None:
            return
        self._conn.execute(
            """
            INSERT INTO PowerMeter (PowerMeter, Channel)
            VALUES (?, ?)
            ON CONFLICT(PowerMeter) DO UPDATE SET
                Channel = excluded.Channel
            """,
            (normalized["PowerMeter"], normalized["Channel"]),
        )

    def _migrate_project_power_meters_collection(self) -> None:
        existing_rows = self._conn.execute("SELECT COUNT(1) AS c FROM PowerMeter").fetchone()
        if existing_rows and int(existing_rows["c"] or 0) > 0:
            return

        rows = self._conn.execute(
            "SELECT data FROM json_documents WHERE collection = ?",
            (self._name,),
        ).fetchall()
        for row in rows:
            try:
                doc = json.loads(row["data"])
            except Exception:
                continue

            if not isinstance(doc, dict):
                continue

            payload_rows = doc.get("rows")
            if isinstance(payload_rows, list):
                for item in payload_rows:
                    if isinstance(item, dict):
                        self._upsert_project_power_meter(item)

    def _seed_default_project_power_meters(self) -> None:
        row = self._conn.execute("SELECT COUNT(1) AS c FROM PowerMeter").fetchone()
        if row and int(row["c"] or 0) > 0:
            return

        for item in self._default_project_power_meters():
            self._upsert_project_power_meter(item)

    def _normalize_configuration_row(self, row: dict[str, Any]) -> Optional[dict[str, str]]:
        parameter = str(row.get("Parameter") or row.get("parameter") or "").strip()
        if parameter == "":
            return None
        return {
            "Parameter": parameter,
            "Value": str(row.get("Value") or row.get("value") or "").strip(),
        }

    def _upsert_configuration(self, row: dict[str, Any]) -> None:
        normalized = self._normalize_configuration_row(row)
        if normalized is None:
            return
        self._conn.execute(
            """
            INSERT INTO Configuration (Parameter, Value)
            VALUES (?, ?)
            ON CONFLICT(Parameter) DO UPDATE SET
                Value = excluded.Value
            """,
            (normalized["Parameter"], normalized["Value"]),
        )

    def _migrate_configuration_collection(self) -> None:
        existing_rows = self._conn.execute("SELECT COUNT(1) AS c FROM Configuration").fetchone()
        if existing_rows and int(existing_rows["c"] or 0) > 0:
            return

        rows = self._conn.execute(
            "SELECT data FROM json_documents WHERE collection = ?",
            (self._name,),
        ).fetchall()
        for row in rows:
            try:
                doc = json.loads(row["data"])
            except Exception:
                continue

            if not isinstance(doc, dict):
                continue

            payload_rows = doc.get("rows")
            if isinstance(payload_rows, list):
                for item in payload_rows:
                    if isinstance(item, dict):
                        self._upsert_configuration(item)
            else:
                self._upsert_configuration(doc)

    def _ensure_tsm_paths_schema(self) -> None:
        rows = self._conn.execute("PRAGMA table_info(TSMPaths)").fetchall()
        if len(rows) == 0:
            return

        path_columns = {"Path1", "Path2", "Path3", "Path4", "Path5", "Path6"}
        column_names = {str(row[1]) for row in rows}
        needs_rebuild = False
        for row in rows:
            name = str(row[1])
            not_null = int(row[3] or 0)
            if name in path_columns and not_null == 1:
                needs_rebuild = True
                break

        if "TSM" in column_names:
            needs_rebuild = True

        if not needs_rebuild:
            return

        self._conn.execute("ALTER TABLE TSMPaths RENAME TO TSMPaths_old")
        self._conn.execute(
            """
            CREATE TABLE TSMPaths (
                Code TEXT NOT NULL,
                Port TEXT NOT NULL,
                Path1 TEXT,
                Path2 TEXT,
                Path3 TEXT,
                Path4 TEXT,
                Path5 TEXT,
                Path6 TEXT,
                PRIMARY KEY (Code, Port)
            )
            """
        )
        self._conn.execute(
            """
            INSERT INTO TSMPaths (Code, Port, Path1, Path2, Path3, Path4, Path5, Path6)
            SELECT Code, Port, Path1, Path2, Path3, Path4, Path5, Path6
            FROM TSMPaths_old
            """
        )
        self._conn.execute("DROP TABLE TSMPaths_old")

    def _normalize_project_tsm_path_row(self, row: dict[str, Any]) -> Optional[dict[str, Any]]:
        code = str(row.get("Code") or row.get("code") or "").strip()
        port = str(row.get("Port") or row.get("port") or "").strip()
        if code == "":
            return None

        def normalize_path_value(*keys: str) -> Any:
            sentinel = object()
            value: Any = sentinel
            for key in keys:
                if key in row:
                    value = row[key]
                    break
            if value is sentinel:
                return None
            if value is None:
                return None
            return str(value).strip()

        return {
            "Code": code,
            "Port": port,
            "Path1": normalize_path_value("Path1", "path1"),
            "Path2": normalize_path_value("Path2", "path2"),
            "Path3": normalize_path_value("Path3", "path3"),
            "Path4": normalize_path_value("Path4", "path4"),
            "Path5": normalize_path_value("Path5", "path5"),
            "Path6": normalize_path_value("Path6", "path6"),
        }

    def _upsert_project_tsm_path(self, row: dict[str, Any]) -> None:
        normalized = self._normalize_project_tsm_path_row(row)
        if normalized is None:
            return
        self._conn.execute(
            """
            INSERT INTO TSMPaths (Code, Port, Path1, Path2, Path3, Path4, Path5, Path6)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?)
            ON CONFLICT(Code, Port) DO UPDATE SET
                Path1 = excluded.Path1,
                Path2 = excluded.Path2,
                Path3 = excluded.Path3,
                Path4 = excluded.Path4,
                Path5 = excluded.Path5,
                Path6 = excluded.Path6
            """,
            (
                normalized["Code"],
                normalized["Port"],
                normalized["Path1"],
                normalized["Path2"],
                normalized["Path3"],
                normalized["Path4"],
                normalized["Path5"],
                normalized["Path6"],
            ),
        )

    def _migrate_project_tsm_paths_collection(self) -> None:
        existing_rows = self._conn.execute("SELECT COUNT(1) AS c FROM TSMPaths").fetchone()
        if existing_rows and int(existing_rows["c"] or 0) > 0:
            return

        rows = self._conn.execute(
            "SELECT data FROM json_documents WHERE collection = ?",
            (self._name,),
        ).fetchall()
        for row in rows:
            try:
                doc = json.loads(row["data"])
            except Exception:
                continue

            if not isinstance(doc, dict):
                continue

            payload_rows = doc.get("rows")
            if isinstance(payload_rows, list):
                for item in payload_rows:
                    if isinstance(item, dict):
                        self._upsert_project_tsm_path(item)

    def _load_docs(self) -> list[dict[str, Any]]:
        with self._lock:
            if self._is_transmitters:
                rows = self._conn.execute("SELECT data FROM transmitters").fetchall()
            elif self._is_instrument_models:
                rows = self._conn.execute("SELECT id, models FROM InstrumentModels").fetchall()
            elif self._is_project_instruments:
                rows = self._conn.execute(
                    "SELECT instrument_name, model, address_main, address_redt, UseRedt FROM instruments"
                ).fetchall()
            elif self._is_project_power_meters:
                rows = self._conn.execute(
                    "SELECT PowerMeter, Channel FROM PowerMeter"
                ).fetchall()
            elif self._is_project_tsm_paths:
                rows = self._conn.execute(
                    "SELECT Code, Port, Path1, Path2, Path3, Path4, Path5, Path6 FROM TSMPaths"
                ).fetchall()
            elif self._is_configuration:
                rows = self._conn.execute(
                    "SELECT Parameter, Value FROM Configuration"
                ).fetchall()
            else:
                rows = self._conn.execute(
                    "SELECT data FROM json_documents WHERE collection = ?",
                    (self._name,),
                ).fetchall()

        if self._is_instrument_models:
            docs: list[dict[str, Any]] = []
            for row in rows:
                try:
                    models = json.loads(row["models"])
                except Exception:
                    models = []
                if not isinstance(models, list):
                    models = []
                docs.append({"_id": str(row["id"]), "models": [str(m) for m in models if m is not None]})
            return docs

        if self._is_project_instruments:
            docs: list[dict[str, Any]] = []
            for row in rows:
                docs.append(
                    {
                        "_id": str(row["instrument_name"]),
                        "instrument_name": str(row["instrument_name"]),
                        "model": str(row["model"]),
                        "address_main": str(row["address_main"]),
                        "address_redt": str(row["address_redt"]),
                        "use_redt": int(row["UseRedt"] or 0) != 0,
                    }
                )
            return docs

        if self._is_project_power_meters:
            docs: list[dict[str, Any]] = []
            for row in rows:
                docs.append(
                    {
                        "_id": str(row["PowerMeter"]),
                        "PowerMeter": str(row["PowerMeter"]),
                        "Channel": str(row["Channel"]),
                    }
                )
            return docs

        if self._is_project_tsm_paths:
            docs: list[dict[str, Any]] = []
            for row in rows:
                docs.append(
                    {
                        "_id": f"{row['Code']}|{row['Port']}",
                        "Code": str(row["Code"]),
                        "Port": str(row["Port"]),
                        "Path1": None if row["Path1"] is None else str(row["Path1"]),
                        "Path2": None if row["Path2"] is None else str(row["Path2"]),
                        "Path3": None if row["Path3"] is None else str(row["Path3"]),
                        "Path4": None if row["Path4"] is None else str(row["Path4"]),
                        "Path5": None if row["Path5"] is None else str(row["Path5"]),
                        "Path6": None if row["Path6"] is None else str(row["Path6"]),
                    }
                )
            return docs

        if self._is_configuration:
            docs: list[dict[str, Any]] = []
            for row in rows:
                docs.append(
                    {
                        "_id": str(row["Parameter"]),
                        "Parameter": str(row["Parameter"]),
                        "Value": str(row["Value"]),
                    }
                )
            return docs

        return [json.loads(row["data"]) for row in rows]

    def _save_doc(self, doc: dict[str, Any]) -> None:
        if self._is_transmitters:
            code = str(doc.get("code") or "").strip()
            if not code:
                raise ValueError("transmitter document requires non-empty 'code'")
            # Keep _id stable and aligned with row id for compatibility with delete/find callers.
            doc["_id"] = code
        elif self._is_instrument_models:
            doc_id = str(doc.get("_id") or "").strip()
            models = doc.get("models")

            if doc_id and isinstance(models, list):
                with self._lock:
                    self._upsert_instrument_models(doc_id, models)
                    self._conn.commit()
                return

            # Backward-compatibility: accept legacy single-document catalog shape.
            with self._lock:
                for key, value in doc.items():
                    if key == "_id" or not isinstance(value, list):
                        continue
                    self._upsert_instrument_models(str(key), value)
                self._conn.commit()
            return
        elif self._is_project_instruments:
            with self._lock:
                if isinstance(doc.get("rows"), list):
                    for item in doc["rows"]:
                        if isinstance(item, dict):
                            self._upsert_project_instrument(item)
                    self._conn.commit()
                    return

                self._upsert_project_instrument(doc)
                self._conn.commit()
            return
        elif self._is_project_power_meters:
            with self._lock:
                if isinstance(doc.get("rows"), list):
                    for item in doc["rows"]:
                        if isinstance(item, dict):
                            self._upsert_project_power_meter(item)
                    self._conn.commit()
                    return

                self._upsert_project_power_meter(doc)
                self._conn.commit()
            return
        elif self._is_project_tsm_paths:
            with self._lock:
                if isinstance(doc.get("rows"), list):
                    for item in doc["rows"]:
                        if isinstance(item, dict):
                            self._upsert_project_tsm_path(item)
                    self._conn.commit()
                    return

                self._upsert_project_tsm_path(doc)
                self._conn.commit()
            return
        elif self._is_configuration:
            with self._lock:
                if isinstance(doc.get("rows"), list):
                    for item in doc["rows"]:
                        if isinstance(item, dict):
                            self._upsert_configuration(item)
                    self._conn.commit()
                    return

                self._upsert_configuration(doc)
                self._conn.commit()
            return

        payload = json.dumps(doc, ensure_ascii=True)
        with self._lock:
            if self._is_transmitters:
                self._conn.execute(
                    """
                    INSERT INTO transmitters (id, data)
                    VALUES (?, ?)
                    ON CONFLICT(id) DO UPDATE SET data = excluded.data
                    """,
                    (code, payload),
                )
            else:
                self._conn.execute(
                    """
                    INSERT INTO json_documents (collection, doc_id, data)
                    VALUES (?, ?, ?)
                    ON CONFLICT(collection, doc_id) DO UPDATE SET data = excluded.data
                    """,
                    (self._name, str(doc["_id"]), payload),
                )
            self._conn.commit()

    def _delete_doc(self, doc_id: str) -> int:
        with self._lock:
            if self._is_transmitters:
                cur = self._conn.execute(
                    "DELETE FROM transmitters WHERE id = ?",
                    (doc_id,),
                )
            elif self._is_instrument_models:
                cur = self._conn.execute(
                    "DELETE FROM InstrumentModels WHERE id = ?",
                    (doc_id,),
                )
            elif self._is_project_instruments:
                cur = self._conn.execute(
                    "DELETE FROM instruments WHERE instrument_name = ?",
                    (doc_id,),
                )
            elif self._is_project_power_meters:
                cur = self._conn.execute(
                    "DELETE FROM PowerMeter WHERE PowerMeter = ?",
                    (doc_id,),
                )
            elif self._is_project_tsm_paths:
                parts = str(doc_id).split("|", 1)
                code = parts[0] if len(parts) > 0 else ""
                port = parts[1] if len(parts) > 1 else ""
                cur = self._conn.execute(
                    "DELETE FROM TSMPaths WHERE Code = ? AND Port = ?",
                    (code, port),
                )
            elif self._is_configuration:
                cur = self._conn.execute(
                    "DELETE FROM Configuration WHERE Parameter = ?",
                    (doc_id,),
                )
            else:
                cur = self._conn.execute(
                    "DELETE FROM json_documents WHERE collection = ? AND doc_id = ?",
                    (self._name, doc_id),
                )
            self._conn.commit()
        return cur.rowcount

    def _get_value(self, doc: dict[str, Any], path: str) -> Any:
        node: Any = doc
        for key in path.split("."):
            if not isinstance(node, dict) or key not in node:
                return None
            node = node[key]
        return node

    def _set_value(self, doc: dict[str, Any], path: str, value: Any) -> None:
        node = doc
        parts = path.split(".")
        for key in parts[:-1]:
            next_node = node.get(key)
            if not isinstance(next_node, dict):
                next_node = {}
                node[key] = next_node
            node = next_node
        node[parts[-1]] = value

    def _matches(self, doc: dict[str, Any], query: dict[str, Any]) -> bool:
        for key, expected in query.items():
            actual = self._get_value(doc, key)
            if isinstance(expected, dict):
                for op, op_value in expected.items():
                    if op == "$in":
                        if actual not in op_value:
                            return False
                    elif op == "$ne":
                        if actual == op_value:
                            return False
                    else:
                        return False
            else:
                if actual != expected:
                    return False
        return True

    def _apply_projection(self, doc: dict[str, Any], projection: Optional[dict[str, int]]) -> dict[str, Any]:
        if not projection:
            return copy.deepcopy(doc)

        include_fields = [k for k, v in projection.items() if v == 1 and k != "_id"]
        if include_fields:
            out = {k: copy.deepcopy(doc.get(k)) for k in include_fields if k in doc}
            if projection.get("_id", 1) == 1 and "_id" in doc:
                out["_id"] = copy.deepcopy(doc["_id"])
            return out

        out = copy.deepcopy(doc)
        for key, value in projection.items():
            if value == 0 and key in out:
                del out[key]
        return out

    def find(
        self,
        query: Optional[dict[str, Any]] = None,
        projection: Optional[dict[str, int]] = None,
    ) -> list[dict[str, Any]]:
        q = query or {}
        docs = [doc for doc in self._load_docs() if self._matches(doc, q)]
        return [self._apply_projection(doc, projection) for doc in docs]

    def find_one(
        self,
        query: Optional[dict[str, Any]] = None,
        projection: Optional[dict[str, int]] = None,
        sort: Optional[list[tuple[str, int]]] = None,
    ) -> Optional[dict[str, Any]]:
        docs = self.find(query=query, projection=projection)
        if sort:
            for field, direction in reversed(sort):
                reverse = direction < 0
                docs.sort(key=lambda d: self._get_value(d, field), reverse=reverse)
        return docs[0] if docs else None

    def update_one(
        self,
        query: dict[str, Any],
        update: dict[str, Any],
        upsert: bool = False,
    ) -> UpdateResult:
        doc = self.find_one(query=query)
        set_payload = update.get("$set", {})
        if doc is not None:
            base = copy.deepcopy(doc)
            for field, value in set_payload.items():
                self._set_value(base, field, value)
            self._save_doc(base)
            return UpdateResult(modified_count=1)

        if not upsert:
            return UpdateResult(modified_count=0)

        new_doc: dict[str, Any] = {}
        for key, value in query.items():
            if isinstance(value, dict):
                continue
            self._set_value(new_doc, key, value)
        for field, value in set_payload.items():
            self._set_value(new_doc, field, value)
        if "_id" not in new_doc:
            new_doc["_id"] = str(uuid4())

        self._save_doc(new_doc)
        return UpdateResult(modified_count=1, upserted_id=str(new_doc["_id"]))

    def delete_one(self, query: dict[str, Any]) -> DeleteResult:
        doc = self.find_one(query=query)
        if not doc:
            return DeleteResult(deleted_count=0)
        deleted = self._delete_doc(str(doc["_id"]))
        return DeleteResult(deleted_count=deleted)

    def distinct(self, field: str, query: Optional[dict[str, Any]] = None) -> list[Any]:
        values: set[Any] = set()
        for doc in self.find(query=query):
            value = self._get_value(doc, field)
            if value is not None:
                values.add(value)
        return list(values)
