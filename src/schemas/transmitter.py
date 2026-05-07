from typing import Annotated, Any, Literal
from pydantic import BaseModel, Field, field_validator
from src.schemas.enums import SystemType, ModulationType
from src.schemas.transmitter_test_parameters import (
    CalibrationSpecRow,
    FrequencySpecRow,
    ModulationIndexSpecRow,
    OnBoardLossSpecRow,
    PowerSpecRow,
    SpuriousBandConfigRow,
    SpuriousSpecRow,
    TestProfileSpuriousSpecRow,
)


# ── Modulation-specific detail schemas ───────────────────────────────────────

class BaseModulationDetails(BaseModel):
    """DL ports, sub-carriers (kHz), and labelled frequency pairs (MHz)."""
    ports: list[list[str]] = Field(
        default=[["EV"], ["AEV"], ["GLOBAL"]],
        description="Each inner list is a single-element row: [[port_name], ...]",
    )
    sub_carriers: list[list[float]] = Field(
        default=[[32], [128]],
        description="Each inner list is a single-element row: [[kHz], ...]",
    )
    frequencies: list[list[str]] = Field(
        default=[["DF", ""], ["F1", ""], ["F2", ""]],
        description="Each inner list is [label, frequency_mhz_as_string]",
    )

    @field_validator("ports", mode="before")
    @classmethod
    def normalize_ports(cls, value: Any) -> list[list[str]]:
        if not isinstance(value, list):
            return [["EV"], ["AEV"], ["GLOBAL"]]
        normalized: list[list[str]] = []
        for row in value:
            if isinstance(row, list) and len(row) > 0:
                port = "" if row[0] is None else str(row[0]).strip()
            else:
                port = "" if row is None else str(row).strip()
            if port:
                normalized.append([port])
        return normalized or [["EV"], ["AEV"], ["GLOBAL"]]

    @field_validator("frequencies", mode="before")
    @classmethod
    def normalize_frequencies(cls, value: Any) -> list[list[str]]:
        if not isinstance(value, list):
            return [["DF", ""], ["F1", ""], ["F2", ""]]
        normalized: list[list[str]] = []
        for row in value:
            if isinstance(row, list):
                label = "" if len(row) < 1 or row[0] is None else str(row[0]).strip()
                frequency = "" if len(row) < 2 or row[1] is None else str(row[1]).strip()
            else:
                label = "" if row is None else str(row).strip()
                frequency = ""
            if label:
                normalized.append([label, frequency])
        return normalized or [["DF", ""], ["F1", ""], ["F2", ""]]


class PskPmDetails(BaseModulationDetails):
    """PSK_PM-specific modulation details and test parameters."""

    power_specs: list[PowerSpecRow] = Field(default_factory=list)
    frequency_specs: list[FrequencySpecRow] = Field(default_factory=list)
    modulation_index_specs: list[ModulationIndexSpecRow] = Field(default_factory=list)
    spurious_specs: list[SpuriousSpecRow] = Field(default_factory=list)
    test_profile_spurious_specs: list[TestProfileSpuriousSpecRow] = Field(default_factory=list)
    on_board_loss_specs: list[OnBoardLossSpecRow] = Field(default_factory=list)
    calibration_specs: list[CalibrationSpecRow] = Field(default_factory=list)


class PskFmDetails(BaseModulationDetails):
    """PSK_FM-specific modulation details.

    Add strong typed fields here as PSK_FM test parameters are finalized.
    """

    test_parameters: dict[str, Any] = Field(
        default_factory=dict,
        description="PSK_FM-specific test parameters keyed by parameter name.",
    )


# ── Discriminated union for modulation details ────────────────────────────────

class BaseTransmitter(BaseModel):
    name: str = Field(..., min_length=3, description="Human-readable transmitter name")
    code: str = Field(..., min_length=3, pattern=r"^[a-zA-Z][a-zA-Z0-9]+$", description="Short unique identifier, no spaces")
    system_type: SystemType = SystemType.Transmitter


