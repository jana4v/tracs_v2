from typing import List
from fastapi import APIRouter, Depends, HTTPException, status

from src.database.connection import (
    get_transmitters_collection,
    get_instruments_collection,
    get_project_instruments_collection,
    get_project_power_meters_collection,
    get_project_tsm_paths_collection,
    get_configuration_collection,
    get_transmitter_misc_collection,
)
from src.database.sqlite_json_store import SQLiteJsonCollection
from src.repositories.test_systems_repo import TestSystemsRepository
from src.repositories.transmitter_repo import TransmitterRepository
from src.schemas.enums import ModulationType
from src.schemas.test_systems import (
    InstrumentCatalogResponse,
    ProjectInstrumentsResponse,
    ProjectInstrumentsSaveRequest,
    ProjectInstrumentsSaveResponse,
    ProjectPowerMetersResponse,
    ProjectPowerMetersSaveRequest,
    ProjectPowerMetersSaveResponse,
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
    configuration_collection: SQLiteJsonCollection = Depends(get_configuration_collection),
) -> TestSystemsRepository:
    return TestSystemsRepository(
        transmitters_collection=transmitters_collection,
        instruments_collection=instruments_collection,
        project_instruments_collection=project_instruments_collection,
        project_power_meters_collection=project_power_meters_collection,
        project_tsm_paths_collection=project_tsm_paths_collection,
        configuration_collection=configuration_collection,
    )


@router.get("/modulation-types", response_model=List[str])
def get_modulation_types():
    """Return the list of supported modulation types."""
    return [m.value for m in ModulationType]


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
