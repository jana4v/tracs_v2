<script setup lang="ts">
import AutoComplete from 'primevue/autocomplete'
import type { SimulatorMnemonic } from '~/types/astra'

definePageMeta({ title: 'TM Simulator' })

const store = useSimulatorStore()
const toast = useToast()
const api   = useAstraApi()

const simulationMode = ref<'FIXED' | 'RANDOM'>('FIXED')
const modeLoading = ref(false)
const simulatorRunning = ref(true)
const simulatorLoading = ref(false)
const searchValue = ref('')

async function toggleSimulationMode() {
  const newMode = simulationMode.value === 'FIXED' ? 'RANDOM' : 'FIXED'
  modeLoading.value = true
  try {
    await api.setSimulatorMode(newMode)
    simulationMode.value = newMode
    toast.add({ severity: 'info', summary: 'Mode Changed', detail: `Simulation mode set to ${newMode}`, life: 3000 })
  } catch (e: any) {
    toast.add({ severity: 'error', summary: 'Failed', detail: e.message || 'Failed to change mode', life: 3000 })
  } finally {
    modeLoading.value = false
  }
}

async function startSimulator() {
  simulatorLoading.value = true
  try {
    await api.startSimulator()
    simulatorRunning.value = true
    store.loadStatus()
    await store.loadSubsystems()
    if (store.selectedSubsystems.length > 0) {
      await store.loadMnemonicsWithRange()
    }
    setTimeout(async () => {
      await store.loadValues()
    }, 500)
    toast.add({ severity: 'success', summary: 'Started', detail: 'Simulator started', life: 3000 })
  } catch (e: any) {
    toast.add({ severity: 'error', summary: 'Failed', detail: e.message || 'Failed to start simulator', life: 3000 })
  } finally {
    simulatorLoading.value = false
  }
}

async function stopSimulator() {
  simulatorLoading.value = true
  try {
    await api.stopSimulator()
    simulatorRunning.value = false
    store.loadStatus()
    setTimeout(async () => {
      await store.loadValues()
    }, 500)
    toast.add({ severity: 'info', summary: 'Stopped', detail: 'Simulator stopped and values cleared', life: 3000 })
  } catch (e: any) {
    toast.add({ severity: 'error', summary: 'Failed', detail: e.message || 'Failed to stop simulator', life: 3000 })
  } finally {
    simulatorLoading.value = false
  }
}

// === Inline editing state ===
const editingMnemonic    = ref<string | null>(null)
const editValue          = ref('')
const filteredSuggestions = ref<string[]>([])

const hasChanges = computed(() => store.hasChanges)
const valueFilteredMnemonics = computed(() => {
  if (!searchValue.value) return store.filteredMnemonics
  const q = searchValue.value.toLowerCase()
  return store.filteredMnemonics.filter(m => 
    m.value?.toLowerCase().includes(q)
  )
})
watch(hasChanges, (val) => console.log('hasChanges changed:', val))

function isChanged(row: SimulatorMnemonic) {
  return store.originalValues[row.mnemonic] !== undefined
    && row.value !== store.originalValues[row.mnemonic]
}

async function startEdit(row: SimulatorMnemonic) {
  editingMnemonic.value    = row.mnemonic
  editValue.value          = row.value
  
  if (row.range && row.range.length > 0) {
    filteredSuggestions.value = row.range
  } else {
    try {
      const result = await api.getMnemonicRange(row.mnemonic)
      if (result && result.range) {
        row.range = result.range
        filteredSuggestions.value = result.range
      } else {
        filteredSuggestions.value = []
      }
    } catch (e) {
      console.error('Failed to fetch range for', row.mnemonic, e)
      filteredSuggestions.value = []
    }
  }
}

