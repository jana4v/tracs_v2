<script setup lang="ts">
import { AgGridVue } from 'ag-grid-vue3'
import type { ColDef, GridApi, GridReadyEvent, RowDragEndEvent, CellValueChangedEvent } from 'ag-grid-community'
import type { UdTmRow, UdTmVersion } from '~/types/astra'

definePageMeta({ title: 'UD TM' })

const store = useUdTmStore()
const settingsStore = useSettingsStore()
const toast = useToast()
const { clearTMCaches } = useMonaco()

let gridApi: GridApi | null = null

// UI state
const addRowCount = ref(1)
const showSaveDialog = ref(false)
const showHistoryPanel = ref(false)
const changeMessage = ref('')
const previewVersion = ref<UdTmVersion | null>(null)

// === Column Definitions ===
const columnDefs = ref<ColDef[]>([
  {
    headerName: '',
    rowDrag: true,
    width: 50,
    suppressHeaderMenuButton: true,
    sortable: false,
    filter: false,
    resizable: false,
    editable: false,
    suppressFillHandle: true,
  },
  {
    headerName: '#',
    valueGetter: (params: any) => params.node?.rowIndex != null ? params.node.rowIndex + 1 : '',
    width: 60,
    sortable: false,
    filter: false,
    resizable: false,
    editable: false,
    suppressFillHandle: true,
  },
  {
    headerName: 'MNEMONIC',
    field: 'mnemonic',
    editable: true,
    flex: 2,
    minWidth: 200,
    cellClass: (params: any) => {
      if (params.data?.mnemonic && store.duplicateMnemonics.has(params.data.mnemonic)) {
        return 'ud-tm-duplicate-cell'
      }
      return ''
    },
  },
  {
    headerName: 'VALUE',
    field: 'value',
    editable: true,
    flex: 1,
    minWidth: 100,
  },
  {
    headerName: 'RANGE',
    field: 'range',
    editable: true,
    flex: 1,
    minWidth: 100,
  },
  {
    headerName: 'LIMIT',
    field: 'limit',
    editable: true,
    flex: 1,
    minWidth: 100,
  },
  {
    headerName: 'TOLERANCE',
    field: 'tolerance',
    editable: true,
    flex: 1,
    minWidth: 100,
  },
  {
    headerName: '',
    width: 60,
    sortable: false,
    filter: false,
    resizable: false,
    editable: false,
    suppressFillHandle: true,
    cellRenderer: (params: any) => {
      const btn = document.createElement('button')
      btn.innerHTML = '<i class="pi pi-trash"></i>'
      btn.className = 'ud-tm-delete-btn'
      btn.addEventListener('click', () => {
        const rowIndex = store.rows.findIndex(r => r.row_number === params.data.row_number)
        if (rowIndex >= 0) {
          store.deleteRow(rowIndex)
          gridApi?.setGridOption('rowData', store.rows)
        }
      })
      return btn
    },
  },
])

const defaultColDef: ColDef = {
  resizable: true,
  suppressMovable: true,
}

// === Grid Events ===
function onGridReady(event: GridReadyEvent) {
  gridApi = event.api
  gridApi.sizeColumnsToFit()
}

function onRowDragEnd(_event: RowDragEndEvent) {
  const reorderedRows: UdTmRow[] = []
  gridApi?.forEachNodeAfterFilterAndSort((node) => {
    if (node.data) reorderedRows.push(node.data)
  })
  store.rows = reorderedRows
  store.renumberRows()
}

function onCellValueChanged(event: CellValueChangedEvent) {
  const rowIndex = store.rows.findIndex(r => r.row_number === event.data.row_number)
  if (rowIndex >= 0) {
    store.rows[rowIndex] = { ...event.data }
  }
  gridApi?.refreshCells({ columns: ['mnemonic'], force: true })
}

// === Actions ===
function handleAddRows() {
  store.addRows(addRowCount.value)
  gridApi?.setGridOption('rowData', store.rows)
}

function openSaveDialog() {
  if (!store.validateMnemonics()) {
    const dupes = [...store.duplicateMnemonics].join(', ')
    toast.add({ severity: 'error', summary: 'Duplicate Mnemonics', detail: `Mnemonics must be unique: ${dupes}`, life: 5000 })
    return
  }
  changeMessage.value = ''
  showSaveDialog.value = true
}

