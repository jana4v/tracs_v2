"""
    ACSExecutor

Execution engine for ASTRA procedures.
Handles full procedure execution with control flow, error handling, and logging.
"""
module ACSExecutor

export run_test, ExecutionResult, ExecutionContext, LogEntry, ErrorInfo
export register_statement!

using Dates
using ..ACSParser
using ..TMInterface
using ..CommandDispatch

# Execution context holds runtime state
mutable struct ExecutionContext
    procedure::ParsedProcedure
    pointer::Int                     # Current line index
    variables::Dict{String, Any}     # User-defined variables
    call_stack::Vector{String}       # For recursive CALL detection
    block_stack::Vector{Symbol}      # Current nesting context
    execution_log::Vector{Any}       # Timestamped execution log (forward declaration)
    mode::Symbol                     # :full, :step, :simulation
    status::Symbol                   # :running, :paused, :completed, :failed, :aborted
    error_info::Union{Nothing, Any}  # ErrorInfo forward declaration
end

struct LogEntry
    timestamp::DateTime
    line_number::Int
    statement::String
    result::String
    status::Symbol  # :ok, :error, :warning
end

struct ErrorInfo
    line_number::Int
    line_text::String
    message::String
    julia_stacktrace::String
end

struct ExecutionResult
    test_name::String
    status::Symbol
    log::Vector{LogEntry}
    duration_seconds::Float64
    error::Union{Nothing, ErrorInfo}
end

# Statement handler registry
const STATEMENT_HANDLERS = Dict{Symbol, Function}()

"""
    capture_output(f::Function) -> String

Capture stdout from a function.
"""
function capture_output(f::Function)::String
    old_stdout = stdout
    rd, wr = redirect_stdout()

    try
        f()
        flush(wr)
        return String(readavailable(rd))
    finally
        redirect_stdout(old_stdout)
        close(rd)
        close(wr)
    end
end

# ===== Statement Handlers =====

function handle_pre_test_req!(ctx::ExecutionContext, line::ParsedLine)
    expr_str = join(line.tokens[2:end], " ")

    # Evaluate the condition
    try
        # Make TM accessors available
        condition = @eval begin
            using ...TMInterface: TM1, TM2, TM3, TM4, TM5, TM6, TM7, TM8, TM9, TM10
            $(Meta.parse(expr_str))
        end

        if !condition
            throw(ErrorException("PRE_TEST_REQ failed: $expr_str"))
        end

        push!(ctx.execution_log, LogEntry(
            now(), line.line_number, line.raw_text,
            "PRE_TEST_REQ passed: $expr_str", :ok
        ))
    catch e
        push!(ctx.execution_log, LogEntry(
            now(), line.line_number, line.raw_text,
            "PRE_TEST_REQ failed: $(sprint(showerror, e))", :error
        ))
        rethrow(e)
    end
end

function handle_send!(ctx::ExecutionContext, line::ParsedLine)
    command = join(line.tokens[2:end], " ")

    result = send_command(command)

    push!(ctx.execution_log, LogEntry(
        now(), line.line_number, line.raw_text,
        "SEND: $command -> $(result.status)", :ok
    ))
end

function handle_sendtcp!(ctx::ExecutionContext, line::ParsedLine)
    if length(line.tokens) < 4
        error("SENDTCP requires: SENDTCP <host> <port> <data>")
    end

    host = line.tokens[2]
    port = parse(Int, line.tokens[3])
    data = join(line.tokens[4:end], " ")

    result = send_tcp_command(host, port, data)

    push!(ctx.execution_log, LogEntry(
        now(), line.line_number, line.raw_text,
        "SENDTCP: $host:$port -> $(result.status)", :ok
    ))
end

function handle_wait!(ctx::ExecutionContext, line::ParsedLine)
    if length(line.tokens) < 2
        error("WAIT requires duration in seconds")
    end

    # Check if it's WAIT UNTIL
    if length(line.tokens) > 2 && uppercase(line.tokens[2]) == "UNTIL"
        handle_wait_until!(ctx, line)
        return
    end

    duration = parse(Float64, line.tokens[2])

    push!(ctx.execution_log, LogEntry(
        now(), line.line_number, line.raw_text,
        "WAIT $duration seconds", :ok
    ))

    sleep(duration)
