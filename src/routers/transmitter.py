from typing import List, Any
from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel, Field

from src.config import settings
from src.database.connection import (
    get_transmitters_collection,
    get_instruments_collection,
    get_project_instruments_collection,
    get_project_power_meters_collection,
    get_project_tsm_paths_collection,
    get_project_transponders_collection,
    get_configuration_collection,
    get_transmitter_misc_collection,
)
from src.database.sqlite_json_store import SQLiteJsonCollection
from src.database.connection import Database
from src.repositories.test_systems_repo import TestSystemsRepository
from src.repositories.test_plan_types_repo import TestPlanTypesRepository
from src.repositories.transmitter_repo import TransmitterRepository
from src.repositories.downlink_cal_calibration_repo import DownlinkCalCalibrationRepository
from src.repositories.mod_index_measurement_repo import ModIndexMeasurementRepository
from src.schemas.enums import ModulationType
from src.schemas.test_systems import (
    InstrumentCatalogResponse,
    ProjectInstrumentsResponse,
    ProjectInstrumentsSaveRequest,
    ProjectInstrumentsSaveResponse,
    ProjectPowerMetersResponse,
    ProjectPowerMetersSaveRequest,
    ProjectPowerMetersSaveResponse,
    ProjectTranspondersResponse,
    ProjectTranspondersSaveRequest,
    ProjectTranspondersSaveResponse,
    TsmPathsResponse,
    TsmPathsSaveRequest,
    TsmPathsSaveResponse,
    ConfigurationValueResponse,
    ConfigurationValueSaveRequest,
)
from src.schemas.transmitter import (
    CalibrationRowsResponse,
    CalibrationRowsUpdateRequest,
    CalibrationRowsUpdateResponse,
    OnBoardLossRowsResponse,
    OnBoardLossRowsUpdateRequest,
    OnBoardLossRowsUpdateResponse,
    ParameterName,
    ParameterRowsResponse,
    ParameterRowsUpdateRequest,
    ParameterRowsUpdateResponse,
    SpuriousBandConfigResponse,
    SpuriousBandConfigSaveRequest,
    SpuriousBandConfigSaveResponse,
    TestProfileSpuriousRowsResponse,
    TestProfileSpuriousRowsUpdateRequest,
    TestProfileSpuriousRowsUpdateResponse,
    TransmitterCreate,
    TransmitterResponse,
)

router = APIRouter(prefix="/api/v2", tags=["Transmitters"])


def get_repo(
    collection: SQLiteJsonCollection = Depends(get_transmitters_collection),
    tsm_paths_collection: SQLiteJsonCollection = Depends(get_project_tsm_paths_collection),
    misc_collection: SQLiteJsonCollection = Depends(get_transmitter_misc_collection),
) -> TransmitterRepository:
    return TransmitterRepository(collection, tsm_paths_collection, misc_collection)


def get_test_systems_repo(
    transmitters_collection: SQLiteJsonCollection = Depends(get_transmitters_collection),
    instruments_collection: SQLiteJsonCollection = Depends(get_instruments_collection),
    project_instruments_collection: SQLiteJsonCollection = Depends(get_project_instruments_collection),
    project_power_meters_collection: SQLiteJsonCollection = Depends(get_project_power_meters_collection),
    project_tsm_paths_collection: SQLiteJsonCollection = Depends(get_project_tsm_paths_collection),
    project_transponders_collection: SQLiteJsonCollection = Depends(get_project_transponders_collection),
    configuration_collection: SQLiteJsonCollection = Depends(get_configuration_collection),
) -> TestSystemsRepository:
    return TestSystemsRepository(
        transmitters_collection=transmitters_collection,
        instruments_collection=instruments_collection,
        project_instruments_collection=project_instruments_collection,
        project_power_meters_collection=project_power_meters_collection,
        project_tsm_paths_collection=project_tsm_paths_collection,
        project_transponders_collection=project_transponders_collection,
        configuration_collection=configuration_collection,
    )


def get_test_plan_types_repo() -> TestPlanTypesRepository:
    from src.database.connection import Database

    return TestPlanTypesRepository(Database._db_path, settings.TEST_PLAN_TYPES_TABLE)


