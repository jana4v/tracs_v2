<template>
  <div class="rt-panel">
    <Toast />
    <div class="rt-header">
      <h2>Specifications — Ranging Threshold</h2>
      <div class="rt-actions">
        <Button icon="pi pi-refresh" label="Refresh" size="small" severity="secondary" :loading="loading" @click="load" />
        <Button label="Save" size="small" :loading="saving" @click="save" />
      </div>
    </div>

    <ag-grid-vue
      class="rt-grid"
      style="width: 100%; height: calc(100vh - 9rem);"
      :theme="isDark ? themeQuartz.withPart(colorSchemeDarkBlue) : themeQuartz.withPart(colorSchemeLightCold)"
      :columnDefs="columnDefs"
      :rowData="rows"
      :defaultColDef="defaultColDef"
      :cellSelection="cellSelectionConfig"
      :enableRangeSelection="true"
      :enableFillHandle="true"
      :suppressContextMenu="false"
      :suppressMovableColumns="true"
      :suppressColumnVirtualisation="true"
      :undoRedoCellEditing="true"
      :undoRedoCellEditingLimit="20"
      @grid-ready="onGridReady"
      @cell-double-clicked="onCellDoubleClicked"
    />

    <!-- FBT editor dialog -->
    <Dialog
      v-model:visible="showFbtDialog"
      modal
      :header="fbtDialogTitle"
      :style="{ width: '620px' }"
      :dismissableMask="true"
    >
      <div class="editor-wrap">
        <HotTable :settings="fbtHotSettings" :data="fbtEditingData" />
      </div>
      <template #footer>
        <Button label="Cancel" icon="pi pi-times" text @click="closeFbtEditor" />
        <Button label="Save" icon="pi pi-check" @click="saveFbtEditor" />
      </template>
    </Dialog>
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
import type { ColDef, GridApi, GridReadyEvent, CellDoubleClickedEvent } from 'ag-grid-community';
import { AgGridVue } from 'ag-grid-vue3';
import Dialog from 'primevue/dialog';
import Button from 'primevue/button';
import { HotTable } from '@handsontable/vue3';
import { registerAllModules } from 'handsontable/registry';
import 'handsontable/styles/handsontable.css';
import 'handsontable/styles/ht-theme-main-no-icons.css';
import {
  useTransmitterApi,
  type RangingThresholdRow,
  type RangingTone,
} from '@/composables/tracsNova/useTransmitterApi';
import { useUiStatePersistence } from '@/composables/tracsNova/useUiStatePersistence';

registerAllModules();
ModuleRegistry.registerModules([AllEnterpriseModule]);

type FbtMatrix = (string | number)[][];
type FbtField = 'fbt' | 'fbt_hot' | 'fbt_cold';

const api = useTransmitterApi();
const toast = useToast();
const isDark = useDark();
const ui = useUiStatePersistence('ui_state:tracsNova:db:specs:rangingThreshold');
ui.registerGrid('main');

const rows = ref<RangingThresholdRow[]>([]);
const tones = ref<RangingTone[]>([]);
const loading = ref(false);
const saving = ref(false);
const gridApi = shallowRef<GridApi | null>(null);

// FBT dialog state
const showFbtDialog = ref(false);
const fbtEditingData = ref<FbtMatrix>([['', '']]);
const activeFbtField = ref<FbtField>('fbt');
const activeFbtNode = shallowRef<any>(null);

const fbtDialogTitle = computed(() => {
  const display = activeFbtField.value === 'fbt' ? 'FBT' : activeFbtField.value === 'fbt_hot' ? 'FBT Hot' : 'FBT Cold';
  const row = (activeFbtNode.value?.data ?? {}) as RangingThresholdRow;
  const id = [row.transponder_code, row.tone_khz ? `${row.tone_khz}KHz` : ''].filter(Boolean).join(' / ');
  return id ? `Edit ${display} — ${id}` : `Edit ${display}`;
});

