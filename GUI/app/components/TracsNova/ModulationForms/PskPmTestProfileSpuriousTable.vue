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
    :suppressColumnVirtualisation="true"
    :undoRedoCellEditing="true"
    :undoRedoCellEditingLimit="10"
    :getRowId="getRowId"
    :getRowHeight="getRowHeight"
    :theme="isDark
      ? themeQuartz.withPart(colorSchemeDarkBlue)
      : themeQuartz.withPart(colorSchemeLightCold)"
    @grid-ready="onGridReady"
  />
</template>

<script lang="ts" setup>
import { shallowRef, computed, markRaw } from 'vue';
import { useDark } from '@vueuse/core';
import { ModuleRegistry } from 'ag-grid-community';
import { AllEnterpriseModule } from 'ag-grid-enterprise';
import {
  colorSchemeDarkBlue,
  colorSchemeLightCold,
  themeQuartz,
} from 'ag-grid-community';
import type { ColDef, GridApi, GridReadyEvent } from 'ag-grid-community';
import { AgGridVue } from 'ag-grid-vue3';
import type { TestProfileSpuriousRow } from '@/composables/tracsNova/useTransmitterApi';
import PskPmTestProfileSpuriousBandsCellAgGrid from './PskPmTestProfileSpuriousBandsCellAgGrid.vue';

ModuleRegistry.registerModules([AllEnterpriseModule]);

const props = defineProps<{
  rowData: TestProfileSpuriousRow[];
  isEditable: boolean;
}>();

const isDark = useDark();
const gridApi = shallowRef<GridApi | null>(null);
const bandsCellRenderer = markRaw(PskPmTestProfileSpuriousBandsCellAgGrid);

const columnDefs = computed<ColDef[]>(() => [
  { field: 'code', headerName: 'Code', editable: false, minWidth: 70, flex: 1 },
  { field: 'port', headerName: 'Port', editable: false, minWidth: 70, flex: 1 },
  { field: 'frequency_label', headerName: 'Freq Label', editable: false, minWidth: 90, flex: 1.4 },
  { field: 'frequency', headerName: 'Frequency (MHz)', editable: false, minWidth: 110, flex: 1.4 },
  {
    field: 'spurious_search_bands',
    headerName: 'Spurious Search Bands',
    editable: false,
    suppressFillHandle: true,
    minWidth: 300,
    flex: 2.3,
    cellRenderer: bandsCellRenderer,
    cellRendererParams: { isEditable: props.isEditable },
    valueSetter: (params) => {
      params.data.spurious_search_bands = params.newValue;
      return true;
    },
  },
  {
    field: 'enable',
    headerName: 'Enable',
    editable: props.isEditable,
    minWidth: 90,
    maxWidth: 110,
    flex: 0.8,
    cellDataType: 'boolean',
    cellRenderer: 'agCheckboxCellRenderer',
    cellEditor: 'agCheckboxCellEditor',
  },
]);

const defaultColDef: ColDef = {
  resizable: true,
  sortable: true,
  filter: true,
  enableRowGroup: true,
};

const autoSizeStrategy = { type: 'fitGridWidth' } as const;

const getRowId = (params: any) =>
  `${params.data.port}|${params.data.frequency_label}|${params.data.frequency}`;

const getRowHeight = (params: any): number => {
  // Dynamic height based on displayed rows (max 3 shown in inline table)
  const data = params?.data;
  if (!data) return 90;
  
  const rows = Math.min(3, Array.isArray(data.spurious_search_bands) ? data.spurious_search_bands.length : 1);
  
  // Bigger baseline so nested tables are readable by default
  return 52 + (rows * 32) + 14;
};

function onGridReady(event: GridReadyEvent) {
  gridApi.value = event.api;
  if (props.rowData.length > 0) {
    gridApi.value.applyTransaction({ add: props.rowData });
  }
}

function setRows(newRows: TestProfileSpuriousRow[]) {
  if (!gridApi.value) return;

  const current: TestProfileSpuriousRow[] = [];
  gridApi.value.forEachNode((n) => { if (n.data) current.push(n.data as TestProfileSpuriousRow); });

  const key = (r: TestProfileSpuriousRow) => `${r.port}|${r.frequency_label}|${r.frequency}`;
  const currentMap = new Map(current.map(r => [key(r), r]));
  const newMap = new Map(newRows.map(r => [key(r), r]));

  const toAdd = newRows.filter(r => !currentMap.has(key(r)));
  const toRemove = current.filter(r => !newMap.has(key(r)));
  const toUpdate = newRows.filter((r) => {
    const ex = currentMap.get(key(r));
    return ex && ex.code !== r.code;
  });

  gridApi.value.applyTransaction({ add: toAdd, remove: toRemove, update: toUpdate });
  gridApi.value.resetRowHeights();
}

function getData(): TestProfileSpuriousRow[] {
  const rows: TestProfileSpuriousRow[] = [];
  gridApi.value?.forEachNode((n) => { if (n.data) rows.push(n.data as TestProfileSpuriousRow); });
  return rows;
}

defineExpose({ getData, setRows });
</script>
