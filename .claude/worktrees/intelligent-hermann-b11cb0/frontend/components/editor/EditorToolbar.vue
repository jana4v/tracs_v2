<script setup lang="ts">
const executionStore = useExecutionStore()
const editorStore = useEditorStore()

const emit = defineEmits<{
  validate: []
  run: []
  pause: []
  resume: []
  'step-start': []
  'step-next': []
  'step-reset': []
  stop: []
}>()

function newFile() {
  editorStore.setContent(`TEST_NAME new-test
PRE_TEST_REQ TM1.STATUS == "OK"
SEND START
WAIT 5
`)
  editorStore.setFileName(null)
  editorStore.setTestName(null as any)
  executionStore.addLog('New file created', 'ok')
}

function openFile() {
  const input = document.createElement('input')
  input.type = 'file'
  input.accept = '.tst'
  input.onchange = (e) => {
    const file = (e.target as HTMLInputElement).files?.[0]
    if (!file) return
    const reader = new FileReader()
    reader.onload = (ev) => {
      editorStore.setContent(ev.target?.result as string)
      editorStore.setFileName(file.name)
      executionStore.addLog(`Opened ${file.name}`, 'ok')
    }
    reader.readAsText(file)
  }
  input.click()
}

function saveFile() {
  const blob = new Blob([editorStore.content], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = editorStore.fileName || 'procedure.tst'
  a.click()
  URL.revokeObjectURL(url)
  editorStore.markClean()
  executionStore.addLog('File saved', 'ok')
}
</script>

<template>
  <div class="flex items-center gap-2 px-3 py-2 border-b border-[var(--astra-border)] bg-[var(--astra-surface)]">
    <!-- File Operations -->
    <div class="flex items-center gap-1">
      <Button icon="pi pi-file" size="small" severity="secondary" text v-tooltip.bottom="'New'" @click="newFile" />
      <Button icon="pi pi-folder-open" size="small" severity="secondary" text v-tooltip.bottom="'Open'" @click="openFile" />
      <Button icon="pi pi-save" size="small" severity="secondary" text v-tooltip.bottom="'Save'" @click="saveFile" />
    </div>

    <Divider layout="vertical" class="mx-1 h-6" />

    <!-- Validation & Execution -->
    <div class="flex items-center gap-1">
      <Button
        icon="pi pi-check-circle"
        label="Validate"
        size="small"
        severity="info"
        outlined
        @click="$emit('validate')"
      />
      <Button
        icon="pi pi-play"
        label="Run"
        size="small"
        severity="success"
        :disabled="executionStore.status === 'running'"
        @click="$emit('run')"
      />
      <Button
        icon="pi pi-pause"
        label="Pause"
        size="small"
        severity="secondary"
        outlined
        :disabled="executionStore.status !== 'running'"
        @click="$emit('pause')"
      />
      <Button
        icon="pi pi-play"
        label="Resume"
        size="small"
        severity="success"
        text
        :disabled="executionStore.status !== 'stepping' || !executionStore.sessionId"
        @click="$emit('resume')"
      />
    </div>

    <Divider layout="vertical" class="mx-1 h-6" />

    <!-- Step Controls -->
    <div class="flex items-center gap-1">
      <Button
        icon="pi pi-step-forward-alt"
        label="Step"
        size="small"
        severity="warn"
        outlined
        :disabled="executionStore.isStepping"
        @click="$emit('step-start')"
      />
      <Button
        icon="pi pi-forward"
        size="small"
        severity="warn"
        text
        :disabled="!executionStore.isStepping"
        v-tooltip.bottom="'Next'"
        @click="$emit('step-next')"
      />
      <Button
        icon="pi pi-replay"
        size="small"
        severity="secondary"
        text
        :disabled="!executionStore.isStepping"
        v-tooltip.bottom="'Reset'"
        @click="$emit('step-reset')"
      />
      <Button
        icon="pi pi-stop"
        size="small"
        severity="danger"
        text
        :disabled="!executionStore.isRunning"
        v-tooltip.bottom="'Stop'"
        @click="$emit('stop')"
      />
    </div>

    <!-- Spacer -->
    <div class="flex-1" />

    <div class="hidden md:flex items-center gap-2 text-xs text-[var(--astra-text)]/70">
      <span>Run speed</span>
      <Slider
        v-model="executionStore.runDelayMs"
        :min="50"
        :max="1000"
        :step="50"
        class="w-32"
      />
      <span class="tabular-nums">{{ executionStore.runDelayMs }}ms</span>
    </div>

    <!-- Status -->
    <div class="flex items-center gap-2 text-xs">
      <Tag
        v-if="editorStore.testName"
        severity="info"
        :value="editorStore.testName"
      />
      <Tag
        v-if="editorStore.isDirty"
        severity="warn"
        value="Unsaved"
      />
      <Tag
        v-if="executionStore.isRunning"
        severity="success"
        :value="executionStore.status"
      />
    </div>
  </div>
</template>
