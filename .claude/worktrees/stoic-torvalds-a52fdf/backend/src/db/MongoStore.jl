"""
    MongoStore

MongoDB data access layer for ASTRA.
Handles procedures, mnemonics (TM/TC/SCO), test results, and TM history.
"""
module MongoStore

export get_tm_mnemonics, get_tc_mnemonics, get_sco_commands, get_all_autocomplete_refs
export save_test_result, get_test_results, get_test_result
export save_tm_snapshot, get_tm_history
export init_mongo, close_mongo, db
export init_procedures_indexes, migrate_to_hybrid_schema
export get_procedures, get_procedure
export get_procedure_versions, get_procedure_version
export save_procedure, delete_procedure, restore_procedure
export parse_xlsx_file, parse_out_file, upsert_tm_pids_bulk, get_tm_mnemonics_catalog, get_tm_subsystems
export save_user_telemetry, get_user_telemetry
export get_user_telemetry_versions, get_user_telemetry_version
export init_user_telemetry_indexes
export save_background_schedule, delete_background_schedule, list_background_schedules, init_background_schedules_indexes

using Mongoc
using JSON3
using Dates
using XLSX

# MongoDB connection
const MONGO_URI = Ref{String}(get(ENV, "MONGO_URI", "mongodb://localhost:27017"))
const DB_NAME = Ref{String}(get(ENV, "ASTRA_DB", "astra"))
const CLIENT = Ref{Union{Nothing, Mongoc.Client}}(nothing)

"""
    init_mongo(uri::String, db_name::String)

Initialize MongoDB connection.
"""
function init_mongo(;
    uri::String = get(ENV, "MONGO_URI", "mongodb://localhost:27017"),
    db_name::String = get(ENV, "ASTRA_DB", "astra")
)
    MONGO_URI[] = uri
    DB_NAME[] = db_name
    CLIENT[] = Mongoc.Client(uri)
    @info "Connected to MongoDB: $uri / $db_name"
end

function get_client()
    if CLIENT[] === nothing
        init_mongo()
    end
    return CLIENT[]
end

function db()
    return get_client()[DB_NAME[]]
end

"""
    close_mongo()

Close MongoDB connection.
"""
function close_mongo()
    if CLIENT[] !== nothing
        # Mongoc.Client doesn't need explicit close in Julia
        CLIENT[] = nothing
        @info "MongoDB connection closed"
    end
end

# =============================================
# Hybrid Schema - Procedures (v2)
# =============================================
# Two collections:
#   - procedures: main metadata, latest content cached
#   - procedure_versions: full version history
# Timestamps: ISO 8601 (UTC).

# ISO 8601 UTC timestamp with Z suffix (API timestamps are UTC)
_iso_now() = string(Dates.now()) * "Z"
const PREVIEW_MAX_LEN = 200

"""
    _normalize_tags(tags::Vector{String}) -> Vector{String}

Normalize tags: trim, lowercase, dedupe.
"""
function _normalize_tags(tags::Vector{String})
    seen = Set{String}()
    out = String[]
    for t in tags
        s = strip(lowercase(t))
        isempty(s) && continue
        s in seen && continue
        push!(seen, s)
        push!(out, s)
    end
    return out
end

"""
    init_procedures_indexes()

Create indexes for the hybrid procedure schema.
"""
function init_procedures_indexes()
    procedures = db()["procedures"]
    procedure_versions = db()["procedure_versions"]

    # Procedures collection indexes
    try
        Mongoc.create_index(procedures, Mongoc.BSON("project" => 1, "test_name" => 1), unique=true)
        @info "Created unique index on procedures: project + test_name"
    catch e
        @warn "Index may already exist: $e"
    end
    try
        Mongoc.create_index(procedures, Mongoc.BSON("project" => 1))
        @info "Created index on procedures: project"
    catch e
        @warn "Index may already exist: $e"
    end
    try
        Mongoc.create_index(procedures, Mongoc.BSON("test_name" => "text", "description" => "text"))
        @info "Created text index on procedures: test_name + description"
    catch e
        @warn "Index may already exist: $e"
    end
    try
        Mongoc.create_index(procedures, Mongoc.BSON("tags" => 1))
        @info "Created index on procedures: tags"
    catch e
        @warn "Index may already exist: $e"
    end

    # Procedure versions indexes
    try
        Mongoc.create_index(procedure_versions, Mongoc.BSON("procedure_id" => 1, "version" => -1))
        @info "Created index on procedure_versions: procedure_id + version"
    catch e
        @warn "Index may already exist: $e"
    end
end

"""
    migrate_to_hybrid_schema()

Migrate data from old versioned_procedures collection to new hybrid schema.
"""
function migrate_to_hybrid_schema(; force::Bool=false)
    old_collection = db()["versioned_procedures"]
    procedures = db()["procedures"]
    procedure_versions = db()["procedure_versions"]

    # Check if already migrated
    count_old = length(collect(Mongoc.find(old_collection, Mongoc.BSON())))
    count_new = length(collect(Mongoc.find(procedures, Mongoc.BSON())))

    @info "Migration check: old=$count_old, new=$count_new, force=$force"

    if count_new > 0 && !force
        @info "Hybrid schema already exists ($count_new procedures). Skipping migration."
        return
    end

    # Clear existing data if force migration
    if force && count_new > 0
        @info "Force migration: clearing existing data..."
        Mongoc.delete_many(procedures, Mongoc.BSON())
        Mongoc.delete_many(procedure_versions, Mongoc.BSON())
    end

    @info "Starting migration from old schema to hybrid..."

    docs = collect(Mongoc.find(old_collection, Mongoc.BSON()))
    @info "Found $(length(docs)) documents to migrate"

    for doc in docs
        try
            test_name = string(doc["test_name"])
            versions_raw = get(doc, "versions", [])

            @info "Processing: $test_name, versions count: $(length(versions_raw))"

            if isempty(versions_raw)
                @info "Skipping $test_name - no versions"
                continue
            end

            # Get latest version
            latest = versions_raw[end]
            latest_version = get(latest, "version", 1)
            latest_content = get(latest, "content", "")
            project = get(latest, "project", "default")
            created_by = get(latest, "created_by", "system")
            created_at = get(latest, "created_at", string(Dates.now()))
            preview_str = length(latest_content) > PREVIEW_MAX_LEN ? latest_content[1:PREVIEW_MAX_LEN] : latest_content

            @info "Inserting $test_name v$latest_version into procedures"

            # Insert into procedures collection (v2 shape with created_by, preview)
            procedure_doc = Dict{String,Any}(
                "test_name" => test_name,
                "project" => project,
                "description" => "",
                "tags" => [],
                "latest_version" => latest_version,
                "latest_content" => latest_content,
                "preview" => preview_str,
                "is_deleted" => false,
                "created_by" => created_by,
                "updated_by" => created_by,
                "created_at" => created_at,
                "updated_at" => created_at,
            )
            # Insert procedure document
            procedure_bson = Mongoc.BSON()
            for (k, v) in procedure_doc
                procedure_bson[k] = v
            end
            result = Mongoc.insert_one(procedures, procedure_bson)
            # Get the inserted ID
            inserted_oid = result.inserted_oid
            procedure_id = string(inserted_oid)

            # Insert all versions into procedure_versions
            for v in versions_raw
                version_bson = Mongoc.BSON()
                version_bson["procedure_id"] = procedure_id
                version_bson["version"] = get(v, "version", 1)
                version_bson["content"] = get(v, "content", "")
                version_bson["project"] = get(v, "project", "default")
                version_bson["created_by"] = get(v, "created_by", "system")
                version_bson["created_at"] = get(v, "created_at", string(now()))
                Mongoc.insert_one(procedure_versions, version_bson)
            end

            @info "Migrated: $test_name ($latest_version versions) - procedure_id: $procedure_id"
        catch e
            @warn "Failed to migrate document: $e"
            continue
        end
    end

    @info "Migration complete!"
