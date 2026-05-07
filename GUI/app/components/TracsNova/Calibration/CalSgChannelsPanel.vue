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
          :rowSelection="rowSelection"
          :rowClassRules="rowClassRules"
          :defaultColDef="defaultColDef"
          :suppressContextMenu="false"
          :suppressMovableColumns="true"
          @grid-ready="onGridReady"
          @selection-changed="onSelectionChanged"
        />

        <div class="panel-footer">
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

        <div v-if="samples.length === 0" class="placeholder-area">
          <i class="pi pi-chart-line" />
          <p>Calibration data will appear while run is active.</p>
        </div>

        <!-- Cal SG Grid -->
        <ag-grid-vue
          v-if="props.calType !== 'inject_cal' && samples.length > 0"
          class="sample-grid"
          style="width: 100%; height: 100%;"
          :theme="isDark
            ? themeQuartz.withPart(colorSchemeDarkBlue)
            : themeQuartz.withPart(colorSchemeLightCold)"
          :columnDefs="calSgSampleColumnDefs"
          :rowData="samples"
          :defaultColDef="sampleDefaultColDef"
          :suppressContextMenu="false"
          :suppressMovableColumns="true"
          :domLayout="'normal'"
          @grid-ready="onSampleGridReady"
        />

        <!-- Inject Cal Grid -->
        <ag-grid-vue
          v-if="props.calType === 'inject_cal' && samples.length > 0"
          class="sample-grid"
          style="width: 100%; height: 100%;"
          :theme="isDark
            ? themeQuartz.withPart(colorSchemeDarkBlue)
            : themeQuartz.withPart(colorSchemeLightCold)"
          :columnDefs="injectCalSampleColumnDefs"
          :rowData="samples"
          :defaultColDef="sampleDefaultColDef"
          :suppressContextMenu="false"
          :suppressMovableColumns="true"
          :domLayout="'normal'"
          @grid-ready="onSampleGridReady"
        />
      </section>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue';
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
  type SpuriousBandConfigRow,
  type Transmitter,
} from '@/composables/tracsNova/useTransmitterApi';
import {
  useCalibrationDataApi,
  type CalSgCompletedFrequenciesResponse,
  type CalSgDataRowsResponse,
} from '@/composables/tracsNova/useCalibrationDataApi';
import {
  useCalibrationRunApi,
  type CalibrationChannel,
  type CalibrationRunSnapshot,
  type CalibrationSample,
} from '@/composables/tracsNova/useCalibrationRunApi';
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

interface CalSgChannelRow {
  code: string;
  frequency: string;
  frequency_label: string;
}

const isDark = useDark();
const toast = useToast();
const api = useTransmitterApi();
const runApi = useCalibrationRunApi();
const calibrationDataApi = useCalibrationDataApi();
const ui = useUiStatePersistence(`ui_state:tracsNova:calibration:${props.calType}:channels`);
ui.registerGrid('channels');
ui.registerGrid('samples');

const rows = ref<CalSgChannelRow[]>([]);
const statusLines = ref<string[]>(['Ready. Select channels and click Start Cal.']);
const samples = ref<CalibrationSample[]>([]);
const isRunning = ref(false);
const progress = ref(0);
const showPromptDialog = ref(false);
const promptMessage = ref('Connect Cal Power Sensor to selected cable and confirm.');
const selectedChannelCount = ref(0);
const runId = ref<string | null>(null);
const completedFrequencySet = ref<Set<string>>(new Set());
const spuriousBandConfigs = ref<SpuriousBandConfigRow[]>([]);

const gridApi = shallowRef<GridApi | null>(null);
const calSgSampleGridApi = shallowRef<GridApi | null>(null);
const injectCalSampleGridApi = shallowRef<GridApi | null>(null);
let eventSource: EventSource | null = null;

const rowSelection = {
  mode: 'multiRow' as const,
  checkboxes: true,
  headerCheckbox: true,
  checkboxLocation: 'selectionColumn' as const,
  enableClickSelection: false,
  hideDisabledCheckboxes: false,
  groupSelects: 'self' as const,
};

const defaultColDef: ColDef = {
  resizable: true,
  sortable: true,
  filter: true,
  minWidth: 120,
  enableRowGroup: false,
};

