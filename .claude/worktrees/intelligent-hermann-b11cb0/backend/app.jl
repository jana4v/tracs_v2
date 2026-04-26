#!/usr/bin/env julia

"""
ASTRA Backend Entry Point
Starts the Genie.jl API server with all ASTRA modules loaded.

Usage:
    julia --project=backend backend/app.jl
    julia --project=backend backend/app.jl 8080
"""

# Activate the project environment
using Pkg
Pkg.activate(@__DIR__)

# Load ASTRA module
include(joinpath(@__DIR__, "src", "ASTRA.jl"))
using .ASTRA

# Parse port from command line
port = length(ARGS) > 0 ? parse(Int, ARGS[1]) : 8080

# Load procedures from files (as fallback / initial import)
procedures_dir = joinpath(@__DIR__, "..", "procedures")
if isdir(procedures_dir)
    @info "Loading procedures from $procedures_dir"
    try
        ASTRA.load_directory(procedures_dir)
    catch e
        @warn "Failed to load procedures directory: $e"
    end
end

# Print welcome
ASTRA.welcome()

# Start the server
ASTRA.start_server(port)

# Keep alive
@info "Press Ctrl+C to stop the server"
try
    while true
        sleep(1)
    end
catch e
    if e isa InterruptException
        @info "Shutting down..."
        ASTRA.stop_server()
    else
        rethrow(e)
    end
end
