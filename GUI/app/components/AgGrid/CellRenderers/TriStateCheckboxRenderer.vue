<script setup>
import { onMounted, ref } from 'vue'

// Props passed by AG Grid
const props = defineProps({
  params: Object, // All parameters passed by AG Grid
})

const emit = defineEmits(['toggle'])

// Extract relevant data from params
const { node, colDef, api } = props.params
const aggData = node.aggData // Aggregated data for the group
const field = colDef.field // The field being rendered
console.log(props.params)
// Determine the checkbox state
const isChecked = ref(false)
const isIndeterminate = ref(false)

onMounted(() => {
  updateCheckboxState()
})

// Recursive function to collect values from all leaf nodes
function collectValues(nodes, field) {
  let values = []

  nodes.forEach((node) => {
    if (node.group) {
      // If the node is a group, recursively process its children
      values = values.concat(collectValues(node.childrenAfterGroup, field))
    }
    else {
      // If the node is a leaf, add its value to the array
      values.push(node.data[field])
    }
  })

  return values
}

// Function to calculate the checkbox state
function updateCheckboxState() {
  if (!aggData || !field)
    return

  // Collect values from all leaf nodes (recursively)
  const values = collectValues(node.childrenAfterGroup, field)
  const allTrue = values.every(value => value === true)
  const allFalse = values.every(value => value === false)
  isChecked.value = allTrue
  isIndeterminate.value = !allTrue && !allFalse
}

// Recursive function to collect updates for all leaf nodes
function collectUpdates(nodes, field, newValue) {
  let updates = []
  nodes.forEach((node) => {
    if (node.group) {
      // If the node is a group, recursively process its children
      updates = updates.concat(collectUpdates(node.childrenAfterGroup, field, newValue))
    }
    else {
      // If the node is a leaf, add it to the updates array
      updates.push({ ...node.data, [field]: newValue })
    }
  })

  return updates
}

// Toggle the checkbox and update child rows
// Toggle the checkbox and update child rows
function toggleValue() {
  const newValue = !isChecked.value // Toggle between true/false
  const updates = collectUpdates(node.childrenAfterGroup, field, newValue)
  // Apply the transaction to update the grid
  api.applyTransaction({
    update: updates,
  })

  // Update the checkbox state
  isChecked.value = newValue
  isIndeterminate.value = false
}
</script>

<template>
  <input

    type="checkbox"
    :checked="isChecked"
    :indeterminate.prop="isIndeterminate"
    @change="toggleValue"
  >
</template>