@router.get("/modulation-types", response_model=List[str])
def get_modulation_types():
    """Return the list of supported modulation types."""
    return [m.value for m in ModulationType]


@router.get("/test-plan-types", response_model=List[str])
def get_test_plan_types(
    repo: TestPlanTypesRepository = Depends(get_test_plan_types_repo),
):
    return repo.list_types()


@router.get("/transmitters", response_model=List[TransmitterResponse])
def list_transmitters(repo: TransmitterRepository = Depends(get_repo)):
    return repo.get_all()


@router.post("/transmitters", response_model=TransmitterResponse, status_code=status.HTTP_200_OK)
def save_transmitter(
    payload: TransmitterCreate,
    repo: TransmitterRepository = Depends(get_repo),
):
    """Create or update a transmitter (upsert by code)."""
    return repo.upsert(payload)


@router.get("/transmitters/parameters/{parameter}", response_model=ParameterRowsResponse)
def get_parameter_rows(
    parameter: ParameterName,
    repo: TransmitterRepository = Depends(get_repo),
):
    """Get parameter rows from all transmitters where the parameter is applicable."""
    rows = repo.get_parameter_rows(parameter)
    return ParameterRowsResponse(parameter=parameter, rows=rows)


@router.put("/transmitters/parameters/{parameter}", response_model=ParameterRowsUpdateResponse)
def update_parameter_rows(
    parameter: ParameterName,
    payload: ParameterRowsUpdateRequest,
    repo: TransmitterRepository = Depends(get_repo),
):
    """Update parameter rows grouped by transmitter and normalize values to dB."""
    summary = repo.update_parameter_rows(
        parameter=parameter,
        updates=[item.model_dump() for item in payload.rows],
    )
    return ParameterRowsUpdateResponse(
        parameter=parameter,
        updated_transmitters=summary["updated_transmitters"],
        updated_rows=summary["updated_rows"],
    )


@router.get("/transmitters/test-profile-spurious", response_model=TestProfileSpuriousRowsResponse)
def get_test_profile_spurious_rows(
    repo: TransmitterRepository = Depends(get_repo),
):
    rows = repo.get_test_profile_spurious_rows()
    return TestProfileSpuriousRowsResponse(rows=rows)


@router.put("/transmitters/test-profile-spurious", response_model=TestProfileSpuriousRowsUpdateResponse)
def update_test_profile_spurious_rows(
    payload: TestProfileSpuriousRowsUpdateRequest,
    repo: TransmitterRepository = Depends(get_repo),
):
    summary = repo.update_test_profile_spurious_rows(
        updates=[item.model_dump() for item in payload.rows],
    )
    return TestProfileSpuriousRowsUpdateResponse(
        updated_transmitters=summary["updated_transmitters"],
        updated_rows=summary["updated_rows"],
    )


@router.get("/transmitters/on-board-losses", response_model=OnBoardLossRowsResponse)
def get_on_board_loss_rows(
    repo: TransmitterRepository = Depends(get_repo),
):
    rows = repo.get_on_board_loss_rows()
    return OnBoardLossRowsResponse(rows=rows)


@router.put("/transmitters/on-board-losses", response_model=OnBoardLossRowsUpdateResponse)
def update_on_board_loss_rows(
    payload: OnBoardLossRowsUpdateRequest,
    repo: TransmitterRepository = Depends(get_repo),
):
    summary = repo.update_on_board_loss_rows(
        updates=[item.model_dump() for item in payload.rows],
    )
    return OnBoardLossRowsUpdateResponse(
        updated_transmitters=summary["updated_transmitters"],
        updated_rows=summary["updated_rows"],
    )


@router.get("/transmitters/calibration", response_model=CalibrationRowsResponse)
def get_calibration_rows(
    repo: TransmitterRepository = Depends(get_repo),
):
    rows = repo.get_calibration_rows()
    return CalibrationRowsResponse(rows=rows)


