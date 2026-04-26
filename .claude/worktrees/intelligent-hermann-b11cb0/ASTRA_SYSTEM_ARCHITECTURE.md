# ASTRA - Automated Satellite Test & Reporting Application
## Complete System Architecture

---

## 1. High-Level System Architecture

### 1.1 System Overview

The ASTRA (Automated Satellite Test & Reporting Application) is a Julia-based satellite test automation platform. Engineers write test procedures in a beginner-friendly DSL (`.tst` files), which are parsed, validated, and executed by a Julia backend. A web-based frontend (Monaco Editor) provides editing, syntax checking, and step-by-step debugging.

### 1.2 Architecture Diagram

```
+================================================================+
|                        WEB BROWSER (Frontend)                   |
|                                                                 |
|  +-------------------+  +----------------+  +----------------+  |
|  | Monaco Editor     |  | Execution      |  | TM Monitor     |  |
|  | - .tst editing    |  | Panel          |  | Panel          |  |
|  | - Syntax highlight|  | - Run / Step   |  | - Live TM vals |  |
|  | - Autocomplete    |  | - Line highlight|  | - TM history   |  |
|  | - Error markers   |  | - Console out  |  | - Alarms       |  |
|  +-------------------+  +----------------+  +----------------+  |
|                              |                                   |
|                    WebSocket / HTTP API                           |
+==================================|===============================+
                                   |
+==================================|===============================+
|                     JULIA BACKEND (Server)                       |
|                                                                  |
|  +------------------------------------------------------------+ |
|  |                   API / Communication Layer                 | |
|  |         (HTTP.jl server + WebSocket channels)               | |
|  +------------------------------------------------------------+ |
|          |              |              |             |            |
|  +------------+  +------------+  +-----------+  +-----------+   |
|  | DSL Parser |  | Syntax     |  | Execution |  | Step      |   |
|  | & Loader   |  | Validator  |  | Engine    |  | Engine    |   |
|  +------------+  +------------+  +-----------+  +-----------+   |
|          |              |              |             |            |
|  +------------------------------------------------------------+ |
|  |                  Core Runtime Services                      | |
|  |  +----------+  +----------+  +----------+  +------------+  | |
|  |  | TM       |  | Command  |  | Variable |  | Procedure  |  | |
|  |  | Interface|  | Dispatch |  | Store    |  | Registry   |  | |
|  |  +----------+  +----------+  +----------+  +------------+  | |
|  +------------------------------------------------------------+ |
|          |              |                                        |
|  +------------------------------------------------------------+ |
|  |               Data & Logging Layer                          | |
|  |  +----------+  +----------+  +----------+                  | |
|  |  | Test     |  | Telemetry|  | Error    |                  | |
|  |  | Results  |  | Logger   |  | Logger   |                  | |
|  |  +----------+  +----------+  +----------+                  | |
|  +------------------------------------------------------------+ |
+==================================|===============================+
                                   |
+==================================|===============================+
|                  HARDWARE / SIMULATION LAYER                     |
|                                                                  |
|  +----------------+  +----------------+  +-------------------+  |
|  | TM Data Source |  | TC Interface   |  | Simulator         |  |
|  | (Telemetry     |  | (Telecommand   |  | (Software         |  |
|  |  Banks TM1-N)  |  |  SEND/SENDTCP) |  |  simulation mode) |  |
|  +----------------+  +----------------+  +-------------------+  |
+==================================================================+
```

### 1.3 Component Summary

| Component | Technology | Responsibility |
|-----------|-----------|----------------|
| Frontend | HTML/JS + Monaco Editor | Editing, visualization, step control |
| Communication | HTTP.jl + WebSockets | Bidirectional real-time messaging |
| DSL Parser | Custom Julia parser | Parse `.tst` files into IR (intermediate representation) |
| Syntax Validator | Julia module | Pre-execution static analysis |
| Execution Engine | Julia module | Full procedure execution |
| Step Engine | Julia module | Line-by-line controlled execution |
| TM Interface | Julia module | Telemetry data access (TM1.xyz_sts) |
| Command Dispatch | Julia module | SEND/SENDTCP routing to hardware |
| Procedure Registry | Julia Dict | Stores loaded procedures by TEST_NAME |
| Data/Logging | Julia + SQLite/JSON | Test results, telemetry logs, error logs |

---

## 2. DSL Specification (ASTRA Language)

### 2.1 File Structure

Each `.tst` file contains exactly **one** `TEST_NAME` declaration on the first line, followed by the procedure body:

```
TEST_NAME 4rw-config1
PRE_TEST_REQ TM1.xyz_sts == "on"
SEND START_RW
WAIT 5
CHECK TM1.RW_STATUS == "READY"
```

### 2.2 Statement Reference

| Statement | Syntax | Description |
|-----------|--------|-------------|
| `TEST_NAME` | `TEST_NAME <name>` | Declares procedure name (first line only) |
| `PRE_TEST_REQ` | `PRE_TEST_REQ <condition>` | Pre-condition that must be true before test runs |
| `SEND` | `SEND <command> [args...]` | Send telecommand to hardware |
| `SENDTCP` | `SENDTCP <host> <port> <data>` | Send command via TCP |
| `WAIT` | `WAIT <seconds>` | Wait for a duration |
| `WAIT UNTIL` | `WAIT UNTIL <condition> TIMEOUT <seconds>` | Wait until condition or timeout |
| `CHECK` | `CHECK <condition> [WITHIN <seconds>]` | Verify telemetry condition |
| `EXPECTED` | `EXPECTED <condition>` | Assert expected telemetry state |
| `ALERT_MSG` | `ALERT_MSG "<message>"` | Display alert to operator |
| `ABORT_TEST` | `ABORT_TEST` | Immediately stop test execution |
| `CALL` | `CALL <procedure_name>` | Execute another procedure |
| `BREAK` | `BREAK` | Exit current loop |

### 2.3 Control Flow

```
IF <condition>
    <statements>
ELSE
    <statements>
END

FOR <var> IN <start> TO <end>
    <statements>
END

WHILE <condition>
    <statements>
END
```

All blocks terminate with `END` (Julia-style), not `ENDIF`/`ENDFOR`.

### 2.4 Optional Blocks

```
ON_FAIL
    <statements>
END

ON_TIMEOUT
    <statements>
END
```

These attach to the preceding `WAIT`, `CHECK`, or `WAIT UNTIL` statement.

### 2.5 TM References

Telemetry references use the form `TM<bank>.<mnemonic>`:

```
TM1.xyz_sts
TM2.rw_speed
TM1.voltage_bus
```

These resolve at runtime to live telemetry values from the TM Interface module.

