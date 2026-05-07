<template>
  <div class="spurious-fbt-cell">
    <div class="fbt-preview">
      <div class="fbt-preview-text">
        {{ compactPreview }}
      </div>
    </div>

    <Dialog
      v-model:visible="showDialog"
      modal
      :header="dialogTitle"
      :style="{ width: '620px' }"
      :dismissableMask="true"
    >
      <div class="editor-wrap">
        <HotTable :settings="hotSettings" :data="editingData" />
      </div>

      <template #footer>
        <Button label="Cancel" icon="pi pi-times" text @click="closeEditor" />
        <Button label="Save" icon="pi pi-check" @click="saveChanges" />
      </template>
    </Dialog>
  </div>
</template>

<script lang="ts" setup>
import { HotTable } from '@handsontable/vue3';
import { registerAllModules } from 'handsontable/registry';
import 'handsontable/styles/handsontable.css';
import 'handsontable/styles/ht-theme-main-no-icons.css';
import Dialog from 'primevue/dialog';
import Button from 'primevue/button';

registerAllModules();

type FbtMatrix = (string | number)[][];

const HEADER_LABELS = new Set(['offset (khz)', 'value (dbc)', 'value (dbc)', 'offset', 'value']);

const props = defineProps<{ params: any }>();

const ensureMatrix = (value: unknown): FbtMatrix => {
  if (Array.isArray(value) && value.length > 0 && value.every((row) => Array.isArray(row))) {
    return value as FbtMatrix;
  }
  return [['', '']];
};

const tableData = ref<FbtMatrix>(ensureMatrix(props.params?.value));
const editingData = ref<FbtMatrix>([]);
const showDialog = ref(false);

const isEditable = computed(() => props.params?.isEditable !== false);

watch(
  () => props.params?.value,
  (val) => {
    tableData.value = ensureMatrix(val);
  },
  { deep: true },
);

const dialogTitle = computed(() => {
  const field = props.params?.colDef?.field;
  const displayName = field === 'fbt' ? 'FBT' : field === 'fbt_hot' ? 'FBT Hot' : 'FBT Cold';

  const row = (props.params?.data ?? {}) as Record<string, unknown>;
  const code = String(row.code ?? '').trim();
  const port = String(row.port ?? '').trim();
  const frequencyLabel = String(row.frequency_label ?? row.frequency ?? '').trim();

  const rowIdentity = [code, port, frequencyLabel]
    .filter((v) => v.length > 0)
    .join('_');

  if (rowIdentity.length > 0) {
    return `Edit ${displayName} - ${rowIdentity}`;
  }

  return `Edit ${displayName} - Row ${(props.params?.node?.rowIndex ?? 0) + 1}`;
});

function isMeaningfulCell(cell: string | number | null | undefined): boolean {
  if (cell === '' || cell === null || cell === undefined) return false;
  const normalized = String(cell)
    .trim()
    .toLowerCase()
    .replace(/\s+/g, ' ')
    .replace(/[()]/g, '');
  return !HEADER_LABELS.has(normalized);
}

const compactPreview = computed(() => {
  const segments = tableData.value
    .map((row) => {
      if (!Array.isArray(row)) return '';
      const pair = row
        .slice(0, 2)
        .filter(cell => isMeaningfulCell(cell))
        .map(cell => formatValue(cell))
        .filter(text => text !== '');
      return pair.join(', ');
    })
    .filter(segment => segment !== '');

  return segments.length > 0 ? segments.join('; ') : '';
});

function formatValue(val: string | number | undefined): string {
  if (!isMeaningfulCell(val)) return '';
  if (typeof val === 'number') return val.toFixed(2);
  return String(val);
}

function openEditor() {
  if (!isEditable.value) return;
  editingData.value = JSON.parse(JSON.stringify(tableData.value));
  showDialog.value = true;
}

function closeEditor() {
  showDialog.value = false;
}

async function saveChanges() {
  const cleanedData = editingData.value.filter((row) =>
    row.some((cell) => cell !== '' && cell !== null && cell !== undefined),
  );
  const finalData = cleanedData.length > 0 ? cleanedData : [['', '']];

  const field = props.params?.colDef?.field;
  if (field && props.params?.node) {
    props.params.node.setDataValue(field, finalData);
    tableData.value = finalData;

    await nextTick();

    setTimeout(() => {
      props.params?.api?.resetRowHeights();
      props.params?.api?.onRowHeightChanged?.();
      props.params?.api?.refreshCells?.({ force: true });
    }, 0);
  }

  showDialog.value = false;
}

defineExpose({
  openEditor,
});

const hotSettings = computed(() => ({
  licenseKey: 'non-commercial-and-evaluation',
  colHeaders: ['Offset (kHz)', 'Value (dBc)'],
  columns: [
    {
      type: 'numeric',
      locale: 'en-US',
      numericFormat: {
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
        useGrouping: true,
      },
    },
    {
      type: 'numeric',
      locale: 'en-US',
      numericFormat: {
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
        useGrouping: true,
      },
    },
  ],
  rowHeaders: true,
  stretchH: 'all',
  width: '100%',
  minRows: 1,
  minSpareRows: 1,
  contextMenu: true,
  height: 380,
  autoWrapRow: true,
  autoWrapCol: true,
  copyPaste: true,
  fillHandle: {
    direction: 'vertical',
    autoInsertRow: true,
  },
  enterMoves: { row: 1, col: 0 },
  tabMoves: { row: 0, col: 1 },
}));
</script>

<style>
.spurious-fbt-cell {
  width: 100% !important;
  height: 100% !important;
  padding: 2px 0 !important;
  pointer-events: auto;
}

.fbt-preview {
  width: 100% !important;
  min-height: 1.1rem;
  padding: 0 !important;
  pointer-events: auto;
  cursor: pointer;
}

.fbt-preview-text {
  color: var(--text-color) !important;
  font-size: 0.76rem !important;
  line-height: 1.35 !important;
  white-space: normal !important;
  overflow-wrap: anywhere;
  font-variant-numeric: tabular-nums;
  min-height: 1.1rem;
}

.editor-wrap {
  margin: 0.75rem 0;
}

.handsontable.htContextMenu {
  background: #111827 !important;
  border: 1px solid #334155 !important;
  color: #e2e8f0 !important;
  z-index: 10000 !important;
}

.handsontable.htContextMenu table tbody tr td {
  background: #111827 !important;
  color: #e2e8f0 !important;
}

.handsontable.htContextMenu table tbody tr:hover td,
.handsontable.htContextMenu table tbody tr.current td {
  background: #1e293b !important;
  color: #f8fafc !important;
}
</style>
