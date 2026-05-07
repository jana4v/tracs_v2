<template>
  <div class="psk-fm-form">
    <div class="flex gap-4">
      <div class="flex-1">
        <HandsonTableHandSonTbl
          :data="ports"
          :hotSettings="portSettings"
          :key="`ports-${renderKey}`"
        />
      </div>
      <div class="flex-1">
        <HandsonTableHandSonTbl
          :data="frequencies"
          :hotSettings="freqSettings"
          :key="`freq-${renderKey}`"
        />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { PskPmDetails } from '@/composables/tracsNova/useTransmitterApi';

const props = withDefaults(
  defineProps<{
    data?: PskPmDetails;
    isEditable?: boolean;
    code?: string;
  }>(),
  {
    data: () => ({
      ports: [['EV'], ['AEV']],
      sub_carriers: [],
      frequencies: [['DF', '']],
      power_specs: [],
      frequency_specs: [],
      modulation_index_specs: [],
      spurious_specs: [],
      calibration_specs: [],
      test_profile_spurious_specs: [],
    }),
    isEditable: false,
    code: '',
  },
);

const renderKey = ref(0);

const ports = ref<string[][]>(JSON.parse(JSON.stringify(props.data.ports ?? [['EV'], ['AEV']])));
const frequencies = ref<string[][]>(JSON.parse(JSON.stringify(props.data.frequencies ?? [['DF', '']])));

function normalizeCell(value: unknown): string {
  return value == null ? '' : String(value).trim();
}

function normalizePorts(rows: unknown[][]): string[][] {
  return rows
    .map((row) => [normalizeCell(row?.[0])])
    .filter((row) => row[0] !== '');
}

function normalizeFrequencies(rows: unknown[][]): string[][] {
  return rows
    .map((row) => [normalizeCell(row?.[0]), normalizeCell(row?.[1])])
    .filter((row) => row[0] !== '');
}

const portSettings = computed(() => ({
  height: 160,
  stretchH: 'all',
  colHeaders: ['UL PORTS'],
  readOnly: !props.isEditable,
}));

const freqSettings = computed(() => ({
  height: 160,
  stretchH: 'all',
  colHeaders: ['Frequency Label', 'Frequency (MHz)'],
  readOnly: !props.isEditable,
}));

watch(
  () => props.isEditable,
  () => {
    renderKey.value += 1;
  },
);

watch(
  () => props.data,
  (newData) => {
    ports.value = JSON.parse(JSON.stringify(newData.ports ?? [['EV'], ['AEV'], ['GLOBAL']]));
    ports.value = JSON.parse(JSON.stringify(newData.ports ?? [['EV'], ['AEV']]));
    frequencies.value = JSON.parse(JSON.stringify(newData.frequencies ?? [['DF', '']]));
    renderKey.value += 1;
  },
  { deep: true },
);

function getData(): PskPmDetails {
  return {
    ports: normalizePorts(ports.value),
    sub_carriers: [],
    frequencies: normalizeFrequencies(frequencies.value),
    power_specs: JSON.parse(JSON.stringify(props.data?.power_specs ?? [])),
    frequency_specs: JSON.parse(JSON.stringify(props.data?.frequency_specs ?? [])),
    modulation_index_specs: JSON.parse(JSON.stringify(props.data?.modulation_index_specs ?? [])),
    spurious_specs: JSON.parse(JSON.stringify(props.data?.spurious_specs ?? [])),
    calibration_specs: JSON.parse(JSON.stringify(props.data?.calibration_specs ?? [])),
    test_profile_spurious_specs: JSON.parse(JSON.stringify(props.data?.test_profile_spurious_specs ?? [])),
  };
}

defineExpose({ getData });
</script>

<style scoped>
.psk-fm-form {
  min-width: 100%;
}
</style>
