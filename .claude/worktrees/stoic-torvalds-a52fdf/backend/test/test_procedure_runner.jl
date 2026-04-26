#!/usr/bin/env julia
"""
Tests for ProcedureRunner module.
Exercises parallel execution, pause/resume, and abort functionality.

Usage:
    julia --project=backend backend/test/test_procedure_runner.jl
"""

using Pkg
Pkg.activate(joinpath(@__DIR__, ".."))

include(joinpath(@__DIR__, "..", "src", "ASTRA.jl"))
using .ASTRA
using Test
using Dates

# Access submodules directly
using .ASTRA: ACSParser, TMInterface, ProcedureRunner, ACSExecutor

# ===== Setup: Initialize TM simulation data and load test procedures =====

println("\n" * "="^60)
println("  ProcedureRunner Test Suite")
println("="^60 * "\n")

# TM values are now read from Redis TM_MAP (populated by external simulator)
# Ensure Redis is running with TM_MAP data before running tests

# Create simple test procedures (short WAITs for fast testing)
test_proc_a = """
TEST_NAME test-proc-a
SEND START_RW
WAIT 1
SEND STOP_RW
"""

test_proc_b = """
TEST_NAME test-proc-b
SEND CONFIGURE_RW
WAIT 1
SEND START_RW
"""

# A procedure with a long wait (for testing pause/abort during wait)
test_proc_slow = """
TEST_NAME test-proc-slow
SEND START_RW
WAIT 10
SEND STOP_RW
"""

# A procedure with a loop (for testing pause between iterations)
test_proc_loop = """
TEST_NAME test-proc-loop
FOR i IN 1 TO 5
    SEND RAMP_RW
    WAIT 1
END
"""

# Load all test procedures into the parser registry
ACSParser.load_from_string(test_proc_a, "test-proc-a.tst")
ACSParser.load_from_string(test_proc_b, "test-proc-b.tst")
ACSParser.load_from_string(test_proc_slow, "test-proc-slow.tst")
ACSParser.load_from_string(test_proc_loop, "test-proc-loop.tst")

println("Loaded test procedures: ", ACSParser.list_procedures())
println()

# ===== Test 1: Start a single procedure =====
@testset "Single procedure start" begin
    run = ProcedureRunner.start_procedure("test-proc-a")

    @test run.id != ""
    @test run.procedure_name == "test-proc-a"
    @test run.started_at <= now()

    # Wait for it to complete (max 5s)
    deadline = time() + 5.0
    while time() < deadline
        status = ProcedureRunner.get_run_status(run.id)
        if status["status"] in ("completed", "failed")
            break
        end
        sleep(0.1)
    end

    status = ProcedureRunner.get_run_status(run.id)
    @test status !== nothing
    @test status["run_id"] == run.id
    @test status["procedure"] == "test-proc-a"
    @test status["status"] == "completed"
    @test status["finished_at"] !== nothing
    println("  [PASS] Single procedure completed successfully")
end

# ===== Test 2: Run multiple procedures in parallel =====
@testset "Parallel procedure execution" begin
    run_a = ProcedureRunner.start_procedure("test-proc-a")
    run_b = ProcedureRunner.start_procedure("test-proc-b")

    @test run_a.id != run_b.id
    @test run_a.procedure_name == "test-proc-a"
    @test run_b.procedure_name == "test-proc-b"

    # Both should appear in the run list
    runs = ProcedureRunner.list_runs()
    run_ids = [r["run_id"] for r in runs]
    @test run_a.id in run_ids
    @test run_b.id in run_ids

    # Wait for both to complete
    deadline = time() + 10.0
    while time() < deadline
        sa = ProcedureRunner.get_run_status(run_a.id)
        sb = ProcedureRunner.get_run_status(run_b.id)
        if sa["status"] in ("completed", "failed") && sb["status"] in ("completed", "failed")
            break
        end
        sleep(0.1)
    end

    sa = ProcedureRunner.get_run_status(run_a.id)
    sb = ProcedureRunner.get_run_status(run_b.id)
    @test sa["status"] == "completed"
    @test sb["status"] == "completed"
    println("  [PASS] Two procedures ran in parallel and both completed")
end

