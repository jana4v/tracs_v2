<template>
  <div class="measure-page">
    <div class="toolbar panel-card">
      <div class="toolbar-item">
        <label>Test Phase</label>
        <Select
          v-model="selectedTestPhase"
          :options="testPhaseOptions"
          class="toolbar-select"
          placeholder="Select Test Phase"
        />
      </div>

      <div class="toolbar-item">
        <label>Sub Test Phase</label>
        <Select
          v-model="selectedSubTestPhase"
          :options="subTestPhaseOptions"
          class="toolbar-select"
          placeholder="Select Sub Test Phase"
        />
      </div>

      <div class="toolbar-item">
        <label>Cal ID</label>
        <Select
          v-model="selectedCalId"
          :options="calIdOptions"
          class="toolbar-select"
          placeholder="Select Cal ID"
        />
      </div>

      <div class="toolbar-item">
        <label>TestPlan Type</label>
        <Select
          v-model="selectedTestPlanType"
          :options="testPlanTypeOptions"
          class="toolbar-select"
          placeholder="Select TestPlan Type"
        />
      </div>

      <div class="toolbar-item mode-item">
        <label>Test Execution Mode</label>
        <Select
          v-model="executionMode"
          :options="executionModeOptions"
          optionLabel="label"
          optionValue="value"
          class="toolbar-select"
          placeholder="Select Mode"
        />
      </div>
    </div>

    <div class="measure-layout">
      <section class="left-pane panel-card">
        <Tabs v-model:value="activeTab" class="measure-tabs">
          <TabList>
            <Tab value="transmitter">Transmitter</Tab>
            <Tab value="receiver">Receiver</Tab>
            <Tab value="transponder">Transponder</Tab>
          </TabList>
          <TabPanels>
            <TabPanel value="transmitter">
              <ag-grid-vue
                class="measure-grid"
                style="width: 100%; height: 100%;"
                :theme="isDark
                  ? themeQuartz.withPart(colorSchemeDarkBlue)
                  : themeQuartz.withPart(colorSchemeLightCold)"
                :columnDefs="transmitterColumnDefs"
                :rowData="transmitterRows"
                :defaultColDef="defaultColDef"
                :rowGroupPanelShow="'always'"
                :enableRangeSelection="true"
                :cellSelection="true"
                :suppressContextMenu="false"
                :suppressMovableColumns="true"
                @cell-value-changed="onTransmitterCellChanged"
                @grid-ready="(e: any) => onGridReady('transmitter', e)"
                @column-moved="() => onGridStateChanged('transmitter')"
                @column-resized="() => onGridStateChanged('transmitter')"
                @column-visible="() => onGridStateChanged('transmitter')"
                @column-pinned="() => onGridStateChanged('transmitter')"
                @sort-changed="() => onGridStateChanged('transmitter')"
                @filter-changed="() => onGridStateChanged('transmitter')"
                @column-row-group-changed="() => onGridStateChanged('transmitter')"
                @row-group-opened="() => onGridStateChanged('transmitter')"
              />
            </TabPanel>
            <TabPanel value="receiver">
              <ag-grid-vue
                class="measure-grid"
                style="width: 100%; height: 100%;"
                :theme="isDark
                  ? themeQuartz.withPart(colorSchemeDarkBlue)
                  : themeQuartz.withPart(colorSchemeLightCold)"
                :columnDefs="receiverColumnDefs"
                :rowData="receiverRows"
                :defaultColDef="defaultColDef"
                :rowGroupPanelShow="'always'"
                :enableRangeSelection="true"
                :cellSelection="true"
                :suppressContextMenu="false"
                :suppressMovableColumns="true"
                @cell-value-changed="onReceiverCellChanged"
                @grid-ready="(e: any) => onGridReady('receiver', e)"
                @column-moved="() => onGridStateChanged('receiver')"
                @column-resized="() => onGridStateChanged('receiver')"
                @column-visible="() => onGridStateChanged('receiver')"
                @column-pinned="() => onGridStateChanged('receiver')"
                @sort-changed="() => onGridStateChanged('receiver')"
                @filter-changed="() => onGridStateChanged('receiver')"
                @column-row-group-changed="() => onGridStateChanged('receiver')"
                @row-group-opened="() => onGridStateChanged('receiver')"
              />
            </TabPanel>
            <TabPanel value="transponder">
              <ag-grid-vue
                class="measure-grid"
                style="width: 100%; height: 100%;"
                :theme="isDark
                  ? themeQuartz.withPart(colorSchemeDarkBlue)
                  : themeQuartz.withPart(colorSchemeLightCold)"
                :columnDefs="transponderColumnDefs"
                :rowData="transponderRows"
                :defaultColDef="defaultColDef"
                :rowGroupPanelShow="'always'"
                :enableRangeSelection="true"
                :cellSelection="true"
                :suppressContextMenu="false"
                :suppressMovableColumns="true"
                @cell-value-changed="onTransponderCellChanged"
                @grid-ready="(e: any) => onGridReady('transponder', e)"
                @column-moved="() => onGridStateChanged('transponder')"
                @column-resized="() => onGridStateChanged('transponder')"
                @column-visible="() => onGridStateChanged('transponder')"
                @column-pinned="() => onGridStateChanged('transponder')"
                @sort-changed="() => onGridStateChanged('transponder')"
                @filter-changed="() => onGridStateChanged('transponder')"
                @column-row-group-changed="() => onGridStateChanged('transponder')"
                @row-group-opened="() => onGridStateChanged('transponder')"
              />
            </TabPanel>
          </TabPanels>
        </Tabs>

        <div class="controls-row">
          <Button label="Start" :disabled="isRunning" @click="startExecution" />
          <Button label="Pause" severity="secondary" :disabled="!isRunning" @click="pauseExecution" />
          <Button label="Abort" severity="danger" outlined :disabled="!isRunning" @click="abortExecution" />
        </div>

        <div class="remarks-row">
          <label>Remarks (optional)</label>
          <Textarea v-model="remarks" rows="3" autoResize placeholder="Enter remarks" />
        </div>

        <div class="status-window" aria-label="Test progress window">
          <div class="status-title">
            Test Progress
            <span v-if="isRunning" class="run-badge">RUNNING {{ Math.round(progress) }}%</span>
          </div>
          <div class="status-body">
            <p v-for="(line, idx) in statusLines" :key="idx" class="status-line">{{ line }}</p>
          </div>
        </div>
      </section>

      <section class="right-pane panel-card">
        <div class="panel-header">
          <h3>Live Test Results</h3>
        </div>

        <ag-grid-vue
          class="live-results-grid"
          style="width: 100%; height: 100%; flex: 1; min-height: 0;"
          :theme="isDark
            ? themeQuartz.withPart(colorSchemeDarkBlue)
            : themeQuartz.withPart(colorSchemeLightCold)"
          :columnDefs="liveColumnDefs"
          :rowData="liveResults"
          :defaultColDef="defaultColDef"
          :suppressContextMenu="false"
          :suppressMovableColumns="true"
        />
      </section>
    </div>

    <Dialog v-model:visible="showMissingDownlinkDialog" modal :closable="false" header="Downlink Cal Missing" :style="{ width: '40rem' }">
      <p class="dialog-text">
        Downlink cal not present for selected channel(s). Continue will skip these channels and proceed.
      </p>
      <div class="dialog-list">
        <div v-for="(item, idx) in missingDownlinkChannels" :key="idx" class="dialog-row">
          {{ item.code || '-' }} / {{ item.port || '-' }} / {{ item.frequency_label || '-' }} / {{ item.frequency }} MHz
        </div>
      </div>
      <template #footer>
        <Button label="Abort" severity="danger" outlined @click="abortMissingDownlink" />
        <Button label="Continue" @click="continueMissingDownlink" />
      </template>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { ModuleRegistry } from 'ag-grid-community';
