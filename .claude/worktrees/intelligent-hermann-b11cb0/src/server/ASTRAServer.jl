"""
    ASTRAServer

HTTP + WebSocket server for ASTRA frontend communication.
"""
module ASTRAServer

export start_server, stop_server

using HTTP
using JSON3
using Sockets
using ..ACSParser
using ..ACSValidator
using ..ACSExecutor
using ..ASTRAStepEngine
using ..TMInterface
using ..CommandDispatch
using ..DataLogger

# Import specific functions we need
import ..ACSParser: load_from_string
import ..ACSValidator: validate_procedure
import ..ACSExecutor: run_test
import ..ASTRAStepEngine: create_step_session, step_next!, step_reset!, get_current_state, get_session, StepState
import ..TMInterface: get_all_mnemonics, get_all_tm_values
import ..DataLogger: log_test_result, get_test_results

# Server instance
const SERVER = Ref{Union{Nothing, HTTP.Server}}(nothing)
const WS_CONNECTIONS = Set{HTTP.WebSocket}()

"""
    handle_load(req::HTTP.Request) -> HTTP.Response

Load a .tst file from uploaded content.
"""
function handle_load(req::HTTP.Request)::HTTP.Response
    try
        body = JSON3.read(req.body)
        content = body.content
        filename = get(body, :filename, "<uploaded>")

        proc = load_from_string(content, filename)

        return HTTP.Response(200, JSON3.write(Dict(
            "success" => true,
            "test_name" => proc.name,
            "line_count" => length(proc.lines)
        )))
    catch e
        return HTTP.Response(400, JSON3.write(Dict(
            "success" => false,
            "error" => sprint(showerror, e)
        )))
    end
end

"""
    handle_validate(req::HTTP.Request) -> HTTP.Response

Validate a loaded procedure.
"""
function handle_validate(req::HTTP.Request)::HTTP.Response
    try
        body = JSON3.read(req.body)
        test_name = body.test_name

        errors = validate_procedure(test_name)

        return HTTP.Response(200, JSON3.write(Dict(
            "valid" => isempty(errors),
            "errors" => [Dict(
                "file" => e.file,
                "line_number" => e.line_number,
                "line_text" => e.line_text,
                "message" => e.message,
                "suggestion" => e.suggestion,
                "severity" => string(e.severity)
            ) for e in errors]
        )))
    catch e
        return HTTP.Response(400, JSON3.write(Dict(
            "valid" => false,
            "errors" => [Dict("message" => sprint(showerror, e))]
        )))
    end
end

"""
    handle_run(req::HTTP.Request) -> HTTP.Response

Run a full test procedure.
"""
function handle_run(req::HTTP.Request)::HTTP.Response
    try
        body = JSON3.read(req.body)
        test_name = body.test_name

        result = run_test(test_name)
        log_test_result(result)

        # Broadcast completion event via WebSocket
        broadcast_event(Dict(
            "type" => "test_complete",
            "data" => Dict(
                "test_name" => result.test_name,
                "status" => string(result.status),
                "duration" => result.duration_seconds
            )
        ))

        return HTTP.Response(200, JSON3.write(Dict(
            "success" => true,
            "status" => string(result.status),
            "duration" => result.duration_seconds,
            "log_entries" => length(result.log)
        )))
    catch e
        return HTTP.Response(500, JSON3.write(Dict(
            "success" => false,
            "error" => sprint(showerror, e)
        )))
    end
end

"""
    handle_step_start(req::HTTP.Request) -> HTTP.Response

Start a step execution session.
"""
function handle_step_start(req::HTTP.Request)::HTTP.Response
    try
        body = JSON3.read(req.body)
        test_name = body.test_name

        session = create_step_session(test_name)
        state = get_current_state(session)

        return HTTP.Response(200, JSON3.write(Dict(
            "success" => true,
            "session_id" => state.session_id,
            "state" => state_to_dict(state)
        )))
    catch e
        return HTTP.Response(400, JSON3.write(Dict(
            "success" => false,
            "error" => sprint(showerror, e)
        )))
    end
end

