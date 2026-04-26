"""
    RedisStore - Redis connection and satellite configuration management

Provides access to the `satellite_config` Redis hash map for storing
test phase and other satellite configuration values.
"""
module RedisStore

export init_redis, close_redis
export get_satellite_config, set_satellite_config
export get_test_phase, set_test_phase
export get_tm_map_value, get_all_tm_map

using Redis

const REDIS_CONN = Ref{Union{Nothing, RedisConnection}}(nothing)
const SATELLITE_CONFIG_KEY = "satellite_config"

"""
    init_redis(; host, port)

Initialize the Redis connection. Uses REDIS_HOST/REDIS_PORT env vars or defaults.
"""
function init_redis(;
    host::String = get(ENV, "REDIS_HOST", "localhost"),
    port::Int = parse(Int, get(ENV, "REDIS_PORT", "6379"))
)
    try
        REDIS_CONN[] = RedisConnection(host=host, port=port)
        @info "Connected to Redis at $host:$port"
    catch e
        @warn "Failed to connect to Redis at $host:$port: $e"
        REDIS_CONN[] = nothing
    end
end

"""
    close_redis()

Close the Redis connection.
"""
function close_redis()
    if REDIS_CONN[] !== nothing
        try
            disconnect(REDIS_CONN[])
        catch; end
        REDIS_CONN[] = nothing
        @info "Redis connection closed"
    end
end

function get_conn()
    if REDIS_CONN[] === nothing
        init_redis()
    end
    return REDIS_CONN[]
end

"""
    get_satellite_config() -> Dict{String,String}

Get all fields from the satellite_config Redis hash map.
"""
function get_satellite_config()::Dict{String,String}
    conn = get_conn()
    if conn === nothing
        return Dict{String,String}()
    end
    try
        result = hgetall(conn, SATELLITE_CONFIG_KEY)
        return result isa Dict ? result : Dict{String,String}()
    catch e
        @warn "Redis HGETALL failed: $e"
        return Dict{String,String}()
    end
end

"""
    set_satellite_config(field::String, value::String)

Set a field in the satellite_config Redis hash map.
"""
function set_satellite_config(field::String, value::String)
    conn = get_conn()
    if conn === nothing
        @warn "Redis not available, cannot set satellite config"
        return
    end
    try
        hset(conn, SATELLITE_CONFIG_KEY, field, value)
    catch e
        @warn "Redis HSET failed: $e"
    end
end

"""
    get_test_phase() -> String

Get the current test_phase from satellite_config.
"""
function get_test_phase()::String
    conn = get_conn()
    if conn === nothing
        return ""
    end
    try
        result = hget(conn, SATELLITE_CONFIG_KEY, "test_phase")
        return result === nothing ? "" : string(result)
    catch e
        @warn "Redis HGET test_phase failed: $e"
        return ""
    end
end

"""
    set_test_phase(phase::String)

Set the test_phase in satellite_config.
"""
function set_test_phase(phase::String)
    set_satellite_config("test_phase", phase)
    @info "Test phase set to: $phase"
end

# =============================================
# TM_MAP - Telemetry values from external simulator
# =============================================

const TM_MAP_KEY = "TM_MAP"

"""
    get_tm_map_value(mnemonic::String) -> Union{String, Nothing}

Get a single telemetry value from the TM_MAP Redis hash.
"""
function get_tm_map_value(mnemonic::String)::Union{String, Nothing}
    conn = get_conn()
    conn === nothing && return nothing
    try
        result = hget(conn, TM_MAP_KEY, mnemonic)
        return result === nothing ? nothing : string(result)
    catch e
        @warn "Redis HGET TM_MAP failed for $mnemonic: $e"
        return nothing
    end
end

"""
    get_all_tm_map() -> Dict{String, String}

Get all telemetry values from the TM_MAP Redis hash.
"""
function get_all_tm_map()::Dict{String, String}
    conn = get_conn()
    conn === nothing && return Dict{String,String}()
    try
        result = hgetall(conn, TM_MAP_KEY)
        return result isa Dict ? result : Dict{String,String}()
    catch e
        @warn "Redis HGETALL TM_MAP failed: $e"
        return Dict{String,String}()
    end
end

end # module RedisStore
