from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    SQLITE_DB_PATH: str = "tracsV2.sqlite"
    TRANSMITTERS_COLLECTION: str = "transmitters"
    INSTRUMENTS_COLLECTION: str = "Instruments"
    PROJECT_INSTRUMENTS_COLLECTION: str = "ProjectInstruments"
    PROJECT_POWER_METERS_COLLECTION: str = "ProjectPowerMeters"
    PROJECT_TSM_PATHS_COLLECTION: str = "ProjectTSMPaths"
    CONFIGURATION_COLLECTION: str = "Configuration"
    TRANSMITTER_MISC_COLLECTION: str = "TransmitterMisc"
    CALIBRATION_DATA_COLLECTION: str = "CalibrationData"
    CALIBRATION_RUNS_COLLECTION: str = "CalibrationRuns"
    CAL_SG_CALIBRATION_TABLE: str = "CalSGCalibrationData"
    INJECT_CAL_CALIBRATION_TABLE: str = "InjectCalCalibrationData"

    class Config:
        env_file = ".env"


settings = Settings()
