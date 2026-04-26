<script setup lang="ts">
import { AgGridVue } from 'ag-grid-vue3'
import type { ColDef, GridApi, GridReadyEvent } from 'ag-grid-community'
import type { SchedulerGridRow } from '~/types/astra'

import PriorityCellRenderer from '~/components/scheduler/PriorityCellRenderer.vue'
import StatusCellRenderer from '~/components/scheduler/StatusCellRenderer.vue'
import ProgressCellRenderer from '~/components/scheduler/ProgressCellRenderer.vue'
import DetailCellRenderer from '~/components/scheduler/DetailCellRenderer.vue'

definePageMeta({ title: 'Test Scheduler' })

const api = useAstraApi()
const settingsStore = useSettingsStore()
const schedulerStore = useSchedulerStore()
const toast = useToast()

let gridApi: GridApi | null = null
const selectedToAdd = ref<string | null>(null)
const newPriority = ref(1)
const testPhaseInput = ref('')
let pollingTimer: ReturnType<typeof setInterval> | null = null

// === Priority filter options ===
const priorityFilterOptions = [
  { label: 'Any Priority', value: null },
  { label: 'High (1-3)', value: 'high' },
  { label: 'Medium (4-6)', value: 'medium' },
  { label: 'Low (7+)', value: 'low' },
]
const filterPriority = ref<string | null>(null)

// === AG Grid Column Definitions ===
const columnDefs = ref<ColDef[]>([
  {
    width: 50,
    pinned: 'left',
    resizable: false,
    sortable: false,
    filter: false,
    suppressHeaderMenuButton: true,
  },
  {
    headerName: 'Test Name',
    field: 'procedure_name',
    flex: 2,
    minWidth: 200,
    cellRenderer: 'agGroupCellRenderer',
    filter: 'agTextColumnFilter',
    sortable: true,
  },
  {
    headerName: 'Priority',
    field: 'priority',
    width: 130,
    cellRenderer: PriorityCellRenderer,
    sortable: true,
  },
  {
    headerName: 'Status',
    field: 'status',
    width: 160,
    cellRenderer: StatusCellRenderer,
    sortable: true,
  },
  {
    headerName: 'Progress',
    field: 'progress',
    width: 200,
    cellRenderer: ProgressCellRenderer,
    sortable: true,
  },
  {
    headerName: 'Assigned Operator',
    field: 'operator',
    width: 160,
    editable: (params: any) => params.data?.status === 'pending',
    sortable: true,
  },
  {
    headerName: 'Start Time',
    field: 'start_time',
    width: 140,
    valueFormatter: (params: any) => {
      if (!params.value) return '--'
      return new Date(params.value).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
    },
    sortable: true,
  },
  {
    headerName: 'End Time',
    field: 'end_time',
    width: 140,
    valueFormatter: (params: any) => {
      if (!params.value) return '--'
      return new Date(params.value).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
    },
    sortable: true,
  },
])

const defaultColDef: ColDef = {
  resizable: true,
  suppressMovable: false,
}

// === Grid Events ===
function onGridReady(event: GridReadyEvent) {
  gridApi = event.api
  gridApi.sizeColumnsToFit()
}

function onCellValueChanged(event: any) {
  if (event.colDef.field === 'operator') {
    schedulerStore.setOperator(event.data.procedure_name, event.newValue)
  }
}

// === Available Procedures ===
async function loadAvailableProcedures() {
  schedulerStore.loading = true
  try {
    const result = await api.getProcedures()
    schedulerStore.availableProcedures = result.procedures
  } catch (e: any) {
    toast.add({ severity: 'error', summary: 'Error', detail: `Failed to load procedures: ${e.message}`, life: 3000 })
  } finally {
    schedulerStore.loading = false
  }
}

