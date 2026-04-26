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

    # Simulate commands and update TM values accordingly
    response = simulate_command_effects!(command)

    return CommandResult(full_cmd, :acknowledged, response, timestamp)
end

"""
    simulate_command_effects!(command) -> String

Acknowledge known commands. TM value changes are handled by the
external simulator app which writes to the Redis TM_MAP hash.
"""
function simulate_command_effects!(command::String)::String
    # --- Reaction Wheels ---
    if command == "START_RW" || command == "RW_INIT"
        return "RW initialized"
    elseif command == "STOP_RW"
        return "RW stopped"
    elseif startswith(command, "RW_SET_SPEED")
        return "RW speed set"

    # --- Solar Array ---
    elseif command == "SA_UNLOCK"
        return "SA unlocked"
    elseif command == "SA_DEPLOY_PRI"
        return "Primary SA deployed"
    elseif command == "SA_DEPLOY_SEC"
        return "Secondary SA deployed"
    elseif command == "SA_SAFE_MODE"
        return "SA safe mode"

    # --- Star Tracker ---
    elseif command == "STR_INIT"
        return "STR initialized"
    elseif command == "STR_START_CAL"
        return "STR calibration started"

    # --- Thrusters ---
    elseif command == "THR_PREHEAT"
        return "Thrusters preheated"
    elseif startswith(command, "THR_FIRE_")
        thr_num = split(replace(command, "THR_FIRE_" => ""))[1]
        return "Thruster $thr_num fired"
    elseif command == "THR_SAFE"
        return "Thrusters safed"

    # --- Thermal ---
    elseif command == "HTR_ENABLE_ZONE_1"
        return "Zone 1 heater enabled"
    elseif command == "HTR_ENABLE_ZONE_2"
        return "Zone 2 heater enabled"
    elseif command == "HTR_DISABLE_ZONE_1"
        return "Zone 1 heater disabled"
    elseif command == "HTR_DISABLE_ZONE_2"
        return "Zone 2 heater disabled"

    # --- Communications ---
    elseif startswith(command, "COMM_SET_FREQ")
        return "Freq set"
    elseif startswith(command, "COMM_SET_POWER")
        return "TX power set"
    elseif command == "COMM_TX_ENABLE"
        return "TX enabled"
    elseif command == "COMM_SEND_BEACON"
        return "Beacon sent"

    # --- AOCS ---
    elseif startswith(command, "AOCS_SET_MODE")
        mode = try split(command)[2] catch; "SAFE" end
        return "AOCS mode set to $mode"
    elseif startswith(command, "AOCS_ROTATE")
        return "Rotation command accepted"

    # --- Power ---
    elseif startswith(command, "PWR_ENABLE_LOAD")
        return "Load enabled"

    # --- Payload Data Handling ---
    elseif command == "PDH_INIT"
        return "PDH initialized"
    elseif command == "PDH_START_ACQ"
        return "PDH acquisition started"
    elseif command == "PDH_STOP_ACQ"
        return "PDH acquisition stopped"
    elseif command == "PDH_PLAYBACK"
        return "PDH playback complete"

    # --- GPS ---
    elseif command == "GPS_COLD_START"
        return "GPS cold start initiated"

    # --- Magnetometer ---
    elseif command == "MAG_START_CAL"
        return "MAG calibration started"

    # --- Ramp / generic ---
    elseif startswith(command, "RAMP")
        return "Ramp command accepted"
    elseif startswith(command, "CONFIGURE")
        return "Configuration accepted"
    else
        return "Command acknowledged"
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
