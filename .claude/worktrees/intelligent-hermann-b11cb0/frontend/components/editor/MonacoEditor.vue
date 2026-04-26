<script setup lang="ts">
import * as monaco from 'monaco-editor'
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'
import jsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker'
import cssWorker from 'monaco-editor/esm/vs/language/css/css.worker?worker'
import htmlWorker from 'monaco-editor/esm/vs/language/html/html.worker?worker'
import tsWorker from 'monaco-editor/esm/vs/language/typescript/ts.worker?worker'

const editorStore = useEditorStore()
const executionStore = useExecutionStore()
const settingsStore = useSettingsStore()
const monacoSettings = useMonacoSettings()
const { setup: setupMonaco } = useMonaco()

const editorContainer = ref<HTMLElement>()
let editorInstance: monaco.editor.IStandaloneCodeEditor | null = null
let currentDecorations: string[] = []

onMounted(() => {
  if (!editorContainer.value) return

  const globalScope = window as unknown as {
    MonacoEnvironment?: { getWorker: (_: string, label: string) => Worker }
  }

  if (!globalScope.MonacoEnvironment) {
    globalScope.MonacoEnvironment = {
      getWorker: (_: string, label: string) => {
        if (label === 'json') return new jsonWorker()
        if (label === 'css' || label === 'scss' || label === 'less') return new cssWorker()
        if (label === 'html' || label === 'handlebars' || label === 'razor') return new htmlWorker()
        if (label === 'typescript' || label === 'javascript') return new tsWorker()
        return new editorWorker()
      },
    }
  }

  setupMonaco(monaco)

  editorInstance = monaco.editor.create(editorContainer.value, {
    value: editorStore.content,
    language: 'astra',
    theme: monacoSettings.editorTheme,
    automaticLayout: true,
    minimap: { enabled: true },
    fontSize: monacoSettings.editorFontSize,
    lineNumbers: 'on',
    scrollBeyondLastLine: false,
    wordWrap: 'on',
    glyphMargin: true,
    folding: true,
    renderLineHighlight: 'line',
    cursorBlinking: 'smooth',
  })

  // Sync content changes
  editorInstance.onDidChangeModelContent(() => {
    const content = editorInstance!.getValue()
    editorStore.setContent(content)
  })
})

// Watch for step execution line highlighting
watch(() => executionStore.currentLine, (line) => {
  if (!editorInstance) return

  if (line > 0) {
    currentDecorations = editorInstance.deltaDecorations(currentDecorations, [
      {
        range: new monaco.Range(line, 1, line, 1),
        options: {
          isWholeLine: true,
          className: 'current-line-decoration',
          glyphMarginClassName: 'current-line-glyph',
        },
      },
    ])
    editorInstance.revealLineInCenter(line)
  } else {
    currentDecorations = editorInstance.deltaDecorations(currentDecorations, [])
  }
})

// Watch for validation markers
watch(() => editorStore.problems, (problems) => {
  if (!editorInstance) return

  const model = editorInstance.getModel()
  if (!model) return

  const markers = problems.map(p => ({
    severity: p.severity === 'error'
      ? monaco.MarkerSeverity.Error
      : monaco.MarkerSeverity.Warning,
    startLineNumber: p.line_number,
    startColumn: 1,
    endLineNumber: p.line_number,
    endColumn: 1000,
    message: p.message,
  }))

  monaco.editor.setModelMarkers(model, 'astra', markers)
})

// Watch for font size changes
watch(() => monacoSettings.editorFontSize, (size) => {
  editorInstance?.updateOptions({ fontSize: size })
})

// Watch for theme changes
watch(() => monacoSettings.editorTheme, (theme) => {
  monaco.editor.setTheme(theme)
})

// External content updates (e.g., loading a file)
watch(() => editorStore.content, (content) => {
  if (!editorInstance) return
  if (editorInstance.getValue() !== content) {
    editorInstance.setValue(content)
  }
})

onUnmounted(() => {
  editorInstance?.dispose()
})
</script>

<template>
  <div ref="editorContainer" class="h-full w-full" />
</template>