const addableProcedures = computed(() => {
  const scheduled = new Set(schedulerStore.rows.map(r => r.procedure_name))
  return schedulerStore.availableProcedures
    .filter(p => !scheduled.has(p.test_name))
    .map(p => ({ label: `${p.test_name} (v${p.latest_version})`, value: p.test_name }))
})

// === Add Procedure ===
function addToScheduler() {
  if (!selectedToAdd.value) return
  schedulerStore.addRow(
    selectedToAdd.value,
    newPriority.value,
    settingsStore.username || '',
  )
  selectedToAdd.value = null
}

// === Remove Selected ===
function removeSelected() {
  if (!gridApi) return
  const selected = gridApi.getSelectedRows() as SchedulerGridRow[]
  for (const row of selected) {
    schedulerStore.removeRow(row.procedure_name)
  }
}

// === Execution ===
async function executeSchedule() {
  if (schedulerStore.rows.length === 0) {
    toast.add({ severity: 'warn', summary: 'Empty', detail: 'Add procedures first', life: 3000 })
    return
  }

  // Sync test phase to Redis
  if (testPhaseInput.value) {
    try {
      await api.setTestPhase(testPhaseInput.value)
      schedulerStore.testPhase = testPhaseInput.value
    } catch (e: any) {
      toast.add({ severity: 'error', summary: 'Phase Error', detail: e.message, life: 3000 })
    }
  }

  schedulerStore.isExecuting = true
  const groups = schedulerStore.priorityGroups

  for (const priority of groups) {
    schedulerStore.currentPriority = priority
    const entries = schedulerStore.rows.filter(
      r => r.priority === priority && r.status === 'pending',
    )
    if (entries.length === 0) continue

    // Load all procedures in this priority group
    for (const entry of entries) {
      try {
        const proc = await api.getProcedure(entry.procedure_name)
        const content = proc.latest_content ?? ''
        if (content) {
          await api.loadProcedure(content, `${entry.procedure_name}.tst`)
        }
      } catch (e: any) {
        schedulerStore.setEntryStatus(entry.procedure_name, 'failed')
        toast.add({ severity: 'error', summary: 'Load Error', detail: `${entry.procedure_name}: ${e.message}`, life: 5000 })
      }
    }

    // Start all non-failed procedures in parallel
    const pendingEntries = entries.filter(e => e.status !== 'failed')
    for (const entry of pendingEntries) {
      try {
        schedulerStore.setEntryStatus(entry.procedure_name, 'queued')
        const result = await api.runnerStart(entry.procedure_name)
        if (result.success) {
          schedulerStore.setRunId(entry.procedure_name, result.run_id)
        } else {
          schedulerStore.setEntryStatus(entry.procedure_name, 'failed')
        }
      } catch (e: any) {
        schedulerStore.setEntryStatus(entry.procedure_name, 'failed')
      }
    }

    // Wait for all runs in this group to complete
    await waitForGroupCompletion(pendingEntries)
  }

  schedulerStore.isExecuting = false
  schedulerStore.currentPriority = null
  toast.add({ severity: 'info', summary: 'Complete', detail: 'All priority groups finished', life: 5000 })
}

async function waitForGroupCompletion(entries: SchedulerGridRow[]) {
  const runIds = entries.map(e => e.run_id).filter(Boolean) as string[]
  if (runIds.length === 0) return

  return new Promise<void>((resolve) => {
    const check = setInterval(async () => {
      let allDone = true
      for (const runId of runIds) {
        try {
          const status = await api.runnerStatus(runId)
          schedulerStore.handleRunnerUpdate(status as any)
          if (!['completed', 'failed', 'aborted'].includes(status.status)) {
            allDone = false
          }
        } catch { /* Run not found, treat as done */ }
      }
      if (allDone) {
        clearInterval(check)
        resolve()
      }
    }, 500)
  })
}

// === Batch Controls ===
async function pauseAll() {
  const running = schedulerStore.rows.filter(r => r.run_id && r.status === 'running')
  for (const row of running) {
    try { await api.runnerPause(row.run_id!) } catch {}
  }
}

