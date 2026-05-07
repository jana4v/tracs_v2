<template>
  <div class="downlink-panel">
    <Dialog
      v-model:visible="showPromptDialog"
      modal
      :closable="false"
      :draggable="false"
      :style="{ width: '34rem' }"
      header="Operator Action Required"
    >
      <p class="prompt-message">{{ promptMessage }}</p>
      <template #footer>
        <Button label="Abort" severity="danger" outlined @click="onPromptAbort" />
        <Button label="Connected" @click="onPromptConnected" />
      </template>
    </Dialog>

    <div class="downlink-layout">
      <section class="left-pane pane-card">
        <div class="panel-header">
          <h3>Select Channels</h3>
          <Button label="Refresh" size="small" severity="secondary" :disabled="isRunning" @click="load" />
        </div>

        <ag-grid-vue
          class="downlink-grid"
          style="width: 100%; height: 100%;"
          :theme="isDark
            ? themeQuartz.withPart(colorSchemeDarkBlue)
            : themeQuartz.withPart(colorSchemeLightCold)"
          :columnDefs="columnDefs"
          :rowData="rows"
          :rowClassRules="rowClassRules"
          :rowSelection="rowSelection"
          :autoGroupColumnDef="autoGroupColumnDef"
          :defaultColDef="defaultColDef"
          :suppressContextMenu="false"
          :suppressMovableColumns="true"
          :suppressGroupChangesColumnVisibility="'suppressHideOnGroup'"
          :groupDefaultExpanded="0"
          rowGroupPanelShow="always"
          groupDisplayType="singleColumn"
          @grid-ready="onGridReady"
          @selection-changed="onSelectionChanged"
        />

        <div class="panel-footer">
          <div class="cal-sg-level-input">
            <label for="downlink-cal-sg-level">Cal SG Level (dBm)</label>
            <InputNumber
              id="downlink-cal-sg-level"
              v-model="calSgLevel"
              :minFractionDigits="1"
              :maxFractionDigits="1"
              :step="0.1"
              :disabled="isRunning"
              inputClass="cal-sg-level-field"
            />
          </div>
          <Button label="Start Cal" class="action-btn" :disabled="isRunning || !props.calId || !hasSelectedChannels" @click="startCal" />
          <Button label="Abort" class="action-btn" severity="danger" outlined :disabled="!isRunning" @click="abortCal" />
        </div>

        <div class="status-window" aria-label="Calibration status window">
          <div class="status-title">
            Status Window
            <span v-if="isRunning" class="run-badge">RUNNING {{ Math.round(progress) }}%</span>
          </div>
          <div class="status-body">
            <p v-for="(line, idx) in statusLines" :key="idx" class="status-line">{{ line }}</p>
          </div>
        </div>
      </section>

      <section class="right-pane pane-card">
        <div class="panel-header right-header">
          <h3>Calibration Data</h3>
        </div>

        <div v-if="displayRows.length === 0" class="placeholder-area">
          <i class="pi pi-chart-line" />
          <p>{{ isRunning ? 'Calibration progress will appear while the run is active.' : 'Calibration data will appear after a run completes.' }}</p>
        </div>

        <ag-grid-vue
          v-if="displayRows.length > 0"
          class="sample-grid"
          style="width: 100%; height: 100%;"
          :theme="isDark
            ? themeQuartz.withPart(colorSchemeDarkBlue)
            : themeQuartz.withPart(colorSchemeLightCold)"
          :columnDefs="calDataColumnDefs"
          :rowData="displayRows"
          :defaultColDef="sampleDefaultColDef"
          :suppressContextMenu="false"
          :suppressMovableColumns="true"
          :domLayout="'normal'"
        />
      </section>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, nextTick } from 'vue';