import { AllEnterpriseModule } from 'ag-grid-enterprise';
import {
  colorSchemeDarkBlue,
  colorSchemeLightCold,
  themeQuartz,
} from 'ag-grid-community';
import type { ColDef } from 'ag-grid-community';
import { AgGridVue } from 'ag-grid-vue3';
import { initMenu } from '@/composables/tracsNova/SideNav';
import {
  useCalibrationDataApi,
  type MeasureRunStartRequest,
  type MeasureMissingChannel,
  type MeasureRunStartResponse,
  type MeasureTableRow,
  type MeasureOptionsResponse,
} from '@/composables/tracsNova/useCalibrationDataApi';
import {
  useTransmitterApi,
  type Transmitter,
  type Receiver,
  type ProjectTransponderRow,
  type ProjectTranspondersResponse,
  type RangingThresholdRow,
} from '@/composables/tracsNova/useTransmitterApi';
import { useUiStatePersistence } from '@/composables/tracsNova/useUiStatePersistence';

ModuleRegistry.registerModules([AllEnterpriseModule]);

interface MeasureRow {
  code: string;
  port: string;
  frequency: string;
  frequency_label: string;
  uplink?: string;
  downlink?: string;
  power_applicable: boolean;
  frequency_applicable: boolean;
  modulation_index_applicable: boolean;
  spurious_applicable: boolean;
  command_threshold_applicable: boolean;
  ranging_threshold_applicable: boolean;
  power_selected: boolean;
  frequency_selected: boolean;
  modulation_index_selected: boolean;
  spurious_selected: boolean;
  command_threshold_selected: boolean;
  ranging_threshold_selected: boolean;
}

interface LiveResultRow {
  time: string;
  code: string;
  port: string;
  frequency: string;
  result: string;
}

const calibrationApi = useCalibrationDataApi();
const transmitterApi = useTransmitterApi();
const isDark = useDark();

const testPhaseOptions = ref<string[]>([]);
const subTestPhaseOptions = Array.from({ length: 10 }, (_, idx) => `Phase${idx + 1}`);
const calIdOptions = ref<string[]>([]);
const testPlanTypeOptions = ref<string[]>([]);
const executionModeOptions = [
  { label: 'TestPlan Mode', value: 'testplan' },
  { label: 'Manual Mode', value: 'manual' },
];

