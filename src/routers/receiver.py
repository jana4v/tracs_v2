from typing import List

from fastapi import APIRouter, Depends, HTTPException, status

from src.database.connection import (
    get_project_tsm_paths_collection,
    get_transmitter_misc_collection,
    get_transmitters_collection,
)
from src.database.sqlite_json_store import SQLiteJsonCollection
from src.repositories.receiver_repo import ReceiverRepository
from src.schemas.receiver import ReceiverCreate, ReceiverResponse

router = APIRouter(prefix='/api/v2', tags=['Receivers'])


def get_repo(
    collection: SQLiteJsonCollection = Depends(get_transmitters_collection),
    tsm_paths_collection: SQLiteJsonCollection = Depends(get_project_tsm_paths_collection),
    misc_collection: SQLiteJsonCollection = Depends(get_transmitter_misc_collection),
) -> ReceiverRepository:
    return ReceiverRepository(collection, tsm_paths_collection, misc_collection)


@router.get('/receivers', response_model=List[ReceiverResponse])
def list_receivers(repo: ReceiverRepository = Depends(get_repo)):
    return repo.get_all()


@router.post('/receivers', response_model=ReceiverResponse, status_code=status.HTTP_200_OK)
def save_receiver(
    payload: ReceiverCreate,
    repo: ReceiverRepository = Depends(get_repo),
):
    """Create or update a receiver (upsert by code)."""
    return repo.upsert(payload)


@router.delete('/receivers/{code}', status_code=status.HTTP_200_OK)
def delete_receiver(
    code: str,
    repo: ReceiverRepository = Depends(get_repo),
):
    deleted = repo.delete(code)
    if not deleted:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Receiver with code '{code}' not found.",
        )
    return {'message': f"Receiver '{code}' deleted successfully."}
