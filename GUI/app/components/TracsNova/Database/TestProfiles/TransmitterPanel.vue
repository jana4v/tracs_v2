<template>
  <div class="tp-tx-panel">
    <Toast />
    <div class="tp-header">
      <h2>Test Profiles — Transmitter</h2>
    </div>

    <!-- ─── Table 1: Spurious Profiles Definitions ─────────────────── -->
    <div class="tp-section">
      <div class="tp-section-header">
        <h3>Spurious Profiles Definitions</h3>
        <div class="actions">
          <Button
            icon="pi pi-plus"
            label="Add Row"
            size="small"
            severity="secondary"
            @click="addDefinitionRow"
          />
          <Button
            icon="pi pi-trash"
            label="Delete"
            size="small"
            severity="danger"
            :disabled="defSelectedRows.length === 0"
            @click="deleteDefinitionRows"
          />
          <Button label="Save" size="small" :loading="savingDefs" @click="saveDefinitions" />
        </div>
      </div>

      <ag-grid-vue
        class="tp-grid"
        style="width: 100%; height: 220px;"
        :theme="isDark
          ? themeQuartz.withPart(colorSchemeDarkBlue)
          : themeQuartz.withPart(colorSchemeLightCold)"
        :columnDefs="defColumnDefs"
        :rowData="defRows"
        :defaultColDef="defaultColDef"
        rowSelection="multiple"
        :suppressContextMenu="false"
        :suppressMovableColumns="true"
        :undoRedoCellEditing="true"
        :undoRedoCellEditingLimit="20"
        @grid-ready="onDefGridReady"
        @selection-changed="onDefSelectionChanged"
      />
    </div>

    <!-- ─── Table 2: Spurious Profile ──────────────────────────────── -->
    <div class="tp-section">
      <div class="tp-section-header">
        <h3>Spurious Profile</h3>
        <div class="actions">
          <Button label="Refresh" size="small" severity="secondary" @click="loadSpuriousProfile" />
          <Button label="Save" size="small" :loading="savingProfile" @click="saveSpuriousProfile" />
        </div>
      </div>

      <ag-grid-vue
        class="tp-grid"
        style="width: 100%; height: 420px;"
        :theme="isDark
          ? themeQuartz.withPart(colorSchemeDarkBlue)
          : themeQuartz.withPart(colorSchemeLightCold)"
        :columnDefs="spuriousProfileColDefs"
        :rowData="spuriousProfileRows"
        :defaultColDef="defaultColDef"
        :suppressContextMenu="false"
        :suppressMovableColumns="true"
        @grid-ready="onProfileGridReady"
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
import { useTransmitterApi } from '@/composables/tracsNova/useTransmitterApi';
import { useUiStatePersistence } from '@/composables/tracsNova/useUiStatePersistence';

ModuleRegistry.registerModules([AllEnterpriseModule]);

const toast = useToast();
const isDark = useDark();
const api = useTransmitterApi();
const ui = useUiStatePersistence('ui_state:tracsNova:db:testProfiles:transmitter');
ui.registerGrid('definitions');
ui.registerGrid('profile');
const DEFAULT_PROFILE_NAME = 'Detailed';
const profileOptions = ref<string[]>([]);

// ── shared column defaults ─────────────────────────────────────
const defaultColDef: ColDef = {
  resizable: true,
  sortable: true,
  filter: true,
  minWidth: 100,
};

// ══════════════════════════════════════════════════════════════════
// TABLE 1 — Spurious Profiles Definitions
// ══════════════════════════════════════════════════════════════════

interface ProfileDefinitionRow {
  profile_name: string;
  enable: boolean;
  start_frequency: number | null;
  stop_frequency: number | null;
}

const defRows = ref<ProfileDefinitionRow[]>([
  { profile_name: 'Detailed', enable: true, start_frequency: null, stop_frequency: null },
  { profile_name: 'Short',    enable: true, start_frequency: null, stop_frequency: null },
  { profile_name: 'Go/No-Go', enable: true, start_frequency: null, stop_frequency: null },
]);

const defSelectedRows = ref<ProfileDefinitionRow[]>([]);
const savingDefs = ref(false);
const defGridApi = shallowRef<GridApi | null>(null);

