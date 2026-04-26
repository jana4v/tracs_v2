<script setup lang="ts">
import type { TestResult } from '~/types/astra'

const props = defineProps<{
  result: TestResult
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
  <div class="space-y-4">
    <!-- Summary -->
    <div class="grid grid-cols-2 gap-4">
      <div>
        <label class="text-xs text-muted">Test Name</label>
        <p class="font-mono text-sm">{{ result.test_name }}</p>
      </div>
      <div>
        <label class="text-xs text-muted">Status</label>
        <div>
          <Tag :severity="severityForStatus(result.status)" :value="result.status" />
        </div>
      </div>
      <div>
        <label class="text-xs text-muted">Duration</label>
        <p class="text-sm">{{ result.duration_seconds?.toFixed(3) }}s</p>
      </div>
      <div>
        <label class="text-xs text-muted">Mode</label>
        <div>
          <Tag severity="info" :value="result.mode" />
        </div>
      </div>
      <div>
        <label class="text-xs text-muted">Started</label>
        <p class="text-sm">{{ new Date(result.started_at).toLocaleString() }}</p>
      </div>
      <div>
        <label class="text-xs text-muted">Completed</label>
        <p class="text-sm">{{ new Date(result.completed_at).toLocaleString() }}</p>
      </div>
    </div>

    <!-- Error -->
    <div v-if="result.error">
      <label class="text-xs text-muted">Error</label>
      <Message severity="error" :closable="false" class="text-xs mt-1">
        {{ result.error }}
      </Message>
    </div>

    <!-- Log Entries -->
    <div v-if="result.log_entries?.length">
      <label class="text-xs text-muted">Execution Log ({{ result.log_entries.length }} entries)</label>
      <DataTable
        :value="result.log_entries"
        :rows="10"
        paginator
        striped-rows
        class="text-xs mt-1"
      >
        <Column field="line_number" header="Line" style="width: 60px" />
        <Column field="statement" header="Statement" />
        <Column field="result" header="Result" style="width: 100px" />
        <Column field="status" header="Status" style="width: 80px">
          <template #body="{ data }">
            <Tag :severity="severityForStatus(data.status)" :value="data.status" class="text-xs" />
          </template>
        </Column>
      </DataTable>
    </div>

    <!-- Variables Snapshot -->
    <div v-if="result.variables_snapshot && Object.keys(result.variables_snapshot).length > 0">
      <label class="text-xs text-muted">Final Variables</label>
      <div class="mt-1 space-y-0.5">
        <div
          v-for="[key, value] in Object.entries(result.variables_snapshot)"
          :key="key"
          class="flex items-center justify-between text-xs font-mono px-2 py-1 rounded bg-[var(--astra-border)]/20"
        >
          <span class="text-[var(--astra-variable)]">{{ key }}</span>
          <span>{{ JSON.stringify(value) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>
