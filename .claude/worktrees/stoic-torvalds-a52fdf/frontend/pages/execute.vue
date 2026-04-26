<script setup lang="ts">
import * as monaco from 'monaco-editor'
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'
import jsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker'
import cssWorker from 'monaco-editor/esm/vs/language/css/css.worker?worker'
import htmlWorker from 'monaco-editor/esm/vs/language/html/html.worker?worker'
import tsWorker from 'monaco-editor/esm/vs/language/typescript/ts.worker?worker'
import type { Procedure } from '~/types/astra'

definePageMeta({ title: 'Execute' })

const api = useAstraApi()
const settingsStore = useSettingsStore()
const monacoSettings = useMonacoSettings()
const executionStore = useExecutionStore()
const { setup: setupMonaco } = useMonaco()

const toast = useToast()

// State
const procedureList = ref<Procedure[]>([])
const selectedProcedure = ref<string | null>(null)
const loadedContent = ref('')
const loadedTestName = ref<string | null>(null)
const loadedVersion = ref<number | null>(null)
const loading = ref(false)

// Manual command snippet
const manualCommand = ref('')

// Editor containers
const mainEditorContainer = ref<HTMLElement>()
const snippetEditorContainer = ref<HTMLElement>()
let mainEditor: monaco.editor.IStandaloneCodeEditor | null = null
let snippetEditor: monaco.editor.IStandaloneCodeEditor | null = null
let monacoInit = false
let currentDecorations: string[] = []

// Execution controls
let runTimer: ReturnType<typeof setTimeout> | null = null

function ensureMonacoEnv() {
  if (monacoInit) return
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
  monacoInit = true
}

// Load procedures for project
async function loadProceduresByProject() {
  loading.value = true
  try {
    const result = await api.getProcedures(settingsStore.globalProject || undefined)
    procedureList.value = result.procedures
  } catch (e: any) {
    toast.add({ severity: 'error', summary: 'Error', detail: `Failed to load procedures: ${e.message}`, life: 3000 })
  } finally {
    loading.value = false
  }
}

// Filterable options for procedure dropdown
const procedureOptions = computed(() =>
  procedureList.value.map(p => ({
    label: `${p.test_name} (v${p.latest_version})`,
    value: p.test_name,
  }))
)

// Load selected procedure (latest version only)
async function loadSelectedProcedure() {
  if (!selectedProcedure.value) return
  loading.value = true
  try {
    const proc = await api.getProcedure(
      selectedProcedure.value,
      settingsStore.globalProject || undefined,
    )
    const content = proc.latest_content ?? ''
    loadedContent.value = content
    loadedTestName.value = proc.test_name
    loadedVersion.value = proc.latest_version
    if (mainEditor) {
      mainEditor.setValue(content)
    }

    // Load into backend parser
    await api.loadProcedure(loadedContent.value, `${selectedProcedure.value}.tst`)
    executionStore.addLog(`Loaded procedure: ${selectedProcedure.value} v${loadedVersion.value}`, 'ok')
  } catch (e: any) {
    toast.add({ severity: 'error', summary: 'Error', detail: `Failed to load: ${e.message}`, life: 3000 })
  } finally {
    loading.value = false
  }
}

// Watch procedure selection
watch(selectedProcedure, () => {
  if (selectedProcedure.value) loadSelectedProcedure()
})

// Execution handlers
async function handleValidate() {
  if (!loadedTestName.value) return
  executionStore.addLog('Validating...', 'info')
  try {
    const result = await api.validateProcedure(loadedTestName.value)
    if (result.valid) {
      executionStore.addLog('Validation passed', 'ok')
      toast.add({ severity: 'success', summary: 'Valid', detail: 'Procedure is valid', life: 3000 })
    } else {
      executionStore.addLog(`Found ${result.errors.length} problem(s)`, 'error')
      // Show markers on editor
      if (mainEditor) {
        const model = mainEditor.getModel()
        if (model) {
          const markers = result.errors.map(p => ({
            severity: p.severity === 'error' ? monaco.MarkerSeverity.Error : monaco.MarkerSeverity.Warning,
            startLineNumber: p.line_number,
            startColumn: 1,
            endLineNumber: p.line_number,
            endColumn: 1000,
            message: p.message,
          }))
          monaco.editor.setModelMarkers(model, 'astra', markers)
        }
      }
    }
  } catch (e: any) {
    executionStore.addLog(`Validation error: ${e.message}`, 'error')
  }
}