end

function handle_wait_until!(ctx::ExecutionContext, line::ParsedLine)
    # Parse: WAIT UNTIL <condition> TIMEOUT <seconds>
    timeout_idx = findfirst(t -> uppercase(t) == "TIMEOUT", line.tokens)

    if timeout_idx === nothing
        error("WAIT UNTIL requires TIMEOUT clause")
    end

    condition_tokens = line.tokens[3:timeout_idx-1]
    condition_str = join(condition_tokens, " ")
    timeout = parse(Float64, line.tokens[timeout_idx+1])

    start_time = time()

    while time() - start_time < timeout
        try
            result = @eval begin
                using ...TMInterface: TM1, TM2, TM3, TM4, TM5, TM6, TM7, TM8, TM9, TM10
                $(Meta.parse(condition_str))
            end

            if result
                push!(ctx.execution_log, LogEntry(
                    now(), line.line_number, line.raw_text,
                    "WAIT UNTIL condition met: $condition_str", :ok
                ))
                return
            end
        catch e
            # Condition evaluation error
        end

        sleep(0.1)  # Poll interval
    end

    # Timeout occurred
    throw(ErrorException("WAIT UNTIL timed out after $timeout seconds: $condition_str"))
end

function handle_check!(ctx::ExecutionContext, line::ParsedLine)
    # Parse: CHECK <condition> [WITHIN <seconds>]
    within_idx = findfirst(t -> uppercase(t) == "WITHIN", line.tokens)

    condition_tokens = if within_idx === nothing
        line.tokens[2:end]
    else
        line.tokens[2:within_idx-1]
    end

    condition_str = join(condition_tokens, " ")

    # Check with optional timeout
    if within_idx !== nothing
        timeout = parse(Float64, line.tokens[within_idx+1])
        start_time = time()

        while time() - start_time < timeout
            try
                result = @eval begin
                    using ...TMInterface: TM1, TM2, TM3, TM4, TM5, TM6, TM7, TM8, TM9, TM10
                    $(Meta.parse(condition_str))
                end

                if result
                    push!(ctx.execution_log, LogEntry(
                        now(), line.line_number, line.raw_text,
                        "CHECK passed: $condition_str", :ok
                    ))
                    return
                end
            catch e
            end

            sleep(0.1)
        end

        throw(ErrorException("CHECK failed within timeout: $condition_str"))
    else
        # Immediate check
        result = @eval begin
            using ...TMInterface: TM1, TM2, TM3, TM4, TM5, TM6, TM7, TM8, TM9, TM10
            $(Meta.parse(condition_str))
        end

        if !result
            throw(ErrorException("CHECK failed: $condition_str"))
        end

        push!(ctx.execution_log, LogEntry(
            now(), line.line_number, line.raw_text,
            "CHECK passed: $condition_str", :ok
        ))
    end
end

function handle_expected!(ctx::ExecutionContext, line::ParsedLine)
    condition_str = join(line.tokens[2:end], " ")

    result = @eval begin
        using ...TMInterface: TM1, TM2, TM3, TM4, TM5, TM6, TM7, TM8, TM9, TM10
        $(Meta.parse(condition_str))
    end

    if !result
        throw(ErrorException("EXPECTED failed: $condition_str"))
    end

    push!(ctx.execution_log, LogEntry(
        now(), line.line_number, line.raw_text,
        "EXPECTED passed: $condition_str", :ok
    ))
end

function handle_alert_msg!(ctx::ExecutionContext, line::ParsedLine)
    message = join(line.tokens[2:end], " ")
    # Remove quotes if present
    message = strip(message, ['"', '\''])

    @warn "ALERT_MSG: $message"

    push!(ctx.execution_log, LogEntry(
        now(), line.line_number, line.raw_text,
        "ALERT: $message", :warning
    ))
