<template>
  <div class="spurious-bands-ag-cell">
    <ag-grid-vue
      style="width: 100%; height: 100%;"
      :gridOptions="gridOptions"
      :columnDefs="columnDefs"
      :rowData="gridRows"
      :defaultColDef="defaultColDef"
      domLayout="autoHeight"
      :undoRedoCellEditing="true"
      :suppressMovableColumns="true"
      :suppressRowHoverHighlight="false"
      :suppressSelectionByClick="false"
      :headerHeight="36"
      :rowHeight="36"
      :getContextMenuItems="getContextMenuItems"
      :theme="isDark
        ? themeQuartz.withPart(colorSchemeDarkBlue)
        : themeQuartz.withPart(colorSchemeLightCold)"
      @grid-ready="onGridReady"
      @cell-value-changed="onCellValueChanged"
      @cell-key-down="onCellKeyDown"
    />
  </div>
</template>

<script lang="ts" setup>
import { ref, shallowRef, computed, watch } from 'vue';
import { useDark } from '@vueuse/core';
import { ModuleRegistry } from 'ag-grid-community';
import { AllEnterpriseModule } from 'ag-grid-enterprise';
import {
  colorSchemeDarkBlue,
  colorSchemeLightCold,
  themeQuartz,
} from 'ag-grid-community';
import type { CellValueChangedEvent, ColDef, GridApi, GridReadyEvent } from 'ag-grid-community';
import { AgGridVue } from 'ag-grid-vue3';

ModuleRegistry.registerModules([AllEnterpriseModule]);

type BandsMatrix = (string | number)[][];

interface BandRow {
  band_start: string | number;
  band_stop: string | number;
}

const props = defineProps<{ params: any }>();
const isDark = useDark();
const gridApi = shallowRef<GridApi | null>(null);
const isUpdatingFromParent = ref(false);

const setFillValue = (params: any) => {
  const values = params?.values;
  if (Array.isArray(values) && values.length > 0) {
    return values[values.length - 1];
  }

  const initialValues = params?.initialValues;
  if (Array.isArray(initialValues) && initialValues.length > 0) {
    return initialValues[0];
  }

  return params?.currentCellValue;
};

// Proper AG Grid v33 fill-handle configuration with range selection ENABLED
const gridOptions = computed(() => ({
  singleClickEdit: false,
  suppressSelectionByClick: false,
  cellSelection: {
    mode: 'range' as const,
    handle: {
      mode: 'fill' as const,
      direction: 'xy' as const,
      suppressClearOnFillReduction: true,
      setFillValue,
    },
  },
}));

const ensureMatrix = (value: unknown): BandsMatrix => {
  if (Array.isArray(value) && value.length > 0 && value.every((row) => Array.isArray(row))) {
    return value as BandsMatrix;
  }
  return [['', '']];
};

const matrixToRows = (matrix: BandsMatrix): BandRow[] => {
  return matrix.map(([band_start, band_stop]) => ({
    band_start: band_start ?? '',
    band_stop: band_stop ?? '',
  }));
};

const rowsToMatrix = (rows: BandRow[]): BandsMatrix => {
  return rows.map((row) => [row.band_start, row.band_stop]);
};

const gridRows = ref<BandRow[]>(matrixToRows(ensureMatrix(props.params?.value)));

watch(
  () => props.params?.value,
  (val) => {
    if (isUpdatingFromParent.value) {
      isUpdatingFromParent.value = false;
      return;
    }

    const nextRows = matrixToRows(ensureMatrix(val));
    if (JSON.stringify(nextRows) !== JSON.stringify(gridRows.value)) {
      gridRows.value = nextRows;
    }
  },
);

// Keep nested grid editable unless parent explicitly passes false.
const isEditable = computed(() => props.params?.isEditable !== false);

const columnDefs = computed<ColDef[]>(() => [
  {
    field: 'band_start',
    headerName: 'Band Start (MHz)',
    editable: isEditable.value,
    suppressFillHandle: false,
    minWidth: 130,
    width: 160,
    cellDataType: 'number',
    valueParser: (params) => {
      const val = params.newValue;
      if (val === '' || val === null || val === undefined) return '';
      const num = Number(val);
      return Number.isNaN(num) ? '' : num;
    },
  },
  {
    field: 'band_stop',
    headerName: 'Band Stop (MHz)',
    editable: isEditable.value,
    suppressFillHandle: false,
    minWidth: 130,
    width: 160,
    cellDataType: 'number',
    valueParser: (params) => {
      const val = params.newValue;
      if (val === '' || val === null || val === undefined) return '';
      const num = Number(val);
      return Number.isNaN(num) ? '' : num;
    },
  },
]);

const defaultColDef: ColDef = {
  resizable: false,
  sortable: false,
  filter: false,
};

function getContextMenuItems(params: any): any[] {
  if (!isEditable.value) return [];

  return [
    {
      name: 'Add Row Below',
      action: () => {
        const currentIndex = params.node?.rowIndex ?? 0;
        const newRow: BandRow = { band_start: '', band_stop: '' };
        gridApi.value?.applyTransaction({ add: [newRow], addIndex: currentIndex + 1 });
        setTimeout(() => gridApi.value?.autoSizeAllColumns(), 0);
        syncAndNotifyParent();
      },
    },
    {
      name: 'separator',
    },
    {
      name: 'Remove Row',
      action: () => {
        if (params.node) {
          gridApi.value?.applyTransaction({ remove: [params.node.data] });
          setTimeout(() => gridApi.value?.autoSizeAllColumns(), 0);
          syncAndNotifyParent();
        }
      },
      disabled: gridRows.value.length <= 1,
    },
  ];
}

