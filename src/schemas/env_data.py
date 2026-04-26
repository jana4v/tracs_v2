from pydantic import BaseModel, Field


class EnvDataRow(BaseModel):
    parameter: str = Field(default="")
    value: str = Field(default="")


class EnvDataRowsResponse(BaseModel):
    rows: list[EnvDataRow] = Field(default_factory=list)


class EnvDataRowsSaveRequest(BaseModel):
    rows: list[EnvDataRow] = Field(default_factory=list)


class EnvDataRowsSaveResponse(BaseModel):
    saved_rows: int


class EnvDataUpsertRequest(BaseModel):
    parameter: str = Field(default="")
    value: str = Field(default="")


class EnvDataDirectorySelectResponse(BaseModel):
    path: str | None = None
