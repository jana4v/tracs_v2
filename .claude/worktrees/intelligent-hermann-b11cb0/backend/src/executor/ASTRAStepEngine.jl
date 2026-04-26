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
import ..ACSExecutor: execute_current_line!, eval_condition, normalize_condition_tokens

struct IfFrame
    end_idx::Int
    else_idx::Union{Int, Nothing}
    took_then::Bool
end

struct LoopFrame
    kind::Symbol
    start_idx::Int
    end_idx::Int
    var_name::Union{String, Nothing}
    current_value::Union{Int, Nothing}
    end_value::Union{Int, Nothing}
    condition_str::Union{String, Nothing}
end

# Session state for step execution
mutable struct StepSession
    id::String                         # Unique session ID
    context::ExecutionContext          # Execution context
    breakpoints::Set{Int}              # Line numbers with breakpoints
    if_stack::Vector{IfFrame}          # Active IF blocks
    loop_stack::Vector{LoopFrame}      # Active loop blocks
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
        LogEntry[], :step, :paused, nothing, nothing
    )

    session = StepSession(session_id, ctx, Set{Int}(), IfFrame[], LoopFrame[])
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
        if line.statement_type == :IF
            output = handle_if_step!(session, line)
        elseif line.statement_type == :ELSE
            output = handle_else_step!(session)
        elseif line.statement_type == :END
            output = handle_end_step!(session)
        elseif line.statement_type == :FOR
            output = handle_for_step!(session, line)
        elseif line.statement_type == :WHILE
            output = handle_while_step!(session, line)
        else
            execute_current_line!(ctx)
            output = "Line $(line.line_number): $(line.statement_type)"
            ctx.pointer += 1
        end

        ctx.status = :paused
    catch e
        ctx.status = :failed
        output = "Error: $(sprint(showerror, e))"
    end

    return get_current_state(session, output)
end

function handle_if_step!(session::StepSession, line::ParsedLine)::String
    ctx = session.context
    condition_str = normalize_condition_tokens(line.tokens[2:end])

    else_idx = nothing
    end_idx = find_matching_end(ctx.procedure.lines, ctx.pointer)

    depth = 0
    for i in (ctx.pointer+1):(end_idx-1)
        stmt = ctx.procedure.lines[i]
        if stmt.statement_type in (:IF, :FOR, :WHILE)
            depth += 1
        elseif stmt.statement_type == :END
            depth -= 1
        elseif stmt.statement_type == :ELSE && depth == 0
            else_idx = i
            break
        end
    end

    condition = eval_condition(condition_str)

    if condition
        push!(session.if_stack, IfFrame(end_idx, else_idx, true))
        ctx.pointer += 1
        return "IF true"
    end

    if else_idx !== nothing
        push!(session.if_stack, IfFrame(end_idx, else_idx, false))
        ctx.pointer = else_idx + 1
        return "IF false, entering ELSE"
    end

    ctx.pointer = end_idx + 1
    return "IF false, skipping block"
end

function handle_else_step!(session::StepSession)::String
    ctx = session.context

    if isempty(session.if_stack)
        ctx.pointer += 1
        return "ELSE"
    end

    frame = session.if_stack[end]
    if frame.took_then
        pop!(session.if_stack)
        ctx.pointer = frame.end_idx + 1
        return "Skipping ELSE"
    end

    ctx.pointer += 1
    return "Entering ELSE"
end

function handle_for_step!(session::StepSession, line::ParsedLine)::String
    ctx = session.context

    if length(line.tokens) < 6
        error("FOR loop syntax: FOR <var> IN <start> TO <end>")
    end

    var_name = line.tokens[2]
    start_val = parse(Int, line.tokens[4])
    end_val = parse(Int, line.tokens[6])

    loop_start = ctx.pointer + 1
    loop_end = find_matching_end(ctx.procedure.lines, ctx.pointer)

    if start_val > end_val
        ctx.pointer = loop_end + 1
        return "FOR skipped (empty range)"
    end

    set_loop_var!(var_name, start_val)
    push!(session.loop_stack, LoopFrame(:FOR, loop_start, loop_end, var_name, start_val, end_val, nothing))
    ctx.pointer = loop_start
    return "FOR start: $var_name=$start_val"
end

function handle_while_step!(session::StepSession, line::ParsedLine)::String
    ctx = session.context
    condition_str = normalize_condition_tokens(line.tokens[2:end])
    loop_start = ctx.pointer + 1
    loop_end = find_matching_end(ctx.procedure.lines, ctx.pointer)

    if eval_condition(condition_str)
        push!(session.loop_stack, LoopFrame(:WHILE, loop_start, loop_end, nothing, nothing, nothing, condition_str))
        ctx.pointer = loop_start
        return "WHILE true"
    end

    ctx.pointer = loop_end + 1
    return "WHILE false, skipping block"
end

function handle_end_step!(session::StepSession)::String
    ctx = session.context

    if !isempty(session.loop_stack) && session.loop_stack[end].end_idx == ctx.pointer
        frame = session.loop_stack[end]

        if frame.kind == :FOR
            current_val = frame.current_value::Int
            end_val = frame.end_value::Int
            if current_val < end_val
                next_val = current_val + 1
                session.loop_stack[end] = LoopFrame(
                    :FOR,
                    frame.start_idx,
                    frame.end_idx,
                    frame.var_name,
                    next_val,
                    end_val,
                    frame.condition_str
                )
                set_loop_var!(frame.var_name::String, next_val)
                ctx.pointer = frame.start_idx
                return "FOR next: $(frame.var_name)=$next_val"
            end

            pop!(session.loop_stack)
            ctx.pointer += 1
            return "FOR complete"
        elseif frame.kind == :WHILE
            condition_str = frame.condition_str::String
            if eval_condition(condition_str)
                ctx.pointer = frame.start_idx
                return "WHILE continue"
            end

            pop!(session.loop_stack)
            ctx.pointer += 1
            return "WHILE complete"
        end
    end

    if !isempty(session.if_stack) && session.if_stack[end].end_idx == ctx.pointer
        pop!(session.if_stack)
        ctx.pointer += 1
        return "END"
    end

    ctx.pointer += 1
    return "END"
end

function set_loop_var!(var_name::String, value::Int)
    Base.eval(ACSExecutor, :($(Symbol(var_name)) = $value))
end

function find_matching_end(lines::Vector{ParsedLine}, start_idx::Int)::Int
    depth = 1
    for i in (start_idx+1):length(lines)
        if lines[i].statement_type in (:IF, :FOR, :WHILE)
            depth += 1
        elseif lines[i].statement_type == :END
            depth -= 1
            if depth == 0
                return i
            end
        end
    end
    error("No matching END found for block at line $(lines[start_idx].line_number)")
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
    empty!(session.if_stack)
    empty!(session.loop_stack)

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
