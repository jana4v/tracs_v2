from typing import Optional
from pydantic import BaseModel, Field

class InstrumentCatalogResponse(BaseModel):
    instruments: dict[str, list[str]] = Field(default_factory=dict)


class ProjectInstrumentRow(BaseModel):
    instrument_name: str = Field(default="")
    model: str = Field(default="")
    address_main: str = Field(default="")
    address_redt: str = Field(default="")
    use_redt: bool = Field(default=False)


class ProjectInstrumentsResponse(BaseModel):
    rows: list[ProjectInstrumentRow] = Field(default_factory=list)


class ProjectInstrumentsSaveRequest(BaseModel):
    rows: list[ProjectInstrumentRow] = Field(default_factory=list)


class ProjectInstrumentsSaveResponse(BaseModel):
    saved_rows: int


class ProjectPowerMeterRow(BaseModel):
    powerMeter: str = Field(default="")
    channel: str = Field(default="A")


class ProjectPowerMetersResponse(BaseModel):
    rows: list[ProjectPowerMeterRow] = Field(default_factory=list)


class ProjectPowerMetersSaveRequest(BaseModel):
    rows: list[ProjectPowerMeterRow] = Field(default_factory=list)


class ProjectPowerMetersSaveResponse(BaseModel):
    saved_rows: int


class TsmPathRow(BaseModel):
    code: str = Field(default="")
    port: str = Field(default="")
    path1: Optional[str] = Field(default=None)
    path2: Optional[str] = Field(default=None)
    path3: Optional[str] = Field(default=None)
    path4: Optional[str] = Field(default=None)
    path5: Optional[str] = Field(default=None)
    path6: Optional[str] = Field(default=None)


class TsmPathsResponse(BaseModel):
    rows: list[TsmPathRow] = Field(default_factory=list)


class TsmPathsSaveRequest(BaseModel):
    rows: list[TsmPathRow] = Field(default_factory=list)


class TsmPathsSaveResponse(BaseModel):
    saved_rows: int


class ConfigurationValueResponse(BaseModel):
    parameter: str
    value: str


class ConfigurationValueSaveRequest(BaseModel):
    value: str = Field(default="")