### 2.6 Inline Julia Code

Any line that does not start with a recognized DSL keyword is treated as inline Julia code:

```
TEST_NAME 4rw-config1
PRE_TEST_REQ TM1.xyz_sts == "on"
SEND START_RW
WAIT 5

# Inline Julia computation
adjusted = TM1.abc + 10
println("Adjusted value: ", adjusted)

IF adjusted > 50
    SEND START_RW
ELSE
    ALERT_MSG "Value too low"
END
```

### 2.7 Comments

Lines starting with `#` or `//` are treated as comments and ignored by the parser.

---

## 3. Module Architecture & Responsibilities

### 3.1 DSL Parser & Loader (`ACSParser` module)

**File:** `src/parser/ACSParser.jl`

**Responsibility:** Read `.tst` files and produce an intermediate representation (IR).

```julia
module ACSParser

export load_file, ParsedProcedure, ParsedLine

# Represents a single parsed line with metadata
struct ParsedLine
    line_number::Int        # Original .tst file line number
    raw_text::String        # Original text
    statement_type::Symbol  # :SEND, :WAIT, :IF, :JULIA_CODE, etc.
    tokens::Vector{String}  # Tokenized parts
    block_depth::Int        # Nesting depth for validation
end

# Represents a complete parsed procedure
struct ParsedProcedure
    name::String
    source_file::String
    lines::Vector{ParsedLine}
end

# Registry: TEST_NAME -> ParsedProcedure
const PROCEDURE_REGISTRY = Dict{String, ParsedProcedure}()

# Known DSL keywords (extensible)
const DSL_KEYWORDS = Set([
    "TEST_NAME", "PRE_TEST_REQ", "SEND", "SENDTCP",
    "WAIT", "CHECK", "EXPECTED", "ALERT_MSG",
    "ABORT_TEST", "CALL", "BREAK",
    "IF", "ELSE", "END", "FOR", "WHILE",
    "ON_FAIL", "ON_TIMEOUT"
])

function load_file(filename::String)::ParsedProcedure
    # 1. Read all lines with line numbers
    # 2. Skip comments and blank lines
    # 3. Extract TEST_NAME from first non-comment line
    # 4. Classify each line by statement type
    # 5. Track block depth (IF/FOR/WHILE increase, END decreases)
    # 6. Store in PROCEDURE_REGISTRY
    # 7. Return ParsedProcedure
end

# Support loading multiple files for cross-procedure CALL
function load_directory(dir::String)
    for file in readdir(dir)
        if endswith(file, ".tst")
            load_file(joinpath(dir, file))
        end
    end
end

end # module
```

**Key Design Decisions:**
- Each `ParsedLine` retains the original line number for error mapping
- Statement classification happens at parse time, not execution time
- `DSL_KEYWORDS` is a `Set` that can be extended at runtime to add new statement types
- The `PROCEDURE_REGISTRY` is a global dictionary keyed by `TEST_NAME`

### 3.2 Syntax Validator (`ACSValidator` module)

**File:** `src/validator/ACSValidator.jl`

**Responsibility:** Static analysis before execution. No side effects.

```julia
module ACSValidator

export validate_procedure, ValidationError

struct ValidationError
    file::String
    line_number::Int
    line_text::String
    message::String
    severity::Symbol  # :error, :warning
end

function validate_procedure(proc::ParsedProcedure)::Vector{ValidationError}
    errors = ValidationError[]

    # 1. Block matching: every IF/FOR/WHILE must have a matching END
    validate_block_matching!(proc, errors)

    # 2. CALL targets: every CALL <name> must reference a known TEST_NAME
    validate_call_targets!(proc, errors)

    # 3. BREAK placement: BREAK only inside FOR/WHILE
    validate_break_placement!(proc, errors)

    # 4. ON_FAIL/ON_TIMEOUT placement: must follow WAIT/CHECK
    validate_handler_placement!(proc, errors)

    # 5. TM reference format: TM<digits>.<name> pattern check
    validate_tm_references!(proc, errors)

    # 6. Inline Julia syntax: Meta.parse() without eval
    validate_julia_syntax!(proc, errors)

    # 7. PRE_TEST_REQ must appear before any SEND/WAIT/CHECK
    validate_pretest_order!(proc, errors)

    return errors
end

# Block matching with depth tracking
function validate_block_matching!(proc, errors)
    stack = Tuple{Symbol, Int}[]  # (block_type, line_number)
    for line in proc.lines
        if line.statement_type in (:IF, :FOR, :WHILE)
            push!(stack, (line.statement_type, line.line_number))
        elseif line.statement_type == :END
            if isempty(stack)
                push!(errors, ValidationError(
                    proc.source_file, line.line_number,
                    line.raw_text, "Unexpected END without matching block opener",
                    :error
                ))
            else
                pop!(stack)
            end
        end
    end
    for (block_type, line_no) in stack
        push!(errors, ValidationError(
            proc.source_file, line_no, "",
            "Missing END for $block_type block", :error
        ))
    end
end

# Validate inline Julia code parses without execution
function validate_julia_syntax!(proc, errors)
    for line in proc.lines
        if line.statement_type == :JULIA_CODE
            try
                Meta.parse(line.raw_text)
            catch e
                push!(errors, ValidationError(
                    proc.source_file, line.line_number,
                    line.raw_text, "Invalid Julia syntax: $e",
                    :error
                ))
            end
        end
    end
end

end # module
```

**Validation Checks:**

| Check | What It Catches |
|-------|----------------|
| Block matching | Missing `END`, extra `END`, mismatched nesting |
| CALL targets | `CALL nonexistent_proc` -> "procedure not found" |
| BREAK placement | `BREAK` outside `FOR`/`WHILE` loop |
| Handler placement | `ON_FAIL` not after `WAIT`/`CHECK` |
| TM references | `TM1.` without mnemonic, `TMX.abc` (non-numeric bank) |
| Julia syntax | `Meta.parse()` catches syntax errors without executing |
| PRE_TEST_REQ order | PRE_TEST_REQ after SEND/WAIT is a warning |

### 3.3 Execution Engine (`ACSExecutor` module)

**File:** `src/executor/ACSExecutor.jl`

**Responsibility:** Execute a parsed procedure fully or step by step.

