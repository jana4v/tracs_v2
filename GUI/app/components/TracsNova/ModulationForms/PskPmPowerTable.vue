<template>
  <ag-grid-vue
    style="width: 100%;"
    :columnDefs="columnDefs"
    :defaultColDef="defaultColDef"
    :autoSizeStrategy="autoSizeStrategy"
    :enableRangeSelection="true"
    :enableFillHandle="true"
    rowGroupPanelShow="always"
    groupDisplayType="singleColumn"
    domLayout="autoHeight"
    :undoRedoCellEditing="true"
    :undoRedoCellEditingLimit="10"
    :getRowId="getRowId"
    :theme="isDark
      ? themeQuartz.withPart(colorSchemeDarkBlue)
      : themeQuartz.withPart(colorSchemeLightCold)"
    @grid-ready="onGridReady"
  />
</template>

<script lang="ts" setup>
import { ModuleRegistry } from 'ag-grid-community';
import { AllEnterpriseModule } from 'ag-grid-enterprise';
import {
  colorSchemeDarkBlue,
  colorSchemeLightCold,
  themeQuartz,
} from 'ag-grid-community';
import type { ColDef, GridApi, GridReadyEvent } from 'ag-grid-community';
import { AgGridVue } from 'ag-grid-vue3';
import type { PowerRow } from '@/composables/tracsNova/useTransmitterApi';

ModuleRegistry.registerModules([AllEnterpriseModule]);

// ── Props ─────────────────────────────────────────────────────────────────────

const props = defineProps<{
  rowData: PowerRow[];
  isEditable: boolean;
}>();

// ── Theme ─────────────────────────────────────────────────────────────────────

const isDark = useDark();

// ── Grid API ──────────────────────────────────────────────────────────────────

const gridApi = shallowRef<GridApi | null>(null);

// ── Column definitions (reactive — editable depends on isEditable prop) ───────

const columnDefs = computed<ColDef[]>(() => [
  { field: 'code',            headerName: 'Code',             editable: false,              minWidth: 70,  flex: 1   },
  { field: 'port',            headerName: 'Port',             editable: false,              minWidth: 70,  flex: 1   },
  { field: 'frequency_label', headerName: 'Freq Label',       editable: false,              minWidth: 90,  flex: 1.5 },
  { field: 'frequency',       headerName: 'Frequency (MHz)',  editable: false,              minWidth: 110, flex: 1.5 },
  { field: 'specification',   headerName: 'Specification (dBm)', editable: props.isEditable, minWidth: 120, flex: 1.5 },
  { field: 'tolerance',       headerName: 'Tolerance (± dB)',   editable: props.isEditable, minWidth: 120, flex: 1.3 },
  { field: 'fbt',             headerName: 'FBT (dBm)',          editable: props.isEditable, minWidth: 95,  flex: 1.1 },
  { field: 'fbt_hot',         headerName: 'FBT Hot (dBm)',      editable: props.isEditable, minWidth: 120, flex: 1.3 },
  { field: 'fbt_cold',        headerName: 'FBT Cold (dBm)',     editable: props.isEditable, minWidth: 120, flex: 1.3 },
]);

const defaultColDef: ColDef = {
  resizable: true,
  sortable: true,
  filter: true,
  enableRowGroup: true,
};

// Stretch all columns to fill the grid width
const autoSizeStrategy = { type: 'fitGridWidth' } as const;

// Stable row identity: port + freq label + freq
const getRowId = (params: any) =>
  `${params.data.port}|${params.data.frequency_label}|${params.data.frequency}`;

// ── Grid ready ────────────────────────────────────────────────────────────────

function onGridReady(event: GridReadyEvent) {
  gridApi.value = event.api;
  if (props.rowData.length > 0) {
    gridApi.value.applyTransaction({ add: props.rowData });
  }
}

// ── Public API ────────────────────────────────────────────────────────────────

/**
 * Merge new rows into the grid using applyTransaction so that existing rows
 * keep their user-edited spec values (specification, tolerance, fbt, fbt_hot,
 * fbt_cold).
 * Only adds new port×freq combinations, removes deleted ones, and updates rows
 * whose code has changed.
 */
function setRows(newRows: PowerRow[]) {
  if (!gridApi.value) return;

  const current: PowerRow[] = [];
  gridApi.value.forEachNode(n => { if (n.data) current.push(n.data); });

  const key = (r: PowerRow) => `${r.port}|${r.frequency_label}|${r.frequency}`;
  const currentMap = new Map(current.map(r => [key(r), r]));
  const newMap     = new Map(newRows.map(r => [key(r), r]));

  const toAdd    = newRows.filter(r => !currentMap.has(key(r)));
  const toRemove = current.filter(r => !newMap.has(key(r)));
  // Update only if the read-only identifier (code) has changed
  const toUpdate = newRows.filter(r => {
    const ex = currentMap.get(key(r));
    return ex && ex.code !== r.code;
  });

  gridApi.value.applyTransaction({ add: toAdd, remove: toRemove, update: toUpdate });
}

/** Return the current grid data as a PowerRow array. */
function getData(): PowerRow[] {
  const rows: PowerRow[] = [];
  gridApi.value?.forEachNode(n => { if (n.data) rows.push(n.data); });
  return rows;
}

defineExpose({ getData, setRows });
</script>