const rowClassRules = {
  'cal-completed-row': (params: any) => {
    if (params?.node?.group || !params?.data) return false;
    const frequency = normalizeFrequency(params.data.frequency);
    return frequency !== '' && completedFrequencySet.value.has(frequency);
  },
};

const hasSelectedChannels = computed(() => {
  return selectedChannelCount.value > 0;
});

const columnDefs: ColDef[] = [
  {
    field: 'code',
    headerName: 'Code',
    editable: false,
    minWidth: 180,
    flex: 1.2,
  },
  {
    field: 'frequency',
    headerName: 'Frequency (MHz)',
    editable: false,
    minWidth: 140,
    flex: 1,
  },
];

// Cal SG column definitions
const calSgSampleColumnDefs: ColDef[] = [
  {
    field: 'code',
    headerName: 'Channel',
    editable: false,
    minWidth: 120,
    flex: 1,
    valueGetter: (params) => formatSampleChannel(params.data as CalibrationSample),
  },
  {
    field: 'frequency',
    headerName: 'Frequency (MHz)',
    editable: false,
    minWidth: 100,
    flex: 0.8,
  },
  {
    field: 'value',
    headerName: 'Value (dB)',
    editable: false,
    minWidth: 110,
    flex: 0.8,
    valueFormatter: (params) => {
      const value = params.value as number | undefined;
      return value !== undefined ? Math.abs(Number(value)).toFixed(1) : '—';
    },
  },
];

// Inject Cal column definitions
const injectCalSampleColumnDefs: ColDef[] = [
  {
    field: 'code',
    headerName: 'Channel',
    editable: false,
    minWidth: 120,
    flex: 1,
    valueGetter: (params) => formatSampleChannel(params.data as CalibrationSample),
  },
  {
    field: 'frequency',
    headerName: 'Frequency (MHz)',
    editable: false,
    minWidth: 100,
    flex: 0.8,
  },
  {
    field: 'sa_loss',
    headerName: 'SA Loss (dB)',
    editable: false,
    minWidth: 110,
    flex: 0.8,
    valueFormatter: (params) => {
      const value = params.value as number | undefined;
      return value !== undefined ? Math.abs(Number(value)).toFixed(1) : '—';
    },
  },
  {
    field: 'dl_pm_loss',
    headerName: 'DL_PM Loss (dB)',
    editable: false,
    minWidth: 130,
    flex: 0.8,
    valueFormatter: (params) => {
      const value = params.value as number | undefined;
      return value !== undefined ? Math.abs(Number(value)).toFixed(1) : '—';
    },
  },
];

// Sample grid default column definition
const sampleDefaultColDef: ColDef = {
  resizable: true,
  sortable: true,
  filter: true,
  minWidth: 100,
  enableRowGroup: false,
};

function onGridReady(event: GridReadyEvent) {
  gridApi.value = event.api;
  onSelectionChanged();
  void selectUncalibratedRowsByDefault();
  ui.onGridReady('channels', event);
}

