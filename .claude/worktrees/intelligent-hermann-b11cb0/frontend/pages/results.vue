<script setup lang="ts">
import type { TestResult } from '~/types/astra'

definePageMeta({ title: 'Results' })

const api = useAstraApi()
const results = ref<TestResult[]>([])
const loading = ref(true)
const selectedResult = ref<TestResult | null>(null)
const detailVisible = ref(false)

onMounted(async () => {
  await fetchResults()
})

async function fetchResults() {
  loading.value = true
  try {
    const data = await api.getResults(100)
    results.value = data.results
  } catch (e) {
    console.error('Failed to fetch results:', e)
  } finally {
    loading.value = false
  }
}

function showDetail(result: TestResult) {
  selectedResult.value = result
  detailVisible.value = true
}

function severityForStatus(status: string) {
  switch (status) {
    case 'passed': return 'success'
    case 'failed': return 'danger'
    case 'aborted': return 'warn'
    default: return 'secondary'
  }
}
</script>

<template>
  <div class="p-4 space-y-4">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-bold text-[var(--astra-text)]">Test Results</h1>
      <Button icon="pi pi-refresh" label="Refresh" severity="secondary" @click="fetchResults" />
    </div>

    <Card class="panel-card">
      <template #content>
        <DataTable
          :value="results"
          :loading="loading"
          :rows="20"
          paginator
          striped-rows
          sort-field="started_at"
          :sort-order="-1"
          class="text-sm"
          @row-click="(e: any) => showDetail(e.data)"
        >
          <Column field="test_name" header="Test Name" sortable />
          <Column field="status" header="Status" sortable>
            <template #body="{ data }">
              <Tag :severity="severityForStatus(data.status)" :value="data.status" />
            </template>
          </Column>
          <Column field="duration_seconds" header="Duration" sortable>
            <template #body="{ data }">
              {{ data.duration_seconds?.toFixed(2) }}s
            </template>
          </Column>
          <Column field="mode" header="Mode">
            <template #body="{ data }">
              <Tag severity="info" :value="data.mode" />
            </template>
          </Column>
          <Column field="started_at" header="Started" sortable>
            <template #body="{ data }">
              {{ new Date(data.started_at).toLocaleString() }}
            </template>
          </Column>
          <template #empty>
            <div class="text-center text-muted py-8">
              No test results found.
            </div>
          </template>
        </DataTable>
      </template>
    </Card>

    <!-- Detail Dialog -->
    <Dialog
      v-model:visible="detailVisible"
      header="Test Result Detail"
      :style="{ width: '700px' }"
      modal
    >
      <ResultDetail v-if="selectedResult" :result="selectedResult" />
    </Dialog>
  </div>
</template>
