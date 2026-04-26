"""
    ACSValidator

Static analysis and syntax validation for ASTRA procedures.
Performs pre-execution checks without side effects.
"""
module ACSValidator

export validate_procedure, ValidationError

using ..ACSParser

struct ValidationError
    file::String
    line_number::Int
    line_text::String
    message::String
    suggestion::String
    severity::Symbol  # :error, :warning, :info
end

"""
    levenshtein_distance(s1::String, s2::String) -> Int

Calculate edit distance between two strings for "Did you mean?" suggestions.
"""
function levenshtein_distance(s1::String, s2::String)::Int
    m, n = length(s1), length(s2)
    d = zeros(Int, m+1, n+1)

    for i in 0:m
        d[i+1, 1] = i
    end
    for j in 0:n
        d[1, j+1] = j
    end

    for j in 1:n
        for i in 1:m
            if s1[i] == s2[j]
                d[i+1, j+1] = d[i, j]
            else
                d[i+1, j+1] = min(d[i, j+1], d[i+1, j], d[i, j]) + 1
            end
        end
    end

    return d[m+1, n+1]
end

"""
    suggest_keyword(unknown::String) -> Union{String, Nothing}

Suggest a similar keyword for typos.
"""
function suggest_keyword(unknown::String)::Union{String, Nothing}
    best_match = nothing
    best_dist = 3  # Max edit distance threshold

    for kw in DSL_KEYWORDS
        d = levenshtein_distance(uppercase(unknown), kw)
        if d < best_dist
            best_dist = d
            best_match = kw
        end
    end

    return best_match
end

"""
    validate_block_matching!(proc::ParsedProcedure, errors::Vector{ValidationError})

Check that all IF/FOR/WHILE blocks have matching END statements.
"""
function validate_block_matching!(proc::ParsedProcedure, errors::Vector{ValidationError})
    stack = Tuple{Symbol, Int, String}[]  # (block_type, line_number, line_text)

    for line in proc.lines
        if line.statement_type in (:IF, :FOR, :WHILE)
            push!(stack, (line.statement_type, line.line_number, line.raw_text))
        elseif line.statement_type == :END
            if isempty(stack)
                push!(errors, ValidationError(
                    proc.source_file,
                    line.line_number,
                    line.raw_text,
                    "Unexpected END without matching block opener",
                    "Remove this END or add a matching IF/FOR/WHILE block",
                    :error
                ))
            else
                pop!(stack)
            end
        end
    end

    # Check for unclosed blocks
    for (block_type, line_no, line_text) in stack
        push!(errors, ValidationError(
            proc.source_file,
            line_no,
            line_text,
            "Missing END for $block_type block",
            "Add an END statement to close this $block_type block",
            :error
        ))
    end
end

"""
    validate_call_targets!(proc::ParsedProcedure, errors::Vector{ValidationError})

Check that all CALL statements reference existing procedures.
"""
function validate_call_targets!(proc::ParsedProcedure, errors::Vector{ValidationError})
    for line in proc.lines
        if line.statement_type == :CALL
            if length(line.tokens) < 2
                push!(errors, ValidationError(
                    proc.source_file,
                    line.line_number,
                    line.raw_text,
                    "CALL statement missing procedure name",
                    "Usage: CALL <procedure_name>",
                    :error
                ))
                continue
            end

            called = line.tokens[2]
            if get_procedure(called) === nothing
                push!(errors, ValidationError(
                    proc.source_file,
                    line.line_number,
                    line.raw_text,
                    "CALL references undefined procedure: $called",
                    "Make sure procedure '$called' is loaded or check for typos",
                    :error
                ))
            end
        end
    end
end

"""
    validate_break_placement!(proc::ParsedProcedure, errors::Vector{ValidationError})

Check that BREAK only appears inside FOR/WHILE loops.
"""
function validate_break_placement!(proc::ParsedProcedure, errors::Vector{ValidationError})
    in_loop = false
    loop_stack = Symbol[]

    for line in proc.lines
        if line.statement_type in (:FOR, :WHILE)
            push!(loop_stack, line.statement_type)
            in_loop = true
        elseif line.statement_type == :END && !isempty(loop_stack)
            pop!(loop_stack)
            in_loop = !isempty(loop_stack)
        elseif line.statement_type == :BREAK
            if !in_loop
                push!(errors, ValidationError(
                    proc.source_file,
                    line.line_number,
                    line.raw_text,
                    "BREAK statement outside of FOR/WHILE loop",
                    "BREAK can only be used inside FOR or WHILE blocks",
                    :error
                ))
            end
        end
    end
end

"""
    validate_tm_references!(proc::ParsedProcedure, errors::Vector{ValidationError})

Check TM reference format: TM<digits>.<name>
"""
function validate_tm_references!(proc::ParsedProcedure, errors::Vector{ValidationError})
    tm_pattern = r"\bTM\d+\.\w+"

    for line in proc.lines
        # Check for malformed TM references
        if occursin(r"\bTM[^\d]", line.raw_text) || occursin(r"\bTM\d+\.[^\w]", line.raw_text)
            push!(errors, ValidationError(
                proc.source_file,
                line.line_number,
                line.raw_text,
                "Malformed TM reference",
                "TM references must be in format: TM<number>.<mnemonic> (e.g., TM1.xyz_sts)",
                :warning
            ))
        end
    end
end

"""
    validate_julia_syntax!(proc::ParsedProcedure, errors::Vector{ValidationError})

Validate inline Julia code syntax using Meta.parse (without execution).
"""
function validate_julia_syntax!(proc::ParsedProcedure, errors::Vector{ValidationError})
    for line in proc.lines
        if line.statement_type == :JULIA_CODE
            try
                Meta.parse(line.raw_text)
            catch e
                msg = sprint(showerror, e)
                push!(errors, ValidationError(
                    proc.source_file,
                    line.line_number,
                    line.raw_text,
                    "Invalid Julia syntax: $msg",
                    "Check for missing operators, parentheses, or quotes",
                    :error
                ))
            end
        end
    end
end

"""
    validate_procedure(proc::ParsedProcedure) -> Vector{ValidationError}

Run all validation checks on a parsed procedure.
"""
function validate_procedure(proc::ParsedProcedure)::Vector{ValidationError}
    errors = ValidationError[]

    validate_block_matching!(proc, errors)
    validate_call_targets!(proc, errors)
    validate_break_placement!(proc, errors)
    validate_tm_references!(proc, errors)
    validate_julia_syntax!(proc, errors)

    return errors
end

"""
    validate_procedure(proc_name::String) -> Vector{ValidationError}

Validate a procedure by name from the registry.
"""
function validate_procedure(proc_name::String)::Vector{ValidationError}
    proc = get_procedure(proc_name)
    if proc === nothing
        return [ValidationError("", 0, "", "Procedure '$proc_name' not found", "", :error)]
    end
    return validate_procedure(proc)
end

end # module ACSValidator
