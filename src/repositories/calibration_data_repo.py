from typing import List, Optional
from src.database.sqlite_json_store import SQLiteJsonCollection


class CalibrationDataRepository:
    def __init__(self, collection: SQLiteJsonCollection) -> None:
        self._col = collection

    def get_cal_ids(self, cal_type: Optional[str] = None) -> List[str]:
        """Return sorted distinct cal_ids, optionally filtered by cal_type."""
        query: dict = {}
        if cal_type:
            query["cal_type"] = cal_type
        result = self._col.distinct("cal_id", query)
        return sorted(result)
