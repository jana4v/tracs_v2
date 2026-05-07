<template>
  <div class="obl-panel">
    <Toast />
    <div class="obl-header">
      <h2>On Board Losses / Receiver</h2>
    </div>

    <div class="obl-section">
      <div class="obl-section-header">
        <h3>Loss</h3>
        <div class="actions">
          <Button label="Refresh" size="small" severity="secondary" @click="load" />
          <Button label="Save" size="small" :loading="saving" @click="save" />
        </div>
      </div>

      <ag-grid-vue
        class="obl-grid"
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
import type { ColDef } from 'ag-grid-community';
import { AgGridVue } from 'ag-grid-vue3';
import Button from 'primevue/button';
import { useTransmitterApi } from '@/composables/tracsNova/useTransmitterApi';
import { useUiStatePersistence } from '@/composables/tracsNova/useUiStatePersistence';

ModuleRegistry.registerModules([AllEnterpriseModule]);

interface LossRow {
  id: number;
  source_type: string;
  code: string;
  port: string;
  frequency: string;
  freq_label: string;
  loss_db: string | number | null;
}

const toast = useToast();
const isDark = useDark();
const api = useTransmitterApi();
const ui = useUiStatePersistence('ui_state:tracsNova:db:onboardLosses:receiver');
ui.registerGrid('main');

function onGridReady(event: any) {
  ui.onGridReady('main', event);
}

const rows = ref<LossRow[]>([]);
const saving = ref(false);

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
  { field: 'code', headerName: 'Code', editable: false, minWidth: 130, flex: 1 },
  { field: 'port', headerName: 'Port', editable: false, minWidth: 120, flex: 1 },
  { field: 'freq_label', headerName: 'Freq Label', editable: false, minWidth: 170, flex: 1.2 },
  { field: 'frequency', headerName: 'Frequency (MHz)', editable: false, minWidth: 140, flex: 1 },
  {
    field: 'loss_db',
    headerName: 'Loss(dB)',
    editable: true,
    minWidth: 140,
    flex: 1,
    valueParser: (p: any) => {
      const n = Number(p.newValue);
      return Number.isFinite(n) ? n : p.oldValue;
    },
  },
];

async function load() {
  const res: any = await api.getOnboardLosses('receiver');
  if (res.error.value) {
    toast.add({ severity: 'error', summary: 'Load Failed', detail: 'Unable to load receiver on-board losses.', life: 3500 });
    return;
  }
  rows.value = (res.data.value ?? []) as LossRow[];
}

async function save() {
  saving.value = true;
  try {
    const res: any = await api.saveOnboardLosses({ rows: rows.value as any });
    if (res.error.value) {
      toast.add({ severity: 'error', summary: 'Save Failed', detail: 'Unable to save on-board losses.', life: 3500 });
      return;
    }
    toast.add({ severity: 'success', summary: 'Saved', detail: `On board losses updated (${rows.value.length} rows).`, life: 3000 });
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
.obl-panel {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 4rem);
  padding: 1.5rem;
  color: #e2e8f0;
  box-sizing: border-box;
}

.obl-header h2 {
  font-size: 1.1rem;
  font-weight: 600;
  color: #22d3ee;
  margin: 0 0 1rem;
}

.obl-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.obl-section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.75rem;
  flex-shrink: 0;
}

.obl-section-header h3 {
  font-size: 0.95rem;
  font-weight: 500;
  color: #cbd5e1;
  margin: 0;
}

.actions { display: flex; gap: 0.5rem; }

.obl-grid { flex: 1; min-height: 0; }
</style>
