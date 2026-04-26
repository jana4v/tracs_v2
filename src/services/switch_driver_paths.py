from __future__ import annotations

import asyncio
import re
from typing import Any, cast

from src.InstrumentApi.Models import InstrumentAddress, InstrumentTypes
from src.InstrumentApi.SwitchDriver.BaseClass import SwitchDriver
from src.InstrumentApi.factory import get_instrument


MODEL_ALIASES = {
    "E4419B": "AG_4419B",
    "N1914A": "AG_N1914A",
    "ML4238": "AS_ML4238",
}

PATH_COLUMNS = ("path1", "path2", "path3", "path4", "path5", "path6")
SWITCH_DRIVER_COMMAND_DELAY_SECONDS = 1.0


def normalize_model(model: str) -> str:
    clean = str(model or "").strip()
    return MODEL_ALIASES.get(clean, clean)


def build_address(raw_address: str) -> InstrumentAddress:
    address = str(raw_address or "").strip()
    if not address:
        raise ValueError("Instrument address is empty")

    if ":" in address:
        parts = address.split(":", 1)
        if parts[0].strip().isdigit() and parts[1].strip().isdigit():
            return InstrumentAddress(
                ip_or_gpib_address=int(parts[1].strip()),
                port_or_gpib_bus=int(parts[0].strip()),
            )

    gpib_resource = re.fullmatch(r"(?i)GPIB\s*(\d+)\s*::\s*(\d+)\s*::\s*INSTR", address)
    if gpib_resource:
        return InstrumentAddress(
            ip_or_gpib_address=int(gpib_resource.group(2)),
            port_or_gpib_bus=int(gpib_resource.group(1)),
        )

    if address.isdigit():
        return InstrumentAddress(ip_or_gpib_address=int(address), port_or_gpib_bus=0)

    return InstrumentAddress(ip_or_gpib_address=address, port_or_gpib_bus=0)


def build_switch_driver(config: dict[str, str]) -> SwitchDriver:
    model = normalize_model(config.get("model", ""))
    if model == "":
        raise ValueError("Switch driver model is empty")

    address = build_address(config.get("address", ""))
    instrument = cast(SwitchDriver | None, get_instrument(InstrumentTypes.SwitchDriver, model, address))
    if instrument is None:
        raise ValueError(f"Unsupported switch driver model: {config.get('model', '')}")
    return instrument


def collect_switch_driver_commands(tsm_path_rows: list[dict[str, Any]]) -> dict[str, list[str]]:
    commands_by_driver: dict[str, list[str]] = {}

    for row in tsm_path_rows:
        if not isinstance(row, dict):
            continue

        for column in PATH_COLUMNS:
            raw_value = row.get(column, row.get(column.capitalize()))
            if raw_value is None:
                continue

            for chunk in str(raw_value).split(";"):
                command = chunk.strip()
                if command == "":
                    continue

                match = re.match(r"(?i)^(D\d+)", command)
                if match is None:
                    raise ValueError(f"Invalid switch path command: {command}")

                driver_key = match.group(1).upper()
                commands_by_driver.setdefault(driver_key, []).append(command.upper())

    return commands_by_driver


def build_switch_driver_map(project_instruments_rows: list[dict[str, Any]]) -> dict[str, SwitchDriver]:
    drivers: dict[str, SwitchDriver] = {}

    for row in project_instruments_rows:
        if not isinstance(row, dict):
            continue

        instrument_name = str(row.get("instrument_name") or row.get("InstrumentName") or "").strip()
        match = re.fullmatch(r"(?i)SDU\s*(\d+)", instrument_name)
        if match is None:
            continue

        address_main = str(row.get("address_main") or row.get("AddressMain") or "").strip()
        address_redt = str(row.get("address_redt") or row.get("AddressRedt") or "").strip()
        use_redt_value = row.get("use_redt", row.get("UseRedt", False))
        use_redt = False
        if isinstance(use_redt_value, bool):
            use_redt = use_redt_value
        elif isinstance(use_redt_value, (int, float)):
            use_redt = int(use_redt_value) != 0
        else:
            use_redt = str(use_redt_value).strip().lower() in {"1", "true", "yes", "on"}

        selected_address = address_redt if use_redt and address_redt != "" else address_main

        driver_key = f"D{int(match.group(1))}"
        drivers[driver_key] = build_switch_driver(
            {
                "model": str(row.get("model") or row.get("Model") or ""),
                "address": selected_address,
            }
        )

    return drivers


async def apply_switch_driver_paths(
    tsm_path_rows: list[dict[str, Any]],
    project_instruments_rows: list[dict[str, Any]],
) -> dict[str, list[str]]:
    commands_by_driver = collect_switch_driver_commands(tsm_path_rows)
    if len(commands_by_driver) == 0:
        return {}

    drivers = build_switch_driver_map(project_instruments_rows)
    if len(drivers) == 0:
        raise ValueError("No SDU switch drivers are configured in Project Instruments")

    total_commands = sum(len(commands) for commands in commands_by_driver.values())
    dispatched_commands = 0

    for driver_key, commands in commands_by_driver.items():
        driver = drivers.get(driver_key)
        if driver is None:
            raise ValueError(f"Switch path references {driver_key}, but matching {driver_key.replace('D', 'SDU')} is not configured")
        for command in commands:
            try:
                await driver.set_switch_position(command)
            except Exception as exc:
                raise ValueError(f"Failed to apply switch path command {command} on {driver_key}: {exc}") from exc
            dispatched_commands += 1
            if dispatched_commands < total_commands:
                await asyncio.sleep(SWITCH_DRIVER_COMMAND_DELAY_SECONDS)

    return commands_by_driver