<template>
  <div class="psk-pm-form">
    <!-- ── Handsontable grids ─────────────────────────────────────────────── -->
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
          :data="subCarriers"
          :hotSettings="subCarrierSettings"
          :key="`sc-${renderKey}`"
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

// ── Props ─────────────────────────────────────────────────────────────────────

const props = withDefaults(
  defineProps<{
    data?: PskPmDetails;
    isEditable?: boolean;
    code?: string;
  }>(),
  {
    data: () => ({
      ports: [['EV'], ['AEV'], ['GLOBAL']],
      sub_carriers: [[32], [128]],
      frequencies: [['DF', ''], ['F1', ''], ['F2', '']],
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

// ── Render key (forces Handsontable remount when needed) ──────────────────────

const renderKey = ref(0);

// ── Local reactive copies of Handsontable data ────────────────────────────────

const ports       = ref<string[][]>(JSON.parse(JSON.stringify(props.data.ports)));
const subCarriers = ref<(number | string)[][]>(JSON.parse(JSON.stringify(props.data.sub_carriers)));
const frequencies = ref<string[][]>(JSON.parse(JSON.stringify(props.data.frequencies)));

// ── Handsontable settings ─────────────────────────────────────────────────────

const portSettings = computed(() => ({
  height: 160,
  stretchH: 'all',
  colHeaders: ['DL PORTS'],
  readOnly: !props.isEditable,
}));

const subCarrierSettings = computed(() => ({
  height: 160,
  stretchH: 'all',
  colHeaders: ['Sub Carriers (kHz)'],
  contextMenu: false,
  readOnly: !props.isEditable,
}));

const freqSettings = computed(() => ({
  height: 160,
  stretchH: 'all',
  colHeaders: ['Frequency Label', 'Frequency (MHz)'],
  readOnly: !props.isEditable,
}));

// Re-render Handsontable tables when editability changes (requires key refresh)
watch(
  () => props.isEditable,
  () => { renderKey.value += 1; },
);

// ── React to parent reloading saved transmitter data ──────────────────────────

watch(
  () => props.data,
  (newData) => {
    ports.value       = JSON.parse(JSON.stringify(newData.ports));
    subCarriers.value = JSON.parse(JSON.stringify(newData.sub_carriers));
    frequencies.value = JSON.parse(JSON.stringify(newData.frequencies));
    renderKey.value += 1;
  },
  { deep: true },
);

// ── Public API ────────────────────────────────────────────────────────────────

function getData(): PskPmDetails {
  return {
    ports:        ports.value,
    sub_carriers: subCarriers.value,
    frequencies:  frequencies.value,
    // Keep existing values untouched; these tables are intentionally hidden here.
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
.psk-pm-form {
  min-width: 100%;
}
</style>
