<template>
  <div class="ts-panel">
    <Toast />
    <div class="ts-header">
      <h2>Test Systems / Power Meter</h2>
    </div>

    <div class="ts-section">
      <div class="ts-section-header">
        <h3>Power Meter</h3>
        <div class="actions">
          <Button label="Refresh" size="small" severity="secondary" @click="load" />
          <Button label="Save" size="small" :loading="saving" @click="save" />
        </div>
      </div>

      <ag-grid-vue
        class="ts-grid"
        :style="{ width: '100%', height: gridHeight }"
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
import type { ColDef } from 'ag-grid-community';
import { AgGridVue } from 'ag-grid-vue3';
import {
  useTransmitterApi,
  type ProjectPowerMeterRow,
  type ProjectPowerMetersResponse,
} from '@/composables/tracsNova/useTransmitterApi';
import { useUiStatePersistence } from '@/composables/tracsNova/useUiStatePersistence';

ModuleRegistry.registerModules([AllEnterpriseModule]);

interface PowerMeterRow {
  powerMeter: string;
  channel: 'A' | 'B';
}

const toast = useToast();
const isDark = useDark();
const api = useTransmitterApi();
const ui = useUiStatePersistence('ui_state:tracsNova:db:testSystems:powerMeter');
ui.registerGrid('main');

function onGridReady(event: any) {
  ui.onGridReady('main', event);
}

const rows = ref<PowerMeterRow[]>([]);
const saving = ref(false);

const gridHeight = computed(() => {
  // Include AG Grid header + row-group panel + rows so the last row is not clipped.
  const headerPx = 48;
  const groupPanelPx = 34;
  const rowPx = 42;
  const safetyBufferPx = 14;
  const count = Math.max(rows.value.length, 1);
  const dynamic = headerPx + groupPanelPx + count * rowPx + 4 + safetyBufferPx;
  const bounded = Math.max(210, Math.min(dynamic, 620));
  return `${bounded}px`;
});

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
    field: 'powerMeter',
    headerName: 'PowerMeter',
    editable: false,
    minWidth: 220,
    flex: 1.2,
  },
  {
    field: 'channel',
    headerName: 'Channel',
    editable: true,
    cellEditor: 'agSelectCellEditor',
    cellEditorParams: { values: ['A', 'B'] },
    minWidth: 140,
    flex: 1,
  },
];

function mapRowsFromCatalog(instruments: Record<string, string[]>): PowerMeterRow[] {
  return Object.keys(instruments)
    .filter((name) => name.toLowerCase().includes('powermeter'))
    .sort((a, b) => a.localeCompare(b, undefined, { numeric: true }))
    .map((name) => {
      const key = name.toLowerCase();
      const defaultChannel: 'A' | 'B' = key.includes('downlinkpowermeter') ? 'B' : 'A';
      return { powerMeter: name, channel: defaultChannel };
    });
}

async function load() {
  const res = await api.getProjectPowerMeters();
  if (res.error.value) {
    toast.add({ severity: 'error', summary: 'Load Failed', detail: 'Unable to load power meter rows.', life: 3500 });
    return;
  }

  const payload = (res.data.value as ProjectPowerMetersResponse) ?? { rows: [] };
  rows.value = (payload.rows ?? []).map((r: ProjectPowerMeterRow) => ({
    powerMeter: String(r.powerMeter ?? ''),
    channel: String(r.channel ?? 'A').toUpperCase() === 'B' ? 'B' : 'A',
  }));

  if (rows.value.length === 0) {
    toast.add({ severity: 'info', summary: 'No Data', detail: 'No PowerMeter keys found in Instruments collection.', life: 3000 });
  }
}

async function save() {
  saving.value = true;
  try {
    const payloadRows = rows.value.map((r) => ({
      powerMeter: r.powerMeter,
      channel: r.channel,
    }));

    const res = await api.saveProjectPowerMeters({ rows: payloadRows });
    if (res.error.value) {
      toast.add({ severity: 'error', summary: 'Save Failed', detail: 'Unable to save power meter rows.', life: 3500 });
      return;
    }

    toast.add({ severity: 'success', summary: 'Saved', detail: 'Power meter rows updated.', life: 3000 });
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

.ts-grid { min-height: 0; }
</style>