function isFbtField(field: unknown): field is FbtField {
  return field === 'fbt' || field === 'fbt_hot' || field === 'fbt_cold';
}

function ensureFbtMatrix(value: unknown): FbtMatrix {
  if (Array.isArray(value) && value.length > 0 && value.every((r) => Array.isArray(r))) {
    return value as FbtMatrix;
  }
  return [['', '']];
}

const fbtHotSettings = computed(() => ({
  licenseKey: 'non-commercial-and-evaluation',
  colHeaders: ['Offset (kHz)', 'Value (dBc)'],
  columns: [
    { type: 'numeric', locale: 'en-US', numericFormat: { minimumFractionDigits: 2, maximumFractionDigits: 2, useGrouping: true } },
    { type: 'numeric', locale: 'en-US', numericFormat: { minimumFractionDigits: 2, maximumFractionDigits: 2, useGrouping: true } },
  ],
  rowHeaders: true,
  stretchH: 'all',
  width: '100%',
  minRows: 1,
  minSpareRows: 1,
  contextMenu: true,
  height: 380,
  autoWrapRow: true,
  autoWrapCol: true,
  copyPaste: true,
  fillHandle: { direction: 'vertical', autoInsertRow: true },
  enterMoves: { row: 1, col: 0 },
  tabMoves: { row: 0, col: 1 },
}));

// Column definitions
const columnDefs = computed<ColDef[]>(() => {
  const toneValues = tones.value.map((t) => t.tone_khz);

  return [
    {
      field: 'transponder_code',
      headerName: 'TpCode',
      editable: false,
      minWidth: 100,
      pinned: 'left',
    },
    {
      field: 'uplink',
      headerName: 'Uplink',
      editable: false,
      minWidth: 160,
    },
    {
      field: 'downlink',
      headerName: 'Downlink',
      editable: false,
      minWidth: 160,
    },
    {
      field: 'max_input_power',
      headerName: 'MaxInput (dBm)',
      editable: true,
      minWidth: 130,
      suppressFillHandle: false,
      cellEditor: 'agNumberCellEditor',
    },
    {
      field: 'tone_khz',
      headerName: 'Tone (KHz)',
      editable: true,
      minWidth: 110,
      suppressFillHandle: false,
      cellEditor: 'agRichSelectCellEditor',
      cellEditorParams: {
        values: toneValues,
        searchType: 'match',
        allowTyping: true,
      },
    },
    {
      field: 'specification',
      headerName: 'Specification (dBm)',
      editable: true,
      minWidth: 120,
      suppressFillHandle: false,
      cellEditor: 'agNumberCellEditor',
    },
    {
      field: 'tolerance',
      headerName: 'Tolerance (dBm)',
      editable: true,
      minWidth: 110,
      suppressFillHandle: false,
      cellEditor: 'agNumberCellEditor',
    },
    {
      field: 'fbt',
      headerName: 'Fbt (dBm)',
      editable: false,
      minWidth: 80,
      suppressFillHandle: true,
      valueFormatter: (p) => {
        const m = ensureFbtMatrix(p.value);
        const filled = m.filter((r) => r.some((c) => c !== '' && c !== null));
        return filled.length > 0 ? `[${filled.length} pts]` : '';
      },
    },
    {
      field: 'fbt_hot',
      headerName: 'Fbt Hot (dBm)',
      editable: false,
      minWidth: 90,
      suppressFillHandle: true,
      valueFormatter: (p) => {
        const m = ensureFbtMatrix(p.value);
        const filled = m.filter((r) => r.some((c) => c !== '' && c !== null));
        return filled.length > 0 ? `[${filled.length} pts]` : '';
      },
    },
    {
      field: 'fbt_cold',
      headerName: 'Fbt Cold (dBm)',
      editable: false,
      minWidth: 90,
      suppressFillHandle: true,
      valueFormatter: (p) => {
        const m = ensureFbtMatrix(p.value);
        const filled = m.filter((r) => r.some((c) => c !== '' && c !== null));
        return filled.length > 0 ? `[${filled.length} pts]` : '';
      },
    },
  ];
});

