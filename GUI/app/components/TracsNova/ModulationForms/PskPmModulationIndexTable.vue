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
import type { ColDef, ColGroupDef, GridApi, GridReadyEvent } from 'ag-grid-community';
import { AgGridVue } from 'ag-grid-vue3';
import type { ModulationIndexRow } from '@/composables/tracsNova/useTransmitterApi';

ModuleRegistry.registerModules([AllEnterpriseModule]);

const props = defineProps<{
  rowData: ModulationIndexRow[];
  tones: (number | string)[];
  isEditable: boolean;
}>();

const isDark = useDark();
const gridApi = shallowRef<GridApi | null>(null);

const columnDefs = computed<(ColDef | ColGroupDef)[]>(() => {
  const baseColumns: ColDef[] = [
    { field: 'code', headerName: 'Code', editable: false, minWidth: 70, flex: 1 },
    { field: 'port', headerName: 'Port', editable: false, minWidth: 70, flex: 1 },
    { field: 'frequency_label', headerName: 'Freq Label', editable: false, minWidth: 90, flex: 1.5 },
    { field: 'frequency', headerName: 'Frequency (MHz)', editable: false, minWidth: 110, flex: 1.5 },
    { field: 'specification', headerName: 'Specification (rad)', editable: props.isEditable, minWidth: 130, flex: 1.5 },
    { field: 'tolerance', headerName: 'Tolerance (± %)', editable: props.isEditable, minWidth: 120, flex: 1.3 },
  ];

  const toneColumns: ColGroupDef[] = props.tones.map((tone) => {
    const toneStr = String(tone).trim();
    return {
      headerName: `${toneStr} kHz Tone`,
      children: [
        { field: `fbt_tone_${toneStr}`, headerName: 'FBT (rad)', editable: props.isEditable, minWidth: 100, flex: 1.1 },
        { field: `fbt_hot_tone_${toneStr}`, headerName: 'FBT Hot (rad)', editable: props.isEditable, minWidth: 120, flex: 1.3 },
        { field: `fbt_cold_tone_${toneStr}`, headerName: 'FBT Cold (rad)', editable: props.isEditable, minWidth: 120, flex: 1.3 },
      ],
    };
  });

  return [...baseColumns, ...toneColumns];
});

const defaultColDef: ColDef = {
  resizable: true,
  sortable: true,
  filter: true,
  enableRowGroup: true,
};

const autoSizeStrategy = { type: 'fitGridWidth' } as const;

const getRowId = (params: any) =>
  `${params.data.port}|${params.data.frequency_label}|${params.data.frequency}`;

function onGridReady(event: GridReadyEvent) {
  gridApi.value = event.api;
  if (props.rowData.length > 0) {
    gridApi.value.applyTransaction({ add: props.rowData });
  }
}

function setRows(newRows: ModulationIndexRow[]) {
  if (!gridApi.value) return;

  const current: ModulationIndexRow[] = [];
  gridApi.value.forEachNode((n) => { if (n.data) current.push(n.data as ModulationIndexRow); });

  const key = (r: ModulationIndexRow) => `${r.port}|${r.frequency_label}|${r.frequency}`;
  const currentMap = new Map(current.map(r => [key(r), r]));
  const newMap = new Map(newRows.map(r => [key(r), r]));

  const toAdd = newRows.filter(r => !currentMap.has(key(r)));
  const toRemove = current.filter(r => !newMap.has(key(r)));

  const toUpdate: ModulationIndexRow[] = [];
  for (const row of newRows) {
    const ex = currentMap.get(key(row));
    if (!ex) continue;

    let changed = false;
    const merged: ModulationIndexRow = { ...ex };

    if (ex.code !== row.code) {
      merged.code = row.code;
      changed = true;
    }

    for (const k of Object.keys(row)) {
      if (!(k in ex)) {
        const val = row[k];
        merged[k] = val === undefined ? null : val;
        changed = true;
      }
    }

    if (changed) toUpdate.push(merged);
  }

  gridApi.value.applyTransaction({ add: toAdd, remove: toRemove, update: toUpdate });
}

function getData(): ModulationIndexRow[] {
  const rows: ModulationIndexRow[] = [];
  gridApi.value?.forEachNode((n) => { if (n.data) rows.push(n.data as ModulationIndexRow); });
  return rows;
}

defineExpose({ getData, setRows });
</script>
