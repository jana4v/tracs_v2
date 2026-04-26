"""
    DataLogger

Logging and data storage for ASTRA test results and telemetry.
"""
module DataLogger

export log_test_result, get_test_results, clear_logs
export log_tm_snapshot, get_tm_history

using Dates
using ..ACSExecutor
using JSON3

# In-memory storage (in production, this would use SQLite)
const TEST_RESULTS = Vector{Any}[]
const TM_HISTORY = Vector{Any}[]

"""
    log_test_result(result::ExecutionResult)

Store a test execution result.
"""
function log_test_result(result::ExecutionResult)
    result_dict = Dict(
        "test_name" => result.test_name,
        "status" => string(result.status),
        "duration_seconds" => result.duration_seconds,
        "timestamp" => now(),
        "log_entries" => length(result.log),
        "error" => if result.error !== nothing
            Dict(
                "line_number" => result.error.line_number,
                "line_text" => result.error.line_text,
                "message" => result.error.message
            )
        else
            nothing
        end
    )

    push!(TEST_RESULTS, result_dict)

    @info "Logged test result for $(result.test_name): $(result.status)"
end

"""
    get_test_results(; limit::Int=100) -> Vector

Get recent test results.
"""
function get_test_results(; limit::Int=100)::Vector
    n = min(limit, length(TEST_RESULTS))
    return TEST_RESULTS[end-n+1:end]
end

"""
    clear_logs()

Clear all logged data.
"""
function clear_logs()
    empty!(TEST_RESULTS)
    empty!(TM_HISTORY)
    @info "Cleared all logs"
end

"""
    log_tm_snapshot(tm_data::Dict)

Log a telemetry snapshot.
"""
function log_tm_snapshot(tm_data::Dict)
    snapshot = Dict(
        "timestamp" => now(),
        "data" => tm_data
    )

    push!(TM_HISTORY, snapshot)

    # Keep only last 1000 snapshots
    if length(TM_HISTORY) > 1000
        popfirst!(TM_HISTORY)
    end
end

"""
    get_tm_history(; limit::Int=100) -> Vector

Get recent TM snapshots.
"""
function get_tm_history(; limit::Int=100)::Vector
    n = min(limit, length(TM_HISTORY))
    return TM_HISTORY[end-n+1:end]
end

"""
    export_results_to_file(filename::String)

Export test results to a JSON file.
"""
function export_results_to_file(filename::String)
    open(filename, "w") do io
        JSON3.write(io, TEST_RESULTS)
    end

    @info "Exported $(length(TEST_RESULTS)) test results to $filename"
end

end # module DataLogger
