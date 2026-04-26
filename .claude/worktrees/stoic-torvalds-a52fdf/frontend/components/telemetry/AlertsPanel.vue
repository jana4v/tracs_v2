<script setup lang="ts">
const executionStore = useExecutionStore()

const alerts = computed(() => {
  return executionStore.log.filter(entry => entry.type === 'warning')
})
</script>

<template>
  <div class="space-y-2 p-2">
    <Message
      v-for="(alert, idx) in alerts"
      :key="idx"
      severity="warn"
      :closable="false"
      class="text-xs"
    >
      <span class="text-xs text-[var(--astra-text)]/30 mr-2">[{{ alert.timestamp }}]</span>
      {{ alert.message }}
    </Message>

    <div v-if="alerts.length === 0" class="text-center text-muted text-xs py-4">
      No alerts.
    </div>
  </div>
</template>
