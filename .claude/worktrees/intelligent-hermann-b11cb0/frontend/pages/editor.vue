<script setup lang="ts">
import MonacoEditor from '~/components/editor/MonacoEditor.vue'
import EditorToolbar from '~/components/editor/EditorToolbar.vue'
import ProblemsPanel from '~/components/editor/ProblemsPanel.vue'
import ConsoleOutput from '~/components/execution/ConsoleOutput.vue'
import VariablesWatch from '~/components/execution/VariablesWatch.vue'
import CallStackPanel from '~/components/execution/CallStackPanel.vue'
import TMPanel from '~/components/telemetry/TMPanel.vue'

definePageMeta({ title: 'Editor' })

const editorStore = useEditorStore()
const executionStore = useExecutionStore()
const api = useAstraApi()

let runTimer: ReturnType<typeof setTimeout> | null = null

// === Validate ===
async function handleValidate() {
  executionStore.addLog('Validating syntax...', 'info')
  try {
    const loadResult = await api.loadProcedure(editorStore.content, editorStore.fileName || '<editor>')
    if (!loadResult.success) {
      executionStore.addLog(`Load error: ${(loadResult as any).error}`, 'error')
      return
    }
    editorStore.setTestName(loadResult.test_name)

    const validateResult = await api.validateProcedure(loadResult.test_name)
    if (validateResult.valid) {
      editorStore.clearProblems()
      executionStore.addLog('Syntax validation passed', 'ok')
    } else {
      editorStore.setProblems(validateResult.errors)
      executionStore.addLog(`Found ${validateResult.errors.length} problems`, 'error')
    }
  } catch (e: any) {
    executionStore.addLog(`Validation error: ${e.message}`, 'error')
  }
}

// === Run (auto-step) ===
async function handleRun() {
  const sessionReady = await ensureStepSession()
  if (!sessionReady) return

  executionStore.setStatus('running')
  executionStore.addLog(`Running test: ${editorStore.testName}`, 'info')
  scheduleNextStep(0)
}

// === Step Execution ===
async function handleStepStart() {
  if (executionStore.status === 'running') {
    stopAutoRun('Paused run mode')
  }

  const sessionReady = await ensureStepSession()
  if (!sessionReady) return

  executionStore.setStatus('stepping')
}

async function handleStepNext() {
  if (!executionStore.sessionId) return
  if (executionStore.status !== 'stepping') return

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
    executionStore.addLog('Step mode reset', 'ok')
    executionStore.setStatus('stepping')
  } catch (e: any) {
    executionStore.addLog(`Reset error: ${e.message}`, 'error')
  }
}

function handleStop() {
  stopAutoRun('Execution stopped')
  executionStore.reset()
}

// Active tab for right panel
const activeTab = ref(0)

async function ensureStepSession(): Promise<boolean> {
  if (!editorStore.testName) {
    await handleValidate()
    if (!editorStore.testName) return false
  }

  if (executionStore.sessionId) return true

  executionStore.addLog(`Starting step mode: ${editorStore.testName}`, 'info')

  try {
    const result = await api.startStepSession(editorStore.testName)
    if (result.success) {
      executionStore.setSessionId(result.session_id)
      executionStore.updateStepState(result.state)
      executionStore.addLog('Step mode ready', 'ok')
      return true
    }

    executionStore.addLog(`Step mode error: ${(result as any).error}`, 'error')
  } catch (e: any) {
    executionStore.addLog(`Step mode error: ${e.message}`, 'error')
  }

  return false
}

function stopAutoRun(message?: string) {
  if (runTimer) {
    clearTimeout(runTimer)
    runTimer = null
  }

  if (message) {
    executionStore.addLog(message, 'warning')
  }
}

function scheduleNextStep(delayMs: number) {
  if (runTimer) clearTimeout(runTimer)
  runTimer = setTimeout(runNextStep, delayMs)
}

async function runNextStep() {
  if (executionStore.status !== 'running') return
  if (!executionStore.sessionId) return

  try {
    const state = await api.stepNext(executionStore.sessionId)
    executionStore.updateStepState(state)

    if (state.status === 'completed') {
      executionStore.setStatus('completed')
      stopAutoRun('Run completed')
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
    executionStore.addLog(`Execution error: ${e.message}`, 'error')
    stopAutoRun()
  }
}

function handlePause() {
  if (executionStore.status !== 'running') return
  stopAutoRun('Run paused')
  executionStore.setStatus('stepping')
}

async function handleResume() {
  if (executionStore.status === 'running') return

  const sessionReady = await ensureStepSession()
  if (!sessionReady) return

  executionStore.setStatus('running')
  executionStore.addLog('Run resumed', 'info')
  scheduleNextStep(0)
}

onUnmounted(() => {
  stopAutoRun()
})
</script>

<template>
  <div class="flex flex-col h-full">
    <!-- Toolbar -->
    <EditorToolbar
      @validate="handleValidate"
      @run="handleRun"
      @pause="handlePause"
      @resume="handleResume"
      @step-start="handleStepStart"
      @step-next="handleStepNext"
      @step-reset="handleStepReset"
      @stop="handleStop"
    />

    <!-- Main Editor Area -->
    <Splitter class="flex-1 overflow-hidden" style="border: none;">
      <SplitterPanel :size="60" :min-size="30">
        <MonacoEditor />
      </SplitterPanel>

      <SplitterPanel :size="40" :min-size="20">
        <TabView v-model:active-index="activeTab" class="h-full flex flex-col">
          <TabPanel header="Console">
            <ConsoleOutput />
          </TabPanel>
          <TabPanel header="Variables">
            <VariablesWatch />
          </TabPanel>
          <TabPanel header="TM">
            <TMPanel />
          </TabPanel>
          <TabPanel header="Call Stack">
            <CallStackPanel />
          </TabPanel>
        </TabView>
      </SplitterPanel>
    </Splitter>

    <!-- Problems Panel -->
    <ProblemsPanel />
  </div>
</template>
