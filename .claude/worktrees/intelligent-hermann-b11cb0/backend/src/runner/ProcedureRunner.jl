"""
    ProcedureRunner

Manages concurrent procedure executions with pause/resume/abort controls.
Each procedure runs in its own Julia Task with control flags checked between statements.
"""
module ProcedureRunner

export start_procedure, pause_procedure, resume_procedure, abort_procedure
export get_run_status, list_runs, cleanup_runs, ProcedureRun

using Dates
using ..ACSParser
using ..ACSExecutor
using ..DataLogger
using ..MongoStore
using ..RedisStore

import ..ACSParser: get_procedure
import ..ACSExecutor: run_test, ExecutionContext, ExecutionResult, LogEntry, AbortException

# ===== Data Structures =====

"""
Control signals for a running procedure.
Uses ReentrantLock + Condition for thread-safe pause/resume.
"""
mutable struct RunControl
    status::Symbol            # :running, :paused, :completed, :failed, :aborted
    pause_flag::Bool          # Whether procedure should pause
    abort_flag::Bool          # Whether procedure should abort
    lock::ReentrantLock       # Protects flag access
    pause_condition::Threads.Condition  # For blocking during pause
end

function RunControl()
    lk = ReentrantLock()
    return RunControl(:running, false, false, lk, Threads.Condition(lk))
end

"""
A single procedure run with its task handle and control signals.
"""
mutable struct ProcedureRun
    id::String                                    # Unique run ID
    procedure_name::String                        # TEST_NAME
    control::RunControl                           # Pause/resume/abort signals
    task::Union{Nothing, Task}                    # Julia Task handle
    context::Union{Nothing, ExecutionContext}      # Execution context
    result::Union{Nothing, ExecutionResult}        # Final result (set on completion)
    started_at::DateTime
    finished_at::Union{Nothing, DateTime}
    error_message::Union{Nothing, String}
end

# Global registry of all procedure runs
const RUNNING_PROCEDURES = Dict{String, ProcedureRun}()
const REGISTRY_LOCK = ReentrantLock()

# Limit concurrent foreground procedure runs to 30.
# Background procedures use a separate semaphore in BackgroundScheduler.
const FOREGROUND_SEMAPHORE = Base.Semaphore(30)

# Callback for broadcasting events (set by GenieApp)
const EVENT_BROADCASTER = Ref{Union{Nothing, Function}}(nothing)

"""
    set_event_broadcaster!(f::Function)

Set the function used to broadcast runner events to connected clients.
Called by GenieApp during setup.
"""
function set_event_broadcaster!(f::Function)
    EVENT_BROADCASTER[] = f
end

# ===== Internal Helpers =====

"""
    generate_run_id() -> String

Generate a unique run ID using timestamp + random suffix.
"""
function generate_run_id()::String
    ts = Dates.format(now(), "yyyymmddHHMMSS")
    suffix = string(rand(1000:9999))
    return "run-$(ts)-$(suffix)"
end

"""
    broadcast_runner_event(run::ProcedureRun, extra::Dict=Dict())

Broadcast a runner state change event.
"""
function broadcast_runner_event(run::ProcedureRun, extra::Dict=Dict{String,Any}())
    if EVENT_BROADCASTER[] !== nothing
        event = Dict{String,Any}(
            "type" => "runner_update",
            "data" => merge(Dict{String,Any}(
                "run_id" => run.id,
                "procedure" => run.procedure_name,
                "status" => string(run.control.status),
                "started_at" => string(run.started_at),
            ), extra),
        )
        try
            EVENT_BROADCASTER[](event)
        catch e
            @warn "Failed to broadcast runner event: $e"
        end
    end
end