@router.put("/transmitters/calibration", response_model=CalibrationRowsUpdateResponse)
def update_calibration_rows(
    payload: CalibrationRowsUpdateRequest,
    repo: TransmitterRepository = Depends(get_repo),
):
    summary = repo.update_calibration_rows(
        updates=[item.model_dump() for item in payload.rows],
    )
    return CalibrationRowsUpdateResponse(
        updated_transmitters=summary["updated_transmitters"],
        updated_rows=summary["updated_rows"],
    )


@router.delete("/transmitters/{code}", status_code=status.HTTP_200_OK)
def delete_transmitter(
    code: str,
    repo: TransmitterRepository = Depends(get_repo),
):
    deleted = repo.delete(code)
    if not deleted:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Transmitter with code '{code}' not found.",
        )
    # Cascade: drop any DownlinkCalCalibrationData rows tied to this code so
    # they don't become orphans (the transmitter document is the source of
    # truth for code/port/frequency).
    try:
        downlink_repo = DownlinkCalCalibrationRepository(
            Database._db_path, settings.DOWNLINK_CAL_CALIBRATION_TABLE
        )
        downlink_repo.delete_for_code(code)
    except Exception:
        pass
    # Cascade: drop ModIndexMeasurement rows for this transmitter.
    try:
        mod_index_repo = ModIndexMeasurementRepository(
            Database._db_path, settings.MOD_INDEX_MEASUREMENT_TABLE
        )
        mod_index_repo.delete_for_code(code, system_kind="transmitter")
    except Exception:
        pass
    return {"message": f"Transmitter '{code}' deleted successfully."}


@router.get("/test-systems/instruments/catalog", response_model=InstrumentCatalogResponse)
def get_instrument_catalog(
    repo: TestSystemsRepository = Depends(get_test_systems_repo),
):
    return InstrumentCatalogResponse(instruments=repo.get_instrument_catalog())


@router.get("/test-systems/project-instruments", response_model=ProjectInstrumentsResponse)
def get_project_instruments(
    repo: TestSystemsRepository = Depends(get_test_systems_repo),
):
    rows = repo.get_project_instruments_rows()
    return ProjectInstrumentsResponse(rows=rows)


@router.put("/test-systems/project-instruments", response_model=ProjectInstrumentsSaveResponse)
def save_project_instruments(
    payload: ProjectInstrumentsSaveRequest,
    repo: TestSystemsRepository = Depends(get_test_systems_repo),
):
    saved_rows = repo.save_project_instruments_rows([r.model_dump() for r in payload.rows])
    return ProjectInstrumentsSaveResponse(saved_rows=saved_rows)


@router.get("/test-systems/project-power-meters", response_model=ProjectPowerMetersResponse)
def get_project_power_meters(
    repo: TestSystemsRepository = Depends(get_test_systems_repo),
):
    rows = repo.get_project_power_meter_rows()
    return ProjectPowerMetersResponse(rows=rows)


@router.put("/test-systems/project-power-meters", response_model=ProjectPowerMetersSaveResponse)
def save_project_power_meters(
    payload: ProjectPowerMetersSaveRequest,
    repo: TestSystemsRepository = Depends(get_test_systems_repo),
):
    saved_rows = repo.save_project_power_meter_rows([r.model_dump() for r in payload.rows])
    return ProjectPowerMetersSaveResponse(saved_rows=saved_rows)


@router.get("/test-systems/project-tsm-paths", response_model=TsmPathsResponse)
def get_project_tsm_paths(
    repo: TestSystemsRepository = Depends(get_test_systems_repo),
):
    rows = repo.get_project_tsm_path_rows()
    return TsmPathsResponse(rows=rows)


@router.put("/test-systems/project-tsm-paths", response_model=TsmPathsSaveResponse)
def save_project_tsm_paths(
    payload: TsmPathsSaveRequest,
    repo: TestSystemsRepository = Depends(get_test_systems_repo),
):
    saved_rows = repo.save_project_tsm_path_rows([r.model_dump() for r in payload.rows])
    return TsmPathsSaveResponse(saved_rows=saved_rows)