function commitEdit(row: SimulatorMnemonic, event?: any) {
  if (editingMnemonic.value !== row.mnemonic) return
  // PrimeVue option-select passes { value } — prefer it over v-model (which may lag behind blur)
  const newValue = (event?.value !== undefined) ? String(event.value) : editValue.value
  console.log('commitEdit:', row.mnemonic, newValue)
  store.updateValue(row.mnemonic, newValue)
  editingMnemonic.value = null
}

function onBlur(row: SimulatorMnemonic) {
  // Delay so @option-select fires first when clicking a dropdown item (append-to="body" blur race)
  setTimeout(() => commitEdit(row), 150)
}

function cancelEdit(row: SimulatorMnemonic) {
  if (editingMnemonic.value !== row.mnemonic) return
  editValue.value       = row.value   // revert
  editingMnemonic.value = null
}

function onComplete(event: { query: string }, range: string[]) {
  const q = event.query.trim().toLowerCase()
  filteredSuggestions.value = q
    ? range.filter(v => v.toLowerCase().includes(q))
    : [...range]
}

// === Actions ===
async function simulate() {
  const changes = store.changedMnemonics || []
  if (!changes || changes.length === 0) {
    toast.add({ severity: 'warn', summary: 'No Changes', detail: 'No values have been modified', life: 3000 })
    return
  }
  
  console.log('Simulating changes:', changes)
  
  store.simulating = true
  try {
    const result = await api.updateSimulatorValues(changes)
    if (result.success) {
      toast.add({ severity: 'success', summary: 'Simulated', detail: `${result.updated} value(s) sent to simulator`, life: 3000 })
      for (const ch of changes) {
        store.originalValues[ch.mnemonic] = ch.value
      }
    }
  } catch (e: any) {
    toast.add({ severity: 'error', summary: 'Simulation Failed', detail: e.message || 'Failed to send values to simulator', life: 5000 })
  } finally {
    store.simulating = false
  }
}

async function resetSimulator() {
  store.simulating = true
  try {
    const result = await api.resetSimulator()
    if (result.success) {
      store.resetAllValues()
      toast.add({ severity: 'info', summary: 'Reset', detail: 'All values reset to initial state', life: 3000 })
    }
  } catch (e: any) {
    toast.add({ severity: 'error', summary: 'Reset Failed', detail: e.message || 'Unknown error', life: 5000 })
  } finally {
    store.simulating = false
  }
}

function clearChanges() {
  store.resetAllValues()
  editingMnemonic.value = null
}

watch(() => [...store.selectedSubsystems], async () => {
  await store.loadValues()
})

onMounted(async () => {
  await store.loadSubsystems()
  store.loadStatus()
  try {
    const result = await api.getSimulatorMode()
    simulationMode.value = result.mode as 'FIXED' | 'RANDOM'
  } catch (e) {
    console.error('Failed to load simulation mode', e)
  }
  try {
    const status = await api.getSimulatorStatus()
    const enable = status?.config?.ENABLE
    simulatorRunning.value = enable === '1'
  } catch (e) {
    console.error('Failed to load simulator status', e)
  }
})
</script>