import { ModuleRegistry } from 'ag-grid-community';
import { AllEnterpriseModule } from 'ag-grid-enterprise';
import {
  colorSchemeDarkBlue,
  colorSchemeLightCold,
  themeQuartz,
} from 'ag-grid-community';
import type { ColDef, GridApi, GridReadyEvent } from 'ag-grid-community';
import { AgGridVue } from 'ag-grid-vue3';
import { useToast } from 'primevue/usetoast';
import {
  useTransmitterApi,
  type CalibrationRowsResponse,
  type SpuriousBandConfigRow,
  type Transmitter,
} from '@/composables/tracsNova/useTransmitterApi';
import {
  useCalibrationRunApi,
  type CalibrationChannel,
  type CalibrationRunSnapshot,
  type CalibrationSample,
} from '@/composables/tracsNova/useCalibrationRunApi';
import {
  useCalibrationDataApi,
  type DownlinkCalDataRowsResponse,
} from '@/composables/tracsNova/useCalibrationDataApi';
import { toNumberOrNull as toNumber } from '@/composables/tracsNova/utils';
import { useUiStatePersistence } from '@/composables/tracsNova/useUiStatePersistence';

ModuleRegistry.registerModules([AllEnterpriseModule]);

const props = defineProps<{
  calId: string;
  calType: string;
  includeSpuriousBands?: boolean;
  triggerGenerateReport?: number;
}>();

const emit = defineEmits<{
  (e: 'update:isRunning', value: boolean): void;
}>();

interface DownlinkChannelRow {
  code: string;
  port: string;
  frequency: string;
  frequency_label: string;
}

const isDark = useDark();
const toast = useToast();
const api = useTransmitterApi();
const runApi = useCalibrationRunApi();
const calibrationDataApi = useCalibrationDataApi();
const ui = useUiStatePersistence(`ui_state:tracsNova:calibration:downlink:${props.calType}`);
ui.registerGrid('channels');

const rows = ref<DownlinkChannelRow[]>([]);
const spuriousBandConfigs = ref<SpuriousBandConfigRow[]>([]);
const statusLines = ref<string[]>(['Ready. Select channels and click Start Cal.']);
const samples = ref<CalibrationSample[]>([]);
const calDataRows = ref<{ code: string; port: string; frequency: number; frequency_label: string; value: number; datetime: string }[]>([]);
const completedRowKeySet = ref<Set<string>>(new Set());
const isRunning = ref(false);
const progress = ref(0);
const runId = ref<string | null>(null);
const showPromptDialog = ref(false);
const promptMessage = ref('Connect Cal Power Sensor to selected cable and confirm.');
const selectedChannelCount = ref(0);
const calSgLevel = ref<number>(10.0);

const gridApi = shallowRef<GridApi | null>(null);
let eventSource: EventSource | null = null;

const rowSelection = {
  mode: 'multiRow' as const,
  checkboxes: true,
  headerCheckbox: true,
  checkboxLocation: 'autoGroupColumn' as const,
  enableClickSelection: false,
  hideDisabledCheckboxes: false,
  groupSelects: 'descendants' as const,
};

const autoGroupColumnDef: ColDef = {
  headerName: 'Group',
  minWidth: 200,
  cellRendererParams: { suppressCount: false },
};

const defaultColDef: ColDef = {
  resizable: true,
  sortable: true,
  filter: true,
  minWidth: 120,
  enableRowGroup: true,
};

const hasSelectedChannels = computed(() => {
  return selectedChannelCount.value > 0;
});

function isSpurBandLabel(value: unknown): boolean {
  return String(value ?? '').trim().toLowerCase() === 'spur_band';
}

const displayRows = computed(() => {
  const includeSpurious = !!props.includeSpuriousBands;

  if (isRunning.value || (calDataRows.value.length === 0 && samples.value.length > 0)) {
    const liveRows = samples.value.map((sample) => ({
      code: String(sample.code ?? ''),
      port: String(sample.port ?? ''),
      frequency: Number(sample.frequency ?? 0),
      frequency_label: String(sample.frequency_label ?? ''),
      value: sample.value !== undefined ? Number(sample.value) : undefined,
      datetime: String(sample.timestamp ?? ''),
    }));

    return includeSpurious
      ? liveRows
      : liveRows.filter((row) => !isSpurBandLabel(row.frequency_label));
  }

  return includeSpurious
    ? calDataRows.value
    : calDataRows.value.filter((row) => !isSpurBandLabel(row.frequency_label));
});

const rowClassRules = {
  'cal-completed-row': (params: any) => {
    if (params?.node?.group || !params?.data) return false;
    return completedRowKeySet.value.has(buildRowKey(params.data));
  },
};

