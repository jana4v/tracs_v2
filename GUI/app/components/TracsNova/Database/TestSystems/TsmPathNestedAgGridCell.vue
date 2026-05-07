<template>
  <div class="nested-path-cell" @mousedown.stop @click.stop>
    <div class="nested-toolbar">
      <button class="mini-btn" type="button" title="Add Row" @click="addRow">+</button>
      <button
        class="mini-btn"
        type="button"
        title="Delete Selected"
        :disabled="selectedCount === 0"
        @click="deleteSelected"
      >
        -
      </button>
    </div>

    <ag-grid-vue
      class="nested-grid"
      :theme="isDark
        ? themeQuartz.withPart(colorSchemeDarkBlue)
        : themeQuartz.withPart(colorSchemeLightCold)"
      style="width: 100%; height: calc(100% - 2rem);"
      :columnDefs="columnDefs"
      :rowData="rows"
      :defaultColDef="defaultColDef"
      :rowSelection="rowSelection"
      :suppressContextMenu="false"
      :suppressMovableColumns="true"
      :undoRedoCellEditing="true"
      :undoRedoCellEditingLimit="20"
      @grid-ready="onGridReady"
      @cell-value-changed="onCellValueChanged"
      @selection-changed="onSelectionChanged"
    />
  </div>
</template>

<script lang="ts" setup>
import type { ColDef, GridApi, GridReadyEvent } from 'ag-grid-community';
import {
  colorSchemeDarkBlue,
  colorSchemeLightCold,
  themeQuartz,
} from 'ag-grid-community';
import { AgGridVue } from 'ag-grid-vue3';

type PathRow = { path: string };

const props = defineProps<{ params: any }>();

const isDark = useDark();
const gridApi = shallowRef<GridApi | null>(null);
const selectedCount = ref(0);

function normalizeRows(value: unknown): PathRow[] {
  if (!Array.isArray(value) || value.length === 0) {
    return [{ path: '' }, { path: '' }, { path: '' }, { path: '' }];
  }
  const mapped = value
    .map((row: any) => ({ path: String(row?.path ?? '') }))
    .slice(0, 50);
  return mapped.length > 0 ? mapped : [{ path: '' }, { path: '' }, { path: '' }, { path: '' }];
}

const rows = ref<PathRow[]>(normalizeRows(props.params?.value));

const defaultColDef: ColDef = {
  editable: true,
  resizable: true,
  sortable: false,
  filter: false,
  minWidth: 120,
};

const rowSelection = {
  mode: 'multiRow' as const,
  checkboxes: true,
  headerCheckbox: false,
};

const columnDefs: ColDef[] = [
  {
    field: 'path',
    headerName: 'Path',
    flex: 1,
  },
];

function updateParentValue() {
  props.params?.node?.setDataValue?.('path', rows.value);

  const rowCount = Math.max(rows.value.length, 4);
  const desiredHeight = Math.min(122 + rowCount * 32, 420);
  props.params?.node?.setRowHeight?.(desiredHeight);
  setTimeout(() => props.params?.api?.onRowHeightChanged?.(), 0);
}

function onGridReady(event: GridReadyEvent) {
  gridApi.value = event.api;
  event.api.sizeColumnsToFit();
  updateParentValue();
}

function onSelectionChanged() {
  selectedCount.value = gridApi.value?.getSelectedRows()?.length ?? 0;
}

function onCellValueChanged() {
  const next: PathRow[] = [];
  gridApi.value?.forEachNode((n) => {
    if (n.data) next.push({ path: String(n.data.path ?? '') });
  });
  rows.value = next;
  updateParentValue();
}

function addRow() {
  rows.value = [...rows.value, { path: '' }];
  gridApi.value?.setGridOption('rowData', rows.value);
  updateParentValue();
}

function deleteSelected() {
  const selected = gridApi.value?.getSelectedRows() ?? [];
  if (selected.length === 0) return;
  rows.value = rows.value.filter((r) => !selected.includes(r));
  if (rows.value.length === 0) rows.value = [{ path: '' }, { path: '' }, { path: '' }, { path: '' }];
  gridApi.value?.setGridOption('rowData', rows.value);
  selectedCount.value = 0;
  updateParentValue();
}
</script>

<style scoped>
.nested-path-cell {
  width: 100%;
  height: 100%;
  min-height: 120px;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.nested-toolbar {
  display: flex;
  justify-content: flex-end;
  gap: 0.25rem;
}

.mini-btn {
  width: 1.5rem;
  height: 1.5rem;
  border: 1px solid #334155;
  background: #0f172a;
  color: #cbd5e1;
  border-radius: 4px;
  cursor: pointer;
  line-height: 1;
}

.mini-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.nested-grid {
  flex: 1;
  min-height: 0;
}
</style>
