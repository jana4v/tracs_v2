<template>
  <div class="tp-panel">
    <Toast />
    <div class="tp-header">
      <h2>Test Plan / Transmitter</h2>
    </div>

    <div class="tp-section">
      <div class="tp-section-header">
        <h3>Transmitter Test Plan</h3>
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
} from '@/composables/tracsNova/useTransmitterApi';

ModuleRegistry.registerModules([AllEnterpriseModule]);

const PM_PARAMS = ['power', 'frequency', 'modulation_index', 'spurious'];

interface TestPlanRow {
  test_plan_name: string;
  code: string;
  port: string;
  frequency_label: string;
  modulation_type: string;
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

function flattenPorts(ports: unknown): string[] {
  if (!Array.isArray(ports)) return [];
  const out: string[] = [];
  const walk = (v: unknown) => {
    if (Array.isArray(v)) {
      v.forEach(walk);
      return;
    }
    if (v === null || v === undefined) return;
    const text = String(v).trim();
    if (text !== '') out.push(text);
  };
  walk(ports);
  return [...new Set(out)];
}

function flattenFrequencies(freqs: unknown): Array<{ label: string; value: string }> {
  if (!Array.isArray(freqs)) return [];
  const out: Array<{ label: string; value: string }> = [];
  for (const row of freqs) {
    if (!Array.isArray(row) || row.length < 1) continue;
    const label = String(row[0] ?? '').trim();
    const value = String(row[1] ?? '').trim();
    if (label !== '') out.push({ label, value });
  }
  return out;
}

function getApplicableParameterNames(tx: Transmitter): string[] {
  const names = new Set<string>();
  const m = String(tx.modulation_type ?? '').toUpperCase();

  if (m === 'PSK_PM') {
    PM_PARAMS.forEach((p) => names.add(p));
  }

  const extra = (tx.modulation_details as any)?.test_parameters;
  if (extra && typeof extra === 'object' && !Array.isArray(extra)) {
    for (const key of Object.keys(extra)) {
      const normalized = String(key).trim().toLowerCase();
      if (normalized) names.add(normalized);
    }
  }

  return [...names];
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
    { field: 'code', headerName: 'Code', editable: false, minWidth: 130 },
    { field: 'port', headerName: 'Port', editable: false, minWidth: 120 },
    { field: 'frequency_label', headerName: 'Frequency Label', editable: false, minWidth: 170 },
  ];

  const paramsCols: ColDef[] = parameterColumns.value.map((param) => ({
    field: param,
    headerName: titleCase(param),
    editable: true,
    minWidth: 170,
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

function buildRows(transmitters: Transmitter[]): TestPlanRow[] {
  const out: TestPlanRow[] = [];
  const planNames = testPlanTypes.value.length > 0
    ? testPlanTypes.value
    : ['Detailed', 'Short', 'Go/No-Go'];

  for (const tx of transmitters) {
    const code = String(tx.code ?? '').trim();
    if (code === '') continue;

    const applicable = getApplicableParameterNames(tx);
    const applicableSet = new Set(applicable);

    const ports = flattenPorts((tx.modulation_details as any)?.ports);
    const freqs = flattenFrequencies((tx.modulation_details as any)?.frequencies);

    const safePorts = ports.length > 0 ? ports : [''];
    const safeFreqs = freqs.length > 0 ? freqs : [{ label: '', value: '' }];

    for (const plan of planNames) {
      for (const port of safePorts) {
        for (const fr of safeFreqs) {
          const row: TestPlanRow = {
            test_plan_name: plan,
            code,
            port,
            frequency_label: fr.label,
            modulation_type: String(tx.modulation_type ?? ''),
            _applicable: {},
          };

          for (const param of parameterColumns.value) {
            const isApplicable = applicableSet.has(param);
            row._applicable[param] = isApplicable;
            row[param] = isApplicable ? true : null;
          }

          out.push(row);
        }
      }
    }
  }

  return out;
}

async function load() {
  const [typesRes, transmittersRes] = await Promise.all([
    api.getTestPlanTypes(),
    api.getTransmitters(),
  ]);

  if (typesRes.error.value) {
    toast.add({ severity: 'error', summary: 'Load Failed', detail: 'Unable to load test plan types.', life: 3500 });
    return;
  }

  testPlanTypes.value = Array.isArray(typesRes.data.value) ? typesRes.data.value as string[] : [];

  if (transmittersRes.error.value) {
    toast.add({ severity: 'error', summary: 'Load Failed', detail: 'Unable to load transmitter rows.', life: 3500 });
    return;
  }

  const list = (transmittersRes.data.value as Transmitter[]) ?? [];
  const transmitters = list.filter((t) => String(t.system_type ?? '').toLowerCase().includes('transmitter'));

  const allParams = new Set<string>();
  for (const tx of transmitters) {
    getApplicableParameterNames(tx).forEach((p) => allParams.add(p));
  }

  const knownOrder = ['power', 'frequency', 'modulation_index', 'spurious'];
  const extras = [...allParams].filter((k) => !knownOrder.includes(k)).sort((a, b) => a.localeCompare(b));
  parameterColumns.value = [...knownOrder.filter((k) => allParams.has(k)), ...extras];

  const built = buildRows(transmitters);
  await applySavedSelections(built);
  rows.value = built;
}

function rowKey(r: { code: string; port: string; frequency_label: string }) {
  return `${r.code}|${r.port}|${r.frequency_label}`;
}

async function applySavedSelections(rs: TestPlanRow[]) {
  const plans = testPlanTypes.value;
  if (plans.length === 0) return;
  const results = await Promise.all(
    plans.map((p) => api.getTestPlanSelections('transmitter', p)),
  );
  const byPlan = new Map<string, Map<string, Record<string, boolean>>>();
  plans.forEach((plan, i) => {
    const res = results[i];
    if (res.error.value || !res.data.value) return;
    const data: any = res.data.value;
    const map = new Map<string, Record<string, boolean>>();
    for (const sr of (data.rows ?? [])) {
      const key = `${String(sr.code ?? '')}|${String(sr.port ?? '')}|${String(sr.frequency_label ?? '')}`;
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
          code: r.code,
          port: r.port,
          frequency_label: r.frequency_label,
          params: parameterColumns.value.reduce<Record<string, boolean>>((acc, p) => {
            acc[p] = r._applicable[p] ? !!r[p] : false;
            return acc;
          }, {}),
        })),
      };
      const res = await api.saveTestPlanSelections('transmitter', payload);
      if (res.error.value) errors.push(plan);
    }
    if (errors.length > 0) {
      toast.add({ severity: 'error', summary: 'Save Failed', detail: `Could not save: ${errors.join(', ')}`, life: 4000 });
    } else {
      toast.add({ severity: 'success', summary: 'Saved', detail: 'Transmitter test plan selections saved.', life: 2500 });
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