end

"""
    get_procedures(; project=nothing, limit=nothing, offset=nothing, tags=nothing, include_deleted=false, deleted_only=false) -> Vector{Dict}

List procedures using hybrid schema.
Optional: limit, offset (pagination), tags (array of strings, procedures must contain all), include_deleted, deleted_only.
"""
function get_procedures(; project=nothing, limit=nothing, offset=nothing, tags=nothing, include_deleted::Bool=false, deleted_only::Bool=false)
    procedures = db()["procedures"]

    # Build query using Dict
    query_dict = Dict{String,Any}()
    if project !== nothing
        query_dict["project"] = project
    end
    if deleted_only
        query_dict["is_deleted"] = true
    elseif !include_deleted
        query_dict["is_deleted"] = false
    end
    if tags !== nothing && !isempty(tags)
        query_dict["tags"] = Mongoc.BSON("\$all" => collect(tags))
    end
    
    query_filter = isempty(query_dict) ? Mongoc.BSON() : Mongoc.BSON(query_dict)

    use_opts = (limit !== nothing && limit > 0) || (offset !== nothing && offset > 0)
    opts = use_opts ? Mongoc.BSON() : nothing
    if use_opts
        if limit !== nothing && limit > 0
            opts["limit"] = limit
        end
        if offset !== nothing && offset > 0
            opts["skip"] = offset
        end
    end

    results = use_opts ? Mongoc.find(procedures, query_filter; options=opts) : Mongoc.find(procedures, query_filter)
    proc_list = Dict[]
    for doc in results
        push!(proc_list, Dict{String,Any}(
            "_id" => string(doc["_id"]),
            "test_name" => string(doc["test_name"]),
            "project" => string(doc["project"]),
            "description" => get(doc, "description", ""),
            "tags" => get(doc, "tags", []),
            "latest_version" => get(doc, "latest_version", 1),
            "preview" => get(doc, "preview", ""),
            "latest_content" => get(doc, "latest_content", ""),
            "created_by" => get(doc, "created_by", ""),
            "updated_by" => get(doc, "updated_by", ""),
            "created_at" => get(doc, "created_at", ""),
            "updated_at" => get(doc, "updated_at", ""),
            "is_deleted" => get(doc, "is_deleted", false),
            "deleted_at" => get(doc, "deleted_at", nothing),
            "deleted_by" => get(doc, "deleted_by", ""),
        ))
    end
    return proc_list
end

"""
    get_procedure(test_name::AbstractString; project=nothing) -> Union{Dict, Nothing}

Get a procedure by name using hybrid schema.
When project is nothing and multiple procedures share the same test_name (in different projects),
throws ArgumentError so the API can return 400. Callers should pass project when known.
"""
function get_procedure(test_name::AbstractString; project=nothing)
    procedures = db()["procedures"]

    if project !== nothing
        query = Mongoc.BSON("test_name" => test_name, "project" => project, "is_deleted" => false)
        result = Mongoc.find_one(procedures, query)
        if result === nothing
            return nothing
        end
        return _procedure_doc_to_dict(result)
    end

    # No project: ensure at most one match to avoid undefined behavior
    query = Mongoc.BSON("test_name" => test_name, "is_deleted" => false)
    results = collect(Mongoc.find(procedures, query))
    if isempty(results)
        return nothing
    end
    if length(results) > 1
        throw(ArgumentError("Multiple procedures match test_name \"$test_name\"; specify ?project= to disambiguate"))
    end
    return _procedure_doc_to_dict(results[1])
end

function _procedure_doc_to_dict(result)
    return Dict{String,Any}(
        "_id" => string(result["_id"]),
        "test_name" => string(result["test_name"]),
        "project" => string(result["project"]),
        "description" => get(result, "description", ""),
        "tags" => get(result, "tags", []),
        "latest_version" => get(result, "latest_version", 1),
        "latest_content" => get(result, "latest_content", ""),
        "preview" => get(result, "preview", ""),
        "is_deleted" => get(result, "is_deleted", false),
        "created_by" => get(result, "created_by", ""),
        "updated_by" => get(result, "updated_by", ""),
        "created_at" => get(result, "created_at", ""),
        "updated_at" => get(result, "updated_at", ""),
        "deleted_at" => get(result, "deleted_at", nothing),
        "deleted_by" => get(result, "deleted_by", ""),
    )
end

"""
    get_procedure_versions(procedure_id::AbstractString) -> Vector{Dict}

Get all versions of a procedure.
"""
function get_procedure_versions(procedure_id::AbstractString)
    versions = db()["procedure_versions"]
    query = Mongoc.BSON("procedure_id" => procedure_id)
    results = collect(Mongoc.find(versions, query))

    # Sort by version descending in Julia
    sorted_results = sort(results, by=x->get(x, "version", 0), rev=true)

    version_list = Dict[]
    for doc in sorted_results
        push!(version_list, Dict{String,Any}(
            "_id" => string(doc["_id"]),
            "procedure_id" => string(doc["procedure_id"]),
            "version" => doc["version"],
            "content" => string(doc["content"]),
            "project" => string(doc["project"]),
            "created_by" => string(doc["created_by"]),
            "created_at" => string(doc["created_at"]),
            "change_message" => get(doc, "change_message", ""),
        ))
    end
    return version_list
end

"""
    get_procedure_version(procedure_id::AbstractString, version::Int) -> Union{Dict, Nothing}

Get a specific version of a procedure.
"""
function get_procedure_version(procedure_id::AbstractString, version::Int)
    versions = db()["procedure_versions"]
    query = Mongoc.BSON("procedure_id" => procedure_id, "version" => version)
    result = Mongoc.find_one(versions, query)

    if result === nothing
        return nothing
    end

    return Dict{String,Any}(
        "_id" => string(result["_id"]),
        "procedure_id" => string(result["procedure_id"]),
        "version" => result["version"],
        "content" => string(result["content"]),
        "project" => string(result["project"]),
        "created_by" => string(result["created_by"]),
        "created_at" => string(result["created_at"]),
        "change_message" => get(result, "change_message", ""),
    )
end

"""
    save_procedure(test_name, content, project, created_by; description="", tags=[], change_message="") -> Dict

Save a new version of a procedure using hybrid schema.
Uses ISO 8601 timestamps; normalizes tags (trim, lowercase, dedupe); stores created_by/updated_by and preview.
change_message is required and stored on the version for user comments/changelog.
"""
function save_procedure(
    test_name::AbstractString,
    content::AbstractString,
    project::AbstractString,
    created_by::AbstractString;
    description::AbstractString = "",
    tags::Vector{String} = String[],
    change_message::AbstractString = ""
)
    procedures = db()["procedures"]
    procedure_versions = db()["procedure_versions"]
    now_time = _iso_now()
    tags_norm = _normalize_tags(collect(tags))
    preview_str = length(content) > PREVIEW_MAX_LEN ? content[1:PREVIEW_MAX_LEN] : content

    # Try to find existing procedure
    existing = Mongoc.find_one(procedures, Mongoc.BSON("test_name" => test_name, "project" => project))

    if existing !== nothing
        procedure_id = string(existing["_id"])
        current_version = get(existing, "latest_version", 0)

        # Check if content changed
        latest_content = get(existing, "latest_content", "")
        if latest_content == content
            return Dict{String,Any}(
                "saved" => false,
                "reason" => "no_change",
                "message" => "Content unchanged from latest version",
            )
        end

        next_version = current_version + 1

        # Update procedures collection: content, preview, updated_by, and optional metadata (description, tags)
        set_fields = Dict{String,Any}(
            "latest_version" => next_version,
            "latest_content" => content,
            "preview" => preview_str,
            "updated_by" => created_by,
            "updated_at" => now_time,
        )
        set_fields["description"] = description
        set_fields["tags"] = tags_norm
        update_doc = Mongoc.BSON("\$set" => Mongoc.BSON(set_fields))
        Mongoc.update_one(procedures, Mongoc.BSON("_id" => existing["_id"]), update_doc)
    else
        # Create new procedure with created_by
        next_version = 1

        procedure_doc = Dict{String,Any}(
            "test_name" => test_name,
            "project" => project,
            "description" => description,
            "tags" => tags_norm,
            "latest_version" => 1,
            "latest_content" => content,
            "preview" => preview_str,
            "is_deleted" => false,
            "created_by" => created_by,
            "updated_by" => created_by,
            "created_at" => now_time,
            "updated_at" => now_time,
        )
        result = Mongoc.insert_one(procedures, Mongoc.BSON(JSON3.write(procedure_doc)))
        procedure_id = string(result.inserted_oid)
    end

    # Insert new version into procedure_versions (ISO timestamp, change_message is required)
    version_doc = Dict{String,Any}(
        "procedure_id" => procedure_id,
        "version" => next_version,
        "content" => content,
        "project" => project,
        "created_by" => created_by,
        "created_at" => now_time,
        "change_message" => change_message,
    )
    Mongoc.insert_one(procedure_versions, Mongoc.BSON(JSON3.write(version_doc)))

    @info "Saved procedure v2: $test_name v$next_version by $created_by"
    return Dict{String,Any}(
        "saved" => true,
        "version" => next_version,
        "test_name" => test_name,
        "project" => project,
        "created_by" => created_by,
        "created_at" => now_time,
    )
