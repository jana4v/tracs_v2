<template>
  <div class="spurious-fbt-json-cell">
    <div class="fbt-data-table" @dblclick="openEditor" :class="{ editable: isEditable }">
      <table>
        <thead>
          <tr>
            <th>Offset (kHz)</th>
            <th>Value (dBc)</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(row, idx) in displayRows" :key="idx">
            <td>{{ formatValue(row[0]) }}</td>
            <td>{{ formatValue(row[1]) }}</td>
          </tr>
        </tbody>
      </table>
      <div class="edit-overlay" v-if="isEditable">
        <i class="pi pi-pencil"></i>
      </div>
    </div>
    
    <Dialog 
      v-model:visible="showDialog" 
      modal 
      :header="dialogTitle"
      :style="{ width: '600px' }"
      :dismissableMask="true"
    >
      <div class="fbt-editor-content">
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

const props = defineProps<{ params: any }>();
const isDark = useDark();

const ensureMatrix = (value: unknown): FbtMatrix => {
  if (Array.isArray(value) && value.length > 0 && value.every(row => Array.isArray(row))) {
    return value as FbtMatrix;
  }
  return [['', '']];
};

const currentValue = ref<FbtMatrix>(ensureMatrix(props.params?.value));
const editingData = ref<FbtMatrix>([]);
const showDialog = ref(false);

const isEditable = computed(() => props.params?.isEditable ?? false);

const displayRows = computed(() => {
  const rows = currentValue.value;
  // Show max 3 rows in display, full data in editor
  return rows.slice(0, 3);
});

function formatValue(val: string | number | undefined): string {
  if (val === '' || val === null || val === undefined) return '-';
  if (typeof val === 'number') return val.toFixed(2);
  return String(val);
}

const dialogTitle = computed(() => {
  const field = props.params?.colDef?.field;
  const displayName = field === 'fbt' ? 'FBT' : field === 'fbt_hot' ? 'FBT Hot' : 'FBT Cold';
  return `Edit ${displayName} - Row ${props.params?.node?.rowIndex + 1 || ''}`;
});

watch(
  () => props.params?.value,
  (val) => {
    currentValue.value = ensureMatrix(val);
  },
  { deep: true },
);

function openEditor() {
  if (!isEditable.value) return;
  // Deep clone current value for editing
  editingData.value = JSON.parse(JSON.stringify(currentValue.value));
  showDialog.value = true;
}

function closeEditor() {
  showDialog.value = false;
}

async function saveChanges() {
  // Filter out completely empty rows
  const cleanedData = editingData.value.filter(row => 
    row.some(cell => cell !== '' && cell !== null && cell !== undefined)
  );
  
  // Ensure at least one row
  const finalData = cleanedData.length > 0 ? cleanedData : [['', '']];
  
  const field = props.params?.colDef?.field;
  if (field && props.params?.node) {
    // Update AG Grid data first
    props.params.node.setDataValue(field, finalData);
    
    // Update local state
    currentValue.value = finalData;
    
    // Wait for Vue reactivity and AG Grid to update
    await nextTick();
    
    // Force row height recalculation with delay for AG Grid internal updates
    setTimeout(() => {
      if (props.params?.api) {
        props.params.api.resetRowHeights();
        props.params.api.refreshCells({ force: true });
      }
    }, 50);
  }
  
  showDialog.value = false;
}

const hotSettings = computed(() => ({
  licenseKey: 'non-commercial-and-evaluation',
  colHeaders: ['Offset (kHz)', 'Value (dBc)'],
  columns: [
    { type: 'numeric', numericFormat: { pattern: '0,0.00' } },
    { type: 'numeric', numericFormat: { pattern: '0,0.00' } }
  ],
  rowHeaders: true,
  stretchH: 'all',
  width: '100%',
  minRows: 1,
  minSpareRows: 1,
  contextMenu: true,
  height: 400,
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

<style scoped>
.spurious-fbt-json-cell {
  width: 100%;
  height: 100%;
  padding: 0;
}

.fbt-data-table {
  position: relative;
  width: 100%;
  height: 100%;
  max-height: 200px;
  overflow-y: auto;
  overflow-x: hidden;
  border: 1px solid var(--surface-300);
  border-radius: 4px;
  transition: all 0.2s;
}

.fbt-data-table.editable {
  cursor: pointer;
}

.fbt-data-table.editable:hover {
  border-color: var(--primary-color);
  background: var(--surface-50);
}

.fbt-data-table table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.8rem;
}

.fbt-data-table thead th {
  background: var(--surface-100);
  padding: 0.25rem 0.5rem;
  text-align: left;
  font-weight: 600;
  font-size: 0.75rem;
  border-bottom: 1px solid var(--surface-300);
  color: var(--text-color-secondary);
}

.fbt-data-table tbody td {
  padding: 0.25rem 0.5rem;
  border-bottom: 1px solid var(--surface-200);
  color: var(--text-color);
}

.fbt-data-table tbody tr:last-child td {
  border-bottom: none;
}

.edit-overlay {
  position: absolute;
  top: 2px;
  right: 2px;
  background: var(--primary-color);
  color: white;
  padding: 0.25rem;
  border-radius: 3px;
  font-size: 0.7rem;
  opacity: 0;
  transition: opacity 0.2s;
  pointer-events: none;
}

.fbt-data-table.editable:hover .edit-overlay {
  opacity: 1;
}

.fbt-editor-content {
  margin: 1rem 0;
}

/* Dark mode adjustments for Handsontable in dialog */
:deep(.ht_master .wtHolder) {
  background: var(--surface-0);
}
</style>

<style>
/* Handsontable context menu dark mode (global scope) */
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
.handsontable.htContextMenu table tbody tr.current td.htDimmed {
  background: #374151 !important;
  color: #cbd5e1 !important;
}

.handsontable.htContextMenu .htSeparator td {
  border-top-color: #334155 !important;
}
</style>
