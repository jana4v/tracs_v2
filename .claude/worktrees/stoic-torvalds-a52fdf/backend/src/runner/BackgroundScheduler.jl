"""
    BackgroundScheduler

Manages ~100 background procedures that run continuously.
Each background procedure is either:
  - IntervalSchedule: re-runs every N seconds forever
  - EventDrivenSchedule: waits for a TM condition, runs, then waits again

Separate from ProcedureRunner (foreground). Uses its own concurrency budget.
Persists registrations to MongoDB so procedures auto-start on server boot.
"""
module BackgroundScheduler

export register_background, remove_background
export start_background, stop_background, start_all, stop_all
export list_background, get_background_status
export auto_start_on_boot, set_event_broadcaster!

using Dates
using ..ACSParser
using ..ACSExecutor
using ..MongoStore
using ..RedisStore

import ..ACSParser: get_procedure
import ..ACSExecutor: run_test, AbortException

# ===== Schedule Types =====

abstract type BackgroundSchedule end

"""
Re-run the procedure every `interval_seconds` after the previous run finishes.
If the run takes longer than interval, the next run starts immediately.
"""
struct IntervalSchedule <: BackgroundSchedule
    interval_seconds::Float64       # e.g. 1.0 for every second
    restart_on_failure::Bool        # restart loop even after failure?
    max_consecutive_failures::Int   # stop looping after this many consecutive failures (0 = unlimited)
end

"""
Poll a TM condition string every `poll_interval` seconds.
When condition becomes true, run the procedure once, then resume polling.
"""
struct EventDrivenSchedule <: BackgroundSchedule
    condition::String               # e.g. "TM.MODE == SAFE"
    poll_interval::Float64          # seconds between condition checks
    restart_on_failure::Bool
    max_consecutive_failures::Int
end

# ===== BackgroundEntry =====

mutable struct BackgroundEntry
    proc_name::String
    schedule::BackgroundSchedule
    task::Union{Nothing, Task}
    status::Symbol                  # :running | :idle | :failed | :stopped
    stop_flag::Bool                 # signal the loop to exit
    last_run_at::Union{Nothing, DateTime}
    last_result::Symbol             # :completed | :failed | :aborted | :unknown
    total_runs::Int
    error_count::Int
    consecutive_failures::Int
    last_error::Union{Nothing, String}
    lock::ReentrantLock
end

function BackgroundEntry(proc_name::String, schedule::BackgroundSchedule)
    return BackgroundEntry(
        proc_name, schedule, nothing, :idle, false,
        nothing, :unknown, 0, 0, 0, nothing,
        ReentrantLock()
    )
end

# ===== Global Registry =====

const BACKGROUND_REGISTRY = Dict{String, BackgroundEntry}()
const REGISTRY_LOCK = ReentrantLock()

# Separate semaphore for background procedures (does not conflict with foreground ≤30 limit)
const BACKGROUND_SEMAPHORE = Base.Semaphore(150)

# Event broadcaster (set by GenieApp)
const EVENT_BROADCASTER = Ref{Union{Nothing, Function}}(nothing)

function set_event_broadcaster!(f::Function)
    EVENT_BROADCASTER[] = f
end

# ===== Internal Broadcast =====

function broadcast_bg_event(entry::BackgroundEntry)
    if EVENT_BROADCASTER[] !== nothing
        event = Dict{String,Any}(
            "type" => "background_update",
            "data" => entry_to_dict(entry),
        )
        try
            EVENT_BROADCASTER[](event)
        catch e
            @warn "BackgroundScheduler: broadcast failed: $e"
        end
    end
end

# ===== Loop Implementations =====

