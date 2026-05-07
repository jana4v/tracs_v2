<template>
  <div class="tp-panel">
    <Toast />
    <div class="tp-header">
      <h2>Systems / Transponder</h2>
    </div>

    <div class="tp-section">
      <div class="tp-section-header">
        <h3>Transponder Management</h3>
        <div class="actions">
          <Button label="Add Row" size="small" icon="pi pi-plus" @click="addRow" />
          <Button
            label="Delete Selected"
            size="small"
            icon="pi pi-trash"
            severity="danger"
            outlined
            :disabled="selectedCount === 0"
            @click="deleteSelected"
          />
          <Button label="Refresh" size="small" severity="secondary" :loading="loading" @click="load" />
          <Button label="Save" size="small" :loading="saving" @click="save" />
        </div>
      </div>

      <ag-grid-vue
        class="tp-grid"
        style="width: 100%; height: calc(100vh - 14rem);"
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
        :undoRedoCellEditing="true"
        :undoRedoCellEditingLimit="20"
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
import type { ColDef, GridApi, GridReadyEvent, SelectionChangedEvent } from 'ag-grid-community';
import { AgGridVue } from 'ag-grid-vue3';
import {
  useTransmitterApi,
  type ProjectTransponderRow,
  type ProjectTranspondersResponse,
  type Transmitter,
} from '@/composables/tracsNova/useTransmitterApi';
import { useUiStatePersistence } from '@/composables/tracsNova/useUiStatePersistence';

ModuleRegistry.registerModules([AllEnterpriseModule]);

interface TransponderRow extends ProjectTransponderRow {}

const toast = useToast();
const isDark = useDark();
const api = useTransmitterApi();
const ui = useUiStatePersistence('ui_state:tracsNova:db:systems:transponder');
ui.registerGrid('main');

const rows = ref<TransponderRow[]>([]);
const receivers = ref<Transmitter[]>([]);
const transmitters = ref<Transmitter[]>([]);
const receiverPortsByCode = ref<Record<string, string[]>>({});
const receiverFreqsByCode = ref<Record<string, string[]>>({});
const transmitterPortsByCode = ref<Record<string, string[]>>({});
const transmitterFreqsByCode = ref<Record<string, string[]>>({});
const loading = ref(false);
const saving = ref(false);
const selectedCount = ref(0);
const gridApi = shallowRef<GridApi | null>(null);