class PskPmTransmitterCreate(BaseTransmitter):
    modulation_type: Literal[ModulationType.PSK_PM] = ModulationType.PSK_PM
    modulation_details: PskPmDetails = Field(default_factory=PskPmDetails)


class PskFmTransmitterCreate(BaseTransmitter):
    modulation_type: Literal[ModulationType.PSK_FM] = ModulationType.PSK_FM
    modulation_details: PskFmDetails = Field(default_factory=PskFmDetails)


TransmitterCreate = Annotated[
    PskPmTransmitterCreate | PskFmTransmitterCreate,
    Field(discriminator="modulation_type"),
]

TransmitterResponse = TransmitterCreate


class TransmitterListResponse(BaseModel):
    transmitters: list[TransmitterResponse]
    total: int


# ── Parameter-centric API schemas ─────────────────────────────────────────────

ParameterName = Literal["power", "frequency", "modulation_index", "spurious"]


class ParameterRowView(BaseModel):
    transmitter_code: str
    transmitter_name: str | None = None
    modulation_type: ModulationType
    row: dict[str, Any]


class ParameterRowsResponse(BaseModel):
    parameter: ParameterName
    unit: str = "dB"
    rows: list[ParameterRowView]


class ParameterRowUpdate(BaseModel):
    transmitter_code: str
    row: dict[str, Any]


class ParameterRowsUpdateRequest(BaseModel):
    rows: list[ParameterRowUpdate]


class ParameterRowsUpdateResponse(BaseModel):
    parameter: ParameterName
    unit: str = "dB"
    updated_transmitters: int
    updated_rows: int


class TestProfileSpuriousRowView(BaseModel):
    transmitter_code: str
    transmitter_name: str | None = None
    modulation_type: ModulationType
    row: TestProfileSpuriousSpecRow


class TestProfileSpuriousRowsResponse(BaseModel):
    rows: list[TestProfileSpuriousRowView]


class TestProfileSpuriousRowUpdate(BaseModel):
    transmitter_code: str
    row: TestProfileSpuriousSpecRow


class TestProfileSpuriousRowsUpdateRequest(BaseModel):
    rows: list[TestProfileSpuriousRowUpdate]


class TestProfileSpuriousRowsUpdateResponse(BaseModel):
    updated_transmitters: int
    updated_rows: int


class OnBoardLossRowView(BaseModel):
    transmitter_code: str
    transmitter_name: str | None = None
    modulation_type: ModulationType
    row: dict[str, Any]


class OnBoardLossRowsResponse(BaseModel):
    unit: str = "dB"
    rows: list[OnBoardLossRowView]


class OnBoardLossRowUpdate(BaseModel):
    transmitter_code: str
    row: dict[str, Any]


class OnBoardLossRowsUpdateRequest(BaseModel):
    rows: list[OnBoardLossRowUpdate]


class OnBoardLossRowsUpdateResponse(BaseModel):
    unit: str = "dB"
    updated_transmitters: int
    updated_rows: int


class CalibrationRowView(BaseModel):
    transmitter_code: str
    transmitter_name: str | None = None
    modulation_type: ModulationType
    row: dict[str, Any]


class CalibrationRowsResponse(BaseModel):
    unit: str = "dB"
    rows: list[CalibrationRowView]


class CalibrationRowUpdate(BaseModel):
    transmitter_code: str
    row: dict[str, Any]


class CalibrationRowsUpdateRequest(BaseModel):
    rows: list[CalibrationRowUpdate]


class CalibrationRowsUpdateResponse(BaseModel):
    unit: str = "dB"
    updated_transmitters: int
    updated_rows: int


class SpuriousBandConfigResponse(BaseModel):
    bands: list[SpuriousBandConfigRow]


class SpuriousBandConfigSaveRequest(BaseModel):
    bands: list[SpuriousBandConfigRow]


class SpuriousBandConfigSaveResponse(BaseModel):
    saved_rows: int
