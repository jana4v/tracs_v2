<script setup lang="ts">
definePageMeta({ title: 'TM Monitor' })

const api = useAstraApi()
const telemetryStore = useTelemetryStore()
const polling = ref(false)
let pollTimer: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  fetchTM()
  startPolling()
})

onUnmounted(() => {
  stopPolling()
})

async function fetchTM() {
  try {
    const data = await api.getTMValues()
    telemetryStore.setBanks(data)
  } catch (e) {
    console.error('TM fetch error:', e)
  }
}

function startPolling() {
  if (pollTimer) return
  polling.value = true
  pollTimer = setInterval(fetchTM, telemetryStore.updateInterval)
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
  polling.value = false
}

function togglePolling() {
  if (polling.value) stopPolling()
  else startPolling()
}
</script>

<template>
  <div class="p-4 space-y-4">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-bold text-[var(--astra-text)]">Telemetry Monitor</h1>
      <div class="flex items-center gap-2">
        <Tag :severity="polling ? 'success' : 'secondary'" :value="polling ? 'Live' : 'Paused'" />
        <Button
          :icon="polling ? 'pi pi-pause' : 'pi pi-play'"
          :label="polling ? 'Pause' : 'Resume'"
          :severity="polling ? 'warn' : 'success'"
          size="small"
          @click="togglePolling"
        />
        <Button icon="pi pi-refresh" severity="secondary" size="small" @click="fetchTM" />
      </div>
    </div>

    <!-- TM Bank Cards -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
      <TMBankCard
        v-for="bankId in telemetryStore.bankIds"
        :key="bankId"
        :bank-id="bankId"
        :data="telemetryStore.getBankData(bankId)"
      />
    </div>

    <!-- Empty state -->
    <Card v-if="telemetryStore.bankIds.length === 0" class="panel-card">
      <template #content>
        <div class="text-center text-muted py-8">
          <i class="pi pi-wave-pulse text-4xl mb-2" />
          <p>No telemetry data available.</p>
          <p class="text-sm">Start the Julia backend and initialize simulation data.</p>
        </div>
      </template>
    </Card>
  </div>
</template>
