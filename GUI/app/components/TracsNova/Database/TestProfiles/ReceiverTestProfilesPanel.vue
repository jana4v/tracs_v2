<template>
  <div class="rtp-panel">
    <Toast />
    <div class="rtp-header">
      <h2>Test Profiles — Receiver</h2>
    </div>

    <!-- Profile type selector -->
    <div class="rtp-selector">
      <label class="selector-label">Profile Type</label>
      <Select
        v-model="selectedProfileType"
        :options="profileTypeOptions"
        option-label="label"
        option-value="value"
        style="min-width: 220px"
        @change="loadProfile"
      />
    </div>

    <!-- Grid section -->
    <div class="rtp-section">
      <div class="rtp-section-header">
        <h3>{{ selectedProfileLabel }}</h3>
        <div class="actions">
          <Button
            icon="pi pi-refresh"
            label="Refresh"
            size="small"
            severity="secondary"
            :loading="loading"
            @click="loadProfile"
          />
          <Button label="Save" size="small" :loading="saving" @click="save" />
        </div>
      </div>

      <ag-grid-vue
        class="rtp-grid"
        style="width: 100%;"
        :theme="isDark
          ? themeQuartz.withPart(colorSchemeDarkBlue)
          : themeQuartz.withPart(colorSchemeLightCold)"
        :columnDefs="columnDefs"
        :rowData="rows"
        :defaultColDef="defaultColDef"
        :cellSelection="cellSelection"
        :suppressContextMenu="false"
        :suppressMovableColumns="true"
        :undoRedoCellEditing="true"
        :undoRedoCellEditingLimit="20"
        @grid-ready="onGridReady"
        @cell-double-clicked="onCellDoubleClicked"
      />
    </div>

    <!-- Levels editor dialog -->
    <Dialog
      v-model:visible="showLevelsDialog"
      modal
      :header="levelsDialogTitle"
      :style="{ width: '480px' }"
      :dismissableMask="true"
    >
      <div class="levels-editor-content">
        <HotTable :settings="hotSettings" :data="levelsEditingData" />
      </div>
      <template #footer>
        <Button label="Cancel" icon="pi pi-times" text @click="closeLevelsEditor" />
        <Button label="Save" icon="pi pi-check" @click="saveLevelsEditor" />
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
import Select from 'primevue/select';
import Dialog from 'primevue/dialog';
import Button from 'primevue/button';
import { HotTable } from '@handsontable/vue3';
import { registerAllModules } from 'handsontable/registry';
import 'handsontable/styles/handsontable.css';
import 'handsontable/styles/ht-theme-main-no-icons.css';
import { useTransmitterApi } from '@/composables/tracsNova/useTransmitterApi';
import { useUiStatePersistence } from '@/composables/tracsNova/useUiStatePersistence';

registerAllModules();
ModuleRegistry.registerModules([AllEnterpriseModule]);

// ── Types ─────────────────────────────────────────────────────────────────────

interface ProfileRow {
  profile_name: string;
  levels: (number | string)[][];
  establish: boolean;
  no_of_cmds_at_threshold: number;
}

// ── State ─────────────────────────────────────────────────────────────────────

const toast = useToast();
const isDark = useDark();
const api = useTransmitterApi();
const ui = useUiStatePersistence('ui_state:tracsNova:db:testProfiles:receiver');
ui.registerGrid('main');

const profileTypeOptions = [
  { label: 'Command Threshold', value: 'command_threshold' },
];

const selectedProfileType = ref<string>('command_threshold');
const rows = ref<ProfileRow[]>([]);
const loading = ref(false);const saving = ref(false);
const gridApi = shallowRef<GridApi | null>(null);

const selectedProfileLabel = computed(
  () => profileTypeOptions.find((o) => o.value === selectedProfileType.value)?.label ?? '',
);

// ── Grid config ───────────────────────────────────────────────────────────────

const defaultColDef: ColDef = {
  resizable: true,
  sortable: false,
  filter: false,
  minWidth: 100,
};

/** Fill-handle enabled for levels, establish, no_of_cmds_at_threshold (cols 2-4) */
const cellSelection = {
  mode: 'range' as const,
  handle: {
    mode: 'fill' as const,
    direction: 'xy' as const,
    suppressClearOnFillReduction: true,
  },
};

const columnDefs = computed<ColDef[]>(() => [
  {
    field: 'profile_name',
    headerName: 'Profile Name',
    editable: false,
    flex: 1,
    minWidth: 140,
  },
  {
    field: 'levels',
    headerName: 'Levels',
    editable: false,
    flex: 2,
    minWidth: 260,
    cellClass: 'levels-cell',
    valueFormatter: (p: any) => formatLevels(p.value),
    tooltipValueGetter: (p: any) => formatLevels(p.value),
  },
  {
    field: 'establish',
    headerName: 'Establish',
    editable: true,
    cellRenderer: 'agCheckboxCellRenderer',
    cellEditor: 'agCheckboxCellEditor',
    minWidth: 110,
    maxWidth: 130,
  },
  {
    field: 'no_of_cmds_at_threshold',
    headerName: 'NoOfCmdsAtThreshold',
    editable: true,
    minWidth: 190,
    flex: 1,
    valueParser: (p: any) => {
      const n = Number(p.newValue);
      return Number.isFinite(n) ? n : p.oldValue;
    },
  },
]);