<template>
  <div class="flex flex-col h-full gap-3 p-4">

    <!-- Top Toolbar -->
    <div class="panel-card p-3 sim-toolbar">
      <div class="flex items-center gap-3 flex-wrap">
        <!-- Search -->
        <span class="p-input-icon-left">
          <i class="pi pi-search" />
          <InputText
            v-model="store.searchQuery"
            placeholder="Search mnemonics..."
            size="small"
            class="w-48"
          />
        </span>

        <span class="p-input-icon-left">
          <i class="pi pi-search" />
          <InputText
            v-model="searchValue"
            placeholder="Search values..."
            size="small"
            class="w-48"
          />
        </span>

        <MultiSelect
          v-model="store.selectedSubsystems"
          :options="store.subsystems.map(s => ({ label: s, value: s }))"
          option-label="label"
          option-value="value"
          placeholder="Subsystems"
          filter
          show-clear
          filter-placeholder="Search subsystems"
          size="small"
          class="w-64 sim-subsystem-select"
          appendTo="body"
          panel-class="sim-subsystem-panel"
          :panelStyle="{ background: 'var(--astra-surface-2)', borderColor: 'var(--astra-border)', color: '#ffffff' }"
          :pt="{
            root:        { style: { background: 'var(--astra-surface-2)', borderColor: 'var(--astra-border)', color: '#ffffff' } },
            label:       { style: { color: '#ffffff', opacity: 1 } },
            placeholder: { style: { color: '#94a3b8', opacity: 1 } },
            input:       { style: { color: '#ffffff' } },
            trigger:     { style: { color: '#ffffff' } },
            panel:       { style: { background: 'var(--astra-surface-2)', borderColor: 'var(--astra-border)', color: '#ffffff' } },
            option:      ({ context }: any) => ({ style: { color: '#ffffff', background: context?.focused || context?.selected ? 'rgba(34,211,238,0.2)' : 'transparent' } }),
            optionLabel: { style: { color: '#ffffff' } },
            filterInput: { style: { color: '#ffffff', background: 'var(--astra-surface-2)', borderColor: 'var(--astra-border)' } },
          }"
          :disabled="store.loading"
        />

        <div class="flex-1" />

        <Button 
          v-if="simulatorRunning" 
          icon="pi pi-stop" 
          label="Stop" 
          size="small" 
          severity="danger" 
          outlined
          :loading="simulatorLoading" 
          @click="stopSimulator" 
        />
        <Button 
          v-else 
          icon="pi pi-play" 
          label="Start" 
          size="small" 
          severity="success" 
          outlined
          :loading="simulatorLoading" 
          @click="startSimulator" 
        />

        <Button icon="pi pi-undo" label="Clear Changes" size="small" severity="secondary" text
          :disabled="!store.hasChanges" @click="clearChanges" />

        <Button icon="pi pi-refresh" label="Reset" size="small" severity="warning" outlined
          :loading="store.simulating" @click="resetSimulator" />

        <div class="flex items-center gap-2 px-3 py-1.5 rounded-md bg-[var(--astra-surface-2)] border border-[var(--astra-border)]">
          <span class="text-xs text-[var(--astra-muted)]">Mode:</span>
          <button
            type="button"
            @click="toggleSimulationMode"
            :disabled="modeLoading"
            class="flex items-center gap-2 px-3 py-1 rounded text-sm font-medium transition-colors"
            :class="simulationMode === 'FIXED' 
              ? 'bg-emerald-600/20 text-emerald-400 border border-emerald-600/50' 
              : 'bg-amber-600/20 text-amber-400 border border-amber-600/50'"
          >
            <i :class="simulationMode === 'FIXED' ? 'pi pi-lock' : 'pi pi-shuffle'"></i>
            {{ simulationMode }}
          </button>
        </div>

        <Button
          icon="pi pi-play"
          :label="`Simulate${store.hasChanges ? ` (${store.changeCount})` : ''}`"
          size="small"
          :loading="store.simulating"
          :disabled="!store.hasChanges"
          class="!bg-cyan-600 !border-cyan-600 hover:!bg-cyan-700 !text-white"
          @click="simulate"
        />
      </div>
    </div>

    <!-- Table -->
    <div class="flex-1 panel-card sim-grid overflow-auto">

      <!-- Error banner -->
      <div v-if="store.loadError"
        class="flex items-center gap-2 px-4 py-2 bg-red-900/40 border border-red-700 rounded text-red-300 text-sm m-2">
        <i class="pi pi-exclamation-triangle" /> {{ store.loadError }}
      </div>

      <!-- Loading overlay -->
      <div v-if="store.loading" class="flex items-center justify-center h-full text-[var(--astra-muted)] gap-2">
        <i class="pi pi-spin pi-spinner" /> Loading mnemonics…
      </div>

      <!-- Empty state -->
      <div v-else-if="!store.loading && valueFilteredMnemonics.length === 0 && store.selectedSubsystems.length === 0"
        class="flex items-center justify-center h-full text-[var(--astra-muted)]">
        Select a subsystem to load mnemonics
      </div>

      <table v-else class="sim-table">
        <thead>
          <tr>
            <th class="col-index">#</th>
            <th class="col-mnemonic">MNEMONIC</th>
            <th class="col-value">VALUE</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(row, i) in valueFilteredMnemonics" :key="row.mnemonic"
            :class="{ 'row-editing': editingMnemonic === row.mnemonic }">

            <!-- Index -->
            <td class="col-index text-[var(--astra-muted)]">{{ i + 1 }}</td>

            <!-- Mnemonic -->
            <td class="col-mnemonic font-mono">{{ row.mnemonic }}</td>

            <!-- Value -->
            <td class="col-value" :class="isChanged(row) ? 'cell-changed' : 'cell-value'">

              <!-- Display mode: double-click to edit -->
              <span
                v-if="editingMnemonic !== row.mnemonic"
                class="cell-display"
                @dblclick="startEdit(row)"
                :title="row.range?.length ? row.range.join(' / ') : ''"
              >
                {{ row.value || '—' }}
                <i v-if="row.range?.length" class="pi pi-chevron-down cell-hint" />
              </span>

              <!-- Edit mode: AutoComplete -->
              <AutoComplete
                v-else
                v-model="editValue"
                :suggestions="filteredSuggestions"
                @complete="onComplete($event, row.range ?? [])"
                @option-select="commitEdit(row, $event)"
                @keydown.enter.prevent="commitEdit(row)"
                @keydown.escape.prevent="cancelEdit(row)"
                @blur="onBlur(row)"
                dropdown
                auto-focus
                fluid
                append-to="body"
                :pt="{
                  root: { style: { width: '100%' } },
                  input: { style: {
                    width: '100%',
                    fontFamily: '\'JetBrains Mono\', \'Fira Code\', monospace',
                    fontSize: '13px',
                    background: '#0f1623',
                    color: '#e2e8f0',
                    border: '1px solid #22d3ee',
                    borderRadius: '4px',
                    padding: '2px 8px',
                    height: '30px',
                  }},
                  dropdownButton: { style: {
                    background: '#1e2d45',
                    border: '1px solid #22d3ee',
                    borderLeft: 'none',
                    color: '#22d3ee',
                    height: '30px',
                  }},
                }"
              />
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Summary Footer -->
    <div class="flex items-center justify-between px-4 py-2.5 panel-card text-xs">
      <div class="flex items-center gap-5 text-[var(--astra-muted)]">
        <span>Total: <span class="font-semibold text-[var(--astra-text)]">{{ store.mnemonics.length }}</span></span>
        <span>Showing: <span class="font-semibold text-[var(--astra-text)]">{{ valueFilteredMnemonics.length }}</span></span>
        <span v-if="store.hasChanges" class="flex items-center gap-1.5 text-amber-400">
          <i class="pi pi-pencil text-xs" /> {{ store.changeCount }} modified
        </span>
      </div>
      <div class="flex items-center gap-4 text-[var(--astra-muted)]">
        <span v-if="store.simulatorStatus" class="flex items-center gap-1.5">
          <i :class="store.simulatorStatus.config?.ENABLE === '1' ? 'pi pi-circle-fill text-emerald-400' : 'pi pi-circle-fill text-red-400'" class="text-xs" />
          Simulator {{ store.simulatorStatus.config?.ENABLE === '1' ? 'Active' : 'Inactive' }}
        </span>
        <span v-if="store.simulatorStatus?.config?.MODE">
          Mode: <span class="font-semibold text-[var(--astra-accent)]">{{ store.simulatorStatus.config.MODE }}</span>
        </span>
        <span class="italic">Double-click VALUE to edit, then click Simulate</span>
      </div>
    </div>

  </div>