"""
    check_control_flags!(run::ProcedureRun)

Called between each DSL statement to handle pause/abort.
This is injected into ExecutionContext as the control_hook.
"""
function check_control_flags!(run::ProcedureRun)
    lock(run.control.lock) do
        # Check abort first (highest priority)
        if run.control.abort_flag
            run.control.status = :aborted
            throw(AbortException("Procedure aborted by user"))
        end

        # Check pause — block until resumed or aborted
        while run.control.pause_flag
            run.control.status = :paused
            broadcast_runner_event(run)
            wait(run.control.pause_condition)

            # Re-check abort after waking (might have been aborted while paused)
            if run.control.abort_flag
                run.control.status = :aborted
                throw(AbortException("Procedure aborted by user"))
            end
        end
    end
end

"""
    run_test_controlled(run::ProcedureRun)

Execute a procedure with control-flag checking between statements.
Delegates to ACSExecutor.run_test with a control_hook set.
"""
function run_test_controlled(run::ProcedureRun)
    try
        run.control.status = :running
        broadcast_runner_event(run)

        proc = get_procedure(run.procedure_name)
        if proc === nothing
            error("Procedure '$(run.procedure_name)' not found")
        end

        # Create execution context with our control hook
        ctx = ExecutionContext(
            proc, 1, Dict{String, Any}(), String[], Symbol[],
            LogEntry[], :full, :running, nothing,
            () -> check_control_flags!(run)
        )
        run.context = ctx

        # Use ACSExecutor.run_test with the pre-built context
        result = run_test(run.procedure_name; mode=:full, parent_ctx=ctx)

        run.result = result
        run.control.status = result.status
        run.finished_at = now()

        # Log the result with test phase from Redis
        try
            log_test_result(result)
            test_phase = ""
            try
                test_phase = RedisStore.get_test_phase()
            catch; end
            MongoStore.save_test_result(result; test_phase=test_phase)
        catch e
            @warn "Failed to log/save test result: $e"
        end

        broadcast_runner_event(run, Dict{String,Any}(
            "duration" => result.duration_seconds,
            "log_entries" => length(result.log),
        ))

    catch e
        run.finished_at = now()

        if e isa AbortException
            run.control.status = :aborted
            run.error_message = e.message
        else
            run.control.status = :failed
            run.error_message = sprint(showerror, e)
        end

        broadcast_runner_event(run, Dict{String,Any}(
            "error" => run.error_message,
        ))
    end
end

# ===== Public API =====

"""
    start_procedure(proc_name::String) -> ProcedureRun

Start a procedure in a new task. Returns the ProcedureRun with a unique run_id.
"""
function start_procedure(proc_name::String)::ProcedureRun
    # Verify procedure exists before spawning
    proc = get_procedure(proc_name)
    if proc === nothing
        error("Procedure '$proc_name' not found in registry. Load it first.")
    end

    run_id = generate_run_id()
    control = RunControl()

    run = ProcedureRun(
        run_id,
        proc_name,
        control,
        nothing,       # task (set below)
        nothing,       # context (set during execution)
        nothing,       # result
        now(),
        nothing,       # finished_at
        nothing,       # error_message
    )

    # Register before spawning to avoid race
    lock(REGISTRY_LOCK) do
        RUNNING_PROCEDURES[run_id] = run
    end

    # Spawn the task, acquiring the foreground semaphore slot
    run.task = Threads.@spawn begin
        Base.acquire(FOREGROUND_SEMAPHORE)
        try
            run_test_controlled(run)
        finally
            Base.release(FOREGROUND_SEMAPHORE)
        end
    end

    @info "Started procedure '$proc_name' as run $run_id"
    return run
end

"""
    pause_procedure(run_id::String) -> Bool

Pause a running procedure. Returns true if the pause signal was sent.
The procedure will pause at the next statement boundary.
"""
function pause_procedure(run_id::AbstractString)::Bool
    run = get(RUNNING_PROCEDURES, run_id, nothing)
    if run === nothing
        @warn "Run $run_id not found"
        return false
    end

    lock(run.control.lock) do
        if run.control.status != :running
            @warn "Run $run_id is not running (status: $(run.control.status))"
            return false
        end
        run.control.pause_flag = true
    end

    @info "Pause signal sent to run $run_id"
    return true
