from pathlib import Path

from src.config import settings
from src.database.sqlite_json_store import SQLiteJsonCollection


class Database:
    _collections: dict[str, SQLiteJsonCollection] = {}
    _db_path: str = str((Path(__file__).resolve().parents[2] / settings.SQLITE_DB_PATH))

    @classmethod
    def get_collection(cls, name: str) -> SQLiteJsonCollection:
        collection = cls._collections.get(name)
        if collection is None:
            collection = SQLiteJsonCollection(cls._db_path, name)
            cls._collections[name] = collection
        return collection


def get_transmitters_collection() -> SQLiteJsonCollection:
    return Database.get_collection(settings.TRANSMITTERS_COLLECTION)


def get_instruments_collection() -> SQLiteJsonCollection:
    return Database.get_collection(settings.INSTRUMENTS_COLLECTION)


def get_project_instruments_collection() -> SQLiteJsonCollection:
    return Database.get_collection(settings.PROJECT_INSTRUMENTS_COLLECTION)


def get_project_power_meters_collection() -> SQLiteJsonCollection:
    return Database.get_collection(settings.PROJECT_POWER_METERS_COLLECTION)


def get_project_tsm_paths_collection() -> SQLiteJsonCollection:
    return Database.get_collection(settings.PROJECT_TSM_PATHS_COLLECTION)


def get_configuration_collection() -> SQLiteJsonCollection:
    return Database.get_collection(settings.CONFIGURATION_COLLECTION)


def get_transmitter_misc_collection() -> SQLiteJsonCollection:
    return Database.get_collection(settings.TRANSMITTER_MISC_COLLECTION)


def get_calibration_data_collection() -> SQLiteJsonCollection:
    return Database.get_collection(settings.CALIBRATION_DATA_COLLECTION)


def get_calibration_runs_collection() -> SQLiteJsonCollection:
    return Database.get_collection(settings.CALIBRATION_RUNS_COLLECTION)
