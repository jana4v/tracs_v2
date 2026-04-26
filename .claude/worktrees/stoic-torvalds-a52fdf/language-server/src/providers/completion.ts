// ASTRA Completion Provider
// Context-aware autocomplete backed by MongoDB

import {
  type Connection,
  type TextDocuments,
  CompletionItemKind,
  InsertTextFormat,
  type CompletionItem,
  type CompletionParams,
} from 'vscode-languageserver'
import { type TextDocument } from 'vscode-languageserver-textdocument'
import { DSL_KEYWORDS, STATEMENT_DOCS, SNIPPET_TEMPLATES } from '../astra-language.js'
import {
  getTMMnemonics,
  getTCMnemonics,
  getSCOCommands,
  getProcedureNames,
} from '../utils/backend-client.js'

/**
 * Register completion provider.
 */
export function registerCompletion(
  connection: Connection,
  documents: TextDocuments<TextDocument>,
): void {
  connection.onCompletion(async (params: CompletionParams): Promise<CompletionItem[]> => {
    const document = documents.get(params.textDocument.uri)
    if (!document) return []

    const line = document.getText({
      start: { line: params.position.line, character: 0 },
      end: params.position,
    })

    const items: CompletionItem[] = []

    // Context: after "TM" followed by a digit and dot (e.g., "TM1.")
    const tmBankMatch = line.match(/TM(\d+)\.\w*$/)
    if (tmBankMatch) {
      const bank = parseInt(tmBankMatch[1])
      const mnemonics = await getTMMnemonics(bank)
      for (const m of mnemonics) {
        items.push({
          label: m.mnemonic,
          kind: CompletionItemKind.Variable,
          detail: `${m.full_ref} (${m.data_type}${m.unit ? `, ${m.unit}` : ''})`,
          documentation: `${m.description}\n\nSubsystem: ${m.subsystem}\nType: ${m.data_type}`,
          insertText: m.mnemonic,
        })
      }
      return items
    }

    // Context: after "TM" but no dot yet -> suggest TM1., TM2., ... TM10.
    if (line.match(/\bTM\d*$/) && !line.match(/TM\d+\./)) {
      for (let i = 1; i <= 10; i++) {
        items.push({
          label: `TM${i}.`,
          kind: CompletionItemKind.Module,
          detail: `Telemetry Bank ${i}`,
          insertText: `TM${i}.`,
          command: { title: 'Trigger completion', command: 'editor.action.triggerSuggest' },
        })
      }
      return items
    }

    // Context: after "TC."
    if (line.match(/TC\.\w*$/)) {
      const commands = await getTCMnemonics()
      for (const tc of commands) {
        items.push({
          label: tc.command,
          kind: CompletionItemKind.Function,
          detail: `${tc.full_ref} [${tc.subsystem}]`,
          documentation: tc.description,
          insertText: tc.command,
        })
      }
      return items
    }

    // Context: after "SCO."
    if (line.match(/SCO\.\w*$/)) {
      const commands = await getSCOCommands()
      for (const sco of commands) {
        items.push({
          label: sco.command,
          kind: CompletionItemKind.Function,
          detail: `${sco.full_ref} [${sco.subsystem}]`,
          documentation: sco.description,
          insertText: sco.command,
        })
      }
      return items
    }

    // Context: after "CALL "
    if (line.match(/\bCALL\s+\w*$/)) {
      const procedures = await getProcedureNames()
      for (const name of procedures) {
        items.push({
          label: name,
          kind: CompletionItemKind.Reference,
          detail: 'Procedure',
          insertText: name,
        })
      }
      return items
    }

    // Context: line start or after whitespace -> DSL keywords + snippets
    const trimmed = line.trimStart()
    if (trimmed === '' || trimmed.match(/^\w*$/)) {
      // DSL keywords
      for (const kw of DSL_KEYWORDS) {
        const doc = STATEMENT_DOCS[kw]
        items.push({
          label: kw,
          kind: CompletionItemKind.Keyword,
          detail: 'ASTRA keyword',
          documentation: doc || undefined,
          insertText: kw + ' ',
        })
      }

      // Snippet templates
      for (const [key, snippet] of Object.entries(SNIPPET_TEMPLATES)) {
        items.push({
          label: snippet.label,
          kind: CompletionItemKind.Snippet,
          detail: snippet.documentation,
          insertText: snippet.insertText,
          insertTextFormat: InsertTextFormat.Snippet,
        })
      }

      return items
    }

    return items
  })
}
