"""
    GenieApp

Genie.jl-based HTTP + WebSocket server for ASTRA.
Replaces ASTRAServer.jl (HTTP.jl-based).
"""
module GenieApp

export start_server, stop_server, broadcast_event

using Genie, Genie.Router, Genie.Requests
using Genie.Renderer.Json: json
using Genie.Renderer: respond
using JSON3
using Mongoc
using Base64
using ..ACSParser
using ..ACSValidator
using ..ACSExecutor
using ..ASTRAStepEngine
using ..TMInterface
using ..CommandDispatch
using ..DataLogger
using ..MongoStore
using ..RedisStore
using ..ProcedureRunner
using ..BackgroundScheduler

# Import specific functions
import ..ACSParser: load_from_string, list_procedures
import ..ACSValidator: validate_procedure
import ..ACSExecutor: run_test
import ..ASTRAStepEngine: create_step_session, step_next!, step_reset!, get_current_state, get_session, StepState
import ..TMInterface: get_all_mnemonics, get_all_tm_values
import ..DataLogger: log_test_result, get_test_results
import ..MongoStore: get_tm_mnemonics, get_tc_mnemonics, get_sco_commands, get_all_autocomplete_refs,
    init_procedures_indexes, migrate_to_hybrid_schema,
    get_procedures, get_procedure,
    get_procedure_versions, get_procedure_version,
    save_procedure, delete_procedure, restore_procedure,
    parse_xlsx_file, parse_out_file, upsert_tm_pids_bulk, get_tm_mnemonics_catalog, init_tm_mnemonics_indexes,
    get_tm_subsystems,
    save_user_telemetry, get_user_telemetry,
    get_user_telemetry_versions, get_user_telemetry_version,
    init_user_telemetry_indexes
import ..ProcedureRunner: start_procedure, pause_procedure, resume_procedure,
    abort_procedure, get_run_status, list_runs, set_event_broadcaster!
import ..BackgroundScheduler: register_background, remove_background,
    start_background, stop_background, start_all, stop_all,
    list_background, get_background_status,
    IntervalSchedule, EventDrivenSchedule
import ..RedisStore: get_satellite_config, set_satellite_config, get_test_phase, set_test_phase

# WebSocket connections tracking (for broadcast)
const WS_CLIENTS = Set{Any}()

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
        "output" => state.output,
    )
end

"""
    broadcast_event(event::Dict)

Broadcast an event to all connected WebSocket clients.
"""
function broadcast_event(event::Dict)
    msg = JSON3.write(event)
    for ws in WS_CLIENTS
        try
            write(ws, msg)
        catch e
            @warn "Failed to send to WebSocket client: $e"
        end
    end
end

