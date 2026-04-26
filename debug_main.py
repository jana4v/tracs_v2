import sys
import os

# Ensure the TRACS_V2 directory is on the Python path so `src.*` imports resolve
sys.path.insert(0, os.path.dirname(__file__))

import uvicorn
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

# Import factory to trigger instrument auto-discovery and registration
from src.InstrumentApi.factory import factory

from src.routers.transmitter import router as transmitter_router
from src.routers.calibration import router as calibration_router
from src.routers.env_data import router as env_data_router

app = FastAPI(
    title="TRACS V2 API",
    description="TRACS V2 backend — database management for RF test systems",
    version="2.0.0",
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
    # Run without reload so pdb/debugger can work properly
    uvicorn.run("debug_main:app", host="0.0.0.0", port=8002, reload=False)