const defColumnDefs = computed<ColDef[]>(() => [
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
    minWidth: 170,
    flex: 1,
  },
  {
    field: 'stop_frequency',
    headerName: 'Stop Frequency (MHz)',
    editable: true,
    cellDataType: 'number',
    minWidth: 170,
    flex: 1,
  },
]);

function onDefGridReady(event: GridReadyEvent) {
  defGridApi.value = event.api;
  event.api.sizeColumnsToFit();
  ui.onGridReady('definitions', event);
}

function onDefSelectionChanged(event: any) {
  defSelectedRows.value = event.api.getSelectedRows() as ProfileDefinitionRow[];
}

function addDefinitionRow() {
  const defaultProfileName = profileOptions.value[0] ?? DEFAULT_PROFILE_NAME;
  defRows.value = [
    ...defRows.value,
    { profile_name: defaultProfileName, enable: true, start_frequency: null, stop_frequency: null },
  ];
}

function deleteDefinitionRows() {
  const selected = defGridApi.value?.getSelectedRows() ?? [];
  defRows.value = defRows.value.filter((r) => !selected.includes(r));
  defSelectedRows.value = [];
}

async function saveDefinitions() {
  savingDefs.value = true;
  try {
    const rows: ProfileDefinitionRow[] = [];
    defGridApi.value?.forEachNode((n) => { if (n.data) rows.push(n.data as ProfileDefinitionRow); });
    const bands = rows.map((r) => ({
      profile_name: String(r.profile_name ?? ''),
      enable: Boolean(r.enable),
      start_frequency: r.start_frequency === null || r.start_frequency === undefined || r.start_frequency === ('' as any)
        ? null
        : Number(r.start_frequency),
      stop_frequency: r.stop_frequency === null || r.stop_frequency === undefined || r.stop_frequency === ('' as any)
        ? null
        : Number(r.stop_frequency),
    }));
    const res = await api.saveSpuriousBandConfigs(bands);
    if (res.error.value) {
      toast.add({ severity: 'error', summary: 'Save Failed', detail: 'Unable to save Spurious Profiles Definitions.', life: 3500 });
      return;
    }
    toast.add({ severity: 'success', summary: 'Saved', detail: 'Spurious Profiles Definitions saved.', life: 3000 });
  } finally {
    savingDefs.value = false;
  }
}

async function loadSpuriousBandConfigs() {
  const res = await api.getSpuriousBandConfigs();
  if (res.error.value) return;
  const data = res.data.value as { bands?: ProfileDefinitionRow[] } | null;
  const bands = Array.isArray(data?.bands) ? data!.bands : [];
  if (bands.length > 0) {
    defRows.value = bands.map((b) => ({
      profile_name: String(b.profile_name ?? ''),
      enable: Boolean(b.enable),
      start_frequency: b.start_frequency ?? null,
      stop_frequency: b.stop_frequency ?? null,
    }));
  }
}

// ══════════════════════════════════════════════════════════════════
// TABLE 2 — Spurious Profile
// ══════════════════════════════════════════════════════════════════

interface SpuriousProfileRow {
  code: string;
  port: string;
  frequency_label: string;
  frequency: number | null;
  profiles: string[];
}

const spuriousProfileRows = ref<SpuriousProfileRow[]>([]);
const savingProfile = ref(false);
const profileGridApi = shallowRef<GridApi | null>(null);

const spuriousProfileColDefs = computed<ColDef[]>(() => [
  { field: 'code',            headerName: 'Code',            editable: false, minWidth: 80  },
  { field: 'port',            headerName: 'Port',            editable: false, minWidth: 80  },
  { field: 'frequency_label', headerName: 'Freq Label',      editable: false, minWidth: 110 },
  { field: 'frequency',       headerName: 'Frequency (MHz)', editable: false, minWidth: 140 },
  {
    field: 'profiles',
    headerName: 'Profile',
    editable: true,
    cellEditor: 'agRichSelectCellEditor',
    cellEditorParams: {
      values: [...profileOptions.value],
      multiSelect: true,
      searchType: 'matchAny',
      filterList: true,
    },
    valueFormatter: (params) =>
      Array.isArray(params.value) ? params.value.join(', ') : (params.value ?? ''),
    minWidth: 220,
    flex: 1,
  },
]);