const sampleDefaultColDef: ColDef = {
  resizable: true,
  sortable: true,
  filter: true,
  minWidth: 100,
};

const calDataColumnDefs: ColDef[] = [
  { field: 'code', headerName: 'Code', minWidth: 120, flex: 1 },
  { field: 'port', headerName: 'Port', minWidth: 100, flex: 1 },
  { field: 'frequency', headerName: 'Frequency (MHz)', minWidth: 150, flex: 1 },
  { field: 'frequency_label', headerName: 'Frequency Label', minWidth: 150, flex: 1 },
  { field: 'value', headerName: 'Value (dB)', minWidth: 120, flex: 1, valueFormatter: (p: any) => p?.value !== undefined && p?.value !== null && p?.value !== '' ? Math.abs(Number(p.value)).toFixed(1) : '' },
  { field: 'datetime', headerName: 'DateTime', minWidth: 200, flex: 1.5 },
];

const columnDefs: ColDef[] = [
  {
    field: 'code',
    headerName: 'Code',
    editable: false,
    minWidth: 130,
    flex: 1,
  },
  {
    field: 'port',
    headerName: 'Port',
    editable: false,
    rowGroup: true,
    minWidth: 120,
    flex: 1,
  },
  {
    field: 'frequency',
    headerName: 'Frequency (MHz)',
    editable: false,
    minWidth: 140,
    flex: 1,
  },
  {
    field: 'frequency_label',
    headerName: 'Frequency Label',
    editable: false,
    minWidth: 180,
    flex: 1.2,
  },
  {
    field: 'select',
    headerName: 'Select',
    editable: false,
    filter: false,
    sortable: false,
    resizable: false,
    minWidth: 100,
    maxWidth: 120,
    valueGetter: () => '',
  },
];

function onGridReady(event: GridReadyEvent) {
  gridApi.value = event.api;
  onSelectionChanged();
  void selectUncalibratedRowsByDefault();
  ui.onGridReady('channels', event);
}

function onSelectionChanged() {
  const apiRef = gridApi.value;
  if (!apiRef) {
    selectedChannelCount.value = 0;
    return;
  }

  selectedChannelCount.value = apiRef
    .getSelectedNodes()
    .filter((node) => !node.group && !!node.data)
    .length;
}

function normalizeCode(value: string) {
  const text = String(value ?? '').trim();
  if (text.toLowerCase().endsWith('_spurious')) {
    return text.slice(0, -'_spurious'.length).trim();
  }
  if (text.toLowerCase().endsWith('_spur')) {
    return text.slice(0, -'_spur'.length).trim();
  }
  return text;
}

function normalizeFrequency(value: string | number) {
  const numeric = Number(value);
  return Number.isFinite(numeric) ? numeric.toFixed(6) : '';
}


function extractOffsets(matrix: unknown): number[] {
  const offsets: number[] = [];
  if (!Array.isArray(matrix)) return offsets;
  for (const row of matrix) {
    if (!Array.isArray(row) || row.length === 0) continue;
    const offset = toNumber(row[0]);
    if (offset === null) continue;
    offsets.push(offset);
  }
  return offsets;
}

function expandRange(startFrequency: number, stopFrequency: number, step: number): number[] {
  const low = Math.min(startFrequency, stopFrequency);
  const high = Math.max(startFrequency, stopFrequency);
  const out: number[] = [];
  let current = low;
  while (current <= high + 1e-9) {
    out.push(Number(current.toFixed(6)));
    current += step;
  }
  const highRounded = Number(high.toFixed(6));
  if (!out.includes(highRounded)) out.push(highRounded);
  return out;
}

function findSpuriousSpecForBase(base: { frequency: string; frequency_label: string }, tx: Transmitter) {
  const specs = tx?.modulation_details?.spurious_specs ?? [];
  return specs.find((s) => {
    return String(s?.frequency ?? '').trim() === String(base.frequency ?? '').trim()
      && String(s?.frequency_label ?? '').trim() === String(base.frequency_label ?? '').trim();
  });
}

