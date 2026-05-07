from pydantic import BaseModel, ConfigDict, Field


ScalarValue = str | int | float | None


class PowerSpecRow(BaseModel):
    code: str
    port: str
    frequency_label: str
    frequency: str
    specification: ScalarValue = None
    tolerance: ScalarValue = None
    fbt: ScalarValue = None
    fbt_hot: ScalarValue = None
    fbt_cold: ScalarValue = None


class FrequencySpecRow(BaseModel):
    code: str
    port: str
    frequency_label: str
    frequency: str
    tolerance: ScalarValue = None
    fbt: ScalarValue = None
    fbt_hot: ScalarValue = None
    fbt_cold: ScalarValue = None


class ModulationIndexSpecRow(BaseModel):
    # Tone-specific columns are dynamic (fbt_tone_32, fbt_hot_tone_32, ...)
    model_config = ConfigDict(extra="allow")

    code: str
    port: str
    frequency_label: str
    frequency: str
    specification: ScalarValue = None
    tolerance: ScalarValue = None


class SpuriousSpecRow(BaseModel):
    code: str
    port: str
    frequency_label: str
    frequency: str
    profile_name: str = ""
    enable: bool = True
    profiles: list[str] = Field(default_factory=list)
    specification: ScalarValue = None
    tolerance: ScalarValue = None
    fbt: list[list[str | int | float]] = Field(default_factory=lambda: [["", ""]])
    fbt_hot: list[list[str | int | float]] = Field(default_factory=lambda: [["", ""]])
    fbt_cold: list[list[str | int | float]] = Field(default_factory=lambda: [["", ""]])


class TestProfileSpuriousSpecRow(BaseModel):
    code: str
    port: str
    frequency_label: str
    frequency: str
    inband: bool = False
    spurband: bool = False


class OnBoardLossSpecRow(BaseModel):
    code: str
    port: str
    frequency_label: str
    frequency: str
    loss_db: ScalarValue = None


class CalibrationSpecRow(BaseModel):
    code: str
    port: str
    frequency_label: str
    frequency: str
    system_loss: ScalarValue = None
    fixed_pad_loss: ScalarValue = None
    antenna_gain: ScalarValue = None
    ground_antenna_gain: ScalarValue = None
    distance: ScalarValue = None
    total_loss: ScalarValue = None


class SpuriousBandConfigRow(BaseModel):
    """Standalone spurious search band entry — not linked to any transmitter."""
    profile_name: str = ""
    enable: bool = True
    start_frequency: float | None = None
    stop_frequency: float | None = None
