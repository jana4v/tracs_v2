<template>
  <div class="ts-panel">
    <div class="ts-header">
      <h2>Test Systems / Instruments</h2>
    </div>

    <div class="ts-section">
      <div class="ts-section-header">
        <div class="title-with-help">
          <h3>Instruments</h3>
          <Button
            icon="pi pi-question-circle"
            text
            rounded
            aria-label="Address format help"
            class="help-btn"
            @click="showAddressHelp"
          />
        </div>
        <div class="actions">
          <Button
            :label="`Low Level GPIB: ${useLowLevelGpib ? 'ON' : 'OFF'}`"
            size="small"
            :severity="useLowLevelGpib ? 'success' : 'secondary'"
            :outlined="!useLowLevelGpib"
            :loading="togglingLowLevelGpib"
            @click="toggleLowLevelGpib"
          />
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
        :rowDragManaged="true"
        :animateRows="true"
        :suppressContextMenu="false"
        :suppressMovableColumns="true"
        :undoRedoCellEditing="true"
        :undoRedoCellEditingLimit="20"
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
import type { ColDef, GridApi, GridReadyEvent } from 'ag-grid-community';
import { AgGridVue } from 'ag-grid-vue3';
import {
  useTransmitterApi,
  type ConfigurationValueResponse,
  type InstrumentCatalogResponse,
  type ProjectInstrumentsResponse,
  type ProjectInstrumentRow,
} from '@/composables/tracsNova/useTransmitterApi';
import { useUiStatePersistence } from '@/composables/tracsNova/useUiStatePersistence';

ModuleRegistry.registerModules([AllEnterpriseModule]);

const toast = useToast();
const isDark = useDark();
const api = useTransmitterApi();
const ui = useUiStatePersistence('ui_state:tracsNova:db:testSystems:instruments');
ui.registerGrid('main');

const ADDRESS_HINT = 'LAN: 172.20.xx.xxx | GPIB/VISA: 0:8 (preferred) or GPIB0::8::INSTR';

interface InstrumentRow {
  instrument_name: string;
  model: string;
  address_main: string;
  address_redt: string;
  use_redt: boolean;
}

const instrumentCatalog = ref<Record<string, string[]>>({});

const rows = ref<InstrumentRow[]>([]);
const saving = ref(false);
const togglingLowLevelGpib = ref(false);
const useLowLevelGpib = ref(false);
const gridApi = shallowRef<GridApi | null>(null);

const defaultColDef: ColDef = {
  resizable: true,
  sortable: false,
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
    colId: 'drag',
    headerName: '',
    rowDrag: true,
    editable: false,
    sortable: false,
    filter: false,
    suppressMovable: true,
    suppressFillHandle: true,
    width: 56,
    maxWidth: 56,
    pinned: 'left',
  },
  {
    field: 'instrument_name',
    headerName: 'Instrument Name',
    editable: false,
    minWidth: 220,
    flex: 1,
  },
  {
    field: 'model',
    headerName: 'Model',
    editable: true,
    suppressFillHandle: true,
    cellEditor: 'agSelectCellEditor',
    cellEditorParams: (params: any) => ({
      values: getModelOptions(String(params?.data?.instrument_name ?? '')),
    }),
    minWidth: 160,
    flex: 1,
  },
  {
    field: 'address_main',
    headerName: 'Address Main',
    editable: true,
    headerTooltip: ADDRESS_HINT,
    tooltipValueGetter: () => ADDRESS_HINT,
    minWidth: 230,
    flex: 1.2,
  },
  {
    field: 'address_redt',
    headerName: 'Address Redt',
    editable: true,
    headerTooltip: ADDRESS_HINT,
    tooltipValueGetter: () => ADDRESS_HINT,
    minWidth: 230,
    flex: 1.2,
  },
  {
    field: 'use_redt',
    headerName: 'UseRedt',
    editable: true,
    cellDataType: 'boolean',
    cellRenderer: 'agCheckboxCellRenderer',
    cellEditor: 'agCheckboxCellEditor',
    minWidth: 130,
    maxWidth: 140,
  },
];

function parseBool(value: unknown): boolean {
  if (typeof value === 'boolean') return value;
  if (typeof value === 'number') return value !== 0;
  const text = String(value ?? '').trim().toLowerCase();
  return ['1', 'true', 'yes', 'on'].includes(text);
}

function getModelOptions(instrumentName: string): string[] {
  return instrumentCatalog.value[instrumentName] ?? [];
}