function buildBandFrequencyList(profileName: string): number[] {
  if (!profileName) return [];
  const out: number[] = [];
  for (const band of spuriousBandConfigs.value) {
    if (!band?.enable) continue;
    if (String(band.profile_name ?? '').trim() !== profileName) continue;
    const start = toNumber(band.start_frequency);
    const stop = toNumber(band.stop_frequency);
    if (start === null || stop === null) continue;
    out.push(...expandRange(start, stop, 100));
  }
  return out;
}

function buildRowKey(row: { code?: string; port?: string; frequency?: string | number; frequency_label?: string }) {
  return [
    normalizeCode(String(row.code ?? '')),
    String(row.port ?? '').trim(),
    normalizeFrequency(row.frequency ?? ''),
    String(row.frequency_label ?? '').trim().toLowerCase(),
  ].join('|');
}

function isCompletedCalibrationRow(row: DownlinkChannelRow): boolean {
  return completedRowKeySet.value.has(buildRowKey(row));
}

async function selectUncalibratedRowsByDefault() {
  await nextTick();
  const apiRef = gridApi.value;
  if (!apiRef) return;

  apiRef.forEachNode((node) => {
    if (node.group || !node.data) return;
    const row = node.data as DownlinkChannelRow;
    node.setSelected(!isCompletedCalibrationRow(row));
  });

  onSelectionChanged();
}

function mapRows(payload: CalibrationRowsResponse): DownlinkChannelRow[] {
  return (payload.rows ?? []).map((item) => ({
    code: String(item.row?.code ?? ''),
    port: String(item.row?.port ?? ''),
    frequency: String(item.row?.frequency ?? ''),
    frequency_label: String(item.row?.frequency_label ?? ''),
  }));
}

function mapRowsFromTransmitters(items: Transmitter[]): DownlinkChannelRow[] {
  const out: DownlinkChannelRow[] = [];
  const seen = new Set<string>();
  const includeSpurious = !!props.includeSpuriousBands;

  const upsertRow = (mapped: DownlinkChannelRow) => {
    const key = `${mapped.code}|${mapped.port}|${mapped.frequency_label}|${mapped.frequency}`;
    if (!seen.has(key)) {
      seen.add(key);
      out.push(mapped);
    }
  };

  for (const tx of items ?? []) {
    const specs = tx?.modulation_details?.power_specs ?? [];
    const txCode = String(tx?.code ?? '').trim();
    for (const row of specs) {
      const baseFrequency = toNumber(row?.frequency);
      const baseLabel = String(row?.frequency_label ?? '').trim();
      const basePort = String(row?.port ?? '').trim();
      const baseCode = String(row?.code ?? txCode ?? '').trim();

      const mapped: DownlinkChannelRow = {
        code: baseCode,
        port: String(row?.port ?? ''),
        frequency: String(row?.frequency ?? ''),
        frequency_label: String(row?.frequency_label ?? ''),
      };

      upsertRow(mapped);

      if (!includeSpurious || baseFrequency === null || baseLabel === '' || baseCode === '') {
        continue;
      }

      const spuriousSpec = findSpuriousSpecForBase(
        { frequency: String(row?.frequency ?? ''), frequency_label: baseLabel },
        tx,
      );
      if (!spuriousSpec) {
        continue;
      }

      const extraFrequencies = new Set<number>();
      for (const field of ['fbt', 'fbt_hot', 'fbt_cold'] as const) {
        for (const offset of extractOffsets(spuriousSpec[field])) {
          extraFrequencies.add(Number((baseFrequency + offset).toFixed(6)));
        }
      }

      const profileName = String((spuriousSpec as any)?.profile_name ?? '').trim();
      for (const f of buildBandFrequencyList(profileName)) {
        extraFrequencies.add(Number(f.toFixed(6)));
      }

      for (const extra of extraFrequencies) {
        upsertRow({
          code: baseCode,
          port: basePort,
          frequency: String(Number(extra.toFixed(6))),
          frequency_label: 'spur_band',
        });
      }
    }
  }

  return out;
}

function collectSelectedChannels(): CalibrationChannel[] {
  const selected: CalibrationChannel[] = [];
  const apiRef = gridApi.value;
  if (!apiRef) return selected;

  for (const node of apiRef.getSelectedNodes()) {
    if (node.group || !node.data) continue;
    selected.push({
      code: String(node.data.code ?? ''),
      port: String(node.data.port ?? ''),
      frequency_label: String(node.data.frequency_label ?? ''),
      frequency: String(node.data.frequency ?? ''),
    });
  }

  return selected;
}