"""
Interval loop: run the procedure, sleep the remaining interval, repeat.
"""
function interval_loop!(entry::BackgroundEntry, schedule::IntervalSchedule)
    @info "BackgroundScheduler: interval loop started for '$(entry.proc_name)' (every $(schedule.interval_seconds)s)"

    while true
        # Check stop flag
        lock(entry.lock) do
            if entry.stop_flag
                entry.status = :stopped
            end
        end
        if entry.status == :stopped
            break
        end

        t_start = time()
        entry.status = :running
        broadcast_bg_event(entry)

        # Acquire background semaphore slot
        Base.acquire(BACKGROUND_SEMAPHORE)
        result_status = :unknown
        err_msg = nothing
        try
            result = run_test(entry.proc_name)
            result_status = result.status
        catch e
            result_status = :failed
            err_msg = sprint(showerror, e)
        finally
            Base.release(BACKGROUND_SEMAPHORE)
        end

        lock(entry.lock) do
            entry.last_run_at = now()
            entry.last_result = result_status
            entry.total_runs += 1

            if result_status == :failed || result_status == :aborted
                entry.error_count += 1
                entry.consecutive_failures += 1
                entry.last_error = err_msg
            else
                entry.consecutive_failures = 0
                entry.last_error = nothing
            end

            # Check failure threshold
            if schedule.max_consecutive_failures > 0 &&
               entry.consecutive_failures >= schedule.max_consecutive_failures
                @warn "BackgroundScheduler: '$(entry.proc_name)' exceeded max failures ($(schedule.max_consecutive_failures)), stopping"
                entry.status = :failed
                entry.stop_flag = true
            elseif (result_status in (:failed, :aborted)) && !schedule.restart_on_failure
                entry.status = :failed
                entry.stop_flag = true
            else
                entry.status = :idle
            end
        end

        broadcast_bg_event(entry)

        # Check stop again before sleeping
        if entry.stop_flag || entry.status in (:stopped, :failed)
            break
        end

        elapsed = time() - t_start
        sleep_time = max(0.0, schedule.interval_seconds - elapsed)
        if sleep_time > 0
            sleep(sleep_time)
        end
    end

    lock(entry.lock) do
        if entry.status != :failed
            entry.status = :stopped
        end
    end
    broadcast_bg_event(entry)
    @info "BackgroundScheduler: loop ended for '$(entry.proc_name)' (status: $(entry.status))"
end

"""
Event-driven loop: poll condition, run when true, repeat.
"""
function event_loop!(entry::BackgroundEntry, schedule::EventDrivenSchedule)
    @info "BackgroundScheduler: event loop started for '$(entry.proc_name)' (condition: $(schedule.condition))"

    while true
        # Phase 1: poll until condition is true
        entry.status = :idle
        broadcast_bg_event(entry)

        while true
            lock(entry.lock) do
                if entry.stop_flag
                    entry.status = :stopped
                end
            end
            if entry.status == :stopped
                break
            end

            cond_met = false
            try
                # Evaluate condition as Julia expression (uses TM values from TMInterface)
                cond_met = eval(Meta.parse(schedule.condition)) === true
            catch
                # Condition may not be evaluable yet; treat as not met
            end

            if cond_met
                break
            end

            sleep(schedule.poll_interval)
        end

        if entry.status == :stopped
            break
        end

        # Phase 2: run the procedure
        entry.status = :running
        broadcast_bg_event(entry)

        Base.acquire(BACKGROUND_SEMAPHORE)
        result_status = :unknown
        err_msg = nothing
        try
            result = run_test(entry.proc_name)
            result_status = result.status
        catch e
            result_status = :failed
            err_msg = sprint(showerror, e)
        finally
            Base.release(BACKGROUND_SEMAPHORE)
        end

        lock(entry.lock) do
            entry.last_run_at = now()
            entry.last_result = result_status
            entry.total_runs += 1

            if result_status == :failed || result_status == :aborted
                entry.error_count += 1
                entry.consecutive_failures += 1
                entry.last_error = err_msg
            else
                entry.consecutive_failures = 0
                entry.last_error = nothing
            end

            if schedule.max_consecutive_failures > 0 &&
               entry.consecutive_failures >= schedule.max_consecutive_failures
                entry.status = :failed
                entry.stop_flag = true
            elseif (result_status in (:failed, :aborted)) && !schedule.restart_on_failure
                entry.status = :failed
                entry.stop_flag = true
            end
        end

        broadcast_bg_event(entry)

        if entry.stop_flag || entry.status in (:stopped, :failed)
            break
        end
    end

    lock(entry.lock) do
        if entry.status != :failed
            entry.status = :stopped
        end
    end
    broadcast_bg_event(entry)
    @info "BackgroundScheduler: event loop ended for '$(entry.proc_name)' (status: $(entry.status))"
