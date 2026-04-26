// Monaco Editor setup: ASTRA language registration, theme, and completions
// Used by MonacoEditor.vue, execute.vue, and any page with Monaco editors

import type * as Monaco from 'monaco-editor'
import type { TCRef, SCORef, EnrichedTMRef } from '~/types/astra'

// Module-level caches (shared across all editors, persist across composable calls)
let subsystemCache: string[] | null = null
const tmSubsystemCache = new Map<string, EnrichedTMRef[]>()
let tcCache: TCRef[] | null = null
let scoCache: SCORef[] | null = null
let completionRegistered = false

const dslKeywords = [
  'TEST_NAME', 'PRE_TEST_REQ', 'SEND', 'SENDTCP',
  'WAIT', 'UNTIL', 'TIMEOUT', 'CHECK', 'WITHIN',
  'EXPECTED', 'ALERT_MSG', 'ABORT_TEST', 'BREAK',
  'CALL', 'IF', 'ELSE', 'END', 'FOR', 'IN', 'TO',
  'WHILE', 'ON_FAIL', 'ON_TIMEOUT', 'AND', 'OR', 'NOT',
]

const dslSnippets = [
  {
    label: 'IF...END',
    insertText: 'IF ${1:condition}\n    ${2}\nEND',
    documentation: 'Conditional block',
  },
  {
    label: 'IF...ELSE...END',
    insertText: 'IF ${1:condition}\n    ${2}\nELSE\n    ${3}\nEND',
    documentation: 'Conditional block with else branch',
  },
  {
    label: 'FOR...END',
    insertText: 'FOR ${1:i} IN ${2:1} TO ${3:10}\n    ${4}\nEND',
    documentation: 'Loop with counter variable',
  },
  {
    label: 'WHILE...END',
    insertText: 'WHILE ${1:condition}\n    ${2}\nEND',
    documentation: 'Conditional loop',
  },
  {
    label: 'ON_FAIL...END',
    insertText: 'ON_FAIL\n    ${1:ALERT_MSG "Error occurred"}\n    ${2:ABORT_TEST}\nEND',
    documentation: 'Error handler block',
  },
  {
    label: 'ON_TIMEOUT...END',
    insertText: 'ON_TIMEOUT\n    ${1:ALERT_MSG "Timeout occurred"}\nEND',
    documentation: 'Timeout handler block',
  },
  {
    label: 'CHECK...WITHIN',
    insertText: 'CHECK ${1:TM.SUBSYSTEM.param} ${2:==} ${3:value} WITHIN ${4:10}',
    documentation: 'Verify telemetry condition within timeout',
  },
  {
    label: 'WAIT UNTIL...TIMEOUT',
    insertText: 'WAIT UNTIL ${1:TM.SUBSYSTEM.param == "value"} TIMEOUT ${2:30}',
    documentation: 'Wait for condition with timeout',
  },
]

const juliaSnippets = [
  {
    label: 'Julia IF...end',
    insertText: 'if ${1:condition}\n    ${2}\nend',
    documentation: 'Julia conditional block',
  },
  {
    label: 'Julia FOR...end',
    insertText: 'for ${1:i} in ${2:1:10}\n    ${3}\nend',
    documentation: 'Julia for loop',
  },
]

// Helper: parse digitalStatus string into state labels
function parseDigitalStates(digitalStatus: string): string[] {
  // Support comma-separated ("A,B,C,D") and legacy semicolon format ("0:OFF;1:ON")
  const sep = digitalStatus.includes(';') ? ';' : ','
  return digitalStatus.split(sep)
    .map(s => s.trim())
    .filter(s => s.length > 0)
    .map(s => { const idx = s.indexOf(':'); return idx >= 0 ? s.slice(idx + 1).trim() : s })
    .filter(s => s.length > 0)
}