function isSnapshotForCurrentCalType(snapshot: Partial<CalibrationRunSnapshot>) {
  const snapshotType = String(snapshot?.cal_type ?? '').trim().toLowerCase();
  const currentType = String(props.calType ?? '').trim().toLowerCase();
  return !!snapshotType && snapshotType === currentType;
}

function isSnapshotForCurrentSelection(snapshot: Partial<CalibrationRunSnapshot>) {
  if (!isSnapshotForCurrentCalType(snapshot)) return false;
  const currentCalId = String(props.calId ?? '').trim();
  if (!currentCalId) return true;
  return String(snapshot?.cal_id ?? '').trim() === currentCalId;
}

function isStaleAbortingSnapshot(snapshot: Partial<CalibrationRunSnapshot>) {
  if (String(snapshot?.state ?? '') !== 'aborting') return false;
  const updatedAtMs = Date.parse(String(snapshot?.updated_at ?? ''));
  if (Number.isNaN(updatedAtMs)) return false;
  return Date.now() - updatedAtMs > 15000;
}

function resetRunUi(message?: string) {
  runId.value = null;
  isRunning.value = false;
  progress.value = 0;
  showPromptDialog.value = false;
  samples.value = [];
  if (message) {
    statusLines.value = [message];
  }
}

function attachRunSnapshot(snapshot: CalibrationRunSnapshot) {
  if (!isSnapshotForCurrentSelection(snapshot)) return;
  if (isStaleAbortingSnapshot(snapshot)) {
    resetRunUi('Ready. Select channels and click Start Cal.');
    return;
  }
  runId.value = snapshot.run_id;
  isRunning.value = ['created', 'awaiting_operator', 'running', 'aborting'].includes(snapshot.state);
  progress.value = snapshot.progress ?? 0;
  statusLines.value = snapshot.status_lines?.length
    ? snapshot.status_lines
    : ['Ready. Select channels and click Start Cal.'];
  samples.value = snapshot.samples ?? [];
  showPromptDialog.value = snapshot.state === 'awaiting_operator' && !!snapshot.prompt_message;
  promptMessage.value = snapshot.prompt_message || 'Connect Cal Power Sensor to selected cable and confirm.';
}

function setupStream(id: string) {
  if (process.server) return;
  if (eventSource) {
    eventSource.close();
    eventSource = null;
  }

  eventSource = new EventSource(runApi.streamUrl(id));
  eventSource.onmessage = () => {
    // noop (named events are used)
  };

  eventSource.addEventListener('snapshot', (evt: MessageEvent) => {
    const payload = JSON.parse(evt.data) as CalibrationRunSnapshot;
    attachRunSnapshot(payload);
  });

  eventSource.addEventListener('status', (evt: MessageEvent) => {
    const payload = JSON.parse(evt.data);
    if (payload?.status_line) {
      statusLines.value = [payload.status_line, ...statusLines.value.filter((x) => x !== payload.status_line)].slice(0, 200);
    }
    if (typeof payload?.progress === 'number') progress.value = payload.progress;
    if (typeof payload?.state === 'string') {
      isRunning.value = ['created', 'awaiting_operator', 'running', 'aborting'].includes(payload.state);
      showPromptDialog.value = payload.state === 'awaiting_operator' && !!payload?.prompt_message;
      if (['completed', 'failed', 'aborted'].includes(payload.state)) {
        showPromptDialog.value = false;
      }
    }
    if (typeof payload?.prompt_message === 'string' && payload.prompt_message.trim() !== '') {
      promptMessage.value = payload.prompt_message;
    }
  });

  eventSource.addEventListener('prompt', (evt: MessageEvent) => {
    const payload = JSON.parse(evt.data);
    promptMessage.value = payload?.prompt_message || 'Connect Cal Power Sensor and confirm.';
    showPromptDialog.value = true;
    if (payload?.status_line) {
      statusLines.value = [payload.status_line, ...statusLines.value.filter((x) => x !== payload.status_line)].slice(0, 200);
    }
  });

  eventSource.addEventListener('sample', (evt: MessageEvent) => {
    const payload = JSON.parse(evt.data);
    if (payload?.sample) {
      samples.value = [payload.sample as CalibrationSample, ...samples.value].slice(0, 120);
    }
    if (payload?.status_line) {
      statusLines.value = [payload.status_line, ...statusLines.value.filter((x) => x !== payload.status_line)].slice(0, 200);
    }
    if (typeof payload?.progress === 'number') progress.value = payload.progress;
  });

  eventSource.addEventListener('end', () => {
    isRunning.value = false;
    emit('update:isRunning', false);
    showPromptDialog.value = false;
    if (eventSource) {
      eventSource.close();
      eventSource = null;
    }
    void load();
    void loadCalibrationDataForCalId();
  });

  eventSource.onerror = () => {
    // Browser reconnects EventSource automatically. Keep silent to avoid toast spam.
  };
}