"""
    handle_step_next(req::HTTP.Request) -> HTTP.Response

Execute next step in a session.
"""
function handle_step_next(req::HTTP.Request)::HTTP.Response
    try
        body = JSON3.read(req.body)
        session_id = body.session_id

        session = get_session(session_id)
        if session === nothing
            return HTTP.Response(404, JSON3.write(Dict(
                "success" => false,
                "error" => "Session not found"
            )))
        end

        state = step_next!(session)

        # Broadcast step update via WebSocket
        broadcast_event(Dict(
            "type" => "step_update",
            "data" => state_to_dict(state)
        ))

        return HTTP.Response(200, JSON3.write(state_to_dict(state)))
    catch e
        return HTTP.Response(500, JSON3.write(Dict(
            "success" => false,
            "error" => sprint(showerror, e)
        )))
    end
end

"""
    handle_step_reset(req::HTTP.Request) -> HTTP.Response

Reset a step execution session.
"""
function handle_step_reset(req::HTTP.Request)::HTTP.Response
    try
        body = JSON3.read(req.body)
        session_id = body.session_id

        session = get_session(session_id)
        if session === nothing
            return HTTP.Response(404, JSON3.write(Dict(
                "success" => false,
                "error" => "Session not found"
            )))
        end

        step_reset!(session)
        state = get_current_state(session)

        return HTTP.Response(200, JSON3.write(state_to_dict(state)))
    catch e
        return HTTP.Response(500, JSON3.write(Dict(
            "success" => false,
            "error" => sprint(showerror, e)
        )))
    end
end

"""
    handle_list_procedures(req::HTTP.Request) -> HTTP.Response

List all loaded procedures.
"""
function handle_list_procedures(req::HTTP.Request)::HTTP.Response
    procedures = list_procedures()
    return HTTP.Response(200, JSON3.write(Dict(
        "procedures" => procedures
    )))
end

"""
    handle_get_tm(req::HTTP.Request) -> HTTP.Response

Get current TM values.
"""
function handle_get_tm(req::HTTP.Request)::HTTP.Response
    tm_values = get_all_tm_values()
    return HTTP.Response(200, JSON3.write(tm_values))
end

"""
    handle_get_mnemonics(req::HTTP.Request) -> HTTP.Response

Get all known TM mnemonics for autocomplete.
"""
function handle_get_mnemonics(req::HTTP.Request)::HTTP.Response
    mnemonics = get_all_mnemonics()
    return HTTP.Response(200, JSON3.write(Dict(
        "mnemonics" => mnemonics
    )))
end

"""
    handle_get_results(req::HTTP.Request) -> HTTP.Response

Get test results.
"""
function handle_get_results(req::HTTP.Request)::HTTP.Response
    results = get_test_results(limit=50)
    return HTTP.Response(200, JSON3.write(Dict(
        "results" => results
    )))
end

"""
    state_to_dict(state::StepState) -> Dict

Convert StepState to dictionary for JSON serialization.
"""
function state_to_dict(state::StepState)::Dict
    return Dict(
        "session_id" => state.session_id,
        "test_name" => state.test_name,
        "line_number" => state.current_line_number,
        "line_text" => state.current_line_text,
        "status" => string(state.status),
        "variables" => state.variables,
        "call_stack" => state.call_stack,
        "output" => state.output
    )
end

"""
    broadcast_event(event::Dict)

Broadcast an event to all connected WebSocket clients.
"""
function broadcast_event(event::Dict)
    msg = JSON3.write(event)
    for ws in WS_CONNECTIONS
        try
            HTTP.send(ws, msg)
        catch e
            @warn "Failed to send to WebSocket client: $e"
        end
    end
end

"""
    handle_websocket(ws::HTTP.WebSocket)

Handle WebSocket connections for real-time events.
"""
function handle_websocket(ws::HTTP.WebSocket)
    push!(WS_CONNECTIONS, ws)
    @info "WebSocket client connected (total: $(length(WS_CONNECTIONS)))"

    try
        # Send initial connection message
        HTTP.send(ws, JSON3.write(Dict(
            "type" => "connected",
            "data" => Dict("message" => "Connected to ASTRA server")
        )))

        # Keep connection alive
        while !eof(ws)
            data = String(readavailable(ws))
            # Echo or handle client messages if needed
        end
    catch e
        @warn "WebSocket error: $e"
    finally
        delete!(WS_CONNECTIONS, ws)
        @info "WebSocket client disconnected (total: $(length(WS_CONNECTIONS)))"
    end
