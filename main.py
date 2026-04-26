import sys
import os

# Ensure the TRACS_V2 directory is on the Python path so `src.*` imports resolve
sys.path.insert(0, os.path.dirname(__file__))

import uvicorn
from contextlib import asynccontextmanager
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

# Import factory to trigger instrument auto-discovery and registration
from src.InstrumentApi.factory import factory

from src.routers.transmitter import router as transmitter_router
from src.routers.calibration import router as calibration_router
from src.routers.env_data import router as env_data_router


@asynccontextmanager
async def lifespan(app: FastAPI):
    # Eagerly initialize calibration tables so they exist before any run starts
    from src.database.connection import Database
    from src.config import settings
    from src.repositories.cal_sg_calibration_repo import CalSgCalibrationRepository
    from src.repositories.inject_cal_calibration_repo import InjectCalCalibrationRepository
    CalSgCalibrationRepository(Database._db_path, settings.CAL_SG_CALIBRATION_TABLE)
    InjectCalCalibrationRepository(Database._db_path, settings.INJECT_CAL_CALIBRATION_TABLE)
    Database.get_collection(settings.CONFIGURATION_COLLECTION)
    yield


app = FastAPI(
    title="TRACS V2 API",
    description="TRACS V2 backend — database management for RF test systems",
    version="2.0.0",
    lifespan=lifespan,
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(transmitter_router)
app.include_router(calibration_router)
app.include_router(env_data_router)


@app.get("/health")
async def health():
    return {"status": "ok", "version": "2.0.0"}


if __name__ == "__main__":
    uvicorn.run("main:app", host="0.0.0.0", port=8001, reload=True)
