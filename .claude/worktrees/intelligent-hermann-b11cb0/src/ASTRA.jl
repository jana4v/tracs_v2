"""
    ASTRA - Automated Satellite Test & Reporting Application

Complete satellite test automation system with beginner-friendly DSL.
"""
module ASTRA

# Re-export key functions
export load_file, load_directory, load_from_string
export validate_procedure
export run_test
export create_step_session, step_next!, step_reset!
export get_tm_value, set_tm_value, initialize_simulation_data
export send_command, set_mode!
export start_server, stop_server

# Include all modules
include("parser/ACSParser.jl")
include("validator/ACSValidator.jl")
include("tm/TMInterface.jl")
include("commands/CommandDispatch.jl")
include("executor/ACSExecutor.jl")
include("executor/ASTRAStepEngine.jl")
include("logging/DataLogger.jl")
include("server/ASTRAServer.jl")

# Use modules
using .ACSParser
using .ACSValidator
using .TMInterface
using .CommandDispatch
using .ACSExecutor
using .ASTRAStepEngine
using .DataLogger
using .ASTRAServer

# Make TM accessors globally available
import .TMInterface: TM1, TM2, TM3, TM4, TM5, TM6, TM7, TM8, TM9, TM10
export TM1, TM2, TM3, TM4, TM5, TM6, TM7, TM8, TM9, TM10

"""
    welcome()

Print welcome message and system info.
"""
function welcome()
    println("""
    ╔═══════════════════════════════════════════════════════════╗
    ║  ASTRA - Automated Satellite Test & Reporting Application                ║
    ║  Version 1.0.0                                             ║
    ║                                                            ║
    ║  Satellite test automation with beginner-friendly DSL     ║
    ╚═══════════════════════════════════════════════════════════╝

    Quick Start:
    ------------
    1. Initialize simulation data:
       julia> ASTRA.initialize_simulation_data()

    2. Load a test procedure:
       julia> ASTRA.load_file("procedures/example.tst")

    3. Validate it:
       julia> ASTRA.validate_procedure("example-test")

    4. Run it:
       julia> ASTRA.run_test("example-test")

    5. Start the web GUI:
       julia> ASTRA.start_server(8080)
       Then open http://localhost:8080 in your browser

    For help:
    - Read ASTRA_SYSTEM_ARCHITECTURE.md for full documentation
    - Check the examples in the procedures/ directory
    """)
end

"""
    __init__()

Module initialization.
"""
function __init__()
    @info "ASTRA system loaded successfully"
    @info "Run ASTRA.welcome() for quick start guide"
end

end # module ASTRA