async function load() {
  const bandRes = await api.getSpuriousBandConfigs();
  if (!bandRes.error.value && bandRes.data.value) {
    const payload = bandRes.data.value as { bands?: SpuriousBandConfigRow[] };
    spuriousBandConfigs.value = payload?.bands ?? [];
  } else {
    spuriousBandConfigs.value = [];
  }

  const res = await api.getCalibrationRows();
  let mappedRows: DownlinkChannelRow[] = [];
  if (!res.error.value) {
    const payload = (res.data.value as CalibrationRowsResponse) ?? { unit: 'dB', rows: [] };
    mappedRows = mapRows(payload);
  }

  const txRes = await api.getTransmitters();
  if (!txRes.error.value && Array.isArray(txRes.data.value)) {
    const transmitterRows = mapRowsFromTransmitters(txRes.data.value as Transmitter[]);
    const mergedRows = [...mappedRows];
    const seen = new Set(mergedRows.map((row) => `${row.code}|${row.port}|${row.frequency_label}|${row.frequency}`));
    for (const row of transmitterRows) {
      const key = `${row.code}|${row.port}|${row.frequency_label}|${row.frequency}`;
      if (!seen.has(key)) {
        seen.add(key);
        mergedRows.push(row);
      }
    }

    if (mergedRows.length > 0) {
      rows.value = mergedRows;
      pushStatus(`Loaded ${mergedRows.length} channel rows (including transmitter expansion).`);
    } else {
      rows.value = [];
      pushStatus('No channel rows found.');
    }
    await selectUncalibratedRowsByDefault();
    return;
  }

  if (mappedRows.length > 0) {
    rows.value = mappedRows;
    pushStatus(`Loaded ${mappedRows.length} channel rows from calibration data.`);
    await selectUncalibratedRowsByDefault();
    return;
  }

  rows.value = [];
  completedRowKeySet.value = new Set();
  gridApi.value?.redrawRows();
  pushStatus('No channel rows found.');
}

async function startCal() {
  const calId = props.calId?.trim();
  if (!calId) {
    toast.add({ severity: 'warn', summary: 'Cal ID Required', detail: 'Please enter/select Cal ID.', life: 3000 });
    return;
  }

  const channels = collectSelectedChannels();
  if (channels.length === 0) {
    toast.add({ severity: 'warn', summary: 'No Channels Selected', detail: 'Select at least one channel to start calibration.', life: 3000 });
    return;
  }

  const res = await runApi.startRun({
    cal_id: calId,
    cal_type: props.calType,
    include_spurious_bands: props.includeSpuriousBands ?? null,
    cal_sg_level: Number(calSgLevel.value ?? 10.0),
    channels,
  });

  if (res.error.value) {
    const msg = (res.error.value as any)?.data?.detail || 'Unable to start calibration run.';
    toast.add({ severity: 'error', summary: 'Start Failed', detail: String(msg), life: 3500 });
    return;
  }

  calDataRows.value = [];
  samples.value = [];
  const snapshot = res.data.value as CalibrationRunSnapshot;
  attachRunSnapshot(snapshot);
  setupStream(snapshot.run_id);
  toast.add({ severity: 'success', summary: 'Started', detail: `Calibration run ${snapshot.run_id}`, life: 2500 });
}

