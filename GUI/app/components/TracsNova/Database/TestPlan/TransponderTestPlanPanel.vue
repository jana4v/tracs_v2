<template>
  <div class="tp-panel">
    <Toast />
    <div class="tp-header">
      <h2>Test Plan / Transponder</h2>
    </div>

    <div class="tp-section">
      <div class="tp-section-header">
        <h3>Transponder Test Plan</h3>
        <div class="actions">
          <Button label="Refresh" size="small" severity="secondary" @click="load" />
          <Button label="Save" size="small" severity="primary" :loading="saving" @click="save" />
        </div>
      </div>

      <ag-grid-vue
        class="tp-grid"
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
import type { ColDef, ICellRendererParams } from 'ag-grid-community';
import { AgGridVue } from 'ag-grid-vue3';
import {
  useTransmitterApi,
  type Transmitter,
  type Receiver,
  type ProjectTranspondersResponse,
  type ProjectTransponderRow,
  type RangingThresholdRow,
} from '@/composables/tracsNova/useTransmitterApi';

ModuleRegistry.registerModules([AllEnterpriseModule]);

// Applicable parameters for a transponder when uplink is PSK_FM and downlink is PSK_PM.
const TRANSPONDER_PARAMS = ['ranging_threshold'];

interface TestPlanRow {
  test_plan_name: string;
  code: string;
  uplink: string;
  downlink: string;
  _applicable: Record<string, boolean>;
  [key: string]: any;
}

const toast = useToast();
const isDark = useDark();
const api = useTransmitterApi();

const rows = ref<TestPlanRow[]>([]);
const parameterColumns = ref<string[]>([]);
const testPlanTypes = ref<string[]>([]);
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

function titleCase(value: string): string {
  return value
    .replaceAll('_', ' ')
    .replace(/\b\w/g, (c) => c.toUpperCase());
}

function checkboxRenderer(params: ICellRendererParams) {
  const field = String(params.colDef?.field ?? '');
  const row = params.data as TestPlanRow | undefined;
  if (!row) return '';

  const applicable = !!row._applicable?.[field];
  if (!applicable) return '';

  const input = document.createElement('input');
  input.type = 'checkbox';
  input.checked = !!params.value;
  input.style.cursor = 'pointer';
  input.addEventListener('click', (e) => e.stopPropagation());
  input.addEventListener('change', () => {
    params.node?.setDataValue(field, input.checked);
  });
  return input;
}

const columnDefs = computed<ColDef[]>(() => {
  const base: ColDef[] = [
    {
      field: 'test_plan_name',
      headerName: 'Test Plan Name',
      editable: false,
      sort: 'asc',
      comparator: (a, b) => {
        const order = testPlanTypes.value;
        const ai = order.indexOf(String(a));
        const bi = order.indexOf(String(b));
        if (ai === -1 && bi === -1) return String(a).localeCompare(String(b));
        if (ai === -1) return 1;
        if (bi === -1) return -1;
        return ai - bi;
      },
    },
    { field: 'code', headerName: 'TpCode', editable: false, minWidth: 130 },
    { field: 'uplink', headerName: 'Uplink', editable: false, minWidth: 170 },
    { field: 'downlink', headerName: 'Downlink', editable: false, minWidth: 170 },
  ];

  const paramsCols: ColDef[] = parameterColumns.value.map((param) => ({
    field: param,
    headerName: titleCase(param),
    editable: true,
    minWidth: 180,
    cellRenderer: checkboxRenderer,
    valueGetter: (p) => {
      const row = p.data as TestPlanRow | undefined;
      if (!row) return null;
      return row._applicable?.[param] ? !!row[param] : null;
    },
    valueSetter: (p) => {
      const row = p.data as TestPlanRow | undefined;
      if (!row || !row._applicable?.[param]) return false;
      row[param] = !!p.newValue;
      return true;
    },
  }));

  return [...base, ...paramsCols];
});

function buildRows(
  rangingRows: RangingThresholdRow[],
  projectTransponders: ProjectTransponderRow[],
  receivers: Receiver[],
  transmitters: Transmitter[],
): TestPlanRow[] {
  const out: TestPlanRow[] = [];
  const planNames = testPlanTypes.value.length > 0
    ? testPlanTypes.value
    : ['Detailed', 'Short', 'Go/No-Go'];

  // Index project transponders by code so we can resolve rx/tx codes.
  const tpByCode = new Map<string, ProjectTransponderRow>();
  for (const t of projectTransponders) {
    const c = String(t?.code ?? '').trim();
    if (c) tpByCode.set(c, t);
  }

  // Index receivers and transmitters by code so we can read modulation_type.
  const rxModByCode = new Map<string, string>();
  for (const r of receivers) {
    const c = String(r?.code ?? '').trim();
    if (c) rxModByCode.set(c, String(r.modulation_type ?? '').toUpperCase());
  }
  const txModByCode = new Map<string, string>();
  for (const t of transmitters) {
    const c = String(t?.code ?? '').trim();
    if (c) txModByCode.set(c, String(t.modulation_type ?? '').toUpperCase());
  }

  // Deduplicate by (transponder_code, uplink, downlink) so the test plan rows match the measure tab.
  const seenLeg = new Set<string>();
  const uniqueLegs: Array<{ code: string; uplink: string; downlink: string; applicable: boolean }> = [];
  for (const r of rangingRows) {
    const code = String(r?.transponder_code ?? '').trim();
    const uplink = String(r?.uplink ?? '').trim();
    const downlink = String(r?.downlink ?? '').trim();
    if (!code) continue;
    const k = `${code}|${uplink}|${downlink}`;
    if (seenLeg.has(k)) continue;
    seenLeg.add(k);

    const tp = tpByCode.get(code);
    const rxMod = tp ? rxModByCode.get(String(tp.rx_code ?? '').trim()) ?? '' : '';
    const txMod = tp ? txModByCode.get(String(tp.tx_code ?? '').trim()) ?? '' : '';
    const applicable = rxMod === 'PSK_FM' && txMod === 'PSK_PM';

    uniqueLegs.push({ code, uplink, downlink, applicable });
  }

  for (const plan of planNames) {
    for (const leg of uniqueLegs) {
      const row: TestPlanRow = {
        test_plan_name: plan,
        code: leg.code,
        uplink: leg.uplink,
        downlink: leg.downlink,
        _applicable: {},
      };
      for (const param of parameterColumns.value) {
        row._applicable[param] = leg.applicable;
        row[param] = leg.applicable ? true : null;
      }
      out.push(row);
    }
  }

  return out;
}