end

# ===== Public API =====

"""
    register_background(proc_name, schedule; persist=true) -> BackgroundEntry

Register a background procedure. Does not start it automatically.
Set persist=true to save to MongoDB (survives server restart).
"""
function register_background(proc_name::String, schedule::BackgroundSchedule; persist::Bool=true)::BackgroundEntry
    entry = BackgroundEntry(proc_name, schedule)

    lock(REGISTRY_LOCK) do
        BACKGROUND_REGISTRY[proc_name] = entry
    end

    if persist
        try
            MongoStore.save_background_schedule(proc_name, schedule)
        catch e
            @warn "BackgroundScheduler: failed to persist schedule for '$proc_name': $e"
        end
    end

    @info "BackgroundScheduler: registered '$proc_name'"
    return entry
end

"""
    remove_background(proc_name) -> Bool

Stop (if running) and remove a background procedure from the registry.
Also removes from MongoDB.
"""
function remove_background(proc_name::String)::Bool
    entry = lock(REGISTRY_LOCK) do
        get(BACKGROUND_REGISTRY, proc_name, nothing)
    end

    if entry === nothing
        return false
    end

    # Stop first
    stop_background(proc_name)

    lock(REGISTRY_LOCK) do
        delete!(BACKGROUND_REGISTRY, proc_name)
    end

    try
        MongoStore.delete_background_schedule(proc_name)
    catch e
        @warn "BackgroundScheduler: failed to remove persisted schedule for '$proc_name': $e"
    end

    @info "BackgroundScheduler: removed '$proc_name'"
    return true
end

"""
    start_background(proc_name) -> Bool

Start (or restart) the background loop for a registered procedure.
"""
function start_background(proc_name::String)::Bool
    entry = lock(REGISTRY_LOCK) do
        get(BACKGROUND_REGISTRY, proc_name, nothing)
    end

    if entry === nothing
        @warn "BackgroundScheduler: '$proc_name' not registered"
        return false
    end

    # Verify procedure exists in parser
    if get_procedure(proc_name) === nothing
        # Try to load from MongoDB
        try
            proc_doc = MongoStore.get_procedure(proc_name)
            if proc_doc !== nothing
                content = get(proc_doc, "latest_content", "")
                if !isempty(content)
                    ACSParser.load_from_string(content, "$proc_name.tst")
                end
            end
        catch; end

        if get_procedure(proc_name) === nothing
            @warn "BackgroundScheduler: procedure '$proc_name' not found in registry"
            return false
        end
    end

    # Reset stop flag and status
    lock(entry.lock) do
        entry.stop_flag = false
        entry.status = :idle
        entry.consecutive_failures = 0
    end

    # Spawn the loop task
    schedule = entry.schedule
    entry.task = Threads.@spawn begin
        if schedule isa IntervalSchedule
            interval_loop!(entry, schedule)
        elseif schedule isa EventDrivenSchedule
            event_loop!(entry, schedule)
        end
    end

    @info "BackgroundScheduler: started '$proc_name'"
    return true
end

"""
    stop_background(proc_name) -> Bool

Signal the background loop to stop gracefully.
"""
function stop_background(proc_name::String)::Bool
    entry = lock(REGISTRY_LOCK) do
        get(BACKGROUND_REGISTRY, proc_name, nothing)
    end

    if entry === nothing
        return false
    end

    lock(entry.lock) do
        entry.stop_flag = true
    end

    @info "BackgroundScheduler: stop signal sent to '$proc_name'"
    return true
end

"""
    start_all() -> Int

Start all registered background procedures. Returns number started.
"""
function start_all()::Int
    names = lock(REGISTRY_LOCK) do
        collect(keys(BACKGROUND_REGISTRY))
    end

    count = 0
    for name in names
        if start_background(name)
            count += 1
        end
    end
    @info "BackgroundScheduler: started $count / $(length(names)) procedures"
    return count