function syncAndNotifyParent() {
  const currentRows: BandRow[] = [];
  gridApi.value?.forEachNode((node) => {
    if (node.data) currentRows.push(node.data as BandRow);
  });

  const newMatrix = rowsToMatrix(currentRows);
  const field = props.params?.colDef?.field;

  if (field && props.params?.node) {
    isUpdatingFromParent.value = true;
    props.params.node.setDataValue(field, newMatrix);
    props.params?.api?.resetRowHeights();
  }
}

function onGridReady(event: GridReadyEvent) {
  gridApi.value = event.api;
  event.api.autoSizeAllColumns();
}

function onCellValueChanged(_event: CellValueChangedEvent) {
  syncAndNotifyParent();
}


async function writeTextToClipboard(text: string) {
  if (!text) return;

  try {
    await navigator.clipboard.writeText(text);
    return;
  } catch {
    // Fallback for clipboard API restrictions in embedded grids
  }

  const ta = document.createElement('textarea');
  ta.value = text;
  ta.style.position = 'fixed';
  ta.style.left = '-9999px';
  document.body.appendChild(ta);
  ta.focus();
  ta.select();
  document.execCommand('copy');
  document.body.removeChild(ta);
}

function buildClipboardTextFromSelection(api: any): string {
  const ranges = api?.getCellRanges?.() ?? [];
  if (!ranges.length) {
    const focused = api?.getFocusedCell?.();
    if (!focused) return '';
    const value = api?.getDisplayedRowAtIndex?.(focused.rowIndex)?.data?.[focused.column.getColId?.() ?? focused.column.colId];
    return value === null || value === undefined ? '' : String(value);
  }

  const range = ranges[0];
  const start = Math.min(range.startRow?.rowIndex ?? 0, range.endRow?.rowIndex ?? 0);
  const end = Math.max(range.startRow?.rowIndex ?? 0, range.endRow?.rowIndex ?? 0);
  const cols = range.columns ?? [];
  const lines: string[] = [];

  for (let r = start; r <= end; r += 1) {
    const rowNode = api.getDisplayedRowAtIndex?.(r);
    const rowData = rowNode?.data ?? {};
    const cells = cols.map((col: any) => {
      const key = col.getColId?.() ?? col.colId;
      const val = rowData[key];
      return val === null || val === undefined ? '' : String(val);
    });
    lines.push(cells.join('\t'));
  }

  return lines.join('\n');
}

function onCellKeyDown(event: any) {
  const keyboardEvent = event?.event as KeyboardEvent | undefined;
  if (!keyboardEvent) return;

  const isCopy = (keyboardEvent.ctrlKey || keyboardEvent.metaKey) && keyboardEvent.key.toLowerCase() === 'c';
  const isPaste = (keyboardEvent.ctrlKey || keyboardEvent.metaKey) && keyboardEvent.key.toLowerCase() === 'v';

  if (isCopy) {
    const copiedByApi = typeof event.api?.copySelectedRangeToClipboard === 'function';
    if (copiedByApi) {
      event.api.copySelectedRangeToClipboard();
    } else {
      const text = buildClipboardTextFromSelection(event.api);
      void writeTextToClipboard(text);
    }
    keyboardEvent.preventDefault();
    keyboardEvent.stopPropagation();
    return;
  }

  if (isPaste) {
    event.api?.pasteFromClipboard?.();
    keyboardEvent.preventDefault();
    keyboardEvent.stopPropagation();
  }
}
</script>

<style scoped>
.spurious-bands-ag-cell {
  width: 100%;
  min-width: 360px;
  padding: 0.25rem 0;
  pointer-events: auto; /* Explicitly enable pointer events for nested grid */
  position: relative;    /* Create stacking context for fill-handle */
  z-index: 1;            /* Ensure nested grid is above parent */
}

.spurious-bands-ag-cell :deep(.ag-root) {
  border: 1px solid var(--ag-border-color);
  border-radius: 4px;
  pointer-events: auto;
}

.spurious-bands-ag-cell :deep(.ag-fill-handle) {
  width: 10px;
  height: 10px;
  border-radius: 1px;
  background: #22d3ee;
  border: 2px solid #0891b2;
  cursor: crosshair;
  pointer-events: auto !important; /* Ensure fill-handle receives drag events */
  z-index: 9999;                   /* Bring fill-handle to top */
}

.spurious-bands-ag-cell :deep(.ag-range-handle) {
  width: 10px;
  height: 10px;
  border-radius: 1px;
  background: #22d3ee;
  border: 2px solid #0891b2;
  cursor: move;
}

.spurious-bands-ag-cell :deep(.ag-cell-range-selected) {
  background-color: rgba(34, 211, 238, 0.15);
}

.spurious-bands-ag-cell :deep(.ag-cell-range-handle) {
  background-color: #22d3ee;
}
</style>
