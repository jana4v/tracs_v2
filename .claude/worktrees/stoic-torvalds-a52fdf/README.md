# ASTRA - Automated Satellite Test & Reporting Application

A complete satellite test automation system with a beginner-friendly Domain-Specific Language (DSL) for writing test procedures.

## Features

- **Beginner-Friendly DSL**: Write test procedures without programming knowledge
- **Julia-Style Syntax**: Explicit `END` keywords (no Python-style indentation errors)
- **Web-Based GUI**: Monaco Editor (VS Code engine) with syntax highlighting
- **Step-by-Step Debugging**: Execute procedures line-by-line with GUI highlighting
- **Cross-Procedure Calls**: Modular procedures with `CALL` statements
- **Inline Julia Code**: Advanced users can embed Julia computations
- **Telemetry Interface**: Easy access to TM data with `TM1.xyz_sts` syntax
- **Simulation Mode**: Test procedures without hardware
- **Real-Time Monitoring**: Live TM values, alerts, and execution logs

## Quick Start

### 1. Install Julia

Download and install Julia 1.9 or later from [julialang.org](https://julialang.org/downloads/)

```bash
cd /e/Code/Mainframe/MainframeAutomation/backend && julia --project=. -e "using Pkg; Pkg.resolve(); Pkg.instantiate()" 2>&1
```
### 2. Activate the Project

```bash
cd MF_AUTOMATION_JULIA_2026
julia --project=.
```

### 3. Install Dependencies

```julia
using Pkg
Pkg.instantiate()
```

### 4. Load ASTRA

```julia
include("src/ASTRA.jl")
using .ASTRA

# Display welcome message
ASTRA.welcome()
```

### 5. Initialize Simulation Data

```julia
ASTRA.initialize_simulation_data()
```

### 6. Start the Web Server

```julia
ASTRA.start_server(8080)
```

Then open http://localhost:8080 in your browser.

## Testing from REPL

You can also run tests directly from the Julia REPL:

```julia
# Load a procedure
ASTRA.load_file("procedures/4rwconfig-1.tst")

# Validate syntax
errors = ASTRA.validate_procedure("4rw-config1")

# Run the test
result = ASTRA.run_test("4rw-config1")

# Check execution status
println("Test: $(result.test_name)")
println("Status: $(result.status)")
println("Duration: $(result.duration_seconds) seconds")
```

## DSL Syntax Reference

### Basic Statements

```
TEST_NAME my-test               # Procedure name (required, first line)
PRE_TEST_REQ TM1.status == "OK" # Pre-condition check
SEND START_RW                   # Send telecommand
SENDTCP 127.0.0.1 5000 "data"  # Send TCP command
WAIT 5                          # Wait 5 seconds
CHECK TM1.volt > 10             # Verify telemetry
CHECK TM1.volt > 10 WITHIN 30   # Check with timeout
EXPECTED TM1.mode == "NOMINAL"  # Assert expected state
ALERT_MSG "Warning message"     # Display alert
ABORT_TEST                      # Stop test immediately
CALL other-procedure            # Execute another procedure
BREAK                           # Exit current loop
```

### Control Flow

```
IF TM1.voltage > 10
    SEND START
ELSE
    ALERT_MSG "Voltage too low"
END

FOR i IN 1 TO 10
    SEND STEP_$(i)
    WAIT 1
END

WHILE TM1.speed < 100
    WAIT 1
END
```

### Error Handlers

```
WAIT UNTIL TM1.ready == true TIMEOUT 30

ON_TIMEOUT
    ALERT_MSG "Timed out waiting for ready"
    ABORT_TEST
END
```

### Inline Julia Code

```
# Julia computation
adjusted = TM1.abc + 10
println("Adjusted value: ", adjusted)

IF adjusted > 50
    SEND START
END
```

## GUI Usage

### Monaco Editor
- **New**: Create a new test procedure
- **Open**: Load a `.tst` file from disk
- **Save**: Save current procedure to disk
- **Validate**: Check syntax without running
- **Run**: Execute the full procedure
- **Step Mode**: Start line-by-line execution
- **Next**: Execute next line (step mode)
- **Reset**: Reset to beginning (step mode)

### Panels
- **Editor (left)**: Monaco Editor with syntax highlighting and error markers
- **Console (top-right)**: Execution log with timestamped output
- **Variables (top-right)**: Current variable values during step execution
- **TM Monitor (bottom-right)**: Live telemetry values
- **Alerts (bottom-right)**: Warning and error messages
- **Problems (bottom)**: Syntax validation errors with line numbers

## Project Structure

```
MF_AUTOMATION_JULIA_2026/
├── src/
│   ├── ASTRA.jl                    # Main module
│   ├── parser/ACSParser.jl        # DSL parser
│   ├── validator/ACSValidator.jl  # Syntax validation
│   ├── executor/ACSExecutor.jl    # Execution engine
│   ├── executor/ASTRAStepEngine.jl  # Step debugger
│   ├── tm/TMInterface.jl          # Telemetry interface
│   ├── commands/CommandDispatch.jl # Command routing
│   ├── server/ASTRAServer.jl        # HTTP/WebSocket server
│   └── logging/DataLogger.jl      # Results logging
├── frontend/
│   ├── index.html                 # Web GUI
│   ├── css/styles.css            # Styling
│   └── js/app.js                 # Monaco Editor integration
├── procedures/
│   ├── 4rwconfig-1.tst           # Example procedure
│   └── 4rwconfig-3.tst           # Example with CALL, loops, Julia
├── Project.toml                   # Julia package manifest
├── ASTRA_SYSTEM_ARCHITECTURE.md   # Complete architecture document
└── README.md                      # This file
```

## Example: Complete Test Procedure

File: `procedures/example.tst`

```
TEST_NAME rw-startup-sequence
PRE_TEST_REQ TM1.power_status == "on" AND TM1.voltage_bus > 24.0

# Initialize reaction wheels
SEND CONFIGURE_RW_MODE NOMINAL
WAIT 2

# Check configuration
CHECK TM1.rw_mode == "CONFIGURED" WITHIN 10

ON_TIMEOUT
    ALERT_MSG "RW configuration timeout"
    ABORT_TEST
END

# Ramp up each wheel
FOR wheel_id IN 1 TO 4
    SEND START_RW_$(wheel_id)
    WAIT 3
    CHECK TM1.rw_$(wheel_id)_speed > 0 WITHIN 5
END

# Verify speeds
CHECK TM1.rw_1_speed <= 1500 AND TM1.rw_2_speed <= 1500

# Julia computation for target speed
target_speed = Int(TM1.target_rpm * 0.9)
println("Computed target: ", target_speed)

IF TM1.rw_1_speed >= target_speed
    ALERT_MSG "RW startup successful"
ELSE
    ALERT_MSG "RW speed below target"
END
```

## Switching Between Simulation and Hardware

By default, ASTRA runs in simulation mode. To switch to hardware mode:

```julia
# Simulation mode (default)
ASTRA.set_mode!(:simulation)

# Hardware mode (requires hardware connection setup)
ASTRA.set_mode!(:hardware)
```

## Adding New TM Parameters

```julia
# Add telemetry values programmatically
ASTRA.set_tm_value(1, :new_parameter, "value")
ASTRA.set_tm_value(2, :sensor_reading, 42.5)

# Access in procedures: TM1.new_parameter
```

## Extending the DSL

Add new DSL statements by registering handlers:

```julia
using .ASTRA.ACSExecutor

# Define handler function
function handle_my_command!(ctx, line)
    arg = join(line.tokens[2:end], " ")
    println("MY_COMMAND: ", arg)
    # ... custom logic
end

# Register it
ACSExecutor.register_statement!(:MY_COMMAND, handle_my_command!)
```

Now `MY_COMMAND my_args` is a valid statement in `.tst` files.

## API Endpoints

The HTTP server provides REST API endpoints:

- `POST /api/load` - Load procedure from content
- `POST /api/validate` - Validate procedure syntax
- `POST /api/run` - Run full test
- `POST /api/step/start` - Start step session
- `POST /api/step/next` - Execute next step
- `POST /api/step/reset` - Reset step session
- `GET /api/procedures` - List loaded procedures
- `GET /api/tm` - Get current TM values
- `GET /api/tm/mnemonics` - Get TM autocomplete list
- `GET /api/results` - Get test results

## Troubleshooting

### "Procedure not found" error
Make sure you've loaded the file first:
```julia
ASTRA.load_file("procedures/my-test.tst")
```

### Syntax errors
Use the Validate button in the GUI or:
```julia
errors = ASTRA.validate_procedure("test-name")
for err in errors
    println("Line $(err.line_number): $(err.message)")
end
```

### Port already in use
Change the port number:
```julia
ASTRA.start_server(8081)
```

## Architecture

See [ASTRA_SYSTEM_ARCHITECTURE.md](ASTRA_SYSTEM_ARCHITECTURE.md) for the complete system architecture including:
- Component diagrams
- Data flow
- Module responsibilities
- Extension points
- Scalability considerations

## License

This system is a prototype for satellite test automation. Modify as needed for your specific use case.

## Support

For issues or questions:
1. Check the architecture document
2. Review example procedures in `procedures/`
3. Run `ASTRA.welcome()` for quick reference
4. Validate procedures before running to catch errors early

---

**Built with Julia | Monaco Editor | WebSocket | Modern Web Technologies**