"""
    register_routes!()

Register all API routes with Genie router.
"""
function register_routes!()
    # ============================================
    # Core DSL Operations
    # ============================================

    # Load procedure from content
    route("/api/v1/load", method=POST) do
        try
            body = JSON3.read(rawpayload())
            content = body.content
            filename = get(body, :filename, "<uploaded>")

            proc = load_from_string(content, filename)

            json(Dict(
                "success" => true,
                "test_name" => proc.name,
                "line_count" => length(proc.lines),
            ))
        catch e
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                400,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Validate procedure
    route("/api/v1/validate", method=POST) do
        try
            body = JSON3.read(rawpayload())
            test_name = body.test_name

            errors = validate_procedure(test_name)

            json(Dict(
                "valid" => isempty(errors),
                "errors" => [Dict(
                    "file" => e.file,
                    "line_number" => e.line_number,
                    "line_text" => e.line_text,
                    "message" => e.message,
                    "suggestion" => e.suggestion,
                    "severity" => string(e.severity),
                ) for e in errors],
            ))
        catch e
            respond(
                JSON3.write(Dict("valid" => false, "errors" => [Dict("message" => sprint(showerror, e))])),
                400,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Run full test
    route("/api/v1/run", method=POST) do
        try
            body = JSON3.read(rawpayload())
            test_name = body.test_name

            result = run_test(test_name)
            log_test_result(result)

            # Try to save to MongoDB
            try
                MongoStore.save_test_result(result)
            catch me
                @warn "Failed to save result to MongoDB: $me"
            end

            broadcast_event(Dict(
                "type" => "test_complete",
                "data" => Dict(
                    "test_name" => result.test_name,
                    "status" => string(result.status),
                    "duration" => result.duration_seconds,
                ),
            ))

            json(Dict(
                "success" => true,
                "status" => string(result.status),
                "duration" => result.duration_seconds,
                "log_entries" => length(result.log),
            ))
        catch e
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # ============================================
    # Step Execution
    # ============================================

    # Start step session
    route("/api/v1/step/start", method=POST) do
        try
            body = JSON3.read(rawpayload())
            test_name = body.test_name

            session = create_step_session(test_name)
            state = get_current_state(session)

            json(Dict(
                "success" => true,
                "session_id" => state.session_id,
                "state" => state_to_dict(state),
            ))
        catch e
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                400,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Step next
    route("/api/v1/step/next", method=POST) do
        try
            body = JSON3.read(rawpayload())
            session_id = body.session_id

            session = get_session(session_id)
            if session === nothing
                return respond(
                    JSON3.write(Dict("success" => false, "error" => "Session not found")),
                    404,
                    Dict("Content-Type" => "application/json"),
                )
            end

            state = step_next!(session)

            broadcast_event(Dict(
                "type" => "step_update",
                "data" => state_to_dict(state),
            ))

            json(state_to_dict(state))
        catch e
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Step reset
    route("/api/v1/step/reset", method=POST) do
        try
            body = JSON3.read(rawpayload())
            session_id = body.session_id

            session = get_session(session_id)
            if session === nothing
                return respond(
                    JSON3.write(Dict("success" => false, "error" => "Session not found")),
                    404,
                    Dict("Content-Type" => "application/json"),
                )
            end

            step_reset!(session)
            state = get_current_state(session)

            json(state_to_dict(state))
        catch e
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # ============================================
    # Procedures API
    # ============================================

    # Initialize indexes and migrate data (add ?force=true to force re-migration)
    route("/api/v1/procedures/init", method=POST) do
        try
            force = false
            try
                force_param = params(:force, "false")
                force = (force_param == "true")
            catch; end

            init_procedures_indexes()
            migrate_to_hybrid_schema(force=force)
            json(Dict("success" => true, "message" => "Indexes created and data migrated"))
        catch e
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Debug: check procedure versions in database
    route("/api/v1/debug/versions") do
        try
            versions_coll = MongoStore.db()["procedure_versions"]
            all_versions = collect(Mongoc.find(versions_coll, Mongoc.BSON()))
            json(Dict("count" => length(all_versions), "versions" => all_versions[1:min(5, length(all_versions))]))
        catch e
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Debug: test insert into procedure_versions
    route("/api/v1/debug/test-insert", method=POST) do
        try
            versions_coll = MongoStore.db()["procedure_versions"]
            test_doc = Mongoc.BSON()
            test_doc["procedure_id"] = "test123"
            test_doc["version"] = 1
            test_doc["content"] = "TEST content"
            test_doc["project"] = "test"
            test_doc["created_by"] = "test"
            test_doc["created_at"] = string(now())
            Mongoc.insert_one(versions_coll, test_doc)
            json(Dict("success" => true, "message" => "Test insert successful"))
        catch e
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Debug: run migration with verbose output
    route("/api/v1/debug/migrate", method=POST) do
        try
            # Manually migrate first document as test
            old_coll = MongoStore.db()["versioned_procedures"]
            procedures = MongoStore.db()["procedure_versions"]

            all_docs = collect(Mongoc.find(old_coll, Mongoc.BSON()))
            json(Dict("old_count" => length(all_docs), "sample" => length(all_docs) > 0 ? string(all_docs[1]["test_name"]) : "none"))
        catch e
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # List all procedures; optional: project, limit, offset, tags (comma-sep), include_deleted, deleted_only
    route("/api/v1/procedures") do
        try
            project = nothing
            try
                p = params(:project, nothing)
                project = (p !== nothing && p != "") ? p : nothing
            catch; end
            limit = nothing
            try
                l = params(:limit, nothing)
                if l !== nothing && l != ""
                    limit = parse(Int, l)
                end
            catch; end
            offset = nothing
            try
                o = params(:offset, nothing)
                if o !== nothing && o != ""
                    offset = parse(Int, o)
                end
            catch; end
            tags = nothing
            try
                t = params(:tags, nothing)
                if t !== nothing && t != ""
                    tags = [strip(s) for s in split(t, ',') if !isempty(strip(s))]
                end
            catch; end
            include_deleted = false
            try
                id = params(:include_deleted, "false")
                include_deleted = (id == "true")
            catch; end
            deleted_only = false
            try
                dod = params(:deleted_only, "false")
                deleted_only = (dod == "true")
            catch; end

            procedures = get_procedures(project=project, limit=limit, offset=offset, tags=tags, include_deleted=include_deleted, deleted_only=deleted_only)
            json(Dict("procedures" => procedures))
        catch e
            @error "procedures list error" exception=(e, catch_backtrace())
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Get single procedure
    route("/api/v1/procedures/:test_name") do
        try
            test_name = params(:test_name)
            project = nothing
            try
                project = params(:project, nothing)
            catch; end
            if project == ""
                project = nothing
            end
            proc = get_procedure(test_name; project=project)

            if proc === nothing
                return json(Dict("success" => false, "error" => "Procedure not found"), 404)
            end

            json(Dict("procedure" => proc))
        catch e
            if e isa ArgumentError
                return respond(
                    JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                    400,
                    Dict("Content-Type" => "application/json"),
                )
            end
            @error "procedure get error" exception=(e, catch_backtrace())
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Get procedure versions
    route("/api/v1/procedures/:test_name/versions") do
        try
            test_name = params(:test_name)
            project = nothing
            try
                project = params(:project, nothing)
            catch; end
            if project == ""
                project = nothing
            end

            proc = get_procedure(test_name; project=project)
            if proc === nothing
                return json(Dict("success" => false, "error" => "Procedure not found"), 404)
            end

            versions = get_procedure_versions(proc["_id"])
            json(Dict("versions" => versions))
        catch e
            if e isa ArgumentError
                return respond(
                    JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                    400,
                    Dict("Content-Type" => "application/json"),
                )
            end
            @error "procedure versions error" exception=(e, catch_backtrace())
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Get specific version
    route("/api/v1/procedures/:test_name/versions/:version") do
        try
            test_name = params(:test_name)
            version = parse(Int, params(:version))
            project = nothing
            try
                project = params(:project, nothing)
            catch; end
            if project == ""
                project = nothing
            end

            proc = get_procedure(test_name; project=project)
            if proc === nothing
                return json(Dict("success" => false, "error" => "Procedure not found"), 404)
            end

            version_data = get_procedure_version(proc["_id"], version)
            if version_data === nothing
                return json(Dict("success" => false, "error" => "Version not found"), 404)
            end

            json(Dict("version" => version_data))
        catch e
            if e isa ArgumentError
                return respond(
                    JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                    400,
                    Dict("Content-Type" => "application/json"),
                )
            end
            @error "procedure version error" exception=(e, catch_backtrace())
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Save procedure
    route("/api/v1/procedures", method=POST) do
        try
            body = JSON3.read(rawpayload())
            test_name = body.test_name
            content = body.content
            project = get(body, :project, "default")
            created_by = get(body, :created_by, "unknown")
            description = get(body, :description, "")
            tags = get(body, :tags, String[])
            change_message = get(body, :change_message, "")

            result = save_procedure(test_name, content, project, created_by; description=description, tags=tags, change_message=change_message)

            # Also load into in-memory parser if saved
            if get(result, "saved", false)
                try
                    load_from_string(content, "$test_name.tst")
                catch; end
            end

            json(result)
        catch e
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Delete procedure - soft delete (optional deleted_by in body)
    route("/api/v1/procedures/:test_name", method=DELETE) do
        try
            test_name = params(:test_name)
            project = nothing
            try
                project = params(:project, nothing)
            catch; end
            if project == ""
                project = nothing
            end
            deleted_by = ""
            try
                body = JSON3.read(rawpayload())
                deleted_by = get(body, :deleted_by, "")
            catch; end

            success = delete_procedure(test_name; project=project, deleted_by=deleted_by)
            json(Dict("success" => success))
        catch e
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Restore procedure - undelete (optional restored_by in body)
    route("/api/v1/procedures/:test_name/restore", method=POST) do
        try
            test_name = params(:test_name)
            project = nothing
            try
                project = params(:project, nothing)
            catch; end
            if project == ""
                project = nothing
            end
            restored_by = ""
            try
                body = JSON3.read(rawpayload())
                restored_by = get(body, :restored_by, "")
            catch; end

            success = restore_procedure(test_name; project=project, restored_by=restored_by)
            json(Dict("success" => success))
        catch e
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # ============================================
    # Procedure Runner (parallel execution)
    # ============================================

    # Start a procedure in a new task
    route("/api/v1/runner/start", method=POST) do
        try
            body = JSON3.read(rawpayload())
            test_name = body.test_name

            run = start_procedure(test_name)

            json(Dict(
                "success" => true,
                "run_id" => run.id,
                "procedure" => run.procedure_name,
                "status" => string(run.control.status),
            ))
        catch e
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                400,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Pause a running procedure
    route("/api/v1/runner/pause", method=POST) do
        try
            body = JSON3.read(rawpayload())
            run_id = body.run_id

            success = pause_procedure(run_id)

            json(Dict(
                "success" => success,
                "run_id" => run_id,
                "status" => success ? "paused" : "unchanged",
            ))
        catch e
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Resume a paused procedure
    route("/api/v1/runner/resume", method=POST) do
        try
            body = JSON3.read(rawpayload())
            run_id = body.run_id

            success = resume_procedure(run_id)

            json(Dict(
                "success" => success,
                "run_id" => run_id,
                "status" => success ? "running" : "unchanged",
            ))
        catch e
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Abort a running or paused procedure
    route("/api/v1/runner/abort", method=POST) do
        try
            body = JSON3.read(rawpayload())
            run_id = body.run_id

            success = abort_procedure(run_id)

            json(Dict(
                "success" => success,
                "run_id" => run_id,
                "status" => success ? "aborted" : "unchanged",
            ))
        catch e
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Get status of a specific run
    route("/api/v1/runner/status/:run_id") do
        try
            run_id = String(payload(:run_id))
            status = get_run_status(run_id)

            if status === nothing
                return respond(
                    JSON3.write(Dict("error" => "Run not found")),
                    404,
                    Dict("Content-Type" => "application/json"),
                )
            end

            respond(
                JSON3.write(status),
                200,
                Dict("Content-Type" => "application/json"),
            )
        catch e
            @error "Runner status error for $(payload(:run_id))" exception=(e, catch_backtrace())
            respond(
                JSON3.write(Dict("error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # List all runs
    route("/api/v1/runner/list") do
        try
            runs = list_runs()
            json(Dict("runs" => runs))
        catch e
            respond(
                JSON3.write(Dict("error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # ============================================
    # Telemetry (real-time in-memory)
    # ============================================

    # Get current TM values
    route("/api/v1/tm") do
        tm_values = get_all_tm_values()
        json(tm_values)
    end

    # Get TM mnemonics for autocomplete (in-memory)
    route("/api/v1/tm/mnemonics") do
        mnemonics = get_all_mnemonics()
        json(Dict("mnemonics" => mnemonics))
    end

    # ============================================
    # Mnemonic Management (MongoDB-backed)
    # ============================================

    # All TM mnemonics from MongoDB (enriched schema)
    route("/api/v1/mnemonics/tm") do
        try
            subsystem = nothing
            try
                s = params(:subsystem, nothing)
                subsystem = (s !== nothing && s != "") ? s : nothing
            catch; end
            mnemonics = MongoStore.get_tm_mnemonics(subsystem=subsystem)
            json(mnemonics)
        catch e
            respond(
                JSON3.write(Dict("error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Distinct TM subsystem names (separate path to avoid :subsystem wildcard conflict)
    route("/api/v1/telemetry/subsystems") do
        try
            subsystems = MongoStore.get_tm_subsystems()
            json(Dict("subsystems" => subsystems))
        catch e
            respond(
                JSON3.write(Dict("error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # TM mnemonics by subsystem
    route("/api/v1/mnemonics/tm/:subsystem") do
        try
            subsystem = String(payload(:subsystem))
            mnemonics = MongoStore.get_tm_mnemonics(subsystem=subsystem)
            json(mnemonics)
        catch e
            respond(
                JSON3.write(Dict("error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # All TC mnemonics
    route("/api/v1/mnemonics/tc") do
        try
            mnemonics = MongoStore.get_tc_mnemonics()
            json(mnemonics)
        catch e
            respond(
                JSON3.write(Dict("error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # All SCO commands
    route("/api/v1/mnemonics/sco") do
        try
            commands = MongoStore.get_sco_commands()
            json(commands)
        catch e
            respond(
                JSON3.write(Dict("error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # All mnemonics combined
    route("/api/v1/mnemonics/all") do
        try
            all_refs = MongoStore.get_all_autocomplete_refs()
            json(all_refs)
        catch e
            respond(
                JSON3.write(Dict("error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Add TM mnemonic (enriched schema - single record)
    route("/api/v1/mnemonics/tm", method=POST) do
        try
            body = JSON3.read(rawpayload())
            record = Dict{String,Any}(
                "cdbPidNo"    => string(body.cdbPidNo),
                "cdbMnemonic" => get(body, :cdbMnemonic, ""),
                "subsystem"   => get(body, :subsystem, ""),
                "type"        => get(body, :type, ""),
                "description" => get(body, :description, ""),
                "unit"        => get(body, :unit, ""),
                "sourceSheet" => get(body, :subsystem, ""),
            )
            result = MongoStore.upsert_tm_pid(record)
            json(Dict("success" => true, "result" => string(result)))
        catch e
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # ============================================
    # TM PID Catalog - Excel Import
    # ============================================

    # Upload .xlsx or .out file and import TM mnemonics
    # Accepts JSON body with base64-encoded data to avoid Genie multipart parser bugs
    route("/api/v1/telemetry/upload", method=POST) do
        try
            body = JSON3.read(rawpayload())
            original_name = string(body.filename)
            data_b64 = string(body.data)

            if isempty(data_b64)
                return respond(
                    JSON3.write(Dict(
                        "success" => false,
                        "error"   => "No file data provided. Send JSON with 'filename' and 'data' (base64)."
                    )),
                    400,
                    Dict("Content-Type" => "application/json"),
                )
            end

            ext = lowercase(splitext(original_name)[2])
            if ext ∉ (".xlsx", ".out")
                return respond(
                    JSON3.write(Dict(
                        "success" => false,
                        "error"   => "Unsupported file type: $ext. Supported: .xlsx, .out"
                    )),
                    400,
                    Dict("Content-Type" => "application/json"),
                )
            end

            # Decode base64 and write to temp file
            file_bytes = base64decode(data_b64)
            tmp_path = joinpath(tempdir(), "astra_upload_$(round(Int, time()))$ext")
            open(tmp_path, "w") do f
                write(f, file_bytes)
            end

            @info "TM upload: saved temp file $tmp_path ($(length(file_bytes)) bytes)"

            # Parse based on file type
            records = try
                if ext == ".xlsx"
                    MongoStore.parse_xlsx_file(tmp_path)
                else
                    MongoStore.parse_out_file(tmp_path)
                end
            catch e
                rm(tmp_path; force=true)
                return respond(
                    JSON3.write(Dict(
                        "success" => false,
                        "error"   => "Failed to parse $(uppercase(ext[2:end])): $(sprint(showerror, e))"
                    )),
                    422,
                    Dict("Content-Type" => "application/json"),
                )
            end

            rm(tmp_path; force=true)

            @info "TM upload: parsed $(length(records)) records from $original_name"

            # Bulk upsert into MongoDB
            stats = MongoStore.upsert_tm_pids_bulk(records; source_file=original_name)

            @info "TM upload complete: inserted=$(stats["inserted"]) updated=$(stats["updated"]) skipped=$(stats["skipped"]) errors=$(length(stats["errors"]))"

            status_code = isempty(stats["errors"]) ? 200 : 207
            respond(
                JSON3.write(Dict(
                    "success"  => true,
                    "filename" => original_name,
                    "stats"    => stats,
                )),
                status_code,
                Dict("Content-Type" => "application/json"),
            )
        catch e
            @error "TM upload endpoint error" exception=(e, catch_backtrace())
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Query tm_mnemonics catalog with pagination
    route("/api/v1/telemetry/catalog") do
        try
            subsystem = nothing
            try
                s = params(:subsystem, nothing)
                subsystem = (s !== nothing && s != "") ? s : nothing
            catch; end
            limit = nothing
            try
                l = params(:limit, nothing)
                if l !== nothing && l != ""
                    limit = parse(Int, l)
                end
            catch; end
            offset = nothing
            try
                o = params(:offset, nothing)
                if o !== nothing && o != ""
                    offset = parse(Int, o)
                end
            catch; end

            records = MongoStore.get_tm_mnemonics_catalog(subsystem=subsystem, limit=limit, offset=offset)
            json(Dict("records" => records, "count" => length(records)))
        catch e
            respond(
                JSON3.write(Dict("error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Add TC mnemonic
    route("/api/v1/mnemonics/tc", method=POST) do
        try
            body = JSON3.read(rawpayload())
            collection = MongoStore.db()["tc_mnemonics"]
            doc_data = Dict{String,Any}(
                "command" => body.command,
                "full_ref" => "TC.$(body.command)",
                "description" => get(body, :description, ""),
                "parameters" => get(body, :parameters, []),
                "subsystem" => get(body, :subsystem, ""),
                "category" => get(body, :category, ""),
            )
            doc = Mongoc.BSON(JSON3.write(doc_data))
            Mongoc.insert_one(collection, doc)
            json(Dict("success" => true))
        catch e
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Add SCO command
    route("/api/v1/mnemonics/sco", method=POST) do
        try
            body = JSON3.read(rawpayload())
            collection = MongoStore.db()["sco_commands"]
            doc_data = Dict{String,Any}(
                "command" => body.command,
                "full_ref" => "SCO.$(body.command)",
                "description" => get(body, :description, ""),
                "subsystem" => get(body, :subsystem, ""),
                "category" => get(body, :category, ""),
            )
            doc = Mongoc.BSON(JSON3.write(doc_data))
            Mongoc.insert_one(collection, doc)
            json(Dict("success" => true))
        catch e
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # ============================================
    # Test Results (MongoDB-backed)
    # ============================================

    route("/api/v1/results") do
        try
            limit = parse(Int, params(:limit, "50"))
            results = MongoStore.get_test_results(limit=limit)
            json(Dict("results" => results))
        catch e
            # Fallback to in-memory
            results = DataLogger.get_test_results(limit=50)
            json(Dict("results" => results))
        end
    end

    route("/api/v1/results/:id") do
        try
            id = payload(:id)
            result = MongoStore.get_test_result(id)
            if result === nothing
                return respond(
                    JSON3.write(Dict("error" => "Result not found")),
                    404,
                    Dict("Content-Type" => "application/json"),
                )
            end
            json(result)
        catch e
            respond(
                JSON3.write(Dict("error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # ============================================
    # Satellite Configuration (Redis-backed)
    # ============================================

    # Get full satellite config
    route("/api/v1/satellite-config") do
        try
            config = get_satellite_config()
            json(config)
        catch e
            respond(
                JSON3.write(Dict("error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Set a satellite config field
    route("/api/v1/satellite-config", method=POST) do
        try
            body = JSON3.read(rawpayload())
            field = string(body.field)
            value = string(body.value)
            set_satellite_config(field, value)
            json(Dict("success" => true, "field" => field, "value" => value))
        catch e
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Get current test phase
    route("/api/v1/satellite-config/test-phase") do
        try
            phase = get_test_phase()
            json(Dict("test_phase" => phase))
        catch e
            respond(
                JSON3.write(Dict("error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Set test phase
    route("/api/v1/satellite-config/test-phase", method=POST) do
        try
            body = JSON3.read(rawpayload())
            phase = string(body.test_phase)
            set_test_phase(phase)
            json(Dict("success" => true, "test_phase" => phase))
        catch e
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # ============================================
    # User Defined Telemetry (UD_TM) — single flat table per project
    # Mnemonics synced to tm_mnemonics (subsystem="UDTM") on save
    # so TM.UDTM.<mnemonic> works via existing autocomplete pipeline
    # ============================================

    # Get UD_TM for a project (single document)
    route("/api/v1/ud-tm") do
        try
            project = nothing
            try; p = Genie.Requests.getpayload(:project, nothing); project = (p !== nothing && p != "") ? p : nothing; catch; end
            if project === nothing
                project = "default"
            end
            doc = MongoStore.get_user_telemetry(project)
            if doc === nothing
                json(Dict("rows" => [], "latest_version" => 0))
            else
                json(doc)
            end
        catch e
            respond(JSON3.write(Dict("error" => sprint(showerror, e))), 500,
                Dict("Content-Type" => "application/json"))
        end
    end

    # Save UD_TM (rows + sync to tm_mnemonics)
    route("/api/v1/ud-tm", method=POST) do
        try
            payload = jsonpayload()
            rows = payload["rows"]
            project = get(payload, "project", "default")
            created_by = get(payload, "created_by", "user")
            change_message = get(payload, "change_message", "")
            changes = get(payload, "changes", [])

            result = MongoStore.save_user_telemetry(
                rows, project, created_by;
                change_message=change_message, changes=changes
            )
            json(result)
        catch e
            respond(JSON3.write(Dict("error" => sprint(showerror, e))), 500,
                Dict("Content-Type" => "application/json"))
        end
    end

    # Version history (must come before :version wildcard)
    route("/api/v1/ud-tm/versions") do
        try
            project = nothing
            try; p = Genie.Requests.getpayload(:project, nothing); project = (p !== nothing && p != "") ? p : nothing; catch; end
            if project === nothing
                project = "default"
            end
            doc = MongoStore.get_user_telemetry(project)
            if doc === nothing
                json(Dict("versions" => []))
            else
                versions = MongoStore.get_user_telemetry_versions(doc["_id"])
                json(Dict("versions" => versions))
            end
        catch e
            respond(JSON3.write(Dict("error" => sprint(showerror, e))), 500,
                Dict("Content-Type" => "application/json"))
        end
    end

    # Get specific version
    route("/api/v1/ud-tm/versions/:version") do
        try
            ver = parse(Int, params(:version))
            project = nothing
            try; p = Genie.Requests.getpayload(:project, nothing); project = (p !== nothing && p != "") ? p : nothing; catch; end
            if project === nothing
                project = "default"
            end
            doc = MongoStore.get_user_telemetry(project)
            if doc === nothing
                respond(JSON3.write(Dict("error" => "No UD_TM found for project")), 404,
                    Dict("Content-Type" => "application/json"))
            else
                version = MongoStore.get_user_telemetry_version(doc["_id"], ver)
                if version === nothing
                    respond(JSON3.write(Dict("error" => "Version $ver not found")), 404,
                        Dict("Content-Type" => "application/json"))
                else
                    json(Dict("version" => version))
                end
            end
        catch e
            respond(JSON3.write(Dict("error" => sprint(showerror, e))), 500,
                Dict("Content-Type" => "application/json"))
        end
    end

    # ============================================
    # Background Scheduler
    # ============================================

    # Register a new background schedule (upsert)
    route("/api/v1/background/register", method=POST) do
        try
            body = JSON3.read(rawpayload())
            proc_name = string(body.proc_name)
            schedule_type = string(get(body, :schedule_type, "interval"))

            schedule = if schedule_type == "event"
                EventDrivenSchedule(
                    string(get(body, :condition, "false")),
                    Float64(get(body, :poll_interval, 0.5)),
                    Bool(get(body, :restart_on_failure, true)),
                    Int(get(body, :max_consecutive_failures, 10)),
                )
            else
                IntervalSchedule(
                    Float64(get(body, :interval_seconds, 1.0)),
                    Bool(get(body, :restart_on_failure, true)),
                    Int(get(body, :max_consecutive_failures, 10)),
                )
            end

            register_background(proc_name, schedule)
            json(Dict("success" => true, "proc_name" => proc_name, "schedule" => schedule_type))
        catch e
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                400,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Remove a background schedule
    route("/api/v1/background/remove", method=DELETE) do
        try
            body = JSON3.read(rawpayload())
            proc_name = string(body.proc_name)
            success = remove_background(proc_name)
            json(Dict("success" => success, "proc_name" => proc_name))
        catch e
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Start one background procedure loop
    route("/api/v1/background/start", method=POST) do
        try
            body = JSON3.read(rawpayload())
            proc_name = string(body.proc_name)
            success = start_background(proc_name)
            json(Dict("success" => success, "proc_name" => proc_name))
        catch e
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Stop one background procedure loop
    route("/api/v1/background/stop", method=POST) do
        try
            body = JSON3.read(rawpayload())
            proc_name = string(body.proc_name)
            success = stop_background(proc_name)
            json(Dict("success" => success, "proc_name" => proc_name))
        catch e
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Start ALL registered background procedures
    route("/api/v1/background/start-all", method=POST) do
        try
            count = start_all()
            json(Dict("success" => true, "started" => count))
        catch e
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Stop ALL background procedure loops
    route("/api/v1/background/stop-all", method=POST) do
        try
            count = stop_all()
            json(Dict("success" => true, "stopped" => count))
        catch e
            respond(
                JSON3.write(Dict("success" => false, "error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # List all registered background procedures with live status
    route("/api/v1/background/list") do
        try
            entries = list_background()
            json(Dict("entries" => entries, "count" => length(entries)))
        catch e
            respond(
                JSON3.write(Dict("error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    # Get status of one background procedure
    route("/api/v1/background/status/:proc_name") do
        try
            proc_name = String(payload(:proc_name))
            status = get_background_status(proc_name)
            if status === nothing
                return respond(
                    JSON3.write(Dict("error" => "Not found: $proc_name")),
                    404,
                    Dict("Content-Type" => "application/json"),
                )
            end
            json(status)
        catch e
            respond(
                JSON3.write(Dict("error" => sprint(showerror, e))),
                500,
                Dict("Content-Type" => "application/json"),
            )
        end
    end

    @info "All ASTRA API routes registered"
end

"""
    start_server(port::Int=8080)

Start the Genie HTTP server.
"""
function start_server(port::Int=8080)
    # Configure Genie
    Genie.config.run_as_server = true
    Genie.config.server_port = port
    Genie.config.cors_headers["Access-Control-Allow-Origin"] = "*"
    Genie.config.cors_headers["Access-Control-Allow-Methods"] = "GET, POST, DELETE, OPTIONS"
    Genie.config.cors_headers["Access-Control-Allow-Headers"] = "Content-Type"

    # Initialize MongoDB
    try
        MongoStore.init_mongo()
        # Initialize indexes for hybrid schema
        try
            MongoStore.init_procedures_indexes()
            MongoStore.init_tm_mnemonics_indexes()
            @info "MongoDB indexes initialized"
        catch e
            @warn "Could not initialize indexes: $e"
        end
    catch e
        @warn "MongoDB not available, running without persistent storage: $e"
    end

    # Initialize Redis
    try
        RedisStore.init_redis()
    catch e
        @warn "Redis not available, satellite config disabled: $e"
    end

    # Register routes
    register_routes!()

    # Wire up event broadcasting for both runners
    set_event_broadcaster!(broadcast_event)
    BackgroundScheduler.set_event_broadcaster!(broadcast_event)

    # Auto-start persisted background procedures
    try
        BackgroundScheduler.auto_start_on_boot()
    catch e
        @warn "BackgroundScheduler auto-start failed: $e"
    end

    # Initialize background_schedules index
    try
        MongoStore.init_background_schedules_indexes()
    catch e
        @warn "Could not initialize background_schedules indexes: $e"
    end

    @info "Starting ASTRA server on port $port..."
    @info "Access the API at http://localhost:$port/api/v1"

    up(port; async=true)
end

"""
    stop_server()

Stop the Genie HTTP server.
"""
function stop_server()
    Genie.down()
    MongoStore.close_mongo()
    RedisStore.close_redis()
    @info "Server stopped"
end

end # module GenieApp
