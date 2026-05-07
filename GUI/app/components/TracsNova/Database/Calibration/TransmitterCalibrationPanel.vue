<template>
  <div class="cal-panel">
    <Toast />
    <div class="cal-header">
      <h2>Calibration / Transmitter</h2>
    </div>

    <div class="cal-section">
      <div class="cal-section-header">
        <h3>Calibration</h3>
        <div class="actions">
          <label class="cal-id-label" for="cal-id-select">Cal ID:</label>
          <Select
            id="cal-id-select"
            v-model="selectedCalId"
            :options="calIdOptions"
            placeholder="Select Cal ID"
            class="cal-id-select"
            :loading="calIdsLoading"
            showClear
            @change="onCalIdChange"
          />
          <Button label="Refresh" size="small" severity="secondary" @click="load" />
          <Button label="Save" size="small" :loading="saving" @click="save" />
        </div>
      </div>

      <ag-grid-vue
        class="cal-grid"
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
        rowGroupPanelShow="always"
        groupDisplayType="singleColumn"
        @cell-value-changed="onCellValueChanged"
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
import type { CellValueChangedEvent, ColDef } from 'ag-grid-community';
import { AgGridVue } from 'ag-grid-vue3';
import Select from 'primevue/select';
import {
  useTransmitterApi,
  type CalibrationRowsResponse,
} from '@/composables/tracsNova/useTransmitterApi';
import {
  useCalibrationDataApi,
  type MeasureOptionsResponse,
  type DownlinkCalDataRowsResponse,
} from '@/composables/tracsNova/useCalibrationDataApi';
import {
  toNumber,
  formatNumber,
  absLossFormatter,
  buildKey,
  computeFspl,
} from '@/composables/tracsNova/utils';
import { useUiStatePersistence } from '@/composables/tracsNova/useUiStatePersistence';

ModuleRegistry.registerModules([AllEnterpriseModule]);

interface CalibrationGridRow {
  transmitter_code: string;
  code: string;
  port: string;
  frequency_label: string;
  frequency: string;
  system_loss: string;
  fixed_pad_loss: string;
  antenna_gain: string;
  ground_antenna_gain: string;
  distance: string;
  fspl: string;
  total_loss: string;
}

const toast = useToast();
const isDark = useDark();
const api = useTransmitterApi();
const calibrationDataApi = useCalibrationDataApi();
const ui = useUiStatePersistence('ui_state:tracsNova:db:calibration:transmitter');
ui.registerGrid('main');

function onGridReady(event: any) {
  ui.onGridReady('main', event);
}

const rows = ref<CalibrationGridRow[]>([]);
const saving = ref(false);

const calIdOptions = ref<string[]>([]);
const selectedCalId = ref<string | null>(null);
const calIdsLoading = ref(false);
ui.bindRefs({ selectedCalId });
// Map keyed by `${code}|${port}|${frequencyHz}` -> system_loss (dB) from downlink-cal.
const downlinkLossMap = ref<Map<string, number>>(new Map());

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
  { field: 'frequency_label', headerName: 'Frequency Label', editable: false, minWidth: 170, flex: 1.2 },
  { field: 'frequency', headerName: 'Frequency (MHz)', editable: false, minWidth: 140, flex: 1 },
  { field: 'system_loss', headerName: 'System Loss(dB)', editable: false, minWidth: 160, flex: 1, valueFormatter: absLossFormatter },
  { field: 'fixed_pad_loss', headerName: 'Fixed Pad Loss(dB)', editable: true, minWidth: 170, flex: 1.1, valueFormatter: absLossFormatter },
  { field: 'antenna_gain', headerName: 'Antenna Gain(dBi)', editable: true, minWidth: 170, flex: 1.1 },
  { field: 'ground_antenna_gain', headerName: 'Ground Antenna Gain(dB)', editable: true, minWidth: 200, flex: 1.2 },
  { field: 'distance', headerName: 'Distance(m)', editable: true, minWidth: 140, flex: 1 },
  { field: 'fspl', headerName: 'FSPL(dB)', editable: false, minWidth: 130, flex: 1, valueFormatter: absLossFormatter },
  { field: 'total_loss', headerName: 'Total Loss(dB)', editable: false, minWidth: 150, flex: 1, valueFormatter: absLossFormatter },
];

function calculateRowDerived(row: CalibrationGridRow): CalibrationGridRow {
  const fspl = computeFspl(toNumber(row.distance), toNumber(row.frequency));
  const total =
    Math.abs(toNumber(row.antenna_gain)) +
    Math.abs(toNumber(row.ground_antenna_gain)) -
    Math.abs(fspl) -
    Math.abs(toNumber(row.system_loss)) -
    Math.abs(toNumber(row.fixed_pad_loss));
  return {
    ...row,
    fspl: formatNumber(fspl),
    total_loss: formatNumber(total),
  };
}

function applyDownlinkLoss(row: CalibrationGridRow): CalibrationGridRow {
  const key = buildKey(row.code, row.port, row.frequency);
  const value = downlinkLossMap.value.get(key);
  // If downlink-cal entry exists for this channel, use it. Otherwise default to 0.
  return { ...row, system_loss: value !== undefined ? formatNumber(value) : '0' };
}