```julia
module ACSExecutor

export run_test, ExecutionResult, ExecutionContext

# Execution context holds runtime state
mutable struct ExecutionContext
    procedure::ParsedProcedure
    pointer::Int                     # Current line index
    variables::Dict{String, Any}     # User-defined variables
    call_stack::Vector{String}       # For recursive CALL detection
    block_stack::Vector{Symbol}      # Current nesting context
    execution_log::Vector{LogEntry}  # Timestamped execution log
    mode::Symbol                     # :full, :step, :simulation
    status::Symbol                   # :running, :paused, :completed,
                                     # :failed, :aborted
    error_info::Union{Nothing, ErrorInfo}
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

# Full execution
function run_test(proc_name::String; mode::Symbol=:full)::ExecutionResult
    proc = PROCEDURE_REGISTRY[proc_name]
    ctx = ExecutionContext(proc, 1, Dict(), String[], Symbol[],
                          LogEntry[], mode, :running, nothing)

    while ctx.pointer <= length(proc.lines) && ctx.status == :running
        execute_current_line!(ctx)
        ctx.pointer += 1
    end

    return build_result(ctx)
end

# Core line execution with error handling
function execute_current_line!(ctx::ExecutionContext)
    line = ctx.procedure.lines[ctx.pointer]

    try
        dispatch_statement!(ctx, line)
    catch e
        ctx.error_info = ErrorInfo(
            line.line_number,
            line.raw_text,
            sprint(showerror, e),
            sprint(showerror, e, catch_backtrace())
        )

        # Check for ON_FAIL handler
        if has_on_fail_handler(ctx)
            execute_on_fail!(ctx)
        else
            ctx.status = :failed
        end
    end
end

# Statement dispatch table (extensible)
const STATEMENT_HANDLERS = Dict{Symbol, Function}()

function dispatch_statement!(ctx, line)
    handler = get(STATEMENT_HANDLERS, line.statement_type, nothing)
    if handler !== nothing
        handler(ctx, line)
    elseif line.statement_type == :JULIA_CODE
        execute_julia_code!(ctx, line)
    else
        error("Unknown statement type: $(line.statement_type)")
    end
end

# Register built-in handlers
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

# Adding new statement types: register handler functions
function register_statement!(keyword::Symbol, handler::Function)
    STATEMENT_HANDLERS[keyword] = handler
    push!(ACSParser.DSL_KEYWORDS, string(keyword))
end

end # module
```

**Key Design: Extensible Statement Handlers**

New DSL statements can be added by calling:
```julia
ACSExecutor.register_statement!(:MY_NEW_CMD, function(ctx, line)
    # custom logic
end)
```

This registers both the handler and the keyword, so the parser and validator also recognize it.

### 3.4 Step Execution Engine (`ASTRAStepEngine` module)

**File:** `src/executor/ASTRAStepEngine.jl`

**Responsibility:** Controlled line-by-line execution for GUI debugging.

```julia
module ASTRAStepEngine

export StepSession, step_next!, step_into!, step_over!, step_reset!,
       get_current_state

# Session state for step execution
mutable struct StepSession
    id::String                         # Unique session ID
    context::ExecutionContext           # Reuses executor context
    breakpoints::Set{Int}              # Line numbers with breakpoints
    call_depth::Int                    # Current CALL nesting depth
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

function step_next!(session::StepSession)::StepState
    ctx = session.context

    if ctx.pointer > length(ctx.procedure.lines)
        ctx.status = :completed
        return get_current_state(session)
    end

    line = ctx.procedure.lines[ctx.pointer]
    output = capture_output() do
        execute_current_line!(ctx)
    end

    ctx.pointer += 1

    # Skip into CALL procedures or step over them based on mode
    # Handle block advancement (skip FALSE branch of IF, etc.)

    return get_current_state(session, output)
end

# Step into a CALL statement (follow into called procedure)
function step_into!(session::StepSession)::StepState
    # Push current context onto call stack
    # Load called procedure
    # Reset pointer to first line of called procedure
end

# Step over a CALL statement (execute entire called procedure, return)
function step_over!(session::StepSession)::StepState
    # Execute called procedure fully
    # Continue from line after CALL
end

# Reset to beginning
function step_reset!(session::StepSession)
    session.context.pointer = 1
    session.context.status = :paused
    empty!(session.context.variables)
    empty!(session.context.call_stack)
end

function get_current_state(session, output="")::StepState
    ctx = session.context
    line = ctx.pointer <= length(ctx.procedure.lines) ?
           ctx.procedure.lines[ctx.pointer] : nothing

    return StepState(
        session.id,
        ctx.procedure.name,
        line !== nothing ? line.line_number : -1,
        line !== nothing ? line.raw_text : "",
        ctx.status,
        copy(ctx.variables),
        copy(ctx.call_stack),
        copy(ctx.block_stack),
        output
    )
end

end # module
```

**Step Execution Features for GUI:**

| Feature | Description |
|---------|-------------|
| Step Next | Execute one line, advance pointer |
| Step Into | Follow into CALL procedure line by line |
| Step Over | Execute CALL procedure fully, return to caller |
| Step Reset | Reset to first line of procedure |
| Breakpoints | Set line numbers where execution pauses |
| State Snapshot | Returns current line, variables, call stack, output |

### 3.5 TM Interface (`TMInterface` module)

**File:** `src/tm/TMInterface.jl`

**Responsibility:** Provide access to telemetry data using `TM<bank>.<mnemonic>` syntax.

```julia
module TMInterface

export TMBank, get_tm_value, set_tm_value, resolve_tm_ref

# TM Bank: a named collection of telemetry parameters
mutable struct TMBank
    bank_id::Int
    parameters::Dict{Symbol, Any}     # mnemonic -> current value
    metadata::Dict{Symbol, Dict}      # mnemonic -> {type, unit, range, etc.}
    last_updated::Dict{Symbol, DateTime}
end

# Registry of TM banks
const TM_BANKS = Dict{Int, TMBank}()

# Resolve "TM1.xyz_sts" string to actual value
function resolve_tm_ref(ref::String)::Any
    m = match(r"^TM(\d+)\.(\w+)$", ref)
    if m === nothing
        error("Invalid TM reference: $ref")
    end
    bank_id = parse(Int, m.captures[1])
    mnemonic = Symbol(m.captures[2])
    return get_tm_value(bank_id, mnemonic)
end

function get_tm_value(bank_id::Int, mnemonic::Symbol)::Any
    bank = get(TM_BANKS, bank_id, nothing)
    if bank === nothing
        error("TM bank TM$bank_id not found")
    end
    val = get(bank.parameters, mnemonic, nothing)
    if val === nothing
        error("Mnemonic $mnemonic not found in TM$bank_id")
    end
    return val
end

function set_tm_value(bank_id::Int, mnemonic::Symbol, value::Any)
    if !haskey(TM_BANKS, bank_id)
        TM_BANKS[bank_id] = TMBank(bank_id, Dict(), Dict(), Dict())
    end
    TM_BANKS[bank_id].parameters[mnemonic] = value
    TM_BANKS[bank_id].last_updated[mnemonic] = now()
end

# Get all known mnemonics for autocomplete support
function get_all_mnemonics()::Vector{String}
    result = String[]
    for (bank_id, bank) in TM_BANKS
        for mnemonic in keys(bank.parameters)
            push!(result, "TM$bank_id.$mnemonic")
        end
    end
    return sort(result)
end

# Julia property access syntax: TM1.xyz_sts
# Implemented via a special struct with Base.getproperty overload
struct TMAccessor
    bank_id::Int
end

function Base.getproperty(tm::TMAccessor, name::Symbol)
    return get_tm_value(getfield(tm, :bank_id), name)
end

# Pre-define TM1 through TM10 as global accessors
for i in 1:10
    @eval const $(Symbol("TM$i")) = TMAccessor($i)
end

end # module
```

