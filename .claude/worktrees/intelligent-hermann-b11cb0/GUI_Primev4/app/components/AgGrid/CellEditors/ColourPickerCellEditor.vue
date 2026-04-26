<script setup>
import { ref, computed, onMounted } from 'vue';
import ColorPicker from 'primevue/colorpicker';

// Props
const props = defineProps({
  params: {
    type: Object,
    required: true,
  },
});

// Reactive References
const dropdownRef = ref(null);
const color = ref(props.params.value);

// Computed Style
const style = computed(() => ({
  width: props.params.eGridCell.style.width || '100%',
}));

// Methods
const getValue = () => {
  return color.value.startsWith('#') ? color.value : `#${color.value}`;
};

const getGui = () => {
  // Implement if needed
};

const afterGuiAttached = () => {
  // Implement if needed
};

const removeEditor = () => {
  // Implement if needed
};

const isPopup = () => {
  return false; // Default behavior
};

const valueChanged = (newColor) => {
  color.value = newColor;
  // Optionally trigger cell update in AG Grid
  // props.params.setValue(newColor);
  // props.params.api.refreshCells({ force: true });
};

// Lifecycle Hook
onMounted(() => {
  // Focus or other initialization logic
  console.log(dropdownRef.value);
});
</script>

<template>
  <ColorPicker
    ref="dropdownRef"
    v-model="color"
    :style="style"
    @update:modelValue="valueChanged"
    inputId="cp-hex"
    format="hex"
    class="flex justify-center mb-4"
  />
</template>