const selectedTestPhase = ref<string>('');
const selectedSubTestPhase = ref<string>('Phase1');
const selectedCalId = ref<string>('');
const selectedTestPlanType = ref<string>('Detailed');
const executionMode = ref<'testplan' | 'manual'>('testplan');
const activeTab = ref('transmitter');
const remarks = ref('');
const isRunning = ref(false);
const progress = ref(0);
const statusLines = ref<string[]>(['Ready. Configure selections and click Start.']);
const liveResults = ref<LiveResultRow[]>([]);

const transmitters = ref<Transmitter[]>([]);
const receivers = ref<Receiver[]>([]);
const projectTransponders = ref<ProjectTransponderRow[]>([]);
const rangingThresholdRows = ref<RangingThresholdRow[]>([]);
const showMissingDownlinkDialog = ref(false);
const missingDownlinkChannels = ref<MeasureMissingChannel[]>([]);
const pendingMeasurePayload = ref<MeasureRunStartRequest | null>(null);

// Saved Test Plan checkbox selections (loaded from /test-plan/selections endpoint).
// Maps row identity -> { param_name: bool } for the currently selected plan.
const savedTransmitterSelections = ref<Map<string, Record<string, boolean>>>(new Map());
const savedReceiverSelections = ref<Map<string, Record<string, boolean>>>(new Map());
const savedTransponderSelections = ref<Map<string, Record<string, boolean>>>(new Map());

// ── Manual-mode localStorage persistence ─────────────────────────────────────
// Manual-mode edits are not persisted to the backend. Instead they are saved
// to localStorage so the user's selections survive a page reload while
// `executionMode === 'manual'`. Switching to TestPlan mode reloads the
// server-side saved selections and ignores the manual cache.
const MANUAL_STORAGE_KEYS = {
  transmitter: 'TracsNova:measure:manual:transmitter',
  receiver: 'TracsNova:measure:manual:receiver',
  transponder: 'TracsNova:measure:manual:transponder',
} as const;

const manualTransmitterSelections = ref<Map<string, Record<string, boolean>>>(new Map());
const manualReceiverSelections = ref<Map<string, Record<string, boolean>>>(new Map());
const manualTransponderSelections = ref<Map<string, Record<string, boolean>>>(new Map());

function loadManualMapFromStorage(key: string): Map<string, Record<string, boolean>> {
  if (typeof window === 'undefined') return new Map();
  try {
    const raw = window.localStorage.getItem(key);
    if (!raw) return new Map();
    const obj = JSON.parse(raw);
    if (!obj || typeof obj !== 'object') return new Map();
    const m = new Map<string, Record<string, boolean>>();
    for (const [k, v] of Object.entries(obj)) {
      if (v && typeof v === 'object') m.set(k, v as Record<string, boolean>);
    }
    return m;
  } catch {
    return new Map();
  }
}

function saveManualMapToStorage(key: string, map: Map<string, Record<string, boolean>>) {
  if (typeof window === 'undefined') return;
  try {
    const obj: Record<string, Record<string, boolean>> = {};
    for (const [k, v] of map.entries()) obj[k] = v;
    window.localStorage.setItem(key, JSON.stringify(obj));
  } catch {
    // ignore quota / serialization errors
  }
}

function loadAllManualSelections() {
  manualTransmitterSelections.value = loadManualMapFromStorage(MANUAL_STORAGE_KEYS.transmitter);
  manualReceiverSelections.value = loadManualMapFromStorage(MANUAL_STORAGE_KEYS.receiver);
  manualTransponderSelections.value = loadManualMapFromStorage(MANUAL_STORAGE_KEYS.transponder);
}

function rememberManualSelection(
  storageKey: string,
  mapRef: typeof manualTransmitterSelections,
  rowKey: string,
  param: string,
  value: boolean,
) {
  const current = mapRef.value.get(rowKey) ?? {};
  const next = { ...current, [param]: value };
  const newMap = new Map(mapRef.value);
  newMap.set(rowKey, next);
  mapRef.value = newMap;
  saveManualMapToStorage(storageKey, newMap);
}

function onTransmitterCellChanged(event: any) {
  if (executionMode.value !== 'manual') return;
  const data = event?.data;
  const field = String(event?.colDef?.field ?? '');
  if (!data || !field.endsWith('_selected')) return;
  const param = field.replace(/_selected$/, '');
  const applicable = `${param}_applicable` in data ? !!data[`${param}_applicable`] : false;
  if (!applicable) return;
  const key = `${String(data.code ?? '')}|${String(data.port ?? '')}|${String(data.frequency_label ?? '')}`;
  rememberManualSelection(MANUAL_STORAGE_KEYS.transmitter, manualTransmitterSelections, key, param, !!event.newValue);
}

function onReceiverCellChanged(event: any) {
  if (executionMode.value !== 'manual') return;
  const data = event?.data;
  const field = String(event?.colDef?.field ?? '');
  if (!data || !field.endsWith('_selected')) return;
  const param = field.replace(/_selected$/, '');
  const applicable = `${param}_applicable` in data ? !!data[`${param}_applicable`] : false;
  if (!applicable) return;
  const key = `${String(data.code ?? '')}|${String(data.port ?? '')}|${String(data.frequency_label ?? '')}`;
  rememberManualSelection(MANUAL_STORAGE_KEYS.receiver, manualReceiverSelections, key, param, !!event.newValue);
}

