<script setup lang="ts">
const executionStore = useExecutionStore()
</script>

<template>
  <div class="h-full overflow-auto p-2">
    <div v-if="executionStore.callStack.length > 0" class="space-y-1">
      <div
        v-for="(frame, idx) in executionStore.callStack"
        :key="idx"
        class="flex items-center gap-2 px-2 py-1 rounded text-xs font-mono"
        :class="idx === 0 ? 'bg-[var(--astra-accent)]/10 text-[var(--astra-accent)]' : 'text-[var(--astra-text)]/70'"
      >
        <i class="pi pi-chevron-right text-xs" />
        <span>{{ frame }}</span>
      </div>
    </div>

    <div v-else class="text-center text-muted text-xs py-4">
      No active call stack. Start step execution to see the call stack.
    </div>
  </div>
</template>
