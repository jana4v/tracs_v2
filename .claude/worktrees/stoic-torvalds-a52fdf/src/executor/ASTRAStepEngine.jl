"""
    ASTRAStepEngine

Step-by-step execution engine for GUI debugging.
"""
module ASTRAStepEngine

export StepSession, step_next!, step_reset!, get_current_state, StepState
export create_step_session, destroy_step_session, get_session

using Dates
using ..ACSParser
using ..ACSExecutor

# Session state for step execution
mutable struct StepSession
    id::String                         # Unique session ID
    context::ExecutionContext          # Execution context
    breakpoints::Set{Int}              # Line numbers with breakpoints
end

# Current state snapshot sent to GUI
struct StepState
    session_id::String
    test_name::String
    current_line_number::Int           # .tst file line number
    current_line_text::String
    status::Symbol                     # :paused, :running, :completed, etc.
    variables::Dict{String, Any}       # Current variable values
    call_stack::Vector{String}         # Active procedure call chain
    block_stack::Vector{Symbol}        # Active block context
    output::String                     # Console output from last step
end

# Active sessions registry
const ACTIVE_SESSIONS = Dict{String, StepSession}()

"""
    create_step_session(proc_name::String) -> StepSession

Create a new step execution session.
"""
function create_step_session(proc_name::String)::StepSession
    proc = get_procedure(proc_name)
    if proc === nothing
        error("Procedure '$proc_name' not found")
    end

    session_id = string(hash(proc_name * string(now())))

    ctx = ExecutionContext(
        proc, 1, Dict{String, Any}(), String[], Symbol[],
        LogEntry[], :step, :paused, nothing
    )

    session = StepSession(session_id, ctx, Set{Int}())
    ACTIVE_SESSIONS[session_id] = session

    @info "Created step session $session_id for $proc_name"

    return session
end

"""
    destroy_step_session(session_id::String)

Destroy a step execution session.
"""
function destroy_step_session(session_id::String)
    delete!(ACTIVE_SESSIONS, session_id)
    @info "Destroyed step session $session_id"
end

"""
    get_session(session_id::String) -> Union{StepSession, Nothing}

Get a step session by ID.
"""
function get_session(session_id::String)::Union{StepSession, Nothing}
    return get(ACTIVE_SESSIONS, session_id, nothing)
end

"""
    capture_step_output(f::Function) -> String

Capture output from a step execution.
"""
function capture_step_output(f::Function)::String
    # Simple output capture
    output = IOBuffer()
    old_logger = global_logger()

    try
        # Redirect logging
        f()
        return String(take!(output))
    catch e
        return "Error: $(sprint(showerror, e))"
    finally
        global_logger(old_logger)
        close(output)
    end
end

"""
    step_next!(session::StepSession) -> StepState

Execute the next line and return current state.
"""
function step_next!(session::StepSession)::StepState
    ctx = session.context

    if ctx.pointer > length(ctx.procedure.lines)
        ctx.status = :completed
        return get_current_state(session, "Procedure completed")
    end

    line = ctx.procedure.lines[ctx.pointer]
    output = ""

    try
        # Execute current line
        if line.statement_type == :IF
            execute_if_block!(ctx)
            output = "Executed IF block"
        elseif line.statement_type == :FOR
            execute_for_loop!(ctx)
            output = "Executed FOR loop"
        elseif line.statement_type == :WHILE
            execute_while_loop!(ctx)
            output = "Executed WHILE loop"
        else
            execute_current_line!(ctx)
            output = "Line $(line.line_number): $(line.statement_type)"
        end

        ctx.pointer += 1
        ctx.status = :paused
    catch e
        ctx.status = :failed
        output = "Error: $(sprint(showerror, e))"
    end

    return get_current_state(session, output)
end

"""
    step_reset!(session::StepSession)

Reset execution to the beginning.
"""
function step_reset!(session::StepSession)
    ctx = session.context
    ctx.pointer = 1
    ctx.status = :paused
    empty!(ctx.variables)
    empty!(ctx.call_stack)
    empty!(ctx.execution_log)
    ctx.error_info = nothing

    @info "Reset step session $(session.id)"
end

"""
    get_current_state(session::StepSession, output::String="") -> StepState

Get a snapshot of the current execution state.
"""
function get_current_state(session::StepSession, output::String="")::StepState
    ctx = session.context

    if ctx.pointer <= length(ctx.procedure.lines)
        line = ctx.procedure.lines[ctx.pointer]
        line_number = line.line_number
        line_text = line.raw_text
    else
        line_number = -1
        line_text = ""
    end

    # Get current variables from Main module
    variables = Dict{String, Any}()
    try
        # Attempt to capture some common variables
        # In a real implementation, we'd track this more carefully
    catch
    end

    return StepState(
        session.id,
        ctx.procedure.name,
        line_number,
        line_text,
        ctx.status,
        variables,
        copy(ctx.call_stack),
        copy(ctx.block_stack),
        output
    )
end

"""
    add_breakpoint!(session::StepSession, line_number::Int)

Add a breakpoint at a specific line number.
"""
function add_breakpoint!(session::StepSession, line_number::Int)
    push!(session.breakpoints, line_number)
end

"""
    remove_breakpoint!(session::StepSession, line_number::Int)

Remove a breakpoint.
"""
function remove_breakpoint!(session::StepSession, line_number::Int)
    delete!(session.breakpoints, line_number)
end

"""
    step_continue!(session::StepSession) -> StepState

Continue execution until next breakpoint or end.
"""
function step_continue!(session::StepSession)::StepState
    ctx = session.context
    ctx.status = :running

    while ctx.pointer <= length(ctx.procedure.lines) && ctx.status == :running
        line = ctx.procedure.lines[ctx.pointer]

        # Check for breakpoint
        if line.line_number in session.breakpoints
            ctx.status = :paused
            return get_current_state(session, "Breakpoint hit at line $(line.line_number)")
        end

        # Execute line
        try
            step_next!(session)
        catch e
            ctx.status = :failed
            return get_current_state(session, "Error: $(sprint(showerror, e))")
        end
    end

    ctx.status = :completed
    return get_current_state(session, "Execution completed")
end

end # module ASTRAStepEngine
