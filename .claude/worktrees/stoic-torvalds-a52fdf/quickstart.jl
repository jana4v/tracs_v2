# ASTRA Quick Start Script
# Run this in Julia REPL to start the system

println("""
╔═══════════════════════════════════════════════════════════╗
║  ASTRA - Automated Satellite Test & Reporting Application                ║
║  Quick Start Script                                        ║
╚═══════════════════════════════════════════════════════════╝
""")

# Load the main module
include("src/ASTRA.jl")
using .ASTRA

println("✓ ASTRA module loaded")

# Initialize simulation data
println("\n→ Initializing simulation telemetry data...")
ASTRA.initialize_simulation_data()
println("✓ Simulation TM data initialized")

# Load example procedures
println("\n→ Loading example procedures...")
if isdir("procedures")
    ASTRA.load_directory("procedures")
    println("✓ Procedures loaded from procedures/")
else
    println("⚠ procedures/ directory not found")
end

# Display loaded procedures
procs = ASTRA.list_procedures()
if !isempty(procs)
    println("\nAvailable procedures:")
    for proc in procs
        println("  - $proc")
    end
end

# Start the web server
println("\n→ Starting web server on port 8080...")
@async ASTRA.start_server(8080)

println("""

✓ ASTRA is ready!

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Web GUI:  http://localhost:8080
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Example REPL Commands:
----------------------
# Run a test
julia> ASTRA.run_test("4rw-config1")

# Validate a procedure
julia> errors = ASTRA.validate_procedure("4rw-config1")

# Set TM values
julia> ASTRA.set_tm_value(1, :test_param, 100)

# Get TM value
julia> ASTRA.TM1.xyz_sts

# Display welcome message
julia> ASTRA.welcome()

Press Ctrl+C to stop the server when done.
""")
