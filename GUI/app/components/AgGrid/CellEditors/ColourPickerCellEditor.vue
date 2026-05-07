<script setup>
import ColorPicker from 'primevue/colorpicker'
import { computed, onMounted, ref } from 'vue'

// Props
const props = defineProps({
  params: {
    type: Object,
    required: true,
  },
})

// Reactive References
const dropdownRef = ref(null)
const color = ref(props.params.value)

// Computed Style
const style = computed(() => ({
  width: props.params.eGridCell.style.width || '100%',
}))

// Methods
function getValue() {
  return color.value.startsWith('#') ? color.value : `#${color.value}`
}

function getGui() {
  // Implement if needed
}

function afterGuiAttached() {
  // Implement if needed
}

function removeEditor() {
  // Implement if needed
}

function isPopup() {
  return false // Default behavior
}

function valueChanged(newColor) {
  color.value = newColor
  // Optionally trigger cell update in AG Grid
  // props.params.setValue(newColor);
  // props.params.api.refreshCells({ force: true });
}

// Lifecycle Hook
onMounted(() => {
  // Focus or other initialization logic
  console.log(dropdownRef.value)
})
</script>

<template>
  <ColorPicker
    ref="dropdownRef"
    v-model="color"
    :style="style"
    input-id="cp-hex"
    format="hex"
    class="flex justify-center mb-4"
    @update:model-value="valueChanged"
  />
</template>