end

"""
    delete_procedure(test_name::AbstractString; project=nothing, deleted_by="") -> Bool

Soft delete a procedure. Sets is_deleted, deleted_at (ISO), and deleted_by for audit.
"""
function delete_procedure(test_name::AbstractString; project=nothing, deleted_by::AbstractString="")
    procedures = db()["procedures"]

    if project !== nothing
        query = Mongoc.BSON("test_name" => test_name, "project" => project)
    else
        query = Mongoc.BSON("test_name" => test_name)
    end

    update_doc = Mongoc.BSON("\$set" => Mongoc.BSON(
        "is_deleted" => true,
        "updated_at" => _iso_now(),
        "deleted_at" => _iso_now(),
        "deleted_by" => deleted_by,
    ))
    result = Mongoc.update_one(procedures, query, update_doc)

    return result.modified_count > 0
end

"""
    restore_procedure(test_name::AbstractString; project=nothing, restored_by::AbstractString="") -> Bool

Restore a soft-deleted procedure. Sets is_deleted=false and clears deleted_at, deleted_by.
"""
function restore_procedure(test_name::AbstractString; project=nothing, restored_by::AbstractString="")
    procedures = db()["procedures"]

    if project !== nothing
        query = Mongoc.BSON("test_name" => test_name, "project" => project, "is_deleted" => true)
    else
        query = Mongoc.BSON("test_name" => test_name, "is_deleted" => true)
    end

    update_doc = Mongoc.BSON(
        "\$set" => Mongoc.BSON(
            "is_deleted" => false,
            "updated_at" => _iso_now(),
            "updated_by" => restored_by,
        ),
        "\$unset" => Mongoc.BSON("deleted_at" => 1, "deleted_by" => 1),
    )
    result = Mongoc.update_one(procedures, query, update_doc)

    return result.modified_count > 0
end

# =============================================
# TM Mnemonics (Enriched Schema from Excel Import)
# =============================================

# Column index -> field name mapping for Excel sheets
const XLSX_COLUMN_MAP = Dict{Int,String}(
    1  => "slNo",
    2  => "subsystem",
    3  => "cdbPidNo",
    4  => "cdbMnemonic",
    5  => "description",
    6  => "type",
    7  => "processingType",
    8  => "noOfWords",
    9  => "channelNo",
    10 => "frameNo",
    11 => "startBit",
    12 => "endBit",
    13 => "samplingRate",
    14 => "dwellAddress",
    15 => "range",
    16 => "resolutionA1",
    17 => "offsetA0",
    18 => "unit",
    19 => "remarks",
    20 => "packageId",
    21 => "wordNo",
    22 => "digitalStatus",
    23 => "condSrc",
    24 => "condSts",
    25 => "gcoMnemonic",
    26 => "pidScope",
    27 => "authenticationStage",
    28 => "lutRef",
    29 => "qualificationLimit",
    30 => "storageLimit",
    31 => "pidAddress",
    32 => "pt",
    33 => "descUpdate",
)

const SKIP_SHEETS = Set(["Information", "Spacecraft Information"])
const LUT_PREFIX = "LUT_TBL_INFO_"

# Type code mapping
const TYPE_MAP = Dict("A" => "ANALOG", "B" => "BINARY", "D" => "DECIMAL")

"""
    _safe_string(val) -> String

Convert any cell value to a clean string.
"""
function _safe_string(val)::String
    val === nothing && return ""
    val isa Missing && return ""
    val isa AbstractFloat && return isinteger(val) ? string(Int(val)) : string(val)
    return strip(string(val))
end

"""
    _try_parse_float(val) -> Union{Float64, String}

Try to parse a value as Float64, return original string if not numeric.
"""
function _try_parse_float(val)
    val === nothing && return ""
    val isa Missing && return ""
    val isa Number && return Float64(val)
    s = strip(string(val))
    isempty(s) && return ""
    # Handle comma-separated values like "32.000,32.000" - take first value
    if occursin(",", s)
        s = strip(first(split(s, ",")))
    end
    v = tryparse(Float64, s)
    return v !== nothing ? v : s
end

"""
    _parse_range(val) -> Union{Vector, String}

Parse range values into arrays. Handles formats like "-20 to 30", "-20~30", numeric pairs.
"""
function _parse_range(val)
    val === nothing && return ""
    val isa Missing && return ""
    s = strip(string(val))
    isempty(s) && return ""

    # Try "min to max" or "min~max" or "min - max" or "min:max" patterns
    for sep in [" to ", "~", " - ", ":"]
        if occursin(sep, s)
            parts = split(s, sep)
            if length(parts) == 2
                lo = tryparse(Float64, strip(parts[1]))
                hi = tryparse(Float64, strip(parts[2]))
                if lo !== nothing && hi !== nothing
                    return [lo, hi]
                end
            end
        end
    end

    return s
end

"""
    _parse_limit(val) -> Union{Vector, String}

Parse qualification/storage limit values into [min, max] arrays.
"""
function _parse_limit(val)
    return _parse_range(val)
end

"""
    _parse_digital_range(digital_status::String) -> Vector{String}

Extract status labels from digitalStatus string like "0:PRESENT;1:ABSENT;" -> ["PRESENT", "ABSENT"].
Removes numeric prefixes (0:, 1:, etc.).
"""
function _parse_digital_range(digital_status::String)::Vector{String}
    labels = String[]
    for part in split(digital_status, ";")
        s = strip(part)
        isempty(s) && continue
        # Remove numeric prefix like "0:", "1:", "2:", etc.
        idx = findfirst(':', s)
        if idx !== nothing
            label = strip(s[idx+1:end])
            !isempty(label) && push!(labels, label)
        else
            push!(labels, s)
        end
    end
    return labels
end

"""
    _map_type_code(val) -> String

Map Excel type codes to readable types: A->ANALOG, B->BINARY, D->DECIMAL.
"""
function _map_type_code(val)::String
    s = _safe_string(val)
    return get(TYPE_MAP, uppercase(s), s)
end

