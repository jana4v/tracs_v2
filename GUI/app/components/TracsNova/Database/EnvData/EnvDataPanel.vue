<template>
  <div class="env-panel">
    <Toast />
    <div class="env-header">
      <h2>ENV Data</h2>
    </div>

    <div class="env-section">
      <div class="env-section-header">
        <h3>AG</h3>
        <div class="actions">
          <Button label="Refresh" size="small" severity="secondary" @click="load" />
          <Button label="Save" size="small" :loading="saving" @click="save" />
        </div>
      </div>

      <ag-grid-vue
        class="env-grid"
        style="width: 100%; height: 100%;"
        :theme="isDark
          ? themeQuartz.withPart(colorSchemeDarkBlue)
          : themeQuartz.withPart(colorSchemeLightCold)"
        :columnDefs="columnDefs"
        :rowData="rows"
        :defaultColDef="defaultColDef"
        :rowSelection="rowSelection"
        :cellSelection="cellSelection"
        :suppressContextMenu="false"
        :suppressMovableColumns="true"
        :stopEditingWhenCellsLoseFocus="false"
        :undoRedoCellEditing="true"
        :undoRedoCellEditingLimit="20"
        @grid-ready="onGridReady"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useToast } from 'primevue/usetoast';
import { ModuleRegistry } from 'ag-grid-community';
import { AllEnterpriseModule } from 'ag-grid-enterprise';
import {
  colorSchemeDarkBlue,
  colorSchemeLightCold,
  themeQuartz,
} from 'ag-grid-community';
import type { ColDef, GridReadyEvent } from 'ag-grid-community';
import { AgGridVue } from 'ag-grid-vue3';
import {
  useTransmitterApi,
  type EnvDataDirectorySelectResponse,
  type EnvDataRow,
  type EnvDataRowsResponse,
} from '@/composables/tracsNova/useTransmitterApi';
import { useUiStatePersistence } from '@/composables/tracsNova/useUiStatePersistence';

ModuleRegistry.registerModules([AllEnterpriseModule]);

interface EnvDataGridRow {
  parameter: string;
  value: string;
}

const toast = useToast();
const isDark = useDark();
const api = useTransmitterApi();
const ui = useUiStatePersistence('ui_state:tracsNova:db:envData');
ui.registerGrid('main');

const rows = ref<EnvDataGridRow[]>([]);
const saving = ref(false);
const gridApi = shallowRef<GridApi | null>(null);

const defaultColDef: ColDef = {
  resizable: true,
  sortable: true,
  filter: true,
  minWidth: 160,
};

const rowSelection = {
  mode: 'multiRow' as const,
  checkboxes: true,
  headerCheckbox: true,
};

const cellSelection = {
  mode: 'range' as const,
  handle: {
    mode: 'fill' as const,
    direction: 'xy' as const,
    suppressClearOnFillReduction: true,
  },
};

const columnDefs: ColDef[] = [
  {
    field: 'parameter',
    headerName: 'Parameter',
    editable: false,
    minWidth: 220,
    flex: 1.2,
  },
  {
    field: 'value',
    headerName: 'Value',
    editable: true,
    minWidth: 220,
    flex: 1,
  },
  {
    headerName: 'Browse',
    editable: false,
    sortable: false,
    filter: false,
    minWidth: 130,
    maxWidth: 140,
    cellRenderer: (params: any) => {
      const button = document.createElement('button');
      button.type = 'button';
      button.className = 'browse-btn';
      button.textContent = 'Browse';
      const isResultsDirectory = String(params?.data?.parameter ?? '') === 'RESULTS_DIRECTORY';
      button.disabled = !isResultsDirectory;
      button.addEventListener('click', () => {
        if (!isResultsDirectory) return;
        const rowIndex = Number(params?.node?.rowIndex ?? -1);
        if (Number.isNaN(rowIndex) || rowIndex < 0) return;
        void pickDirectoryForRow(rowIndex);
      });
      return button;
    },
  },
];

function onGridReady(event: GridReadyEvent) {
  gridApi.value = event.api;
  event.api.sizeColumnsToFit();
  ui.onGridReady('main', event);
}

async function pickDirectoryForRow(rowIndex: number): Promise<void> {
  const row = rows.value[rowIndex];
  if (!row || row.parameter !== 'RESULTS_DIRECTORY') return;

  const selected = await selectDirectoryValue();
  if (!selected) return;

  rows.value[rowIndex] = {
    ...row,
    value: selected,
  };
  gridApi.value?.applyTransaction({ update: [rows.value[rowIndex]] });
}

async function selectDirectoryValue(): Promise<string | null> {
  const res = await api.selectEnvDataDirectory();
  if (res.error.value) {
    toast.add({ severity: 'error', summary: 'Directory Select Failed', detail: 'Unable to open native folder picker.', life: 3000 });
    return null;
  }

  const payload = res.data.value as EnvDataDirectorySelectResponse | null;
  const selected = String(payload?.path ?? '').trim();
  return selected.length > 0 ? selected : null;
}

function normalizeRows(rowsIn: EnvDataGridRow[]): EnvDataGridRow[] {
  return rowsIn
    .map((row) => ({
      parameter: String(row.parameter ?? '').trim(),
      value: String(row.value ?? ''),
    }))
    .filter((row) => row.parameter.length > 0);
}

async function load() {
  gridApi.value?.stopEditing();

  const res = await api.getEnvDataRows();
  if (res.error.value) {
    toast.add({ severity: 'error', summary: 'Load Failed', detail: 'Unable to load ENV Data rows.', life: 3500 });
    return;
  }

  const payload = (res.data.value as EnvDataRowsResponse) ?? { rows: [] };
  rows.value = (payload.rows ?? []).map((row: EnvDataRow) => ({
    parameter: String(row.parameter ?? ''),
    value: String(row.value ?? ''),
  }));
}

async function save() {
  saving.value = true;
  try {
    // Commit any active in-cell edit before collecting payload rows.
    gridApi.value?.stopEditing();

    const normalized = normalizeRows(rows.value);
    const res = await api.saveEnvDataRows({ rows: normalized });
    if (res.error.value) {
      toast.add({ severity: 'error', summary: 'Save Failed', detail: 'Unable to save ENV Data rows.', life: 3500 });
      return;
    }

    await load();
    toast.add({ severity: 'success', summary: 'Saved', detail: 'ENV Data updated.', life: 2500 });
  } finally {
    saving.value = false;
  }
}

onMounted(async () => {
  await load();
  await ui.load();
});
</script>

<style scoped>
.env-panel {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 4rem);
  padding: 1.5rem;
  color: #e2e8f0;
  box-sizing: border-box;
}

.env-header h2 {
  font-size: 1.1rem;
  font-weight: 600;
  color: #22d3ee;
  margin: 0 0 1rem;
}

.env-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.env-section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.75rem;
  flex-shrink: 0;
}

.env-section-header h3 {
  font-size: 0.95rem;
  font-weight: 500;
  color: #cbd5e1;
  margin: 0;
}

.actions {
  display: flex;
  gap: 0.5rem;
}

.env-grid {
  flex: 1;
  min-height: 0;
}

:deep(.browse-btn) {
  border: 1px solid #334155;
  background: #0f172a;
  color: #cbd5e1;
  border-radius: 4px;
  padding: 0.2rem 0.55rem;
  cursor: pointer;
  font-size: 0.75rem;
}

:deep(.browse-btn:disabled) {
  opacity: 0.45;
  cursor: not-allowed;
}
</style>
