from __future__ import annotations

from typing import Optional

from fastapi import APIRouter, HTTPException, status

from src.database.connection import Database
from src.repositories.env_data_repo import EnvDataRepository
from src.schemas.env_data import (
    EnvDataDirectorySelectResponse,
    EnvDataRow,
    EnvDataRowsResponse,
    EnvDataRowsSaveRequest,
    EnvDataRowsSaveResponse,
    EnvDataUpsertRequest,
)

router = APIRouter(prefix="/api/v2", tags=["ENV Data"])
_repo: Optional[EnvDataRepository] = None


def get_repo() -> EnvDataRepository:
    global _repo
    if _repo is None:
        _repo = EnvDataRepository(Database._db_path, "EnvData")
    return _repo


@router.get("/env-data", response_model=EnvDataRowsResponse)
def get_env_data_rows():
    repo = get_repo()
    return EnvDataRowsResponse(rows=[EnvDataRow(**row) for row in repo.list_rows()])


@router.post("/env-data", response_model=EnvDataRow, status_code=status.HTTP_201_CREATED)
def create_env_data_row(payload: EnvDataUpsertRequest):
    repo = get_repo()
    try:
        row = repo.upsert_row(payload.parameter, payload.value)
    except ValueError as exc:
        raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail=str(exc)) from exc
    return EnvDataRow(**row)


@router.put("/env-data/{parameter}", response_model=EnvDataRow)
def update_env_data_row(parameter: str, payload: EnvDataUpsertRequest):
    repo = get_repo()
    try:
        row = repo.upsert_row(parameter, payload.value)
    except ValueError as exc:
        raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail=str(exc)) from exc
    return EnvDataRow(**row)


@router.delete("/env-data/{parameter}", status_code=status.HTTP_200_OK)
def delete_env_data_row(parameter: str):
    repo = get_repo()
    deleted = repo.delete_row(parameter)
    if not deleted:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="Row not found")
    return {"deleted": True, "parameter": parameter}


@router.put("/env-data", response_model=EnvDataRowsSaveResponse)
def save_env_data_rows(payload: EnvDataRowsSaveRequest):
    repo = get_repo()
    saved_rows = repo.replace_rows([row.model_dump() for row in payload.rows])
    return EnvDataRowsSaveResponse(saved_rows=saved_rows)


@router.get("/env-data/select-directory", response_model=EnvDataDirectorySelectResponse)
def select_env_data_directory():
    try:
        import tkinter as tk
        from tkinter import filedialog

        root = tk.Tk()
        root.withdraw()
        root.attributes("-topmost", True)
        selected = filedialog.askdirectory(title="Select Results Directory")
        root.destroy()
        selected_text = str(selected or "").strip()
        return EnvDataDirectorySelectResponse(path=selected_text or None)
    except Exception as exc:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Unable to open directory picker: {exc}",
        ) from exc