function formatLevels(value: unknown): string {
  if (!Array.isArray(value)) return '';
  return (value as any[][]).map((r) => `${r?.[0] ?? ''},${r?.[1] ?? ''}`).join('; ');
}

// ── Data loading ──────────────────────────────────────────────────────────────

async function loadProfile() {
  loading.value = true;
  try {
    const res: any = await api.getReceiverTestProfile(selectedProfileType.value);
    if (res.error.value) {
      toast.add({ severity: 'error', summary: 'Load failed', detail: 'Unable to load receiver test profiles.', life: 4000 });
      return;
    }
    rows.value = (res.data.value?.rows ?? []) as ProfileRow[];
  } catch (err: any) {
    toast.add({ severity: 'error', summary: 'Load failed', detail: err?.message ?? String(err), life: 4000 });
  } finally {
    loading.value = false;
  }
}

// ── Save ──────────────────────────────────────────────────────────────────────

async function save() {
  saving.value = true;
  try {
    // Collect all row data from the grid (may have been edited in-place via fill-handle)
    const currentRows: ProfileRow[] = [];
    gridApi.value?.forEachNode((node) => {
      if (node.data) currentRows.push(node.data as ProfileRow);
    });

    const res: any = await api.saveReceiverTestProfile({
      profile_type: selectedProfileType.value,
      rows: currentRows,
    });

    if (res.error.value) {
      toast.add({ severity: 'error', summary: 'Save failed', detail: 'Unable to save receiver test profiles.', life: 4000 });
      return;
    }

    toast.add({ severity: 'success', summary: 'Saved', detail: 'Receiver test profile saved.', life: 3000 });
  } catch (err: any) {
    toast.add({ severity: 'error', summary: 'Save failed', detail: err?.message ?? String(err), life: 4000 });
  } finally {
    saving.value = false;
  }
}

// ── Grid ready ────────────────────────────────────────────────────────────────

function onGridReady(params: GridReadyEvent) {
  gridApi.value = params.api;
  ui.onGridReady('main', params);
}

// ── Levels popup editor ──────────────────────────────────────────────────────

const showLevelsDialog = ref(false);
const levelsEditingData = ref<(number | string)[][]>([]);
const levelsEditingRowIndex = ref<number>(-1);

const levelsDialogTitle = computed(() => {
  const idx = levelsEditingRowIndex.value;
  const name = idx >= 0 && idx < rows.value.length ? rows.value[idx]?.profile_name : '';
  return `Edit Levels${name ? ` — ${name}` : ''}`;
});

const hotSettings = computed(() => ({
  licenseKey: 'non-commercial-and-evaluation',
  colHeaders: ['Level (dBm)', 'Number of Commands'],
  columns: [
    { type: 'numeric' },
    { type: 'numeric' },
  ],
  rowHeaders: true,
  stretchH: 'all',
  width: '100%',
  height: 320,
  minRows: 1,
  minSpareRows: 1,
  contextMenu: true,
  fillHandle: { direction: 'vertical' as const, autoInsertRow: true },
  copyPaste: true,
}));

function onCellDoubleClicked(event: CellDoubleClickedEvent) {
  if (event.colDef?.field !== 'levels') return;
  const idx = event.rowIndex ?? -1;
  if (idx < 0) return;
  levelsEditingRowIndex.value = idx;
  const current = (event.data?.levels ?? []) as (number | string)[][];
  levelsEditingData.value = JSON.parse(JSON.stringify(
    Array.isArray(current) && current.length > 0 ? current : [[-60, 10]],
  ));
  showLevelsDialog.value = true;
}

function closeLevelsEditor() {
  showLevelsDialog.value = false;
}

function saveLevelsEditor() {
  const cleaned = levelsEditingData.value.filter(
    (r) => r.some((c) => c !== '' && c !== null && c !== undefined),
  );
  const final = cleaned.length > 0 ? cleaned : [[-60, 10]];
  const idx = levelsEditingRowIndex.value;
  if (idx >= 0 && idx < rows.value.length) {
    rows.value[idx] = { ...rows.value[idx], levels: final };
    nextTick(() => {
      const node = gridApi.value?.getDisplayedRowAtIndex(idx);
      if (node) {
        node.setData(rows.value[idx]);
      }
    });
  }
  showLevelsDialog.value = false;
}

// ── Mount ─────────────────────────────────────────────────────────────────────

onMounted(async () => {
  ui.bindRefs({ selectedProfileType });
  await ui.load();
  await loadProfile();
});
</script>

<style scoped>
.rtp-panel {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 4rem);
  padding: 1.5rem;
  box-sizing: border-box;
  overflow: hidden;
}

.rtp-header h2 {
  margin: 0 0 1rem;
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text-color);
}

.rtp-selector {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 1rem;
  flex-shrink: 0;
}

.selector-label {
  font-size: 0.9rem;
  font-weight: 500;
  color: var(--text-color-secondary);
  white-space: nowrap;
}

.rtp-section {
  display: flex;
  flex-direction: column;
}

.rtp-section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.75rem;
  flex-shrink: 0;
}

.rtp-section-header h3 {
  margin: 0;
  font-size: 0.95rem;
  font-weight: 500;
  color: var(--text-color-secondary);
}

.actions {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.rtp-grid {
  /* header ~42px + 3 rows ~42px each = ~168px total */
  height: 170px;
  width: 100%;
}

.levels-editor-content {
  margin: 0.75rem 0;
}

:deep(.levels-cell) {
  cursor: pointer;
}
</style>