**TM Access Mechanism:**

The `TMAccessor` struct with `Base.getproperty` override allows DSL code to write `TM1.xyz_sts` directly. When Julia encounters `TM1.xyz_sts`, it calls `getproperty(TM1, :xyz_sts)` which looks up the telemetry value from the TM bank. This is transparent to the user -- no quoting, no macros.

### 3.6 Command Dispatch (`CommandDispatch` module)

**File:** `src/commands/CommandDispatch.jl`

**Responsibility:** Route SEND/SENDTCP commands to hardware or simulator.

```julia
module CommandDispatch

export send_command, send_tcp_command, CommandResult

struct CommandResult
    command::String
    status::Symbol      # :sent, :acknowledged, :failed, :timeout
    response::String
    timestamp::DateTime
end

# Abstraction over hardware vs simulation
abstract type CommandTarget end

struct HardwareTarget <: CommandTarget
    connection::Any     # Hardware connection handle
end

struct SimulatorTarget <: CommandTarget
    sim_state::Dict     # Simulated state
end

# Active target (switchable between hardware and simulation)
const ACTIVE_TARGET = Ref{CommandTarget}()

function send_command(command::String, args...)::CommandResult
    target = ACTIVE_TARGET[]
    return dispatch_to_target(target, command, args...)
end

function send_tcp_command(host::String, port::Int, data::String)::CommandResult
    # TCP socket connection and send
end

# Switch between simulation and live modes
function set_mode!(mode::Symbol)
    if mode == :simulation
        ACTIVE_TARGET[] = SimulatorTarget(Dict())
    elseif mode == :hardware
        ACTIVE_TARGET[] = HardwareTarget(establish_hw_connection())
    end
end

end # module
```

### 3.7 Communication Layer (`ASTRAServer` module)

**File:** `src/server/ASTRAServer.jl`

**Responsibility:** HTTP + WebSocket server bridging frontend and Julia backend.

```julia
module ASTRAServer

using HTTP, JSON3

export start_server

# HTTP API Endpoints
# ==================
#
# POST   /api/load          Load a .tst file
# POST   /api/validate      Validate loaded procedure (syntax check)
# POST   /api/run           Run full procedure
# POST   /api/step/start    Start step session
# POST   /api/step/next     Execute next step
# POST   /api/step/into     Step into CALL
# POST   /api/step/over     Step over CALL
# POST   /api/step/reset    Reset step session
# GET    /api/state         Get current execution state
# GET    /api/procedures    List all loaded procedures
# GET    /api/tm            Get current TM values
# GET    /api/tm/mnemonics  Get all known TM mnemonics (for autocomplete)
# GET    /api/results       Get test results
#
# WebSocket /ws/events      Real-time event stream

function start_server(port::Int=8080)
    router = HTTP.Router()

    # File operations
    HTTP.register!(router, "POST", "/api/load", handle_load)
    HTTP.register!(router, "POST", "/api/validate", handle_validate)

    # Execution
    HTTP.register!(router, "POST", "/api/run", handle_run)

    # Step execution
    HTTP.register!(router, "POST", "/api/step/start", handle_step_start)
    HTTP.register!(router, "POST", "/api/step/next", handle_step_next)
    HTTP.register!(router, "POST", "/api/step/into", handle_step_into)
    HTTP.register!(router, "POST", "/api/step/over", handle_step_over)
    HTTP.register!(router, "POST", "/api/step/reset", handle_step_reset)

    # State / Data
    HTTP.register!(router, "GET", "/api/state", handle_get_state)
    HTTP.register!(router, "GET", "/api/procedures", handle_list_procedures)
    HTTP.register!(router, "GET", "/api/tm", handle_get_tm)
    HTTP.register!(router, "GET", "/api/tm/mnemonics", handle_get_mnemonics)
    HTTP.register!(router, "GET", "/api/results", handle_get_results)

    # WebSocket for real-time events
    HTTP.register!(router, "/ws/events", handle_websocket)

    HTTP.serve(router, "0.0.0.0", port)
end

# WebSocket event types sent to frontend:
# {
#   "type": "step_update",
#   "data": {
#     "line_number": 12,
#     "line_text": "SEND START_RW",
#     "status": "paused",
#     "output": "SEND: START_RW",
#     "variables": {"adjusted": 60},
#     "call_stack": ["4rw-config3", "4rw-config1"]
#   }
# }
#
# {
#   "type": "tm_update",
#   "data": {"TM1.xyz_sts": "on", "TM1.abc": 42}
# }
#
# {
#   "type": "error",
#   "data": {
#     "line_number": 15,
#     "message": "PRE_TEST_REQ failed: TM1.xyz_sts == \"on\""
#   }
# }
#
# {
#   "type": "test_complete",
#   "data": {"test_name": "4rw-config1", "status": "passed", "duration": 12.5}
# }

end # module
```

**API Response Formats:**

```json
// POST /api/validate response
{
  "valid": false,
  "errors": [
    {
      "file": "4rwconfig-1.tst",
      "line_number": 12,
      "line_text": "IF TM1.VOLT >",
      "message": "Incomplete expression after IF",
      "severity": "error"
    }
  ]
}

// POST /api/step/next response
{
  "line_number": 8,
  "line_text": "SEND START_RW",
  "status": "paused",
  "output": "SEND: START_RW",
  "variables": {"i": 2, "adjusted": 55},
  "call_stack": ["4rw-config3"]
}

// GET /api/tm response
{
  "TM1.xyz_sts": "on",
  "TM1.abc": 42,
  "TM1.voltage_bus": 28.3,
  "TM2.rw_speed": 1500
}
```

---

## 4. Data Flow

### 4.1 Procedure Load & Validate Flow

