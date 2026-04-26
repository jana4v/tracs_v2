// ASTRA Document Symbols Provider
// Provides document outline for DSL files

import {
  type Connection,
  type TextDocuments,
  type DocumentSymbolParams,
  type DocumentSymbol,
  SymbolKind,
} from 'vscode-languageserver'
import { type TextDocument } from 'vscode-languageserver-textdocument'
import { classifyDocument } from '../parser/tokenizer.js'
import { BLOCK_OPENERS } from '../astra-language.js'

/**
 * Register document symbols provider.
 */
export function registerDocumentSymbols(
  connection: Connection,
  documents: TextDocuments<TextDocument>,
): void {
  connection.onDocumentSymbol((params: DocumentSymbolParams): DocumentSymbol[] => {
    const document = documents.get(params.textDocument.uri)
    if (!document) return []

    const content = document.getText()
    const lines = classifyDocument(content)
    const symbols: DocumentSymbol[] = []

    let testNameSymbol: DocumentSymbol | null = null

    for (const line of lines) {
      if (line.isEmpty || line.isComment) continue

      const { keyword, lineNumber, trimmed } = line

      if (keyword === 'TEST_NAME') {
        const name = trimmed.replace('TEST_NAME', '').trim()
        testNameSymbol = {
          name: name || '<unnamed>',
          kind: SymbolKind.Module,
          range: {
            start: { line: lineNumber - 1, character: 0 },
            end: { line: lineNumber - 1, character: line.text.length },
          },
          selectionRange: {
            start: { line: lineNumber - 1, character: 0 },
            end: { line: lineNumber - 1, character: line.text.length },
          },
          children: [],
        }
        symbols.push(testNameSymbol)
      }

      // Block structures
      if (keyword && (BLOCK_OPENERS as readonly string[]).includes(keyword)) {
        const symbolKind = keyword === 'ON_FAIL' || keyword === 'ON_TIMEOUT'
          ? SymbolKind.Event
          : SymbolKind.Struct

        const label = keyword === 'IF'
          ? `IF ${trimmed.replace('IF', '').trim()}`
          : keyword === 'FOR'
            ? trimmed
            : keyword === 'WHILE'
              ? `WHILE ${trimmed.replace('WHILE', '').trim()}`
              : keyword

        const symbol: DocumentSymbol = {
          name: label,
          kind: symbolKind,
          range: {
            start: { line: lineNumber - 1, character: 0 },
            end: { line: lineNumber - 1, character: line.text.length },
          },
          selectionRange: {
            start: { line: lineNumber - 1, character: 0 },
            end: { line: lineNumber - 1, character: line.text.length },
          },
        }

        if (testNameSymbol) {
          testNameSymbol.children = testNameSymbol.children || []
          testNameSymbol.children.push(symbol)
        } else {
          symbols.push(symbol)
        }
      }

      // CALL statements
      if (keyword === 'CALL') {
        const target = trimmed.replace('CALL', '').trim()
        const symbol: DocumentSymbol = {
          name: `CALL ${target}`,
          kind: SymbolKind.Function,
          range: {
            start: { line: lineNumber - 1, character: 0 },
            end: { line: lineNumber - 1, character: line.text.length },
          },
          selectionRange: {
            start: { line: lineNumber - 1, character: 0 },
            end: { line: lineNumber - 1, character: line.text.length },
          },
        }

        if (testNameSymbol) {
          testNameSymbol.children = testNameSymbol.children || []
          testNameSymbol.children.push(symbol)
        } else {
          symbols.push(symbol)
        }
      }
    }

    return symbols
  })
}