export function useMonaco() {
  function registerAstraLanguage(monaco: typeof Monaco) {
    // Register the ASTRA language if not already registered
    const languages = monaco.languages.getLanguages()
    if (languages.some((lang: any) => lang.id === 'astra')) return

    monaco.languages.register({ id: 'astra' })

    // Monarch tokenizer
    monaco.languages.setMonarchTokensProvider('astra', {
      keywords: dslKeywords,
      tokenizer: {
        root: [
          [/\b(TEST_NAME)\b/, 'keyword.declaration'],
          [/\b(IF|ELSE|END|FOR|WHILE|IN|TO)\b/, 'keyword.control'],
          [/\b(SEND|SENDTCP|WAIT|CHECK|EXPECTED|CALL)\b/, 'keyword.command'],
          [/\b(PRE_TEST_REQ|ALERT_MSG|ABORT_TEST|BREAK)\b/, 'keyword.action'],
          [/\b(ON_FAIL|ON_TIMEOUT|UNTIL|TIMEOUT|WITHIN)\b/, 'keyword.handler'],
          [/\b(AND|OR|NOT)\b/, 'keyword.operator'],
          [/\bTM\d*\.\w+\.[\w][\w+\-.]*/, 'variable.tm'],
          [/\bTC\.\w+\b/, 'variable.tc'],
          [/\bSCO\.\w+\b/, 'variable.sco'],
          [/"[^"]*"/, 'string'],
          [/\d+(\.\d+)?/, 'number'],
          [/#.*$/, 'comment'],
          [/\/\/.*$/, 'comment'],
          [/[=><!]+/, 'operator'],
        ],
      },
    } as any)
  }

  function registerAstraTheme(monaco: typeof Monaco) {
    // Dark theme
    monaco.editor.defineTheme('ASTRA-dark', {
      base: 'vs-dark',
      inherit: true,
      rules: [
        { token: 'keyword.declaration', foreground: 'C586C0', fontStyle: 'bold' },
        { token: 'keyword.control', foreground: 'C586C0' },
        { token: 'keyword.command', foreground: '4EC9B0' },
        { token: 'keyword.action', foreground: 'DCDCAA' },
        { token: 'keyword.handler', foreground: '569CD6' },
        { token: 'keyword.operator', foreground: 'D4D4D4' },
        { token: 'variable.tm', foreground: '9CDCFE' },
        { token: 'variable.tc', foreground: '4FC1FF' },
        { token: 'variable.sco', foreground: 'CE9178' },
        { token: 'string', foreground: 'CE9178' },
        { token: 'number', foreground: 'B5CEA8' },
        { token: 'comment', foreground: '6A9955' },
        { token: 'operator', foreground: 'D4D4D4' },
      ],
      colors: {
        'editor.background': '#1e1e2e',
        'editor.foreground': '#cdd6f4',
        'editorLineNumber.foreground': '#585b70',
        'editorLineNumber.activeForeground': '#cdd6f4',
        'editor.selectionBackground': '#45475a',
        'editor.lineHighlightBackground': '#252536',
      },
    })

    // Light theme
    monaco.editor.defineTheme('ASTRA-light', {
      base: 'vs',
      inherit: true,
      rules: [
        { token: 'keyword.declaration', foreground: 'AF00DB', fontStyle: 'bold' },
        { token: 'keyword.control', foreground: 'AF00DB' },
        { token: 'keyword.command', foreground: '267F99' },
        { token: 'keyword.action', foreground: '795E26' },
        { token: 'keyword.handler', foreground: '0000FF' },
        { token: 'keyword.operator', foreground: '000000' },
        { token: 'variable.tm', foreground: '001080' },
        { token: 'variable.tc', foreground: '088F8F' },
        { token: 'variable.sco', foreground: 'A31515' },
        { token: 'string', foreground: 'A31515' },
        { token: 'number', foreground: '098658' },
        { token: 'comment', foreground: '008000' },
        { token: 'operator', foreground: '000000' },
      ],
      colors: {
        'editor.background': '#ffffff',
        'editor.foreground': '#000000',
        'editorLineNumber.foreground': '#237893',
        'editorLineNumber.activeForeground': '#0b0f19',
        'editor.selectionBackground': '#ADD6FF',
        'editor.lineHighlightBackground': '#f1f8ff',
      },
    })
  }

  function registerAstraCompletions(monaco: typeof Monaco) {
    // Only register once globally (it's per-language, not per-editor)
    if (completionRegistered) return
    completionRegistered = true

    const api = useAstraApi()
    const mnemonicsStore = useMnemonicsStore()

    async function getSubsystems(): Promise<string[]> {
      if (subsystemCache) return subsystemCache
      try {
        const result = await api.getTMSubsystems()
        subsystemCache = [...new Set(result.subsystems)]
        return subsystemCache
      } catch {
        return []
      }
    }

    async function getTMForSubsystem(subsystem: string): Promise<EnrichedTMRef[]> {
      if (tmSubsystemCache.has(subsystem)) return tmSubsystemCache.get(subsystem)!
      try {
        const data = await api.getTMBySubsystem(subsystem)
        tmSubsystemCache.set(subsystem, data)
        return data
      } catch {
        return []
      }
    }

    async function getTCMnemonics(): Promise<TCRef[]> {
      if (tcCache) return tcCache
      if (mnemonicsStore.tcMnemonics.length > 0) {
        tcCache = mnemonicsStore.tcMnemonics
        return tcCache
      }
      try {
        tcCache = await api.getAllTCMnemonics()
        return tcCache
      } catch {
        return []
      }
    }

    async function getSCOCommands(): Promise<SCORef[]> {
      if (scoCache) return scoCache
      if (mnemonicsStore.scoCommands.length > 0) {
        scoCache = mnemonicsStore.scoCommands
        return scoCache
      }
      try {
        scoCache = await api.getAllSCOCommands()
        return scoCache
      } catch {
        return []
      }
    }

    monaco.languages.registerCompletionItemProvider('astra', {
      triggerCharacters: ['.', ' ', '=', '>', '<'],
      provideCompletionItems: async (model, position) => {
        const line = model.getLineContent(position.lineNumber).slice(0, position.column - 1)
        const word = model.getWordUntilPosition(position)
        const range = {
          startLineNumber: position.lineNumber,
          endLineNumber: position.lineNumber,
          startColumn: word.startColumn,
          endColumn: word.endColumn,
        }

        const suggestions: Monaco.languages.CompletionItem[] = []

        // Pattern 1: Value suggestions after comparison operators
        // e.g. TM.AOC.xyz_sts == , TM1.PWR.voltage >= , TM.THR.temp <
        const valueMatch = line.match(/TM\d*\.(\w+)\.([\w+\-.]+)\s*(==|>=|<=|>|<)\s*$/)
        if (valueMatch) {
          const subsystem = valueMatch[1]
          const mnemonic = valueMatch[2]
          const mnemonics = await getTMForSubsystem(subsystem)
          const ref = mnemonics.find(m => m.cdbMnemonic === mnemonic)

          if (ref) {
            if (ref.type === 'BINARY') {
              // Suggest digital states from range array, fallback to digitalStatus string
              const states = Array.isArray(ref.range) && ref.range.length > 0 && typeof ref.range[0] === 'string'
                ? ref.range as string[]
                : ref.digitalStatus ? parseDigitalStates(ref.digitalStatus) : []
              for (const state of states) {
                suggestions.push({
                  label: `"${state}"`,
                  kind: monaco.languages.CompletionItemKind.EnumMember,
                  detail: `${ref.type} state`,
                  documentation: `Digital status from ${ref.cdbPidNo}`,
                  insertText: `"${state}"`,
                  range,
                })
              }
            } else if (Array.isArray(ref.range) && ref.range.length === 2) {
              // Analog/Decimal: suggest min and max values
              const [min, max] = ref.range
              const unitStr = ref.unit ? ` ${ref.unit}` : ''
              suggestions.push({
                label: `${min}`,
                kind: monaco.languages.CompletionItemKind.Value,
                detail: `Min value (range: ${min} to ${max}${unitStr})`,
                documentation: `${ref.description}\nRange: ${min} to ${max}${unitStr}`,
                insertText: `${min}`,
                range,
              })
              suggestions.push({
                label: `${max}`,
                kind: monaco.languages.CompletionItemKind.Value,
                detail: `Max value (range: ${min} to ${max}${unitStr})`,
                documentation: `${ref.description}\nRange: ${min} to ${max}${unitStr}`,
                insertText: `${max}`,
                range,
              })
            }
          }
          return { suggestions }
        }

        // Pattern 2: Mnemonic suggestions after subsystem
        // e.g. TM.AOC. , TM1.PWR. , TM2.ACM.xyz
        const mnemonicMatch = line.match(/TM\d*\.(\w+)\.[\w+\-.]*$/)
        if (mnemonicMatch) {
          const subsystem = mnemonicMatch[1]
          const mnemonics = await getTMForSubsystem(subsystem)
          for (const m of mnemonics) {
            const unitStr = m.unit ? `, ${m.unit}` : ''
            suggestions.push({
              label: m.cdbMnemonic,
              kind: monaco.languages.CompletionItemKind.Variable,
              detail: `${m.cdbPidNo} (${m.type}${unitStr})`,
              documentation: `${m.description}\n\nSubsystem: ${m.subsystem}\nType: ${m.type}\nPID: ${m.cdbPidNo}`,
              insertText: m.cdbMnemonic,
              range,
            })
          }
          return { suggestions }
        }

        // Pattern 3: Subsystem suggestions after TM.
        // e.g. TM. , TM1. , TM2. , TM3. , TM4.
        if (/\bTM\d*\.\w*$/.test(line) && !/TM\d*\.\w+\./.test(line)) {
          const subsystems = await getSubsystems()
          for (const sub of subsystems) {
            suggestions.push({
              label: sub,
              kind: monaco.languages.CompletionItemKind.Module,
              detail: `Subsystem ${sub}`,
              insertText: sub,
              range,
            })
          }
          return { suggestions }
        }

        // TC command completions (e.g. TC.START_RW)
        if (/TC\.\w*$/.test(line)) {
          const commands = await getTCMnemonics()
          for (const tc of commands) {
            suggestions.push({
              label: tc.command,
              kind: monaco.languages.CompletionItemKind.Function,
              detail: `${tc.full_ref} [${tc.subsystem}]`,
              documentation: tc.description,
              insertText: tc.command,
              range,
            })
          }
          return { suggestions }
        }

        // SCO command completions (e.g. SCO.REBOOT)
        if (/SCO\.\w*$/.test(line)) {
          const commands = await getSCOCommands()
          for (const sco of commands) {
            suggestions.push({
              label: sco.command,
              kind: monaco.languages.CompletionItemKind.Function,
              detail: `${sco.full_ref} [${sco.subsystem}]`,
              documentation: sco.description,
              insertText: sco.command,
              range,
            })
          }
          return { suggestions }
        }

        // DSL keywords and snippets (at line start or partial keyword)
        const trimmed = line.trimStart()
        if (trimmed === '' || /^\w*$/.test(trimmed)) {
          for (const kw of dslKeywords) {
            suggestions.push({
              label: kw,
              kind: monaco.languages.CompletionItemKind.Keyword,
              detail: 'ASTRA keyword',
              insertText: `${kw} `,
              range,
            })
          }

          for (const snippet of dslSnippets) {
            suggestions.push({
              label: snippet.label,
              kind: monaco.languages.CompletionItemKind.Snippet,
              detail: snippet.documentation,
              insertText: snippet.insertText,
              insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
              range,
            })
          }

          for (const snippet of juliaSnippets) {
            suggestions.push({
              label: snippet.label,
              kind: monaco.languages.CompletionItemKind.Snippet,
              detail: snippet.documentation,
              insertText: snippet.insertText,
              insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
              range,
            })
          }
        }

        return { suggestions }
      },
    })
  }

  function setup(monaco: typeof Monaco) {
    registerAstraLanguage(monaco)
    registerAstraTheme(monaco)
    registerAstraCompletions(monaco)
  }

  function clearTMCaches() {
    subsystemCache = null
    tmSubsystemCache.clear()
  }

  return {
    setup,
    registerAstraLanguage,
    registerAstraTheme,
    registerAstraCompletions,
    clearTMCaches,
  }
}