end

function handle_abort_test!(ctx::ExecutionContext, line::ParsedLine)
    push!(ctx.execution_log, LogEntry(
        now(), line.line_number, line.raw_text,
        "ABORT_TEST called", :error
    ))

    ctx.status = :aborted
    throw(ErrorException("Test aborted by ABORT_TEST"))
end

function handle_call!(ctx::ExecutionContext, line::ParsedLine)
    if length(line.tokens) < 2
        error("CALL requires procedure name")
    end

    called_name = line.tokens[2]

    # Check for recursive calls
    if called_name in ctx.call_stack
        error("Recursive CALL detected: $called_name")
    end

    push!(ctx.execution_log, LogEntry(
        now(), line.line_number, line.raw_text,
        "CALL $called_name", :ok
    ))

    # Execute the called procedure
    push!(ctx.call_stack, called_name)
    try
        run_test(called_name; mode=ctx.mode, parent_ctx=ctx)
    finally
        pop!(ctx.call_stack)
    end
end

function handle_break!(ctx::ExecutionContext, line::ParsedLine)
    # Set a flag to break out of loop
    # This will be caught by the loop handler
    throw(ErrorException("__BREAK__"))
end

function handle_if!(ctx::ExecutionContext, line::ParsedLine)
    # IF handling is done in execute_block
    push!(ctx.block_stack, :IF)
end

function handle_else!(ctx::ExecutionContext, line::ParsedLine)
    # ELSE handling is done in execute_block
end

function handle_end!(ctx::ExecutionContext, line::ParsedLine)
    # END handling is done in execute_block
    if !isempty(ctx.block_stack)
        pop!(ctx.block_stack)
    end
end

function handle_for!(ctx::ExecutionContext, line::ParsedLine)
    # FOR handling is done in execute_block
    push!(ctx.block_stack, :FOR)
end

function handle_while!(ctx::ExecutionContext, line::ParsedLine)
    # WHILE handling is done in execute_block
    push!(ctx.block_stack, :WHILE)
end

function handle_on_fail!(ctx::ExecutionContext, line::ParsedLine)
    # ON_FAIL is handled by error catching logic
    push!(ctx.block_stack, :ON_FAIL)
end

function handle_on_timeout!(ctx::ExecutionContext, line::ParsedLine)
    # ON_TIMEOUT is handled by timeout catching logic
    push!(ctx.block_stack, :ON_TIMEOUT)
end

# Register handlers
function __init__()
    STATEMENT_HANDLERS[:PRE_TEST_REQ] = handle_pre_test_req!
    STATEMENT_HANDLERS[:SEND]         = handle_send!
    STATEMENT_HANDLERS[:SENDTCP]      = handle_sendtcp!
    STATEMENT_HANDLERS[:WAIT]         = handle_wait!
    STATEMENT_HANDLERS[:CHECK]        = handle_check!
    STATEMENT_HANDLERS[:EXPECTED]     = handle_expected!
    STATEMENT_HANDLERS[:ALERT_MSG]    = handle_alert_msg!
    STATEMENT_HANDLERS[:ABORT_TEST]   = handle_abort_test!
    STATEMENT_HANDLERS[:CALL]         = handle_call!
    STATEMENT_HANDLERS[:BREAK]        = handle_break!
    STATEMENT_HANDLERS[:IF]           = handle_if!
    STATEMENT_HANDLERS[:ELSE]         = handle_else!
    STATEMENT_HANDLERS[:END]          = handle_end!
    STATEMENT_HANDLERS[:FOR]          = handle_for!
    STATEMENT_HANDLERS[:WHILE]        = handle_while!
    STATEMENT_HANDLERS[:ON_FAIL]      = handle_on_fail!
    STATEMENT_HANDLERS[:ON_TIMEOUT]   = handle_on_timeout!
end

