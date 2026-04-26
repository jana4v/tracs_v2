<script setup>
import { ref } from 'vue';

// Props passed by AG Grid
const props = defineProps({
  value: Boolean, // The value of the cell
  node: Object,   // The row node
  data: Object,   // The entire row data
});

const { params } = props;
const { aggData} = params.node;
const { field } = params.colDef;
console.log(params,aggData)
// Emit an event to notify the parent about the toggle
const emit = defineEmits(['toggle']);

// Toggle the checkbox value
const toggleValue = () => {
  const newValue = !props.value;
  emit('toggle', { rowData: props.data, value: newValue });
};
onMounted(() => {
  console.log(props)
})
</script>

<template>
  <input
    type="checkbox"
    :checked="aggData[field]"
    @change="toggleValue"
  />
</template>