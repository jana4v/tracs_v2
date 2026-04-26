<script setup lang="ts">
import type { TestResult } from '~/types/astra'

const props = defineProps<{
  results: TestResult[]
  loading?: boolean
}>()

const emit = defineEmits<{
  select: [result: TestResult]
}>()

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
  <DataTable
    :value="props.results"
    :loading="props.loading"
    :rows="20"
    paginator
    striped-rows
    sort-field="started_at"
    :sort-order="-1"
    class="text-sm"
    selection-mode="single"
    @row-click="(e: any) => $emit('select', e.data)"
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
      <div class="text-center text-muted py-8">No test results found.</div>
    </template>
  </DataTable>
</template>
