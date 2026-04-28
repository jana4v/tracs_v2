"""
System catalog router — additive REST API on top of the new normalized schema.

Endpoints
---------
- GET    /api/v2/system-catalog/transmitters/{code}/ports
- POST   /api/v2/system-catalog/transmitters/{code}/ports
- PATCH  /api/v2/system-catalog/ports/{port_id}
- DELETE /api/v2/system-catalog/ports/{port_id}

- GET    /api/v2/system-catalog/transmitters/{code}/frequencies
- POST   /api/v2/system-catalog/transmitters/{code}/frequencies
- PATCH  /api/v2/system-catalog/frequencies/{frequency_id}
- DELETE /api/v2/system-catalog/frequencies/{frequency_id}

- GET    /api/v2/system-catalog/transmitters/{code}/spec-rows
- POST   /api/v2/system-catalog/transmitters/{code}/spec-rows
- DELETE /api/v2/system-catalog/spec-rows/{row_id}

- GET    /api/v2/system-catalog/transmitters/{code}/loss-rows
- POST   /api/v2/system-catalog/transmitters/{code}/loss-rows

- GET    /api/v2/system-catalog/transmitters/{code}/calibration-rows
- POST   /api/v2/system-catalog/transmitters/{code}/calibration-rows

- POST   /api/v2/system-catalog/migrate  (one-shot JSON -> tables migration)

These endpoints are deliberately additive. The legacy
`/api/v2/transmitters/...` JSON-backed endpoints continue to work unchanged.
"""

from typing import Any, Optional

from fastapi import APIRouter, Body, Depends, HTTPException, Query, status
from pydantic import BaseModel, Field

from src.database.connection import get_transmitters_collection
from src.database.sqlite_json_store import SQLiteJsonCollection
from src.database.system_catalog import (
    ALL_PARAMETER_TYPES,
    ALL_SYSTEM_KINDS,
    PARAMETER_TYPE_COMMAND_THRESHOLD,
    SYSTEM_KIND_RECEIVER,
    SYSTEM_KIND_TRANSMITTER,
    SystemCatalogStore,
    get_catalog_store,
)
from src.repositories.system_catalog_repo import SystemCatalogRepository


router = APIRouter(prefix="/api/v2/system-catalog", tags=["SystemCatalog"])


def _get_store() -> SystemCatalogStore:
    return get_catalog_store()


def get_catalog_repo(
    transmitters: SQLiteJsonCollection = Depends(get_transmitters_collection),
    store: SystemCatalogStore = Depends(_get_store),
) -> SystemCatalogRepository:
    return SystemCatalogRepository(store, transmitters)


# ---------- request/response models ----------
class PortCreate(BaseModel):
    port_name: str = Field(..., min_length=1)
    sort_order: int = 0


class PortUpdate(BaseModel):
    port_name: str = Field(..., min_length=1)


class FrequencyCreate(BaseModel):
    frequency_label: str = Field(..., min_length=1)
    frequency_hz: str = ""
    sort_order: int = 0


class FrequencyUpdate(BaseModel):
    frequency_label: Optional[str] = None
    frequency_hz: Optional[str] = None


class SpecRowCreate(BaseModel):
    parameter_type: str
    port_id: int
    frequency_id: int
    payload: dict[str, Any] = Field(default_factory=dict)
    sort_order: int = 0


class LossRowCreate(BaseModel):
    port_id: int
    frequency_id: int
    loss_db: Optional[float] = None
    payload: dict[str, Any] = Field(default_factory=dict)
    sort_order: int = 0


class CalibrationRowCreate(BaseModel):
    port_id: int
    frequency_id: int
    payload: dict[str, Any] = Field(default_factory=dict)
    sort_order: int = 0


class TestPlanRowCreate(BaseModel):
    plan_name: str = Field(..., min_length=1)
    port_id: Optional[int] = None
    frequency_id: Optional[int] = None
    payload: dict[str, Any] = Field(default_factory=dict)
    sort_order: int = 0


class ProfileRowCreate(BaseModel):
    profile_name: str = Field(..., min_length=1)
    payload: dict[str, Any] = Field(default_factory=dict)
    sort_order: int = 0


def _validate_kind(system_kind: str) -> str:
    kind = str(system_kind or "").strip().lower()
    if kind not in ALL_SYSTEM_KINDS:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"system_kind must be one of {ALL_SYSTEM_KINDS}",
        )
    return kind