# ===== Test 3: Abort a running procedure =====
@testset "Abort procedure" begin
    run = ProcedureRunner.start_procedure("test-proc-slow")

    # Give it a moment to start running
    sleep(0.3)

    status = ProcedureRunner.get_run_status(run.id)
    @test status["status"] == "running"

    # Abort it
    success = ProcedureRunner.abort_procedure(run.id)
    @test success == true

    # Wait for the abort to take effect
    deadline = time() + 3.0
    while time() < deadline
        status = ProcedureRunner.get_run_status(run.id)
        if status["status"] == "aborted"
            break
        end
        sleep(0.1)
    end

    status = ProcedureRunner.get_run_status(run.id)
    @test status["status"] == "aborted"
    @test status["finished_at"] !== nothing
    println("  [PASS] Procedure was aborted successfully")
end

# ===== Test 4: Pause and resume a running procedure =====
@testset "Pause and resume procedure" begin
    run = ProcedureRunner.start_procedure("test-proc-loop")

    # Give it a moment to start running
    sleep(0.3)

    status = ProcedureRunner.get_run_status(run.id)
    @test status["status"] == "running"

    # Pause it
    success = ProcedureRunner.pause_procedure(run.id)
    @test success == true

    # Wait for pause to take effect
    deadline = time() + 3.0
    while time() < deadline
        status = ProcedureRunner.get_run_status(run.id)
        if status["status"] == "paused"
            break
        end
        sleep(0.1)
    end

    status = ProcedureRunner.get_run_status(run.id)
    @test status["status"] == "paused"

    # Record the line number while paused
    paused_line = status["current_line"]

    # Wait a bit — line should NOT advance while paused
    sleep(1.0)
    status2 = ProcedureRunner.get_run_status(run.id)
    @test status2["status"] == "paused"
    @test status2["current_line"] == paused_line

    # Resume
    success = ProcedureRunner.resume_procedure(run.id)
    @test success == true

    # Wait for it to complete
    deadline = time() + 15.0
    while time() < deadline
        status = ProcedureRunner.get_run_status(run.id)
        if status["status"] in ("completed", "failed")
            break
        end
        sleep(0.1)
    end

    status = ProcedureRunner.get_run_status(run.id)
    @test status["status"] == "completed"
    println("  [PASS] Procedure was paused, verified frozen, resumed, and completed")
end

# ===== Test 5: Abort a paused procedure =====
@testset "Abort while paused" begin
    run = ProcedureRunner.start_procedure("test-proc-slow")

    sleep(0.3)

    # Pause first
    ProcedureRunner.pause_procedure(run.id)
    deadline = time() + 3.0
    while time() < deadline
        status = ProcedureRunner.get_run_status(run.id)
        if status["status"] == "paused"
            break
        end
        sleep(0.1)
    end

    status = ProcedureRunner.get_run_status(run.id)
    @test status["status"] == "paused"

    # Now abort while paused
    success = ProcedureRunner.abort_procedure(run.id)
    @test success == true

    deadline = time() + 3.0
    while time() < deadline
        status = ProcedureRunner.get_run_status(run.id)
        if status["status"] == "aborted"
            break
        end
        sleep(0.1)
    end

    status = ProcedureRunner.get_run_status(run.id)
    @test status["status"] == "aborted"
    println("  [PASS] Procedure aborted while paused")
end

# ===== Test 6: Start nonexistent procedure =====
@testset "Error handling" begin
    @test_throws ErrorException ProcedureRunner.start_procedure("nonexistent-proc")

    # Pause/resume/abort on nonexistent run_id should return false
    @test ProcedureRunner.pause_procedure("fake-id") == false
    @test ProcedureRunner.resume_procedure("fake-id") == false
    @test ProcedureRunner.abort_procedure("fake-id") == false

    # get_run_status for nonexistent should return nothing
    @test ProcedureRunner.get_run_status("fake-id") === nothing
    println("  [PASS] Error handling works correctly")
end

# ===== Test 7: List runs =====
@testset "List runs" begin
    runs = ProcedureRunner.list_runs()
    @test length(runs) >= 5  # We've started at least 5 runs above
    @test all(r -> haskey(r, "run_id"), runs)
    @test all(r -> haskey(r, "procedure"), runs)
    @test all(r -> haskey(r, "status"), runs)
    println("  [PASS] List runs returns all tracked runs")
end

# ===== Test 8: Cleanup old runs =====
@testset "Cleanup runs" begin
    before_count = length(ProcedureRunner.list_runs())
    # Cleanup with 0 seconds threshold — should remove all completed/failed/aborted
    ProcedureRunner.cleanup_runs(0)
    after_count = length(ProcedureRunner.list_runs())
    @test after_count < before_count
    println("  [PASS] Cleanup removed finished runs")
end

println("\n" * "="^60)
println("  All ProcedureRunner tests completed!")
println("="^60 * "\n")