function onTransponderCellChanged(event: any) {
  if (executionMode.value !== 'manual') return;
  const data = event?.data;
  const field = String(event?.colDef?.field ?? '');
  if (!data || !field.endsWith('_selected')) return;
  const param = field.replace(/_selected$/, '');
  const applicable = `${param}_applicable` in data ? !!data[`${param}_applicable`] : false;
  if (!applicable) return;
  const key = `${String(data.code ?? '')}|${String(data.uplink ?? '')}|${String(data.downlink ?? '')}`;
  rememberManualSelection(MANUAL_STORAGE_KEYS.transponder, manualTransponderSelections, key, param, !!event.newValue);
}

// ── UI state persistence (Configuration table) ───────────────────────────────
// Persists toolbar selections, active tab, remarks, AG-Grid column state and
// expanded row-group keys to the backend so the page restores its layout.
const ui = useUiStatePersistence('ui_state:tracsNova:index');
ui.bindRefs({
  selectedTestPhase,
  selectedSubTestPhase,
  selectedCalId,
  selectedTestPlanType,
  executionMode,
  activeTab,
  remarks,
});
ui.registerGrid('transmitter');
ui.registerGrid('receiver');
ui.registerGrid('transponder');

function onGridReady(kind: 'transmitter' | 'receiver' | 'transponder', event: any) {
  ui.onGridReady(kind, event);
}

function onGridStateChanged(kind: 'transmitter' | 'receiver' | 'transponder') {
  ui.notifyGridChanged(kind);
}

const defaultColDef: ColDef = {
  resizable: true,
  sortable: true,
  filter: true,
  minWidth: 110,
  enableRowGroup: true,
};

const transmitterColumnDefs: ColDef[] = [
  { field: 'code', headerName: 'Code', minWidth: 120, flex: 1 },
  { field: 'port', headerName: 'Port', minWidth: 110, flex: 1 },
  { field: 'frequency', headerName: 'Frequency (MHz)', minWidth: 130, flex: 1 },
  { field: 'frequency_label', headerName: 'Freqcy Label', minWidth: 150, flex: 1 },
  buildCheckboxCol('power_selected', 'Power', 'power_applicable'),
  buildCheckboxCol('frequency_selected', 'Frequency', 'frequency_applicable'),
  buildCheckboxCol('modulation_index_selected', 'Modulation Index', 'modulation_index_applicable'),
  buildCheckboxCol('spurious_selected', 'Spurious', 'spurious_applicable'),
];

const receiverColumnDefs: ColDef[] = [
  { field: 'code', headerName: 'Code', minWidth: 120, flex: 1 },
  { field: 'port', headerName: 'Port', minWidth: 110, flex: 1 },
  { field: 'frequency', headerName: 'Frequency (MHz)', minWidth: 130, flex: 1 },
  { field: 'frequency_label', headerName: 'Freqcy Label', minWidth: 150, flex: 1 },
  buildCheckboxCol('command_threshold_selected', 'Command Threshold', 'command_threshold_applicable'),
];

const transponderColumnDefs: ColDef[] = [
  { field: 'code', headerName: 'TpCode', minWidth: 120, flex: 1 },
  { field: 'uplink', headerName: 'Uplink', minWidth: 150, flex: 1 },
  { field: 'downlink', headerName: 'Downlink', minWidth: 150, flex: 1 },
  buildCheckboxCol('ranging_threshold_selected', 'Ranging Threshold', 'ranging_threshold_applicable'),
];

const liveColumnDefs: ColDef[] = [
  { field: 'code', headerName: 'Code', minWidth: 110, flex: 1 },
  { field: 'port', headerName: 'Port', minWidth: 90, flex: 1 },
  { field: 'frequency', headerName: 'Frequency (MHz)', minWidth: 120, flex: 1 },
  { field: 'result', headerName: 'Result', minWidth: 180, flex: 1.4 },
];

const transmitterRows = computed<MeasureRow[]>(() => buildRows(executionMode.value, transmitters.value, 'transmitter'));
const receiverRows = computed<MeasureRow[]>(() => buildRows(executionMode.value, receivers.value, 'receiver'));
const transponderRows = computed<MeasureRow[]>(() => buildTransponderRows(rangingThresholdRows.value));

function buildCheckboxCol(field: keyof MeasureRow, headerName: string, applicableField: keyof MeasureRow): ColDef {
  return {
    field: field as string,
    headerName,
    minWidth: 130,
    flex: 1,
    enableFillHandle: true,
    cellRenderer: 'agCheckboxCellRenderer',
    cellEditor: 'agCheckboxCellEditor',
    editable: (params) => Boolean(params.data?.[applicableField]),
    valueGetter: (params) => {
      if (!params.data?.[applicableField]) return null;
      return Boolean(params.data?.[field]);
    },
    valueSetter: (params) => {
      if (!params.data?.[applicableField]) return false;
      params.data[field] = Boolean(params.newValue) as never;
      return true;
    },
  };
}