</template>

<style scoped>
/* ── Layout ── */
.sim-toolbar { position: relative; z-index: 50; }
.sim-grid    { position: relative; z-index: 1; }

/* ── Table ── */
.sim-table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
  font-size: 13px;
}

/* ── Header ── */
.sim-table thead tr {
  position: sticky;
  top: 0;
  z-index: 10;
  background: var(--astra-surface-2, #151f2e);
}

.sim-table th {
  padding: 0 12px;
  height: 40px;
  text-align: left;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.08em;
  color: var(--astra-muted, #94a3b8);
  border-bottom: 1px solid var(--astra-border, #263245);
  white-space: nowrap;
}

/* ── Column widths ── */
.col-index    { width: 60px; text-align: center; }
.col-mnemonic { width: 50%; }
.col-value    { width: 50%; }

/* ── Rows ── */
.sim-table tbody tr {
  height: 40px;
  border-bottom: 1px solid rgba(38, 50, 69, 0.6);
  transition: background 0.1s;
}

.sim-table tbody tr:hover {
  background: rgba(255, 255, 255, 0.03);
}

.sim-table tbody tr.row-editing {
  background: rgba(34, 211, 238, 0.05);
}

/* ── Cells ── */
.sim-table td {
  padding: 0 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: middle;
}

td.col-index { text-align: center; }

.font-mono {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
}

/* ── Value cell states ── */
.cell-value {
  color: var(--astra-accent, #22d3ee);
}

.cell-changed {
  color: #f59e0b;
  background: rgba(251, 191, 36, 0.1);
  border-left: 3px solid #f59e0b;
  font-weight: 600;
}

/* ── Display mode span ── */
.cell-display {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: default;
  user-select: none;
  width: 100%;
}

.cell-display:hover .cell-hint {
  opacity: 0.5;
}

.cell-hint {
  font-size: 9px;
  opacity: 0;
  transition: opacity 0.15s;
  flex-shrink: 0;
}

/* ── Subsystem panel ── */
:deep(.sim-subsystem-panel) {
  background: var(--astra-surface-2);
  border: 1px solid var(--astra-border);
  color: #ffffff;
  z-index: 1100;
}

/* PrimeVue v4 uses p-multiselect-option (not p-multiselect-item) */
:deep(.sim-subsystem-panel .p-multiselect-option)                         { color: #ffffff !important; }
:deep(.sim-subsystem-panel .p-multiselect-option.p-selected)              { color: #ffffff !important; background: rgba(34,211,238,0.18); }
:deep(.sim-subsystem-panel .p-multiselect-option.p-focus)                 { color: #ffffff !important; background: rgba(34,211,238,0.12); }
:deep(.sim-subsystem-panel .p-multiselect-option:hover)                   { color: #ffffff !important; background: rgba(34,211,238,0.12); }
:deep(.sim-subsystem-panel .p-multiselect-option span)                    { color: #ffffff !important; }
:deep(.sim-subsystem-panel .p-multiselect-option .p-checkbox .p-checkbox-box) { border-color: var(--astra-muted); background: transparent; }
:deep(.sim-subsystem-panel .p-multiselect-option.p-selected .p-checkbox .p-checkbox-box) { border-color: var(--astra-accent); background: var(--astra-accent); }
/* filter input */
:deep(.sim-subsystem-panel .p-multiselect-filter)                         { color: #ffffff !important; }
:deep(.sim-subsystem-panel .p-multiselect-filter::placeholder)            { color: #cbd5e1 !important; }
/* trigger label */
:deep(.sim-subsystem-select .p-multiselect-label)                         { color: #ffffff !important; }
:deep(.sim-subsystem-select .p-multiselect-label.p-placeholder)           { color: #cbd5e1 !important; }
</style>
