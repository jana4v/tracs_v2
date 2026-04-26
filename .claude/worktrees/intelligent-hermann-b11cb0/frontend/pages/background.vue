<script setup lang="ts">
import type { BackgroundEntry, BackgroundRegisterRequest } from '~/types/astra'

definePageMeta({ title: 'Background' })

const bgStore = useBackgroundStore()
const { isConnected, onMessage } = useWebSocket()

// ── Register dialog ───────────────────────────────────────
const showRegister = ref(false)
const regForm = reactive<BackgroundRegisterRequest>({
  proc_name: '',
  schedule_type: 'interval',
  interval_seconds: 1,
  restart_on_failure: true,
  max_consecutive_failures: 10,
  condition: '',
  poll_interval: 0.5,
})

async function submitRegister() {
  await bgStore.register({ ...regForm })
  showRegister.value = false
  regForm.proc_name = ''
}

// ── AG Grid ───────────────────────────────────────────────
const gridApi = ref<any>(null)

const columnDefs = [
  {
    field: 'proc_name',
    headerName: 'Procedure',
    flex: 2,
    cellRenderer: (params: any) => `<span style="font-family:monospace">${params.value}</span>`,
  },
  {
    headerName: 'Schedule',
    flex: 1,
    valueGetter: (params: any) => {
      const s = params.data.schedule
      if (s.type === 'interval') return `Every ${s.interval_seconds}s`
      return `Event: ${s.condition}`
    },
  },
  {
    field: 'status',
    headerName: 'Status',
    width: 110,
    cellRenderer: (params: any) => {
      const map: Record<string, string> = {
        running: '<span class="bg-status running">● running</span>',
        idle: '<span class="bg-status idle">○ idle</span>',
        failed: '<span class="bg-status failed">✕ failed</span>',
        stopped: '<span class="bg-status stopped">■ stopped</span>',
      }
      return map[params.value] ?? params.value
    },
  },
  {
    field: 'last_run_at',
    headerName: 'Last Run',
    flex: 1,
    valueFormatter: (p: any) => p.value ? new Date(p.value).toLocaleTimeString() : '—',
  },
  {
    field: 'next_run_at',
    headerName: 'Next Run',
    flex: 1,
    valueFormatter: (p: any) => p.value ? new Date(p.value).toLocaleTimeString() : '—',
  },
  {
    field: 'total_runs',
    headerName: 'Runs',
    width: 80,
  },
  {
    field: 'consecutive_failures',
    headerName: 'Err',
    width: 70,
    cellRenderer: (params: any) => {
      const n = params.value as number
      return n > 0
        ? `<span style="color:#f87171;font-weight:600">${n}</span>`
        : `<span style="color:#4ade80">${n}</span>`
    },
  },
  {
    headerName: 'Actions',
    width: 100,
    cellRenderer: (params: any) => {
      const s = params.data.status
      if (s === 'running' || s === 'idle') {
        return `<button class="grid-action-btn stop-btn" data-proc="${params.data.proc_name}">■ Stop</button>`
      }
      return `<button class="grid-action-btn start-btn" data-proc="${params.data.proc_name}">▶ Start</button>`
    },
  },
]

const defaultColDef = { sortable: true, resizable: true }

function onGridReady(params: any) {
  gridApi.value = params.api
}

function onCellClicked(event: any) {
  const target = event.event?.target as HTMLElement | null
  if (!target) return
  const proc = target.getAttribute('data-proc')
  if (!proc) return
  if (target.classList.contains('stop-btn')) {
    bgStore.stopOne(proc)
  }
  else if (target.classList.contains('start-btn')) {
    bgStore.startOne(proc)
  }
}

// ── WebSocket integration ─────────────────────────────────
onMessage((event) => {
  try {
    const msg = JSON.parse(event.data)
    if (msg.type === 'background_update' && msg.data) {
      bgStore.handleWsUpdate(msg.data)
      // Refresh the row in the grid
      if (gridApi.value) {
        gridApi.value.applyTransactionAsync({ update: [msg.data] })
      }
    }
  }
  catch { /* ignore non-JSON */ }
})

// ── Lifecycle ─────────────────────────────────────────────
onMounted(() => bgStore.fetchList())
</script>