"""
    parse_xlsx_file(filepath::AbstractString) -> Vector{Dict{String,Any}}

Parse a .xlsx TM mnemonics file exported from Adminel API.
Returns a vector of record dicts with proper types.
Skips Information, Spacecraft Information, and LUT_TBL_INFO_* sheets.
"""
function parse_xlsx_file(filepath::AbstractString)::Vector{Dict{String,Any}}
    xf = XLSX.readxlsx(filepath)
    source_file = basename(filepath)
    records = Dict{String,Any}[]

    for sheet_name in XLSX.sheetnames(xf)
        # Skip non-subsystem sheets
        sheet_name in SKIP_SHEETS && continue
        startswith(sheet_name, LUT_PREFIX) && continue

        sheet = xf[sheet_name]
        data = XLSX.getdata(sheet)
        nrows = size(data, 1)
        ncols = size(data, 2)

        nrows < 2 && continue  # header only or empty

        # Row 1 = header, data starts at row 2
        for row_idx in 2:nrows
            # Read cdbPidNo (column 3) first to skip empty rows
            pid_val = ncols >= 3 ? _safe_string(data[row_idx, 3]) : ""
            isempty(pid_val) && continue

            record = Dict{String,Any}()

            for (col_idx, field_name) in XLSX_COLUMN_MAP
                col_idx > ncols && continue
                raw = data[row_idx, col_idx]

                # Apply type-specific conversions
                if field_name == "type"
                    record[field_name] = _map_type_code(raw)
                elseif field_name in ("samplingRate", "resolutionA1", "offsetA0")
                    record[field_name] = _try_parse_float(raw)
                elseif field_name == "range"
                    record[field_name] = _parse_range(raw)
                elseif field_name in ("qualificationLimit", "storageLimit")
                    record[field_name] = _parse_limit(raw)
                else
                    record[field_name] = _safe_string(raw)
                end
            end

            record["sourceSheet"] = sheet_name
            record["sourceFile"] = source_file

            # For BINARY type: derive range from digitalStatus if range is empty
            rec_type = get(record, "type", "")
            rec_range = get(record, "range", "")
            rec_ds = get(record, "digitalStatus", "")
            if rec_type == "BINARY" && (rec_range == "" || rec_range isa AbstractString && isempty(strip(rec_range))) && !isempty(rec_ds)
                record["range"] = _parse_digital_range(rec_ds)
            end

            push!(records, record)
        end
    end

    @info "Parsed $(length(records)) TM PID records from $source_file"
    return records
end

# =============================================
# .out File Parser (Parameter Table Definition)
# =============================================

const OUT_TYPE_MAP = Dict{String,String}(
    "STATUS"     => "BINARY",
    "EUCN"       => "ANALOG",
    "EUCN-16B"   => "ANALOG",
    "TMCD-16B"   => "ANALOG",
    "TMCD"       => "ANALOG",
    "TMCH-16B"   => "ANALOG",
    "TMCH"       => "ANALOG",
    "AEXP"       => "ANALOG",
    "LKPTBL"     => "ANALOG",
    "EUCN-HRMIN" => "ANALOG",
    "EUCN-1750-" => "ANALOG",
)

"""
    parse_out_file(filepath::AbstractString) -> Vector{Dict{String,Any}}

Parse a mainframe Parameter Table Definition .out file.
Returns records in the same format as parse_xlsx_file for use with upsert_tm_pids_bulk.
"""
function parse_out_file(filepath::AbstractString)::Vector{Dict{String,Any}}
    source_file = basename(filepath)
    lines = readlines(filepath)
    records = Dict{String,Any}[]

    # Skip header lines (first 7) and group into primary + continuation
    groups = Vector{Vector{String}}()
    for i in 8:length(lines)
        line = lines[i]
        isempty(strip(line)) && continue

        if occursin(r"^[A-Z]{3}\d{5}", line)
            # Primary row — start new group
            push!(groups, [line])
        elseif !isempty(groups)
            # Continuation row — append to current group
            push!(groups[end], line)
        end
    end

    for group in groups
        rec = _parse_out_group(group, source_file)
        if rec !== nothing
            push!(records, rec)
        end
    end

    @info "Parsed $(length(records)) TM PID records from .out file: $source_file"
    return records
end

"""Parse a group of lines (primary + continuations) into a tm_mnemonics record."""
function _parse_out_group(group::Vector{String}, source_file::String)::Union{Dict{String,Any}, Nothing}
    tokens = split(group[1])
    length(tokens) < 4 && return nothing

    cdbPidNo = String(tokens[1])
    cdbMnemonic = String(tokens[2])
    processingType = String(tokens[3])
    subsystem = String(tokens[4])

    mapped_type = get(OUT_TYPE_MAP, processingType, "ANALOG")

    # Find calibration anchor: "SIMPLE" or "NONE"
    cal_idx = findfirst(t -> t in ("SIMPLE", "NONE"), tokens)

    highLimit = ""
    lowLimit = ""
    tolerance = ""
    unit_val = ""
    resolutionA1 = ""
    offsetA0 = ""
    lutRef = ""
    description = ""
    digital_states = String[]

    if cal_idx !== nothing
        cal_type = String(tokens[cal_idx])
        rest = tokens[cal_idx+1:end]

        if cal_type == "SIMPLE" && length(rest) >= 4
            highLimit = _try_parse_float_out(rest[1])
            lowLimit = _try_parse_float_out(rest[2])
            tolerance = _try_parse_float_out(rest[3])
            unit_val = String(rest[4])
            after = rest[5:end]
        elseif cal_type == "NONE" && length(rest) >= 2
            tolerance = _try_parse_float_out(rest[1])
            unit_val = String(rest[2])
            after = rest[3:end]
        else
            after = SubString{String}[]
        end

        # Parse remaining tokens based on type
        if processingType == "STATUS"
            for t in after
                s = String(t)
                if occursin(r"^\d{2}:", s) && !startswith(s, "%")
                    push!(digital_states, s)
                end
            end
        elseif processingType == "AEXP"
            # Expression — rejoin remaining tokens (skip format strings)
            expr_parts = [String(t) for t in after if !startswith(t, "%")]
            description = join(expr_parts, " ")
        elseif processingType == "LKPTBL"
            # LUT reference
            lut_parts = [String(t) for t in after if !startswith(t, "%") && !occursin(r"^\d+\.\d+$", String(t))]
            if !isempty(lut_parts)
                lutRef = lut_parts[end]
            end
        else
            # EUCN, EUCN-16B, TMCD-16B, TMCH-16B: resolutionA1, offsetA0
            numeric_after = [String(t) for t in after if !startswith(t, "%")]
            if length(numeric_after) >= 2
                resolutionA1 = _try_parse_float_out(numeric_after[1])
                offsetA0 = _try_parse_float_out(numeric_after[2])
            elseif length(numeric_after) == 1
                resolutionA1 = _try_parse_float_out(numeric_after[1])
            end
        end
    end

    # Parse continuation lines for additional digital states
    for i in 2:length(group)
        for t in split(group[i])
            s = String(t)
            if occursin(r"^\d{2}:", s) && !startswith(s, "%")
                push!(digital_states, s)
            end
        end
    end

    # Build range and digitalStatus
    if mapped_type == "BINARY" && !isempty(digital_states)
        # Extract labels from "00:PRESENT" format
        labels = [split(s, ":"; limit=2)[2] for s in digital_states]
        range_val = labels
        digitalStatus = join(digital_states, ";")
    elseif mapped_type == "ANALOG"
        hl = highLimit isa Float64 ? highLimit : tryparse(Float64, string(highLimit))
        ll = lowLimit isa Float64 ? lowLimit : tryparse(Float64, string(lowLimit))
        if hl !== nothing && ll !== nothing
            range_val = [ll, hl]
        else
            range_val = ""
        end
        digitalStatus = ""
    else
        range_val = ""
        digitalStatus = ""
    end

    return Dict{String,Any}(
        "cdbPidNo"       => cdbPidNo,
        "cdbMnemonic"    => cdbMnemonic,
        "type"           => mapped_type,
        "processingType" => processingType,
        "subsystem"      => subsystem,
        "sourceSheet"    => subsystem,
        "sourceFile"     => source_file,
        "range"          => range_val,
        "digitalStatus"  => digitalStatus,
        "tolerance"      => tolerance,
        "resolutionA1"   => resolutionA1,
        "offsetA0"       => offsetA0,
        "unit"           => unit_val,
        "lutRef"         => lutRef,
        "description"    => description,
    )