async function confirmSave() {
  showSaveDialog.value = false
  try {
    const result = await store.save(
      settingsStore.project || 'default',
      settingsStore.username || 'unknown',
      changeMessage.value,
    )
    if (result.saved) {
      clearTMCaches()
      toast.add({ severity: 'success', summary: 'Saved', detail: `Version ${result.version} saved`, life: 3000 })
    }
  } catch (e: any) {
    toast.add({ severity: 'error', summary: 'Save Failed', detail: e.message || 'Unknown error', life: 5000 })
  }
}

async function openHistory() {
  await store.loadVersions(settingsStore.project || 'default')
  showHistoryPanel.value = true
}

function previewVersionData(version: UdTmVersion) {
  previewVersion.value = version
}

// === Lifecycle ===
onMounted(async () => {
  await store.loadRows(settingsStore.project || 'default')
  gridApi?.setGridOption('rowData', store.rows)
})
</script>

<template>
  <div class="flex flex-col h-full gap-3 p-4">
    <!-- Top Toolbar -->
    <div class="panel-card p-3">
      <div class="flex items-center gap-3 flex-wrap">
        <!-- Add Rows -->
        <Button
          icon="pi pi-plus"
          :label="`+${addRowCount} Rows`"
          size="small"
          severity="secondary"
          outlined
          @click="handleAddRows"
        />

        <InputNumber
          v-model="addRowCount"
          :min="1"
          :max="100"
          show-buttons
          button-layout="horizontal"
          increment-button-icon="pi pi-plus"
          decrement-button-icon="pi pi-minus"
          size="small"
          class="w-28"
        />

        <div class="flex-1" />

        <!-- Save -->
        <Button
          icon="pi pi-save"
          label="Save"
          size="small"
          :loading="store.saving"
          :disabled="!store.isDirty"
          class="!bg-blue-600 !border-blue-600 hover:!bg-blue-700 !text-white"
          @click="openSaveDialog"
        />

        <Divider layout="vertical" class="mx-0 h-6" />

        <!-- History -->
        <Button
          icon="pi pi-history"
          label="History"
          size="small"
          severity="secondary"
          text
          :disabled="store.latestVersion === 0"
          @click="openHistory"
        />
      </div>
    </div>

    <!-- AG Grid Table -->
    <div class="flex-1 panel-card overflow-hidden rounded-2xl">
      <AgGridVue
        class="ag-theme-alpine-dark h-full w-full"
        :row-data="store.rows"
        :column-defs="columnDefs"
        :default-col-def="defaultColDef"
        :animate-rows="true"
        :row-drag-managed="true"
        :get-row-id="(params: any) => String(params.data.row_number)"
        :row-height="42"
        :header-height="40"
        :single-click-edit="true"
        :stop-editing-when-cells-lose-focus="true"
        :cell-selection="true"
        :suppress-clipboard-paste="false"
        :enable-cell-text-selection="true"
        :clipboard-delimitator="'\t'"
        :process-data-from-clipboard="undefined"
        :undo-redo-cell-editing="true"
        :undo-redo-cell-editing-limit="20"
        @grid-ready="onGridReady"
        @row-drag-end="onRowDragEnd"
        @cell-value-changed="onCellValueChanged"
      />
    </div>

    <!-- Summary Footer -->
    <div class="flex items-center justify-between px-4 py-2.5 panel-card text-xs">
      <div class="flex items-center gap-5 text-[var(--astra-muted)]">
        <span>
          Rows: <span class="font-semibold text-[var(--astra-text)]">{{ store.totalRowCount }}</span>
        </span>
        <span v-if="store.duplicateMnemonics.size > 0" class="flex items-center gap-1.5 text-red-400">
          <i class="pi pi-exclamation-triangle text-xs" />
          {{ store.duplicateMnemonics.size }} duplicate mnemonic(s)
        </span>
      </div>
      <div class="flex items-center gap-4 text-[var(--astra-muted)]">
        <span v-if="store.latestVersion > 0">
          Version: <span class="font-semibold text-[var(--astra-accent)]">{{ store.latestVersion }}</span>
        </span>
        <span class="flex items-center gap-1.5">
          <i :class="store.isDirty ? 'pi pi-circle-fill text-amber-400' : 'pi pi-check-circle text-emerald-400'" class="text-xs" />
          {{ store.isDirty ? 'Modified' : 'Saved' }}
        </span>
        <span class="text-[var(--astra-muted)] italic">
          Syntax: TM.UDTM.&lt;mnemonic&gt;
        </span>
      </div>
    </div>

    <!-- Save Dialog -->
    <Dialog
      v-model:visible="showSaveDialog"
      header="Save UD TM"
      :modal="true"
      :style="{ width: '400px' }"
    >
      <div class="flex flex-col gap-3 py-2">
        <div>
          <label class="text-sm text-[var(--astra-muted)]">Change message (optional):</label>
          <InputText v-model="changeMessage" placeholder="What changed?" class="w-full mt-1" />
        </div>
      </div>
      <template #footer>
        <Button label="Cancel" size="small" severity="secondary" text @click="showSaveDialog = false" />
        <Button label="Save" size="small" icon="pi pi-save" @click="confirmSave" />
      </template>
    </Dialog>

    <!-- Version History Drawer -->
    <Drawer
      v-model:visible="showHistoryPanel"
      position="right"
      header="Version History"
      :style="{ width: '420px' }"
    >
      <div class="flex flex-col gap-0">
        <div
          v-for="v in store.versions"
          :key="v._id"
          class="border-b border-[var(--astra-border)] p-3 hover:bg-[var(--astra-surface-2)] cursor-pointer transition-colors"
          @click="previewVersionData(v)"
        >
          <div class="flex items-center justify-between mb-1">
            <span class="font-semibold text-[var(--astra-accent)]">Version {{ v.version }}</span>
            <span class="text-xs text-[var(--astra-muted)]">{{ v.created_by }}</span>
          </div>
          <div class="text-xs text-[var(--astra-muted)] mb-1">
            {{ new Date(v.created_at).toLocaleString() }}
          </div>
          <div v-if="v.change_message" class="text-xs text-[var(--astra-text)] mb-2 italic">
            "{{ v.change_message }}"
          </div>
          <div v-if="v.changes && v.changes.length > 0" class="flex flex-col gap-0.5">
            <div
              v-for="(ch, ci) in v.changes"
              :key="ci"
              class="text-xs px-2 py-0.5 rounded"
              :class="{
                'text-emerald-400 bg-emerald-400/10': ch.type === 'added',
                'text-amber-400 bg-amber-400/10': ch.type === 'modified',
                'text-red-400 bg-red-400/10': ch.type === 'deleted',
                'text-blue-400 bg-blue-400/10': ch.type === 'reordered',
              }"
            >
              <span class="font-semibold uppercase">{{ ch.type }}</span>
              row {{ ch.row_number }} ({{ ch.mnemonic }})
              <span v-if="ch.field">
                — {{ ch.field }}: "{{ ch.old_value }}" → "{{ ch.new_value }}"
              </span>
            </div>
          </div>
        </div>

        <div v-if="store.versions.length === 0" class="p-4 text-center text-[var(--astra-muted)] text-sm">
          No version history available
        </div>
      </div>
    </Drawer>

    <!-- Version Preview Dialog -->
    <Dialog
      v-model:visible="previewVersion"
      :header="`Version ${previewVersion?.version} Preview`"
      :modal="true"
      :style="{ width: '700px', maxHeight: '80vh' }"
    >
      <table v-if="previewVersion" class="w-full text-xs">
        <thead>
          <tr class="bg-[var(--astra-surface-2)] text-[var(--astra-muted)]">
            <th class="p-2 text-left">#</th>
            <th class="p-2 text-left">MNEMONIC</th>
            <th class="p-2 text-left">VALUE</th>
            <th class="p-2 text-left">RANGE</th>
            <th class="p-2 text-left">LIMIT</th>
            <th class="p-2 text-left">TOLERANCE</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="row in previewVersion.rows"
            :key="row.row_number"
            class="border-b border-[var(--astra-border)]"
          >
            <td class="p-2 text-[var(--astra-muted)]">{{ row.row_number }}</td>
            <td class="p-2 font-mono">{{ row.mnemonic }}</td>
            <td class="p-2">{{ row.value }}</td>
            <td class="p-2">{{ row.range }}</td>
            <td class="p-2">{{ row.limit }}</td>
            <td class="p-2">{{ row.tolerance }}</td>
          </tr>
        </tbody>
      </table>
    </Dialog>
  </div>
</template>

<style scoped>
:deep(.ud-tm-delete-btn) {
  background: transparent;
  border: none;
  color: var(--astra-muted);
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 4px;
  transition: all 0.15s;
}

:deep(.ud-tm-delete-btn:hover) {
  color: var(--astra-error);
  background: rgba(248, 113, 113, 0.1);
}

:deep(.ud-tm-duplicate-cell) {
  background: rgba(248, 113, 113, 0.15) !important;
  border-left: 3px solid var(--astra-error) !important;
}
</style>