async function loadTestPlanTypes() {
  const res = await api.getTestPlanTypes();
  if (res.error.value) {
    toast.add({ severity: 'error', summary: 'Load Failed', detail: 'Unable to load test plan types.', life: 3500 });
    return;
  }

  const options = Array.isArray(res.data.value) ? res.data.value as string[] : [];
  profileOptions.value = options;
  if (options.length > 0) {
    defRows.value = options.map((profile_name) => ({
      profile_name,
      enable: true,
      start_frequency: null,
      stop_frequency: null,
    }));
  }
}

function onProfileGridReady(event: GridReadyEvent) {
  profileGridApi.value = event.api;
  event.api.sizeColumnsToFit();
  ui.onGridReady('profile', event);
}

// Cache the full server-side spurious rows so we can preserve fields the
// grid does not display (specification, tolerance, fbt*, etc.) when saving.
interface SpuriousServerRow {
  transmitter_code: string;
  row: Record<string, any>;
}
const spuriousServerRows = ref<SpuriousServerRow[]>([]);

function rowKey(code: string, port: string, label: string, freq: string | number | null): string {
  return `${code}|${port}|${label}|${freq ?? ''}`;
}

async function loadSpuriousProfile() {
  const res = await api.getParameterRows('spurious');
  if (res.error.value) {
    toast.add({ severity: 'error', summary: 'Load Failed', detail: 'Unable to load Spurious Profile rows.', life: 3500 });
    return;
  }
  const data = res.data.value as { rows?: Array<{ transmitter_code: string; row: Record<string, any> }> } | null;
  const rows = Array.isArray(data?.rows) ? data!.rows : [];
  spuriousServerRows.value = rows.map((r) => ({ transmitter_code: r.transmitter_code, row: { ...r.row } }));
  spuriousProfileRows.value = rows.map((r) => ({
    code: String(r.row?.code ?? r.transmitter_code ?? ''),
    port: String(r.row?.port ?? ''),
    frequency_label: String(r.row?.frequency_label ?? ''),
    frequency: r.row?.frequency === '' || r.row?.frequency === null || r.row?.frequency === undefined
      ? null
      : Number(r.row.frequency),
    profiles: Array.isArray(r.row?.profiles) ? [...r.row.profiles] : [],
  }));
}

async function saveSpuriousProfile() {
  savingProfile.value = true;
  try {
    const editedRows: SpuriousProfileRow[] = [];
    profileGridApi.value?.forEachNode((n) => { if (n.data) editedRows.push(n.data as SpuriousProfileRow); });

    // Build a map of edited profiles[] keyed by (code|port|label|freq).
    const editedByKey = new Map<string, string[]>();
    for (const r of editedRows) {
      editedByKey.set(
        rowKey(r.code, r.port, r.frequency_label, r.frequency),
        Array.isArray(r.profiles) ? [...r.profiles] : [],
      );
    }

    // Merge edits into the cached server rows so we keep all other fields.
    const payloadRows = spuriousServerRows.value.map((sr) => {
      const key = rowKey(
        String(sr.row?.code ?? sr.transmitter_code ?? ''),
        String(sr.row?.port ?? ''),
        String(sr.row?.frequency_label ?? ''),
        sr.row?.frequency ?? '',
      );
      const editedProfiles = editedByKey.get(key);
      const mergedRow = editedProfiles !== undefined
        ? { ...sr.row, profiles: editedProfiles }
        : sr.row;
      return { transmitter_code: sr.transmitter_code, row: mergedRow };
    });

    const res = await api.saveParameterRows('spurious', { rows: payloadRows });
    if (res.error.value) {
      toast.add({ severity: 'error', summary: 'Save Failed', detail: 'Unable to save Spurious Profile.', life: 3500 });
      return;
    }
    toast.add({ severity: 'success', summary: 'Saved', detail: 'Spurious Profile saved.', life: 3000 });
  } finally {
    savingProfile.value = false;
  }
}

onMounted(async () => {
  // Sequence test plan types (fallback definitions) before saved band configs
  // so that saved bands override the fallback rather than racing with it.
  await loadTestPlanTypes();
  await loadSpuriousBandConfigs();
  await loadSpuriousProfile();
  await ui.load();
});
</script>

<style scoped>
.tp-tx-panel {
  padding: 1.5rem;
  color: #e2e8f0;
}

.tp-header h2 {
  font-size: 1.2rem;
  font-weight: 600;
  color: #22d3ee;
  margin: 0 0 1.5rem;
}

.tp-section {
  margin-bottom: 2rem;
}

.tp-section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.75rem;
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
</style>