end

"""
    serve_static_file(req::HTTP.Request, root::String) -> HTTP.Response

Serve static files from the specified root directory.
"""
function serve_static_file(req::HTTP.Request, root::String)::HTTP.Response
    # Get the requested path
    path = HTTP.URIs.unescapeuri(req.target)

    # Default to index.html for root path
    if path == "/" || path == ""
        path = "/index.html"
    end

    # Remove leading slash and build file path
    filepath = joinpath(root, lstrip(path, '/'))

    # Security: prevent directory traversal
    if !startswith(abspath(filepath), abspath(root))
        return HTTP.Response(403, "Forbidden")
    end

    # Check if file exists
    if !isfile(filepath)
        return HTTP.Response(404, "Not Found: $path")
    end

    # Determine content type based on extension
    content_type = if endswith(filepath, ".html")
        "text/html"
    elseif endswith(filepath, ".css")
        "text/css"
    elseif endswith(filepath, ".js")
        "application/javascript"
    elseif endswith(filepath, ".json")
        "application/json"
    elseif endswith(filepath, ".png")
        "image/png"
    elseif endswith(filepath, ".jpg") || endswith(filepath, ".jpeg")
        "image/jpeg"
    elseif endswith(filepath, ".svg")
        "image/svg+xml"
    else
        "application/octet-stream"
    end

    # Read and serve the file
    try
        content = read(filepath)
        return HTTP.Response(200, ["Content-Type" => content_type], body=content)
    catch e
        return HTTP.Response(500, "Internal Server Error: $(sprint(showerror, e))")
    end
end

"""
    start_server(port::Int=8080; cors::Bool=true)

Start the HTTP + WebSocket server.
"""
function start_server(port::Int=8080; cors::Bool=true)
    router = HTTP.Router()

    # CORS middleware
    function cors_middleware(handler)
        return function(req::HTTP.Request)
            if cors
                headers = [
                    "Access-Control-Allow-Origin" => "*",
                    "Access-Control-Allow-Methods" => "GET, POST, OPTIONS",
                    "Access-Control-Allow-Headers" => "Content-Type"
                ]

                if req.method == "OPTIONS"
                    return HTTP.Response(200, headers)
                end

                resp = handler(req)
                for (k, v) in headers
                    HTTP.setheader(resp, k => v)
                end
                return resp
            else
                return handler(req)
            end
        end
    end

    # Register routes
    HTTP.register!(router, "POST", "/api/load", cors_middleware(handle_load))
    HTTP.register!(router, "POST", "/api/validate", cors_middleware(handle_validate))
    HTTP.register!(router, "POST", "/api/run", cors_middleware(handle_run))
    HTTP.register!(router, "POST", "/api/step/start", cors_middleware(handle_step_start))
    HTTP.register!(router, "POST", "/api/step/next", cors_middleware(handle_step_next))
    HTTP.register!(router, "POST", "/api/step/reset", cors_middleware(handle_step_reset))
    HTTP.register!(router, "GET", "/api/procedures", cors_middleware(handle_list_procedures))
    HTTP.register!(router, "GET", "/api/tm", cors_middleware(handle_get_tm))
    HTTP.register!(router, "GET", "/api/tm/mnemonics", cors_middleware(handle_get_mnemonics))
    HTTP.register!(router, "GET", "/api/results", cors_middleware(handle_get_results))

    # Create a wrapper that handles API routes via router, and everything else as static files
    function request_handler(req::HTTP.Request)
        # If it's an API route, use the router
        if startswith(req.target, "/api")
            return router(req)
        else
            # Otherwise serve static files
            return serve_static_file(req, "frontend")
        end
    end

    @info "Starting ASTRA server on port $port..."
    @info "Access the GUI at http://localhost:$port"

    SERVER[] = HTTP.serve(request_handler, "0.0.0.0", port; verbose=false)
end

"""
    stop_server()

Stop the HTTP server.
"""
function stop_server()
    if SERVER[] !== nothing
        close(SERVER[])
        SERVER[] = nothing
        @info "Server stopped"
    end
end

end # module ASTRAServer
