// ASTRA Hover Provider
// Shows documentation on hover for DSL keywords and TM references

import {
  type Connection,
  type TextDocuments,
  type HoverParams,
  type Hover,
} from 'vscode-languageserver'
import { type TextDocument } from 'vscode-languageserver-textdocument'
import { STATEMENT_DOCS, DSL_KEYWORDS } from '../astra-language.js'

const KEYWORD_SET = new Set<string>(DSL_KEYWORDS)

/**
 * Register hover provider.
 */
export function registerHover(
  connection: Connection,
  documents: TextDocuments<TextDocument>,
): void {
  connection.onHover((params: HoverParams): Hover | null => {
    const document = documents.get(params.textDocument.uri)
    if (!document) return null

    const text = document.getText()
    const offset = document.offsetAt(params.position)

    // Extract the word at cursor position
    const word = getWordAtOffset(text, offset)
    if (!word) return null

    // DSL keyword
    if (KEYWORD_SET.has(word.text)) {
      const doc = STATEMENT_DOCS[word.text]
      if (doc) {
        return {
          contents: {
            kind: 'markdown',
            value: `**${word.text}** _(ASTRA keyword)_\n\n${doc}`,
          },
          range: {
            start: document.positionAt(word.start),
            end: document.positionAt(word.end),
          },
        }
      }
    }

    // TM reference
    const tmMatch = word.text.match(/^TM(\d+)\.(\w+)$/)
    if (tmMatch) {
      return {
        contents: {
          kind: 'markdown',
          value: `**Telemetry Parameter**\n\n- Bank: TM${tmMatch[1]}\n- Mnemonic: \`${tmMatch[2]}\`\n- Full reference: \`${word.text}\``,
        },
        range: {
          start: document.positionAt(word.start),
          end: document.positionAt(word.end),
        },
      }
    }

    // TC reference
    const tcMatch = word.text.match(/^TC\.(\w+)$/)
    if (tcMatch) {
      return {
        contents: {
          kind: 'markdown',
          value: `**Telecommand**\n\n- Command: \`${tcMatch[1]}\`\n- Full reference: \`${word.text}\``,
        },
        range: {
          start: document.positionAt(word.start),
          end: document.positionAt(word.end),
        },
      }
    }

    // SCO reference
    const scoMatch = word.text.match(/^SCO\.(\w+)$/)
    if (scoMatch) {
      return {
        contents: {
          kind: 'markdown',
          value: `**Spacecraft Operation**\n\n- Command: \`${scoMatch[1]}\`\n- Full reference: \`${word.text}\``,
        },
        range: {
          start: document.positionAt(word.start),
          end: document.positionAt(word.end),
        },
      }
    }

    return null
  })
}

interface WordAtOffset {
  text: string
  start: number
  end: number
}

function getWordAtOffset(text: string, offset: number): WordAtOffset | null {
  if (offset < 0 || offset >= text.length) return null

  // Expand left and right to find word boundaries
  let start = offset
  let end = offset

  // Include word chars and dots (for TM1.xyz_sts, TC.command, SCO.command)
  while (start > 0 && /[\w.]/.test(text[start - 1])) start--
  while (end < text.length && /[\w.]/.test(text[end])) end++

  if (start === end) return null

  return { text: text.slice(start, end), start, end }
}
