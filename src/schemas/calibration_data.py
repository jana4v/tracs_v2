from datetime import datetime
from typing import List, Literal, Optional

from pydantic import BaseModel, Field


class CalibrationDataPoint(BaseModel):
    freq: float
    value: float


class CalibrationDataDocument(BaseModel):
    cal_id: str
    cal_type: str  # uplink | downlink | fixedpad | calSG | injectcal
    date_time: datetime
    port: str
    data: List[CalibrationDataPoint]


class CalIdsResponse(BaseModel):
    cal_ids: List[str]


class CalSgCompletedFrequenciesResponse(BaseModel):
    cal_id: str
    frequencies: List[float]


class CalSgDataRow(BaseModel):
    frequency: float
    value: Optional[float] = None
    sa_loss: Optional[float] = None
    dl_pm_loss: Optional[float] = None
    datetime: str


class CalSgDataRowsResponse(BaseModel):
    cal_id: str
    rows: List[CalSgDataRow]


class DownlinkCalDataRow(BaseModel):
    code: str
    port: str
    frequency: float
    frequency_label: str = ""
    value: float
    datetime: str


class DownlinkCalDataRowsResponse(BaseModel):
    cal_id: str
    rows: List[DownlinkCalDataRow]


class CalibrationChannel(BaseModel):
    code: str
    port: str
    frequency_label: str
    frequency: str


class CalibrationRunStartRequest(BaseModel):
    cal_id: str
    cal_type: str
    include_spurious_bands: bool | None = None
    cal_sg_level: float = 10.0
    channels: List[CalibrationChannel]


class CalibrationRunPromptResponseRequest(BaseModel):
    action: Literal["connected", "abort"]


class CalibrationSample(BaseModel):
    code: str
    port: str
    frequency_label: str
    frequency: str
    value: Optional[float] = None
    sa_loss: Optional[float] = None
    dl_pm_loss: Optional[float] = None
    timestamp: datetime


class CalibrationRunSnapshot(BaseModel):
    run_id: str
    cal_id: str
    cal_type: str
    state: str
    progress: float
    status_lines: List[str]
    prompt_message: Optional[str] = None
    prompt_channel: Optional[CalibrationChannel] = None
    samples: List[CalibrationSample] = Field(default_factory=list)
    channels: List[CalibrationChannel] = Field(default_factory=list)
    created_at: datetime
    updated_at: datetime
    operator_connected: bool = False


class CalibrationRunAbortResponse(BaseModel):
    run_id: str
    state: str
    message: str


class CalibrationReportGenerateRequest(BaseModel):
    cal_id: str
    cal_type: str


class CalibrationReportGenerateResponse(BaseModel):
    cal_id: str
    cal_type: str
    satellite_name: str
    results_directory: str
    calibration_data_directory: str
    pdf_path: str | None = None
    excel_path: str | None = None
    pdf_generated: bool
    excel_rows_appended: int
    message: str


class MeasureOptionsResponse(BaseModel):
    test_phases: List[str]
    cal_ids: List[str]
    test_plan_types: List[str] = []
    default_cal_id: str | None = None
    default_test_plan_type: str | None = None


class MeasureTableRow(BaseModel):
    code: str
    port: str
    frequency_label: str
    frequency: str
    power_selected: bool = False
    frequency_selected: bool = False
    modulation_index_selected: bool = False
    spurious_selected: bool = False
    command_threshold_selected: bool = False
    ranging_threshold_selected: bool = False


class MeasureRunStartRequest(BaseModel):
    test_phase: str
    sub_test_phase: str
    cal_id: str
    test_plan_type: str
    execution_mode: str
    remarks: str | None = None
    continue_on_missing_downlink_cal: bool = False
    transmitter_rows: List[MeasureTableRow] = Field(default_factory=list)
    receiver_rows: List[MeasureTableRow] = Field(default_factory=list)
    transponder_rows: List[MeasureTableRow] = Field(default_factory=list)


class MeasureMissingChannel(BaseModel):
    system_kind: Literal["transmitter", "receiver", "transponder"]
    code: str
    port: str
    frequency_label: str
    frequency: float
    parameter: str


class MeasureRunResultRow(BaseModel):
    system_kind: Literal["transmitter", "receiver", "transponder"]
    code: str
    port: str
    frequency_label: str
    frequency: float
    parameter: str
    measured_value: float
    applied_loss: float
    final_value: float
    # Power-measurement breakdown (optional; zero when not applicable)
    raw_value: float = 0.0
    system_loss: float = 0.0
    fixed_pad_loss: float = 0.0
    antenna_gain: float = 0.0
    ground_antenna_gain: float = 0.0
    distance: float = 0.0
    fspl: float = 0.0
    total_loss_calibration: float = 0.0
    on_board_loss: float = 0.0
    status: str
    message: str
    timestamp: datetime


class MeasureRunStartResponse(BaseModel):
    message: str
    processed_rows: int
    requires_confirmation: bool = False
    missing_downlink_channels: List[MeasureMissingChannel] = Field(default_factory=list)
    requested_parameters: List[str] = Field(default_factory=list)
    results: List[MeasureRunResultRow] = Field(default_factory=list)
