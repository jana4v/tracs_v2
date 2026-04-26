<template>
  <div>
    <div class="grid">
      <div class="col-12">
        <h1 class="mb-0 mt-3 text-3xl flex text-primary-600 font-bold">
          Status
        </h1>
        <ProgressBar :value="progressValue" class="mt-3 mb-3" />
        <div
          class="fixed-size-textarea"
          :style="{ width, height }"
        >
          <div class="mb-4 m-2 text-2xl flex text-primary-800 font-bold">
            {{ store.summary }}
          </div>
          <div
            v-for="(item, index) in store.status"
            :key="index"
            class="mb-2 m-2 text-2xl flex text-primary-700 font-italic"
          >
            {{ item }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from "vue";

const props = defineProps({
  store: { type: Object, required: true }, // Pinia store instance
  width: { type: String, default: '100%' },
  height: { type: String, default: '200px' }
});

const progressValue = computed(() => {
  const val = Number(props.store.progress);
  return isNaN(val) ? 0 : Math.max(0, Math.min(100, val));
});
</script>

<style lang="scss" scoped>
.fixed-size-textarea {
  overflow: auto;
  resize: none;
  margin-top: 0;
  background-color: var(--surface-ground);
  border: 2px solid rgb(0, 0, 0);
  transition: border-color 0.2s;

  // Dark theme support
  .dark & {
    background-color: var(--surface-ground);
    border: 2px solid rgb(111, 107, 107);
  }
}
</style>
