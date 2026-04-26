"""
    TMInterface

Telemetry interface for ASTRA.
Provides access to telemetry data using TM<bank>.<mnemonic> syntax.
"""
module TMInterface

export TMBank, get_tm_value, set_tm_value, resolve_tm_ref
export get_all_mnemonics, get_all_tm_values, clear_tm_data, TMAccessor, initialize_simulation_data
export TM1, TM2, TM3, TM4, TM5, TM6, TM7, TM8, TM9, TM10

using Dates

# TM Bank: a named collection of telemetry parameters
mutable struct TMBank
    bank_id::Int
    parameters::Dict{Symbol, Any}     # mnemonic -> current value
    metadata::Dict{Symbol, Dict{Symbol, Any}}  # mnemonic -> {type, unit, range, etc.}
    last_updated::Dict{Symbol, DateTime}
end

# Registry of TM banks
const TM_BANKS = Dict{Int, TMBank}()

"""
    resolve_tm_ref(ref::String) -> Any

Resolve "TM1.xyz_sts" string to actual value.
"""
function resolve_tm_ref(ref::String)::Any
    m = match(r"^TM(\d+)\.(\w+)$", ref)
    if m === nothing
        error("Invalid TM reference: $ref (expected format: TM<number>.<mnemonic>)")
    end
    bank_id = parse(Int, m.captures[1])
    mnemonic = Symbol(m.captures[2])
    return get_tm_value(bank_id, mnemonic)
end

"""
    get_tm_value(bank_id::Int, mnemonic::Symbol) -> Any

Get a telemetry value from a bank.
"""
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

"""
    set_tm_value(bank_id::Int, mnemonic::Symbol, value::Any)

Set a telemetry value in a bank.
"""
function set_tm_value(bank_id::Int, mnemonic::Symbol, value::Any)
    if !haskey(TM_BANKS, bank_id)
        TM_BANKS[bank_id] = TMBank(bank_id, Dict(), Dict(), Dict())
    end
    TM_BANKS[bank_id].parameters[mnemonic] = value
    TM_BANKS[bank_id].last_updated[mnemonic] = now()
end

"""
    get_all_mnemonics() -> Vector{String}

Get all known mnemonics for autocomplete support.
"""
function get_all_mnemonics()::Vector{String}
    result = String[]
    for (bank_id, bank) in TM_BANKS
        for mnemonic in keys(bank.parameters)
            push!(result, "TM$bank_id.$mnemonic")
        end
    end
    return sort(result)
end

"""
    get_all_tm_values() -> Dict{String, Any}

Get all current TM values as a dictionary.
"""
function get_all_tm_values()::Dict{String, Any}
    result = Dict{String, Any}()
    for (bank_id, bank) in TM_BANKS
        for (mnemonic, value) in bank.parameters
            result["TM$bank_id.$mnemonic"] = value
        end
    end
    return result
end

"""
    clear_tm_data()

Clear all telemetry data.
"""
function clear_tm_data()
    empty!(TM_BANKS)
end

"""
    TMAccessor

Struct that allows property access syntax: TM1.xyz_sts
"""
struct TMAccessor
    bank_id::Int
end

function Base.getproperty(tm::TMAccessor, name::Symbol)
    bank_id = getfield(tm, :bank_id)
    return get_tm_value(bank_id, name)
end

function Base.setproperty!(tm::TMAccessor, name::Symbol, value)
    bank_id = getfield(tm, :bank_id)
    set_tm_value(bank_id, name, value)
end

# Pre-define TM1 through TM10 as global accessors
const TM1 = TMAccessor(1)
const TM2 = TMAccessor(2)
const TM3 = TMAccessor(3)
const TM4 = TMAccessor(4)
const TM5 = TMAccessor(5)
const TM6 = TMAccessor(6)
const TM7 = TMAccessor(7)
const TM8 = TMAccessor(8)
const TM9 = TMAccessor(9)
const TM10 = TMAccessor(10)

"""
    initialize_simulation_data()

Initialize some sample TM data for simulation/testing.
"""
function initialize_simulation_data()
    # TM Bank 1 - Power System
    set_tm_value(1, :xyz_sts, "on")
    set_tm_value(1, :abc, 42)
    set_tm_value(1, :voltage_bus, 28.3)
    set_tm_value(1, :VOLT, 28.3)
    set_tm_value(1, :STATUS, "OK")
    set_tm_value(1, :RW_STATUS, "READY")
    set_tm_value(1, :RW_SPEED, 0)
    set_tm_value(1, :RW_MODE, "NOMINAL")

    # TM Bank 2 - Reaction Wheels
    set_tm_value(2, :rw_speed, 1500)
    set_tm_value(2, :xyz_sts, "on")
    set_tm_value(2, :STATUS, "OK")
    set_tm_value(2, :RW_STATUS, "READY")

    @info "Initialized simulation TM data"
end

end # module TMInterface
