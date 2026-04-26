"""
    ACSParser

DSL Parser for ASTRA (Automated Satellite Test & Reporting Application).
Reads .tst files and produces intermediate representation with line tracking.
"""
module ACSParser

export load_file, load_directory, load_from_string, ParsedProcedure, ParsedLine
export get_procedure, list_procedures, clear_registry
export DSL_KEYWORDS

using Dates

# Known DSL keywords (extensible)
const DSL_KEYWORDS = Set([
    "TEST_NAME", "PRE_TEST_REQ", "SEND", "SENDTCP",
    "WAIT", "CHECK", "EXPECTED", "ALERT_MSG",
    "ABORT_TEST", "CALL", "BREAK",
    "IF", "ELSE", "END", "FOR", "IN", "TO", "WHILE",
    "ON_FAIL", "ON_TIMEOUT", "UNTIL", "TIMEOUT", "WITHIN"
])

# Represents a single parsed line with metadata
struct ParsedLine
    line_number::Int        # Original .tst file line number
    raw_text::String        # Original text
    statement_type::Symbol  # :SEND, :WAIT, :IF, :JULIA_CODE, etc.
    tokens::Vector{String}  # Tokenized parts
    block_depth::Int        # Nesting depth for validation
end

# Represents a complete parsed procedure
struct ParsedProcedure
    name::String
    source_file::String
    lines::Vector{ParsedLine}
    loaded_at::DateTime
end

# Registry: TEST_NAME -> ParsedProcedure
const PROCEDURE_REGISTRY = Dict{String, ParsedProcedure}()

"""
    classify_statement(line::AbstractString) -> Symbol

Determine the statement type from the first token.
"""
function classify_statement(line::AbstractString)::Symbol
    tokens = split(strip(line))
    if isempty(tokens)
        return :BLANK
    end

    first_token = uppercase(tokens[1])

    # Check if it's a known DSL keyword
    if first_token in DSL_KEYWORDS
        return Symbol(first_token)
    end

    # Otherwise treat as inline Julia code
    return :JULIA_CODE
end

"""
    tokenize_line(line::AbstractString) -> Vector{String}

Split a line into tokens, preserving quoted strings.
"""
function tokenize_line(line::AbstractString)::Vector{String}
    tokens = String[]
    current = ""
    in_string = false

    for char in line
        if char == '"'
            in_string = !in_string
            current *= char
        elseif (char == ' ' || char == '\t') && !in_string
            if !isempty(current)
                push!(tokens, current)
                current = ""
            end
        else
            current *= char
        end
    end

    if !isempty(current)
        push!(tokens, current)
    end

    return tokens
end

"""
    load_from_string(content::String, source_name::String="<string>") -> ParsedProcedure

Parse procedure from a string.
"""
function load_from_string(content::String, source_name::String="<string>")::ParsedProcedure
    lines_arr = split(content, '\n')
    parsed_lines = ParsedLine[]
    test_name = ""
    block_depth = 0

    for (idx, line) in enumerate(lines_arr)
        line_strip = strip(line)

        # Skip blank lines and comments
        if isempty(line_strip) || startswith(line_strip, "#") || startswith(line_strip, "//")
            continue
        end

        # Extract TEST_NAME (must be first non-comment line)
        if isempty(test_name) && startswith(line_strip, "TEST_NAME")
            tokens = tokenize_line(line_strip)
            if length(tokens) >= 2
                test_name = tokens[2]
            else
                error("Invalid TEST_NAME declaration at line $idx: $line_strip")
            end
            continue
        end

        # Classify statement type
        stmt_type = classify_statement(line_strip)
        tokens = tokenize_line(line_strip)

        # Track block depth
        if stmt_type in (:IF, :FOR, :WHILE, :ON_FAIL, :ON_TIMEOUT)
            block_depth += 1
        elseif stmt_type == :END
            block_depth = max(0, block_depth - 1)
        end

        push!(parsed_lines, ParsedLine(
            idx,
            line,
            stmt_type,
            tokens,
            block_depth
        ))
    end

    if isempty(test_name)
        error("No TEST_NAME found in $source_name")
    end

    proc = ParsedProcedure(test_name, source_name, parsed_lines, now())

    # Store in registry
    PROCEDURE_REGISTRY[test_name] = proc

    return proc
end

"""
    load_file(filename::String) -> ParsedProcedure

Load and parse a .tst file.
"""
function load_file(filename::String)::ParsedProcedure
    if !isfile(filename)
        error("File not found: $filename")
    end

    content = read(filename, String)
    return load_from_string(content, filename)
end

"""
    load_directory(dir::String)

Load all .tst files from a directory.
"""
function load_directory(dir::String)
    if !isdir(dir)
        error("Directory not found: $dir")
    end

    count = 0
    for file in readdir(dir)
        if endswith(file, ".tst")
            try
                load_file(joinpath(dir, file))
                count += 1
            catch e
                @warn "Failed to load $file: $e"
            end
        end
    end

    @info "Loaded $count procedures from $dir"
end

"""
    get_procedure(name::String) -> Union{ParsedProcedure, Nothing}

Retrieve a parsed procedure by name.
"""
function get_procedure(name::String)::Union{ParsedProcedure, Nothing}
    return get(PROCEDURE_REGISTRY, name, nothing)
end

"""
    list_procedures() -> Vector{String}

List all loaded procedure names.
"""
function list_procedures()::Vector{String}
    return sort(collect(keys(PROCEDURE_REGISTRY)))
end

"""
    clear_registry()

Clear all loaded procedures.
"""
function clear_registry()
    empty!(PROCEDURE_REGISTRY)
end

end # module ACSParser
