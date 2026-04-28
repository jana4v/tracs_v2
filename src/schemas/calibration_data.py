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
    default_cal_id: str | None = None
