<script setup lang="ts">
const executionStore = useExecutionStore()
const consoleEl = ref<HTMLElement>()

// Auto-scroll to bottom on new log entries
watch(() => executionStore.log.length, () => {
  nextTick(() => {
    if (consoleEl.value) {
      consoleEl.value.scrollTop = consoleEl.value.scrollHeight
    }
  })
})
</script>

<template>
  <div ref="consoleEl" class="h-full overflow-auto p-2 font-mono text-xs bg-[#0d0d1a]">
    <div
      v-for="(entry, idx) in executionStore.log"
      :key="idx"
      class="py-0.5"
      :class="{
        'text-[var(--astra-success)]': entry.type === 'ok',
        'text-[var(--astra-error)]': entry.type === 'error',
        'text-[var(--astra-warning)]': entry.type === 'warning',
        'text-[var(--astra-text)]/70': entry.type === 'info',
      }"
    >
      <span class="text-[var(--astra-text)]/30">[{{ entry.timestamp }}]</span>
      {{ entry.message }}
    </div>

    <div v-if="executionStore.log.length === 0" class="text-[var(--astra-text)]/30 py-4 text-center">
      Console output will appear here...
    </div>
  </div>
</template>
