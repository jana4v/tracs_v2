// ASTRA DSL structural validator
// Ports block-matching and structural checks from ACSValidator.jl

import { classifyDocument, type LineClassification, TokenType } from './tokenizer.js'
import { BLOCK_OPENERS, DSL_KEYWORDS } from '../astra-language.js'
import { levenshteinDistance } from '../utils/levenshtein.js'

export interface ValidationDiagnostic {
  line: number
  startCol: number
  endCol: number
  message: string
  severity: 'error' | 'warning'
  suggestion?: string
}

interface BlockStackEntry {
  keyword: string
  line: number
}

/**
 * Validate an ASTRA document and return diagnostics.
 */
export function validateDocument(
  content: string,
  knownProcedures?: Set<string>,
  knownTMRefs?: Set<string>,
  knownTCRefs?: Set<string>,
  knownSCORefs?: Set<string>,
): ValidationDiagnostic[] {
  const diagnostics: ValidationDiagnostic[] = []
  const lines = classifyDocument(content)
  const blockStack: BlockStackEntry[] = []

  let hasTestName = false

  for (const line of lines) {
    if (line.isEmpty || line.isComment) continue

    const { keyword, tokens, lineNumber } = line

    // Check TEST_NAME
    if (keyword === 'TEST_NAME') {
      if (hasTestName) {
        diagnostics.push({
          line: lineNumber,
          startCol: 0,
          endCol: line.text.length,
          message: 'Duplicate TEST_NAME declaration. Only one TEST_NAME per file is allowed.',
          severity: 'error',
        })
      }
      hasTestName = true

      // Check if name is provided
      const nameToken = tokens.find(t => t.type === TokenType.Identifier)
      if (!nameToken) {
        diagnostics.push({
          line: lineNumber,
          startCol: 0,
          endCol: line.text.length,
          message: 'TEST_NAME requires a procedure name.',
          severity: 'error',
          suggestion: 'TEST_NAME my-test',
        })
      }
      continue
    }

    // First non-comment line must be TEST_NAME
    if (!hasTestName && keyword !== null) {
      diagnostics.push({
        line: lineNumber,
        startCol: 0,
        endCol: line.text.length,
        message: 'First statement must be TEST_NAME.',
        severity: 'error',
        suggestion: 'Add TEST_NAME <name> as the first line.',
      })
      hasTestName = true // Don't repeat this error
    }

    // Block openers
    if (keyword && (BLOCK_OPENERS as readonly string[]).includes(keyword)) {
      blockStack.push({ keyword, line: lineNumber })
      continue
    }

    // ELSE - must be inside IF
    if (keyword === 'ELSE') {
      const top = blockStack[blockStack.length - 1]
      if (!top || top.keyword !== 'IF') {
        diagnostics.push({
          line: lineNumber,
          startCol: 0,
          endCol: line.text.length,
          message: 'ELSE without matching IF.',
          severity: 'error',
        })
      }
      continue
    }

    // END - closes blocks
    if (keyword === 'END') {
      if (blockStack.length === 0) {
        diagnostics.push({
          line: lineNumber,
          startCol: 0,
          endCol: line.text.length,
          message: 'END without matching block opener (IF, FOR, WHILE, ON_FAIL, ON_TIMEOUT).',
          severity: 'error',
        })
      } else {
        blockStack.pop()
      }
      continue
    }

    // BREAK - must be inside FOR or WHILE
    if (keyword === 'BREAK') {
      const hasLoop = blockStack.some(b => b.keyword === 'FOR' || b.keyword === 'WHILE')
      if (!hasLoop) {
        diagnostics.push({
          line: lineNumber,
          startCol: 0,
          endCol: line.text.length,
          message: 'BREAK can only be used inside a FOR or WHILE loop.',
          severity: 'error',
        })
      }
      continue
    }

    // CALL - validate target procedure
    if (keyword === 'CALL' && knownProcedures) {
      const targetToken = tokens.find(t => t.type === TokenType.Identifier)
      if (targetToken && !knownProcedures.has(targetToken.value)) {
        diagnostics.push({
          line: lineNumber,
          startCol: targetToken.start,
          endCol: targetToken.end,
          message: `Unknown procedure: "${targetToken.value}".`,
          severity: 'warning',
          suggestion: 'Check that the procedure is loaded.',
        })
      }
      continue
    }

    // Validate TM references
    for (const token of tokens) {
      if (token.type === TokenType.TMReference && knownTMRefs) {
        if (!knownTMRefs.has(token.value)) {
          diagnostics.push({
            line: lineNumber,
            startCol: token.start,
            endCol: token.end,
            message: `Unknown TM reference: "${token.value}".`,
            severity: 'warning',
          })
        }
      }

      if (token.type === TokenType.TCReference && knownTCRefs) {
        if (!knownTCRefs.has(token.value)) {
          diagnostics.push({
            line: lineNumber,
            startCol: token.start,
            endCol: token.end,
            message: `Unknown TC reference: "${token.value}".`,
            severity: 'warning',
          })
        }
      }

      if (token.type === TokenType.SCOReference && knownSCORefs) {
        if (!knownSCORefs.has(token.value)) {
          diagnostics.push({
            line: lineNumber,
            startCol: token.start,
            endCol: token.end,
            message: `Unknown SCO reference: "${token.value}".`,
            severity: 'warning',
          })
        }
      }
    }

    // Check for misspelled keywords (Julia code lines)
    if (line.isJuliaCode) {
      const firstWord = line.trimmed.split(/\s+/)[0].toUpperCase()
      for (const kw of DSL_KEYWORDS) {
        if (firstWord !== kw && levenshteinDistance(firstWord, kw) <= 2 && firstWord.length > 2) {
          diagnostics.push({
            line: lineNumber,
            startCol: 0,
            endCol: firstWord.length,
            message: `Did you mean "${kw}"?`,
            severity: 'warning',
            suggestion: kw,
          })
          break
        }
      }
    }
  }

  // Check unclosed blocks
  for (const block of blockStack) {
    diagnostics.push({
      line: block.line,
      startCol: 0,
      endCol: 100,
      message: `Unclosed ${block.keyword} block. Missing END.`,
      severity: 'error',
    })
  }

  return diagnostics
}
