<script setup>
import { ref, computed, watch } from "vue";
import Checkbox from "primevue/checkbox";

// Props: Receive parameters from the parent
const props = defineProps({
  parameters: {
    type: Array,
    required: true,
  },
  title: {
    type: String,
    required: true,
  },
});

// Emits: Send selected parameters back to the parent
const emit = defineEmits(["update:selectedParams"]);

// Reactive state for selected parameters
const selectedParams = ref([]);

// Watch for changes in selectedParams and emit the updated value to the parent
watch(selectedParams, (newVal) => {
  emit("update:selectedParams", newVal);
});

// Flag for "Select All" checkbox
const selectAll = ref(false);

// Computed property to maintain the order of selection
const orderedSelectedParams = computed(() => {
  return selectedParams.value;
});

// Function to toggle "Select All" and "Deselect All"
const toggleSelectAll = () => {
  if (selectAll.value) {
    // Select all parameters
    selectedParams.value = [...props.parameters];
  } else {
    // Deselect all parameters
    selectedParams.value = [];
  }
};
</script>

<template>
  <div class="container">
    <!-- Select All / Deselect All Checkbox -->
    <h3>{{ title }}</h3>
    <div class="mb-6 font-bold">
      <Checkbox
        v-model="selectAll"
        @change="toggleSelectAll"
        inputId="select_all"
        binary
        name="select_all"
        value="select_all"
      />
      <label class="ml-2" for="select_all">
        {{ selectAll ? "Deselect All" : "Select All" }}
      </label>
    </div>
    <div class="checkbox-wrapper">
      <div class="grid grid-cols-4 gap-2">
        <div v-for="(param, index) in parameters" :key="param" class="font-bold">
          <Checkbox
            v-model="selectedParams"
            :inputId="param"
            name="param"
            :value="param"
          />
          <label class="ml-2" :for="param">{{ param }}</label>
        </div>
      </div>
    </div>
    <!-- Display Selected Parameters
      <div class="mt-4">
        <h5>Selected Parameters:</h5>
        <ul>
          <li v-for="(param, index) in orderedSelectedParams" :key="index">{{ param }}</li>
        </ul>
      </div> -->
  </div>
</template>

<style scoped>
.container {
  border: 2px solid #ccc; /* Add a border */
  padding: 10px; /* Add padding inside the box */
  border-radius: 5px; /* Optional: Add rounded corners */
}

</style>
