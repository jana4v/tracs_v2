from typing import Any, Optional
from src.database.sqlite_json_store import SQLiteJsonCollection


def parse_bool(value: Any) -> bool:
    if isinstance(value, bool):
        return value
    if isinstance(value, int):
        return value != 0
    if isinstance(value, str):
        return value.strip().lower() in ("1", "true", "yes")
    return False


class TestSystemsRepository:
    def __init__(
        self,
        transmitters_collection: SQLiteJsonCollection,
        instruments_collection: SQLiteJsonCollection,
        project_instruments_collection: SQLiteJsonCollection,
        project_power_meters_collection: SQLiteJsonCollection,
        project_tsm_paths_collection: SQLiteJsonCollection,
        configuration_collection: SQLiteJsonCollection,
    ):
        self.transmitters_collection = transmitters_collection
        self.instruments_collection = instruments_collection
        self.project_instruments_collection = project_instruments_collection
        self.project_power_meters_collection = project_power_meters_collection
        self.project_tsm_paths_collection = project_tsm_paths_collection
        self.configuration_collection = configuration_collection

    def get_instrument_catalog(self) -> dict[str, list[str]]:
        out: dict[str, list[str]] = {}
        docs = self.instruments_collection.find({}, {"_id": 1, "models": 1})

        # New shape: one row per instrument type => {"_id": "CalPowerMeter", "models": [...]}
        for doc in docs:
            instrument_type = str(doc.get("_id", "")).strip()
            if instrument_type == "":
                continue
            models = doc.get("models")
            if isinstance(models, list):
                out[instrument_type] = [str(v) for v in models if v is not None]

        if len(out) > 0:
            return out

        # Legacy fallback: single catalog document in json_documents.
        legacy_doc = self.instruments_collection.find_one({}, {"_id": 0}) or {}
        for key, value in legacy_doc.items():
            if key == "_id":
                continue
            if isinstance(value, list):
                out[str(key)] = [str(v) for v in value if v is not None]
            else:
                out[str(key)] = []
        return out

    def _default_rows_from_catalog(self, catalog: dict[str, list[str]]) -> list[dict[str, Any]]:
        rows: list[dict[str, Any]] = []
        for instrument_name, models in catalog.items():
            default_model = str(models[0]) if len(models) > 0 else ""
            rows.append(
                {
                    "instrument_name": instrument_name,
                    "model": default_model,
                    "address_main": "",
                    "address_redt": "",
                    "use_redt": False,
                }
            )
        return rows

    def _normalize_rows_shape(self, rows: list[dict[str, Any]]) -> list[dict[str, Any]]:
        if len(rows) == 0:
            return rows

        def parse_bool(value: Any) -> bool:
            if isinstance(value, bool):
                return value
            if isinstance(value, (int, float)):
                return int(value) != 0
            return str(value or "").strip().lower() in {"1", "true", "yes", "on"}

        # Already in new shape.
        if any("address_main" in r or "address_redt" in r for r in rows if isinstance(r, dict)):
            normalized: list[dict[str, Any]] = []
            for row in rows:
                if not isinstance(row, dict):
                    continue
                normalized.append(
                    {
                        "instrument_name": str(row.get("instrument_name", "")),
                        "model": str(row.get("model", "")),
                        "address_main": str(row.get("address_main", "")),
                        "address_redt": str(row.get("address_redt", "")),
                        "use_redt": parse_bool(row.get("use_redt", False)),
                    }
                )
            return normalized

        # Legacy shape fallback: {instrument_name, model, used_as, address}
        grouped: dict[tuple[str, str], dict[str, Any]] = {}
        for row in rows:
            if not isinstance(row, dict):
                continue
            instrument_name = str(row.get("instrument_name", ""))
            model = str(row.get("model", ""))
            key = (instrument_name, model)
            item = grouped.get(key)
            if item is None:
                item = {
                    "instrument_name": instrument_name,
                    "model": model,
                    "address_main": "",
                    "address_redt": "",
                    "use_redt": False,
                }
                grouped[key] = item

            used_as = str(row.get("used_as", "")).lower()
            address = str(row.get("address", ""))
            if used_as == "redt":
                item["address_redt"] = address
            else:
                item["address_main"] = address

        return list(grouped.values())

    def get_project_instruments_rows(self) -> list[dict[str, Any]]:
        table_rows = self.project_instruments_collection.find(
            {},
            {
                "_id": 0,
                "instrument_name": 1,
                "model": 1,
                "address_main": 1,
                "address_redt": 1,
                "use_redt": 1,
            },
        )
        if len(table_rows) > 0:
            return self._normalize_rows_shape(table_rows)

        stored = self.project_instruments_collection.find_one(
            {"project": "default"},
            {"_id": 0, "rows": 1},
        )
        if stored and isinstance(stored.get("rows"), list):
            return self._normalize_rows_shape(stored["rows"])

        catalog = self.get_instrument_catalog()
        return self._default_rows_from_catalog(catalog)

    def save_project_instruments_rows(self, rows: list[dict[str, Any]]) -> int:
        normalized = self._normalize_rows_shape(rows)
        normalized = [r for r in normalized if str(r.get("instrument_name", "")).strip() != ""]

        incoming_ids = {str(r["instrument_name"]).strip() for r in normalized}
        existing = self.project_instruments_collection.find({}, {"_id": 1})
        for doc in existing:
            doc_id = str(doc.get("_id", "")).strip()
            if doc_id and doc_id not in incoming_ids:
                self.project_instruments_collection.delete_one({"_id": doc_id})

        for row in normalized:
            instrument_name = str(row["instrument_name"]).strip()
            payload = {
                "_id": instrument_name,
                "instrument_name": instrument_name,
                "model": str(row.get("model", "")),
                "address_main": str(row.get("address_main", "")),
                "address_redt": str(row.get("address_redt", "")),
                "use_redt": parse_bool(row.get("use_redt", False)),
            }
            self.project_instruments_collection.update_one(
                {"_id": instrument_name},
                {"$set": payload},
                upsert=True,
            )

        return len(normalized)

    def _default_power_meter_rows_from_catalog(self, catalog: dict[str, list[str]]) -> list[dict[str, Any]]:
        rows: list[dict[str, Any]] = []
        keys = sorted(catalog.keys(), key=lambda x: x.lower())
        for name in keys:
            low = str(name).lower()
            if "powermeter" not in low:
                continue
            default_channel = "B" if "downlinkpowermeter" in low else "A"
            rows.append(
                {
                    "powerMeter": str(name),
                    "channel": default_channel,
                }
            )
        return rows

    def _normalize_power_meter_rows(self, rows: list[dict[str, Any]]) -> list[dict[str, Any]]:
        out: list[dict[str, Any]] = []
        for row in rows:
            if not isinstance(row, dict):
                continue
            name = str(row.get("powerMeter", "")).strip()
            if name == "":
                continue
            channel = str(row.get("channel", "A")).upper()
            out.append(
                {
                    "powerMeter": name,
                    "channel": "B" if channel == "B" else "A",
                }
            )
        return out

    def get_project_power_meter_rows(self) -> list[dict[str, Any]]:
        catalog = self.get_instrument_catalog()
        defaults = self._default_power_meter_rows_from_catalog(catalog)

        table_rows = self.project_power_meters_collection.find(
            {},
            {
                "_id": 0,
                "PowerMeter": 1,
                "Channel": 1,
            },
        )
        if len(table_rows) > 0:
            saved_rows = self._normalize_power_meter_rows(
                [
                    {
                        "powerMeter": str(row.get("PowerMeter", "")),
                        "channel": str(row.get("Channel", "A")),
                    }
                    for row in table_rows
                ]
            )
            saved_by_name = {r["powerMeter"]: r for r in saved_rows}

            merged: list[dict[str, Any]] = []
            for default_row in defaults:
                name = default_row["powerMeter"]
                merged.append(saved_by_name.get(name, default_row))

            if len(merged) > 0:
                return merged

        stored = self.project_power_meters_collection.find_one(
            {"project": "default"},
            {"_id": 0, "rows": 1},
        )
        if not stored or not isinstance(stored.get("rows"), list):
            return defaults

        saved_rows = self._normalize_power_meter_rows(stored["rows"])
        saved_by_name = {r["powerMeter"]: r for r in saved_rows}

        merged: list[dict[str, Any]] = []
        for default_row in defaults:
            name = default_row["powerMeter"]
            merged.append(saved_by_name.get(name, default_row))
        return merged

    def save_project_power_meter_rows(self, rows: list[dict[str, Any]]) -> int:
        normalized = self._normalize_power_meter_rows(rows)

        incoming_ids = {str(r["powerMeter"]).strip() for r in normalized}
        existing = self.project_power_meters_collection.find({}, {"_id": 1})
        for doc in existing:
            doc_id = str(doc.get("_id", "")).strip()
            if doc_id and doc_id not in incoming_ids:
                self.project_power_meters_collection.delete_one({"_id": doc_id})

        for row in normalized:
            meter_name = str(row["powerMeter"]).strip()
            payload = {
                "_id": meter_name,
                "PowerMeter": meter_name,
                "Channel": "B" if str(row.get("channel", "A")).upper() == "B" else "A",
            }
            self.project_power_meters_collection.update_one(
                {"_id": meter_name},
                {"$set": payload},
                upsert=True,
            )

        return len(normalized)

    def _flatten_ports(self, ports: Any) -> list[str]:
        if not isinstance(ports, list):
            return []

        out: list[str] = []

        def walk(value: Any) -> None:
            if isinstance(value, list):
                for item in value:
                    walk(item)
                return
            if value is None:
                return
            text = str(value).strip()
            if text != "":
                out.append(text)

        walk(ports)
        seen: set[str] = set()
        unique: list[str] = []
        for item in out:
            if item in seen:
                continue
            seen.add(item)
            unique.append(item)
        return unique

    def _default_tsm_path_rows_from_systems(self) -> list[dict[str, Any]]:
        docs = self.transmitters_collection.find(
            {"system_type": "Transmitter"},
            {"_id": 0, "code": 1, "modulation_details": 1},
        )

        rows: list[dict[str, Any]] = []
        for doc in docs:
            code = str(doc.get("code", "")).strip()
            if code == "":
                continue

            ports = self._flatten_ports((doc.get("modulation_details") or {}).get("ports"))
            if len(ports) == 0:
                ports = [""]

            for port in ports:
                rows.append(
                    {
                        "code": code,
                        "port": port,
                        "path1": None,
                        "path2": None,
                        "path3": None,
                        "path4": None,
                        "path5": None,
                        "path6": None,
                    }
                )

        # Keep a dedicated calibration injection row available regardless of systems.
        if not any(str(row.get("code", "")).strip() == "INJECT_CAL" for row in rows):
            rows.append(
                {
                    "code": "INJECT_CAL",
                    "port": "",
                    "path1": None,
                    "path2": None,
                    "path3": None,
                    "path4": None,
                    "path5": None,
                    "path6": None,
                }
            )

        rows.sort(key=lambda item: (str(item["code"]), str(item["port"])))
        return rows

    def _normalize_tsm_path_rows(self, rows: list[dict[str, Any]]) -> list[dict[str, Any]]:
        out: list[dict[str, Any]] = []
        for row in rows:
            if not isinstance(row, dict):
                continue
            code = str(row.get("code", row.get("Code", ""))).strip()
            if code == "":
                continue

            def normalize_nullable(*keys: str) -> Any:
                sentinel = object()
                value: Any = sentinel
                for key in keys:
                    if key in row:
                        value = row[key]
                        break
                if value is sentinel or value is None:
                    return None
                return str(value).strip()

            out.append(
                {
                    "code": code,
                    "port": str(row.get("port", row.get("Port", ""))).strip(),
                    "path1": normalize_nullable("path1", "Path1"),
                    "path2": normalize_nullable("path2", "Path2"),
                    "path3": normalize_nullable("path3", "Path3"),
                    "path4": normalize_nullable("path4", "Path4"),
                    "path5": normalize_nullable("path5", "Path5"),
                    "path6": normalize_nullable("path6", "Path6"),
                }
            )
        out.sort(key=lambda item: (item["code"], item["port"]))
        return out

    def _sync_tsm_path_rows(self, overrides: Optional[list[dict[str, Any]]] = None) -> list[dict[str, Any]]:
        desired_rows = self._default_tsm_path_rows_from_systems()
        existing_rows = self._normalize_tsm_path_rows(
            self.project_tsm_paths_collection.find(
                {},
                {
                    "_id": 0,
                    "Code": 1,
                    "Port": 1,
                    "Path1": 1,
                    "Path2": 1,
                    "Path3": 1,
                    "Path4": 1,
                    "Path5": 1,
                    "Path6": 1,
                },
            )
        )
        source_rows = self._normalize_tsm_path_rows(overrides if overrides is not None else existing_rows)
        source_by_key = {(str(row["code"]), str(row["port"])): row for row in source_rows}

        merged_rows: list[dict[str, Any]] = []
        for desired in desired_rows:
            key = (str(desired["code"]), str(desired["port"]))
            stored = source_by_key.get(key)
            merged_rows.append({**desired, **(stored or {})})

        existing_ids = {str(doc.get("_id", "")).strip() for doc in self.project_tsm_paths_collection.find({}, {"_id": 1})}
        desired_ids = {f"{row['code']}|{row['port']}" for row in merged_rows}
        for doc_id in existing_ids:
            if doc_id and doc_id not in desired_ids:
                self.project_tsm_paths_collection.delete_one({"_id": doc_id})

        for row in merged_rows:
            payload = {
                "_id": f"{row['code']}|{row['port']}",
                "Code": row["code"],
                "Port": row["port"],
                "Path1": row["path1"],
                "Path2": row["path2"],
                "Path3": row["path3"],
                "Path4": row["path4"],
                "Path5": row["path5"],
                "Path6": row["path6"],
            }
            self.project_tsm_paths_collection.update_one(
                {"_id": payload["_id"]},
                {"$set": payload},
                upsert=True,
            )

        return merged_rows

    def get_project_tsm_path_rows(self) -> list[dict[str, Any]]:
        return self._sync_tsm_path_rows()

    def get_project_tsm_path_rows_for_code(self, code: str, port: str | None = None) -> list[dict[str, Any]]:
        target_code = str(code or "").strip()
        target_port = None if port is None else str(port).strip()
        if target_code == "":
            return []

        rows = self.get_project_tsm_path_rows()
        matches: list[dict[str, Any]] = []
        for row in rows:
            row_code = str(row.get("code", "")).strip()
            row_port = str(row.get("port", "")).strip()
            if row_code != target_code:
                continue
            if target_port is not None and row_port != target_port:
                continue
            matches.append(row)
        return matches

    def save_project_tsm_path_rows(self, rows: list[dict[str, Any]]) -> int:
        merged_rows = self._sync_tsm_path_rows(overrides=rows)
        return len(merged_rows)

    def get_configuration_value(self, parameter: str, default: str = "") -> str:
        key = str(parameter or "").strip()
        if key == "":
            return str(default)
        doc = self.configuration_collection.find_one({"Parameter": key}, {"_id": 0, "Value": 1})
        if not doc:
            return str(default)
        return str(doc.get("Value", default))

    def set_configuration_value(self, parameter: str, value: str) -> None:
        key = str(parameter or "").strip()
        if key == "":
            return
        payload = {
            "_id": key,
            "Parameter": key,
            "Value": str(value or ""),
        }
        self.configuration_collection.update_one(
            {"_id": key},
            {"$set": payload},
            upsert=True,
        )
