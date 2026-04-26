<script setup lang="ts">
definePageMeta({ title: 'Dashboard' })

const api = useAstraApi()
const proceduresStore = useProceduresStore()
const telemetryStore = useTelemetryStore()

const recentResults = ref<any[]>([])
const systemStatus = ref({
  backendConnected: false,
  proceduresLoaded: 0,
  tmBanksActive: 0,
  mode: 'simulation',
})

onMounted(async () => {
  try {
    const [procData, tmData, resultData] = await Promise.allSettled([
      api.getProcedures(),
      api.getTMValues(),
      api.getResults(10),
    ])

    if (procData.status === 'fulfilled') {
      proceduresStore.setVersionedList(procData.value.procedures)
      systemStatus.value.proceduresLoaded = procData.value.procedures.length
      systemStatus.value.backendConnected = true
    }

    if (tmData.status === 'fulfilled') {
      telemetryStore.setBanks(tmData.value)
      systemStatus.value.tmBanksActive = telemetryStore.bankIds.length
    }

    if (resultData.status === 'fulfilled') {
      recentResults.value = resultData.value.results
    }
  } catch (e) {
    console.error('Dashboard load error:', e)
  }
})
</script>

<template>
  <div class="p-4 space-y-4">
    <h1 class="text-2xl font-bold text-[var(--astra-text)]">ASTRA Dashboard</h1>

    <!-- Status Cards -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      <Card class="panel-card">
        <template #title>
          <div class="flex items-center gap-2">
            <i class="pi pi-server text-[var(--astra-accent)]" />
            <span class="text-sm">Backend Status</span>
          </div>
        </template>
        <template #content>
          <Tag
            :severity="systemStatus.backendConnected ? 'success' : 'danger'"
            :value="systemStatus.backendConnected ? 'Online' : 'Offline'"
          />
        </template>
      </Card>

      <Card class="panel-card">
        <template #title>
          <div class="flex items-center gap-2">
            <i class="pi pi-file text-[var(--astra-accent)]" />
            <span class="text-sm">Procedures</span>
          </div>
        </template>
        <template #content>
          <span class="text-2xl font-bold">{{ systemStatus.proceduresLoaded }}</span>
          <span class="text-sm text-muted ml-1">loaded</span>
        </template>
      </Card>

      <Card class="panel-card">
        <template #title>
          <div class="flex items-center gap-2">
            <i class="pi pi-wave-pulse text-[var(--astra-accent)]" />
            <span class="text-sm">TM Banks</span>
          </div>
        </template>
        <template #content>
          <span class="text-2xl font-bold">{{ systemStatus.tmBanksActive }}</span>
          <span class="text-sm text-muted ml-1">active</span>
        </template>
      </Card>

      <Card class="panel-card">
        <template #title>
          <div class="flex items-center gap-2">
            <i class="pi pi-cog text-[var(--astra-accent)]" />
            <span class="text-sm">Mode</span>
          </div>
        </template>
        <template #content>
          <Tag severity="info" :value="systemStatus.mode" />
        </template>
      </Card>
    </div>

    <!-- Recent Test Results -->
    <Card class="panel-card">
      <template #title>
        <div class="flex items-center gap-2">
          <i class="pi pi-history" />
          <span>Recent Test Results</span>
        </div>
      </template>
      <template #content>
        <DataTable
          :value="recentResults"
          :rows="10"
          striped-rows
          class="text-sm"
        >
          <Column field="test_name" header="Test Name" />
          <Column field="status" header="Status">
            <template #body="{ data }">
              <Tag
                :severity="data.status === 'passed' ? 'success' : data.status === 'failed' ? 'danger' : 'warn'"
                :value="data.status"
              />
            </template>
          </Column>
          <Column field="duration_seconds" header="Duration">
            <template #body="{ data }">
              {{ data.duration_seconds?.toFixed(2) }}s
            </template>
          </Column>
          <Column field="started_at" header="Date">
            <template #body="{ data }">
              {{ new Date(data.started_at).toLocaleString() }}
            </template>
          </Column>
          <template #empty>
            <div class="text-center text-muted py-4">
              No test results yet. Run a test from the Editor.
            </div>
          </template>
        </DataTable>
      </template>
    </Card>
  </div>
</template>
