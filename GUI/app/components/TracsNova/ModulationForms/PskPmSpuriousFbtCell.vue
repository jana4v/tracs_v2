<template>
  <div class="spurious-fbt-cell" :class="isDark ? 'ht-theme-main-dark' : 'ht-theme-main'">
    <HotTable :settings="hotSettings" :data="tableData" />
  </div>
</template>

<script lang="ts" setup>
import { HotTable } from '@handsontable/vue3';
import { registerAllModules } from 'handsontable/registry';
import 'handsontable/styles/handsontable.css';
import 'handsontable/styles/ht-theme-main-no-icons.css';

registerAllModules();

type FbtMatrix = (string | number)[][];

const props = defineProps<{ params: any }>();
const isDark = useDark();

const ensureMatrix = (value: unknown): FbtMatrix => {
  if (Array.isArray(value) && value.length > 0 && value.every(row => Array.isArray(row))) {
    return value as FbtMatrix;
  }
  return [['', '']];
};

const tableData = ref<FbtMatrix>(ensureMatrix(props.params?.value));

if (props.params?.value !== tableData.value) {
  props.params?.node?.setDataValue(props.params?.colDef?.field, tableData.value);
}

watch(
  () => props.params?.value,
  (val) => {
    tableData.value = ensureMatrix(val);
  },
  { deep: true },
);

const hotSettings = computed(() => ({
  licenseKey: 'non-commercial-and-evaluation',
  colHeaders: ['Offset (kHz)', 'Value (dBc)'],
  columns: [{ type: 'text' }, { type: 'text' }],
  rowHeaders: false,
  stretchH: 'all',
  width: '100%',
  minRows: 1,
  readOnly: !props.params?.isEditable,
  contextMenu: props.params?.isEditable ? ['row_above', 'row_below', 'remove_row'] : false,
  height: Math.max(84, (tableData.value.length * 28) + 34),
  afterChange: (changes: any, source: string) => {
    if (!changes || source === 'loadData') return;
  },
  afterCreateRow: () => props.params?.api?.resetRowHeights(),
  afterRemoveRow: () => props.params?.api?.resetRowHeights(),
}));
</script>

<style scoped>
.spurious-fbt-cell {
  width: 100%;
  min-width: 240px;
  padding: 0.35rem 0;
}
</style>

<style>
/* Handsontable context menu is rendered outside component scope, so use global selectors. */
.handsontable.htContextMenu {
  background: #111827 !important;
  border: 1px solid #334155 !important;
  color: #e2e8f0 !important;
}

.handsontable.htContextMenu table tbody tr td {
  background: #111827 !important;
  color: #e2e8f0 !important;
}

.handsontable.htContextMenu table tbody tr td .htItemWrapper {
  color: inherit !important;
}

.handsontable.htContextMenu table tbody tr:hover td,
.handsontable.htContextMenu table tbody tr.current td {
  background: #1e293b !important;
  color: #f8fafc !important;
}

.handsontable.htContextMenu table tbody tr td.htDimmed,
.handsontable.htContextMenu table tbody tr td.htDisabled {
  background: #1f2937 !important;
  color: #94a3b8 !important;
}

.handsontable.htContextMenu table tbody tr:hover td.htDimmed,
.handsontable.htContextMenu table tbody tr:hover td.htDisabled,
.handsontable.htContextMenu table tbody tr.current td.htDimmed,
.handsontable.htContextMenu table tbody tr.current td.htDisabled {
  background: #374151 !important;
  color: #cbd5e1 !important;
}

.handsontable.htContextMenu .htSeparator td {
  border-top-color: #334155 !important;
}
</style>
