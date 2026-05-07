<template>
  <div class="tx-card card">
    <!-- Header row -->
    <div class="tx-card-header">
      <span class="tx-card-title">
        {{ isNew ? 'New Transmitter' : transmitter.name }}
      </span>
      <div class="tx-card-actions">
        <Button
          v-if="!editMode"
          icon="pi pi-pencil"
          label="Edit"
          size="small"
          severity="secondary"
          @click="startEdit"
        />
        <Button
          v-if="editMode"
          icon="pi pi-check"
          label="Save"
          size="small"
          @click="handleSave"
        />
        <Button
          icon="pi pi-trash"
          label="Delete"
          size="small"
          severity="danger"
          :disabled="isNew"
          @click="handleDelete"
        />
      </div>
    </div>

    <!-- Row 1: Name, Code, and Modulation Type on a single row -->
    <div class="flex gap-4 mt-2">
      <div class="flex-1">
        <label class="field-label">Name</label>
        <InputText
          v-model="form.name"
          placeholder="e.g. C Transmitter 1"
          :disabled="!editMode"
          :class="{ 'p-invalid': errors.name }"
          class="w-full"
        />
        <small class="p-error">{{ errors.name || '&nbsp;' }}</small>
      </div>

      <div class="flex-1">
        <label class="field-label">Code</label>
        <InputText
          v-model="form.code"
          placeholder="e.g. CTX1"
          :disabled="!editMode || !isNew"
          :class="{ 'p-invalid': errors.code }"
          class="w-full"
        />
        <small class="p-error">{{ errors.code || '&nbsp;' }}</small>
      </div>

      <div v-if="form.name.trim() && form.code.trim()" class="flex-1">
        <label class="field-label">Modulation Type</label>
        <Select
          v-model="form.modulation_type"
          :options="modulationOptions"
          placeholder="Select modulation"
          :disabled="!editMode"
          :class="{ 'p-invalid': errors.modulation }"
          class="w-full"
        />
        <small class="p-error">{{ errors.modulation || '&nbsp;' }}</small>
      </div>
    </div>

    <!-- Modulation-specific form (only shown when dropdown is visible) -->
    <div v-if="form.name.trim() && form.code.trim() && form.modulation_type === 'PSK_PM'" class="mt-2">
      <TracsNovaModulationFormsPskPmForm
        ref="modulationFormRef"
        :data="form.modulation_details"
        :is-editable="editMode"
        :code="form.code"
      />
    </div>
    <div v-else-if="form.name.trim() && form.code.trim() && form.modulation_type && form.modulation_type !== ''" class="mt-2">
      <InlineMessage severity="warn">Form for {{ form.modulation_type }} is not yet available.</InlineMessage>
    </div>
  </div>
</template>

<script lang="ts" setup>
import * as yup from 'yup';
import { useTransmitterStore } from '@/stores/tracsNova/transmitter';
import { useTransmitterApi } from '@/composables/tracsNova/useTransmitterApi';
import type { Transmitter, PskPmDetails } from '@/composables/tracsNova/useTransmitterApi';

// ── Props / Emits ─────────────────────────────────────────────────────────────

const props = defineProps<{
  transmitter?: Transmitter;
}>();

const emit = defineEmits<{
  (e: 'saved'): void;
  (e: 'deleted'): void;
}>();

// ── State ─────────────────────────────────────────────────────────────────────

const store = useTransmitterStore();
const api = useTransmitterApi();

const isNew = computed(() => !props.transmitter);
const editMode = ref(isNew.value);

const defaultDetails: PskPmDetails = {
  ports: [['EV'], ['AEV'], ['GLOBAL']],
  sub_carriers: [[32], [128]],
  frequencies: [['DF', ''], ['F1', ''], ['F2', '']],
};

const form = reactive({
  name: props.transmitter?.name ?? '',
  code: props.transmitter?.code ?? '',
  modulation_type: props.transmitter?.modulation_type ?? '',
  modulation_details: props.transmitter?.modulation_details
    ? JSON.parse(JSON.stringify(props.transmitter.modulation_details))
    : defaultDetails,
});

const errors = reactive({ name: '', code: '', modulation: '' });

const modulationOptions = ref<string[]>([]);
const modulationFormRef = ref<{ getData: () => PskPmDetails } | null>(null);

// ── Validation schemas ────────────────────────────────────────────────────────

const nameSchema = yup.string().matches(/^[a-zA-Z][a-zA-Z0-9 ]{4,}[a-zA-Z0-9]+$/, 'Min 6 chars, start with letter');
const codeSchema = yup.string().matches(/^[a-zA-Z][a-zA-Z0-9]{2,}$/, 'Min 3 chars, no spaces');

// ── Load modulation types ─────────────────────────────────────────────────────

onMounted(async () => {
  const res = await api.getModulationTypes();
  if (!res.error.value && res.data.value) {
    modulationOptions.value = res.data.value as string[];
  }
});

// ── Actions ───────────────────────────────────────────────────────────────────

function startEdit() {
  editMode.value = true;
}

async function handleSave() {
  // Clear previous errors
  errors.name = '';
  errors.code = '';
  errors.modulation = '';

  // Validate name
  const nameValid = await nameSchema.isValid(form.name);
  if (!nameValid) { errors.name = 'Min 6 chars, must start with a letter'; return; }

  // Validate code
  const codeValid = await codeSchema.isValid(form.code);
  if (!codeValid) { errors.code = 'Min 3 chars, no spaces allowed'; return; }

  // Validate modulation
  if (!form.modulation_type) { errors.modulation = 'Please select a modulation type'; return; }

  // Duplicate checks (skip own code when editing)
  const ownCode = isNew.value ? undefined : props.transmitter?.code;
  if (store.nameExists(form.name, ownCode)) { errors.name = 'Name already exists'; return; }
  if (isNew.value && store.codeExists(form.code)) { errors.code = 'Code already exists'; return; }

  // Gather modulation details from child form
  const modulationDetails = modulationFormRef.value?.getData() ?? form.modulation_details;

  const success = await store.save({
    name: form.name,
    code: form.code,
    modulation_type: form.modulation_type,
    modulation_details: modulationDetails,
  });

  if (success) {
    editMode.value = false;
    emit('saved');
  } else {
    errors.name = 'Failed to save. Please try again.';
  }
}

async function handleDelete() {
  if (isNew.value) return;
  const tx = props.transmitter!;
  const label = tx.name ? `${tx.name} (${tx.code})` : tx.code;
  const { confirmCriticalDelete } = useConfirmation();
  confirmCriticalDelete('Transmitter', label, async () => {
    const success = await store.remove(tx.code);
    if (success) emit('deleted');
  });
}
</script>

<style scoped>
.tx-card {
  background: #0f2040;
  border: 1px solid #1e3a5f;
  border-radius: 8px;
  padding: 1rem 1.25rem 1.25rem;
}

.tx-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.tx-card-title {
  font-size: 1.15rem;
  font-weight: 600;
  color: #22d3ee;
}

.tx-card-actions {
  display: flex;
  gap: 0.5rem;
}

.field-label {
  display: block;
  font-size: 0.875rem;
  color: #94a3b8;
  margin-bottom: 0.35rem;
}
</style>
