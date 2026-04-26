"""
    TMInterface

Telemetry interface for ASTRA.
Reads telemetry values from Redis TM_MAP hash populated by an external simulator.
Supports both TM.SUBSYSTEM.mnemonic (3-part) and TM1.mnemonic (2-part legacy) syntax.
"""
module TMInterface

export resolve_tm_ref, get_all_tm_values, get_all_mnemonics, clear_tm_data
export TMAccessor, TMSubsystemAccessor
export TM, TM1, TM2, TM3, TM4

using ..RedisStore

"""
    TMSubsystemAccessor

Intermediate accessor for 3-part syntax: TM.AOC returns this,
then .mnemonic does the Redis TM_MAP lookup.
"""
struct TMSubsystemAccessor
    subsystem::String
end

function Base.getproperty(sa::TMSubsystemAccessor, name::Symbol)
    mnemonic = String(name)
    val = get_tm_map_value(mnemonic)
    if val === nothing
        error("Mnemonic $mnemonic not found in TM_MAP")
    end
    parsed = tryparse(Float64, val)
    return parsed !== nothing ? parsed : val
end

"""
    TMAccessor

Accessor for TM references.
- TM (bank_id=nothing): 3-part syntax, TM.SUBSYSTEM returns TMSubsystemAccessor
- TM1-TM4 (bank_id=Int): 2-part legacy syntax, TM1.mnemonic does direct Redis lookup
"""
struct TMAccessor
    bank_id::Union{Int, Nothing}
end

function Base.getproperty(tm::TMAccessor, name::Symbol)
    bank_id = getfield(tm, :bank_id)
    if bank_id === nothing
        # TM.SUBSYSTEM → return subsystem accessor
        return TMSubsystemAccessor(String(name))
    else
        # TM1.mnemonic → direct Redis lookup (legacy 2-part)
        mnemonic = String(name)
        val = get_tm_map_value(mnemonic)
        if val === nothing
            error("Mnemonic $mnemonic not found in TM_MAP")
        end
        parsed = tryparse(Float64, val)
        return parsed !== nothing ? parsed : val
    end
end

# Pre-define accessors
const TM  = TMAccessor(nothing)
const TM1 = TMAccessor(1)
const TM2 = TMAccessor(2)
const TM3 = TMAccessor(3)
const TM4 = TMAccessor(4)

"""
    resolve_tm_ref(ref::String) -> Any

Resolve a TM reference string to its current value from Redis TM_MAP.
Supports:
  - 3-part: TM.SUBSYSTEM.mnemonic or TM1.SUBSYSTEM.mnemonic
  - 2-part: TM1.mnemonic (legacy)
"""
function resolve_tm_ref(ref::String)::Any
    # 3-part: TM.SUBSYSTEM.mnemonic or TM1.SUBSYSTEM.mnemonic
    m3 = match(r"^TM\d*\.(\w+)\.([\w+\-.]+)$", ref)
    if m3 !== nothing
        mnemonic = m3.captures[2]
        val = get_tm_map_value(mnemonic)
        val === nothing && error("Mnemonic $mnemonic not found in TM_MAP")
        parsed = tryparse(Float64, val)
        return parsed !== nothing ? parsed : val
    end
    # 2-part legacy: TM1.mnemonic
    m2 = match(r"^TM(\d+)\.(\w+)$", ref)
    if m2 !== nothing
        mnemonic = m2.captures[2]
        val = get_tm_map_value(mnemonic)
        val === nothing && error("Mnemonic $mnemonic not found in TM_MAP")
        parsed = tryparse(Float64, val)
        return parsed !== nothing ? parsed : val
    end
    error("Invalid TM reference: $ref")
end

"""
    get_all_tm_values() -> Dict{String, Any}

Get all current TM values from Redis TM_MAP.
"""
function get_all_tm_values()::Dict{String, Any}
    raw = get_all_tm_map()
    result = Dict{String, Any}()
    for (k, v) in raw
        parsed = tryparse(Float64, v)
        result[k] = parsed !== nothing ? parsed : v
    end
    return result
end

"""
    get_all_mnemonics() -> Vector{String}

Get all mnemonic names from Redis TM_MAP.
"""
function get_all_mnemonics()::Vector{String}
    return sort(collect(keys(get_all_tm_map())))
end

"""
    clear_tm_data()

No-op: TM data is managed by the external simulator via Redis TM_MAP.
"""
function clear_tm_data()
    # No-op
end

end # module TMInterface