async function ensureStepSession(): Promise<boolean> {
  if (!loadedTestName.value) {
    toast.add({ severity: 'warn', summary: 'Warning', detail: 'No procedure loaded', life: 3000 })
    return false
  }

  if (executionStore.sessionId) return true

  executionStore.addLog(`Starting step session: ${loadedTestName.value}`, 'info')
  try {
    const result = await api.startStepSession(loadedTestName.value)
    if (result.success) {
      executionStore.setSessionId(result.session_id)
      executionStore.updateStepState(result.state)
      executionStore.addLog('Step session ready', 'ok')
      return true
    }
    executionStore.addLog(`Step mode error: ${(result as any).error}`, 'error')
  } catch (e: any) {
    executionStore.addLog(`Step mode error: ${e.message}`, 'error')
  }
  return false
}

async function handleRun() {
  const ready = await ensureStepSession()
  if (!ready) return
  executionStore.setStatus('running')
  executionStore.addLog(`Running: ${loadedTestName.value}`, 'info')
  scheduleNextStep(0)
}

async function handleStepStart() {
  if (executionStore.status === 'running') stopAutoRun('Paused')
  const ready = await ensureStepSession()
  if (!ready) return
  executionStore.setStatus('stepping')
}

async function handleStepNext() {
  if (!executionStore.sessionId || executionStore.status !== 'stepping') return
  try {
    const state = await api.stepNext(executionStore.sessionId)
    executionStore.updateStepState(state)
  } catch (e: any) {
    executionStore.addLog(`Step error: ${e.message}`, 'error')
  }
}

async function handleStepReset() {
  if (!executionStore.sessionId) return
  stopAutoRun()
  try {
    const state = await api.stepReset(executionStore.sessionId)
    executionStore.updateStepState(state)
    executionStore.addLog('Reset', 'ok')
    executionStore.setStatus('stepping')
  } catch (e: any) {
    executionStore.addLog(`Reset error: ${e.message}`, 'error')
  }
}

function handleStop() {
  stopAutoRun('Stopped')
  executionStore.reset()
}

function handlePause() {
  if (executionStore.status !== 'running') return
  stopAutoRun('Paused')
  executionStore.setStatus('stepping')
}

function stopAutoRun(msg?: string) {
  if (runTimer) { clearTimeout(runTimer); runTimer = null }
  if (msg) executionStore.addLog(msg, 'warning')
}

function scheduleNextStep(delayMs: number) {
  if (runTimer) clearTimeout(runTimer)
  runTimer = setTimeout(runNextStep, delayMs)
}

async function runNextStep() {
  if (executionStore.status !== 'running' || !executionStore.sessionId) return
  try {
    const state = await api.stepNext(executionStore.sessionId)
    executionStore.updateStepState(state)
    if (state.status === 'completed') {
      executionStore.setStatus('completed')
      stopAutoRun('Completed')
      return
    }
    if (state.status === 'failed' || state.status === 'error') {
      executionStore.setStatus('error')
      stopAutoRun('Execution failed')
      return
    }
    scheduleNextStep(executionStore.runDelayMs)
  } catch (e: any) {
    executionStore.setStatus('error')
    executionStore.addLog(`Error: ${e.message}`, 'error')
    stopAutoRun()
  }
}

// Send manual command snippet
async function sendManualCommand() {
  const cmd = manualCommand.value.trim()
  if (!cmd) return
  executionStore.addLog(`> ${cmd}`, 'info')
  try {
    // Load the snippet as a mini procedure, then step through it
    const loadResult = await api.loadProcedure(`TEST_NAME _manual_cmd\n${cmd}`, '<manual>')
    if (loadResult.success) {
      const session = await api.startStepSession('_manual_cmd')
      if (session.success) {
        // Step through all lines
        let state = session.state
        while (state.status !== 'completed' && state.status !== 'error') {
          state = await api.stepNext(session.session_id)
          if (state.output) executionStore.addLog(state.output, 'info')
        }
        executionStore.addLog(`Manual command: ${state.status}`, state.status === 'completed' ? 'ok' : 'error')
      }
    } else {
      executionStore.addLog(`Load error: ${(loadResult as any).error}`, 'error')
    }
  } catch (e: any) {
    executionStore.addLog(`Command error: ${e.message}`, 'error')
  }
  manualCommand.value = ''
  if (snippetEditor) snippetEditor.setValue('')
}

