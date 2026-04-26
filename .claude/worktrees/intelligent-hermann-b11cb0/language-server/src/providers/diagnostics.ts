// ASTRA Diagnostics Provider
// Validates documents and publishes diagnostics

import {
  type Connection,
  type TextDocuments,
  DiagnosticSeverity,
  type Diagnostic,
} from 'vscode-languageserver'
import { type TextDocument } from 'vscode-languageserver-textdocument'
import { validateDocument, type ValidationDiagnostic } from '../parser/validator.js'
import {
  getTMRefSet,
  getTCRefSet,
  getSCORefSet,
  getProcedureNames,
} from '../utils/backend-client.js'

let debounceTimer: ReturnType<typeof setTimeout> | null = null

/**
 * Register diagnostics provider.
 */
export function registerDiagnostics(
  connection: Connection,
  documents: TextDocuments<TextDocument>,
): void {
  // Validate on document change (debounced)
  documents.onDidChangeContent((change) => {
    if (debounceTimer) clearTimeout(debounceTimer)
    debounceTimer = setTimeout(() => {
      validateAndPublish(connection, change.document)
    }, 300)
  })

  // Validate on open
  documents.onDidOpen((event) => {
    validateAndPublish(connection, event.document)
  })

  // Clear diagnostics on close
  documents.onDidClose((event) => {
    connection.sendDiagnostics({ uri: event.document.uri, diagnostics: [] })
  })
}

async function validateAndPublish(
  connection: Connection,
  document: TextDocument,
): Promise<void> {
  const content = document.getText()

  // Fetch known references from MongoDB (cached)
  let tmRefs: Set<string> | undefined
  let tcRefs: Set<string> | undefined
  let scoRefs: Set<string> | undefined
  let procedures: Set<string> | undefined

  try {
    const [tm, tc, sco, procs] = await Promise.allSettled([
      getTMRefSet(),
      getTCRefSet(),
      getSCORefSet(),
      getProcedureNames(),
    ])

    if (tm.status === 'fulfilled') tmRefs = tm.value
    if (tc.status === 'fulfilled') tcRefs = tc.value
    if (sco.status === 'fulfilled') scoRefs = sco.value
    if (procs.status === 'fulfilled') procedures = new Set(procs.value)
  } catch {
    // Continue without MongoDB data - just structural validation
  }

  const issues = validateDocument(content, procedures, tmRefs, tcRefs, scoRefs)

  const diagnostics: Diagnostic[] = issues.map(issue => ({
    severity: issue.severity === 'error'
      ? DiagnosticSeverity.Error
      : DiagnosticSeverity.Warning,
    range: {
      start: { line: issue.line - 1, character: issue.startCol },
      end: { line: issue.line - 1, character: issue.endCol },
    },
    message: issue.message + (issue.suggestion ? ` (${issue.suggestion})` : ''),
    source: 'astra',
  }))

  connection.sendDiagnostics({ uri: document.uri, diagnostics })
}
