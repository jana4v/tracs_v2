"""
    ASTRA - Automated Satellite Test & Reporting Application

Complete satellite test automation system with beginner-friendly DSL.
Backend powered by Genie.jl with MongoDB persistent storage.
TM values read from Redis TM_MAP hash (populated by external simulator).
"""
module ASTRA

# Re-export key functions
export load_file, load_directory, load_from_string
export validate_procedure
export run_test
export create_step_session, step_next!, step_reset!
export resolve_tm_ref, get_all_tm_values
export send_command, set_mode!
export start_server, stop_server
export start_procedure, pause_procedure, resume_procedure, abort_procedure
export get_run_status, list_runs
export get_test_phase, set_test_phase, get_satellite_config, set_satellite_config

# Include all modules (order matters: RedisStore before TMInterface, MongoStore before ACSExecutor)
include("parser/ACSParser.jl")
include("validator/ACSValidator.jl")
include("db/RedisStore.jl")
include("tm/TMInterface.jl")
include("commands/CommandDispatch.jl")
include("db/MongoStore.jl")
include("executor/ACSExecutor.jl")
include("executor/ASTRAStepEngine.jl")
include("logging/DataLogger.jl")
include("runner/ProcedureRunner.jl")
include("runner/BackgroundScheduler.jl")
include("server/GenieApp.jl")

# Use modules
using .ACSParser
using .ACSValidator
using .RedisStore
using .TMInterface
using .CommandDispatch
using .ACSExecutor
using .ASTRAStepEngine
using .DataLogger
using .MongoStore
using .ProcedureRunner
using .BackgroundScheduler
using .GenieApp

# Make TM accessors globally available
import .TMInterface: TM, TM1, TM2, TM3, TM4
export TM, TM1, TM2, TM3, TM4

"""
    welcome()

Print welcome message and system info.
"""
function welcome()
    println("""
    ╔═══════════════════════════════════════════════════════════╗
    ║  ASTRA - Automated Satellite Test & Reporting Application ║
    ║  Version 2.1.0                                            ║
    ║                                                           ║
    ║  Julia + Genie.jl + MongoDB + Redis + Nuxt 4 Frontend     ║
    ╚═══════════════════════════════════════════════════════════╝

    Quick Start:
    ------------
    1. Ensure Redis is running with TM_MAP populated by simulator

    2. Start the API server:
       julia> ASTRA.start_server(8080)

    3. Access the Nuxt frontend at http://localhost:3000
       Or use the API directly at http://localhost:8080/api/v1

    4. Load a test procedure via API or Nuxt editor

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