function keyOf(row: { port: string; frequency_label: string; frequency: string }) {
  return `${row.port}|${row.frequency_label}|${row.frequency}`;
}

function collectBaseRows(tx: Transmitter): Array<{ port: string; frequency_label: string; frequency: string }> {
  const details: any = tx?.modulation_details ?? {};
  const out = new Map<string, { port: string; frequency_label: string; frequency: string }>();

  for (const field of ['power_specs', 'frequency_specs', 'modulation_index_specs', 'spurious_specs']) {
    const rows = Array.isArray(details?.[field]) ? details[field] : [];
    for (const row of rows) {
      const item = {
        port: String(row?.port ?? '').trim(),
        frequency_label: String(row?.frequency_label ?? '').trim(),
        frequency: String(row?.frequency ?? '').trim(),
      };
      if (!item.port || !item.frequency_label || !item.frequency) continue;
      out.set(keyOf(item), item);
    }
  }

  if (out.size === 0) {
    const ports = Array.isArray(details?.ports) ? details.ports : [];
    const freqs = Array.isArray(details?.frequencies) ? details.frequencies : [];
    for (const p of ports) {
      const port = String(Array.isArray(p) ? p[0] : '').trim();
      if (!port) continue;
      for (const f of freqs) {
        const label = String(Array.isArray(f) ? f[0] : '').trim();
        const freq = String(Array.isArray(f) ? f[1] : '').trim();
        if (!label || !freq) continue;
        out.set(`${port}|${label}|${freq}`, { port, frequency_label: label, frequency: freq });
      }
    }
  }

  return Array.from(out.values());
}

function hasSpecRow(rows: any[], base: { port: string; frequency_label: string; frequency: string }): boolean {
  return rows.some((r) =>
    String(r?.port ?? '').trim() === base.port
    && String(r?.frequency_label ?? '').trim() === base.frequency_label
    && String(r?.frequency ?? '').trim() === base.frequency,
  );
}

function buildRows(mode: 'testplan' | 'manual', systems: Transmitter[], kind: 'transmitter' | 'receiver'): MeasureRow[] {
  const output: MeasureRow[] = [];
  const savedMap = kind === 'transmitter' ? savedTransmitterSelections.value : savedReceiverSelections.value;
  const manualMap = kind === 'transmitter' ? manualTransmitterSelections.value : manualReceiverSelections.value;
  for (const tx of systems) {
    const details: any = tx?.modulation_details ?? {};
    const baseRows = collectBaseRows(tx);
    const modulationType = String(tx?.modulation_type ?? '').trim().toLowerCase();
    const isPskPm = modulationType === 'psk_pm';
    const isPskFm = modulationType === 'psk_fm';

    const powerSpecs = Array.isArray(details?.power_specs) ? details.power_specs : [];
    const freqSpecs = Array.isArray(details?.frequency_specs) ? details.frequency_specs : [];
    const modSpecs = Array.isArray(details?.modulation_index_specs) ? details.modulation_index_specs : [];
    const spurSpecs = Array.isArray(details?.spurious_specs) ? details.spurious_specs : [];

    for (const base of baseRows) {
      const fromPlan = {
        power: hasSpecRow(powerSpecs, base),
        frequency: hasSpecRow(freqSpecs, base),
        modulation: hasSpecRow(modSpecs, base),
        spurious: hasSpecRow(spurSpecs, base),
      };

      const txApplicable = kind === 'transmitter' && isPskPm;
      const rxApplicable = kind === 'receiver' && isPskFm;

      const code = String(tx?.code ?? '');
      const savedKey = `${code}|${base.port}|${base.frequency_label}`;
      const saved = mode === 'testplan' ? savedMap.get(savedKey) : undefined;
      const manual = mode === 'manual' ? manualMap.get(savedKey) : undefined;

      const useSaved = (param: string, fallback: boolean): boolean => {
        if (mode === 'testplan') {
          if (saved && Object.prototype.hasOwnProperty.call(saved, param)) return !!saved[param];
          return fallback;
        }
        // manual mode: localStorage override, else unchecked
        if (manual && Object.prototype.hasOwnProperty.call(manual, param)) return !!manual[param];
        return false;
      };

      output.push({
        code,
        port: base.port,
        frequency: base.frequency,
        frequency_label: base.frequency_label,
        power_applicable: txApplicable,
        frequency_applicable: txApplicable,
        modulation_index_applicable: txApplicable,
        spurious_applicable: txApplicable,
        command_threshold_applicable: rxApplicable,
        ranging_threshold_applicable: false,
        power_selected: txApplicable ? useSaved('power', fromPlan.power) : false,
        frequency_selected: txApplicable ? useSaved('frequency', fromPlan.frequency) : false,
        modulation_index_selected: txApplicable ? useSaved('modulation_index', fromPlan.modulation) : false,
        spurious_selected: txApplicable ? useSaved('spurious', fromPlan.spurious) : false,
        command_threshold_selected: rxApplicable ? useSaved('command_threshold', false) : false,
        ranging_threshold_selected: false,
      });
    }
  }

  return output;
}

