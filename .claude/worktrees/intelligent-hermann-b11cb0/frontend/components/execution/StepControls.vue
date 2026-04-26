<script setup lang="ts">
const executionStore = useExecutionStore()

const emit = defineEmits<{
  'step-start': []
  'step-next': []
  'step-reset': []
  stop: []
}>()
</script>

<template>
  <div class="flex items-center gap-2 p-2">
    <ButtonGroup>
      <Button
        icon="pi pi-step-forward-alt"
        label="Start"
        size="small"
        severity="warn"
        outlined
        :disabled="executionStore.isRunning"
        @click="$emit('step-start')"
      />
      <Button
        icon="pi pi-forward"
        label="Next"
        size="small"
        severity="warn"
        :disabled="!executionStore.isStepping"
        @click="$emit('step-next')"
      />
      <Button
        icon="pi pi-replay"
        label="Reset"
        size="small"
        severity="secondary"
        :disabled="!executionStore.isStepping"
        @click="$emit('step-reset')"
      />
      <Button
        icon="pi pi-stop"
        label="Stop"
        size="small"
        severity="danger"
        :disabled="!executionStore.isRunning"
        @click="$emit('stop')"
      />
    </ButtonGroup>

    <div class="flex-1" />

    <div v-if="executionStore.currentLine > 0" class="text-xs text-muted">
      Line {{ executionStore.currentLine }}
    </div>
  </div>
</template>
