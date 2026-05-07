<!-- CustomTooltip.vue -->
<script setup>
import { defineProps } from 'vue'

// Define props that AG Grid will provide to the tooltip component
const props = defineProps({
  params: Object,
})

// Optional: You can use the onMounted lifecycle hook if you need to do something once the component mounts
</script>

<template>
  <div>
    <div v-if="params.data.tooltip.length > 0" class="card">
      <!-- Customize your tooltip content here -->
      <p class="mb-0 text-xl w-10 font-semibold text-primary">
        {{ params.data.tooltip }}
      </p>
    </div>
  </div>
</template>

<style scoped></style>