// Watch line highlighting
watch(() => executionStore.currentLine, (line) => {
  if (!mainEditor) return
  if (line > 0) {
    currentDecorations = mainEditor.deltaDecorations(currentDecorations, [{
      range: new monaco.Range(line, 1, line, 1),
      options: {
        isWholeLine: true,
        className: 'current-line-decoration',
        glyphMarginClassName: 'current-line-glyph',
      },
    }])
    mainEditor.revealLineInCenter(line)
  } else {
    currentDecorations = mainEditor.deltaDecorations(currentDecorations, [])
  }
})

// Active tab for console/variables panel
const activeTab = ref(0)

onMounted(() => {
  ensureMonacoEnv()
  loadProceduresByProject()

  nextTick(() => {
    if (mainEditorContainer.value) {
      mainEditor = monaco.editor.create(mainEditorContainer.value, {
        value: '',
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
        readOnly: true,
      })
    }
    if (snippetEditorContainer.value) {
      snippetEditor = monaco.editor.create(snippetEditorContainer.value, {
        value: '',
        language: 'astra',
        theme: monacoSettings.editorTheme,
        automaticLayout: true,
        minimap: { enabled: false },
        fontSize: monacoSettings.editorFontSize,
        lineNumbers: 'on',
        scrollBeyondLastLine: false,
        wordWrap: 'on',
        glyphMargin: false,
        folding: false,
        renderLineHighlight: 'line',
        cursorBlinking: 'smooth',
      })
      snippetEditor.onDidChangeModelContent(() => {
        manualCommand.value = snippetEditor!.getValue()
      })
    }
  })
})

// Watch project changes to reload procedure list
watch(() => settingsStore.globalProject, () => {
  loadProceduresByProject()
})

watch(() => monacoSettings.editorFontSize, (size) => {
  mainEditor?.updateOptions({ fontSize: size })
  snippetEditor?.updateOptions({ fontSize: size })
})

watch(() => monacoSettings.editorTheme, (theme) => {
  monaco.editor.setTheme(theme)
})

onUnmounted(() => {
  stopAutoRun()
  mainEditor?.dispose()
  snippetEditor?.dispose()
})
</script>

