from typing import Any, List, Optional
from pydantic import TypeAdapter
from src.database.sqlite_json_store import SQLiteJsonCollection
from src.schemas.transmitter_test_parameters import (
    FrequencySpecRow,
    ModulationIndexSpecRow,
    PowerSpecRow,
    SpuriousSpecRow,
)
from src.schemas.transmitter import TransmitterCreate, TransmitterResponse
from src.schemas.enums import ModulationType, SystemType
from src.database.system_catalog import get_catalog_store
from src.repositories.system_catalog_repo import SystemCatalogRepository


transmitter_response_adapter = TypeAdapter(TransmitterResponse)

PARAMETER_CONFIG: dict[str, dict[str, Any]] = {
    "power": {
        "details_key": "power_specs",
        "row_model": PowerSpecRow,
        "applicable": {ModulationType.PSK_PM.value},
        "db_fields": {"specification", "tolerance", "fbt", "fbt_hot", "fbt_cold"},
    },
    "frequency": {
        "details_key": "frequency_specs",
        "row_model": FrequencySpecRow,
        "applicable": {ModulationType.PSK_PM.value},
        "db_fields": {"tolerance", "fbt", "fbt_hot", "fbt_cold"},
    },
    "modulation_index": {
        "details_key": "modulation_index_specs",
        "row_model": ModulationIndexSpecRow,
        "applicable": {ModulationType.PSK_PM.value},
        "db_fields": {"specification", "tolerance"},
    },
    "spurious": {
        "details_key": "spurious_specs",
        "row_model": SpuriousSpecRow,
        "applicable": {ModulationType.PSK_PM.value},
        "db_fields": {"profile_name", "enable", "profiles", "specification", "tolerance", "fbt", "fbt_hot", "fbt_cold"},
    },
}