end

"""Try to parse a token as Float64, return the float or the original string."""
function _try_parse_float_out(s)
    val = tryparse(Float64, String(s))
    return val !== nothing ? val : String(s)
end

"""
    upsert_tm_pid(record::Dict{String,Any}; source_file::String="") -> Symbol

Upsert a single TM PID record into tm_mnemonics.
Returns :inserted, :updated, or :skipped.
Tracks mnemonic and digitalStatus changes in tm_mnemonics_change_history.
"""
function upsert_tm_pid(record::Dict{String,Any}; source_file::String="")::Symbol
    collection = db()["tm_mnemonics"]
    history_collection = db()["tm_mnemonics_change_history"]

    pid = get(record, "cdbPidNo", "")
    isempty(pid) && return :skipped

    now_ts = _iso_now()

    # Build the document matching the user's schema
    doc = Dict{String,Any}(
        "subsystem"           => get(record, "sourceSheet", get(record, "subsystem", "")),
        "cdbPidNo"            => pid,
        "cdbMnemonic"         => get(record, "cdbMnemonic", ""),
        "type"                => get(record, "type", ""),
        "processingType"      => get(record, "processingType", ""),
        "samplingRate"        => get(record, "samplingRate", ""),
        "dwellAddress"        => get(record, "dwellAddress", ""),
        "pidAddress"          => get(record, "pidAddress", ""),
        "pt"                  => get(record, "pt", ""),
        "range"               => get(record, "range", ""),
        "resolutionA1"        => get(record, "resolutionA1", ""),
        "offsetA0"            => get(record, "offsetA0", ""),
        "tolerance"           => get(record, "tolerance", ""),
        "unit"                => get(record, "unit", ""),
        "digitalStatus"       => get(record, "digitalStatus", ""),
        "condSrc"             => get(record, "condSrc", ""),
        "condSts"             => get(record, "condSts", ""),
        "gcoMnemonic"         => get(record, "gcoMnemonic", ""),
        "pidScope"            => get(record, "pidScope", ""),
        "lutRef"              => get(record, "lutRef", ""),
        "qualificationLimit"  => get(record, "qualificationLimit", ""),
        "storageLimit"        => get(record, "storageLimit", ""),
        "description"         => get(record, "description", ""),
        "descUpdate"          => get(record, "descUpdate", ""),
        "sourceSheet"         => get(record, "sourceSheet", ""),
        "sourceFile"          => isempty(source_file) ? get(record, "sourceFile", "") : source_file,
    )

    # Check if document already exists
    existing = Mongoc.find_one(collection, Mongoc.BSON("_id" => pid))

    if existing === nothing
        # First insert
        doc["_id"] = pid
        doc["createdAt"] = now_ts
        bson_doc = Mongoc.BSON(JSON3.write(doc))
        Mongoc.insert_one(collection, bson_doc)
        return :inserted
    end

    # Detect changes in tracked fields
    changes = Dict{String,Any}()

    old_mnemonic = get(existing, "cdbMnemonic", "")
    new_mnemonic = doc["cdbMnemonic"]
    if old_mnemonic != new_mnemonic && !isempty(old_mnemonic)
        changes["mnemonic"] = Dict(old_mnemonic => new_mnemonic)
    end

    old_ds = get(existing, "digitalStatus", "")
    new_ds = doc["digitalStatus"]
    if old_ds != new_ds && !isempty(old_ds)
        changes["digital_tm"] = Dict(old_ds => new_ds)
    end

    # Update the document (preserve createdAt)
    set_payload = copy(doc)
    set_payload["updatedAt"] = now_ts
    update_op = Mongoc.BSON("\$set" => Mongoc.BSON(JSON3.write(set_payload)))
    Mongoc.update_one(collection, Mongoc.BSON("_id" => pid), update_op)

    # Record change history if any tracked fields changed
    if !isempty(changes)
        change_entry = Dict{String,Any}(
            "timestamp" => now_ts,
            "changes"   => changes,
        )
        existing_hist = Mongoc.find_one(history_collection, Mongoc.BSON("_id" => pid))
        if existing_hist === nothing
            hist_doc = Dict{String,Any}(
                "_id"     => pid,
                "history" => [change_entry],
            )
            Mongoc.insert_one(history_collection, Mongoc.BSON(JSON3.write(hist_doc)))
        else
            push_op = Mongoc.BSON(
                "\$push" => Mongoc.BSON("history" => Mongoc.BSON(JSON3.write(change_entry)))
            )
            Mongoc.update_one(history_collection, Mongoc.BSON("_id" => pid), push_op)
        end
    end

    return :updated
end

"""
    upsert_tm_pids_bulk(records::Vector{Dict{String,Any}}; source_file::String="") -> Dict

Bulk upsert TM PID records. Returns stats: {total, inserted, updated, skipped, errors}.
"""
function upsert_tm_pids_bulk(records::Vector{Dict{String,Any}}; source_file::String="")::Dict{String,Any}
    inserted = 0
    updated  = 0
    skipped  = 0
    errors   = String[]

    for record in records
        try
            result = upsert_tm_pid(record; source_file=source_file)
            if result == :inserted
                inserted += 1
            elseif result == :updated
                updated += 1
            else
                skipped += 1
            end
        catch e
            pid = get(record, "cdbPidNo", "UNKNOWN")
            push!(errors, "[$pid] $(sprint(showerror, e))")
        end
    end

    @info "TM bulk upsert: total=$(length(records)) inserted=$inserted updated=$updated skipped=$skipped errors=$(length(errors))"
    return Dict{String,Any}(
        "total"    => length(records),
        "inserted" => inserted,
        "updated"  => updated,
        "skipped"  => skipped,
        "errors"   => errors,
    )
end

"""
    get_tm_mnemonics(; subsystem=nothing) -> Vector{Dict}

Get TM mnemonics from the enriched tm_mnemonics collection.
Optionally filtered by subsystem (Excel sheet name).
"""
function get_tm_mnemonics(; subsystem=nothing)
    collection = db()["tm_mnemonics"]
    filter = subsystem !== nothing ? Mongoc.BSON("subsystem" => subsystem) : Mongoc.BSON()
    results = Mongoc.find(collection, filter)
    mnemonics = Dict[]
    for doc in results
        push!(mnemonics, Dict{String,Any}(
            "_id"                => string(doc["_id"]),
            "subsystem"          => get(doc, "subsystem", ""),
            "cdbPidNo"           => get(doc, "cdbPidNo", ""),
            "cdbMnemonic"        => get(doc, "cdbMnemonic", ""),
            "type"               => get(doc, "type", ""),
            "processingType"     => get(doc, "processingType", ""),
            "samplingRate"       => get(doc, "samplingRate", ""),
            "dwellAddress"       => get(doc, "dwellAddress", ""),
            "pidAddress"         => get(doc, "pidAddress", ""),
            "pt"                 => get(doc, "pt", ""),
            "range"              => get(doc, "range", ""),
            "resolutionA1"       => get(doc, "resolutionA1", ""),
            "offsetA0"           => get(doc, "offsetA0", ""),
            "tolerance"          => get(doc, "tolerance", ""),
            "unit"               => get(doc, "unit", ""),
            "digitalStatus"      => get(doc, "digitalStatus", ""),
            "condSrc"            => get(doc, "condSrc", ""),
            "condSts"            => get(doc, "condSts", ""),
            "gcoMnemonic"        => get(doc, "gcoMnemonic", ""),
            "pidScope"           => get(doc, "pidScope", ""),
            "lutRef"             => get(doc, "lutRef", ""),
            "qualificationLimit" => get(doc, "qualificationLimit", ""),
            "storageLimit"       => get(doc, "storageLimit", ""),
            "description"        => get(doc, "description", ""),
            "descUpdate"         => get(doc, "descUpdate", ""),
            "sourceSheet"        => get(doc, "sourceSheet", ""),
            "sourceFile"         => get(doc, "sourceFile", ""),
            "createdAt"          => get(doc, "createdAt", ""),
        ))
    end
    return mnemonics
