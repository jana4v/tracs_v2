<script setup>
import { computed, reactive, ref } from 'vue'

// Props
const props = defineProps(['params'])

// Refs
const dropdownRef = ref(null)
const color = ref(props.params.value)

// Computed style
const style = computed(() => {
  return {
    width: props.params.eGridCell.style.width,
    backgroundColor: color.value,
    color: getContrastColor(color.value),
  }
})

// Reactive data
const data = reactive({
  options: [],
  option: {},
})

// Methods
function getContrastColor(hexColor) {
  if (!hexColor)
    return '#000000'
  const r = Number.parseInt(hexColor.slice(1, 3), 16)
  const g = Number.parseInt(hexColor.slice(3, 5), 16)
  const b = Number.parseInt(hexColor.slice(5, 7), 16)
  const yiq = (r * 299 + g * 587 + b * 114) / 1000
  return yiq >= 128 ? '#000000' : '#FFFFFF'
}

function getValue() {
  // console.log(color.value);
  return color.value
}

function getGui() {}

function afterGuiAttached() {}

function removeEditor() {}

function isPopup() {
  return false // Default is false, so this method could be omitted.
}

function ValueChanged() {
  setTimeout(() => props.params.api.redrawRows(), 100)
}

function valueChanged() {
  props.params.setValue(color.value)
  props.params.api.refreshCells({ force: true })
}
</script>

<template>
  <div class="text-center" :style="style">
    {{ color }}
  </div>
</template>
