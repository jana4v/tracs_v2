"""
    CommandDispatch

Routes SEND/SENDTCP commands to hardware or simulator.
"""
module CommandDispatch

export send_command, send_tcp_command, CommandResult
export set_mode!, get_mode, CommandTarget, HardwareTarget, SimulatorTarget

using Dates
using Sockets

struct CommandResult
    command::String
    status::Symbol      # :sent, :acknowledged, :failed, :timeout
    response::String
    timestamp::DateTime
end

# Abstraction over hardware vs simulation
abstract type CommandTarget end

struct HardwareTarget <: CommandTarget
    connection::Union{Nothing, Any}
end

mutable struct SimulatorTarget <: CommandTarget
    sim_state::Dict{String, Any}
    command_log::Vector{Tuple{DateTime, String}}
end

SimulatorTarget() = SimulatorTarget(Dict{String, Any}(), Tuple{DateTime, String}[])

# Active target (switchable between hardware and simulation)
const ACTIVE_TARGET = Ref{CommandTarget}(SimulatorTarget())
const CURRENT_MODE = Ref{Symbol}(:simulation)

"""
    dispatch_to_simulator(target::SimulatorTarget, command::String, args...) -> CommandResult

Simulate command execution.
"""
function dispatch_to_simulator(target::SimulatorTarget, command::String, args...)::CommandResult
    timestamp = now()
    full_cmd = join([command, args...], " ")

    # Log the command
    push!(target.command_log, (timestamp, full_cmd))

    # Simulate some commands
    if command == "START_RW"
        target.sim_state["rw_status"] = "RUNNING"
        return CommandResult(full_cmd, :acknowledged, "RW started", timestamp)
    elseif command == "STOP_RW"
        target.sim_state["rw_status"] = "STOPPED"
        return CommandResult(full_cmd, :acknowledged, "RW stopped", timestamp)
    elseif startswith(command, "CONFIGURE")
        return CommandResult(full_cmd, :acknowledged, "Configuration accepted", timestamp)
    elseif startswith(command, "RAMP")
        return CommandResult(full_cmd, :acknowledged, "Ramp command accepted", timestamp)
    else
        return CommandResult(full_cmd, :sent, "Command sent (simulated)", timestamp)
    end
end

"""
    dispatch_to_hardware(target::HardwareTarget, command::String, args...) -> CommandResult

Send command to real hardware.
"""
function dispatch_to_hardware(target::HardwareTarget, command::String, args...)::CommandResult
    timestamp = now()
    full_cmd = join([command, args...], " ")

    # TODO: Implement actual hardware communication
    # This would use the hardware connection to send the command
    # For now, we'll return a placeholder

    @warn "Hardware mode not yet implemented, simulating instead"
    return CommandResult(full_cmd, :sent, "Hardware command would be sent here", timestamp)
end

"""
    send_command(command::String, args...) -> CommandResult

Send a command to the active target.
"""
function send_command(command::String, args...)::CommandResult
    target = ACTIVE_TARGET[]

    if target isa SimulatorTarget
        return dispatch_to_simulator(target, command, args...)
    elseif target isa HardwareTarget
        return dispatch_to_hardware(target, command, args...)
    else
        error("Unknown target type: $(typeof(target))")
    end
end

"""
    send_tcp_command(host::String, port::Int, data::String) -> CommandResult

Send a TCP command.
"""
function send_tcp_command(host::String, port::Int, data::String)::CommandResult
    timestamp = now()

    if CURRENT_MODE[] == :simulation
        @info "SENDTCP (simulated): $host:$port <- $data"
        return CommandResult("SENDTCP $host:$port", :sent, "Simulated TCP send", timestamp)
    end

    try
        # Actual TCP connection
        sock = connect(host, port)
        write(sock, data)
        response = String(readavailable(sock))
        close(sock)

        return CommandResult("SENDTCP $host:$port", :acknowledged, response, timestamp)
    catch e
        return CommandResult("SENDTCP $host:$port", :failed, "Error: $(sprint(showerror, e))", timestamp)
    end
end

"""
    set_mode!(mode::Symbol)

Switch between :simulation and :hardware modes.
"""
function set_mode!(mode::Symbol)
    if mode == :simulation
        ACTIVE_TARGET[] = SimulatorTarget()
        CURRENT_MODE[] = :simulation
        @info "Switched to SIMULATION mode"
    elseif mode == :hardware
        ACTIVE_TARGET[] = HardwareTarget(nothing)
        CURRENT_MODE[] = :hardware
        @warn "Hardware mode selected but not fully implemented"
    else
        error("Invalid mode: $mode (must be :simulation or :hardware)")
    end
end

"""
    get_mode() -> Symbol

Get the current execution mode.
"""
get_mode() = CURRENT_MODE[]

"""
    get_command_log() -> Vector{Tuple{DateTime, String}}

Get the log of all commands (simulation mode only).
"""
function get_command_log()::Vector{Tuple{DateTime, String}}
    target = ACTIVE_TARGET[]
    if target isa SimulatorTarget
        return target.command_log
    else
        return Tuple{DateTime, String}[]
    end
end

end # module CommandDispatch