```
User writes .tst file in Monaco Editor
         |
         v
[Frontend] POST /api/load {content: "TEST_NAME 4rw-config1\n..."}
         |
         v
[ACSParser.load_from_string()]
  - Reads lines, extracts TEST_NAME
  - Classifies each line (statement type, tokens, line numbers)
  - Stores ParsedProcedure in PROCEDURE_REGISTRY
         |
         v
[ACSValidator.validate_procedure()]
  - Block matching (IF/FOR/WHILE <-> END)
  - CALL target resolution
  - TM reference format check
  - Julia code syntax check (Meta.parse)
  - BREAK placement validation
         |
         v
[Response] { valid: true/false, errors: [...] }
         |
         v
[Frontend] Monaco displays error markers on specific lines
```

### 4.2 Full Execution Flow

```
User clicks "Run Test" in GUI
         |
         v
[Frontend] POST /api/run {test_name: "4rw-config1"}
         |
         v
[ACSExecutor.run_test("4rw-config1")]
         |
    +----+----+
    | For each line in procedure:
    |    |
    |    v
    | [dispatch_statement!()]
    |    |
    |    +---> SEND ---------> [CommandDispatch.send_command()]
    |    |                              |
    |    |                              v
    |    |                     [Hardware / Simulator]
    |    |
    |    +---> WAIT ---------> [sleep() or poll with timeout]
    |    |
    |    +---> CHECK --------> [TMInterface.resolve_tm_ref()]
    |    |                              |
    |    |                              v
    |    |                     [Compare with expected value]
    |    |
    |    +---> CALL ---------> [Recursive: run_test(called_proc)]
    |    |
    |    +---> IF/FOR/WHILE -> [Block execution with nesting]
    |    |
    |    +---> JULIA_CODE ---> [eval(Meta.parse(line))]
    |    |
    |    v
    | [Log result to execution_log]
    | [Send WebSocket event to frontend]
    +----+
         |
         v
[ExecutionResult] { status, log, duration, error }
         |
         v
[DataLogger] Store results in test_results database
         |
         v
[WebSocket] { type: "test_complete", data: {...} }
         |
         v
[Frontend] Display results summary
```

### 4.3 Step Execution Flow

```
User clicks "Step" in GUI
         |
         v
[Frontend] POST /api/step/next {session_id: "abc123"}
         |
         v
[ASTRAStepEngine.step_next!(session)]
  - Get current line at pointer
  - Execute single statement
  - Capture output
  - Advance pointer (handling block logic)
  - Build StepState snapshot
         |
         v
[StepState] {
  current_line_number: 8,
  current_line_text: "SEND START_RW",
  variables: {adjusted: 55},
  call_stack: ["4rw-config3"],
  output: "SEND: START_RW"
}
         |
         v
[WebSocket] { type: "step_update", data: StepState }
         |
         v
[Frontend Monaco Editor]
  - Highlight line 8 with decoration
  - Show variables in watch panel
  - Append output to console panel
  - Update call stack display
```

### 4.4 TM Data Flow

```
[Hardware TM Stream] ----> [TMInterface]
  (periodic or event-based)      |
                                 v
                          [TM_BANKS Dict]
                           TM1.xyz_sts = "on"
                           TM1.abc = 42
                           TM2.rw_speed = 1500
                                 |
        +------------------------+------------------------+
        |                        |                        |
        v                        v                        v
  [DSL Execution]          [WebSocket Push]        [REST API GET]
  TM1.xyz_sts == "on"      periodic TM updates     /api/tm
  resolves to true         to frontend monitor     on-demand query
```

---

## 5. Error Handling & Reporting Design

### 5.1 Error Categories

| Category | When | Example | User-Facing Message |
|----------|------|---------|---------------------|
| Parse Error | Loading .tst file | Unrecognized line format | `Line 12: Unrecognized statement "SNED START_RW". Did you mean "SEND"?` |
| Validation Error | Syntax check | Missing END | `Line 5: IF block opened but no matching END found` |
| Validation Warning | Syntax check | PRE_TEST_REQ after SEND | `Line 8: PRE_TEST_REQ should appear before any SEND statements` |
| Runtime Error | During execution | TM value doesn't match | `Line 15: CHECK failed: TM1.xyz_sts == "on" (actual: "off")` |
| Julia Error | Inline code execution | Division by zero | `Line 20: Julia runtime error: DivideError()` |
| Timeout Error | WAIT UNTIL | Condition not met | `Line 10: WAIT UNTIL timed out after 30s: TM1.rw_speed > 100` |
| CALL Error | Cross-procedure call | Procedure not found | `Line 6: CALL "4rw-config9" - procedure not found` |
| Hardware Error | SEND/SENDTCP | Connection lost | `Line 12: SEND failed: hardware connection timeout` |

### 5.2 Error Structure

Every error includes:

```julia
struct ASTRAError
    category::Symbol          # :parse, :validation, :runtime, :julia,
                              # :timeout, :call, :hardware
    file::String              # Source .tst filename
    line_number::Int          # Original .tst line number
    line_text::String         # The exact line that caused the error
    message::String           # Human-readable description
    suggestion::String        # Optional fix suggestion
    severity::Symbol          # :error, :warning, :info
    timestamp::DateTime
    call_stack::Vector{String} # Procedure call chain if inside CALL
end
```

### 5.3 Error Reporting to GUI

Errors are sent via WebSocket as structured JSON:

```json
{
  "type": "error",
  "data": {
    "category": "runtime",
    "file": "4rwconfig-1.tst",
    "line_number": 15,
    "line_text": "CHECK TM1.xyz_sts == \"on\"",
    "message": "CHECK failed: TM1.xyz_sts == \"on\" (actual: \"off\")",
    "suggestion": "Verify that the subsystem is powered on before running this check",
    "severity": "error",
    "call_stack": ["4rw-config3", "4rw-config1"]
  }
}
```

Monaco Editor renders this as:
- Red squiggly underline on line 15
- Hover tooltip showing the error message
- Error marker in the minimap gutter
- Entry in the Problems panel

### 5.4 "Did You Mean?" Suggestions

For common typos in DSL keywords, the parser uses Levenshtein distance:

```julia
function suggest_keyword(unknown::String)::Union{String, Nothing}
    best_match = nothing
    best_dist = 3  # Max edit distance threshold
    for kw in DSL_KEYWORDS
        d = levenshtein_distance(uppercase(unknown), kw)
        if d < best_dist
            best_dist = d
            best_match = kw
        end
    end
    return best_match
end
```

Example: `SNED` -> `Did you mean "SEND"?`

### 5.5 Line Number Mapping Through CALL Chains

