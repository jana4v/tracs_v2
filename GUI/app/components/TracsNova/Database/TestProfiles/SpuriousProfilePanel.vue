<template>
  <div class="tp-panel">
    <Toast />
    <div class="tp-header">
      <h2>Test Profiles — Transmitter / Spurious / Profile</h2>
    </div>

    <div class="tp-section">
      <div class="tp-section-header">
        <h3>Spurious Profile</h3>
        <div class="actions">
          <Button label="Refresh" size="small" severity="secondary" @click="load" />
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
        :suppressContextMenu="false"
        :suppressMovableColumns="true"
        rowGroupPanelShow="always"
        groupDisplayType="singleColumn"
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
import type { CellSelectionOptions, ColDef, GridApi, GridReadyEvent } from 'ag-grid-community';
import { AgGridVue } from 'ag-grid-vue3';
import {
  useTransmitterApi,
  type TestProfileSpuriousRowsResponse,
} from '@/composables/tracsNova/useTransmitterApi';
import { toNumberOrNull } from '@/composables/tracsNova/utils';
import { useUiStatePersistence } from '@/composables/tracsNova/useUiStatePersistence';

ModuleRegistry.registerModules([AllEnterpriseModule]);

const toast = useToast();
const isDark = useDark();
const api = useTransmitterApi();
const ui = useUiStatePersistence('ui_state:tracsNova:db:testProfiles:spuriousProfile');
ui.registerGrid('main');

interface SpuriousProfileRow {
  transmitter_code: string;
  code: string;
  port: string;
  frequency_label: string;
  frequency: number | null;
  inband: boolean;
  spurband: boolean;
}

const rows = ref<SpuriousProfileRow[]>([]);
const saving = ref(false);
const gridApi = shallowRef<GridApi | null>(null);

const cellSelection: boolean | CellSelectionOptions = {
  handle: { mode: 'fill', direction: 'y', suppressClearOnFillReduction: true },
};

const defaultColDef: ColDef = { resizable: true, sortable: true, filter: true, minWidth: 100, enableRowGroup: true };

const columnDefs: ColDef[] = [
  { field: 'code',            headerName: 'Code',            editable: false, minWidth: 80  },
  { field: 'port',            headerName: 'Port',            editable: false, minWidth: 80  },
  { field: 'frequency_label', headerName: 'Freq Label',      editable: false, minWidth: 110 },
  { field: 'frequency',       headerName: 'Frequency (MHz)', editable: false, minWidth: 140 },
  {
    field: 'inband',
    headerName: 'Inband',
    editable: true,
    cellRenderer: 'agCheckboxCellRenderer',
    cellEditor: 'agCheckboxCellEditor',
    suppressFillHandle: false,
    filter: false,
    minWidth: 110,
  },
  {
    field: 'spurband',
    headerName: 'Spurband',
    editable: true,
    cellRenderer: 'agCheckboxCellRenderer',
    cellEditor: 'agCheckboxCellEditor',
    suppressFillHandle: false,
    filter: false,
    minWidth: 120,
  },
];

function onGridReady(event: GridReadyEvent) {
  gridApi.value = event.api;
  event.api.sizeColumnsToFit();
  ui.onGridReady('main', event);
}

function uniqueByKey(items: SpuriousProfileRow[]) {
  const seen = new Set<string>();
  return items.filter((item) => {
    const key = `${item.transmitter_code}|${item.code}|${item.port}|${item.frequency_label}|${item.frequency ?? ''}`;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}

function mapRows(payload: TestProfileSpuriousRowsResponse): SpuriousProfileRow[] {
  return (payload.rows ?? []).map((item) => {
    return {
      transmitter_code: String(item.transmitter_code ?? ''),
      code: String(item.row?.code ?? ''),
      port: String(item.row?.port ?? ''),
      frequency_label: String(item.row?.frequency_label ?? ''),
      frequency: toNumberOrNull(item.row?.frequency),
      inband: Boolean(item.row?.inband),
      spurband: Boolean(item.row?.spurband),
    };
  });
}

async function load() {
  const res = await api.getTestProfileSpuriousRows();
  if (!res.error.value && res.data.value) {
    const payload = res.data.value as TestProfileSpuriousRowsResponse;
    rows.value = uniqueByKey(mapRows(payload));
    return;
  }

  toast.add({ severity: 'error', summary: 'Load Failed', detail: 'Unable to load spurious profile rows.', life: 3500 });
}

async function save() {
  saving.value = true;
  try {
    const data: SpuriousProfileRow[] = [];
    gridApi.value?.forEachNode((n) => { if (n.data) data.push(n.data as SpuriousProfileRow); });

    const payload = {
      rows: data
        .map((r) => ({
          transmitter_code: String(r.transmitter_code ?? ''),
          row: {
            code: String(r.code ?? ''),
            port: String(r.port ?? ''),
            frequency_label: String(r.frequency_label ?? ''),
            frequency: r.frequency === null ? '' : String(r.frequency),
            inband: Boolean(r.inband),
            spurband: Boolean(r.spurband),
          },
        }))
        .filter((r) => r.transmitter_code !== ''),
    };

    const res = await api.saveTestProfileSpuriousRows(payload);
    if (res.error.value) {
      toast.add({ severity: 'error', summary: 'Save Failed', detail: 'Unable to save spurious profile rows.', life: 3500 });
      return;
    }

    const summary = res.data.value as any;
    toast.add({
      severity: 'success',
      summary: 'Saved',
      detail: `Spurious Profile updated (${summary?.updated_rows ?? 0} rows).`,
      life: 3000,
    });

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
