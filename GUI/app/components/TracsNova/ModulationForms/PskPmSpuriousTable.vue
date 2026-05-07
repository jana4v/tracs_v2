<template>
  <ag-grid-vue
    style="width: 100%;"
    :gridOptions="gridOptions"
    :columnDefs="columnDefs"
    :defaultColDef="defaultColDef"
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
import { ref, shallowRef, computed, markRaw, watch, nextTick } from 'vue';
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
import type { SpuriousRow } from '@/composables/tracsNova/useTransmitterApi';
import PskPmSpuriousFbtCellAgGrid from './PskPmSpuriousFbtCellAgGrid.vue';

ModuleRegistry.registerModules([AllEnterpriseModule]);

const props = defineProps<{
  rowData: SpuriousRow[];
  isEditable: boolean;
}>();

const isDark = useDark();
const gridApi = shallowRef<GridApi | null>(null);

// Create three distinct component objects so AG Grid doesn't deduplicate them.
// markRaw({ ...component }) produces a new object each time — critical for
// multi-column reuse of the same Vue component as a cell renderer.
const fbtRenderer = markRaw({ ...PskPmSpuriousFbtCellAgGrid });
const fbtHotRenderer = markRaw({ ...PskPmSpuriousFbtCellAgGrid });
const fbtColdRenderer = markRaw({ ...PskPmSpuriousFbtCellAgGrid });

// Simple pass-through editor for fill handle support on nested grid columns
const fbtCellEditor = markRaw({
  template: '<div></div>',
  setup() {
    return { getValue: () => undefined };
  },
});

// Custom fill value handler to support filling with nested grid data
const setFillValue = (params: any) => {
  const values = params?.values;
  if (Array.isArray(values) && values.length > 0) {
    const sourceValue = values[values.length - 1];
    // Deep copy the array to prevent reference sharing
    if (Array.isArray(sourceValue)) {
      return sourceValue.map(row => Array.isArray(row) ? [...row] : row);
    }
    return sourceValue;
  }
  const currentValue = params?.currentCellValue;
  if (Array.isArray(currentValue)) {
    return currentValue.map(row => Array.isArray(row) ? [...row] : row);
  }
  return currentValue;
};

const gridOptions = computed(() => ({
  singleClickEdit: false,
  suppressClickEdit: true,  // Prevent single click edit
  suppressColumnVirtualisation: true,  // Disable column virtualization to keep all cells rendered
  suppressAnimationFrame: true,  // Disable animation frame updates that can destroy renderers
  cellSelection: {
    mode: 'range' as const,
    handle: {
      mode: 'fill' as const,
      direction: 'xy' as const,
      suppressClearOnFillReduction: true,
      setFillValue,
    },
  },
  onCellMouseDown: (params: any) => {
    // Block AG Grid edit mode on FBT cells, but allow fill-handle drags through
    const field = params.colDef?.field;
    if (['fbt', 'fbt_hot', 'fbt_cold'].includes(field)) {
      const target = params.event?.target as HTMLElement | null;
      if (target?.closest?.('.ag-fill-handle')) return; // let fill handle work
      params.event?.stopPropagation?.();
      params.event?.preventDefault?.();
    }
  },
}));

const columnDefs = computed<ColDef[]>(() => [
  { field: 'code',            headerName: 'Code',                 editable: false,            minWidth: 70,  flex: 1 },
  { field: 'port',            headerName: 'Port',                 editable: false,            minWidth: 70,  flex: 1 },
  { field: 'frequency_label', headerName: 'Freq Label',           editable: false,            minWidth: 90,  flex: 1.2 },
  { field: 'frequency',       headerName: 'Frequency (MHz)',      editable: false,            minWidth: 110, flex: 1.3 },
  { field: 'specification',   headerName: 'Specification (dBc)',  editable: props.isEditable, minWidth: 130, flex: 1.2 },
  { field: 'tolerance',       headerName: 'Tolerance (dBc)',      editable: props.isEditable, minWidth: 120, flex: 1.1 },
  {
    field: 'fbt',
    headerName: 'FBT',
    editable: false,  // Not editable by AG Grid - editing happens inside HandsOnTable
    suppressFillHandle: false,
    minWidth: 280,
    flex: 2,
    cellRenderer: fbtRenderer,
    cellRendererParams: { isEditable: props.isEditable },
    cellClass: 'fbt-cell-renderer',
    valueSetter: (params) => {
      params.data.fbt = params.newValue;
      return true;
    },
    suppressKeyboardEvent: (params) => {
      // Don't let keyboard events bubble to AG Grid
      if (params.event?.type === 'keydown' || params.event?.type === 'keyup') {
        return true;
      }
      return false;
    },
  },
  {
    field: 'fbt_hot',
    headerName: 'FBT Hot',
    editable: false,  // Not editable by AG Grid - editing happens inside HandsOnTable
    suppressFillHandle: false,
    minWidth: 280,
    flex: 2,
    cellRenderer: fbtHotRenderer,
    cellRendererParams: { isEditable: props.isEditable },
    cellClass: 'fbt-cell-renderer',
    valueSetter: (params) => {
      params.data.fbt_hot = params.newValue;
      return true;
    },
    suppressKeyboardEvent: (params) => {
      if (params.event?.type === 'keydown' || params.event?.type === 'keyup') {
        return true;
      }
      return false;
    },
  },
  {
    field: 'fbt_cold',
    headerName: 'FBT Cold',
    editable: false,  // Not editable by AG Grid - editing happens inside HandsOnTable
    suppressFillHandle: false,
    minWidth: 280,
    flex: 2,
    cellRenderer: fbtColdRenderer,
    cellRendererParams: { isEditable: props.isEditable },
    cellClass: 'fbt-cell-renderer',
    valueSetter: (params) => {
      params.data.fbt_cold = params.newValue;
      return true;
    },
    suppressKeyboardEvent: (params) => {
      if (params.event?.type === 'keydown' || params.event?.type === 'keyup') {
        return true;
      }
      return false;
    },
  },
]);

