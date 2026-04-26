import asyncio
import json
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional
from uuid import uuid4

from src.calibration import procedure_factory
from src.calibration.base import CalibrationDependencies
from src.database.sqlite_json_store import SQLiteJsonCollection

from src.schemas.calibration_data import (
    CalibrationChannel,
    CalibrationRunAbortResponse,
    CalibrationRunPromptResponseRequest,
    CalibrationRunSnapshot,
    CalibrationRunStartRequest,
    CalibrationSample,
)


ACTIVE_STATES = {"created", "awaiting_operator", "running", "aborting"}


@dataclass
class CalibrationRunRuntime:
    run_id: str
    cal_id: str
    cal_type: str
    channels: List[CalibrationChannel]
    state: str = "created"
    progress: float = 0.0
    status_lines: List[str] = field(default_factory=list)
    samples: List[CalibrationSample] = field(default_factory=list)
    created_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc))
    updated_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc))
    prompt_message: Optional[str] = None
    prompt_channel: Optional[CalibrationChannel] = None
    operator_connected: bool = False
    abort_requested: bool = False
    prompt_event: asyncio.Event = field(default_factory=asyncio.Event)
    abort_event: asyncio.Event = field(default_factory=asyncio.Event)
    lock: asyncio.Lock = field(default_factory=asyncio.Lock)
    subscribers: List[asyncio.Queue] = field(default_factory=list)
    has_executor: bool = False


