<script setup lang="ts">
import * as monaco from 'monaco-editor'
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'

const props = defineProps<{ params: any }>()

const api = useAstraApi()
const schedulerStore = useSchedulerStore()
const monacoSettings = useMonacoSettings()
const { setup: setupMonaco } = useMonaco()

const editorContainer = ref<HTMLElement>()
let editorInstance: monaco.editor.IStandaloneCodeEditor | null = null
let currentDecorations: string[] = []

const row = computed(() => props.params.data)
const loading = ref(false)
const content = ref('')

async function loadContent() {
  if (!row.value?.procedure_name) return
  if (row.value.procedure_content) {
    content.value = row.value.procedure_content
    return
  }

  loading.value = true
  try {
    const proc = await api.getProcedure(row.value.procedure_name)
    const text = proc.latest_content ?? ''
    content.value = text
    schedulerStore.setProcedureContent(row.value.procedure_name, text)
  } catch (e) {
    console.error('Failed to load procedure content:', e)
    content.value = '# Failed to load procedure content'
  } finally {
    loading.value = false
  }
}

async function handleAbort() {
  if (!row.value?.run_id) return
  try { await api.runnerAbort(row.value.run_id) } catch {}
}

async function handlePause() {
  if (!row.value?.run_id) return
  try { await api.runnerPause(row.value.run_id) } catch {}
}

async function handleResume() {
  if (!row.value?.run_id) return
  try { await api.runnerResume(row.value.run_id) } catch {}
}

function handleToggleStepMode() {
  if (!row.value) return
  schedulerStore.toggleStepMode(row.value.procedure_name)
}

function handleRemove() {
  if (!row.value) return
  schedulerStore.removeRow(row.value.procedure_name)
}

onMounted(async () => {
  await loadContent()

  if (!editorContainer.value) return

  const globalScope = window as any
  if (!globalScope.MonacoEnvironment) {
    globalScope.MonacoEnvironment = {
      getWorker: (_: string, _label: string) => new editorWorker(),
    }
  }
  setupMonaco(monaco)

  editorInstance = monaco.editor.create(editorContainer.value, {
    value: content.value,
    language: 'astra',
    theme: monacoSettings.editorTheme,
    automaticLayout: true,
    minimap: { enabled: false },
    fontSize: 12,
    lineNumbers: 'on',
    scrollBeyondLastLine: false,
    wordWrap: 'on',
    readOnly: true,
    glyphMargin: true,
    folding: true,
    renderLineHighlight: 'line',
    domReadOnly: true,
  })
})

watch(
  () => row.value?.current_line,
  (line) => {
    if (!editorInstance || !line || line <= 0) {
      if (editorInstance) {
        currentDecorations = editorInstance.deltaDecorations(currentDecorations, [])
      }
      return
    }
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
  },
)

watch(content, (newContent) => {
  if (editorInstance && editorInstance.getValue() !== newContent) {
    editorInstance.setValue(newContent)
  }
})

// Watch for theme changes
watch(() => monacoSettings.editorTheme, (theme) => {
  monaco.editor.setTheme(theme)
})

onUnmounted(() => {
  editorInstance?.dispose()
})

const isActive = computed(() =>
  row.value?.status === 'running' || row.value?.status === 'paused',
)
const canRemove = computed(() =>
  row.value?.status === 'pending' || row.value?.status === 'completed'
  || row.value?.status === 'failed' || row.value?.status === 'aborted',
)
</script>

<template>
  <div class="flex flex-col" style="height: 340px;">
    <!-- Control Bar -->
    <div class="flex items-center gap-2 px-3 py-2 border-b border-[var(--astra-border)] bg-[#0c1120]">
      <i class="pi pi-file-edit text-xs text-[var(--astra-accent)]" />
      <span class="text-xs font-semibold text-[var(--astra-accent)]">
        {{ row?.procedure_name }}
      </span>

      <Tag
        v-if="row?.status"
        :severity="row?.status === 'completed' ? 'success' : row?.status === 'failed' ? 'danger' : 'info'"
        :value="row?.status"
        class="text-xs ml-2"
      />

      <div class="flex-1" />

      <Button
        v-if="isActive"
        icon="pi pi-stop"
        label="Abort"
        size="small"
        severity="danger"
        outlined
        @click="handleAbort"
      />
      <Button
        v-if="row?.status === 'running'"
        icon="pi pi-pause"
        label="Pause"
        size="small"
        severity="warn"
        outlined
        @click="handlePause"
      />
      <Button
        v-if="row?.status === 'paused'"
        icon="pi pi-play"
        label="Resume"
        size="small"
        severity="success"
        outlined
        @click="handleResume"
      />

      <Divider v-if="isActive" layout="vertical" class="mx-1 h-5" />

      <Button
        v-if="isActive"
        :icon="row?.step_mode ? 'pi pi-stop-circle' : 'pi pi-step-forward-alt'"
        :label="row?.step_mode ? 'Exit Step' : 'Step Mode'"
        size="small"
        :severity="row?.step_mode ? 'warn' : 'secondary'"
        outlined
        @click="handleToggleStepMode"
      />

      <Button
        v-if="canRemove"
        icon="pi pi-trash"
        label="Remove"
        size="small"
        severity="secondary"
        text
        @click="handleRemove"
      />
    </div>

    <!-- Monaco Editor -->
    <div class="flex-1 relative">
      <div v-if="loading" class="absolute inset-0 flex items-center justify-center bg-[#0c1120]/80 z-10">
        <i class="pi pi-spin pi-spinner text-xl text-[var(--astra-accent)]" />
      </div>
      <div ref="editorContainer" class="h-full w-full" />
    </div>

    <!-- Error bar -->
    <div
      v-if="row?.error"
      class="px-3 py-1.5 bg-red-500/10 border-t border-red-500/30 text-xs text-red-400 truncate"
    >
      <i class="pi pi-exclamation-triangle mr-1" />
      {{ row.error }}
    </div>
  </div>
</template>
