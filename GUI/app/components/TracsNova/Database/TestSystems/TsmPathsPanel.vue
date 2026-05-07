<template>
  <div class="ts-panel">
    <Toast />
    <div class="ts-header">
      <h2>Test Systems / TSM Paths</h2>
    </div>

    <div class="ts-section">
      <div class="ts-section-header">
        <h3>TSM Paths</h3>
        <div class="actions">
          <Button label="Refresh" size="small" severity="secondary" @click="load" />
          <Button label="Save" size="small" :loading="saving" @click="save" />
        </div>
      </div>

      <ag-grid-vue
        class="ts-grid"
        style="width: 100%; height: 100%;"
        :theme="isDark
          ? themeQuartz.withPart(colorSchemeDarkBlue)
          : themeQuartz.withPart(colorSchemeLightCold)"
        :columnDefs="columnDefs"
        :rowData="rows"
        :defaultColDef="defaultColDef"
        :cellSelection="cellSelection"
        :rowSelection="rowSelection"
        :suppressContextMenu="false"
        :suppressMovableColumns="true"
        :stopEditingWhenCellsLoseFocus="false"
        :undoRedoCellEditing="true"
        :undoRedoCellEditingLimit="20"
        rowGroupPanelShow="always"
        groupDisplayType="singleColumn"
        @grid-ready="onGridReady"
        @selection-changed="onSelectionChanged"
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
import type { ColDef, GridApi, GridReadyEvent } from 'ag-grid-community';
import { AgGridVue } from 'ag-grid-vue3';
import {
  useTransmitterApi,
  type InstrumentCatalogResponse,
  type TsmPathsResponse,
} from '@/composables/tracsNova/useTransmitterApi';
import { useUiStatePersistence } from '@/composables/tracsNova/useUiStatePersistence';

ModuleRegistry.registerModules([AllEnterpriseModule]);

const toast = useToast();
const isDark = useDark();
const api = useTransmitterApi();
const ui = useUiStatePersistence('ui_state:tracsNova:db:testSystems:tsmPaths');
ui.registerGrid('main');

interface TsmPathRow {
  code: string;
  port: string;
  path1: string | null;
  path2: string | null;
  path3: string | null;
  path4: string | null;
  path5: string | null;
  path6: string | null;
}

const rows = ref<TsmPathRow[]>([]);
const saving = ref(false);
const gridApi = shallowRef<GridApi | null>(null);

const rowSelection = {
  mode: 'multiRow' as const,
  checkboxes: true,
  headerCheckbox: true,
};

const defaultColDef: ColDef = {
  resizable: true,
  sortable: true,
  filter: true,
  minWidth: 120,
  enableRowGroup: true,
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
    field: 'code',
    headerName: 'Code',
    editable: false,
    minWidth: 140,
    flex: 1,
  },
  { field: 'port', headerName: 'Port', editable: false, minWidth: 120, flex: 1 },
  { field: 'path1', headerName: 'Path1', editable: true, minWidth: 130, flex: 1 },
  { field: 'path2', headerName: 'Path2', editable: true, minWidth: 130, flex: 1 },
  { field: 'path3', headerName: 'Path3', editable: true, minWidth: 130, flex: 1 },
  { field: 'path4', headerName: 'Path4', editable: true, minWidth: 130, flex: 1 },
  { field: 'path5', headerName: 'Path5', editable: true, minWidth: 130, flex: 1 },
  { field: 'path6', headerName: 'Path6', editable: true, minWidth: 130, flex: 1 },
];

function onGridReady(event: GridReadyEvent) {
  gridApi.value = event.api;
  event.api.sizeColumnsToFit();
  ui.onGridReady('main', event);
}

function onSelectionChanged(event: any) {
  event.api.getSelectedRows() as TsmPathRow[];
}

function normalizeNullablePath(value: unknown): string | null {
  if (value === null || value === undefined) return null;
  const text = String(value).trim();
  return text === '' ? null : text;
}

async function load() {
  const tsmPathsRes = await api.getProjectTsmPaths();

  if (tsmPathsRes.error.value) {
    toast.add({ severity: 'error', summary: 'Load Failed', detail: 'Unable to fetch TSM path rows.', life: 3500 });
    return;
  }

  const data = ((tsmPathsRes.data.value as TsmPathsResponse) ?? { rows: [] }).rows ?? [];
  rows.value = data.map((row) => ({
    code: String(row.code ?? ''),
    port: String(row.port ?? ''),
    path1: normalizeNullablePath(row.path1),
    path2: normalizeNullablePath(row.path2),
    path3: normalizeNullablePath(row.path3),
    path4: normalizeNullablePath(row.path4),
    path5: normalizeNullablePath(row.path5),
    path6: normalizeNullablePath(row.path6),
  }));
}

async function save() {
  saving.value = true;
  try {
    const payload = {
      rows: rows.value.map((row) => ({
        code: String(row.code ?? ''),
        port: String(row.port ?? ''),
        path1: normalizeNullablePath(row.path1),
        path2: normalizeNullablePath(row.path2),
        path3: normalizeNullablePath(row.path3),
        path4: normalizeNullablePath(row.path4),
        path5: normalizeNullablePath(row.path5),
        path6: normalizeNullablePath(row.path6),
      })),
    };
    const res = await api.saveProjectTsmPaths(payload);
    if (res.error.value) {
      throw res.error.value;
    }
    await load();
    toast.add({ severity: 'success', summary: 'Saved', detail: 'TSM Paths updated.', life: 3000 });
  } catch {
    toast.add({ severity: 'error', summary: 'Save Failed', detail: 'Unable to save TSM path rows.', life: 3500 });
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
.ts-panel {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 4rem);
  padding: 1.5rem;
  color: #e2e8f0;
  box-sizing: border-box;
}

.ts-header h2 {
  font-size: 1.1rem;
  font-weight: 600;
  color: #22d3ee;
  margin: 0 0 1rem;
}

.ts-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.ts-section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.75rem;
  flex-shrink: 0;
}

.ts-section-header h3 {
  font-size: 0.95rem;
  font-weight: 500;
  color: #cbd5e1;
  margin: 0;
}

.actions { display: flex; gap: 0.5rem; }

.ts-grid { flex: 1; min-height: 0; }
</style>