end

"""
    stop_all() -> Int

Signal all background loops to stop. Returns number signalled.
"""
function stop_all()::Int
    names = lock(REGISTRY_LOCK) do
        collect(keys(BACKGROUND_REGISTRY))
    end

    count = 0
    for name in names
        if stop_background(name)
            count += 1
        end
    end
    @info "BackgroundScheduler: stop signal sent to $count procedures"
    return count
end

"""
    list_background() -> Vector{Dict}

Return status of all registered background procedures.
"""
function list_background()::Vector{Dict}
    entries = lock(REGISTRY_LOCK) do
        collect(values(BACKGROUND_REGISTRY))
    end
    return [entry_to_dict(e) for e in entries]
end

"""
    get_background_status(proc_name) -> Union{Dict, Nothing}
"""
function get_background_status(proc_name::String)::Union{Dict, Nothing}
    entry = lock(REGISTRY_LOCK) do
        get(BACKGROUND_REGISTRY, proc_name, nothing)
    end
    entry === nothing && return nothing
    return entry_to_dict(entry)
end

"""
    auto_start_on_boot()

Load all enabled background schedules from MongoDB and start their loops.
Called during ASTRA server startup.
"""
function auto_start_on_boot()
    @info "BackgroundScheduler: loading persisted schedules from MongoDB..."
    schedules = Dict[]
    try
        schedules = MongoStore.list_background_schedules(enabled_only=true)
    catch e
        @warn "BackgroundScheduler: could not load persisted schedules: $e"
        return
    end

    count = 0
    for sched in schedules
        try
            proc_name = sched["proc_name"]
            schedule_type = get(sched, "schedule_type", "interval")

            schedule = if schedule_type == "interval"
                IntervalSchedule(
                    Float64(get(sched, "interval_seconds", 1.0)),
                    Bool(get(sched, "restart_on_failure", true)),
                    Int(get(sched, "max_consecutive_failures", 10)),
                )
            else
                EventDrivenSchedule(
                    get(sched, "condition", "false"),
                    Float64(get(sched, "poll_interval", 0.5)),
                    Bool(get(sched, "restart_on_failure", true)),
                    Int(get(sched, "max_consecutive_failures", 10)),
                )
            end

            register_background(proc_name, schedule; persist=false)
            start_background(proc_name)
            count += 1
        catch e
            @warn "BackgroundScheduler: failed to auto-start '$(get(sched, "proc_name", "?"))': $e"
        end
    end

    @info "BackgroundScheduler: auto-started $count background procedures"
end

# ===== Serialization =====

function entry_to_dict(entry::BackgroundEntry)::Dict
    sched = entry.schedule
    sched_dict = if sched isa IntervalSchedule
        Dict{String,Any}(
            "type" => "interval",
            "interval_seconds" => sched.interval_seconds,
            "restart_on_failure" => sched.restart_on_failure,
            "max_consecutive_failures" => sched.max_consecutive_failures,
        )
    else
        Dict{String,Any}(
            "type" => "event",
            "condition" => sched.condition,
            "poll_interval" => sched.poll_interval,
            "restart_on_failure" => sched.restart_on_failure,
            "max_consecutive_failures" => sched.max_consecutive_failures,
        )
    end

    # Compute next_run_at for interval schedules
    next_run_at = nothing
    if sched isa IntervalSchedule && entry.last_run_at !== nothing && entry.status == :idle
        next_run_at = string(entry.last_run_at + Millisecond(round(Int, sched.interval_seconds * 1000)))
    end

    return Dict{String,Any}(
        "proc_name"             => entry.proc_name,
        "schedule"              => sched_dict,
        "status"                => string(entry.status),
        "last_run_at"           => entry.last_run_at !== nothing ? string(entry.last_run_at) : nothing,
        "next_run_at"           => next_run_at,
        "last_result"           => string(entry.last_result),
        "total_runs"            => entry.total_runs,
        "error_count"           => entry.error_count,
        "consecutive_failures"  => entry.consecutive_failures,
        "last_error"            => entry.last_error,
    )
end

end # module BackgroundScheduler