@router.get("/test-systems/project-transponders", response_model=ProjectTranspondersResponse)
def get_project_transponders(
    repo: TestSystemsRepository = Depends(get_test_systems_repo),
):
    rows = repo.get_project_transponder_rows()
    return ProjectTranspondersResponse(rows=rows)


@router.put("/test-systems/project-transponders", response_model=ProjectTranspondersSaveResponse)
def save_project_transponders(
    payload: ProjectTranspondersSaveRequest,
    repo: TestSystemsRepository = Depends(get_test_systems_repo),
):
    saved_rows = repo.save_project_transponder_rows([r.model_dump() for r in payload.rows])
    return ProjectTranspondersSaveResponse(saved_rows=saved_rows)


@router.get("/test-systems/configuration/{parameter}", response_model=ConfigurationValueResponse)
def get_configuration_value(
    parameter: str,
    repo: TestSystemsRepository = Depends(get_test_systems_repo),
):
    value = repo.get_configuration_value(parameter)
    return ConfigurationValueResponse(parameter=parameter, value=value)


@router.put("/test-systems/configuration/{parameter}", response_model=ConfigurationValueResponse)
def save_configuration_value(
    parameter: str,
    payload: ConfigurationValueSaveRequest,
    repo: TestSystemsRepository = Depends(get_test_systems_repo),
):
    repo.set_configuration_value(parameter, payload.value)
    return ConfigurationValueResponse(parameter=parameter, value=payload.value)


@router.get("/spurious-bands", response_model=SpuriousBandConfigResponse)
def get_spurious_band_configs(repo: TransmitterRepository = Depends(get_repo)):
    """Get standalone spurious search band configurations."""
    return SpuriousBandConfigResponse(bands=repo.get_spurious_band_configs())


@router.put("/spurious-bands", response_model=SpuriousBandConfigSaveResponse)
def save_spurious_band_configs(
    payload: SpuriousBandConfigSaveRequest,
    repo: TransmitterRepository = Depends(get_repo),
):
    """Save standalone spurious search band configurations."""
    saved_rows = repo.save_spurious_band_configs([b.model_dump() for b in payload.bands])
    return SpuriousBandConfigSaveResponse(saved_rows=saved_rows)


# ── Transponder Test Profiles ─────────────────────────────────────────────────

class TransponderTestProfileRow(BaseModel):
    profile_name: str = Field(default="")
    levels: List[List[Any]] = Field(default_factory=lambda: [
        [-60], [-70], [-80], [-90], [-100], [-105]
    ])
    mod_index_at_threshold: float = Field(default=0.62)
    tones: List[str] = Field(default_factory=list)


class TransponderTestProfileResponse(BaseModel):
    profile_type: str
    rows: List[TransponderTestProfileRow]


class TransponderTestProfileSaveRequest(BaseModel):
    profile_type: str
    rows: List[TransponderTestProfileRow]


class TransponderTestProfileSaveResponse(BaseModel):
    profile_type: str
    saved_rows: int


def get_transponder_test_profiles_repo():
    from src.database.connection import Database
    from src.repositories.receiver_test_profiles_repo import ReceiverTestProfilesRepository
    return ReceiverTestProfilesRepository(Database._db_path, settings.TRANSPONDER_TEST_PROFILES_TABLE)


@router.get("/transponder-test-profiles/{profile_type}", response_model=TransponderTestProfileResponse)
def get_transponder_test_profile(
    profile_type: str,
    repo=Depends(get_transponder_test_profiles_repo),
):
    from src.database.connection import Database
    from src.database.system_catalog import SystemCatalogStore
    plan_types_repo = TestPlanTypesRepository(Database._db_path, settings.TEST_PLAN_TYPES_TABLE)
    plan_types = plan_types_repo.list_types()

    catalog_store = SystemCatalogStore(Database._db_path)
    all_tones: List[str] = [t["tone_khz"] for t in catalog_store.list_ranging_tones()]

    saved = repo.get_profile(profile_type)
    saved_by_name: dict = {r.get("profile_name", ""): r for r in saved if isinstance(r, dict)}

    default_levels = [[-60], [-70], [-80], [-90], [-100], [-105]]

    rows: list[TransponderTestProfileRow] = []
    for plan_name in plan_types:
        if plan_name in saved_by_name:
            s = saved_by_name[plan_name]
            saved_tones = s.get("tones", [])
            rows.append(TransponderTestProfileRow(
                profile_name=plan_name,
                levels=s.get("levels", default_levels),
                mod_index_at_threshold=float(s.get("mod_index_at_threshold", 0.62)),
                tones=saved_tones if saved_tones else all_tones,
            ))
        else:
            rows.append(TransponderTestProfileRow(
                profile_name=plan_name,
                levels=default_levels,
                mod_index_at_threshold=0.62,
                tones=all_tones,
            ))

    return TransponderTestProfileResponse(profile_type=profile_type, rows=rows)


