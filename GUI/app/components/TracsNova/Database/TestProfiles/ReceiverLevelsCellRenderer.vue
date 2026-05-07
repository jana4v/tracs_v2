<template>
  <div class="levels-cell" @dblclick="openEditor" :title="displayText">
    <span class="levels-text">{{ displayText }}</span>
    <i v-if="isEditable" class="pi pi-pencil edit-icon" />

    <Dialog
      v-model:visible="showDialog"
      modal
      header="Edit Levels"
      :style="{ width: '480px' }"
      :dismissableMask="true"
    >
      <div class="levels-editor-content">
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

type LevelMatrix = (number | string)[][];

const DEFAULT_LEVELS: LevelMatrix = [
  [-60, 10], [-70, 10], [-80, 10], [-90, 10], [-100, 10], [-105, 10],
];

const props = defineProps<{ params: any }>();

const ensureMatrix = (value: unknown): LevelMatrix => {
  if (Array.isArray(value) && value.length > 0 && value.every((r) => Array.isArray(r))) {
    return value as LevelMatrix;
  }
  return DEFAULT_LEVELS.map((r) => [...r]);
};

const currentValue = ref<LevelMatrix>(ensureMatrix(props.params?.value));
const editingData = ref<LevelMatrix>([]);
const showDialog = ref(false);

const isEditable = computed(() => props.params?.isEditable ?? true);

/** Display text: "-60,10; -70,10; -80,10; ..." */
const displayText = computed(() =>
  currentValue.value.map((r) => `${r[0]},${r[1]}`).join('; '),
);

watch(
  () => props.params?.value,
  (val) => { currentValue.value = ensureMatrix(val); },
  { deep: true },
);

function openEditor() {
  if (!isEditable.value) return;
  editingData.value = JSON.parse(JSON.stringify(currentValue.value));
  showDialog.value = true;
}

function closeEditor() {
  showDialog.value = false;
}

async function saveChanges() {
  const cleaned = editingData.value.filter(
    (r) => r.some((c) => c !== '' && c !== null && c !== undefined),
  );
  const final = cleaned.length > 0 ? cleaned : DEFAULT_LEVELS.map((r) => [...r]);

  const field = props.params?.colDef?.field;
  if (field && props.params?.node) {
    props.params.node.setDataValue(field, final);
    currentValue.value = final;
    await nextTick();
    setTimeout(() => {
      props.params?.api?.resetRowHeights();
      props.params?.api?.refreshCells({ force: true });
    }, 50);
  }
  showDialog.value = false;
}

const hotSettings = computed(() => ({
  licenseKey: 'non-commercial-and-evaluation',
  colHeaders: ['Level (dBm)', 'Number of Commands'],
  columns: [
    { type: 'numeric' },
    { type: 'numeric' },
  ],
  rowHeaders: true,
  stretchH: 'all',
  width: '100%',
  height: 320,
  minRows: 1,
  minSpareRows: 1,
  contextMenu: true,
  fillHandle: { direction: 'vertical', autoInsertRow: true },
  autoWrapRow: true,
  autoWrapCol: true,
  copyPaste: true,
  enterMoves: { row: 1, col: 0 },
  tabMoves: { row: 0, col: 1 },
}));
</script>

<style scoped>
.levels-cell {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  width: 100%;
  height: 100%;
  padding: 0 0.4rem;
  cursor: pointer;
  overflow: hidden;
}

.levels-text {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 0.78rem;
  color: var(--text-color);
}

.edit-icon {
  font-size: 0.65rem;
  color: var(--text-color-secondary);
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.15s;
}

.levels-cell:hover .edit-icon {
  opacity: 1;
}

.levels-editor-content {
  margin: 0.75rem 0;
}
</style>
