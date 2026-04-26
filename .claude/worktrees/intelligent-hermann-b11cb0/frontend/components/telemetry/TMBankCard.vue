<script setup lang="ts">
const props = defineProps<{
  bankId: number
  data: Record<string, any>
}>()

const entries = computed(() => {
  return Object.entries(props.data).sort(([a], [b]) => a.localeCompare(b))
})
</script>

<template>
  <Card class="panel-card">
    <template #title>
      <div class="flex items-center gap-2">
        <i class="pi pi-wave-pulse text-[var(--astra-accent)]" />
        <span class="text-sm">TM Bank {{ bankId }}</span>
        <Tag severity="info" :value="`${entries.length} params`" class="text-xs ml-auto" />
      </div>
    </template>
    <template #content>
      <div class="space-y-1">
        <div
          v-for="[key, value] in entries"
          :key="key"
          class="flex items-center justify-between px-2 py-1 rounded text-xs hover:bg-[var(--astra-border)]/30"
        >
          <span class="font-mono text-[var(--astra-variable)]">TM{{ bankId }}.{{ key }}</span>
          <span
            class="font-mono"
            :class="typeof value === 'string' ? 'text-[#ce9178]' : 'text-[#b5cea8]'"
          >
            {{ JSON.stringify(value) }}
          </span>
        </div>
      </div>

      <div v-if="entries.length === 0" class="text-center text-muted text-xs py-2">
        No parameters
      </div>
    </template>
  </Card>
</template>