end

"""
    get_tm_mnemonics_catalog(; subsystem=nothing, limit=nothing, offset=nothing) -> Vector{Dict}

Query tm_mnemonics with pagination support.
"""
function get_tm_mnemonics_catalog(; subsystem=nothing, limit=nothing, offset=nothing)::Vector{Dict}
    collection = db()["tm_mnemonics"]
    query = subsystem !== nothing ? Mongoc.BSON("subsystem" => subsystem) : Mongoc.BSON()

    use_opts = (limit !== nothing && limit > 0) || (offset !== nothing && offset > 0)
    opts = use_opts ? Mongoc.BSON() : nothing
    if use_opts
        if limit !== nothing && limit > 0
            opts["limit"] = limit
        end
        if offset !== nothing && offset > 0
            opts["skip"] = offset
        end
    end

    results = use_opts ?
        Mongoc.find(collection, query; options=opts) :
        Mongoc.find(collection, query)

    records = Dict[]
    for doc in results
        push!(records, Dict{String,Any}(pairs(doc)))
    end
    return records
end

"""
    init_tm_mnemonics_indexes()

Create indexes for tm_mnemonics collection.
"""
function init_tm_mnemonics_indexes()
    catalog = db()["tm_mnemonics"]
    try
        Mongoc.create_index(catalog, Mongoc.BSON("subsystem" => 1))
        @info "Created index on tm_mnemonics: subsystem"
    catch e
        @warn "tm_mnemonics subsystem index may already exist: $e"
    end
    try
        Mongoc.create_index(catalog, Mongoc.BSON("cdbMnemonic" => 1))
        @info "Created index on tm_mnemonics: cdbMnemonic"
    catch e
        @warn "tm_mnemonics cdbMnemonic index may already exist: $e"
    end
end

"""
    get_tm_subsystems() -> Vector{String}

Get distinct subsystem names from tm_mnemonics collection, sorted alphabetically.
"""
function get_tm_subsystems()::Vector{String}
    collection = db()["tm_mnemonics"]
    opts = Mongoc.BSON("projection" => Mongoc.BSON("subsystem" => 1, "_id" => 0))
    results = Mongoc.find(collection, Mongoc.BSON(); options=opts)
    seen = Set{String}()
    for doc in results
        s = get(doc, "subsystem", "")
        if s isa AbstractString && !isempty(s)
            push!(seen, s)
        end
    end
    return sort(collect(seen))
end

# =============================================
# TC Mnemonics
# =============================================

"""
    get_tc_mnemonics() -> Vector{Dict}

Get all TC mnemonics.
"""
function get_tc_mnemonics()
    collection = db()["tc_mnemonics"]
    results = Mongoc.find(collection)
    mnemonics = Dict[]
    for doc in results
        push!(mnemonics, Dict(
            "command" => doc["command"],
            "full_ref" => doc["full_ref"],
            "description" => get(doc, "description", ""),
            "parameters" => get(doc, "parameters", []),
            "subsystem" => get(doc, "subsystem", ""),
            "category" => get(doc, "category", ""),
        ))
    end
    return mnemonics
end

# =============================================
# SCO Commands
# =============================================

"""
    get_sco_commands() -> Vector{Dict}

Get all SCO commands.
"""
function get_sco_commands()
    collection = db()["sco_commands"]
    results = Mongoc.find(collection)
    commands = Dict[]
    for doc in results
        push!(commands, Dict(
            "command" => doc["command"],
            "full_ref" => doc["full_ref"],
            "description" => get(doc, "description", ""),
            "subsystem" => get(doc, "subsystem", ""),
            "category" => get(doc, "category", ""),
        ))
    end
    return commands
end

# =============================================
# All autocomplete refs
# =============================================

"""
    get_all_autocomplete_refs() -> Dict

Get all TM + TC + SCO refs for autocomplete.
"""
function get_all_autocomplete_refs()
    return Dict(
        "tm" => get_tm_mnemonics(),
        "tc" => get_tc_mnemonics(),
        "sco" => get_sco_commands(),
    )
end

# =============================================
# Test Results
# =============================================

"""
    save_test_result(result; test_phase::String="") -> Nothing

Save a test result to MongoDB. Optionally records the test phase.
"""
function save_test_result(result; test_phase::String="")
    collection = db()["test_results"]
    doc_data = Dict{String,Any}(
        "test_name" => result.test_name,
        "status" => string(result.status),
        "duration_seconds" => result.duration_seconds,
        "started_at" => string(now() - Second(round(Int, result.duration_seconds))),
        "completed_at" => string(now()),
        "mode" => get(ENV, "ASTRA_MODE", "simulation"),
        "test_phase" => test_phase,
        "log_entries" => [Dict{String,Any}(
            "timestamp" => string(entry.timestamp),
            "line_number" => entry.line_number,
            "statement" => entry.statement,
            "result" => entry.result,
            "status" => string(entry.status),
        ) for entry in result.log],
        "error" => hasproperty(result, :error) ? result.error : nothing,
        "variables_snapshot" => Dict{String,Any}(),
    )
    doc = Mongoc.BSON(JSON3.write(doc_data))
    Mongoc.insert_one(collection, doc)
    @info "Saved test result: $(result.test_name) - $(result.status) [phase: $test_phase]"
end

"""
    get_test_results(; limit=50) -> Vector{Dict}

Get recent test results.
"""
function get_test_results(; limit::Int=50)
    collection = db()["test_results"]
    options = Mongoc.BSON(
        "sort" => Mongoc.BSON("started_at" => -1),
        "limit" => limit,
    )
    results = Mongoc.find(collection, Mongoc.BSON(); options=options)
    test_results = Dict[]
    for doc in results
        push!(test_results, Dict(
            "test_name" => doc["test_name"],
            "status" => doc["status"],
            "duration_seconds" => get(doc, "duration_seconds", 0.0),
            "started_at" => get(doc, "started_at", ""),
            "completed_at" => get(doc, "completed_at", ""),
            "mode" => get(doc, "mode", "simulation"),
            "error" => get(doc, "error", nothing),
        ))
    end
    return test_results
end

"""
    get_test_result(id::String) -> Union{Dict, Nothing}

Get a single test result by ID.
"""
function get_test_result(id::String)
    collection = db()["test_results"]
    oid = Mongoc.BSONObjectId(id)
    result = Mongoc.find_one(collection, Mongoc.BSON("_id" => oid))
    if result === nothing
        return nothing
    end
    return Dict(pairs(result))
end

# =============================================
# TM History
# =============================================

"""
    save_tm_snapshot(banks::Dict)

Save a TM snapshot to MongoDB.
"""
function save_tm_snapshot(banks::Dict)
    collection = db()["tm_history"]
    doc = Mongoc.BSON(
        "timestamp" => string(now()),
        "banks" => banks,
    )
    Mongoc.insert_one(collection, doc)
end

"""
    get_tm_history(; limit=100) -> Vector{Dict}

Get TM history snapshots.
"""
function get_tm_history(; limit::Int=100)
    collection = db()["tm_history"]
    options = Mongoc.BSON(
        "sort" => Mongoc.BSON("timestamp" => -1),
        "limit" => limit,
    )
    results = Mongoc.find(collection, Mongoc.BSON(); options=options)
    history = Dict[]
    for doc in results
        push!(history, Dict(pairs(doc)))
    end
    return history
end

# =============================================
# User Defined Telemetry (UD_TM)
# =============================================