"""
    execute_julia_code!(ctx::ExecutionContext, line::ParsedLine)

Execute inline Julia code.
"""
function execute_julia_code!(ctx::ExecutionContext, line::ParsedLine)
    try
        # Evaluate in Main module with TM accessors available
        result = @eval Main begin
            using ..ASTRA.TMInterface: TM1, TM2, TM3, TM4, TM5, TM6, TM7, TM8, TM9, TM10
            $(Meta.parse(line.raw_text))
        end

        push!(ctx.execution_log, LogEntry(
            now(), line.line_number, line.raw_text,
            "Julia code executed", :ok
        ))
    catch e
        push!(ctx.execution_log, LogEntry(
            now(), line.line_number, line.raw_text,
            "Julia error: $(sprint(showerror, e))", :error
        ))
        rethrow(e)
    end
end

"""
    dispatch_statement!(ctx::ExecutionContext, line::ParsedLine)

Dispatch a statement to its handler.
"""
function dispatch_statement!(ctx::ExecutionContext, line::ParsedLine)
    handler = get(STATEMENT_HANDLERS, line.statement_type, nothing)

    if handler !== nothing
        handler(ctx, line)
    elseif line.statement_type == :JULIA_CODE
        execute_julia_code!(ctx, line)
    elseif line.statement_type == :BLANK
        # Skip blank lines
    else
        error("Unknown statement type: $(line.statement_type)")
    end
end

"""
    execute_current_line!(ctx::ExecutionContext)

Execute the current line with error handling.
"""
function execute_current_line!(ctx::ExecutionContext)
    if ctx.pointer > length(ctx.procedure.lines)
        return
    end

    line = ctx.procedure.lines[ctx.pointer]

    try
        dispatch_statement!(ctx, line)
    catch e
        # Check for BREAK exception
        if occursin("__BREAK__", sprint(showerror, e))
            rethrow(e)  # Re-throw to be caught by loop handler
        end

        ctx.error_info = ErrorInfo(
            line.line_number,
            line.raw_text,
            sprint(showerror, e),
            sprint(showerror, e, catch_backtrace())
        )

        ctx.status = :failed
        rethrow(e)
    end
end

"""
    find_matching_end(lines::Vector{ParsedLine}, start_idx::Int) -> Int

Find the matching END for a block starting at start_idx.
"""
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
    execute_if_block!(ctx::ExecutionContext)

Execute an IF...ELSE...END block.
"""
function execute_if_block!(ctx::ExecutionContext)
    start_line = ctx.procedure.lines[ctx.pointer]
    condition_str = join(start_line.tokens[2:end], " ")

    # Evaluate condition
    condition = @eval begin
        using ...TMInterface: TM1, TM2, TM3, TM4, TM5, TM6, TM7, TM8, TM9, TM10
        $(Meta.parse(condition_str))
    end

    # Find ELSE and END
    else_idx = nothing
    end_idx = find_matching_end(ctx.procedure.lines, ctx.pointer)

    depth = 0
    for i in (ctx.pointer+1):(end_idx-1)
        line = ctx.procedure.lines[i]
        if line.statement_type in (:IF, :FOR, :WHILE)
            depth += 1
        elseif line.statement_type == :END
            depth -= 1
        elseif line.statement_type == :ELSE && depth == 0
            else_idx = i
            break
        end
    end

    if condition
        # Execute THEN branch
        ctx.pointer += 1
        end_of_then = else_idx === nothing ? end_idx : else_idx
        while ctx.pointer < end_of_then
            execute_current_line!(ctx)
            ctx.pointer += 1
        end
    else
        # Execute ELSE branch if it exists
        if else_idx !== nothing
            ctx.pointer = else_idx + 1
            while ctx.pointer < end_idx
                execute_current_line!(ctx)
                ctx.pointer += 1
            end
        end
    end

    # Jump to after END
    ctx.pointer = end_idx
end

"""
    execute_for_loop!(ctx::ExecutionContext)

