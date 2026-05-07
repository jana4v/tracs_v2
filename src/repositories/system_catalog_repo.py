"""
SystemCatalogRepository — high-level operations over the system catalog and
dataset row tables, plus a one-shot migration from the legacy
`transmitters.modulation_details` JSON layout.

This is additive to the existing `TransmitterRepository`. It does not modify
or remove any existing storage.
"""

from __future__ import annotations

from typing import Any, Optional

from src.database.system_catalog import (
    ALL_PARAMETER_TYPES,
    ALL_SYSTEM_KINDS,
    PARAMETER_TYPE_COMMAND_THRESHOLD,
    PARAMETER_TYPE_FREQUENCY,
    PARAMETER_TYPE_MODULATION_INDEX,
    PARAMETER_TYPE_POWER,
    PARAMETER_TYPE_RANGING_THRESHOLD,
    PARAMETER_TYPE_SPURIOUS,
    SYSTEM_KIND_RECEIVER,
    SYSTEM_KIND_TRANSPONDER,
    SYSTEM_KIND_TRANSMITTER,
    SystemCatalogStore,
)
from src.database.sqlite_json_store import SQLiteJsonCollection


# Mapping from `modulation_details` JSON keys to parameter_type discriminator.
_DETAILS_KEY_TO_PARAMETER: dict[str, str] = {
    "power_specs": PARAMETER_TYPE_POWER,
    "frequency_specs": PARAMETER_TYPE_FREQUENCY,
    "modulation_index_specs": PARAMETER_TYPE_MODULATION_INDEX,
    "spurious_specs": PARAMETER_TYPE_SPURIOUS,
}

_STRUCTURAL_FIELDS: tuple[str, ...] = (
    "code",
    "port",
    "frequency_label",
    "frequency",
)


def _norm_str(value: Any) -> str:
    if value is None:
        return ""
    return str(value).strip()


def _split_payload(row: dict[str, Any]) -> dict[str, Any]:
    """Strip structural fields from a row; what remains is the JSON payload."""
    return {k: v for k, v in row.items() if k not in _STRUCTURAL_FIELDS}


