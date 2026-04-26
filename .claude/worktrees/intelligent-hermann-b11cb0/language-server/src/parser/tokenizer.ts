// ASTRA DSL line-by-line tokenizer
// Ports classification logic from ACSParser.jl

import { DSL_KEYWORDS } from '../astra-language.js'

export enum TokenType {
  Keyword = 'keyword',
  TMReference = 'tm_reference',
  TCReference = 'tc_reference',
  SCOReference = 'sco_reference',
  String = 'string',
  Number = 'number',
  Operator = 'operator',
  Identifier = 'identifier',
  Comment = 'comment',
  Whitespace = 'whitespace',
  Unknown = 'unknown',
}

export interface Token {
  type: TokenType
  value: string
  start: number
  end: number
}

export interface LineClassification {
  lineNumber: number
  text: string
  trimmed: string
  keyword: string | null
  tokens: Token[]
  isComment: boolean
  isEmpty: boolean
  isJuliaCode: boolean
}

const TM_PATTERN = /^TM(\d+)\.(\w+)$/
const TC_PATTERN = /^TC\.(\w+)$/
const SCO_PATTERN = /^SCO\.(\w+)$/
const KEYWORD_SET = new Set(DSL_KEYWORDS)

/**
 * Tokenize a single line of ASTRA DSL code.
 */
export function tokenizeLine(text: string): Token[] {
  const tokens: Token[] = []
  let pos = 0

  while (pos < text.length) {
    // Skip whitespace
    if (/\s/.test(text[pos])) {
      const start = pos
      while (pos < text.length && /\s/.test(text[pos])) pos++
      tokens.push({ type: TokenType.Whitespace, value: text.slice(start, pos), start, end: pos })
      continue
    }

    // Comment
    if (text[pos] === '#' || (text[pos] === '/' && text[pos + 1] === '/')) {
      tokens.push({ type: TokenType.Comment, value: text.slice(pos), start: pos, end: text.length })
      break
    }

    // String
    if (text[pos] === '"') {
      const start = pos
      pos++
      while (pos < text.length && text[pos] !== '"') pos++
      if (pos < text.length) pos++ // closing quote
      tokens.push({ type: TokenType.String, value: text.slice(start, pos), start, end: pos })
      continue
    }

    // Number
    if (/\d/.test(text[pos])) {
      const start = pos
      while (pos < text.length && /[\d.]/.test(text[pos])) pos++
      tokens.push({ type: TokenType.Number, value: text.slice(start, pos), start, end: pos })
      continue
    }

    // Operator
    if (/[=><!+\-*\/]/.test(text[pos])) {
      const start = pos
      while (pos < text.length && /[=><!+\-*\/]/.test(text[pos])) pos++
      tokens.push({ type: TokenType.Operator, value: text.slice(start, pos), start, end: pos })
      continue
    }

    // Word (keyword, identifier, TM/TC/SCO reference)
    if (/[a-zA-Z_$]/.test(text[pos])) {
      const start = pos
      while (pos < text.length && /[\w.$]/.test(text[pos])) pos++
      const word = text.slice(start, pos)

      let type: TokenType
      if (TM_PATTERN.test(word)) {
        type = TokenType.TMReference
      } else if (TC_PATTERN.test(word)) {
        type = TokenType.TCReference
      } else if (SCO_PATTERN.test(word)) {
        type = TokenType.SCOReference
      } else if (KEYWORD_SET.has(word as any)) {
        type = TokenType.Keyword
      } else {
        type = TokenType.Identifier
      }

      tokens.push({ type, value: word, start, end: pos })
      continue
    }

    // Unknown character
    tokens.push({ type: TokenType.Unknown, value: text[pos], start: pos, end: pos + 1 })
    pos++
  }

  return tokens
}

/**
 * Classify a single line of ASTRA DSL code.
 */
export function classifyLine(text: string, lineNumber: number): LineClassification {
  const trimmed = text.trim()
  const tokens = tokenizeLine(text)

  const isComment = trimmed.startsWith('#') || trimmed.startsWith('//')
  const isEmpty = trimmed === ''

  // Find first keyword token
  const firstNonWs = tokens.find(t => t.type !== TokenType.Whitespace)
  const keyword = firstNonWs?.type === TokenType.Keyword ? firstNonWs.value : null

  // If no keyword and not a comment/empty, it's Julia code
  const isJuliaCode = !isComment && !isEmpty && keyword === null

  return {
    lineNumber,
    text,
    trimmed,
    keyword,
    tokens,
    isComment,
    isEmpty,
    isJuliaCode,
  }
}

/**
 * Classify all lines of an ASTRA document.
 */
export function classifyDocument(content: string): LineClassification[] {
  return content.split('\n').map((line, idx) => classifyLine(line, idx + 1))
}