const defaultColDef: ColDef = {
  sortable: true,
  filter: true,
  resizable: true,
  minWidth: 80,
};

const cellSelectionConfig = {
  mode: 'range' as const,
  handle: {
    mode: 'fill' as const,
    direction: 'xy' as const,
    suppressClearOnFillReduction: true,
  },
};

function onGridReady(event: GridReadyEvent) {
  gridApi.value = event.api;
  ui.onGridReady('main', event);
}

function onCellDoubleClicked(event: CellDoubleClickedEvent) {
  const field = event.colDef?.field ?? '';
  if (!isFbtField(field)) return;

  const target = event.event?.target as HTMLElement | null;
  if (target?.closest?.('.ag-fill-handle')) return;

  activeFbtField.value = field;
  activeFbtNode.value = event.node;
  fbtEditingData.value = ensureFbtMatrix(event.value).map((r) => [...r]);
  showFbtDialog.value = true;
}

function closeFbtEditor() {
  showFbtDialog.value = false;
}

async function saveFbtEditor() {
  const cleaned = fbtEditingData.value.filter((r) => r.some((c) => c !== '' && c !== null));
  const final = cleaned.length > 0 ? cleaned : [['', '']];
  if (activeFbtField.value && activeFbtNode.value) {
    activeFbtNode.value.setDataValue(activeFbtField.value, final);
  }
  showFbtDialog.value = false;
}

async function load() {
  loading.value = true;
  try {
    const [tonesRes, rowsRes] = await Promise.all([
      api.getRangingTones(),
      api.getRangingThresholdRows(),
    ]);

    if (tonesRes.error.value || rowsRes.error.value) {
      toast.add({ severity: 'error', summary: 'Load Failed', detail: 'Unable to load ranging threshold data.', life: 3500 });
      return;
    }

    tones.value = Array.isArray(tonesRes.data.value) ? (tonesRes.data.value as RangingTone[]) : [];
    const rawRows = Array.isArray(rowsRes.data.value) ? (rowsRes.data.value as RangingThresholdRow[]) : [];

    // Attach tone_khz display value from tones list
    const toneById = new Map(tones.value.map((t) => [t.id, t.tone_khz]));
    rows.value = rawRows.map((r) => ({
      ...r,
      tone_khz: toneById.get(r.tone_id) ?? '',
    }));
  } finally {
    loading.value = false;
  }
}

async function save() {
  saving.value = true;
  try {
    const gridRows: RangingThresholdRow[] = [];
    gridApi.value?.forEachNode((n) => { if (n.data) gridRows.push(n.data as RangingThresholdRow); });

    // Resolve tone_id back from tone_khz if user changed the dropdown
    const toneByKhz = new Map(tones.value.map((t) => [t.tone_khz, t.id]));
    const toSave = gridRows.map((r) => ({
      ...r,
      tone_id: (r.tone_khz ? toneByKhz.get(r.tone_khz) : undefined) ?? r.tone_id,
    }));

    const res = await api.saveRangingThresholdRows({ rows: toSave });
    if (res.error.value) {
      toast.add({ severity: 'error', summary: 'Save Failed', detail: 'Unable to save ranging threshold rows.', life: 3500 });
      return;
    }

    toast.add({ severity: 'success', summary: 'Saved', detail: `Ranging threshold updated (${toSave.length} rows).`, life: 3000 });
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
.rt-panel {
  padding: 1rem;
  display: flex;
  flex-direction: column;
  height: 100%;
}

.rt-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.75rem;
}

.rt-header h2 {
  margin: 0;
  font-size: 1.1rem;
  font-weight: 600;
}

.rt-actions {
  display: flex;
  gap: 0.5rem;
}

.editor-wrap {
  padding: 0.5rem 0;
}
</style>