# ---------- ports ----------
@router.get("/transmitters/{code}/ports")
def list_transmitter_ports(
    code: str,
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> list[dict[str, Any]]:
    return repo.get_transmitter_ports(code)


@router.get("/{system_kind}/{code}/ports")
def list_system_ports(
    system_kind: str,
    code: str,
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> list[dict[str, Any]]:
    kind = _validate_kind(system_kind)
    return repo.get_system_ports(kind, code)


@router.post("/transmitters/{code}/ports", status_code=status.HTTP_201_CREATED)
def create_transmitter_port(
    code: str,
    body: PortCreate,
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> dict[str, Any]:
    port_id = repo.upsert_transmitter_port(code, body.port_name, body.sort_order)
    return {"port_id": port_id, "system_code": code, "port_name": body.port_name}


@router.post("/{system_kind}/{code}/ports", status_code=status.HTTP_201_CREATED)
def create_system_port(
    system_kind: str,
    code: str,
    body: PortCreate,
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> dict[str, Any]:
    kind = _validate_kind(system_kind)
    port_id = repo.upsert_system_port(kind, code, body.port_name, body.sort_order)
    return {
        "port_id": port_id,
        "system_kind": kind,
        "system_code": code,
        "port_name": body.port_name,
    }


@router.patch("/ports/{port_id}")
def rename_port(
    port_id: int,
    body: PortUpdate,
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> dict[str, Any]:
    repo.rename_transmitter_port(port_id, body.port_name)
    return {"port_id": port_id, "port_name": body.port_name}


@router.delete("/ports/{port_id}")
def delete_port(
    port_id: int,
    force: bool = Query(False),
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> dict[str, Any]:
    try:
        repo.delete_transmitter_port(port_id, force=force)
    except ValueError as exc:
        raise HTTPException(
            status_code=status.HTTP_409_CONFLICT, detail=str(exc)
        ) from exc
    return {"port_id": port_id, "deleted": True}


# ---------- frequencies ----------
@router.get("/transmitters/{code}/frequencies")
def list_transmitter_frequencies(
    code: str,
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> list[dict[str, Any]]:
    return repo.get_transmitter_frequencies(code)


@router.get("/{system_kind}/{code}/frequencies")
def list_system_frequencies(
    system_kind: str,
    code: str,
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> list[dict[str, Any]]:
    kind = _validate_kind(system_kind)
    return repo.get_system_frequencies(kind, code)


@router.post(
    "/transmitters/{code}/frequencies", status_code=status.HTTP_201_CREATED
)
def create_transmitter_frequency(
    code: str,
    body: FrequencyCreate,
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> dict[str, Any]:
    fid = repo.upsert_transmitter_frequency(
        code, body.frequency_label, body.frequency_hz, body.sort_order
    )
    return {
        "frequency_id": fid,
        "system_code": code,
        "frequency_label": body.frequency_label,
        "frequency_hz": body.frequency_hz,
    }


@router.post("/{system_kind}/{code}/frequencies", status_code=status.HTTP_201_CREATED)
def create_system_frequency(
    system_kind: str,
    code: str,
    body: FrequencyCreate,
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> dict[str, Any]:
    kind = _validate_kind(system_kind)
    fid = repo.upsert_system_frequency(
        kind,
        code,
        body.frequency_label,
        body.frequency_hz,
        body.sort_order,
    )
    return {
        "frequency_id": fid,
        "system_kind": kind,
        "system_code": code,
        "frequency_label": body.frequency_label,
        "frequency_hz": body.frequency_hz,
    }


@router.patch("/frequencies/{frequency_id}")
def rename_frequency(
    frequency_id: int,
    body: FrequencyUpdate,
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> dict[str, Any]:
    if body.frequency_label is None and body.frequency_hz is None:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="At least one of frequency_label or frequency_hz must be provided",
        )
    repo.rename_transmitter_frequency(
        frequency_id, body.frequency_label, body.frequency_hz
    )
    return {
        "frequency_id": frequency_id,
        "frequency_label": body.frequency_label,
        "frequency_hz": body.frequency_hz,
    }


@router.delete("/frequencies/{frequency_id}")
def delete_frequency(
    frequency_id: int,
    force: bool = Query(False),
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> dict[str, Any]:
    try:
        repo.delete_transmitter_frequency(frequency_id, force=force)
    except ValueError as exc:
        raise HTTPException(
            status_code=status.HTTP_409_CONFLICT, detail=str(exc)
        ) from exc
    return {"frequency_id": frequency_id, "deleted": True}


# ---------- spec rows ----------
@router.get("/transmitters/{code}/spec-rows")
def list_spec_rows(
    code: str,
    parameter_type: Optional[str] = Query(None),
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> list[dict[str, Any]]:
    if parameter_type is not None and parameter_type not in ALL_PARAMETER_TYPES:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"parameter_type must be one of {ALL_PARAMETER_TYPES}",
        )
    if parameter_type == PARAMETER_TYPE_COMMAND_THRESHOLD:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="command_threshold is receiver data; use /api/v2/system-catalog/receivers/spec-rows or /api/v2/system-catalog/receiver/{code}/spec-rows",
        )
    return repo.get_spec_rows(system_code=code, parameter_type=parameter_type)


@router.get("/transmitters/spec-rows")
def list_transmitter_spec_rows_all(
    parameter_type: Optional[str] = Query(None),
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> list[dict[str, Any]]:
    if parameter_type is not None and parameter_type not in ALL_PARAMETER_TYPES:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"parameter_type must be one of {ALL_PARAMETER_TYPES}",
        )
    if parameter_type == PARAMETER_TYPE_COMMAND_THRESHOLD:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="command_threshold is receiver data; use /api/v2/system-catalog/receivers/spec-rows",
        )
    return repo.get_spec_rows(
        system_kind=SYSTEM_KIND_TRANSMITTER,
        system_code=None,
        parameter_type=parameter_type,
    )


@router.get("/receivers/spec-rows")
def list_receiver_spec_rows_all(
    parameter_type: Optional[str] = Query(None),
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> list[dict[str, Any]]:
    if parameter_type is not None and parameter_type not in ALL_PARAMETER_TYPES:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"parameter_type must be one of {ALL_PARAMETER_TYPES}",
        )
    return repo.get_spec_rows(
        system_kind=SYSTEM_KIND_RECEIVER,
        system_code=None,
        parameter_type=parameter_type,
    )


@router.get("/{system_kind}/{code}/spec-rows")
def list_system_spec_rows(
    system_kind: str,
    code: str,
    parameter_type: Optional[str] = Query(None),
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> list[dict[str, Any]]:
    kind = _validate_kind(system_kind)
    if parameter_type is not None and parameter_type not in ALL_PARAMETER_TYPES:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"parameter_type must be one of {ALL_PARAMETER_TYPES}",
        )
    return repo.get_spec_rows(
        system_kind=kind,
        system_code=code,
        parameter_type=parameter_type,
    )


@router.post("/transmitters/{code}/spec-rows", status_code=status.HTTP_201_CREATED)
def upsert_spec_row(
    code: str,
    body: SpecRowCreate,
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> dict[str, Any]:
    if body.parameter_type not in ALL_PARAMETER_TYPES:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"parameter_type must be one of {ALL_PARAMETER_TYPES}",
        )
    row_id = repo.upsert_spec_row(
        system_kind=SYSTEM_KIND_TRANSMITTER,
        system_code=code,
        parameter_type=body.parameter_type,
        port_id=body.port_id,
        frequency_id=body.frequency_id,
        payload=body.payload,
        sort_order=body.sort_order,
    )
    return {"id": row_id}


@router.post("/{system_kind}/{code}/spec-rows", status_code=status.HTTP_201_CREATED)
def upsert_system_spec_row(
    system_kind: str,
    code: str,
    body: SpecRowCreate,
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> dict[str, Any]:
    kind = _validate_kind(system_kind)
    if body.parameter_type not in ALL_PARAMETER_TYPES:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"parameter_type must be one of {ALL_PARAMETER_TYPES}",
        )
    row_id = repo.upsert_spec_row(
        system_kind=kind,
        system_code=code,
        parameter_type=body.parameter_type,
        port_id=body.port_id,
        frequency_id=body.frequency_id,
        payload=body.payload,
        sort_order=body.sort_order,
    )
    return {"id": row_id}


@router.delete("/spec-rows/{row_id}")
def delete_spec_row(
    row_id: int,
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> dict[str, Any]:
    repo.delete_spec_row(row_id)
    return {"id": row_id, "deleted": True}


# ---------- loss rows ----------
@router.get("/transmitters/{code}/loss-rows")
def list_loss_rows(
    code: str,
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> list[dict[str, Any]]:
    return repo.get_loss_rows(system_kind=SYSTEM_KIND_TRANSMITTER, system_code=code)


@router.get("/transmitters/loss-rows")
def list_transmitter_loss_rows_all(
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> list[dict[str, Any]]:
    return repo.get_loss_rows(system_kind=SYSTEM_KIND_TRANSMITTER, system_code=None)


@router.get("/{system_kind}/{code}/loss-rows")
def list_system_loss_rows(
    system_kind: str,
    code: str,
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> list[dict[str, Any]]:
    kind = _validate_kind(system_kind)
    return repo.get_loss_rows(system_kind=kind, system_code=code)


@router.post("/transmitters/{code}/loss-rows", status_code=status.HTTP_201_CREATED)
def upsert_loss_row(
    code: str,
    body: LossRowCreate,
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> dict[str, Any]:
    row_id = repo.upsert_loss_row(
        system_kind=SYSTEM_KIND_TRANSMITTER,
        system_code=code,
        port_id=body.port_id,
        frequency_id=body.frequency_id,
        loss_db=body.loss_db,
        payload=body.payload,
        sort_order=body.sort_order,
    )
    return {"id": row_id}


@router.post("/{system_kind}/{code}/loss-rows", status_code=status.HTTP_201_CREATED)
def upsert_system_loss_row(
    system_kind: str,
    code: str,
    body: LossRowCreate,
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> dict[str, Any]:
    kind = _validate_kind(system_kind)
    row_id = repo.upsert_loss_row(
        system_kind=kind,
        system_code=code,
        port_id=body.port_id,
        frequency_id=body.frequency_id,
        loss_db=body.loss_db,
        payload=body.payload,
        sort_order=body.sort_order,
    )
    return {"id": row_id}


# ---------- calibration rows ----------
@router.get("/transmitters/{code}/calibration-rows")
def list_calibration_rows(
    code: str,
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> list[dict[str, Any]]:
    return repo.get_calibration_rows(system_kind=SYSTEM_KIND_TRANSMITTER, system_code=code)


@router.get("/{system_kind}/{code}/calibration-rows")
def list_system_calibration_rows(
    system_kind: str,
    code: str,
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> list[dict[str, Any]]:
    kind = _validate_kind(system_kind)
    return repo.get_calibration_rows(system_kind=kind, system_code=code)


@router.post(
    "/transmitters/{code}/calibration-rows", status_code=status.HTTP_201_CREATED
)
def upsert_calibration_row(
    code: str,
    body: CalibrationRowCreate,
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> dict[str, Any]:
    row_id = repo.upsert_calibration_row(
        system_kind=SYSTEM_KIND_TRANSMITTER,
        system_code=code,
        port_id=body.port_id,
        frequency_id=body.frequency_id,
        payload=body.payload,
        sort_order=body.sort_order,
    )
    return {"id": row_id}


@router.post("/{system_kind}/{code}/calibration-rows", status_code=status.HTTP_201_CREATED)
def upsert_system_calibration_row(
    system_kind: str,
    code: str,
    body: CalibrationRowCreate,
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> dict[str, Any]:
    kind = _validate_kind(system_kind)
    row_id = repo.upsert_calibration_row(
        system_kind=kind,
        system_code=code,
        port_id=body.port_id,
        frequency_id=body.frequency_id,
        payload=body.payload,
        sort_order=body.sort_order,
    )
    return {"id": row_id}


@router.get("/{system_kind}/{code}/test-plan-rows")
def list_system_test_plan_rows(
    system_kind: str,
    code: str,
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> list[dict[str, Any]]:
    kind = _validate_kind(system_kind)
    return repo.get_test_plan_rows(kind, code)


@router.post("/{system_kind}/{code}/test-plan-rows", status_code=status.HTTP_201_CREATED)
def upsert_system_test_plan_row(
    system_kind: str,
    code: str,
    body: TestPlanRowCreate,
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> dict[str, Any]:
    kind = _validate_kind(system_kind)
    row_id = repo.upsert_test_plan_row(
        system_kind=kind,
        system_code=code,
        plan_name=body.plan_name,
        port_id=body.port_id,
        frequency_id=body.frequency_id,
        payload=body.payload,
        sort_order=body.sort_order,
    )
    return {"id": row_id}


@router.get("/{system_kind}/{code}/profile-rows")
def list_system_profile_rows(
    system_kind: str,
    code: str,
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> list[dict[str, Any]]:
    kind = _validate_kind(system_kind)
    return repo.get_profile_rows(kind, code)


@router.post("/{system_kind}/{code}/profile-rows", status_code=status.HTTP_201_CREATED)
def upsert_system_profile_row(
    system_kind: str,
    code: str,
    body: ProfileRowCreate,
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> dict[str, Any]:
    kind = _validate_kind(system_kind)
    row_id = repo.upsert_profile_row(
        system_kind=kind,
        system_code=code,
        profile_name=body.profile_name,
        payload=body.payload,
        sort_order=body.sort_order,
    )
    return {"id": row_id}


# ---------- migration ----------
@router.post("/migrate")
def run_migration(
    force: bool = Body(False, embed=True),
    repo: SystemCatalogRepository = Depends(get_catalog_repo),
) -> dict[str, int]:
    """Populate the catalog and row tables from legacy transmitter JSON."""
    return repo.migrate_from_legacy_transmitters(force=force)