"""
    init_user_telemetry_indexes()

Create indexes on user_telemetry and user_telemetry_versions collections.
"""
function init_user_telemetry_indexes()
    Mongoc.write_command(db(), Mongoc.BSON(
        "createIndexes" => "user_telemetry",
        "indexes" => [
            Mongoc.BSON("key" => Mongoc.BSON("project" => 1, "name" => 1), "name" => "project_name_unique", "unique" => true),
        ],
    ))
    Mongoc.write_command(db(), Mongoc.BSON(
        "createIndexes" => "user_telemetry_versions",
        "indexes" => [
            Mongoc.BSON("key" => Mongoc.BSON("ud_tm_id" => 1, "version" => -1), "name" => "ud_tm_version_idx"),
        ],
    ))
    @info "User telemetry indexes created"
end

"""
    get_user_telemetry(project) -> Union{Dict, Nothing}

Get the single user telemetry document for a project.
"""
function get_user_telemetry(project::AbstractString)::Union{Dict, Nothing}
    collection = db()["user_telemetry"]
    doc = Mongoc.find_one(collection, Mongoc.BSON("project" => project))
    if doc === nothing
        return nothing
    end
    rows_raw = get(doc, "rows", [])
    rows = [Dict{String,Any}(
        "row_number" => get(r, "row_number", i),
        "mnemonic" => get(r, "mnemonic", ""),
        "value" => get(r, "value", ""),
        "range" => get(r, "range", ""),
        "limit" => get(r, "limit", ""),
        "tolerance" => get(r, "tolerance", ""),
    ) for (i, r) in enumerate(rows_raw)]
    return Dict{String,Any}(
        "_id" => string(doc["_id"]),
        "project" => get(doc, "project", ""),
        "latest_version" => get(doc, "latest_version", 1),
        "rows" => rows,
        "created_by" => get(doc, "created_by", ""),
        "updated_by" => get(doc, "updated_by", ""),
        "created_at" => get(doc, "created_at", ""),
        "updated_at" => get(doc, "updated_at", ""),
    )
end

"""
    _is_digital_range(range_str) -> Bool

Check if a range string contains digital states.
Supports formats: "A,B,C,D" (comma-separated) or "0:OFF;1:ON" (legacy index:label).
A range is digital if it contains commas or semicolons with non-numeric labels.
A pure numeric range like "0:100" or "0.0:50.5" is NOT digital.
"""
function _is_digital_range(range_str::AbstractString)::Bool
    s = strip(range_str)
    isempty(s) && return false
    # Comma-separated states: "A,B,C,D"
    if occursin(",", s)
        parts = strip.(split(s, ","))
        # If all parts are pure numbers, it's not digital
        all_numeric = all(p -> tryparse(Float64, p) !== nothing, parts)
        return !all_numeric
    end
    # Legacy format: "0:OFF;1:ON"
    if occursin(";", s) && occursin(":", s)
        return true
    end
    return false
end

"""
    _parse_digital_states(range_str) -> Vector{String}

Parse digital state labels from range string.
"A,B,C,D" -> ["A","B","C","D"]
"0:OFF;1:ON" -> ["OFF","ON"]
"""
function _parse_digital_states(range_str::AbstractString)::Vector{String}
    s = strip(range_str)
    if occursin(",", s)
        return [strip(p) for p in split(s, ",") if !isempty(strip(p))]
    end
    if occursin(";", s)
        # Legacy "0:OFF;1:ON" format - extract labels
        parts = split(s, ";")
        labels = String[]
        for p in parts
            p = strip(p)
            isempty(p) && continue
            if occursin(":", p)
                push!(labels, strip(split(p, ":"; limit=2)[2]))
            else
                push!(labels, p)
            end
        end
        return labels
    end
    return [s]
end

"""
    _parse_numeric_range(range_str) -> Union{Vector{Float64}, Nothing}

Try to parse range as "min:max" numeric format. Returns [min, max] or nothing.
"""
function _parse_numeric_range(range_str::AbstractString)::Union{Vector{Float64}, Nothing}
    m = match(r"^([\d.+-]+)\s*:\s*([\d.+-]+)$", range_str)
    if m === nothing
        return nothing
    end
    min_val = tryparse(Float64, m.captures[1])
    max_val = tryparse(Float64, m.captures[2])
    if min_val !== nothing && max_val !== nothing
        return [min_val, max_val]
    end
    return nothing
end

"""
    _sync_udtm_to_tm_mnemonics(rows, project)

Sync UDTM rows to tm_mnemonics collection (subsystem="UDTM").
Upserts current mnemonics and removes deleted ones.
"""
function _sync_udtm_to_tm_mnemonics(rows::Vector{Dict{String,Any}}, project::AbstractString)
    tm_coll = db()["tm_mnemonics"]
    now_time = _iso_now()

    # Get current UDTM mnemonics in tm_mnemonics
    existing = Mongoc.find(tm_coll, Mongoc.BSON("subsystem" => "UDTM"))
    old_mnemonics = Set{String}()
    for doc in existing
        push!(old_mnemonics, get(doc, "cdbMnemonic", ""))
    end

    # Upsert each row into tm_mnemonics
    new_mnemonics = Set{String}()
    for row in rows
        mnemonic = get(row, "mnemonic", "")
        if isempty(mnemonic)
            continue
        end
        push!(new_mnemonics, mnemonic)

        range_str = get(row, "range", "")
        is_digital = _is_digital_range(range_str)

        if is_digital
            digital_states = _parse_digital_states(range_str)
            tm_type = "BINARY"
            tm_range = digital_states
            tm_digital = join(digital_states, ",")
        else
            numeric_range = _parse_numeric_range(range_str)
            tm_type = "ANALOG"
            tm_range = numeric_range !== nothing ? numeric_range : []
            tm_digital = ""
        end

        tm_doc = Dict{String,Any}(
            "cdbPidNo" => "UDTM_$(mnemonic)",
            "cdbMnemonic" => mnemonic,
            "subsystem" => "UDTM",
            "sourceSheet" => "UDTM",
            "type" => tm_type,
            "description" => "User Defined TM",
            "unit" => "",
            "digitalStatus" => tm_digital,
            "range" => tm_range,
            "updatedAt" => now_time,
        )

        # Use cdbMnemonic as upsert key (with subsystem to avoid collisions)
        filter = Mongoc.BSON("cdbMnemonic" => mnemonic, "subsystem" => "UDTM")
        update = Mongoc.BSON("\$set" => Mongoc.BSON(JSON3.write(tm_doc)),
                             "\$setOnInsert" => Mongoc.BSON("createdAt" => now_time))
        Mongoc.update_one(tm_coll, filter, update; options=Mongoc.BSON("upsert" => true))
    end

    # Remove UDTM mnemonics that no longer exist
    removed = setdiff(old_mnemonics, new_mnemonics)
    for mnemonic in removed
        if !isempty(mnemonic)
            Mongoc.delete_one(tm_coll, Mongoc.BSON("cdbMnemonic" => mnemonic, "subsystem" => "UDTM"))
        end
    end

    @info "Synced $(length(new_mnemonics)) UDTM mnemonics to tm_mnemonics, removed $(length(removed))"
end

