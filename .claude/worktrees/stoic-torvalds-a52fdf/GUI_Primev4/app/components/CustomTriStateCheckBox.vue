<template>
  
    <div  class="tri-state-checkbox"  :class="{ 'true-state': internalState === 1, 'few-state': internalState === 2 }" @click="toggleState">
      <span v-if="internalState === 1" class="checkmark">&#x2714;</span> <!-- Checkmark for true -->
      <span v-if="internalState === 2" class="dash">&#x2013;</span> <!-- Dash for null -->
      <!-- Blank for false, nothing rendered -->
     
    </div>
  </template>
  
  <script setup>
 
  // Define the prop to accept the initial state
  const props = defineProps({
    modelValue: {
      type: Number,
      default:0,
    }
  });
  
  // Define the emits to update the v-model in the parent
  const emit = defineEmits(['update:modelValue']);
  
  // Create a local state that reacts to changes from the parent via v-model
  const internalState = ref(props.modelValue);
  // Watch for changes from the parent component and update internal state accordingly
  watch(
    () => props.modelValue,
    (newValue) => {
      internalState.value = newValue;
    }
  );
  
  // Method to toggle the state when the checkbox is clicked
  const toggleState = () => {
    if (internalState.value === 1) {
      internalState.value = 0; // From true to null
    } else {
      internalState.value = 1; // From false to true
    }
    emit('update:modelValue', internalState.value);
  };
  </script>
  
  <style scoped>
  .tri-state-checkbox {
  cursor: pointer;
  user-select: none;
  border: 2px solid #ccc; /* Box border */
  width: 20px; /* Box width */
  height: 20px; /* Box height */
  display: flex;
  justify-content: center; /* Center content horizontally */
  align-items: center; /* Center content vertically */
  box-sizing: border-box; /* Include padding and border in the element's total width and height */
}
 /* Green background for true state */
.true-state {
  background-color: green;
}

/* Yellow background for null state */
.few-state {
  background-color: yellow;
}

/* Customize the appearance of the checkmark and dash */
.tri-state-checkbox span {
  display: block;
  width: 100%;
  text-align: center;
}
  </style>
  