async function abortCal() {
  if (!runId.value) return;
  const res = await runApi.abortRun(runId.value);
  if (res.error.value) {
    toast.add({ severity: 'error', summary: 'Abort Failed', detail: 'Unable to abort run.', life: 3000 });
    return;
  }
  showPromptDialog.value = false;
}

async function onPromptConnected() {
  if (!runId.value) return;
  showPromptDialog.value = false;
  const res = await runApi.respondPrompt(runId.value, { action: 'connected' });
  if (res.error.value) {
    toast.add({ severity: 'error', summary: 'Action Failed', detail: 'Unable to send operator confirmation.', life: 3000 });
    return;
  }
  attachRunSnapshot(res.data.value as CalibrationRunSnapshot);
}

async function onPromptAbort() {
  if (!runId.value) return;
  showPromptDialog.value = false;
  const res = await runApi.respondPrompt(runId.value, { action: 'abort' });
  if (res.error.value) {
    toast.add({ severity: 'error', summary: 'Action Failed', detail: 'Unable to send operator abort.', life: 3000 });
    return;
  }
  attachRunSnapshot(res.data.value as CalibrationRunSnapshot);
}

function pushStatus(message: string) {
  const t = new Date();
  const stamp = `${t.toLocaleTimeString()}`;
  statusLines.value = [`[${stamp}] ${message}`, ...statusLines.value].slice(0, 200);
}

async function loadCalibrationDataForCalId() {
  const calId = String(props.calId ?? '').trim();
  if (calId === '') {
    calDataRows.value = [];
    completedRowKeySet.value = new Set();
    gridApi.value?.redrawRows();
    return;
  }

  const res = await calibrationDataApi.getDownlinkCalData(calId);
  if (res.error.value || !res.data.value) {
    calDataRows.value = [];
    completedRowKeySet.value = new Set();
    gridApi.value?.redrawRows();
    return;
  }

  const payload = res.data.value as DownlinkCalDataRowsResponse;
  calDataRows.value = (payload.rows ?? []).map((row) => ({
    code: row.code,
    port: row.port,
    frequency: row.frequency,
    frequency_label: row.frequency_label ?? '',
    value: row.value,
    datetime: row.datetime,
  }));
  completedRowKeySet.value = new Set(calDataRows.value.map((row) => buildRowKey(row)));
  gridApi.value?.redrawRows();
  await selectUncalibratedRowsByDefault();
}

async function generateReport() {
  const calId = props.calId?.trim();
  if (!calId) {
    pushStatus('Generate Report: Cal ID is required.');
    toast.add({ severity: 'warn', summary: 'Cal ID Required', detail: 'Please enter/select a Cal ID to generate report.', life: 3000 });
    return;
  }

  pushStatus(`Generating report for Cal ID: ${calId} ...`);
  const res = await calibrationDataApi.generateReport({ cal_id: calId, cal_type: props.calType });

  if (res.error.value) {
    const msg = (res.error.value as any)?.data?.detail || 'Unable to generate report.';
    pushStatus(`Report failed: ${msg}`);
    toast.add({ severity: 'error', summary: 'Generate Failed', detail: String(msg), life: 4200 });
    return;
  }

  const payload = res.data.value as import('@/composables/tracsNova/useCalibrationDataApi').CalibrationReportGenerateResponse;
  const lines: string[] = [];
  if (payload.pdf_generated) lines.push(`PDF: ${payload.pdf_path ?? 'saved'}`);
  if (payload.excel_rows_appended > 0) lines.push(`Excel: ${payload.excel_rows_appended} row(s) appended`);
  lines.forEach((l) => pushStatus(l));
  pushStatus(`Report ready: ${payload.message}`);
  toast.add({ severity: 'success', summary: 'Report Ready', detail: payload.message, life: 3200 });
}

onMounted(async () => {
  ui.bindRefs({ calSgLevel });
  await ui.load();
  await load();
  void loadCalibrationDataForCalId();

  const active = await runApi.getActiveRun();
  if (!active.error.value && active.data.value) {
    const snapshot = active.data.value as CalibrationRunSnapshot;
    if (isSnapshotForCurrentSelection(snapshot)) {
      attachRunSnapshot(snapshot);
      if (snapshot.run_id) setupStream(snapshot.run_id);
    } else {
      resetRunUi('Ready. Select channels and click Start Cal.');
    }
  } else {
    resetRunUi('Ready. Select channels and click Start Cal.');
  }
});