class TransmitterRepository:
    """CRUD operations for transmitters."""

    def __init__(self, collection: SQLiteJsonCollection, tsm_paths_collection: Optional[SQLiteJsonCollection] = None, misc_collection: Optional[SQLiteJsonCollection] = None):
        self.collection = collection
        self.tsm_paths_collection = tsm_paths_collection
        self.misc_collection = misc_collection or collection
        self._catalog_repo: Optional[SystemCatalogRepository] = None

    def _get_catalog_repo(self) -> Optional[SystemCatalogRepository]:
        if self._catalog_repo is not None:
            return self._catalog_repo
        try:
            self._catalog_repo = SystemCatalogRepository(get_catalog_store(), self.collection)
            return self._catalog_repo
        except Exception:
            return None

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

    def _sync_tsm_rows_for_transmitter(self, code: str, ports: list[str]) -> None:
        if self.tsm_paths_collection is None:
            return

        valid_ports = ports if len(ports) > 0 else [""]
        desired_ids = {f"{code}|{port}" for port in valid_ports}

        existing_rows = self.tsm_paths_collection.find(
            {"Code": code},
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
        existing_by_port = {str(row.get("Port", "")): row for row in existing_rows}

        existing_ids = {str(doc.get("_id", "")).strip() for doc in self.tsm_paths_collection.find({"Code": code}, {"_id": 1})}
        for doc_id in existing_ids:
            if doc_id and doc_id not in desired_ids:
                self.tsm_paths_collection.delete_one({"_id": doc_id})

        for port in valid_ports:
            current = existing_by_port.get(port, {})
            payload = {
                "_id": f"{code}|{port}",
                "Code": code,
                "Port": port,
                "Path1": None if current.get("Path1") is None else str(current.get("Path1")),
                "Path2": None if current.get("Path2") is None else str(current.get("Path2")),
                "Path3": None if current.get("Path3") is None else str(current.get("Path3")),
                "Path4": None if current.get("Path4") is None else str(current.get("Path4")),
                "Path5": None if current.get("Path5") is None else str(current.get("Path5")),
                "Path6": None if current.get("Path6") is None else str(current.get("Path6")),
            }
            self.tsm_paths_collection.update_one(
                {"_id": payload["_id"]},
                {"$set": payload},
                upsert=True,
            )

    def _delete_tsm_rows_for_transmitter(self, code: str) -> None:
        if self.tsm_paths_collection is None:
            return
        existing_ids = [str(doc.get("_id", "")).strip() for doc in self.tsm_paths_collection.find({"Code": code}, {"_id": 1})]
        for doc_id in existing_ids:
            if doc_id:
                self.tsm_paths_collection.delete_one({"_id": doc_id})

    def get_all_for_system_type(self, system_type: SystemType) -> List[TransmitterResponse]:
        cursor = self.collection.find(
            {"system_type": system_type.value},
            {"_id": 0},
        )
        return [transmitter_response_adapter.validate_python(doc) for doc in cursor]

    def get_all(self) -> List[TransmitterResponse]:
        return self.get_all_for_system_type(SystemType.Transmitter)

    def get_by_code_for_system_type(self, code: str, system_type: SystemType) -> Optional[TransmitterResponse]:
        doc = self.collection.find_one(
            {"system_type": system_type.value, "code": code},
            {"_id": 0},
        )
        return transmitter_response_adapter.validate_python(doc) if doc else None

    def get_by_code(self, code: str) -> Optional[TransmitterResponse]:
        return self.get_by_code_for_system_type(code, SystemType.Transmitter)

    def upsert_for_system_type(self, transmitter: TransmitterCreate, system_type: SystemType) -> TransmitterResponse:
        """Insert or update by code (upsert semantics)."""
        doc = transmitter.model_dump()
        doc["system_type"] = system_type.value
        self.collection.update_one(
            {"system_type": system_type.value, "code": transmitter.code},
            {"$set": doc},
            upsert=True,
        )
        if system_type == SystemType.Transmitter:
            ports = self._flatten_ports((doc.get("modulation_details") or {}).get("ports"))
            self._sync_tsm_rows_for_transmitter(str(transmitter.code), ports)

        catalog_repo = self._get_catalog_repo()
        if catalog_repo is not None:
            details = (doc.get("modulation_details") or {}) if isinstance(doc.get("modulation_details"), dict) else {}
            catalog_repo.sync_system_catalog_from_form(
                system_kind=str(system_type.value).strip().lower(),
                system_code=str(transmitter.code),
                ports=details.get("ports") if isinstance(details.get("ports"), list) else [],
                frequencies=details.get("frequencies") if isinstance(details.get("frequencies"), list) else [],
            )

        return transmitter_response_adapter.validate_python(doc)

    def upsert(self, transmitter: TransmitterCreate) -> TransmitterResponse:
        return self.upsert_for_system_type(transmitter, SystemType.Transmitter)

    def delete_for_system_type(self, code: str, system_type: SystemType) -> bool:
        result = self.collection.delete_one(
            {"system_type": system_type.value, "code": code}
        )
        if result.deleted_count > 0:
            if system_type == SystemType.Transmitter:
                self._delete_tsm_rows_for_transmitter(code)
            catalog_repo = self._get_catalog_repo()
            if catalog_repo is not None:
                catalog_repo.delete_system(str(system_type.value).strip().lower(), code)
        return result.deleted_count > 0

    def delete(self, code: str) -> bool:
        return self.delete_for_system_type(code, SystemType.Transmitter)

    def code_exists(self, code: str) -> bool:
        return self.collection.find_one(
            {"system_type": SystemType.Transmitter.value, "code": code},
            {"_id": 1},
        ) is not None

    def name_exists(self, name: str, exclude_code: Optional[str] = None) -> bool:
        query: dict = {"system_type": SystemType.Transmitter.value, "name": name}
        if exclude_code:
            query["code"] = {"$ne": exclude_code}
        return self.collection.find_one(query, {"_id": 1}) is not None

    def _to_db_number(self, value: Any) -> Any:
        if value is None or value == "":
            return value
        if isinstance(value, bool):
            return value
        if isinstance(value, (int, float)):
            return round(float(value), 6)
        if isinstance(value, str):
            try:
                return round(float(value.strip()), 6)
            except ValueError:
                return value
        return value

    def _normalize_db_value(self, value: Any) -> Any:
        if isinstance(value, list):
            return [self._normalize_db_value(item) for item in value]
        return self._to_db_number(value)

    def _build_key(self, row: dict[str, Any]) -> str:
        return f"{row.get('port','')}|{row.get('frequency_label','')}|{row.get('frequency','')}"

    def _extract_ports_frequencies(self, details: dict[str, Any]) -> list[tuple[str, str, str]]:
        ports = details.get("ports", []) or []
        frequencies = details.get("frequencies", []) or []

        combos: list[tuple[str, str, str]] = []
        for port_row in ports:
            if not isinstance(port_row, list) or len(port_row) == 0:
                continue
            port = str(port_row[0] or "").strip()
            if not port:
                continue

            for freq_row in frequencies:
                if not isinstance(freq_row, list) or len(freq_row) < 2:
                    continue
                label = str(freq_row[0] or "").strip()
                freq = str(freq_row[1] or "").strip()
                if not label or not freq:
                    continue
                combos.append((port, label, freq))

        return combos

    def _default_row(self, parameter: str, tx_code: str, port: str, label: str, freq: str, details: dict[str, Any]) -> dict[str, Any]:
        base = {
            "code": tx_code,
            "port": port,
            "frequency_label": label,
            "frequency": freq,
        }

        if parameter == "power":
            return {
                **base,
                "specification": None,
                "tolerance": 0.5,
                "fbt": None,
                "fbt_hot": None,
                "fbt_cold": None,
            }

        if parameter == "frequency":
            return {
                **base,
                "tolerance": 4,
                "fbt": None,
                "fbt_hot": None,
                "fbt_cold": None,
            }

        if parameter == "modulation_index":
            row = {
                **base,
                "specification": None,
                "tolerance": 20,
            }
            tones = details.get("sub_carriers", []) or []
            for tone_row in tones:
                if not isinstance(tone_row, list) or len(tone_row) == 0:
                    continue
                tone = str(tone_row[0] or "").strip()
                if not tone:
                    continue
                row[f"fbt_tone_{tone}"] = None
                row[f"fbt_hot_tone_{tone}"] = None
                row[f"fbt_cold_tone_{tone}"] = None
            return row

        # spurious
        return {
            **base,
            "profile_name": "",
            "enable": True,
            "profiles": [],
            "specification": None,
            "tolerance": -50,
            "fbt": [["", ""]],
            "fbt_hot": [["", ""]],
            "fbt_cold": [["", ""]],
        }

    def _merge_with_defaults(self, parameter: str, tx_code: str, details: dict[str, Any], existing: list[dict[str, Any]]) -> list[dict[str, Any]]:
        combos = self._extract_ports_frequencies(details)
        existing_map: dict[str, dict[str, Any]] = {}
        for row in existing:
            if isinstance(row, dict):
                existing_map[self._build_key(row)] = row

        merged_rows: list[dict[str, Any]] = []
        for port, label, freq in combos:
            default = self._default_row(parameter, tx_code, port, label, freq, details)
            key = self._build_key(default)
            stored = existing_map.get(key)
            merged_rows.append({**default, **(stored or {})})

        # Keep any stored rows that don't match current port/frequency combinations
        combo_keys = {self._build_key({"port": p, "frequency_label": l, "frequency": f}) for p, l, f in combos}
        for key, row in existing_map.items():
            if key not in combo_keys:
                merged_rows.append(row)

        return merged_rows

    def get_parameter_rows(self, parameter: str) -> list[dict[str, Any]]:
        config = PARAMETER_CONFIG.get(parameter)
        if not config:
            raise ValueError(f"Unsupported parameter: {parameter}")

        details_key = config["details_key"]
        applicable = config["applicable"]

        cursor = self.collection.find(
            {
                "system_type": SystemType.Transmitter.value,
                "modulation_type": {"$in": list(applicable)},
            },
            {
                "_id": 0,
                "code": 1,
                "name": 1,
                "modulation_type": 1,
                "modulation_details": 1,
            },
        )

        rows: list[dict[str, Any]] = []
        for doc in cursor:
            tx_code = doc.get("code")
            tx_name = doc.get("name")
            modulation_type = doc.get("modulation_type")
            details = doc.get("modulation_details") or {}
            items = details.get(details_key, [])
            merged_items = self._merge_with_defaults(
                parameter=parameter,
                tx_code=tx_code,
                details=details,
                existing=items if isinstance(items, list) else [],
            )

            for row in merged_items:
                if isinstance(row, dict):
                    row.setdefault("code", tx_code)
                    rows.append(
                        {
                            "transmitter_code": tx_code,
                            "transmitter_name": tx_name,
                            "modulation_type": modulation_type,
                            "row": row,
                        }
                    )

        return rows

    def update_parameter_rows(self, parameter: str, updates: list[dict[str, Any]]) -> dict[str, int]:
        config = PARAMETER_CONFIG.get(parameter)
        if not config:
            raise ValueError(f"Unsupported parameter: {parameter}")

        details_key = config["details_key"]
        row_model = config["row_model"]
        db_fields = config["db_fields"]
        applicable = config["applicable"]

        grouped: dict[str, list[dict[str, Any]]] = {}
        for item in updates:
            tx_code = item.get("transmitter_code")
            row = item.get("row")
            if not tx_code or not isinstance(row, dict):
                continue
            grouped.setdefault(tx_code, []).append(row)

        updated_transmitters = 0
        updated_rows = 0

        for tx_code, rows in grouped.items():
            doc = self.collection.find_one(
                {
                    "system_type": SystemType.Transmitter.value,
                    "code": tx_code,
                    "modulation_type": {"$in": list(applicable)},
                },
                {"_id": 1, "code": 1},
            )
            if not doc:
                continue

            normalized_rows: list[dict[str, Any]] = []
            for row in rows:
                row["code"] = tx_code
                validated_row = row_model(**row).model_dump()

                for field_name in db_fields:
                    if field_name in validated_row:
                        validated_row[field_name] = self._normalize_db_value(validated_row[field_name])

                normalized_rows.append(validated_row)

            self.collection.update_one(
                {"_id": doc["_id"]},
                {"$set": {f"modulation_details.{details_key}": normalized_rows}},
            )

            updated_transmitters += 1
            updated_rows += len(normalized_rows)

        return {
            "updated_transmitters": updated_transmitters,
            "updated_rows": updated_rows,
        }

    def get_on_board_loss_rows(self) -> list[dict[str, Any]]:
        applicable = {ModulationType.PSK_PM.value}

        cursor = self.collection.find(
            {
                "system_type": SystemType.Transmitter.value,
                "modulation_type": {"$in": list(applicable)},
            },
            {
                "_id": 0,
                "code": 1,
                "name": 1,
                "modulation_type": 1,
                "modulation_details": 1,
            },
        )

        rows: list[dict[str, Any]] = []
        for doc in cursor:
            tx_code = doc.get("code")
            tx_name = doc.get("name")
            modulation_type = doc.get("modulation_type")
            details = doc.get("modulation_details") or {}

            power_rows = details.get("power_specs", [])
            merged_power_rows = self._merge_with_defaults(
                parameter="power",
                tx_code=tx_code,
                details=details,
                existing=power_rows if isinstance(power_rows, list) else [],
            )

            stored_loss_rows = details.get("on_board_loss_specs", [])
            stored_map: dict[str, dict[str, Any]] = {}
            if isinstance(stored_loss_rows, list):
                for item in stored_loss_rows:
                    if not isinstance(item, dict):
                        continue
                    stored_map[self._build_key(item)] = item

            for power_row in merged_power_rows:
                if not isinstance(power_row, dict):
                    continue

                key = self._build_key(power_row)
                stored = stored_map.get(key, {})
                row = {
                    "code": str(power_row.get("code", tx_code or "")),
                    "port": str(power_row.get("port", "")),
                    "frequency_label": str(power_row.get("frequency_label", "")),
                    "frequency": str(power_row.get("frequency", "")),
                    "loss_db": stored.get("loss_db", 0),
                }

                rows.append(
                    {
                        "transmitter_code": tx_code,
                        "transmitter_name": tx_name,
                        "modulation_type": modulation_type,
                        "row": row,
                    }
                )

        return rows

    def update_on_board_loss_rows(self, updates: list[dict[str, Any]]) -> dict[str, int]:
        applicable = {ModulationType.PSK_PM.value}

        grouped: dict[str, list[dict[str, Any]]] = {}
        for item in updates:
            tx_code = item.get("transmitter_code")
            row = item.get("row")
            if not tx_code or not isinstance(row, dict):
                continue
            grouped.setdefault(tx_code, []).append(row)

        updated_transmitters = 0
        updated_rows = 0

        for tx_code, rows in grouped.items():
            doc = self.collection.find_one(
                {
                    "system_type": SystemType.Transmitter.value,
                    "code": tx_code,
                    "modulation_type": {"$in": list(applicable)},
                },
                {"_id": 1, "code": 1},
            )
            if not doc:
                continue

            normalized_rows: list[dict[str, Any]] = []
            for row in rows:
                normalized_rows.append(
                    {
                        "code": str(row.get("code", tx_code)),
                        "port": str(row.get("port", "")),
                        "frequency_label": str(row.get("frequency_label", "")),
                        "frequency": str(row.get("frequency", "")),
                        "loss_db": self._normalize_db_value(0 if str(row.get("loss_db", "")).strip() == "" else row.get("loss_db", 0)),
                    }
                )

            self.collection.update_one(
                {"_id": doc["_id"]},
                {"$set": {"modulation_details.on_board_loss_specs": normalized_rows}},
            )

            updated_transmitters += 1
            updated_rows += len(normalized_rows)

        return {
            "updated_transmitters": updated_transmitters,
            "updated_rows": updated_rows,
        }

    def get_calibration_rows(self) -> list[dict[str, Any]]:
        applicable = {ModulationType.PSK_PM.value}

        cursor = self.collection.find(
            {
                "system_type": SystemType.Transmitter.value,
                "modulation_type": {"$in": list(applicable)},
            },
            {
                "_id": 0,
                "code": 1,
                "name": 1,
                "modulation_type": 1,
                "modulation_details": 1,
            },
        )

        rows: list[dict[str, Any]] = []
        for doc in cursor:
            tx_code = doc.get("code")
            tx_name = doc.get("name")
            modulation_type = doc.get("modulation_type")
            details = doc.get("modulation_details") or {}

            power_rows = details.get("power_specs", [])
            merged_power_rows = self._merge_with_defaults(
                parameter="power",
                tx_code=tx_code,
                details=details,
                existing=power_rows if isinstance(power_rows, list) else [],
            )

            stored_loss_rows = details.get("on_board_loss_specs", [])
            stored_loss_map: dict[str, dict[str, Any]] = {}
            if isinstance(stored_loss_rows, list):
                for item in stored_loss_rows:
                    if isinstance(item, dict):
                        stored_loss_map[self._build_key(item)] = item

            stored_cal_rows = details.get("calibration_specs", [])
            stored_cal_map: dict[str, dict[str, Any]] = {}
            if isinstance(stored_cal_rows, list):
                for item in stored_cal_rows:
                    if isinstance(item, dict):
                        stored_cal_map[self._build_key(item)] = item

            for power_row in merged_power_rows:
                if not isinstance(power_row, dict):
                    continue

                key = self._build_key(power_row)
                loss_row = stored_loss_map.get(key, {})
                cal_row = stored_cal_map.get(key, {})

                system_loss = loss_row.get("loss_db", "")
                fixed_pad_loss = cal_row.get("fixed_pad_loss", 0)
                antenna_gain = cal_row.get("antenna_gain", 0)

                def _num(v: Any) -> float:
                    try:
                        return float(v)
                    except Exception:
                        return 0.0

                total_loss = _num(antenna_gain) - _num(system_loss) - _num(fixed_pad_loss)

                row = {
                    "code": str(power_row.get("code", tx_code or "")),
                    "port": str(power_row.get("port", "")),
                    "frequency_label": str(power_row.get("frequency_label", "")),
                    "frequency": str(power_row.get("frequency", "")),
                    "system_loss": system_loss,
                    "fixed_pad_loss": fixed_pad_loss,
                    "antenna_gain": antenna_gain,
                    "total_loss": total_loss,
                }

                rows.append(
                    {
                        "transmitter_code": tx_code,
                        "transmitter_name": tx_name,
                        "modulation_type": modulation_type,
                        "row": row,
                    }
                )

        return rows

    def update_calibration_rows(self, updates: list[dict[str, Any]]) -> dict[str, int]:
        applicable = {ModulationType.PSK_PM.value}

        grouped: dict[str, list[dict[str, Any]]] = {}
        for item in updates:
            tx_code = item.get("transmitter_code")
            row = item.get("row")
            if not tx_code or not isinstance(row, dict):
                continue
            grouped.setdefault(tx_code, []).append(row)

        updated_transmitters = 0
        updated_rows = 0

        for tx_code, rows in grouped.items():
            doc = self.collection.find_one(
                {
                    "system_type": SystemType.Transmitter.value,
                    "code": tx_code,
                    "modulation_type": {"$in": list(applicable)},
                },
                {"_id": 1, "code": 1},
            )
            if not doc:
                continue

            normalized_rows: list[dict[str, Any]] = []
            for row in rows:
                system_loss = 0 if str(row.get("system_loss", "")).strip() == "" else row.get("system_loss", 0)
                fixed_pad_loss = 0 if str(row.get("fixed_pad_loss", "")).strip() == "" else row.get("fixed_pad_loss", 0)
                antenna_gain = 0 if str(row.get("antenna_gain", "")).strip() == "" else row.get("antenna_gain", 0)

                def _num(v: Any) -> float:
                    try:
                        return float(v)
                    except Exception:
                        return 0.0

                total_loss = _num(antenna_gain) - _num(system_loss) - _num(fixed_pad_loss)

                normalized_rows.append(
                    {
                        "code": str(row.get("code", tx_code)),
                        "port": str(row.get("port", "")),
                        "frequency_label": str(row.get("frequency_label", "")),
                        "frequency": str(row.get("frequency", "")),
                        "system_loss": self._normalize_db_value(system_loss),
                        "fixed_pad_loss": self._normalize_db_value(fixed_pad_loss),
                        "antenna_gain": self._normalize_db_value(antenna_gain),
                        "total_loss": self._normalize_db_value(total_loss),
                    }
                )

            self.collection.update_one(
                {"_id": doc["_id"]},
                {"$set": {"modulation_details.calibration_specs": normalized_rows}},
            )

            updated_transmitters += 1
            updated_rows += len(normalized_rows)

        return {
            "updated_transmitters": updated_transmitters,
            "updated_rows": updated_rows,
        }

    # ── Spurious Band Config (standalone, not per-transmitter) ────────────────

    _SPURIOUS_BAND_CONFIG_KEY = "spurious_band_config"

    def get_spurious_band_configs(self) -> list[dict[str, Any]]:
        doc = self.misc_collection.find_one({"_type": self._SPURIOUS_BAND_CONFIG_KEY}, {"_id": 0})
        if not doc:
            return []
        return doc.get("bands", [])

    def save_spurious_band_configs(self, bands: list[dict[str, Any]]) -> int:
        normalized = [
            {
                "profile_name": str(b.get("profile_name") or ""),
                "enable": bool(b.get("enable", True)),
                "start_frequency": self._to_db_number(b.get("start_frequency")),
                "stop_frequency": self._to_db_number(b.get("stop_frequency")),
            }
            for b in bands
        ]
        self.misc_collection.update_one(
            {"_type": self._SPURIOUS_BAND_CONFIG_KEY},
            {"$set": {"_type": self._SPURIOUS_BAND_CONFIG_KEY, "bands": normalized}},
            upsert=True,
        )
        return len(normalized)