async function load() {
  const [typesRes, transpondersRes, rangingRes, receiversRes, transmittersRes] = await Promise.all([
    api.getTestPlanTypes(),
    api.getProjectTransponders(),
    api.getRangingThresholdRows(),
    api.getReceivers(),
    api.getTransmitters(),
  ]);

  if (typesRes.error.value) {
    toast.add({ severity: 'error', summary: 'Load Failed', detail: 'Unable to load test plan types.', life: 3500 });
    return;
  }
  testPlanTypes.value = Array.isArray(typesRes.data.value) ? typesRes.data.value as string[] : [];

  if (transpondersRes.error.value || rangingRes.error.value) {
    toast.add({ severity: 'error', summary: 'Load Failed', detail: 'Unable to load transponder data.', life: 3500 });
    return;
  }

  const tpPayload = (transpondersRes.data.value as ProjectTranspondersResponse) ?? { rows: [] };
  const projectTransponders = Array.isArray(tpPayload?.rows) ? tpPayload.rows : [];
  const rangingRows = Array.isArray(rangingRes.data.value) ? (rangingRes.data.value as RangingThresholdRow[]) : [];
  const receivers = Array.isArray(receiversRes.data.value) ? (receiversRes.data.value as Receiver[]) : [];
  const transmitters = Array.isArray(transmittersRes.data.value) ? (transmittersRes.data.value as Transmitter[]) : [];

  parameterColumns.value = [...TRANSPONDER_PARAMS];
  const built = buildRows(rangingRows, projectTransponders, receivers, transmitters);
  await applySavedSelections(built);
  rows.value = built;
}

function rowKey(r: { code: string; uplink: string; downlink: string }) {
  return `${r.code}|${r.uplink}|${r.downlink}`;
}

async function applySavedSelections(rs: TestPlanRow[]) {
  const plans = testPlanTypes.value;
  if (plans.length === 0) return;
  const results = await Promise.all(
    plans.map((p) => api.getTestPlanSelections('transponder', p)),
  );
  const byPlan = new Map<string, Map<string, Record<string, boolean>>>();
  plans.forEach((plan, i) => {
    const res = results[i];
    if (res.error.value || !res.data.value) return;
    const data: any = res.data.value;
    const map = new Map<string, Record<string, boolean>>();
    for (const sr of (data.rows ?? [])) {
      const key = `${String(sr.transponder_code ?? '')}|${String(sr.uplink ?? '')}|${String(sr.downlink ?? '')}`;
      const params = (sr.params && typeof sr.params === 'object') ? sr.params : {};
      map.set(key, params);
    }
    byPlan.set(plan, map);
  });
  for (const r of rs) {
    const planMap = byPlan.get(r.test_plan_name);
    if (!planMap) continue;
    const saved = planMap.get(rowKey(r));
    if (!saved) continue;
    for (const param of parameterColumns.value) {
      if (!r._applicable[param]) continue;
      if (Object.prototype.hasOwnProperty.call(saved, param)) {
        r[param] = !!saved[param];
      }
    }
  }
}

async function save() {
  if (saving.value) return;
  saving.value = true;
  try {
    const groups = new Map<string, TestPlanRow[]>();
    for (const r of rows.value) {
      const list = groups.get(r.test_plan_name) ?? [];
      list.push(r);
      groups.set(r.test_plan_name, list);
    }
    const errors: string[] = [];
    for (const [plan, planRows] of groups) {
      const payload = {
        test_plan_name: plan,
        rows: planRows.map((r) => ({
          transponder_code: r.code,
          uplink: r.uplink,
          downlink: r.downlink,
          params: parameterColumns.value.reduce<Record<string, boolean>>((acc, p) => {
            acc[p] = r._applicable[p] ? !!r[p] : false;
            return acc;
          }, {}),
        })),
      };
      const res = await api.saveTestPlanSelections('transponder', payload);
      if (res.error.value) errors.push(plan);
    }
    if (errors.length > 0) {
      toast.add({ severity: 'error', summary: 'Save Failed', detail: `Could not save: ${errors.join(', ')}`, life: 4000 });
    } else {
      toast.add({ severity: 'success', summary: 'Saved', detail: 'Transponder test plan selections saved.', life: 2500 });
    }
  } finally {
    saving.value = false;
  }
}

onMounted(() => {
  void load();
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

.actions { display: flex; gap: 0.5rem; }

.tp-grid { flex: 1; min-height: 0; }
</style>