function onSampleGridReady(event: GridReadyEvent) {
  if (props.calType === 'inject_cal') {
    injectCalSampleGridApi.value = event.api;
  } else {
    calSgSampleGridApi.value = event.api;
  }
  ui.onGridReady('samples', event);
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

function isCompletedCalibrationRow(row: CalSgChannelRow): boolean {
  const frequency = normalizeFrequency(row.frequency);
  return frequency !== '' && completedFrequencySet.value.has(frequency);
}

async function selectUncalibratedRowsByDefault() {
  await nextTick();
  const apiRef = gridApi.value;
  if (!apiRef) return;

  apiRef.forEachNode((node) => {
    if (node.group || !node.data) return;
    const row = node.data as CalSgChannelRow;
    node.setSelected(!isCompletedCalibrationRow(row));
  });

  onSelectionChanged();
}

function collectSelectedChannels(): CalibrationChannel[] {
  const selected: CalibrationChannel[] = [];
  const apiRef = gridApi.value;
  if (!apiRef) return selected;

  for (const node of apiRef.getSelectedNodes()) {
    if (node.group || !node.data) continue;
    selected.push({
      code: String(node.data.code ?? '').trim(),
      port: '',
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
  emit('update:isRunning', false);
  progress.value = 0;
  showPromptDialog.value = false;
  samples.value = [];
  if (message) {
    statusLines.value = [message];
  }
}

function pushStatus(message: string) {
  const t = new Date();
  const stamp = `${t.toLocaleTimeString()}`;
  statusLines.value = [`[${stamp}] ${message}`, ...statusLines.value].slice(0, 200);
}

function attachRunSnapshot(snapshot: CalibrationRunSnapshot) {
  if (!isSnapshotForCurrentSelection(snapshot)) return;
  if (isStaleAbortingSnapshot(snapshot)) {
    resetRunUi('Ready. Select channels and click Start Cal.');
    return;
  }
  runId.value = snapshot.run_id;
  isRunning.value = ['created', 'awaiting_operator', 'running', 'aborting'].includes(snapshot.state);
  emit('update:isRunning', isRunning.value);
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
  eventSource.onmessage = () => {};

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
      emit('update:isRunning', isRunning.value);
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
      const sample = payload.sample as CalibrationSample;
      samples.value = [sample, ...samples.value].slice(0, 120);
      const freq = normalizeFrequency(sample.frequency);
      if (freq !== '') {
        completedFrequencySet.value = new Set(completedFrequencySet.value).add(freq);
        gridApi.value?.redrawRows();
        if (props.calType === 'inject_cal') {
          injectCalSampleGridApi.value?.redrawRows();
        } else {
          calSgSampleGridApi.value?.redrawRows();
        }
      }
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

  eventSource.onerror = () => {};
}

function formatDbm(value: number): string {
  if (!Number.isFinite(value)) return String(value);
  return value.toFixed(1);
}

function getCodeForFrequency(frequency: unknown): string {
  const key = normalizeFrequency(frequency);
  if (key === '') return '';
  const match = rows.value.find((row) => normalizeFrequency(row.frequency) === key);
  return String(match?.code ?? '').trim();
}

function normalizeFrequency(value: unknown): string {
  const text = String(value ?? '').trim();
  if (text === '') return '';
  const parsed = Number(text);
  if (Number.isNaN(parsed)) return text;
  return parsed.toFixed(6);
}

function formatSampleChannel(sample: CalibrationSample): string {
  const code = String(sample.code ?? '').trim();
  const port = String(sample.port ?? '').trim();
  if (code !== '' && port !== '') {
    return `${code}/${port}`;
  }
  if (code !== '') {
    return code;
  }
  if (port !== '') {
    return port;
  }
  const mappedCode = getCodeForFrequency(sample.frequency);
  if (mappedCode !== '') {
    return mappedCode;
  }
  return String(sample.frequency_label ?? '').trim() || 'N/A';
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

async function loadCompletedRowsForCalId() {
  const calId = String(props.calId ?? '').trim();
  if (calId === '') {
    completedFrequencySet.value = new Set();
    gridApi.value?.redrawRows();
    await selectUncalibratedRowsByDefault();
    return;
  }

  const res = await calibrationDataApi.getCalSgCompletedFrequencies(calId, props.calType);
  if (res.error.value || !res.data.value) {
    completedFrequencySet.value = new Set();
    gridApi.value?.redrawRows();
    await selectUncalibratedRowsByDefault();
    return;
  }

  const payload = res.data.value as CalSgCompletedFrequenciesResponse;
  completedFrequencySet.value = new Set((payload.frequencies ?? []).map((f) => Number(f).toFixed(6)));
  gridApi.value?.redrawRows();
  await selectUncalibratedRowsByDefault();
}

async function loadCalibrationDataForCalId() {
  const calId = String(props.calId ?? '').trim();
  if (calId === '') {
    samples.value = [];
    return;
  }

  const res = await calibrationDataApi.getCalSgData(calId, props.calType);
  if (res.error.value || !res.data.value) {
    samples.value = [];
    return;
  }

  const payload = res.data.value as CalSgDataRowsResponse;
  samples.value = (payload.rows ?? []).map((row) => ({
    code: getCodeForFrequency(row.frequency),
    port: '',
    frequency_label: '',
    frequency: String(row.frequency ?? ''),
    value: row.value !== undefined ? Number(row.value) : undefined,
    sa_loss: row.sa_loss !== undefined ? Number(row.sa_loss) : undefined,
    dl_pm_loss: row.dl_pm_loss !== undefined ? Number(row.dl_pm_loss) : undefined,
    timestamp: String(row.datetime ?? ''),
  }));
}

function mapRowsFromTransmitters(items: Transmitter[]): CalSgChannelRow[] {
  const aggregated = new Map<string, { frequency: string; frequency_label: string; codes: Set<string>; hasSpuriousFrequency: boolean }>();
  const includeSpurious = !!props.includeSpuriousBands;

  const upsertRow = (frequency: string, frequencyLabel: string, code: string, isSpuriousFrequency = false) => {
    const freqNormalized = normalizeFrequency(frequency);
    if (freqNormalized === '') return;
    const key = freqNormalized;
    const existing = aggregated.get(key);
    if (existing) {
      if (code !== '') existing.codes.add(code);
      existing.hasSpuriousFrequency = existing.hasSpuriousFrequency || isSpuriousFrequency;
      if (existing.hasSpuriousFrequency) {
        existing.frequency_label = '';
      } else if (existing.frequency_label === '' && frequencyLabel !== '') {
        existing.frequency_label = frequencyLabel;
      }
      return;
    }
    const codes = new Set<string>();
    if (code !== '') codes.add(code);
    aggregated.set(key, {
      frequency: String(Number(freqNormalized)),
      frequency_label: isSpuriousFrequency ? '' : frequencyLabel,
      codes,
      hasSpuriousFrequency: isSpuriousFrequency,
    });
  };

  for (const tx of items ?? []) {
    const specs = tx?.modulation_details?.power_specs ?? [];
    const txCode = String(tx?.code ?? '').trim();
    for (const base of specs) {
      const baseFrequencyText = String(base?.frequency ?? '').trim();
      const baseFrequency = toNumber(baseFrequencyText);
      const baseLabel = String(base?.frequency_label ?? '').trim();
      if (baseFrequency === null || baseLabel === '') continue;

      upsertRow(String(baseFrequency), baseLabel, txCode, false);

      if (!includeSpurious) continue;

      const spuriousSpec = findSpuriousSpecForBase(
        { frequency: String(base?.frequency ?? ''), frequency_label: baseLabel },
        tx,
      );
      if (!spuriousSpec) continue;

      const extraFrequencies = new Set<number>();
      for (const field of ['fbt', 'fbt_hot', 'fbt_cold'] as const) {
        for (const offset of extractOffsets(spuriousSpec[field])) {
          extraFrequencies.add(Number((baseFrequency + offset).toFixed(6)));
        }
      }

      const profileName = String(spuriousSpec.profile_name ?? '').trim();
      const bandFrequencies = new Set<number>();
      for (const f of buildBandFrequencyList(profileName)) {
        const normalizedBandFreq = Number(f.toFixed(6));
        extraFrequencies.add(normalizedBandFreq);
        bandFrequencies.add(normalizedBandFreq);
      }

      for (const extra of extraFrequencies) {
        const extraFrequencyText = Number(extra).toFixed(6);
        const isBandFrequency = bandFrequencies.has(Number(extra.toFixed(6)));
        const extraLabel = isBandFrequency ? '' : baseLabel;
        upsertRow(extraFrequencyText, extraLabel, txCode, true);
      }
    }
  }

  const out: CalSgChannelRow[] = Array.from(aggregated.values()).map((row) => {
    const codeList = Array.from(row.codes).sort();
    const displayCodes = row.hasSpuriousFrequency
      ? codeList.map((code) => `${code}_spurious`)
      : codeList;

    return {
      code: displayCodes.join(', '),
      frequency: row.frequency,
      frequency_label: '',
    };
  });

  out.sort((a, b) => {
    const af = toNumber(a.frequency) ?? 0;
    const bf = toNumber(b.frequency) ?? 0;
    if (af !== bf) return af - bf;
    return a.frequency_label.localeCompare(b.frequency_label);
  });

  return out;
}

function mapRowsFromReceiverFrequencies(items: Transmitter[]): CalSgChannelRow[] {
  const aggregated = new Map<string, { frequency: string; frequency_label: string; codes: Set<string> }>();

  for (const rx of items ?? []) {
    const systemType = String((rx as any)?.system_type ?? '').trim().toLowerCase();
    if (systemType !== 'receiver') continue;

    const rxCode = String((rx as any)?.code ?? '').trim();
    const frequencies = ((rx as any)?.modulation_details as any)?.frequencies ?? [];
    if (!Array.isArray(frequencies)) continue;

    for (const row of frequencies) {
      if (!Array.isArray(row) || row.length < 2) continue;
      const label = String(row[0] ?? '').trim();
      const frequencyNumber = toNumber(row[1]);
      if (frequencyNumber === null) continue;

      const normalized = normalizeFrequency(frequencyNumber);
      if (normalized === '') continue;
      const existing = aggregated.get(normalized);
      const receiverCode = rxCode || 'receiver';
      if (existing) {
        existing.codes.add(receiverCode);
        if (existing.frequency_label === '' && label !== '') {
          existing.frequency_label = label;
        }
        continue;
      }
      const codes = new Set<string>();
      codes.add(receiverCode);
      aggregated.set(normalized, {
        frequency: String(Number(normalized)),
        frequency_label: label,
        codes,
      });
    }
  }

  const out: CalSgChannelRow[] = Array.from(aggregated.values()).map((row) => ({
    code: Array.from(row.codes).sort().join(', '),
    frequency: row.frequency,
    frequency_label: row.frequency_label,
  }));

  out.sort((a, b) => {
    const af = toNumber(a.frequency) ?? 0;
    const bf = toNumber(b.frequency) ?? 0;
    if (af !== bf) return af - bf;
    return a.frequency_label.localeCompare(b.frequency_label);
  });

  return out;
}

function mergeChannelRows(items: CalSgChannelRow[]): CalSgChannelRow[] {
  const byFrequency = new Map<string, { codes: Set<string>; frequency_label: string; frequency: string }>();

  for (const row of items) {
    const normalized = normalizeFrequency(row.frequency);
    if (normalized === '') continue;
    const key = normalized;
    const existing = byFrequency.get(key);
    const rowCodes = String(row.code ?? '')
      .split(',')
      .map((code) => code.trim())
      .filter((code) => code !== '');

    if (existing) {
      rowCodes.forEach((code) => existing.codes.add(code));
      if (existing.frequency_label === '' && String(row.frequency_label ?? '').trim() !== '') {
        existing.frequency_label = String(row.frequency_label ?? '').trim();
      }
      continue;
    }

    byFrequency.set(key, {
      frequency: String(Number(normalized)),
      frequency_label: String(row.frequency_label ?? '').trim(),
      codes: new Set(rowCodes),
    });
  }

  const out: CalSgChannelRow[] = Array.from(byFrequency.values()).map((row) => ({
    code: Array.from(row.codes).sort().join(', '),
    frequency: row.frequency,
    frequency_label: row.frequency_label,
  }));

  out.sort((a, b) => {
    const af = toNumber(a.frequency) ?? 0;
    const bf = toNumber(b.frequency) ?? 0;
    if (af !== bf) return af - bf;
    return a.frequency_label.localeCompare(b.frequency_label);
  });

  return out;
}

async function load() {
  const [txRes, bandRes] = await Promise.all([
    api.getTransmitters(),
    api.getSpuriousBandConfigs(),
  ]);

  if (!bandRes.error.value && bandRes.data.value) {
    const payload = bandRes.data.value as { bands?: SpuriousBandConfigRow[] };
    spuriousBandConfigs.value = payload?.bands ?? [];
  } else {
    spuriousBandConfigs.value = [];
  }

  if (!txRes.error.value && Array.isArray(txRes.data.value)) {
    const txItems = txRes.data.value as Transmitter[];
    const receiverRows = mapRowsFromReceiverFrequencies(txItems);
    const fallbackRows = props.calType === 'inject_cal'
      ? mergeChannelRows([
        ...mapRowsFromTransmitters(txItems),
        ...receiverRows,
      ])
      : mapRowsFromTransmitters(txItems);
    rows.value = fallbackRows;
    await loadCompletedRowsForCalId();
    if (props.calType === 'inject_cal') {
      const sourceLabel = receiverRows.length > 0 ? 'transmitter + receiver' : 'transmitter only (no receiver systems found)';
      pushStatus(`Loaded ${fallbackRows.length} Inject Cal frequency rows (${sourceLabel}).`);
    } else {
      pushStatus(`Loaded ${fallbackRows.length} frequency rows.`);
    }
    return;
  }

  rows.value = [];
  completedFrequencySet.value = new Set();
  gridApi.value?.redrawRows();
  pushStatus('No channel rows found.');
}

async function startCal() {
  const calId = props.calId?.trim();
  if (!calId) {
    toast.add({ severity: 'warn', summary: 'Cal ID Required', detail: 'Please enter/select Cal ID.', life: 3000 });
    return;
  }

  if (!hasSelectedChannels.value) {
    toast.add({ severity: 'warn', summary: 'No Channels Selected', detail: 'Select at least one channel to start calibration.', life: 3000 });
    return;
  }

  const channels = collectSelectedChannels();
  const res = await runApi.startRun({
    cal_id: calId,
    cal_type: props.calType,
    include_spurious_bands: props.includeSpuriousBands ?? null,
    channels,
  });

  if (res.error.value) {
    const msg = (res.error.value as any)?.data?.detail || 'Unable to start calibration run.';
    toast.add({ severity: 'error', summary: 'Start Failed', detail: String(msg), life: 3500 });
    return;
  }

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

onMounted(async () => {
  resetRunUi('Ready. Select channels and click Start Cal.');
  await load();
  await loadCalibrationDataForCalId();
  await ui.load();

  const active = await runApi.getActiveRun();
  if (!active.error.value && active.data.value) {
    const snapshot = active.data.value as CalibrationRunSnapshot;
    if (isSnapshotForCurrentSelection(snapshot)) {
      attachRunSnapshot(snapshot);
      if (snapshot.run_id) setupStream(snapshot.run_id);
    }
  }
});

watch(
  () => [props.calType, props.calId, props.includeSpuriousBands],
  async () => {
    if (eventSource) {
      eventSource.close();
      eventSource = null;
    }

    resetRunUi('Ready. Select channels and click Start Cal.');
    await load();
    await loadCalibrationDataForCalId();

    const active = await runApi.getActiveRun();
    if (!active.error.value && active.data.value) {
      const snapshot = active.data.value as CalibrationRunSnapshot;
      if (isSnapshotForCurrentSelection(snapshot)) {
        attachRunSnapshot(snapshot);
        if (snapshot.run_id) setupStream(snapshot.run_id);
      }
    }
  }
);

async function generateReport() {
  const calId = props.calId?.trim();
  if (!calId) {
    pushStatus('Generate Report: Cal ID is required.');
    toast.add({ severity: 'warn', summary: 'Cal ID Required', detail: 'Please enter/select a Cal ID to generate report.', life: 3000 });
    return;
  }

  pushStatus(`Generating report for Cal ID: ${calId} …`);
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
  background: #0a1425;
  border: 1px solid #1f2f4a;
  border-radius: 10px;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.left-pane,
.right-pane {
  padding: 0.75rem;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.5rem;
}

.panel-header h3 {
  color: #e2e8f0;
  font-size: 1rem;
  margin: 0;
}

.right-header {
  margin-bottom: 0.75rem;
}

.downlink-grid {
  flex: 1;
  min-height: 220px;
}

.panel-footer {
  display: flex;
  gap: 0.5rem;
  margin-top: 0.75rem;
}

.action-btn {
  min-width: 7.5rem;
}

.status-window {
  margin-top: 0.75rem;
  border: 1px solid #1f2f4a;
  border-radius: 8px;
  background: #071120;
  min-height: 130px;
  max-height: 190px;
  display: flex;
  flex-direction: column;
}

.status-title {
  padding: 0.5rem 0.75rem;
  border-bottom: 1px solid #1f2f4a;
  color: #94a3b8;
  font-size: 0.8rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.run-badge {
  color: #22d3ee;
  font-weight: 700;
}

.status-body {
  padding: 0.5rem 0.75rem;
  overflow: auto;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.8rem;
  line-height: 1.35;
  color: #cbd5e1;
}

.status-line {
  margin: 0 0 0.35rem;
}

.placeholder-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  gap: 0.6rem;
  color: #64748b;
}

.placeholder-area .pi {
  font-size: 2rem;
  color: #22d3ee;
}

.sample-grid {
  flex: 1;
  min-height: 0;
}

:deep(.cal-completed-row .ag-cell) {
  background: #10361f !important;
  color: #d1fae5 !important;
}

:deep(.cal-completed-row .ag-group-value) {
  color: #d1fae5 !important;
}

@media (max-width: 1200px) {
  .downlink-layout {
    grid-template-columns: 1fr;
  }
}
</style>