end

"""
    resume_procedure(run_id::String) -> Bool

Resume a paused procedure. Returns true if the resume signal was sent.
"""
function resume_procedure(run_id::AbstractString)::Bool
    run = get(RUNNING_PROCEDURES, run_id, nothing)
    if run === nothing
        @warn "Run $run_id not found"
        return false
    end

    lock(run.control.lock) do
        if run.control.status != :paused && !run.control.pause_flag
            @warn "Run $run_id is not paused (status: $(run.control.status))"
            return false
        end
        run.control.pause_flag = false
        run.control.status = :running
        notify(run.control.pause_condition)
    end

    broadcast_runner_event(run)
    @info "Resume signal sent to run $run_id"
    return true
end

"""
    abort_procedure(run_id::String) -> Bool

Abort a running or paused procedure. Returns true if the abort signal was sent.
"""
function abort_procedure(run_id::AbstractString)::Bool
    run = get(RUNNING_PROCEDURES, run_id, nothing)
    if run === nothing
        @warn "Run $run_id not found"
        return false
    end

    lock(run.control.lock) do
        if run.control.status in (:completed, :failed, :aborted)
            @warn "Run $run_id already finished (status: $(run.control.status))"
            return false
        end
        run.control.abort_flag = true
        run.control.pause_flag = false  # Unblock if paused
        notify(run.control.pause_condition)
    end

    @info "Abort signal sent to run $run_id"
    return true
end

"""
    get_run_status(run_id::String) -> Union{Dict, Nothing}

Get the current status of a procedure run.
"""
function get_run_status(run_id::AbstractString)::Union{Dict, Nothing}
    run = get(RUNNING_PROCEDURES, run_id, nothing)
    if run === nothing
        return nothing
    end
    return run_to_dict(run)
end

"""
    list_runs() -> Vector{Dict}

List all procedure runs (active and completed).
"""
function list_runs()::Vector{Dict}
    runs = Dict[]
    lock(REGISTRY_LOCK) do
        for (_, run) in RUNNING_PROCEDURES
            push!(runs, run_to_dict(run))
        end
    end
    # Sort by started_at descending
    sort!(runs, by=r -> r["started_at"], rev=true)
    return runs
end

"""
    cleanup_runs(max_age_seconds::Int=3600)

Remove completed/failed/aborted runs older than max_age_seconds.
"""
function cleanup_runs(max_age_seconds::Int=3600)
    cutoff = now() - Second(max_age_seconds)
    lock(REGISTRY_LOCK) do
        for (id, run) in collect(RUNNING_PROCEDURES)
            if run.control.status in (:completed, :failed, :aborted)
                if run.finished_at !== nothing && run.finished_at < cutoff
                    delete!(RUNNING_PROCEDURES, id)
                    @info "Cleaned up run $id"
                end
            end
        end
    end
end

"""
    run_to_dict(run::ProcedureRun) -> Dict

Convert a ProcedureRun to a dictionary for JSON serialization.
"""
function run_to_dict(run::ProcedureRun)::Dict
    d = Dict{String,Any}(
        "run_id" => run.id,
        "procedure" => run.procedure_name,
        "status" => string(run.control.status),
        "started_at" => string(run.started_at),
        "finished_at" => run.finished_at !== nothing ? string(run.finished_at) : nothing,
        "error" => run.error_message,
    )

    # Add execution progress if context is available (safe against concurrent access)
    try
        ctx = run.context
        if ctx !== nothing
            d["current_line"] = ctx.pointer
            d["total_lines"] = length(ctx.procedure.lines)
            d["log_entries"] = length(ctx.execution_log)
        end
    catch; end

    # Add result info if completed
    try
        if run.result !== nothing
            d["duration"] = run.result.duration_seconds
        end
    catch; end

    return d
end

end # module ProcedureRunner