Execute a FOR...END loop.
"""
function execute_for_loop!(ctx::ExecutionContext)
    start_line = ctx.procedure.lines[ctx.pointer]

    # Parse: FOR var IN start TO end
    if length(start_line.tokens) < 6
        error("FOR loop syntax: FOR <var> IN <start> TO <end>")
    end

    var_name = start_line.tokens[2]
    start_val = parse(Int, start_line.tokens[4])
    end_val = parse(Int, start_line.tokens[6])

    loop_start = ctx.pointer + 1
    loop_end = find_matching_end(ctx.procedure.lines, ctx.pointer)

    for i in start_val:end_val
        # Set loop variable in Main module
        @eval Main $(Symbol(var_name)) = $i

        # Execute loop body
        ctx.pointer = loop_start
        while ctx.pointer < loop_end
            try
                execute_current_line!(ctx)
                ctx.pointer += 1
            catch e
                if occursin("__BREAK__", sprint(showerror, e))
                    break
                end
                rethrow(e)
            end
        end
    end

    ctx.pointer = loop_end
end

"""
    execute_while_loop!(ctx::ExecutionContext)

Execute a WHILE...END loop.
"""
function execute_while_loop!(ctx::ExecutionContext)
    start_line = ctx.procedure.lines[ctx.pointer]
    condition_str = join(start_line.tokens[2:end], " ")

    loop_start = ctx.pointer + 1
    loop_end = find_matching_end(ctx.procedure.lines, ctx.pointer)

    while true
        # Evaluate condition
        condition = @eval begin
            using ...TMInterface: TM1, TM2, TM3, TM4, TM5, TM6, TM7, TM8, TM9, TM10
            $(Meta.parse(condition_str))
        end

        if !condition
            break
        end

        # Execute loop body
        ctx.pointer = loop_start
        while ctx.pointer < loop_end
            try
                execute_current_line!(ctx)
                ctx.pointer += 1
            catch e
                if occursin("__BREAK__", sprint(showerror, e))
                    break
                end
                rethrow(e)
            end
        end
    end

    ctx.pointer = loop_end
end

"""
    run_test(proc_name::String; mode::Symbol=:full, parent_ctx=nothing) -> ExecutionResult

Execute a procedure fully.
"""
function run_test(proc_name::String; mode::Symbol=:full, parent_ctx=nothing)::ExecutionResult
    proc = get_procedure(proc_name)
    if proc === nothing
        error("Procedure '$proc_name' not found")
    end

    start_time = time()

    # Use parent context if this is a CALL, otherwise create new context
    ctx = if parent_ctx !== nothing
        parent_ctx
    else
        ExecutionContext(
            proc, 1, Dict{String, Any}(), String[], Symbol[],
            LogEntry[], mode, :running, nothing
        )
    end

    # Save current procedure and pointer if this is a CALL
    saved_proc = ctx.procedure
    saved_pointer = ctx.pointer

    # Switch to called procedure
    ctx.procedure = proc
    ctx.pointer = 1

    try
        while ctx.pointer <= length(proc.lines) && ctx.status == :running
            line = proc.lines[ctx.pointer]

            # Handle block structures
            if line.statement_type == :IF
                execute_if_block!(ctx)
            elseif line.statement_type == :FOR
                execute_for_loop!(ctx)
            elseif line.statement_type == :WHILE
                execute_while_loop!(ctx)
            else
                execute_current_line!(ctx)
            end

            ctx.pointer += 1
        end

        if ctx.status == :running
            ctx.status = :completed
        end
    catch e
        if ctx.status != :aborted
            ctx.status = :failed
        end
    finally
        # Restore previous procedure state if this was a CALL
        if parent_ctx !== nothing
            ctx.procedure = saved_proc
            ctx.pointer = saved_pointer
        end
    end

    duration = time() - start_time

    return ExecutionResult(
        proc_name,
        ctx.status,
        ctx.execution_log,
        duration,
        ctx.error_info
    )
end

"""
    register_statement!(keyword::Symbol, handler::Function)

Register a new DSL statement handler.
"""
function register_statement!(keyword::Symbol, handler::Function)
    STATEMENT_HANDLERS[keyword] = handler
    push!(ACSParser.DSL_KEYWORDS, string(keyword))
    @info "Registered new statement: $keyword"
end

end # module ACSExecutor