function normalizeRow(row: Partial<ProjectInstrumentRow>): InstrumentRow {
  const instrumentName = String(row.instrument_name ?? '');
  const models = getModelOptions(instrumentName);
  const model = String(row.model ?? '').trim();
  const finalModel = model && models.includes(model) ? model : (models[0] ?? '');

  return {
    instrument_name: instrumentName,
    model: finalModel,
    address_main: String((row as any).address_main ?? ''),
    address_redt: String((row as any).address_redt ?? ''),
    use_redt: parseBool((row as any).use_redt),
  };
}

function defaultRowsFromCatalog(catalog: Record<string, string[]>): InstrumentRow[] {
  const out: InstrumentRow[] = [];
  Object.entries(catalog).forEach(([instrumentName, models]) => {
    const defaultModel = models[0] ?? '';
    out.push({ instrument_name: instrumentName, model: defaultModel, address_main: '', address_redt: '', use_redt: false });
  });
  return out;
}

function onGridReady(event: GridReadyEvent) {
  gridApi.value = event.api;
  event.api.sizeColumnsToFit();
  ui.onGridReady('main', event);
}

function showAddressHelp() {
  toast.add({
    severity: 'info',
    summary: 'Address Format',
    detail: 'LAN: 172.20.xx.xxx | GPIB/VISA: 0:8 (preferred) or GPIB0::8::INSTR',
    life: 5000,
  });
}

async function save() {
  saving.value = true;
  try {
    const payloadRows: InstrumentRow[] = [];
    if (typeof gridApi.value?.forEachNodeAfterFilterAndSort === 'function') {
      gridApi.value.forEachNodeAfterFilterAndSort((n) => {
        if (n.data) payloadRows.push(n.data as InstrumentRow);
      });
    } else {
      gridApi.value?.forEachNode((n) => {
        if (n.data) payloadRows.push(n.data as InstrumentRow);
      });
    }

    const res = await api.saveProjectInstruments({
      rows: payloadRows.map((row) => ({
        instrument_name: String(row.instrument_name ?? ''),
        model: String(row.model ?? ''),
        address_main: String(row.address_main ?? ''),
        address_redt: String(row.address_redt ?? ''),
        use_redt: parseBool(row.use_redt),
      })),
    });
    if (res.error.value) {
      toast.add({ severity: 'error', summary: 'Save Failed', detail: 'Unable to save instruments.', life: 3500 });
      return;
    }

    rows.value = payloadRows.map((r) => normalizeRow(r));
    toast.add({ severity: 'success', summary: 'Saved', detail: 'Instruments updated.', life: 3000 });
  } finally {
    saving.value = false;
  }
}

async function toggleLowLevelGpib() {
  const previous = useLowLevelGpib.value;
  const next = !previous;
  useLowLevelGpib.value = next;
  togglingLowLevelGpib.value = true;
  try {
    const res = await api.saveConfigurationValue('USE_LOW_LEVEL_GPIB', { value: next ? '1' : '0' });
    if (res.error.value) {
      throw res.error.value;
    }
    toast.add({ severity: 'success', summary: 'Updated', detail: `Low Level GPIB ${next ? 'enabled' : 'disabled'}.`, life: 2200 });
  } catch {
    useLowLevelGpib.value = previous;
    toast.add({ severity: 'error', summary: 'Update Failed', detail: 'Unable to update Low Level GPIB setting.', life: 3200 });
  } finally {
    togglingLowLevelGpib.value = false;
  }
}

async function load() {
  const catalogRes = await api.getInstrumentCatalog();
  if (catalogRes.error.value) {
    toast.add({ severity: 'error', summary: 'Load Failed', detail: 'Unable to load instruments catalog.', life: 3500 });
    return;
  }

  const catalogPayload = (catalogRes.data.value as InstrumentCatalogResponse) ?? { instruments: {} };
  instrumentCatalog.value = catalogPayload.instruments ?? {};

  const lowLevelRes = await api.getConfigurationValue('USE_LOW_LEVEL_GPIB');
  if (!lowLevelRes.error.value && lowLevelRes.data.value) {
    const payload = lowLevelRes.data.value as ConfigurationValueResponse;
    useLowLevelGpib.value = parseBool(payload?.value);
  } else {
    useLowLevelGpib.value = false;
  }

  const projectRes = await api.getProjectInstruments();
  if (!projectRes.error.value && projectRes.data.value) {
    const projectPayload = projectRes.data.value as ProjectInstrumentsResponse;
    rows.value = (projectPayload.rows ?? []).map((r) => normalizeRow(r));
  } else {
    rows.value = defaultRowsFromCatalog(instrumentCatalog.value);
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

.title-with-help {
  display: flex;
  align-items: center;
  gap: 0.35rem;
}

.help-btn {
  color: #60a5fa;
}

.actions { display: flex; gap: 0.5rem; }

.ts-grid { flex: 1; min-height: 0; }

:global(.p-toast) {
  top: 120px !important;
  z-index: 99999 !important;
}
</style>