function buildTransponderRows(rows: RangingThresholdRow[]): MeasureRow[] {
  const seen = new Set<string>();
  const output: MeasureRow[] = [];
  // Index project transponders by code so we can resolve port/frequency for the API payload.
  const tpByCode = new Map<string, ProjectTransponderRow>();
  for (const t of projectTransponders.value) {
    const c = String(t?.code ?? '').trim();
    if (c) tpByCode.set(c, t);
  }
  const mode = executionMode.value;
  const savedMap = savedTransponderSelections.value;
  const manualMap = manualTransponderSelections.value;
  for (const r of rows) {
    const code = String(r?.transponder_code ?? '').trim();
    const uplink = String(r?.uplink ?? '').trim();
    const downlink = String(r?.downlink ?? '').trim();
    if (!code) continue;
    const key = `${code}|${uplink}|${downlink}`;
    if (seen.has(key)) continue;
    seen.add(key);
    const tp = tpByCode.get(code);
    const saved = mode === 'testplan' ? savedMap.get(key) : undefined;
    const manual = mode === 'manual' ? manualMap.get(key) : undefined;
    let rangingSelected = false;
    if (mode === 'testplan' && saved && Object.prototype.hasOwnProperty.call(saved, 'ranging_threshold')) {
      rangingSelected = !!saved.ranging_threshold;
    } else if (mode === 'manual' && manual && Object.prototype.hasOwnProperty.call(manual, 'ranging_threshold')) {
      rangingSelected = !!manual.ranging_threshold;
    }
    output.push({
      code,
      port: String(tp?.rx_port ?? '').trim(),
      frequency: String(tp?.rx_freq ?? '').trim(),
      frequency_label: 'ranging',
      uplink,
      downlink,
      power_applicable: false,
      frequency_applicable: false,
      modulation_index_applicable: false,
      spurious_applicable: false,
      command_threshold_applicable: false,
      ranging_threshold_applicable: true,
      power_selected: false,
      frequency_selected: false,
      modulation_index_selected: false,
      spurious_selected: false,
      command_threshold_selected: false,
      ranging_threshold_selected: rangingSelected,
    });
  }
  return output;
}

function pushStatus(message: string) {
  const stamp = new Date().toLocaleTimeString();
  statusLines.value = [`[${stamp}] ${message}`, ...statusLines.value].slice(0, 200);
}

function pushLive(result: string) {
  const row: LiveResultRow = {
    time: new Date().toLocaleTimeString(),
    code: transmitterRows.value[0]?.code ?? '-',
    port: transmitterRows.value[0]?.port ?? '-',
    frequency: transmitterRows.value[0]?.frequency ?? '-',
    result,
  };
  liveResults.value = [row, ...liveResults.value].slice(0, 200);
}

function toApiRow(row: MeasureRow): MeasureTableRow {
  return {
    code: String(row.code ?? '').trim(),
    port: String(row.port ?? '').trim(),
    frequency_label: String(row.frequency_label ?? '').trim(),
    frequency: String(row.frequency ?? '').trim(),
    power_selected: Boolean(row.power_selected),
    frequency_selected: Boolean(row.frequency_selected),
    modulation_index_selected: Boolean(row.modulation_index_selected),
    spurious_selected: Boolean(row.spurious_selected),
    command_threshold_selected: Boolean(row.command_threshold_selected),
    ranging_threshold_selected: Boolean(row.ranging_threshold_selected),
  };
}

async function runMeasureExecution(payload: MeasureRunStartRequest) {
  const res = await calibrationApi.startMeasureRun(payload);
  if (res.error.value || !res.data.value) {
    isRunning.value = false;
    progress.value = 0;
    pushStatus('Measure execution failed to start.');
    return;
  }

  const response = res.data.value as MeasureRunStartResponse;
  if (response.requires_confirmation && Array.isArray(response.missing_downlink_channels) && response.missing_downlink_channels.length > 0) {
    pendingMeasurePayload.value = payload;
    missingDownlinkChannels.value = response.missing_downlink_channels;
    showMissingDownlinkDialog.value = true;
    isRunning.value = false;
    progress.value = 0;
    pushStatus('Downlink cal missing for selected channel(s). Awaiting user decision.');
    return;
  }

  const rows = Array.isArray(response.results) ? response.results : [];
  progress.value = 100;
  isRunning.value = false;

  for (const row of rows) {
    pushStatus(row.message);
    liveResults.value = [
      {
        time: new Date(row.timestamp).toLocaleTimeString(),
        code: row.code,
        port: row.port,
        frequency: String(row.frequency),
        result: `${row.parameter.toUpperCase()}: ${row.final_value.toFixed(1)} dBm (${row.status})`,
      },
      ...liveResults.value,
    ].slice(0, 200);
  }

  if (rows.length === 0) {
    pushLive('No selected parameter rows to execute');
  }
}