const defaultColDef: ColDef = {
  resizable: true,
  sortable: true,
  filter: true,
  enableRowGroup: true,
};

const getRowId = (params: any) =>
  `${params.data.port}|${params.data.frequency_label}|${params.data.frequency}`;

const getRowHeight = (params: any): number => {
  const data = params?.data;
  if (!data) return 90;

  const toCompactText = (value: unknown) => {
    if (!Array.isArray(value)) return '';
    return value
      .map((row: any) => Array.isArray(row) ? row.filter((cell: any) => cell !== '' && cell !== null && cell !== undefined).join(', ') : '')
      .filter((segment: string) => segment !== '')
      .join('; ');
  };

  const maxTextLength = Math.max(
    toCompactText(data.fbt).length,
    toCompactText(data.fbt_hot).length,
    toCompactText(data.fbt_cold).length,
  );

  if (maxTextLength <= 48) return 78;
  if (maxTextLength <= 120) return 104;
  return 132;
};

// ── When isEditable changes, refresh FBT cells so the nested cell renderers
// receive updated params. force:false updates params via refresh() without
// destroying/recreating the cell renderer Vue components. ────────────────
watch(() => props.isEditable, () => {
  nextTick(() => {
    gridApi.value?.refreshCells({
      columns: ['fbt', 'fbt_hot', 'fbt_cold'],
    });
  });
});

function onGridReady(event: GridReadyEvent) {
  gridApi.value = event.api;
  
  if (props.rowData.length > 0) {
    // Ensure all FBT fields are initialized before adding to grid
    const normalizedData = props.rowData.map(row => ({
      ...row,
      fbt: Array.isArray(row.fbt) ? row.fbt : [['', '']],
      fbt_hot: Array.isArray(row.fbt_hot) ? row.fbt_hot : [['', '']],
      fbt_cold: Array.isArray(row.fbt_cold) ? row.fbt_cold : [['', '']],
    }));
    gridApi.value.applyTransaction({ add: normalizedData });
  }
}

function setRows(newRows: SpuriousRow[]) {
  if (!gridApi.value) return;

  // Ensure all FBT fields are initialized
  const normalizedRows = newRows.map(row => ({
    ...row,
    fbt: Array.isArray(row.fbt) ? row.fbt : [['', '']],
    fbt_hot: Array.isArray(row.fbt_hot) ? row.fbt_hot : [['', '']],
    fbt_cold: Array.isArray(row.fbt_cold) ? row.fbt_cold : [['', '']],
  }));

  const current: SpuriousRow[] = [];
  gridApi.value.forEachNode((n) => { if (n.data) current.push(n.data as SpuriousRow); });

  const key = (r: SpuriousRow) => `${r.port}|${r.frequency_label}|${r.frequency}`;
  const currentMap = new Map(current.map(r => [key(r), r]));
  const newMap = new Map(normalizedRows.map(r => [key(r), r]));

  const toAdd = normalizedRows.filter(r => !currentMap.has(key(r)));
  const toRemove = current.filter(r => !newMap.has(key(r)));

  const toUpdate = normalizedRows.filter((r) => {
    const ex = currentMap.get(key(r));
    return ex && ex.code !== r.code;
  });

  gridApi.value.applyTransaction({ add: toAdd, remove: toRemove, update: toUpdate });
  gridApi.value.resetRowHeights();
}

function getData(): SpuriousRow[] {
  const rows: SpuriousRow[] = [];
  gridApi.value?.forEachNode((n) => { if (n.data) rows.push(n.data as SpuriousRow); });
  return rows;
}

defineExpose({ getData, setRows });
</script>

<style scoped>
/*
 * FBT cell styling - ensure the nested HandsOnTable stays visible and interactive
 */
:deep(.fbt-cell-renderer) {
  overflow: visible !important;
  max-height: none !important;
  height: 100% !important;
}

:deep(.ag-cell) {
  overflow: visible !important;
}

:deep(.ag-row) {
  overflow: visible !important;
}
</style>