<template>
  <div class="bg-page">
    <!-- Header bar -->
    <div class="bg-header">
      <div class="bg-summary">
        <span class="summary-chip running">● {{ bgStore.runningCount }} running</span>
        <span class="summary-chip idle">○ {{ bgStore.idleCount }} idle</span>
        <span class="summary-chip stopped">■ {{ bgStore.stoppedCount }} stopped</span>
        <span class="summary-chip failed">✕ {{ bgStore.failedCount }} failed</span>
        <span class="summary-total">/ {{ bgStore.totalCount }} total</span>
      </div>

      <div class="bg-actions">
        <Button
          label="Register"
          icon="pi pi-plus"
          size="small"
          @click="showRegister = true"
        />
        <Button
          label="Start All"
          icon="pi pi-play"
          severity="success"
          size="small"
          :disabled="bgStore.totalCount === 0"
          @click="bgStore.startAll()"
        />
        <Button
          label="Stop All"
          icon="pi pi-stop"
          severity="danger"
          size="small"
          :disabled="bgStore.runningCount + bgStore.idleCount === 0"
          @click="bgStore.stopAll()"
        />
        <Button
          icon="pi pi-refresh"
          size="small"
          text
          :loading="bgStore.loading"
          @click="bgStore.fetchList()"
        />
        <Tag :severity="isConnected ? 'success' : 'danger'" :value="isConnected ? 'Live' : 'Offline'" />
      </div>
    </div>

    <!-- AG Grid -->
    <div class="grid-wrap ag-theme-alpine-dark">
      <AgGridVue
        :row-data="bgStore.entries"
        :column-defs="columnDefs"
        :default-col-def="defaultColDef"
        :get-row-id="(p: any) => p.data.proc_name"
        animate-rows
        style="height: 100%; width: 100%;"
        @grid-ready="onGridReady"
        @cell-clicked="onCellClicked"
      />
    </div>

    <!-- Register dialog -->
    <Dialog
      v-model:visible="showRegister"
      header="Register Background Procedure"
      :modal="true"
      :style="{ width: '480px' }"
    >
      <div class="reg-form">
        <div class="field">
          <label>Procedure Name</label>
          <InputText v-model="regForm.proc_name" class="w-full" placeholder="e.g. tlm-health-monitor" />
        </div>

        <div class="field">
          <label>Schedule Type</label>
          <SelectButton
            v-model="regForm.schedule_type"
            :options="[{ label: 'Interval', value: 'interval' }, { label: 'Event-Driven', value: 'event' }]"
            option-label="label"
            option-value="value"
            :allow-empty="false"
          />
        </div>

        <template v-if="regForm.schedule_type === 'interval'">
          <div class="field">
            <label>Interval (seconds)</label>
            <InputNumber v-model="regForm.interval_seconds" :min="0.1" :max="3600" :step="0.1" class="w-full" />
          </div>
        </template>

        <template v-else>
          <div class="field">
            <label>Condition</label>
            <InputText v-model="regForm.condition" class="w-full" placeholder='e.g. TM.MODE == "SAFE"' />
          </div>
          <div class="field">
            <label>Poll Interval (seconds)</label>
            <InputNumber v-model="regForm.poll_interval" :min="0.1" :max="60" :step="0.1" class="w-full" />
          </div>
        </template>

        <div class="field row">
          <div class="field half">
            <label>Max Consecutive Failures</label>
            <InputNumber v-model="regForm.max_consecutive_failures" :min="0" :max="1000" class="w-full" />
          </div>
          <div class="field half check-field">
            <Checkbox v-model="regForm.restart_on_failure" binary />
            <label>Restart on failure</label>
          </div>
        </div>
      </div>

      <template #footer>
        <Button label="Cancel" text @click="showRegister = false" />
        <Button
          label="Register"
          icon="pi pi-check"
          :disabled="!regForm.proc_name"
          @click="submitRegister"
        />
      </template>
    </Dialog>
  </div>
</template>

<style scoped lang="scss">
.bg-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  gap: 0.75rem;
}

.bg-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 0.5rem;
  padding: 0.5rem 0;
}

.bg-summary {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.summary-chip {
  font-size: 0.8rem;
  font-weight: 600;
  padding: 0.2rem 0.6rem;
  border-radius: 999px;

  &.running { color: #4ade80; background: rgba(74,222,128,.12); }
  &.idle     { color: #94a3b8; background: rgba(148,163,184,.1); }
  &.stopped  { color: #64748b; background: rgba(100,116,139,.1); }
  &.failed   { color: #f87171; background: rgba(248,113,113,.12); }
}

.summary-total {
  font-size: 0.8rem;
  color: #94a3b8;
}

.bg-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.grid-wrap {
  flex: 1;
  min-height: 0;
  border-radius: 8px;
  overflow: hidden;
}

/* Cell renderer styles (not scoped — applied globally via :global) */
:global(.bg-status) {
  font-size: 0.78rem;
  font-weight: 600;
  padding: 0.15rem 0.45rem;
  border-radius: 4px;
}
:global(.bg-status.running) { color: #4ade80; }
:global(.bg-status.idle)    { color: #94a3b8; }
:global(.bg-status.failed)  { color: #f87171; }
:global(.bg-status.stopped) { color: #64748b; }

:global(.grid-action-btn) {
  font-size: 0.75rem;
  padding: 0.2rem 0.55rem;
  border-radius: 4px;
  border: 1px solid;
  cursor: pointer;
  font-weight: 600;
  background: transparent;
}
:global(.start-btn) { color: #4ade80; border-color: #4ade80; }
:global(.stop-btn)  { color: #f87171; border-color: #f87171; }
:global(.start-btn:hover) { background: rgba(74,222,128,.15); }
:global(.stop-btn:hover)  { background: rgba(248,113,113,.15); }

.reg-form {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 0.25rem 0;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;

  label { font-size: 0.8rem; color: #94a3b8; }

  &.row {
    flex-direction: row;
    align-items: flex-end;
    gap: 1rem;
  }
  &.half { flex: 1; }
  &.check-field {
    flex-direction: row;
    align-items: center;
    gap: 0.5rem;
    padding-bottom: 0.25rem;
  }
}
</style>