When an error occurs inside a called procedure, the error includes the full call chain:

```
Error at line 8 of "4rw-config1" (called from line 3 of "4rw-config3"):
  CHECK failed: TM1.xyz_sts == "on" (actual: "off")

Call stack:
  4rw-config3:3  ->  CALL 4rw-config1
  4rw-config1:8  ->  CHECK TM1.xyz_sts == "on"   <-- ERROR
```

---

## 6. Frontend Architecture (Monaco Editor GUI)

### 6.1 Frontend Components

```
+------------------------------------------------------------------+
|  Toolbar                                                          |
|  [Open] [Save] [Validate] [Run] [Step] [Step Into] [Reset] [Stop]|
+------------------------------------------------------------------+
|                    |                      |                        |
|  Monaco Editor     |  Execution Panel     |  TM Monitor Panel     |
|  (left, 50%)       |  (top-right, 25%)    |  (bottom-right, 25%)  |
|                    |                      |                        |
|  1  TEST_NAME ...  |  Console Output:     |  TM1.xyz_sts: "on"    |
|  2  PRE_TEST_REQ.. |  > SEND: START_RW    |  TM1.abc: 42          |
|  3  SEND START_RW  |  > WAIT: 5           |  TM1.voltage: 28.3    |
|> 4  WAIT 5     <-- |  > CHECK: OK         |  TM2.rw_speed: 1500   |
|  5  CHECK ...      |                      |                        |
|  6  IF adjusted... |  Variables Watch:     |  Alerts:               |
|  7    SEND ...     |  adjusted = 55       |  [!] Voltage low       |
|  8  ELSE           |  i = 2               |                        |
|  9    ALERT_MSG .. |                      |                        |
| 10  END            |  Call Stack:          |                        |
|                    |  4rw-config3:3        |                        |
|                    |  4rw-config1:4 <--    |                        |
+------------------------------------------------------------------+
|  Problems Panel                                                   |
|  Line 12: Missing END for IF block                                |
|  Line 18: CALL "4rw-configX" - procedure not found                |
+------------------------------------------------------------------+
```

### 6.2 Custom Language Definition for Monaco

```javascript
// Register ASTRA language
monaco.languages.register({ id: 'ASTRA' });

monaco.languages.setMonarchTokensProvider('ASTRA', {
    keywords: [
        'TEST_NAME', 'PRE_TEST_REQ', 'SEND', 'SENDTCP',
        'WAIT', 'UNTIL', 'TIMEOUT', 'CHECK', 'WITHIN',
        'EXPECTED', 'ALERT_MSG', 'ABORT_TEST', 'BREAK',
        'CALL', 'IF', 'ELSE', 'END', 'FOR', 'IN', 'TO',
        'WHILE', 'ON_FAIL', 'ON_TIMEOUT', 'AND', 'OR', 'NOT'
    ],
    tokenizer: {
        root: [
            [/\b(TEST_NAME)\b/, 'keyword.declaration'],
            [/\b(IF|ELSE|END|FOR|WHILE|IN|TO)\b/, 'keyword.control'],
            [/\b(SEND|SENDTCP|WAIT|CHECK|EXPECTED|CALL)\b/, 'keyword.command'],
            [/\b(PRE_TEST_REQ|ALERT_MSG|ABORT_TEST|BREAK)\b/, 'keyword.action'],
            [/\b(ON_FAIL|ON_TIMEOUT|UNTIL|TIMEOUT|WITHIN)\b/, 'keyword.handler'],
            [/\b(AND|OR|NOT)\b/, 'keyword.operator'],
            [/\bTM\d+\.\w+\b/, 'variable.tm'],
            [/"[^"]*"/, 'string'],
            [/\d+(\.\d+)?/, 'number'],
            [/#.*$/, 'comment'],
            [/\/\/.*$/, 'comment'],
            [/[=><!]+/, 'operator'],
        ]
    }
});
```

### 6.3 Autocomplete Provider

```javascript
monaco.languages.registerCompletionItemProvider('ASTRA', {
    provideCompletionItems: async (model, position) => {
        const word = model.getWordUntilPosition(position);
        let suggestions = [];

        // DSL keyword suggestions
        const keywords = ['SEND', 'SENDTCP', 'WAIT', 'CHECK', ...];
        for (const kw of keywords) {
            suggestions.push({
                label: kw,
                kind: monaco.languages.CompletionItemKind.Keyword,
                insertText: kw + ' '
            });
        }

        // TM variable suggestions (fetched from backend)
        const tmResponse = await fetch('/api/tm/mnemonics');
        const mnemonics = await tmResponse.json();
        for (const tm of mnemonics) {
            suggestions.push({
                label: tm,
                kind: monaco.languages.CompletionItemKind.Variable,
                insertText: tm
            });
        }

        // Procedure name suggestions for CALL
        if (word.word.startsWith('CALL')) {
            const procResponse = await fetch('/api/procedures');
            const procedures = await procResponse.json();
            for (const proc of procedures) {
                suggestions.push({
                    label: 'CALL ' + proc,
                    kind: monaco.languages.CompletionItemKind.Function,
                    insertText: 'CALL ' + proc
                });
            }
        }

        return { suggestions };
    }
});
```

---

## 7. Project Structure