class CalibrationRunService:
    def __init__(self, runs_collection: SQLiteJsonCollection, dependencies: CalibrationDependencies) -> None:
        self._runs_collection = runs_collection
        self._dependencies = dependencies
        self._runs: Dict[str, CalibrationRunRuntime] = {}
        self._guard = asyncio.Lock()

    async def start_run(self, payload: CalibrationRunStartRequest) -> CalibrationRunSnapshot:
        async with self._guard:
            active = self._get_active_run_locked()
            if active is not None:
                raise ValueError(f"Calibration already running: {active.run_id}")

            run_id = str(uuid4())
            runtime = CalibrationRunRuntime(
                run_id=run_id,
                cal_id=payload.cal_id,
                cal_type=procedure_factory.normalize_cal_type(payload.cal_type),
                channels=payload.channels,
                has_executor=True,
            )
            self._runs[run_id] = runtime

            self._persist(runtime)
            asyncio.create_task(self._execute_run(runtime, payload))
            return self._snapshot(runtime)

    async def abort_run(self, run_id: str) -> CalibrationRunAbortResponse:
        runtime = self._runs.get(run_id)
        if runtime is None:
            runtime = await self._rehydrate_active_run(run_id)
        if runtime is None:
            return CalibrationRunAbortResponse(run_id=run_id, state="not_found", message="Run not found")

        async with runtime.lock:
            runtime.abort_requested = True
            runtime.abort_event.set()
            runtime.prompt_event.set()
            runtime.state = "aborting"
            runtime.prompt_message = None
            self._append_status(runtime, "Abort requested by user")
            self._persist(runtime)
            self._broadcast(runtime, "status")
            return CalibrationRunAbortResponse(run_id=run_id, state=runtime.state, message="Abort requested")

    async def operator_prompt_response(
        self,
        run_id: str,
        payload: CalibrationRunPromptResponseRequest,
    ) -> CalibrationRunSnapshot:
        runtime = self._runs.get(run_id)
        if runtime is None:
            runtime = await self._rehydrate_active_run(run_id)
        if runtime is None:
            raise ValueError("Run not found")

        async with runtime.lock:
            if payload.action == "connected":
                runtime.operator_connected = True
                runtime.prompt_message = None
                self._append_status(runtime, "Operator confirmed sensor is connected")
            else:
                runtime.abort_requested = True
                runtime.abort_event.set()
                runtime.state = "aborting"
                runtime.prompt_message = None
                self._append_status(runtime, "Operator aborted calibration")
            runtime.prompt_event.set()
            self._persist(runtime)
            self._broadcast(runtime, "status")

        if payload.action == "connected" and not runtime.has_executor:
            runtime.has_executor = True
            asyncio.create_task(self._run_measurement_loop(runtime))

        return self._snapshot(runtime)

    async def get_snapshot(self, run_id: str) -> Optional[CalibrationRunSnapshot]:
        runtime = self._runs.get(run_id)
        if runtime is not None:
            return self._snapshot(runtime)

        doc = await asyncio.to_thread(
            self._runs_collection.find_one,
            {"run_id": run_id},
            {"_id": 0},
        )
        if not doc:
            return None
        snapshot = CalibrationRunSnapshot.model_validate(doc)
        return await self._reconcile_orphaned_active_snapshot(snapshot)

    async def get_active_snapshot(self) -> Optional[CalibrationRunSnapshot]:
        active = self._get_active_run_locked()
        if active is not None:
            return self._snapshot(active)

        doc = await asyncio.to_thread(
            self._runs_collection.find_one,
            {"state": {"$in": list(ACTIVE_STATES)}},
            {"_id": 0},
            sort=[("updated_at", -1)],
        )
        if not doc:
            return None
        snapshot = CalibrationRunSnapshot.model_validate(doc)
        snapshot = await self._reconcile_orphaned_active_snapshot(snapshot)
        if snapshot.state not in ACTIVE_STATES:
            return None
        return snapshot

    async def get_latest_snapshot(self, cal_type: str | None = None) -> Optional[CalibrationRunSnapshot]:
        query: dict[str, Any] = {}
        if cal_type is not None and str(cal_type).strip() != "":
            query["cal_type"] = procedure_factory.normalize_cal_type(cal_type)

        doc = await asyncio.to_thread(
            self._runs_collection.find_one,
            query,
            {"_id": 0},
            sort=[("updated_at", -1)],
        )
        if not doc:
            return None
        return CalibrationRunSnapshot.model_validate(doc)

    async def stream_events(self, run_id: str):
        runtime = self._runs.get(run_id)
        if runtime is None:
            snapshot = await self.get_snapshot(run_id)
            if snapshot is None:
                yield "event: end\ndata: {\"message\":\"Run not found\"}\n\n"
                return
            yield f"event: snapshot\ndata: {snapshot.model_dump_json()}\n\n"
            yield "event: end\ndata: {\"message\":\"Run not active\"}\n\n"
            return

        queue: asyncio.Queue = asyncio.Queue()
        runtime.subscribers.append(queue)
        try:
            yield f"event: snapshot\ndata: {self._snapshot(runtime).model_dump_json()}\n\n"
            while True:
                payload = await queue.get()
                event = payload.get("event", "status")
                yield f"event: {event}\ndata: {json.dumps(payload)}\n\n"
                if payload.get("state") in {"completed", "failed", "aborted"}:
                    break
        finally:
            if queue in runtime.subscribers:
                runtime.subscribers.remove(queue)

    async def _execute_run(self, runtime: CalibrationRunRuntime, payload: CalibrationRunStartRequest) -> None:
        try:
            procedure = procedure_factory.get(payload.cal_type)
            if procedure is None:
                await self._mark_terminal(runtime, "failed", f"Unsupported calibration type: {payload.cal_type}")
                return
            await procedure.execute(self, runtime, payload, self._dependencies)
        except Exception as exc:
            await self._mark_terminal(runtime, "failed", f"Calibration failed: {exc}")

    async def prompt_operator(self, runtime: CalibrationRunRuntime, message: str, channel: Optional[CalibrationChannel]) -> None:
        async with runtime.lock:
            runtime.state = "awaiting_operator"
            runtime.prompt_channel = channel
            runtime.prompt_message = message
            runtime.progress = max(5.0, runtime.progress)
            self._append_status(runtime, message)
            runtime.prompt_event.clear()
            self._persist(runtime)
            self._broadcast(runtime, "prompt")

        await runtime.prompt_event.wait()

    async def set_running(self, runtime: CalibrationRunRuntime, message: str) -> None:
        async with runtime.lock:
            runtime.state = "running"
            runtime.prompt_message = None
            runtime.progress = max(10.0, runtime.progress)
            self._append_status(runtime, message)
            self._persist(runtime)
            self._broadcast(runtime, "status")

    async def push_status(self, runtime: CalibrationRunRuntime, message: str) -> None:
        async with runtime.lock:
            self._append_status(runtime, message)
            self._persist(runtime)
            self._broadcast(runtime, "status")

    async def record_measurement(
        self,
        runtime: CalibrationRunRuntime,
        channel: CalibrationChannel,
        frequency: float,
        value: float | None = None,
        sa_loss: float | None = None,
        dl_pm_loss: float | None = None,
        timestamp: datetime | None = None,
        progress: float = 0.0,
    ) -> None:
        if timestamp is None:
            timestamp = datetime.now(timezone.utc)
        
        sample = CalibrationSample(
            code=channel.code,
            port=channel.port,
            frequency_label=channel.frequency_label,
            frequency=str(frequency),
            value=value,
            sa_loss=sa_loss,
            dl_pm_loss=dl_pm_loss,
            timestamp=timestamp,
        )
        
        # Generate status message based on available loss fields
        if sa_loss is not None and dl_pm_loss is not None:
            status_msg = f"Measured {channel.code}/{channel.port} @ {frequency}: SA Loss={sa_loss:.1f} dB, DL_PM Loss={dl_pm_loss:.1f} dB"
        elif value is not None:
            status_msg = f"Measured {channel.code}/{channel.port} @ {frequency}: {value:.1f} dBm"
        else:
            status_msg = f"Measured {channel.code}/{channel.port} @ {frequency}"
        
        async with runtime.lock:
            runtime.samples.insert(0, sample)
            runtime.samples = runtime.samples[:100]
            runtime.progress = progress
            self._append_status(runtime, status_msg)
            self._persist(runtime)
            self._broadcast(runtime, "sample")

    async def complete_runtime(self, runtime: CalibrationRunRuntime, message: str) -> None:
        await self._mark_terminal(runtime, "completed", message)

    async def abort_runtime(self, runtime: CalibrationRunRuntime, message: str) -> None:
        await self._mark_terminal(runtime, "aborted", message)

    async def _mark_terminal(self, runtime: CalibrationRunRuntime, state: str, message: str) -> None:
        async with runtime.lock:
            runtime.state = state
            runtime.prompt_message = None
            runtime.progress = 100.0 if state == "completed" else runtime.progress
            self._append_status(runtime, message)
            self._persist(runtime)
            self._broadcast(runtime, "status")

    def _append_status(self, runtime: CalibrationRunRuntime, message: str) -> None:
        now = datetime.now(timezone.utc)
        runtime.updated_at = now
        ts = now.strftime("%H:%M:%S")
        runtime.status_lines.insert(0, f"[{ts}] {message}")
        runtime.status_lines = runtime.status_lines[:200]

    def _snapshot(self, runtime: CalibrationRunRuntime) -> CalibrationRunSnapshot:
        return CalibrationRunSnapshot(
            run_id=runtime.run_id,
            cal_id=runtime.cal_id,
            cal_type=runtime.cal_type,
            state=runtime.state,
            progress=runtime.progress,
            status_lines=runtime.status_lines,
            prompt_message=runtime.prompt_message,
            prompt_channel=runtime.prompt_channel,
            samples=runtime.samples,
            channels=runtime.channels,
            created_at=runtime.created_at,
            updated_at=runtime.updated_at,
            operator_connected=runtime.operator_connected,
        )

    async def _rehydrate_active_run(self, run_id: str) -> Optional[CalibrationRunRuntime]:
        doc = await asyncio.to_thread(
            self._runs_collection.find_one,
            {"run_id": run_id},
            {"_id": 0},
        )
        if not doc:
            return None

        snapshot = CalibrationRunSnapshot.model_validate(doc)
        if snapshot.state not in ACTIVE_STATES:
            return None

        runtime = CalibrationRunRuntime(
            run_id=snapshot.run_id,
            cal_id=snapshot.cal_id,
            cal_type=snapshot.cal_type,
            channels=snapshot.channels,
            state=snapshot.state,
            progress=snapshot.progress,
            status_lines=list(snapshot.status_lines),
            samples=list(snapshot.samples),
            created_at=snapshot.created_at,
            updated_at=snapshot.updated_at,
            prompt_message=snapshot.prompt_message,
            prompt_channel=snapshot.prompt_channel,
            operator_connected=snapshot.operator_connected,
            has_executor=False,
        )
        self._runs[run_id] = runtime
        return runtime

    def _persist(self, runtime: CalibrationRunRuntime) -> None:
        payload = self._snapshot(runtime).model_dump(mode="json")
        try:
            loop = asyncio.get_running_loop()
            loop.create_task(
                asyncio.to_thread(
                    self._runs_collection.update_one,
                    {"run_id": runtime.run_id},
                    {"$set": payload},
                    upsert=True,
                )
            )
        except RuntimeError:
            self._runs_collection.update_one(
                {"run_id": runtime.run_id},
                {"$set": payload},
                upsert=True,
            )

    def _broadcast(self, runtime: CalibrationRunRuntime, event_name: str) -> None:
        snapshot = self._snapshot(runtime)
        payload: Dict[str, Any] = {
            "event": event_name,
            "run_id": snapshot.run_id,
            "state": snapshot.state,
            "progress": snapshot.progress,
            "status_line": snapshot.status_lines[0] if snapshot.status_lines else "",
            "prompt_message": snapshot.prompt_message,
            "prompt_channel": snapshot.prompt_channel.model_dump() if snapshot.prompt_channel else None,
            "sample": snapshot.samples[0].model_dump(mode="json") if snapshot.samples else None,
            "updated_at": snapshot.updated_at.isoformat(),
        }
        for q in list(runtime.subscribers):
            q.put_nowait(payload)

    async def _reconcile_orphaned_active_snapshot(self, snapshot: CalibrationRunSnapshot) -> CalibrationRunSnapshot:
        if snapshot.state not in ACTIVE_STATES:
            return snapshot

        runtime = self._runs.get(snapshot.run_id)
        if runtime is not None:
            return self._snapshot(runtime)

        now = datetime.now(timezone.utc)
        ts = now.strftime("%H:%M:%S")
        recovery_line = f"[{ts}] Run state recovered after restart; marking run as failed"
        merged_status = [recovery_line, *list(snapshot.status_lines)]
        updated = snapshot.model_copy(
            update={
                "state": "failed",
                "prompt_message": None,
                "status_lines": merged_status[:200],
                "updated_at": now,
            }
        )

        await asyncio.to_thread(
            self._runs_collection.update_one,
            {"run_id": updated.run_id},
            {"$set": updated.model_dump(mode="json")},
            upsert=True,
        )

        return updated

    def _get_active_run_locked(self) -> Optional[CalibrationRunRuntime]:
        for run in self._runs.values():
            if run.state in ACTIVE_STATES:
                return run
        return None