<template>
  <div class="flex flex-col h-full">
    <!-- Toolbar -->
    <div class="flex items-center gap-2 px-3 py-2 border-b border-[var(--astra-border)] bg-[var(--astra-surface)]">
      <!-- Procedure Selector -->
      <div class="flex items-center gap-2">
        <span class="text-xs text-[var(--astra-muted)]">Project:</span>
        <Tag severity="info" :value="settingsStore.globalProject" class="text-xs" />
        <Select
          v-model="selectedProcedure"
          :options="procedureOptions"
          option-label="label"
          option-value="value"
          placeholder="Select Procedure..."
          class="w-72 text-xs"
          size="small"
          filter
          :loading="loading"
        />
      </div>

      <Divider layout="vertical" class="mx-1 h-6" />

      <!-- Execution Controls -->
      <div class="flex items-center gap-1">
        <Button icon="pi pi-check-circle" label="Validate" size="small" severity="info" outlined :disabled="!loadedTestName" @click="handleValidate" />
        <Button icon="pi pi-play" label="Run" size="small" severity="success" :disabled="!loadedTestName || executionStore.status === 'running'" @click="handleRun" />
        <Button icon="pi pi-pause" label="Pause" size="small" severity="secondary" outlined :disabled="executionStore.status !== 'running'" @click="handlePause" />
        <Button icon="pi pi-step-forward-alt" label="Step" size="small" severity="warn" outlined :disabled="executionStore.isStepping" @click="handleStepStart" />
        <Button icon="pi pi-forward" size="small" severity="warn" text :disabled="!executionStore.isStepping" v-tooltip.bottom="'Next'" @click="handleStepNext" />
        <Button icon="pi pi-replay" size="small" severity="secondary" text :disabled="!executionStore.isStepping" v-tooltip.bottom="'Reset'" @click="handleStepReset" />
        <Button icon="pi pi-stop" size="small" severity="danger" text :disabled="!executionStore.isRunning" v-tooltip.bottom="'Stop'" @click="handleStop" />
      </div>

      <div class="flex-1" />

      <!-- Run speed slider -->
      <div class="hidden md:flex items-center gap-2 text-xs text-[var(--astra-text)]/70">
        <span>Speed</span>
        <Slider v-model="executionStore.runDelayMs" :min="50" :max="1000" :step="50" class="w-24" />
        <span class="tabular-nums">{{ executionStore.runDelayMs }}ms</span>
      </div>

      <!-- Status -->
      <div class="flex items-center gap-2 text-xs">
        <Tag v-if="loadedTestName" severity="info" :value="loadedTestName" />
        <Tag v-if="loadedVersion" severity="secondary" :value="`v${loadedVersion}`" />
        <Tag v-if="executionStore.isRunning" severity="success" :value="executionStore.status" />
      </div>
    </div>

    <!-- Main Area -->
    <Splitter class="flex-1 overflow-hidden" style="border: none;">
      <!-- Editor Panel -->
      <SplitterPanel :size="60" :min-size="30">
        <div class="flex flex-col h-full">
          <!-- Read-only main editor -->
          <div ref="mainEditorContainer" class="flex-1" />

          <!-- Manual Command Snippet -->
          <div class="border-t border-[var(--astra-border)]">
            <div class="flex items-center gap-2 px-3 py-1 bg-[var(--astra-surface)]">
              <i class="pi pi-terminal text-xs text-[var(--astra-accent)]" />
              <span class="text-xs font-medium text-[var(--astra-text)]">Manual Command</span>
              <div class="flex-1" />
              <Button
                icon="pi pi-send"
                label="Send"
                size="small"
                severity="success"
                outlined
                :disabled="!manualCommand.trim()"
                @click="sendManualCommand"
              />
            </div>
            <div ref="snippetEditorContainer" class="h-28" />
          </div>
        </div>
      </SplitterPanel>

      <!-- Console / Variables Panel -->
      <SplitterPanel :size="40" :min-size="20" class="overflow-x-auto overflow-y-auto">
        <TabView v-model:active-index="activeTab" class="h-full flex flex-col" :style="{ backgroundColor: 'var(--astra-surface)' }" :pt="{ nav: { style: 'background-color: var(--astra-surface-2)' } }">
          <TabPanel header="Console" :pt="{ root: { style: 'background-color: var(--astra-surface)' } }">
            <div class="min-h-0 flex-1 p-2 font-mono text-lg" style="max-height: 100%;">
              <div
                v-for="(entry, i) in executionStore.log"
                :key="i"
                class="py-0.5 whitespace-nowrap"
                :class="{
                  'text-[var(--astra-success)]': entry.type === 'ok',
                  'text-[var(--astra-error)]': entry.type === 'error',
                  'text-[var(--astra-warning)]': entry.type === 'warning',
                  'text-[var(--astra-text)]/70': entry.type === 'info',
                }"
              >
                <span class="text-[var(--astra-muted)]">{{ entry.timestamp }}</span>
                {{ entry.message }}
              </div>
              <div v-if="executionStore.log.length === 0" class="text-[var(--astra-muted)] text-center py-4">
                No output yet
              </div>
            </div>
          </TabPanel>
          <TabPanel header="Variables" :pt="{ root: { style: 'background-color: var(--astra-surface)' } }">
            <div class="min-h-0 flex-1  p-2" style="max-height: 100%;">
              <div v-if="Object.keys(executionStore.variables).length === 0" class="text-[var(--astra-muted)] text-lg text-center py-4">
                No variables
              </div>
              <div v-else class="space-y-1">
                <div
                  v-for="(value, key) in executionStore.variables"
                  :key="key"
                  class="flex items-center gap-2 text-lg py-0.5 px-2 rounded hover:bg-[var(--astra-border)]/30 whitespace-nowrap"
                >
                  <span class="font-medium text-[var(--astra-accent)]">{{ key }}</span>
                  <span class="text-[var(--astra-muted)]">=</span>
                  <span class="text-[var(--astra-text)]">{{ value }}</span>
                </div>
              </div>
            </div>
          </TabPanel>
          <TabPanel header="Call Stack" :pt="{ root: { style: 'background-color: var(--astra-surface)' } }">
            <div class="min-h-0 flex-1  p-2" style="max-height: 100%;">
              <div v-if="executionStore.callStack.length === 0" class="text-[var(--astra-muted)] text-lg text-center py-4">
                Empty call stack
              </div>
              <div v-else class="space-y-1">
                <div
                  v-for="(frame, i) in executionStore.callStack"
                  :key="i"
                  class="text-lg py-0.5 px-2 rounded hover:bg-[var(--astra-border)]/30 text-[var(--astra-text)] whitespace-nowrap"
                >
                  {{ frame }}
                </div>
              </div>
            </div>
          </TabPanel>
        </TabView>
      </SplitterPanel>
    </Splitter>
  </div>
</template>