async function stopAll() {
  const active = schedulerStore.rows.filter(
    r => r.run_id && (r.status === 'running' || r.status === 'paused'),
  )
  for (const row of active) {
    try { await api.runnerAbort(row.run_id!) } catch {}
  }
  schedulerStore.isExecuting = false
}

// === Test Phase ===
async function loadTestPhase() {
  try {
    const result = await api.getTestPhase()
    testPhaseInput.value = result.test_phase
    schedulerStore.testPhase = result.test_phase
  } catch { /* Redis not available */ }
}

// === Status Polling ===
function startPolling() {
  if (pollingTimer) return
  pollingTimer = setInterval(async () => {
    const active = schedulerStore.rows.filter(
      r => r.run_id && (r.status === 'running' || r.status === 'paused'),
    )
    for (const row of active) {
      try {
        const status = await api.runnerStatus(row.run_id!)
        schedulerStore.handleRunnerUpdate(status as any)
      } catch { /* ignore */ }
    }
    if (gridApi) gridApi.refreshCells({ force: true })
  }, 1000)
}

function stopPolling() {
  if (pollingTimer) {
    clearInterval(pollingTimer)
    pollingTimer = null
  }
}

// === Lifecycle ===
watch(() => schedulerStore.isExecuting, (executing) => {
  if (executing) startPolling()
  else stopPolling()
})

onMounted(async () => {
  await loadAvailableProcedures()
  await loadTestPhase()
  await schedulerStore.restoreFromBackend()

  const hasActive = schedulerStore.rows.some(
    r => r.status === 'running' || r.status === 'paused',
  )
  if (hasActive) {
    schedulerStore.isExecuting = true
    startPolling()
  }
})

onUnmounted(() => stopPolling())
</script>

