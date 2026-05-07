<template>
  <div class="cal-id-selector">
    <label class="selector-label">Cal ID</label>
    <AutoComplete
      v-model="localCalId"
      :suggestions="filteredOptions"
      placeholder="Type or select Cal ID…"
      :loading="loading"
      dropdown
      class="cal-id-autocomplete"
      input-class="cal-id-ac-input"
      @complete="onComplete"
    />

    <label class="selector-label cal-type-label">Cal Type</label>
    <Select
      v-model="localCalType"
      :options="calTypeOptions"
      option-label="label"
      option-value="value"
      class="cal-type-select"
      placeholder="Select Cal Type"
    />

    <div v-if="showSpuriousBandsCheckbox" class="spurious-bands-container">
      <InputSwitch
        v-model="localIncludeSpuriousBands"
        input-id="include-spurious-bands"
        class="spurious-bands-switch"
      />
      <label for="include-spurious-bands" class="spurious-bands-label">Include Spurious Bands</label>
      <Button
        label="Generate Report"
        size="small"
        severity="secondary"
        outlined
        :disabled="!canGenerateReport"
        @click="emit('generate-report')"
      />
    </div>

  </div>
</template>

<script lang="ts" setup>
import { useCalibrationDataApi } from '@/composables/tracsNova/useCalibrationDataApi';
import { useCalibrationRunApi } from '@/composables/tracsNova/useCalibrationRunApi';
import type { CalIdsResponse } from '@/composables/tracsNova/useCalibrationDataApi';
import type { CalibrationRunSnapshot } from '@/composables/tracsNova/useCalibrationRunApi';

const props = defineProps<{
  modelValue: string;
  calType: string;
  includeSpuriousBands?: boolean;
  isRunning?: boolean;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void;
  (e: 'update:calType', value: string): void;
  (e: 'update:includeSpuriousBands', value: boolean): void;
  (e: 'generate-report'): void;
}>();

const { getCalIds } = useCalibrationDataApi();
const { getLatestRun } = useCalibrationRunApi();

const calIdOptions = ref<string[]>([]);
const filteredOptions = ref<string[]>([]);
const loading = ref(false);
const calTypeOptions = [
  { label: 'Uplink', value: 'uplink' },
  { label: 'Downlink', value: 'downlink' },
  { label: 'TVAC Ref', value: 'tvac_ref' },
  { label: 'Fixed Pad', value: 'fixed_pad' },
  { label: 'Cal SG', value: 'cal_sg' },
  { label: 'Inject Cal', value: 'inject_cal' },
];

// Two-way binding
const localCalId = computed({
  get: () => props.modelValue,
  set: (val: string | null) => emit('update:modelValue', val ?? ''),
});

const localCalType = computed({
  get: () => props.calType,
  set: (val: string | null) => emit('update:calType', val ?? 'uplink'),
});

const localIncludeSpuriousBands = computed({
  get: () => props.includeSpuriousBands ?? true,
  set: (val: boolean) => emit('update:includeSpuriousBands', val),
});

const showSpuriousBandsCheckbox = computed(() => {
  const showFor = ['downlink', 'cal_sg', 'inject_cal', 'tvac_ref'];
  return showFor.includes(props.calType);
});

const canGenerateReport = computed(() => {
  return ['cal_sg', 'inject_cal', 'downlink'].includes(props.calType) && props.modelValue.trim().length > 0 && !props.isRunning;
});

function onComplete(event: { query: string }) {
  const q = event.query.trim().toLowerCase();
  filteredOptions.value = q
    ? calIdOptions.value.filter((id) => id.toLowerCase().includes(q))
    : [...calIdOptions.value];
}

onMounted(async () => {
  await loadCalIds();
  await applyDefaultCalId();
});

watch(
  () => props.calType,
  async () => {
    await loadCalIds();
    await applyDefaultCalId();
  }
);

async function loadCalIds() {
  loading.value = true;
  try {
    const { data, error } = await getCalIds(props.calType);
    if (!error.value) {
      const res = data.value as CalIdsResponse;
      calIdOptions.value = res?.cal_ids ?? [];
      filteredOptions.value = [...calIdOptions.value];
    }
  } finally {
    loading.value = false;
  }
}

async function applyDefaultCalId() {
  if (!['cal_sg', 'inject_cal', 'downlink'].includes(props.calType)) return;
  if (props.modelValue.trim().length > 0) return;

  const latest = await getLatestRun(props.calType);
  if (latest.error.value || !latest.data.value) return;

  const snapshot = latest.data.value as CalibrationRunSnapshot;
  const latestCalId = String(snapshot?.cal_id ?? '').trim();
  if (!latestCalId) return;

  emit('update:modelValue', latestCalId);
}
</script>

<style scoped>
.cal-id-selector {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1.25rem;
  margin-top: 0.75rem;
  background: #0d1b2e;
  border-bottom: 1px solid #1e3050;
  flex-shrink: 0;
}

.selector-label {
  color: #94a3b8;
  font-size: 0.85rem;
  font-weight: 600;
  white-space: nowrap;
  min-width: 4.5rem;
}

:deep(.cal-id-autocomplete) {
  width: 300px;
}

:deep(.cal-id-ac-input) {
  background: #071120 !important;
  border-color: #1e3050 !important;
  color: #e2e8f0 !important;
  font-size: 0.9rem;
  width: 100%;
}

:deep(.cal-id-ac-input:focus) {
  border-color: #22d3ee !important;
  box-shadow: 0 0 0 1px #22d3ee40 !important;
}

.cal-type-label {
  margin-left: 0.25rem;
}

:deep(.cal-type-select) {
  width: 180px;
}

:deep(.cal-type-select .p-select-label) {
  color: #e2e8f0 !important;
}

:deep(.cal-type-select .p-select-dropdown) {
  color: #94a3b8 !important;
}

.spurious-bands-container {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.spurious-bands-label {
  color: #94a3b8;
  font-size: 0.85rem;
  cursor: pointer;
  white-space: nowrap;
}

:deep(.spurious-bands-switch .p-inputswitch) {
  height: 1.5rem;
  width: 2.5rem;
}

:deep(.spurious-bands-switch .p-inputswitch.p-inputswitch-checked) {
  background-color: #22d3ee !important;
}

/* ── Light theme overrides ──────────────────────────────────────────────── */
html:not(.dark) .cal-id-selector {
  background: var(--p-surface-0);
  border-bottom-color: var(--p-content-border-color);
}
html:not(.dark) .selector-label {
  color: var(--p-text-muted-color);
}
html:not(.dark) :deep(.cal-id-ac-input) {
  background: var(--p-surface-0) !important;
  border-color: var(--p-form-field-border-color) !important;
  color: var(--p-text-color) !important;
}
html:not(.dark) :deep(.cal-type-select .p-select-label) {
  color: var(--p-text-color) !important;
}

</style>
