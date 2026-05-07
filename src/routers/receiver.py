from typing import Any, List

from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel, Field

from src.config import settings
from src.database.connection import (
    get_project_tsm_paths_collection,
    get_transmitter_misc_collection,
    get_transmitters_collection,
)
from src.database.connection import Database
from src.database.sqlite_json_store import SQLiteJsonCollection
from src.repositories.receiver_repo import ReceiverRepository
from src.repositories.receiver_test_profiles_repo import ReceiverTestProfilesRepository
from src.repositories.downlink_cal_calibration_repo import DownlinkCalCalibrationRepository
from src.repositories.mod_index_measurement_repo import ModIndexMeasurementRepository
from src.schemas.receiver import ReceiverCreate, ReceiverResponse


# ── Receiver Test Profiles schemas ────────────────────────────────────────────

class ReceiverTestProfileRow(BaseModel):
    profile_name: str = Field(default="")
    levels: List[List[Any]] = Field(default_factory=lambda: [
        [-60, 10], [-70, 10], [-80, 10], [-90, 10], [-100, 10], [-105, 10]
    ])
    establish: bool = Field(default=True)
    no_of_cmds_at_threshold: int = Field(default=500)


class ReceiverTestProfileResponse(BaseModel):
    profile_type: str
    rows: List[ReceiverTestProfileRow]


class ReceiverTestProfileSaveRequest(BaseModel):
    profile_type: str
    rows: List[ReceiverTestProfileRow]


class ReceiverTestProfileSaveResponse(BaseModel):
    profile_type: str
    saved_rows: int


# ── Dependency ─────────────────────────────────────────────────────────────────

def get_test_profiles_repo() -> ReceiverTestProfilesRepository:
    return ReceiverTestProfilesRepository(Database._db_path, settings.RECEIVER_TEST_PROFILES_TABLE)


router = APIRouter(prefix='/api/v2', tags=['Receivers'])


def get_repo(
    collection: SQLiteJsonCollection = Depends(get_transmitters_collection),
    tsm_paths_collection: SQLiteJsonCollection = Depends(get_project_tsm_paths_collection),
    misc_collection: SQLiteJsonCollection = Depends(get_transmitter_misc_collection),
) -> ReceiverRepository:
    return ReceiverRepository(collection, tsm_paths_collection, misc_collection)


@router.get('/receivers', response_model=List[ReceiverResponse])
def list_receivers(repo: ReceiverRepository = Depends(get_repo)):
    return repo.get_all()


@router.post('/receivers', response_model=ReceiverResponse, status_code=status.HTTP_200_OK)
def save_receiver(
    payload: ReceiverCreate,
    repo: ReceiverRepository = Depends(get_repo),
):
    """Create or update a receiver (upsert by code)."""
    return repo.upsert(payload)


@router.delete('/receivers/{code}', status_code=status.HTTP_200_OK)
def delete_receiver(
    code: str,
    repo: ReceiverRepository = Depends(get_repo),
):
    deleted = repo.delete(code)
    if not deleted:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Receiver with code '{code}' not found.",
        )
    # Cascade: drop derived calibration / measurement rows so they don't
    # become orphans. The receiver document is the source of truth for
    # code/port/frequency.
    try:
        downlink_repo = DownlinkCalCalibrationRepository(
            Database._db_path, settings.DOWNLINK_CAL_CALIBRATION_TABLE
        )
        downlink_repo.delete_for_code(code)
    except Exception:
        pass
    try:
        mod_index_repo = ModIndexMeasurementRepository(
            Database._db_path, settings.MOD_INDEX_MEASUREMENT_TABLE
        )
        mod_index_repo.delete_for_code(code, system_kind="receiver")
    except Exception:
        pass
    return {'message': f"Receiver '{code}' deleted successfully."}


# ── Receiver Test Profiles endpoints ─────────────────────────────────────────

@router.get('/receiver-test-profiles/{profile_type}', response_model=ReceiverTestProfileResponse)
def get_receiver_test_profile(
    profile_type: str,
    test_plan_types_names: list[str] | None = None,
    repo: ReceiverTestProfilesRepository = Depends(get_test_profiles_repo),
):
    """
    Return the saved receiver test profile rows for the given profile_type.
    If no data has been saved yet, auto-generate a default row per test-plan-type.
    The list of test plan type names is resolved on-demand from the TestPlanType table.
    """
    from src.repositories.test_plan_types_repo import TestPlanTypesRepository
    plan_types_repo = TestPlanTypesRepository(Database._db_path, settings.TEST_PLAN_TYPES_TABLE)
    plan_types = plan_types_repo.list_types()

    saved = repo.get_profile(profile_type)

    # Build a lookup by profile_name for any existing saved rows
    saved_by_name: dict[str, dict] = {r.get("profile_name", ""): r for r in saved if isinstance(r, dict)}

    default_levels = [[-60, 10], [-70, 10], [-80, 10], [-90, 10], [-100, 10], [-105, 10]]

    rows: list[ReceiverTestProfileRow] = []
    for plan_name in plan_types:
        if plan_name in saved_by_name:
            s = saved_by_name[plan_name]
            rows.append(ReceiverTestProfileRow(
                profile_name=plan_name,
                levels=s.get("levels", default_levels),
                establish=bool(s.get("establish", True)),
                no_of_cmds_at_threshold=int(s.get("no_of_cmds_at_threshold", 10)),
            ))
        else:
            rows.append(ReceiverTestProfileRow(
                profile_name=plan_name,
                levels=default_levels,
                establish=True,
                no_of_cmds_at_threshold=500,
            ))

    return ReceiverTestProfileResponse(profile_type=profile_type, rows=rows)


@router.put('/receiver-test-profiles', response_model=ReceiverTestProfileSaveResponse)
def save_receiver_test_profile(
    payload: ReceiverTestProfileSaveRequest,
    repo: ReceiverTestProfilesRepository = Depends(get_test_profiles_repo),
):
    rows_data = [row.model_dump() for row in payload.rows]
    repo.save_profile(payload.profile_type, rows_data)
    return ReceiverTestProfileSaveResponse(
        profile_type=payload.profile_type,
        saved_rows=len(rows_data),
    )
