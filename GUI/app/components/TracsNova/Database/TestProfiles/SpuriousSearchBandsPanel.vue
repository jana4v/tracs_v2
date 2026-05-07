<template>
  <div class="tp-panel">
    <Toast />
    <div class="tp-header">
      <h2>Test Profiles — Transmitter / Spurious / Bands</h2>
    </div>

    <div class="tp-section">
      <div class="tp-section-header">
        <h3>Spurious Search Bands</h3>
        <div class="actions">
          <Button
            icon="pi pi-plus"
            label="Add Row"
            size="small"
            severity="secondary"
            @click="addRow"
          />
          <Button
            icon="pi pi-trash"
            label="Delete"
            size="small"
            severity="danger"
            :disabled="selectedRows.length === 0"
            @click="deleteRows"
          />
          <Button label="Save" size="small" :loading="saving" @click="save" />
        </div>
      </div>

      <ag-grid-vue
        class="tp-grid"
        style="width: 100%; height: 100%;"
        :theme="isDark
          ? themeQuartz.withPart(colorSchemeDarkBlue)
          : themeQuartz.withPart(colorSchemeLightCold)"
        :columnDefs="columnDefs"
        :rowData="rows"
        :defaultColDef="defaultColDef"
          :cellSelection="cellSelection"
        rowSelection="multiple"
        :suppressContextMenu="false"
        :suppressMovableColumns="true"
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
  type SpuriousBandConfigRow,
} from '@/composables/tracsNova/useTransmitterApi';
import { useUiStatePersistence } from '@/composables/tracsNova/useUiStatePersistence';

ModuleRegistry.registerModules([AllEnterpriseModule]);

const toast = useToast();
const isDark = useDark();
const api = useTransmitterApi();
const ui = useUiStatePersistence('ui_state:tracsNova:db:testProfiles:spuriousBands');
ui.registerGrid('main');

const DEFAULT_PROFILE_NAME = 'Detailed';

const rows = ref<SpuriousBandConfigRow[]>([]);
const selectedRows = ref<SpuriousBandConfigRow[]>([]);
const profileOptions = ref<string[]>([]);
const saving = ref(false);
const gridApi = shallowRef<GridApi | null>(null);

const defaultColDef: ColDef = { resizable: true, sortable: true, filter: true, minWidth: 100 };

const cellSelection = {
  mode: 'range' as const,
  handle: { mode: 'fill' as const, direction: 'xy' as const, suppressClearOnFillReduction: true },
};

const columnDefs = computed<ColDef[]>(() => [
  {
    field: 'profile_name',
    headerName: 'Profile Name',
    editable: true,
    cellEditor: 'agSelectCellEditor',
    cellEditorParams: { values: [...profileOptions.value] },
    checkboxSelection: true,
    headerCheckboxSelection: true,
    minWidth: 160,
    flex: 1,
  },
  {
    field: 'enable',
    headerName: 'Enable',
    editable: true,
    cellDataType: 'boolean',
    cellRenderer: 'agCheckboxCellRenderer',
    minWidth: 100,
    maxWidth: 120,
  },
  {
    field: 'start_frequency',
    headerName: 'Start Frequency (MHz)',
    editable: true,
    cellDataType: 'number',
    minWidth: 180,
    flex: 1,
  },
  {
    field: 'stop_frequency',
    headerName: 'Stop Frequency (MHz)',
    editable: true,
    cellDataType: 'number',
    minWidth: 180,
    flex: 1,
  },
]);

function onGridReady(event: GridReadyEvent) {
  gridApi.value = event.api;
  event.api.sizeColumnsToFit();
  ui.onGridReady('main', event);
}

function onSelectionChanged(event: any) {
  selectedRows.value = event.api.getSelectedRows() as SpuriousBandConfigRow[];
}

async function load() {
  const [typesRes, bandsRes] = await Promise.all([
    api.getTestPlanTypes(),
    api.getSpuriousBandConfigs(),
  ]);

  if (typesRes.error.value) {
    toast.add({ severity: 'error', summary: 'Load Failed', detail: 'Unable to load test plan types.', life: 3500 });
    return;
  }

  profileOptions.value = Array.isArray(typesRes.data.value) ? typesRes.data.value as string[] : [];

  if (bandsRes.error.value) {
    toast.add({ severity: 'error', summary: 'Load Failed', detail: 'Unable to load spurious band configs.', life: 3500 });
    return;
  }
  rows.value = ((bandsRes.data.value as any)?.bands ?? []) as SpuriousBandConfigRow[];
}

function addRow() {
  const defaultProfileName = profileOptions.value[0] ?? DEFAULT_PROFILE_NAME;
  rows.value = [
    ...rows.value,
    { profile_name: defaultProfileName, enable: true, start_frequency: null, stop_frequency: null },
  ];
}

function deleteRows() {
  const sel = gridApi.value?.getSelectedRows() ?? [];
  rows.value = rows.value.filter((r) => !sel.includes(r));
  selectedRows.value = [];
}

async function save() {
  saving.value = true;
  try {
    const data: SpuriousBandConfigRow[] = [];
    gridApi.value?.forEachNode((n) => { if (n.data) data.push(n.data as SpuriousBandConfigRow); });

    const res = await api.saveSpuriousBandConfigs(data);
    if (res.error.value) {
      toast.add({ severity: 'error', summary: 'Save Failed', detail: 'Unable to save spurious band configs.', life: 3500 });
      return;
    }

    toast.add({ severity: 'success', summary: 'Saved', detail: 'Spurious Search Bands saved.', life: 3000 });
    await load();
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
.tp-panel {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 4rem);
  padding: 1.5rem;
  color: #e2e8f0;
  box-sizing: border-box;
}

.tp-header h2 {
  font-size: 1.1rem;
  font-weight: 600;
  color: #22d3ee;
  margin: 0 0 1rem;
}

.tp-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.tp-section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.75rem;
  flex-shrink: 0;
}

.tp-section-header h3 {
  font-size: 0.95rem;
  font-weight: 500;
  color: #cbd5e1;
  margin: 0;
}

.actions { display: flex; gap: 0.5rem; }

.tp-grid { flex: 1; min-height: 0; }
</style>