@router.put("/transponder-test-profiles", response_model=TransponderTestProfileSaveResponse)
def save_transponder_test_profile(
    payload: TransponderTestProfileSaveRequest,
    repo=Depends(get_transponder_test_profiles_repo),
):
    rows_data = [row.model_dump() for row in payload.rows]
    repo.save_profile(payload.profile_type, rows_data)
    return TransponderTestProfileSaveResponse(
        profile_type=payload.profile_type,
        saved_rows=len(rows_data),
    )


# ── Test Plan Selections (per-system-kind, per-test-plan-type) ───────────────


class TestPlanSelectionRow(BaseModel):
    """A single saved checkbox row for a system within a test plan.

    For transmitter / receiver kinds, identifying fields are code/port/frequency_label.
    For transponder kind, identifying fields are transponder_code/uplink/downlink.
    Either set may be provided; missing fields are stored as empty strings.
    """
    code: str = ""
    port: str = ""
    frequency_label: str = ""
    transponder_code: str = ""
    uplink: str = ""
    downlink: str = ""
    params: dict = Field(default_factory=dict)


class TestPlanSelectionsResponse(BaseModel):
    system_kind: str
    test_plan_name: str
    rows: List[TestPlanSelectionRow]


class TestPlanSelectionsSaveRequest(BaseModel):
    test_plan_name: str
    rows: List[TestPlanSelectionRow]


class TestPlanSelectionsSaveResponse(BaseModel):
    system_kind: str
    test_plan_name: str
    saved_rows: int


def get_test_plan_selections_repo():
    from src.database.connection import Database
    from src.repositories.test_plan_selections_repo import TestPlanSelectionsRepository
    return TestPlanSelectionsRepository(Database._db_path, settings.TEST_PLAN_SELECTIONS_TABLE)


def _validate_system_kind(system_kind: str) -> str:
    kind = (system_kind or "").strip().lower()
    if kind not in ("transmitter", "receiver", "transponder"):
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Invalid system_kind '{system_kind}'. Must be one of: transmitter, receiver, transponder.",
        )
    return kind


@router.get(
    "/test-plan/selections/{system_kind}/{test_plan_name}",
    response_model=TestPlanSelectionsResponse,
)
def get_test_plan_selections(
    system_kind: str,
    test_plan_name: str,
    repo=Depends(get_test_plan_selections_repo),
):
    kind = _validate_system_kind(system_kind)
    saved = repo.get_selections(kind, test_plan_name)
    rows = [TestPlanSelectionRow(**(r if isinstance(r, dict) else {})) for r in saved]
    return TestPlanSelectionsResponse(
        system_kind=kind,
        test_plan_name=test_plan_name,
        rows=rows,
    )


@router.put(
    "/test-plan/selections/{system_kind}",
    response_model=TestPlanSelectionsSaveResponse,
)
def save_test_plan_selections(
    system_kind: str,
    payload: TestPlanSelectionsSaveRequest,
    repo=Depends(get_test_plan_selections_repo),
):
    kind = _validate_system_kind(system_kind)
    rows_data = [row.model_dump() for row in payload.rows]
    repo.save_selections(kind, payload.test_plan_name, rows_data)
    return TestPlanSelectionsSaveResponse(
        system_kind=kind,
        test_plan_name=payload.test_plan_name,
        saved_rows=len(rows_data),
    )

