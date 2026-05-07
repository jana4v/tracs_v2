<template>
  <div class="rx-card card">
    <div class="rx-card-header">
      <span class="rx-card-title">
        {{ isNew ? 'New Receiver' : receiver.name }}
      </span>
      <div class="rx-card-actions">
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

    <div class="flex gap-4 mt-2">
      <div class="flex-1">
        <label class="field-label">Name</label>
        <InputText
          v-model="form.name"
          placeholder="e.g. C Receiver 1"
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
          placeholder="e.g. CRX1"
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

    <div v-if="form.name.trim() && form.code.trim() && form.modulation_type === 'PSK_FM'" class="mt-2">
      <TracsNovaModulationFormsPskFmForm
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
import { useReceiverStore } from '@/stores/tracsNova/receiver';
import type { Receiver, PskPmDetails } from '@/composables/tracsNova/useTransmitterApi';

const props = defineProps<{
  receiver?: Receiver;
}>();

const emit = defineEmits<{
  (e: 'saved'): void;
  (e: 'deleted'): void;
}>();

const store = useReceiverStore();

const isNew = computed(() => !props.receiver);
const editMode = ref(isNew.value);

const defaultDetails: PskPmDetails = {
  ports: [['EV'], ['AEV']],
  sub_carriers: [],
  frequencies: [['DF', '']],
};

const form = reactive({
  name: props.receiver?.name ?? '',
  code: props.receiver?.code ?? '',
  modulation_type: props.receiver?.modulation_type ?? 'PSK_FM',
  modulation_details: props.receiver?.modulation_details
    ? JSON.parse(JSON.stringify(props.receiver.modulation_details))
    : defaultDetails,
});

const errors = reactive({ name: '', code: '', modulation: '' });

const modulationOptions = ref<string[]>(['PSK_FM']);
const modulationFormRef = ref<{ getData: () => PskPmDetails } | null>(null);
const toast = useToast();

const nameSchema = yup.string().matches(/^[a-zA-Z][a-zA-Z0-9 ]{4,}[a-zA-Z0-9]+$/, 'Min 6 chars, start with letter');
const codeSchema = yup.string().matches(/^[a-zA-Z][a-zA-Z0-9]{2,}$/, 'Min 3 chars, no spaces');

function startEdit() {
  editMode.value = true;
}

async function handleSave() {
  errors.name = '';
  errors.code = '';
  errors.modulation = '';

  const nameValid = await nameSchema.isValid(form.name);
  if (!nameValid) { errors.name = 'Min 6 chars, must start with a letter'; return; }

  const codeValid = await codeSchema.isValid(form.code);
  if (!codeValid) { errors.code = 'Min 3 chars, no spaces allowed'; return; }

  if (!form.modulation_type) { errors.modulation = 'Please select a modulation type'; return; }

  const ownCode = isNew.value ? undefined : props.receiver?.code;
  if (store.nameExists(form.name, ownCode)) { errors.name = 'Name already exists'; return; }
  if (isNew.value && store.codeExists(form.code)) { errors.code = 'Code already exists'; return; }

  const modulationDetails = modulationFormRef.value?.getData() ?? form.modulation_details;

  const success = await store.save({
    name: form.name,
    code: form.code,
    modulation_type: form.modulation_type,
    modulation_details: modulationDetails,
    system_type: 'Receiver',
  });

  if (success) {
    editMode.value = false;
    emit('saved');
  } else {
    errors.name = 'Failed to save. Please try again.';
    toast.add({
      severity: 'error',
      summary: 'Save Failed',
      detail: 'Receiver payload contains invalid or incomplete values.',
      life: 3500,
    });
  }
}

async function handleDelete() {
  if (isNew.value) return;
  const rx = props.receiver!;
  const label = rx.name ? `${rx.name} (${rx.code})` : rx.code;
  const { confirmCriticalDelete } = useConfirmation();
  confirmCriticalDelete('Receiver', label, async () => {
    const success = await store.remove(rx.code);
    if (success) emit('deleted');
  });
}
</script>

<style scoped>
.rx-card {
  background: #0f2040;
  border: 1px solid #1e3a5f;
  border-radius: 8px;
  padding: 1rem 1.25rem 1.25rem;
}

.rx-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.rx-card-title {
  font-size: 1.15rem;
  font-weight: 600;
  color: #22d3ee;
}

.rx-card-actions {
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