"""
    save_user_telemetry(rows, project, created_by; change_message="", changes=[])

Save user telemetry rows (single flat table per project) with version history.
Also syncs mnemonics to tm_mnemonics collection for TM.UDTM. autocompletion.
"""
function save_user_telemetry(
    rows::Vector,
    project::AbstractString,
    created_by::AbstractString;
    change_message::AbstractString = "",
    changes::Vector = []
)
    ut = db()["user_telemetry"]
    utv = db()["user_telemetry_versions"]
    now_time = _iso_now()

    # Convert rows to Vector{Dict} for JSON serialization
    rows_dicts = [Dict{String,Any}(
        "row_number" => get(r, "row_number", i),
        "mnemonic" => get(r, "mnemonic", ""),
        "value" => get(r, "value", ""),
        "range" => get(r, "range", ""),
        "limit" => get(r, "limit", ""),
        "tolerance" => get(r, "tolerance", ""),
    ) for (i, r) in enumerate(rows)]

    changes_dicts = [Dict{String,Any}(pairs(c)) for c in changes]

    existing = Mongoc.find_one(ut, Mongoc.BSON("project" => project))

    if existing !== nothing
        ud_tm_id = string(existing["_id"])
        current_version = get(existing, "latest_version", 0)
        next_version = current_version + 1

        set_fields = Dict{String,Any}(
            "latest_version" => next_version,
            "rows" => rows_dicts,
            "updated_by" => created_by,
            "updated_at" => now_time,
        )
        update_doc = Mongoc.BSON("\$set" => Mongoc.BSON(JSON3.write(set_fields)))
        Mongoc.update_one(ut, Mongoc.BSON("_id" => existing["_id"]), update_doc)
    else
        next_version = 1

        doc = Dict{String,Any}(
            "project" => project,
            "latest_version" => 1,
            "rows" => rows_dicts,
            "created_by" => created_by,
            "updated_by" => created_by,
            "created_at" => now_time,
            "updated_at" => now_time,
        )
        result = Mongoc.insert_one(ut, Mongoc.BSON(JSON3.write(doc)))
        ud_tm_id = string(result.inserted_oid)
    end

    # Insert version record
    version_doc = Dict{String,Any}(
        "ud_tm_id" => ud_tm_id,
        "version" => next_version,
        "rows" => rows_dicts,
        "created_by" => created_by,
        "created_at" => now_time,
        "change_message" => change_message,
        "changes" => changes_dicts,
    )
    Mongoc.insert_one(utv, Mongoc.BSON(JSON3.write(version_doc)))

    # Sync to tm_mnemonics for TM.UDTM. autocompletion
    try
        _sync_udtm_to_tm_mnemonics(rows_dicts, project)
    catch e
        @warn "Failed to sync UDTM to tm_mnemonics: $e"
    end

    @info "Saved user telemetry v$next_version by $created_by (project=$project)"
    return Dict{String,Any}(
        "saved" => true,
        "version" => next_version,
        "project" => project,
    )
end

"""
    get_user_telemetry_versions(ud_tm_id) -> Vector{Dict}

Get version history for a user telemetry document.
"""
function get_user_telemetry_versions(ud_tm_id::AbstractString)::Vector{Dict}
    utv = db()["user_telemetry_versions"]
    opts = Mongoc.BSON("sort" => Mongoc.BSON("version" => -1))
    results = Mongoc.find(utv, Mongoc.BSON("ud_tm_id" => ud_tm_id); options=opts)
    versions = Dict[]
    for doc in results
        push!(versions, Dict{String,Any}(
            "_id" => string(doc["_id"]),
            "ud_tm_id" => get(doc, "ud_tm_id", ""),
            "version" => get(doc, "version", 0),
            "rows" => get(doc, "rows", []),
            "created_by" => get(doc, "created_by", ""),
            "created_at" => get(doc, "created_at", ""),
            "change_message" => get(doc, "change_message", ""),
            "changes" => get(doc, "changes", []),
        ))
    end
    return versions
end

"""
    get_user_telemetry_version(ud_tm_id, version) -> Union{Dict, Nothing}

Get a specific version of user telemetry.
"""
function get_user_telemetry_version(ud_tm_id::AbstractString, version::Int)::Union{Dict, Nothing}
    utv = db()["user_telemetry_versions"]
    doc = Mongoc.find_one(utv, Mongoc.BSON("ud_tm_id" => ud_tm_id, "version" => version))
    if doc === nothing
        return nothing
    end
    return Dict{String,Any}(
        "_id" => string(doc["_id"]),
        "ud_tm_id" => get(doc, "ud_tm_id", ""),
        "version" => get(doc, "version", 0),
        "rows" => get(doc, "rows", []),
        "created_by" => get(doc, "created_by", ""),
        "created_at" => get(doc, "created_at", ""),
        "change_message" => get(doc, "change_message", ""),
        "changes" => get(doc, "changes", []),
    )
end

# =============================================
# Background Schedules
# =============================================
# Collection: background_schedules
# Each document represents one registered background procedure.

"""
    init_background_schedules_indexes()

Create indexes for the background_schedules collection.
"""
function init_background_schedules_indexes()
    coll = db()["background_schedules"]
    Mongoc.create_index(coll, Mongoc.BSON("""{"proc_name": 1}"""), Mongoc.BSON("""{"unique": true}"""))
    @info "MongoStore: background_schedules indexes created"
end

"""
    save_background_schedule(proc_name, schedule) -> Bool

Upsert a background schedule document. schedule is a BackgroundSchedule struct
(passed as a Dict from BackgroundScheduler for decoupling).
"""
function save_background_schedule(proc_name::String, schedule)
    coll = db()["background_schedules"]

    # Build schedule dict without depending on BackgroundScheduler types
    sched_type = string(typeof(schedule))
    sched_doc = if occursin("IntervalSchedule", sched_type)
        Dict{String,Any}(
            "schedule_type"            => "interval",
            "interval_seconds"         => schedule.interval_seconds,
            "restart_on_failure"       => schedule.restart_on_failure,
            "max_consecutive_failures" => schedule.max_consecutive_failures,
        )
    else
        Dict{String,Any}(
            "schedule_type"            => "event",
            "condition"                => schedule.condition,
            "poll_interval"            => schedule.poll_interval,
            "restart_on_failure"       => schedule.restart_on_failure,
            "max_consecutive_failures" => schedule.max_consecutive_failures,
        )
    end

    doc_data = merge(sched_doc, Dict{String,Any}(
        "proc_name"  => proc_name,
        "enabled"    => true,
        "updated_at" => _iso_now(),
    ))

    filter_bson = Mongoc.BSON(JSON3.write(Dict("proc_name" => proc_name)))
    update_bson = Mongoc.BSON(JSON3.write(Dict("\$set" => doc_data, "\$setOnInsert" => Dict("created_at" => _iso_now()))))
    Mongoc.update_one(coll, filter_bson, update_bson; options=Mongoc.BSON("""{"upsert": true}"""))
    return true
end

"""
    delete_background_schedule(proc_name) -> Bool

Remove a background schedule document (disable + remove from MongoDB).
"""
function delete_background_schedule(proc_name::String)::Bool
    coll = db()["background_schedules"]
    filter_bson = Mongoc.BSON(JSON3.write(Dict("proc_name" => proc_name)))
    result = Mongoc.delete_one(coll, filter_bson)
    return result.reply["deletedCount"] > 0
end

"""
    list_background_schedules(; enabled_only=true) -> Vector{Dict}

Return all background schedule documents.
"""
function list_background_schedules(; enabled_only::Bool=true)::Vector{Dict}
    coll = db()["background_schedules"]
    filter = enabled_only ? Dict("enabled" => true) : Dict()
    docs = collect(Mongoc.find(coll, Mongoc.BSON(JSON3.write(filter))))
    return [Dict{String,Any}(
        "proc_name"                => get(d, "proc_name", ""),
        "schedule_type"            => get(d, "schedule_type", "interval"),
        "interval_seconds"         => get(d, "interval_seconds", 1.0),
        "condition"                => get(d, "condition", ""),
        "poll_interval"            => get(d, "poll_interval", 0.5),
        "restart_on_failure"       => get(d, "restart_on_failure", true),
        "max_consecutive_failures" => get(d, "max_consecutive_failures", 10),
        "enabled"                  => get(d, "enabled", true),
        "created_at"               => get(d, "created_at", ""),
        "updated_at"               => get(d, "updated_at", ""),
    ) for d in docs]
end

end # module MongoStore