<template>
  <div class="flex flex-col h-full gap-3 p-4">
    <!-- Top Toolbar -->
    <div class="panel-card p-3">
      <div class="flex items-center gap-3 flex-wrap">
        <!-- Add procedure -->
        <Select
          v-model="selectedToAdd"
          :options="addableProcedures"
          option-label="label"
          option-value="value"
          placeholder="Filter Tests..."
          class="w-64"
          size="small"
          filter
          :loading="schedulerStore.loading"
          :pt="{
            root: { class: 'bg-[var(--astra-surface-2)]' },
            panel: { class: 'bg-[var(--astra-surface-2)] border-[var(--astra-border)]' },
            item: { class: 'text-[var(--astra-text)] hover:bg-[var(--astra-accent)]/10' },
          }"
        />

        <!-- Priority filter -->
        <Select
          v-model="filterPriority"
          :options="priorityFilterOptions"
          option-label="label"
          option-value="value"
          placeholder="Any Priority"
          class="w-40"
          size="small"
          :pt="{
            root: { class: 'bg-[var(--astra-surface-2)]' },
            panel: { class: 'bg-[var(--astra-surface-2)] border-[var(--astra-border)]' },
            item: { class: 'text-[var(--astra-text)] hover:bg-[var(--astra-accent)]/10' },
          }"
        />

        <!-- Priority input -->
        <div class="flex items-center gap-1.5">
          <span class="text-xs text-[var(--astra-muted)] whitespace-nowrap">Priority:</span>
          <InputNumber
            v-model="newPriority"
            :min="1"
            :max="99"
            size="small"
            class="w-20"
            show-buttons
            button-layout="horizontal"
            :step="1"
            increment-button-icon="pi pi-plus"
            decrement-button-icon="pi pi-minus"
          />
        </div>

        <!-- Test Phase -->
        <div class="flex items-center gap-1.5">
          <span class="text-xs text-[var(--astra-muted)] whitespace-nowrap">Phase:</span>
          <InputText
            v-model="testPhaseInput"
            placeholder="e.g. Phase-2A"
            size="small"
            class="w-28"
          />
        </div>

        <div class="flex-1" />

        <!-- Action Buttons -->
        <Button
          icon="pi pi-plus"
          label="Add"
          size="small"
          severity="secondary"
          outlined
          :disabled="!selectedToAdd"
          @click="addToScheduler"
        />
        <Button
          icon="pi pi-trash"
          size="small"
          severity="secondary"
          text
          v-tooltip.bottom="'Remove selected'"
          @click="removeSelected"
        />

        <Divider layout="vertical" class="mx-0 h-6" />

        <Button
          icon="pi pi-play"
          label="Schedule"
          size="small"
          :disabled="schedulerStore.rows.length === 0 || schedulerStore.isExecuting"
          class="!bg-blue-600 !border-blue-600 hover:!bg-blue-700 !text-white"
          @click="executeSchedule"
        />
        <Button
          icon="pi pi-pause"
          label="Pause"
          size="small"
          severity="secondary"
          outlined
          :disabled="schedulerStore.activeRunCount === 0"
          @click="pauseAll"
        />
        <Button
          icon="pi pi-stop"
          label="Stop"
          size="small"
          severity="danger"
          :disabled="schedulerStore.activeRunCount === 0"
          @click="stopAll"
        />
      </div>
    </div>

    <!-- AG Grid Table -->
    <div class="flex-1 panel-card overflow-hidden rounded-2xl">
      <AgGridVue
        class="ag-theme-alpine-dark h-full w-full"
        :row-data="schedulerStore.rows"
        :column-defs="columnDefs"
        :default-col-def="defaultColDef"
        :row-selection="{ mode: 'multiRow', headerCheckbox: true, checkboxes: true, enableClickSelection: false }"
        :animate-rows="true"
        :master-detail="true"
        :detail-cell-renderer="DetailCellRenderer"
        :detail-row-height="340"
        :detail-row-auto-height="false"
        :get-row-id="(params: any) => params.data.id"
        :row-height="48"
        :header-height="42"
        @grid-ready="onGridReady"
        @cell-value-changed="onCellValueChanged"
      />
    </div>

    <!-- Summary Footer -->
    <div
      v-if="schedulerStore.rows.length > 0"
      class="flex items-center justify-between px-4 py-2.5 panel-card text-xs"
    >
      <div class="flex items-center gap-5 text-[var(--astra-muted)]">
        <span>
          <span class="font-semibold text-[var(--astra-text)]">{{ schedulerStore.rows.length }}</span>
          Tests
        </span>
        <span>
          <span class="font-semibold text-[var(--astra-text)]">{{ schedulerStore.priorityGroups.length }}</span>
          Groups
        </span>
        <span>
          <span class="font-semibold text-cyan-400">{{ schedulerStore.activeRunCount }}</span>
          Active
        </span>
        <span v-if="schedulerStore.testPhase" class="flex items-center gap-1">
          Phase:
          <span class="font-semibold text-[var(--astra-accent)]">{{ schedulerStore.testPhase }}</span>
        </span>
      </div>
      <div class="flex items-center gap-4 text-[var(--astra-muted)]">
        <span class="flex items-center gap-1.5">
          <i class="pi pi-check-circle text-emerald-400 text-xs" />
          <span class="font-semibold text-emerald-400">{{ schedulerStore.completedCount }}</span>
          Passed
        </span>
        <span class="flex items-center gap-1.5">
          <i class="pi pi-times-circle text-red-400 text-xs" />
          <span class="font-semibold text-red-400">{{ schedulerStore.failedCount }}</span>
          Failed
        </span>
        <span class="flex items-center gap-1.5">
          <i class="pi pi-clock text-slate-400 text-xs" />
          <span class="font-semibold text-slate-400">{{ schedulerStore.pendingCount }}</span>
          Pending
        </span>
      </div>
    </div>
  </div>
</template>