class SystemCatalogRepository:
    def __init__(
        self,
        store: SystemCatalogStore,
        transmitters_collection: SQLiteJsonCollection,
    ) -> None:
        self._store = store
        self._transmitters = transmitters_collection

    def _validate_system_kind(self, system_kind: str) -> str:
        kind = _norm_str(system_kind).lower()
        if kind not in ALL_SYSTEM_KINDS:
            raise ValueError(f"Unsupported system_kind: {system_kind}")
        return kind

    def _load_system_form_details(
        self,
        system_kind: str,
        system_code: str,
    ) -> dict[str, Any]:
        doc = self._transmitters.find_one(
            query={
                "system_type": system_kind.title(),
                "code": system_code,
            }
        )
        if not isinstance(doc, dict):
            return {}
        details = doc.get("modulation_details")
        return details if isinstance(details, dict) else {}

    def _ensure_system_catalog_from_stored_form(
        self,
        system_kind: str,
        system_code: str,
    ) -> None:
        details = self._load_system_form_details(system_kind, system_code)
        if not details:
            return
        self.sync_system_catalog_from_form(
            system_kind=system_kind,
            system_code=system_code,
            ports=details.get("ports") if isinstance(details.get("ports"), list) else [],
            frequencies=details.get("frequencies") if isinstance(details.get("frequencies"), list) else [],
        )

    def _normalize_ports(self, ports: list[Any]) -> list[str]:
        out: list[str] = []
        for entry in ports:
            if isinstance(entry, list) and len(entry) > 0:
                name = _norm_str(entry[0])
            else:
                name = _norm_str(entry)
            if name:
                out.append(name)
        return out

    def _normalize_frequencies(self, frequencies: list[Any]) -> list[tuple[str, str]]:
        out: list[tuple[str, str]] = []
        for entry in frequencies:
            if isinstance(entry, list) and len(entry) > 0:
                label = _norm_str(entry[0])
                hz = _norm_str(entry[1]) if len(entry) > 1 else ""
            else:
                label = _norm_str(entry)
                hz = ""
            if label:
                out.append((label, hz))
        return out

    # ---------- catalog reads ----------
    def get_transmitter_ports(self, system_code: str) -> list[dict[str, Any]]:
        return self._store.list_ports(SYSTEM_KIND_TRANSMITTER, system_code)

    def get_transmitter_frequencies(self, system_code: str) -> list[dict[str, Any]]:
        return self._store.list_frequencies(SYSTEM_KIND_TRANSMITTER, system_code)

    def get_system_ports(self, system_kind: str, system_code: str) -> list[dict[str, Any]]:
        kind = self._validate_system_kind(system_kind)
        ports = self._store.list_ports(kind, system_code)
        if ports:
            return ports
        self._ensure_system_catalog_from_stored_form(kind, system_code)
        return self._store.list_ports(kind, system_code)

    def get_system_frequencies(self, system_kind: str, system_code: str) -> list[dict[str, Any]]:
        kind = self._validate_system_kind(system_kind)
        frequencies = self._store.list_frequencies(kind, system_code)
        if frequencies:
            return frequencies
        self._ensure_system_catalog_from_stored_form(kind, system_code)
        return self._store.list_frequencies(kind, system_code)

    # ---------- catalog writes ----------
    def upsert_transmitter_port(
        self, system_code: str, port_name: str, sort_order: int = 0
    ) -> int:
        return self._store.upsert_port(
            SYSTEM_KIND_TRANSMITTER, system_code, port_name, sort_order
        )

    def upsert_transmitter_frequency(
        self,
        system_code: str,
        frequency_label: str,
        frequency_hz: str = "",
        sort_order: int = 0,
    ) -> int:
        return self._store.upsert_frequency(
            SYSTEM_KIND_TRANSMITTER,
            system_code,
            frequency_label,
            frequency_hz,
            sort_order,
        )

    def upsert_system_port(
        self,
        system_kind: str,
        system_code: str,
        port_name: str,
        sort_order: int = 0,
    ) -> int:
        kind = self._validate_system_kind(system_kind)
        return self._store.upsert_port(kind, system_code, port_name, sort_order)

    def upsert_system_frequency(
        self,
        system_kind: str,
        system_code: str,
        frequency_label: str,
        frequency_hz: str = "",
        sort_order: int = 0,
    ) -> int:
        kind = self._validate_system_kind(system_kind)
        return self._store.upsert_frequency(
            kind,
            system_code,
            frequency_label,
            frequency_hz,
            sort_order,
        )

    def rename_transmitter_port(self, port_id: int, new_name: str) -> None:
        self._store.rename_port(port_id, new_name)

    def rename_transmitter_frequency(
        self,
        frequency_id: int,
        new_label: Optional[str] = None,
        new_hz: Optional[str] = None,
    ) -> None:
        self._store.rename_frequency(frequency_id, new_label, new_hz)

    def delete_transmitter_port(self, port_id: int, force: bool = False) -> None:
        if not force:
            count = self._store.port_dependent_count(port_id)
            if count > 0:
                raise ValueError(
                    f"Port has {count} dependent rows; pass force=True to cascade-delete"
                )
        self._store.delete_port(port_id)

    def delete_transmitter_frequency(
        self, frequency_id: int, force: bool = False
    ) -> None:
        if not force:
            count = self._store.frequency_dependent_count(frequency_id)
            if count > 0:
                raise ValueError(
                    f"Frequency has {count} dependent rows; pass force=True to cascade-delete"
                )
        self._store.delete_frequency(frequency_id)

    # ---------- spec/loss/calibration reads ----------
    def get_spec_rows(
        self,
        system_kind: str = SYSTEM_KIND_TRANSMITTER,
        system_code: Optional[str] = None,
        parameter_type: Optional[str] = None,
    ) -> list[dict[str, Any]]:
        kind = self._validate_system_kind(system_kind)
        if kind == SYSTEM_KIND_RECEIVER and parameter_type == PARAMETER_TYPE_COMMAND_THRESHOLD:
            self.sync_receiver_command_threshold_rows(system_code)
        return self._store.list_spec_rows(kind, system_code, parameter_type)

    def sync_receiver_command_threshold_rows(
        self,
        system_code: Optional[str] = None,
    ) -> list[dict[str, Any]]:
        receiver_query: dict[str, Any] = {"system_type": "Receiver"}
        if system_code is not None:
            receiver_query["code"] = system_code

        receivers = self._transmitters.find(query=receiver_query)
        receiver_codes = {
            _norm_str(receiver.get("code"))
            for receiver in receivers
            if _norm_str(receiver.get("code"))
        }

        existing_rows = self._store.list_spec_rows(
            SYSTEM_KIND_RECEIVER,
            system_code,
            PARAMETER_TYPE_COMMAND_THRESHOLD,
        )
        existing_by_key: dict[tuple[str, int, int], dict[str, Any]] = {}
        for row in existing_rows:
            key = (
                _norm_str(row.get("system_code")),
                int(row.get("port_id") or 0),
                int(row.get("frequency_id") or 0),
            )
            existing_by_key[key] = row

        valid_keys: set[tuple[str, int, int]] = set()

        for receiver in receivers:
            code = _norm_str(receiver.get("code"))
            if not code:
                continue

            ports = self.get_system_ports(SYSTEM_KIND_RECEIVER, code)
            frequencies = self.get_system_frequencies(SYSTEM_KIND_RECEIVER, code)

            sort_index = 0
            for port in ports:
                port_id = int(port.get("port_id") or 0)
                if not port_id:
                    continue
                for frequency in frequencies:
                    frequency_id = int(frequency.get("frequency_id") or 0)
                    if not frequency_id:
                        continue

                    key = (code, port_id, frequency_id)
                    valid_keys.add(key)
                    existing = existing_by_key.get(key)
                    payload = {
                        "max_input_power": -60,
                        "specification": None,
                        "tolerance": 0.5,
                        "fbt": None,
                        "fbt_hot": None,
                        "fbt_cold": None,
                    }
                    if existing is not None and isinstance(existing.get("payload"), dict):
                        payload.update(existing["payload"])

                    self._store.upsert_spec_row(
                        SYSTEM_KIND_RECEIVER,
                        code,
                        PARAMETER_TYPE_COMMAND_THRESHOLD,
                        port_id,
                        frequency_id,
                        payload,
                        sort_order=sort_index,
                    )
                    sort_index += 1

        for row in existing_rows:
            code = _norm_str(row.get("system_code"))
            key = (
                code,
                int(row.get("port_id") or 0),
                int(row.get("frequency_id") or 0),
            )
            if code not in receiver_codes or key not in valid_keys:
                self._store.delete_spec_row(int(row["id"]))

        return self._store.list_spec_rows(
            SYSTEM_KIND_RECEIVER,
            system_code,
            PARAMETER_TYPE_COMMAND_THRESHOLD,
        )

    def get_loss_rows(
        self,
        system_kind: str = SYSTEM_KIND_TRANSMITTER,
        system_code: Optional[str] = None,
    ) -> list[dict[str, Any]]:
        kind = self._validate_system_kind(system_kind)
        return self._store.list_loss_rows_for_kind(kind, system_code)

    def get_calibration_rows(
        self,
        system_kind: str = SYSTEM_KIND_TRANSMITTER,
        system_code: Optional[str] = None,
    ) -> list[dict[str, Any]]:
        kind = self._validate_system_kind(system_kind)
        return self._store.list_calibration_rows_for_kind(kind, system_code)

    def get_test_plan_rows(
        self,
        system_kind: str,
        system_code: Optional[str] = None,
    ) -> list[dict[str, Any]]:
        kind = self._validate_system_kind(system_kind)
        return self._store.list_test_plan_rows(kind, system_code)

    def get_profile_rows(
        self,
        system_kind: str,
        system_code: Optional[str] = None,
    ) -> list[dict[str, Any]]:
        kind = self._validate_system_kind(system_kind)
        return self._store.list_profile_rows(kind, system_code)

    # ---------- spec/loss/calibration writes ----------
    def upsert_spec_row(
        self,
        system_kind: str,
        system_code: str,
        parameter_type: str,
        port_id: int,
        frequency_id: int,
        payload: dict[str, Any],
        sort_order: int = 0,
    ) -> int:
        kind = self._validate_system_kind(system_kind)
        if parameter_type not in ALL_PARAMETER_TYPES:
            raise ValueError(f"Unsupported parameter_type: {parameter_type}")
        return self._store.upsert_spec_row(
            kind,
            system_code,
            parameter_type,
            port_id,
            frequency_id,
            payload,
            sort_order,
        )

    def upsert_loss_row(
        self,
        system_kind: str,
        system_code: str,
        port_id: int,
        frequency_id: int,
        loss_db: Optional[float],
        payload: Optional[dict[str, Any]] = None,
        sort_order: int = 0,
    ) -> int:
        kind = self._validate_system_kind(system_kind)
        return self._store.upsert_loss_row(
            kind,
            system_code,
            port_id,
            frequency_id,
            loss_db,
            payload,
            sort_order,
        )

    def upsert_calibration_row(
        self,
        system_kind: str,
        system_code: str,
        port_id: int,
        frequency_id: int,
        payload: dict[str, Any],
        sort_order: int = 0,
    ) -> int:
        kind = self._validate_system_kind(system_kind)
        return self._store.upsert_calibration_row(
            kind,
            system_code,
            port_id,
            frequency_id,
            payload,
            sort_order,
        )

    def upsert_test_plan_row(
        self,
        system_kind: str,
        system_code: str,
        plan_name: str,
        payload: dict[str, Any],
        port_id: Optional[int] = None,
        frequency_id: Optional[int] = None,
        sort_order: int = 0,
    ) -> int:
        kind = self._validate_system_kind(system_kind)
        return self._store.upsert_test_plan_row(
            kind,
            system_code,
            plan_name,
            payload,
            port_id,
            frequency_id,
            sort_order,
        )

    def upsert_profile_row(
        self,
        system_kind: str,
        system_code: str,
        profile_name: str,
        payload: dict[str, Any],
        sort_order: int = 0,
    ) -> int:
        kind = self._validate_system_kind(system_kind)
        return self._store.upsert_profile_row(
            kind,
            system_code,
            profile_name,
            payload,
            sort_order,
        )

    def delete_spec_row(self, row_id: int) -> None:
        self._store.delete_spec_row(row_id)

    # ---------- system-level rename / delete ----------
    def rename_transmitter_system_code(self, old_code: str, new_code: str) -> None:
        self._store.rename_system_code(SYSTEM_KIND_TRANSMITTER, old_code, new_code)

    def delete_transmitter_system(self, system_code: str) -> None:
        self._store.delete_system(SYSTEM_KIND_TRANSMITTER, system_code)

    def rename_system_code(self, system_kind: str, old_code: str, new_code: str) -> None:
        kind = self._validate_system_kind(system_kind)
        self._store.rename_system_code(kind, old_code, new_code)

    def delete_system(self, system_kind: str, system_code: str) -> None:
        kind = self._validate_system_kind(system_kind)
        self._store.delete_system(kind, system_code)

    def sync_system_catalog_from_form(
        self,
        system_kind: str,
        system_code: str,
        ports: list[Any],
        frequencies: list[Any],
    ) -> dict[str, int]:
        """
        Sync source-of-truth form arrays into system catalog while preserving IDs.

        Strategy:
        - Match rows by index first: rename in place when names changed.
        - Update sort order.
        - Delete extra trailing catalog rows (cascades dependent data).
        - Insert missing new rows.
        """
        kind = self._validate_system_kind(system_kind)
        normalized_ports = self._normalize_ports(ports)
        normalized_freqs = self._normalize_frequencies(frequencies)

        existing_ports = self._store.list_ports(kind, system_code)
        existing_freqs = self._store.list_frequencies(kind, system_code)

        renamed_ports = 0
        inserted_ports = 0
        deleted_ports = 0
        renamed_freqs = 0
        inserted_freqs = 0
        deleted_freqs = 0

        max_ports = max(len(existing_ports), len(normalized_ports))
        for index in range(max_ports):
            has_existing = index < len(existing_ports)
            has_target = index < len(normalized_ports)

            if has_existing and has_target:
                row = existing_ports[index]
                target_name = normalized_ports[index]
                if str(row.get("port_name") or "") != target_name:
                    self._store.rename_port(int(row["port_id"]), target_name)
                    renamed_ports += 1
                self._store.update_port_sort_order(int(row["port_id"]), index)
                continue

            if has_existing and not has_target:
                self._store.delete_port(int(existing_ports[index]["port_id"]))
                deleted_ports += 1
                continue

            if has_target and not has_existing:
                self._store.upsert_port(kind, system_code, normalized_ports[index], index)
                inserted_ports += 1

        max_freqs = max(len(existing_freqs), len(normalized_freqs))
        for index in range(max_freqs):
            has_existing = index < len(existing_freqs)
            has_target = index < len(normalized_freqs)

            if has_existing and has_target:
                row = existing_freqs[index]
                target_label, target_hz = normalized_freqs[index]
                if str(row.get("frequency_label") or "") != target_label:
                    self._store.rename_frequency(int(row["frequency_id"]), new_label=target_label)
                    renamed_freqs += 1
                if str(row.get("frequency_hz") or "") != target_hz:
                    self._store.rename_frequency(int(row["frequency_id"]), new_hz=target_hz)
                self._store.update_frequency_sort_order(int(row["frequency_id"]), index)
                continue

            if has_existing and not has_target:
                self._store.delete_frequency(int(existing_freqs[index]["frequency_id"]))
                deleted_freqs += 1
                continue

            if has_target and not has_existing:
                label, hz = normalized_freqs[index]
                self._store.upsert_frequency(kind, system_code, label, hz, index)
                inserted_freqs += 1

        return {
            "renamed_ports": renamed_ports,
            "inserted_ports": inserted_ports,
            "deleted_ports": deleted_ports,
            "renamed_frequencies": renamed_freqs,
            "inserted_frequencies": inserted_freqs,
            "deleted_frequencies": deleted_freqs,
        }

    # ---------- migration from legacy transmitter JSON ----------
    def migrate_from_legacy_transmitters(self, force: bool = False) -> dict[str, int]:
        """
        Populate catalog + row tables from the existing
        `transmitters.modulation_details` JSON layout.

        Idempotent: each row is upsert-keyed on (system_code, port_id, frequency_id)
        so re-running is safe. Skipped automatically if the named migration has
        already run, unless force=True.
        """
        migration_name = "transmitter_json_v1"
        if not force and self._store.has_migration(migration_name):
            return {
                "skipped": 1,
                "transmitters": 0,
                "ports": 0,
                "frequencies": 0,
                "spec_rows": 0,
                "loss_rows": 0,
            }

        stats = {
            "skipped": 0,
            "transmitters": 0,
            "ports": 0,
            "frequencies": 0,
            "spec_rows": 0,
            "loss_rows": 0,
        }

        cursor = self._transmitters.find(
            {"system_type": "Transmitter"},
            {
                "_id": 0,
                "code": 1,
                "modulation_details": 1,
            },
        )

        for doc in cursor:
            code = _norm_str(doc.get("code"))
            if not code:
                continue
            details = doc.get("modulation_details") or {}
            if not isinstance(details, dict):
                continue

            stats["transmitters"] += 1

            # Build port catalog from the transmitter's ports list.
            port_id_by_name: dict[str, int] = {}
            for index, entry in enumerate(details.get("ports") or []):
                if isinstance(entry, list) and len(entry) > 0:
                    port_name = _norm_str(entry[0])
                else:
                    port_name = _norm_str(entry)
                if not port_name:
                    continue
                if port_name in port_id_by_name:
                    continue
                pid = self._store.upsert_port(
                    SYSTEM_KIND_TRANSMITTER, code, port_name, sort_order=index
                )
                port_id_by_name[port_name] = pid
                stats["ports"] += 1

            # Build frequency catalog from the transmitter's frequencies list.
            freq_id_by_label: dict[str, int] = {}
            for index, entry in enumerate(details.get("frequencies") or []):
                if not isinstance(entry, list) or len(entry) < 1:
                    continue
                label = _norm_str(entry[0])
                if not label:
                    continue
                hz = _norm_str(entry[1]) if len(entry) > 1 else ""
                if label in freq_id_by_label:
                    continue
                fid = self._store.upsert_frequency(
                    SYSTEM_KIND_TRANSMITTER,
                    code,
                    label,
                    hz,
                    sort_order=index,
                )
                freq_id_by_label[label] = fid
                stats["frequencies"] += 1

            # Helper to resolve port/freq IDs with on-the-fly catalog inserts so
            # spec/loss rows referencing labels not in the explicit ports/freqs
            # arrays still migrate cleanly.
            def resolve_port(name: str) -> Optional[int]:
                name = _norm_str(name)
                if not name:
                    return None
                pid = port_id_by_name.get(name)
                if pid is not None:
                    return pid
                pid = self._store.upsert_port(
                    SYSTEM_KIND_TRANSMITTER,
                    code,
                    name,
                    sort_order=len(port_id_by_name),
                )
                port_id_by_name[name] = pid
                stats["ports"] += 1
                return pid

            def resolve_freq(label: str, hz: str = "") -> Optional[int]:
                label = _norm_str(label)
                if not label:
                    return None
                fid = freq_id_by_label.get(label)
                if fid is not None:
                    if hz:
                        # Backfill the Hz value if it was missing on first insert.
                        self._store.rename_frequency(fid, new_hz=hz)
                    return fid
                fid = self._store.upsert_frequency(
                    SYSTEM_KIND_TRANSMITTER,
                    code,
                    label,
                    hz,
                    sort_order=len(freq_id_by_label),
                )
                freq_id_by_label[label] = fid
                stats["frequencies"] += 1
                return fid

            # Migrate spec rows.
            for details_key, parameter_type in _DETAILS_KEY_TO_PARAMETER.items():
                rows = details.get(details_key)
                if not isinstance(rows, list):
                    continue
                for index, row in enumerate(rows):
                    if not isinstance(row, dict):
                        continue
                    port_id = resolve_port(row.get("port", ""))
                    freq_id = resolve_freq(
                        row.get("frequency_label", ""),
                        _norm_str(row.get("frequency", "")),
                    )
                    if port_id is None or freq_id is None:
                        continue
                    payload = _split_payload(row)
                    self._store.upsert_spec_row(
                        system_kind=SYSTEM_KIND_TRANSMITTER,
                        system_code=code,
                        parameter_type=parameter_type,
                        port_id=port_id,
                        frequency_id=freq_id,
                        payload=payload,
                        sort_order=index,
                    )
                    stats["spec_rows"] += 1

            # Migrate on-board loss rows.
            loss_rows = details.get("on_board_loss_specs")
            if isinstance(loss_rows, list):
                for index, row in enumerate(loss_rows):
                    if not isinstance(row, dict):
                        continue
                    port_id = resolve_port(row.get("port", ""))
                    freq_id = resolve_freq(
                        row.get("frequency_label", ""),
                        _norm_str(row.get("frequency", "")),
                    )
                    if port_id is None or freq_id is None:
                        continue
                    payload = _split_payload(row)
                    loss_db_raw = payload.pop("loss_db", None)
                    try:
                        loss_db: Optional[float] = (
                            None if loss_db_raw in (None, "") else float(loss_db_raw)
                        )
                    except (TypeError, ValueError):
                        loss_db = None
                    self._store.upsert_loss_row(
                        system_kind=SYSTEM_KIND_TRANSMITTER,
                        system_code=code,
                        port_id=port_id,
                        frequency_id=freq_id,
                        loss_db=loss_db,
                        payload=payload,
                        sort_order=index,
                    )
                    stats["loss_rows"] += 1

        self._store.mark_migration(migration_name)
        return stats

    # ---------- ranging tones ----------
    def get_ranging_tones(self) -> list[dict[str, Any]]:
        return self._store.list_ranging_tones()

    # ---------- ranging threshold ----------
    def sync_transponder_ranging_threshold_rows(self) -> None:
        """Auto-populate ranging_threshold_rows for every transponder × tone combination."""
        tones = self._store.list_ranging_tones()
        if not tones:
            return

        transponders = self._transmitters.find(
            {"system_type": "Transponder"},
            {"_id": 0, "code": 1, "name": 1},
        )
        transponder_codes: list[str] = [
            _norm_str(t.get("code"))
            for t in transponders
            if _norm_str(t.get("code"))
        ]

        # Also pull the project-transponder rows to get uplink/downlink info.
        from src.database.connection import Database
        project_tp_col = Database.get_collection("ProjectTransponders")
        project_tp_rows: list[dict[str, Any]] = project_tp_col.find({}, {"_id": 0})
        uplink_by_code: dict[str, str] = {}
        downlink_by_code: dict[str, str] = {}
        for row in project_tp_rows:
            code = _norm_str(row.get("Code") or row.get("code"))
            if not code:
                continue
            rx_code = _norm_str(row.get("RxCode") or row.get("rx_code"))
            rx_port = _norm_str(row.get("RxPort") or row.get("rx_port"))
            rx_freq = _norm_str(row.get("RxFreq") or row.get("rx_freq"))
            tx_code = _norm_str(row.get("TxCode") or row.get("tx_code"))
            tx_port = _norm_str(row.get("TxPort") or row.get("tx_port"))
            tx_freq = _norm_str(row.get("TxFreq") or row.get("tx_freq"))
            uplink_by_code[code] = "_".join(x for x in [rx_code, rx_port, rx_freq] if x)
            downlink_by_code[code] = "_".join(x for x in [tx_code, tx_port, tx_freq] if x)
            # Register transponder code even if not in system catalog
            if code not in transponder_codes:
                transponder_codes.append(code)

        existing_rows = self._store.list_ranging_threshold_rows()
        existing_map: dict[tuple[str, int], dict[str, Any]] = {
            (_norm_str(r["transponder_code"]), int(r["tone_id"])): r
            for r in existing_rows
        }

        valid_keys: set[tuple[str, int]] = set()
        sort_index = 0
        for code in transponder_codes:
            uplink = uplink_by_code.get(code, "")
            downlink = downlink_by_code.get(code, "")
            for tone in tones:
                tone_id = int(tone["id"])
                key = (code, tone_id)
                valid_keys.add(key)
                existing = existing_map.get(key)
                self._store.upsert_ranging_threshold_row(
                    transponder_code=code,
                    tone_id=tone_id,
                    uplink=uplink if uplink else (existing["uplink"] if existing else ""),
                    downlink=downlink if downlink else (existing["downlink"] if existing else ""),
                    max_input_power=existing["max_input_power"] if existing is not None else -60,
                    specification=existing["specification"] if existing is not None else None,
                    tolerance=existing["tolerance"] if existing is not None else None,
                    fbt=existing["fbt"] if existing is not None else None,
                    fbt_hot=existing["fbt_hot"] if existing is not None else None,
                    fbt_cold=existing["fbt_cold"] if existing is not None else None,
                    sort_order=sort_index,
                )
                sort_index += 1

        # Remove stale rows (transponder no longer exists)
        for row in existing_rows:
            key = (_norm_str(row["transponder_code"]), int(row["tone_id"]))
            if key not in valid_keys:
                self._store.delete_ranging_threshold_row(int(row["id"]))

    def get_ranging_threshold_rows(self) -> list[dict[str, Any]]:
        self.sync_transponder_ranging_threshold_rows()
        return self._store.list_ranging_threshold_rows()

    def upsert_ranging_threshold_row(
        self,
        transponder_code: str,
        tone_id: int,
        uplink: str,
        downlink: str,
        max_input_power: Optional[float],
        specification: Optional[float],
        tolerance: Optional[float],
        fbt: Any,
        fbt_hot: Any,
        fbt_cold: Any,
        sort_order: int = 0,
    ) -> int:
        return self._store.upsert_ranging_threshold_row(
            transponder_code=transponder_code,
            tone_id=tone_id,
            uplink=uplink,
            downlink=downlink,
            max_input_power=max_input_power,
            specification=specification,
            tolerance=tolerance,
            fbt=fbt,
            fbt_hot=fbt_hot,
            fbt_cold=fbt_cold,
            sort_order=sort_order,
        )

    # ---------- onboard losses ----------
    def sync_onboard_losses(self, source_type: str) -> None:
        """
        Sync onboard_losses rows from the transmitter/receiver catalog.

        For each system of the given source_type, fetch its ports and frequencies
        from system_ports / system_frequencies and upsert one row per port×frequency.
        Stale rows (deleted systems / ports / frequencies) are removed.
        """
        kind = source_type  # 'transmitter' or 'receiver'
        system_type_label = kind.title()  # 'Transmitter' or 'Receiver'

        systems = self._transmitters.find(query={"system_type": system_type_label})
        valid_keys: set[tuple[str, str, str]] = set()
        sort_index = 0
        for system in systems:
            code = _norm_str(system.get("code"))
            if not code:
                continue
            ports = self.get_system_ports(kind, code)
            frequencies = self.get_system_frequencies(kind, code)
            for port in ports:
                port_name = _norm_str(port.get("port_name"))
                for freq in frequencies:
                    freq_label = _norm_str(freq.get("frequency_label"))
                    frequency_hz = _norm_str(freq.get("frequency_hz"))
                    key = (code, port_name, freq_label)
                    valid_keys.add(key)
                    self._store.upsert_onboard_loss(
                        source_type=kind,
                        code=code,
                        port=port_name,
                        frequency=frequency_hz,
                        freq_label=freq_label,
                        loss_db=None,  # preserve existing value handled by upsert
                        sort_order=sort_index,
                    )
                    sort_index += 1

        # Remove stale rows
        existing = self._store.list_onboard_losses(kind)
        for row in existing:
            key = (
                _norm_str(row.get("code")),
                _norm_str(row.get("port")),
                _norm_str(row.get("freq_label")),
            )
            if key not in valid_keys:
                self._store.delete_onboard_loss(int(row["id"]))

    def get_onboard_losses(self, source_type: str) -> list[dict[str, Any]]:
        self.sync_onboard_losses(source_type)
        return self._store.list_onboard_losses(source_type)

    def save_onboard_losses(self, rows: list[dict[str, Any]]) -> int:
        updated = 0
        for row in rows:
            row_id = row.get("id")
            if row_id is None:
                continue
            loss_raw = row.get("loss_db")
            try:
                loss_db: Optional[float] = None if loss_raw in (None, "") else float(loss_raw)
            except (TypeError, ValueError):
                loss_db = None
            self._store.update_onboard_loss_db(int(row_id), loss_db)
            updated += 1
        return updated