```
MF_AUTOMATION_JULIA_2026/
|
+-- src/
|   +-- ASTRA.jl                     # Main module entry point
|   |
|   +-- parser/
|   |   +-- ACSParser.jl            # DSL file parser & loader
|   |   +-- Tokenizer.jl            # Line tokenization
|   |
|   +-- validator/
|   |   +-- ACSValidator.jl         # Static syntax validation
|   |   +-- BlockChecker.jl         # IF/FOR/WHILE block matching
|   |   +-- TMValidator.jl          # TM reference validation
|   |
|   +-- executor/
|   |   +-- ACSExecutor.jl          # Full execution engine
|   |   +-- ASTRAStepEngine.jl        # Step-by-step execution
|   |   +-- StatementHandlers.jl    # Handler functions for each DSL statement
|   |   +-- BlockExecutor.jl        # IF/FOR/WHILE block execution logic
|   |   +-- JuliaEvaluator.jl       # Inline Julia code evaluation
|   |
|   +-- tm/
|   |   +-- TMInterface.jl          # Telemetry data access
|   |   +-- TMAccessor.jl           # TM1.xyz_sts property access
|   |   +-- TMSimulator.jl          # Simulated TM data for testing
|   |
|   +-- commands/
|   |   +-- CommandDispatch.jl       # SEND/SENDTCP routing
|   |   +-- HardwareTarget.jl        # Hardware connection
|   |   +-- SimulatorTarget.jl       # Simulation mode
|   |
|   +-- server/
|   |   +-- ASTRAServer.jl            # HTTP + WebSocket server
|   |   +-- APIHandlers.jl          # REST endpoint handlers
|   |   +-- WSEventBus.jl           # WebSocket event broadcasting
|   |
|   +-- logging/
|       +-- DataLogger.jl            # Test results & telemetry logging
|       +-- ErrorLogger.jl           # Error log storage
|
+-- frontend/
|   +-- index.html                   # Main web page
|   +-- css/
|   |   +-- styles.css               # Layout and theme
|   |   +-- step-highlight.css       # Line highlighting styles
|   +-- js/
|   |   +-- app.js                   # Main application logic
|   |   +-- editor.js                # Monaco Editor setup
|   |   +-- ASTRA-language.js         # Language definition & highlighting
|   |   +-- autocomplete.js          # Autocomplete provider
|   |   +-- websocket.js             # WebSocket connection manager
|   |   +-- execution-panel.js       # Execution controls & output
|   |   +-- tm-monitor.js            # TM value display
|   |   +-- problems-panel.js        # Error/warning display
|
+-- test/
|   +-- test_parser.jl               # Parser unit tests
|   +-- test_validator.jl            # Validator unit tests
|   +-- test_executor.jl             # Execution engine unit tests
|   +-- test_step_engine.jl          # Step execution tests
|   +-- test_tm_interface.jl         # TM access tests
|   +-- fixtures/
|       +-- valid_procedure.tst       # Valid test fixture
|       +-- syntax_error.tst          # Error test fixture
|       +-- nested_blocks.tst         # Nested block test fixture
|       +-- cross_call.tst            # Cross-procedure CALL fixture
|
+-- procedures/                       # User test procedure files
|   +-- 4rwconfig-1.tst
|   +-- 4rwconfig-2.tst
|   +-- 4rwconfig-3.tst
|
+-- docs/
|   +-- dsl_reference.md              # DSL syntax reference
|   +-- user_guide.md                 # User guide for engineers
|
+-- Project.toml                      # Julia package manifest
+-- Manifest.toml                     # Julia dependency lock
```

---

## 8. Execution Modes

### 8.1 Full Execution Mode

- Runs all lines of a procedure sequentially
- Handles block logic (IF/FOR/WHILE) automatically
- Follows CALL statements into other procedures
- Stops on ABORT_TEST or unhandled error
- Returns `ExecutionResult` with full log

### 8.2 Step Execution Mode

- GUI controls advancement line by line
- Each `step_next!()` executes one line and pauses
- `step_into!()` follows into CALL procedures
- `step_over!()` executes CALL fully and returns
- Current line, variables, and call stack sent to GUI after each step

### 8.3 Simulation Mode

- No real hardware commands are sent
- `SEND` and `SENDTCP` are logged but not dispatched
- TM values come from `TMSimulator` (configurable fake data)
- Useful for procedure development and validation without hardware

### 8.4 Hardware-in-the-Loop Mode

- Real hardware connections are active
- `SEND` dispatches commands to actual equipment
- TM values come from live telemetry stream
- Full safety checks are enforced (PRE_TEST_REQ)

---

## 9. Modularity, Scalability & Future Extensions

### 9.1 Adding New DSL Statements

The system is designed so new statements can be added without modifying existing code:

```julia
# In a separate file or plugin:
using ASTRA

# 1. Register the keyword
# 2. Register the handler
ACSExecutor.register_statement!(:DISPLAY, function(ctx, line)
    msg = join(line.tokens[2:end], " ")
    println("DISPLAY: $msg")
    push!(ctx.execution_log, LogEntry(now(), line.line_number,
          line.raw_text, "DISPLAY: $msg", :ok))
    # Send to GUI via WebSocket
    broadcast_event("display", Dict("message" => msg))
end)
```

The `register_statement!` function adds both the handler and the keyword to the parser's recognized list.

### 9.2 Plugin Architecture (Future)

```julia
# Plugin interface
abstract type ACSPlugin end

function on_load(plugin::ACSPlugin) end
function on_test_start(plugin::ACSPlugin, test_name::String) end
function on_statement(plugin::ACSPlugin, ctx, line) end
function on_test_end(plugin::ACSPlugin, result::ExecutionResult) end
function on_error(plugin::ACSPlugin, error::ASTRAError) end

# Example: Logging plugin
struct LoggingPlugin <: ACSPlugin
    log_file::String
end

function on_statement(p::LoggingPlugin, ctx, line)
    open(p.log_file, "a") do f
        println(f, "[$(now())] Line $(line.line_number): $(line.raw_text)")
    end
end
```

### 9.3 Scalability Considerations

| Aspect | Current Design | Future Extension |
|--------|---------------|-----------------|
| Multiple simultaneous tests | Single execution context | Thread-safe execution with `@spawn` per session |
| Large procedure files | Vector of lines | Chunked loading, lazy parsing |
| Many TM parameters | Dict lookup | Indexed TM database, memory-mapped telemetry |
| Multiple users | Single-user server | Multi-session WebSocket rooms with auth |
| Test scheduling | Manual run | Job queue with priority scheduling |
| Results storage | In-memory log | SQLite/PostgreSQL persistent storage |
| Hardware targets | Single target | Multiple concurrent hardware connections |

### 9.4 Future Extension Ideas

1. **Procedure Templates**: Pre-built procedure skeletons that users can fill in
2. **Drag-and-Drop Block Editor**: Visual block editor generating DSL code (for users who find even the DSL too complex)
3. **Test Suite Orchestrator**: Run multiple procedures in sequence with dependency graphs
4. **Telemetry Plotting**: Real-time plots of TM values during test execution
5. **Procedure Diff/Version Control**: Track changes to procedures over time
6. **Role-Based Access**: Operator vs Engineer vs Admin permissions
7. **Report Generation**: Auto-generate PDF test reports from execution logs
8. **Batch Execution**: Run same test with different parameter sets
9. **Conditional CALL**: `CALL proc1 IF condition` syntax
10. **Parameterized Procedures**: `CALL proc1 speed=100 mode="fast"`

---

## 10. Technology Stack Summary

| Layer | Technology | Why |
|-------|-----------|-----|
| Backend Language | Julia | Required per constraints; excellent scientific computing, metaprogramming |
| HTTP Server | HTTP.jl | Native Julia HTTP server, WebSocket support |
| JSON Handling | JSON3.jl | Fast, ergonomic JSON serialization |
| Frontend Editor | Monaco Editor | VS Code engine; syntax highlighting, autocomplete, error markers |
| Communication | WebSocket + REST | WebSocket for real-time events, REST for request-response |
| Data Storage | SQLite.jl | Lightweight, embedded database for test results |
| TM Simulation | Custom Julia module | Configurable simulated telemetry data |
| Testing | Julia Test stdlib | Unit and integration tests |