function mapRows(payload: CalibrationRowsResponse): CalibrationGridRow[] {
  return (payload.rows ?? []).map((item) => {
    const row: CalibrationGridRow = {
      transmitter_code: String(item.transmitter_code ?? ''),
      code: String(item.row?.code ?? ''),
      port: String(item.row?.port ?? ''),
      frequency_label: String(item.row?.frequency_label ?? ''),
      frequency: String(item.row?.frequency ?? ''),
      system_loss: String(item.row?.system_loss ?? '0'),
      fixed_pad_loss: String(item.row?.fixed_pad_loss ?? '0'),
      antenna_gain: String(item.row?.antenna_gain ?? '0'),
      ground_antenna_gain: String((item.row as any)?.ground_antenna_gain ?? '0'),
      distance: String((item.row as any)?.distance ?? '0'),
      fspl: '0',
      total_loss: '0',
    };
    return calculateRowDerived(applyDownlinkLoss(row));
  });
}

function recalculateAllRows() {
  rows.value = rows.value.map((r) => calculateRowDerived(applyDownlinkLoss(r)));
}

function onCellValueChanged(event: CellValueChangedEvent) {
  const field = String(event.colDef.field ?? '');
  if (
    field !== 'fixed_pad_loss' &&
    field !== 'antenna_gain' &&
    field !== 'ground_antenna_gain' &&
    field !== 'distance'
  ) return;

  const updated = calculateRowDerived(event.data as CalibrationGridRow);
  Object.assign(event.data as CalibrationGridRow, updated);
  event.api?.refreshCells({ rowNodes: [event.node], columns: ['fspl', 'total_loss'], force: true });
}

async function loadCalIds() {
  calIdsLoading.value = true;
  try {
    const res = await calibrationDataApi.getMeasureOptions();
    if (res.error.value) {
      calIdOptions.value = [];
      return;
    }
    const payload = (res.data.value as MeasureOptionsResponse) ?? { cal_ids: [], default_cal_id: null, test_phases: [], test_plan_types: [], default_test_plan_type: null };
    calIdOptions.value = payload.cal_ids ?? [];
    if (selectedCalId.value === null) {
      selectedCalId.value = payload.default_cal_id ?? (calIdOptions.value.length > 0 ? calIdOptions.value[0] : null);
    }
  } finally {
    calIdsLoading.value = false;
  }
}

async function loadDownlinkLossMap(calId: string | null) {
  downlinkLossMap.value = new Map();
  const id = String(calId ?? '').trim();
  if (id === '') return;
  const res = await calibrationDataApi.getDownlinkCalData(id);
  if (res.error.value || !res.data.value) return;
  const payload = res.data.value as DownlinkCalDataRowsResponse;
  const map = new Map<string, number>();
  for (const r of payload.rows ?? []) {
    const key = buildKey(r.code, r.port, r.frequency);
    map.set(key, Number(r.value));
  }
  downlinkLossMap.value = map;
}

async function onCalIdChange() {
  await loadDownlinkLossMap(selectedCalId.value);
  recalculateAllRows();
}

async function load() {
  await Promise.all([loadCalIds(), loadDownlinkLossMap(selectedCalId.value)]);
  const res = await api.getCalibrationRows();
  if (res.error.value) {
    toast.add({ severity: 'error', summary: 'Load Failed', detail: 'Unable to load calibration rows.', life: 3500 });
    return;
  }

  const payload = (res.data.value as CalibrationRowsResponse) ?? { unit: 'dB', rows: [] };
  rows.value = mapRows(payload);
  // After cal-ids load may have set selectedCalId, ensure downlink loss map is loaded for it.
  if (selectedCalId.value && downlinkLossMap.value.size === 0) {
    await loadDownlinkLossMap(selectedCalId.value);
    recalculateAllRows();
  }
}

async function save() {
  saving.value = true;
  try {
    recalculateAllRows();

    const payloadRows = rows.value
      .map((r) => ({
        transmitter_code: r.transmitter_code,
        row: {
          code: r.code,
          port: r.port,
          frequency_label: r.frequency_label,
          frequency: r.frequency,
          system_loss: r.system_loss,
          fixed_pad_loss: r.fixed_pad_loss,
          antenna_gain: r.antenna_gain,
          ground_antenna_gain: r.ground_antenna_gain,
          distance: r.distance,
          total_loss: r.total_loss,
        },
      }))
      .filter((r) => r.transmitter_code !== '');

    const res = await api.saveCalibrationRows({ rows: payloadRows });
    if (res.error.value) {
      toast.add({ severity: 'error', summary: 'Save Failed', detail: 'Unable to save calibration rows.', life: 3500 });
      return;
    }

    const summary = res.data.value as any;
    toast.add({
      severity: 'success',
      summary: 'Saved',
      detail: `Calibration updated (${summary?.updated_rows ?? 0} rows).`,
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
  // After UI state restore, refresh data dependent on selectedCalId.
  await onCalIdChange();
});
</script>

<style scoped>
.cal-panel {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 4rem);
  padding: 1.5rem;
  color: #e2e8f0;
  box-sizing: border-box;
}

.cal-header h2 {
  font-size: 1.1rem;
  font-weight: 600;
  color: #22d3ee;
  margin: 0 0 1rem;
}

.cal-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.cal-section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.75rem;
  flex-shrink: 0;
}

.cal-section-header h3 {
  font-size: 0.95rem;
  font-weight: 500;
  color: #cbd5e1;
  margin: 0;
}

.actions { display: flex; gap: 0.5rem; }

.cal-grid { flex: 1; min-height: 0; }

/* ── Light theme overrides ──────────────────────────────────────────────── */
html:not(.dark) .cal-panel {
  color: var(--p-text-color);
}
html:not(.dark) .cal-header h2 {
  color: var(--p-primary-color);
}
html:not(.dark) .cal-section-header h3 {
  color: var(--p-text-color);
}
</style>
