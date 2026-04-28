from typing import List

from src.database.sqlite_json_store import SQLiteJsonCollection
from src.repositories.transmitter_repo import TransmitterRepository
from src.schemas.enums import SystemType
from src.schemas.receiver import ReceiverCreate, ReceiverResponse


class ReceiverRepository(TransmitterRepository):
    """Receiver CRUD operations backed by the same systems collection."""

    def __init__(
        self,
        collection: SQLiteJsonCollection,
        tsm_paths_collection: SQLiteJsonCollection | None = None,
        misc_collection: SQLiteJsonCollection | None = None,
    ):
        super().__init__(collection, tsm_paths_collection, misc_collection)

    def get_all(self) -> List[ReceiverResponse]:
        return self.get_all_for_system_type(SystemType.Receiver)

    def upsert(self, receiver: ReceiverCreate) -> ReceiverResponse:
        return self.upsert_for_system_type(receiver, SystemType.Receiver)

    def delete(self, code: str) -> bool:
        return self.delete_for_system_type(code, SystemType.Receiver)