---

## 11. Key Design Decisions & Rationale

### Why Julia-style `END` instead of Python-style indentation?
Users are non-programmers. Indentation errors are invisible and confusing. Explicit `END` keywords make block boundaries visible and produce clear error messages ("Missing END for IF block at line 5").

### Why a custom parser instead of Julia macros?
The ChatGPT conversation explored Julia macros (`@test`, `@send`, etc.) but this requires users to write valid Julia syntax with `@` prefixes and `begin/end` blocks. A custom parser allows the DSL to look exactly like:
```
SEND START_RW
WAIT 5
IF TM1.volt > 5
```
instead of:
```julia
@send "START_RW"
@wait 5
@if_block TM1.volt > 5 begin ... end
```

The custom parser approach is more work but produces a much cleaner DSL for non-programmers.

### Why `eval(Meta.parse())` for inline Julia code?
This allows advanced users to write arbitrary Julia code inside procedures while keeping the DSL simple for beginners. `Meta.parse()` is also used during validation to check syntax without execution.

### Why WebSocket instead of polling?
Step execution and TM monitoring need real-time updates. WebSocket provides push-based events without the latency and overhead of polling.

### Why a single `PROCEDURE_REGISTRY` dictionary?
Simplicity. Each `.tst` file has one `TEST_NAME`. All files in the procedures directory are loaded into a flat dictionary. `CALL proc_name` is a simple dictionary lookup. No complex module system needed.

---

## 12. Example: Complete Procedure Execution Walkthrough

### Input File: `4rwconfig-3.tst`

```
TEST_NAME 4rw-config3
CALL 4rw-config1
CALL 4rw-config2

# Julia computation
adjusted = TM1.abc + 10
println("Adjusted value: ", adjusted)

IF adjusted > 50
    SEND START_RW
ELSE
    ALERT_MSG "Value too low"
END

FOR i IN 1 TO 3
    SEND RAMP_RW_$(i)
    WAIT 2
END

CHECK TM1.RW_SPEED <= 100

ON_FAIL
    ALERT_MSG "RW Configuration failed"
    ABORT_TEST
END
```

### Execution Trace:

```
=== Loading procedures ===
  Loaded: 4rw-config1 (from 4rwconfig-1.tst, 6 lines)
  Loaded: 4rw-config2 (from 4rwconfig-2.tst, 5 lines)
  Loaded: 4rw-config3 (from 4rwconfig-3.tst, 18 lines)

=== Syntax Validation: 4rw-config3 ===
  Line  1: CALL 4rw-config1           -> OK (procedure exists)
  Line  2: CALL 4rw-config2           -> OK (procedure exists)
  Line  4: adjusted = TM1.abc + 10    -> OK (valid Julia code)
  Line  5: println(...)                -> OK (valid Julia code)
  Line  7: IF adjusted > 50           -> OK (block opened, depth=1)
  Line  8:     SEND START_RW          -> OK
  Line  9: ELSE                       -> OK
  Line 10:     ALERT_MSG "..."        -> OK
  Line 11: END                        -> OK (block closed, depth=0)
  Line 13: FOR i IN 1 TO 3            -> OK (block opened, depth=1)
  Line 14:     SEND RAMP_RW_$(i)      -> OK
  Line 15:     WAIT 2                 -> OK
  Line 16: END                        -> OK (block closed, depth=0)
  Line 18: CHECK TM1.RW_SPEED <= 100  -> OK
  Line 20: ON_FAIL                    -> OK (block opened, depth=1)
  Line 21:     ALERT_MSG "..."        -> OK
  Line 22:     ABORT_TEST             -> OK
  Line 23: END                        -> OK (block closed, depth=0)
  Validation passed.

=== Running TEST: 4rw-config3 ===
  [Line 1] CALL 4rw-config1
    === Running TEST: 4rw-config1 ===
      PRE_TEST_REQ: TM1.xyz_sts == "on" -> PASS
      SEND: START_RW -> Acknowledged
      WAIT: 5 seconds
      CHECK: TM1.RW_STATUS == "READY" -> PASS
    === END TEST: 4rw-config1 ===
  [Line 2] CALL 4rw-config2
    === Running TEST: 4rw-config2 ===
      ... (similar)
    === END TEST: 4rw-config2 ===
  [Line 4] Julia: adjusted = TM1.abc + 10 => 52
  [Line 5] Julia: println("Adjusted value: 52")
  [Line 7] IF 52 > 50 => true
  [Line 8]   SEND: START_RW -> Acknowledged
  [Line 11] END IF
  [Line 13] FOR i = 1 TO 3
  [Line 14]   SEND: RAMP_RW_1 -> Acknowledged
  [Line 15]   WAIT: 2 seconds
  [Line 14]   SEND: RAMP_RW_2 -> Acknowledged
  [Line 15]   WAIT: 2 seconds
  [Line 14]   SEND: RAMP_RW_3 -> Acknowledged
  [Line 15]   WAIT: 2 seconds
  [Line 16] END FOR
  [Line 18] CHECK: TM1.RW_SPEED <= 100 -> PASS
=== END TEST: 4rw-config3 === (PASSED, 23.5s)
```

---

## 13. Security Considerations

| Concern | Mitigation |
|---------|-----------|
| Inline Julia code executing malicious code | Sandboxed `eval` with restricted module access; configurable whitelist of allowed functions |
| WebSocket hijacking | Authentication tokens on WebSocket handshake |
| Unauthorized test execution | Role-based access control on API endpoints |
| Hardware damage from bad commands | PRE_TEST_REQ enforcement; command validation against allowed command database |
| Data integrity | Immutable execution logs; append-only result storage |

---

## 14. Julia Dependencies (Project.toml)

```toml
[deps]
HTTP = "cd3eb016-35fb-5094-929b-558a96fad6f3"
JSON3 = "0f8b85d8-7281-11e9-16c2-39a750bddbf1"
SQLite = "0aa819cd-b072-5ff4-a722-6bc24af294d9"
Dates = "ade2ca70-3891-5945-98fb-dc099432e06a"
Sockets = "6462fe0b-24de-5631-8697-dd941f90decc"
Logging = "56ddb016-857b-54e1-b83d-db4d58db5568"
```

---

*Architecture Version: 1.0*
*Date: 2026-02-09*
*System: ASTRA - Automated Satellite Test & Reporting Application*