async function startExecution() {
  if (isRunning.value) return;
  const calId = String(selectedCalId.value ?? '').trim();
  if (calId === '') {
    pushStatus('Select Cal ID before starting execution.');
    return;
  }

  isRunning.value = true;
  progress.value = 5;
  pushStatus(`Execution started in ${executionMode.value === 'testplan' ? 'TestPlan' : 'Manual'} mode.`);

  const payload: MeasureRunStartRequest = {
    test_phase: String(selectedTestPhase.value ?? '').trim(),
    sub_test_phase: String(selectedSubTestPhase.value ?? '').trim(),
    cal_id: calId,
    test_plan_type: String(selectedTestPlanType.value ?? '').trim(),
    execution_mode: executionMode.value,
    remarks: String(remarks.value ?? '').trim(),
    continue_on_missing_downlink_cal: false,
    transmitter_rows: transmitterRows.value.map(toApiRow),
    receiver_rows: receiverRows.value.map(toApiRow),
    transponder_rows: transponderRows.value.map(toApiRow),
  };

  await runMeasureExecution(payload);
}

function abortMissingDownlink() {
  showMissingDownlinkDialog.value = false;
  pendingMeasurePayload.value = null;
  missingDownlinkChannels.value = [];
  isRunning.value = false;
  progress.value = 0;
  pushStatus('Execution aborted by user due to missing downlink cal channels.');
}

async function continueMissingDownlink() {
  const payload = pendingMeasurePayload.value;
  showMissingDownlinkDialog.value = false;
  missingDownlinkChannels.value = [];
  pendingMeasurePayload.value = null;
  if (!payload) return;

  isRunning.value = true;
  progress.value = 5;
  pushStatus('Continuing execution by skipping channels with missing downlink cal.');
  await runMeasureExecution({
    ...payload,
    continue_on_missing_downlink_cal: true,
  });
}

function pauseExecution() {
  if (!isRunning.value) return;
  pushStatus('Execution paused.');
  pushLive('Execution paused');
}

function abortExecution() {
  if (!isRunning.value) return;
  isRunning.value = false;
  progress.value = 0;
  pushStatus('Execution aborted by user.');
  pushLive('Execution aborted');
}

async function loadMeasureOptions() {
  const res = await calibrationApi.getMeasureOptions();
  if (res.error.value || !res.data.value) {
    pushStatus('Unable to load measure options.');
    return;
  }

  const payload = res.data.value as MeasureOptionsResponse;
  testPhaseOptions.value = payload.test_phases ?? [];
  calIdOptions.value = payload.cal_ids ?? [];
  testPlanTypeOptions.value = payload.test_plan_types ?? [];

  if (!selectedTestPhase.value && testPhaseOptions.value.length > 0) {
    selectedTestPhase.value = testPhaseOptions.value[0];
  }

  if (!selectedCalId.value) {
    selectedCalId.value = payload.default_cal_id ?? calIdOptions.value[0] ?? '';
  }

  if (!testPlanTypeOptions.value.includes(selectedTestPlanType.value)) {
    selectedTestPlanType.value = payload.default_test_plan_type ?? testPlanTypeOptions.value[0] ?? '';
  }
}

async function loadTransmitters() {
  const res = await transmitterApi.getTransmitters();
  if (res.error.value || !Array.isArray(res.data.value)) {
    pushStatus('Unable to load transmitter data for measure table.');
    return;
  }
  transmitters.value = res.data.value as Transmitter[];
}

async function loadReceivers() {
  const res = await transmitterApi.getReceivers();
  if (res.error.value || !Array.isArray(res.data.value)) {
    pushStatus('Unable to load receiver data for measure table.');
    return;
  }
  receivers.value = res.data.value as Receiver[];
}

async function loadTransponders() {
  const res = await transmitterApi.getProjectTransponders();
  if (res.error.value || !res.data.value) {
    pushStatus('Unable to load transponder data for measure table.');
    return;
  }
  const payload = res.data.value as ProjectTranspondersResponse;
  projectTransponders.value = Array.isArray(payload?.rows) ? payload.rows : [];
}

async function loadRangingThresholdRows() {
  const res = await transmitterApi.getRangingThresholdRows();
  if (res.error.value || !Array.isArray(res.data.value)) {
    pushStatus('Unable to load ranging threshold data for measure table.');
    return;
  }
  rangingThresholdRows.value = res.data.value as RangingThresholdRow[];
}

async function loadSavedTestPlanSelections() {
  // Manual mode: clear any saved selections so all checkboxes render unchecked.
  if (executionMode.value !== 'testplan') {
    savedTransmitterSelections.value = new Map();
    savedReceiverSelections.value = new Map();
    savedTransponderSelections.value = new Map();
    return;
  }
  const planName = (selectedTestPlanType.value ?? '').trim();
  if (!planName) {
    savedTransmitterSelections.value = new Map();
    savedReceiverSelections.value = new Map();
    savedTransponderSelections.value = new Map();
    return;
  }

  const [txRes, rxRes, tpRes] = await Promise.all([
    transmitterApi.getTestPlanSelections('transmitter', planName),
    transmitterApi.getTestPlanSelections('receiver', planName),
    transmitterApi.getTestPlanSelections('transponder', planName),
  ]);

  const buildSysMap = (data: any): Map<string, Record<string, boolean>> => {
    const m = new Map<string, Record<string, boolean>>();
    for (const sr of (data?.rows ?? [])) {
      const key = `${String(sr.code ?? '')}|${String(sr.port ?? '')}|${String(sr.frequency_label ?? '')}`;
      m.set(key, (sr.params && typeof sr.params === 'object') ? sr.params : {});
    }
    return m;
  };
  const buildTpMap = (data: any): Map<string, Record<string, boolean>> => {
    const m = new Map<string, Record<string, boolean>>();
    for (const sr of (data?.rows ?? [])) {
      const key = `${String(sr.transponder_code ?? '')}|${String(sr.uplink ?? '')}|${String(sr.downlink ?? '')}`;
      m.set(key, (sr.params && typeof sr.params === 'object') ? sr.params : {});
    }
    return m;
  };

  savedTransmitterSelections.value = txRes.error.value ? new Map() : buildSysMap(txRes.data.value);
  savedReceiverSelections.value = rxRes.error.value ? new Map() : buildSysMap(rxRes.data.value);
  savedTransponderSelections.value = tpRes.error.value ? new Map() : buildTpMap(tpRes.data.value);
}