const defaultColDef: ColDef = {
  resizable: true,
  sortable: false,
  filter: true,
  editable: true,
  minWidth: 130,
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

function unique(values: string[]): string[] {
  return [...new Set(values.map((v) => String(v).trim()).filter((v) => v !== ''))];
}

function extractPorts(system?: Transmitter): string[] {
  const raw = (system?.modulation_details as any)?.ports;
  if (!Array.isArray(raw)) return [];
  const out: string[] = [];
  for (const row of raw) {
    if (Array.isArray(row) && row.length > 0) out.push(String(row[0] ?? ''));
    else out.push(String(row ?? ''));
  }
  return unique(out);
}

function extractFreqLabels(system?: Transmitter): string[] {
  const raw = (system?.modulation_details as any)?.frequencies;
  if (!Array.isArray(raw)) return [];
  const out: string[] = [];
  for (const row of raw) {
    if (Array.isArray(row) && row.length > 0) out.push(String(row[0] ?? ''));
    else out.push(String(row ?? ''));
  }
  return unique(out);
}

function getRxPortOptions(code: string): string[] {
  const key = String(code ?? '').trim();
  const fromCatalog = receiverPortsByCode.value[key] ?? [];
  if (fromCatalog.length > 0) return fromCatalog;
  return extractPorts(getReceiverByCode(key));
}

function getRxFreqOptions(code: string): string[] {
  const key = String(code ?? '').trim();
  const fromCatalog = receiverFreqsByCode.value[key] ?? [];
  if (fromCatalog.length > 0) return fromCatalog;
  return extractFreqLabels(getReceiverByCode(key));
}

function getTxPortOptions(code: string): string[] {
  const key = String(code ?? '').trim();
  const fromCatalog = transmitterPortsByCode.value[key] ?? [];
  if (fromCatalog.length > 0) return fromCatalog;
  return extractPorts(getTransmitterByCode(key));
}

function getTxFreqOptions(code: string): string[] {
  const key = String(code ?? '').trim();
  const fromCatalog = transmitterFreqsByCode.value[key] ?? [];
  if (fromCatalog.length > 0) return fromCatalog;
  return extractFreqLabels(getTransmitterByCode(key));
}

function getReceiverCodeOptions(): string[] {
  return unique(receivers.value.map((r) => String(r.code ?? '')));
}

function getTransmitterCodeOptions(): string[] {
  return unique(transmitters.value.map((t) => String(t.code ?? '')));
}

function getReceiverByCode(code: string): Transmitter | undefined {
  return receivers.value.find((r) => String(r.code ?? '') === String(code ?? ''));
}

function getTransmitterByCode(code: string): Transmitter | undefined {
  return transmitters.value.find((t) => String(t.code ?? '') === String(code ?? ''));
}

const richSelectSearchParams = {
  allowTyping: true,
  filterList: true,
  searchType: 'matchAny' as const,
  highlightMatch: true,
};

const columnDefs = computed<ColDef[]>(() => [
  {
    field: 'name',
    headerName: 'Name',
    editable: true,
    minWidth: 170,
    flex: 1,
  },
  {
    field: 'code',
    headerName: 'Code',
    editable: true,
    minWidth: 140,
    flex: 1,
  },
  {
    field: 'rx_code',
    headerName: 'RxCode',
    editable: true,
    cellEditor: 'agRichSelectCellEditor',
    cellEditorParams: {
      values: getReceiverCodeOptions(),
      ...richSelectSearchParams,
    },
    onCellValueChanged: (p: any) => {
      const selected = getReceiverByCode(String(p.data?.rx_code ?? ''));
      const ports = getRxPortOptions(String(selected?.code ?? p.data?.rx_code ?? ''));
      const freqs = getRxFreqOptions(String(selected?.code ?? p.data?.rx_code ?? ''));
      if (!ports.includes(String(p.data?.rx_port ?? ''))) p.data.rx_port = '';
      if (!freqs.includes(String(p.data?.rx_freq ?? ''))) p.data.rx_freq = '';
      p.api.refreshCells({ rowNodes: [p.node], force: true });
    },
    minWidth: 150,
    flex: 1,
  },
  {
    field: 'rx_port',
    headerName: 'RxPort',
    editable: true,
    cellEditor: 'agRichSelectCellEditor',
    cellEditorParams: (p: any) => ({
      values: getRxPortOptions(String(p?.data?.rx_code ?? '')),
      ...richSelectSearchParams,
    }),
    minWidth: 150,
    flex: 1,
  },
  {
    field: 'rx_freq',
    headerName: 'RxFreq',
    editable: true,
    cellEditor: 'agRichSelectCellEditor',
    cellEditorParams: (p: any) => ({
      values: getRxFreqOptions(String(p?.data?.rx_code ?? '')),
      ...richSelectSearchParams,
    }),
    minWidth: 150,
    flex: 1,
  },
  {
    field: 'tx_code',
    headerName: 'TxCode',
    editable: true,
    cellEditor: 'agRichSelectCellEditor',
    cellEditorParams: {
      values: getTransmitterCodeOptions(),
      ...richSelectSearchParams,
    },
    onCellValueChanged: (p: any) => {
      const selected = getTransmitterByCode(String(p.data?.tx_code ?? ''));
      const ports = getTxPortOptions(String(selected?.code ?? p.data?.tx_code ?? ''));
      const freqs = getTxFreqOptions(String(selected?.code ?? p.data?.tx_code ?? ''));
      if (!ports.includes(String(p.data?.tx_port ?? ''))) p.data.tx_port = '';
      if (!freqs.includes(String(p.data?.tx_freq ?? ''))) p.data.tx_freq = '';
      p.api.refreshCells({ rowNodes: [p.node], force: true });
    },
    minWidth: 150,
    flex: 1,
  },
  {
    field: 'tx_port',
    headerName: 'TxPort',
    editable: true,
    cellEditor: 'agRichSelectCellEditor',
    cellEditorParams: (p: any) => ({
      values: getTxPortOptions(String(p?.data?.tx_code ?? '')),
      ...richSelectSearchParams,
    }),
    minWidth: 150,
    flex: 1,
  },
  {
    field: 'tx_freq',
    headerName: 'TxFreq',
    editable: true,
    cellEditor: 'agRichSelectCellEditor',
    cellEditorParams: (p: any) => ({
      values: getTxFreqOptions(String(p?.data?.tx_code ?? '')),
      ...richSelectSearchParams,
    }),
    minWidth: 150,
    flex: 1,
  },
]);

function onGridReady(event: GridReadyEvent) {
  gridApi.value = event.api;
  ui.onGridReady('main', event);
}

function onSelectionChanged(event: SelectionChangedEvent) {
  selectedCount.value = event.api.getSelectedNodes().length;
}

function addRow() {
  rows.value = [
    ...rows.value,
    {
      name: '',
      code: '',
      rx_code: '',
      rx_port: '',
      rx_freq: '',
      tx_code: '',
      tx_port: '',
      tx_freq: '',
    },
  ];
}

function deleteSelected() {
  const selectedRows = gridApi.value?.getSelectedRows() ?? [];
  if (selectedRows.length === 0) return;
  const selected = new Set(selectedRows);
  const labels = selectedRows
    .map((r) => (r.code ? (r.name ? `${r.name} (${r.code})` : r.code) : r.name))
    .filter(Boolean)
    .slice(0, 5)
    .join(', ');
  const summary = selectedRows.length === 1
    ? labels || 'selected row'
    : `${selectedRows.length} transponders${labels ? `: ${labels}${selectedRows.length > 5 ? ', ...' : ''}` : ''}`;
  const { confirmCriticalDelete } = useConfirmation();
  confirmCriticalDelete('Transponder(s)', summary, () => {
    rows.value = rows.value.filter((row) => !selected.has(row));
    selectedCount.value = 0;
  });
}

function normalizeRow(row: Partial<ProjectTransponderRow>): TransponderRow {
  return {
    name: String(row.name ?? ''),
    code: String(row.code ?? ''),
    rx_code: String(row.rx_code ?? ''),
    rx_port: String(row.rx_port ?? ''),
    rx_freq: String(row.rx_freq ?? ''),
    tx_code: String(row.tx_code ?? ''),
    tx_port: String(row.tx_port ?? ''),
    tx_freq: String(row.tx_freq ?? ''),
  };
}

async function load() {
  loading.value = true;
  try {
    const [rxRes, txRes, transponderRes] = await Promise.all([
      api.getReceivers(),
      api.getTransmitters(),
      api.getProjectTransponders(),
    ]);

    if (rxRes.error.value || txRes.error.value || transponderRes.error.value) {
      toast.add({
        severity: 'error',
        summary: 'Load Failed',
        detail: 'Unable to load transponder management data.',
        life: 3500,
      });
      return;
    }

    receivers.value = Array.isArray(rxRes.data.value) ? (rxRes.data.value as Transmitter[]) : [];
    transmitters.value = Array.isArray(txRes.data.value) ? (txRes.data.value as Transmitter[]) : [];

    const receiverCodes = getReceiverCodeOptions();
    const transmitterCodes = getTransmitterCodeOptions();

    const rxPortEntries = await Promise.all(
      receiverCodes.map(async (code) => {
        const res = await api.getSystemCatalogSystemPorts('receiver', code);
        const values = Array.isArray(res.data.value)
          ? unique((res.data.value as any[]).map((r) => String(r?.port_name ?? '')))
          : [];
        return [code, values] as const;
      }),
    );
    const rxFreqEntries = await Promise.all(
      receiverCodes.map(async (code) => {
        const res = await api.getSystemCatalogSystemFrequencies('receiver', code);
        const values = Array.isArray(res.data.value)
          ? unique((res.data.value as any[]).map((r) => String(r?.frequency_label ?? '')))
          : [];
        return [code, values] as const;
      }),
    );

    const txPortEntries = await Promise.all(
      transmitterCodes.map(async (code) => {
        const res = await api.getSystemCatalogSystemPorts('transmitter', code);
        const values = Array.isArray(res.data.value)
          ? unique((res.data.value as any[]).map((r) => String(r?.port_name ?? '')))
          : [];
        return [code, values] as const;
      }),
    );
    const txFreqEntries = await Promise.all(
      transmitterCodes.map(async (code) => {
        const res = await api.getSystemCatalogSystemFrequencies('transmitter', code);
        const values = Array.isArray(res.data.value)
          ? unique((res.data.value as any[]).map((r) => String(r?.frequency_label ?? '')))
          : [];
        return [code, values] as const;
      }),
    );

    receiverPortsByCode.value = Object.fromEntries(rxPortEntries);
    receiverFreqsByCode.value = Object.fromEntries(rxFreqEntries);
    transmitterPortsByCode.value = Object.fromEntries(txPortEntries);
    transmitterFreqsByCode.value = Object.fromEntries(txFreqEntries);

    const payload = (transponderRes.data.value as ProjectTranspondersResponse) ?? { rows: [] };
    rows.value = (payload.rows ?? []).map((r) => normalizeRow(r));
  } finally {
    loading.value = false;
  }
}

async function save() {
  saving.value = true;
  try {
    const payloadRows: TransponderRow[] = [];
    gridApi.value?.forEachNode((node) => {
      if (node.data) payloadRows.push(normalizeRow(node.data as ProjectTransponderRow));
    });

    const filteredRows = payloadRows.filter((r) => r.name.trim() !== '' || r.code.trim() !== '');

    const res = await api.saveProjectTransponders({ rows: filteredRows });
    if (res.error.value) {
      toast.add({ severity: 'error', summary: 'Save Failed', detail: 'Unable to save transponder rows.', life: 3500 });
      return;
    }

    rows.value = filteredRows;
    toast.add({ severity: 'success', summary: 'Saved', detail: 'Transponder rows updated.', life: 3000 });
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

.actions {
  display: flex;
  gap: 0.5rem;
}

.tp-grid {
  min-height: 0;
}
</style>