watch(
  () => [props.calType, props.calId],
  async () => {
    if (eventSource) {
      eventSource.close();
      eventSource = null;
    }

    void loadCalibrationDataForCalId();

    const active = await runApi.getActiveRun();
    if (!active.error.value && active.data.value) {
      const snapshot = active.data.value as CalibrationRunSnapshot;
      if (isSnapshotForCurrentSelection(snapshot)) {
        attachRunSnapshot(snapshot);
        if (snapshot.run_id) setupStream(snapshot.run_id);
        return;
      }
    }

    resetRunUi('Ready. Select channels and click Start Cal.');
  }
);

watch(
  () => props.includeSpuriousBands,
  async () => {
    await load();
  }
);

watch(
  () => props.triggerGenerateReport,
  (val, old) => {
    if (val !== undefined && val !== old && val > 0) {
      void generateReport();
    }
  }
);

onBeforeUnmount(() => {
  if (eventSource) {
    eventSource.close();
    eventSource = null;
  }
});
</script>

<style scoped>
.downlink-panel {
  height: 100%;
  min-height: 0;
  padding: 1rem 1.25rem;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.prompt-message {
  color: #94a3b8;
  line-height: 1.5;
  margin: 0;
}

.downlink-layout {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

.pane-card {
  min-height: 0;
  background: #0d1b2e;
  border: 1px solid #1e3050;
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.left-pane,
.right-pane {
  min-width: 0;
  height: 100%;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem 1rem;
  border-bottom: 1px solid #1e3050;
}

.panel-header h3 {
  margin: 0;
  font-size: 0.95rem;
  font-weight: 600;
  color: #22d3ee;
}

.downlink-grid {
  flex: 1;
  min-height: 0;
  height: 100%;
}

:deep(.downlink-grid .ag-root-wrapper),
:deep(.downlink-grid .ag-root),
:deep(.downlink-grid .ag-body-viewport) {
  min-height: 180px;
}

.panel-footer {
  display: flex;
  justify-content: flex-end;
  align-items: flex-end;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  border-top: 1px solid #1e3050;
}

.cal-sg-level-input {
  margin-right: auto;
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
}

.cal-sg-level-input label {
  color: #94a3b8;
  font-size: 0.78rem;
  font-weight: 600;
}

:deep(.cal-sg-level-field) {
  width: 7.5rem;
}

.action-btn {
  min-height: 2.5rem;
  padding: 0 1rem;
  font-size: 0.95rem;
  font-weight: 600;
}

.status-window {
  border-top: 1px solid #1e3050;
  background: #091425;
  min-height: 160px;
  max-height: 260px;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.status-title {
  padding: 0.55rem 1rem;
  color: #22d3ee;
  font-size: 0.85rem;
  font-weight: 600;
  border-bottom: 1px solid #1e3050;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.run-badge {
  color: #0f172a;
  background: #22d3ee;
  border-radius: 999px;
  padding: 0.12rem 0.5rem;
  font-size: 0.72rem;
  font-weight: 700;
}

.status-body {
  padding: 0.55rem 1rem;
  overflow-y: auto;
  color: #94a3b8;
  font-size: 0.82rem;
  line-height: 1.35;
}

.status-line {
  margin: 0 0 0.25rem 0;
}

.right-header {
  justify-content: flex-start;
}

.sample-grid {
  flex: 1;
  min-height: 0;
  height: 100%;
}

:deep(.cal-completed-row .ag-cell) {
  background: #10361f !important;
  color: #d1fae5 !important;
}

:deep(.cal-completed-row .ag-group-value) {
  color: #d1fae5 !important;
}

.placeholder-area {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.65rem;
  color: #64748b;
}

.placeholder-area .pi {
  font-size: 2rem;
  color: #22d3ee;
}

@media (max-width: 1100px) {
  .downlink-layout {
    grid-template-columns: 1fr;
  }
}
</style>
