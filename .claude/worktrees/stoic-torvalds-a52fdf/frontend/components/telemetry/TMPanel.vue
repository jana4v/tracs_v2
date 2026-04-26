<script setup lang="ts">
const api = useAstraApi()
const telemetryStore = useTelemetryStore()
let pollTimer: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  fetchTM()
  pollTimer = setInterval(fetchTM, 2000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})

async function fetchTM() {
  try {
    const data = await api.getTMValues()
    telemetryStore.setBanks(data)
  } catch {
    // Silently fail
  }
}

const tmEntries = computed(() => {
  return Object.entries(telemetryStore.banks)
    .sort(([a], [b]) => a.localeCompare(b))
    .slice(0, 20) // Show max 20 in compact view
})
</script>

<template>
  <div class="h-full overflow-auto p-2">
    <div class="space-y-0.5">
      <div
        v-for="[key, value] in tmEntries"
        :key="key"
        class="flex items-center justify-between px-2 py-0.5 rounded text-xs hover:bg-[var(--astra-border)]/30"
      >
        <span class="font-mono text-[var(--astra-variable)]">{{ key }}</span>
        <span class="font-mono">{{ JSON.stringify(value) }}</span>
      </div>
    </div>

    <div v-if="tmEntries.length === 0" class="text-center text-muted text-xs py-4">
      No TM data available.
    </div>

    <div v-if="Object.keys(telemetryStore.banks).length > 20" class="text-center text-xs text-muted mt-2">
      <NuxtLink to="/tm-monitor" class="text-[var(--astra-accent)] hover:underline">
        View all {{ Object.keys(telemetryStore.banks).length }} parameters →
      </NuxtLink>
    </div>
  </div>
</template>