watch([executionMode, selectedTestPlanType], () => {
  void loadSavedTestPlanSelections();
});

onMounted(async () => {
  loadAllManualSelections();
  await Promise.all([
    loadMeasureOptions(),
    loadTransmitters(),
    loadReceivers(),
    loadTransponders(),
    loadRangingThresholdRows(),
  ]);
  await loadSavedTestPlanSelections();
  await ui.load();
});

// Re-apply persisted grid state when underlying row data is rebuilt.
watch(transmitterRows, () => ui.reapplyGridState('transmitter'));
watch(receiverRows, () => ui.reapplyGridState('receiver'));
watch(transponderRows, () => ui.reapplyGridState('transponder'));

definePageMeta({
  title: 'TRACS-Nova Measure',
});

initMenu(0);
</script>

<style scoped>
.measure-page {
  height: calc(100vh - 4rem);
  min-height: 0;
  padding: 0.75rem 1.25rem 1rem;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  overflow: hidden;
  background: #081525;
}

.panel-card {
  background: #0d1b2e;
  border: 1px solid #1e3050;
  border-radius: 8px;
}

.toolbar {
  padding: 0.75rem;
  display: grid;
  grid-template-columns: repeat(5, minmax(170px, 1fr));
  gap: 0.75rem;
}

.toolbar-item {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.toolbar-item label {
  color: #94a3b8;
  font-size: 0.78rem;
  font-weight: 600;
}

.toolbar-select {
  width: 100%;
}

.mode-item {
  min-width: 220px;
}

.measure-layout {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: 1.65fr 1fr;
  gap: 0.75rem;
}

.left-pane,
.right-pane {
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.measure-tabs {
  flex: 1;
  min-height: 0;
  padding: 0.5rem;
  display: flex;
  flex-direction: column;
}

/* Make PrimeVue Tabs fill the available height so AG-Grid can use 100% */
.measure-tabs :deep(.p-tabpanels) {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 0;
}

.measure-tabs :deep(.p-tabpanel) {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 0.5rem 0 0;
}

.measure-tabs :deep(.p-tabpanel > *) {
  flex: 1;
  min-height: 0;
}

.tab-placeholder {
  min-height: 300px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #64748b;
}

.controls-row {
  display: flex;
  gap: 0.5rem;
  padding: 0.4rem 0.75rem;
  flex-shrink: 0;
}

.remarks-row {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  padding: 0.4rem 0.75rem;
  flex-shrink: 0;
}

.remarks-row label {
  color: #94a3b8;
  font-size: 0.8rem;
  font-weight: 600;
}

.status-window {
  margin: 0.5rem 0.75rem 0.75rem;
  border: 1px solid #1e3050;
  border-radius: 8px;
  background: #091425;
  min-height: 180px;
  max-height: 240px;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.status-title {
  padding: 0.55rem 0.75rem;
  color: #22d3ee;
  font-size: 0.82rem;
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
  padding: 0.12rem 0.45rem;
  font-size: 0.72rem;
  font-weight: 700;
}

.status-body {
  padding: 0.55rem 0.75rem;
  overflow-y: auto;
  color: #94a3b8;
  font-size: 0.8rem;
  line-height: 1.35;
}

.status-line {
  margin: 0 0 0.24rem 0;
}

.panel-header {
  padding: 0.7rem 0.85rem;
  border-bottom: 1px solid #1e3050;
  flex-shrink: 0;
}

.panel-header h3 {
  margin: 0;
  color: #22d3ee;
  font-size: 0.95rem;
}

.live-results-grid {
  flex: 1;
  min-height: 0;
}

@media (max-width: 1350px) {
  .toolbar {
    grid-template-columns: repeat(3, minmax(170px, 1fr));
  }
}

@media (max-width: 1100px) {
  .measure-layout {
    grid-template-columns: 1fr;
  }
  .toolbar {
    grid-template-columns: repeat(2, minmax(170px, 1fr));
  }
}

/* ── Light theme overrides ──────────────────────────────────────────────── */
html:not(.dark) .measure-page {
  background: var(--p-surface-50);
}
html:not(.dark) .panel-card {
  background: var(--p-surface-0);
  border-color: var(--p-content-border-color);
}
html:not(.dark) .toolbar-item label {
  color: var(--p-text-muted-color);
}
</style>
