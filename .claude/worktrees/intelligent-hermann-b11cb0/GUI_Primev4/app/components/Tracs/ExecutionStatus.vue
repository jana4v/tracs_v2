<template>
  <div>
    <div class="grid">
      <div class="col-12">
        <h1 class="mb-0 mt-3 text-3xl flex text-primary-600 font-bold">Status</h1>
        <ProgressBar :value="test_execution_status_store.progress"  class="mt-3 mb-3"/>
        <div readonly :class="{ 'fixed-size-textarea-light': !isDarkTheme, 'fixed-size-textarea-dark': isDarkTheme }">
          <div class="mb-4 m-2 text-2xl flex text-primary-800 font-bold">
           {{ test_execution_status_store.summary }}
          </div>
          <div v-for="item in test_execution_status_store.status" :key="item" class="p-mb-2 m-2 text-2xl flex text-primary-700 font-italic">
           {{ item }}
          </div>
        
        </div>
      </div>
    </div>
  </div>
</template>

<script  setup>
import { useLayout } from '@/layouts/composables/layout';
const { isDarkTheme } = useLayout();
const test_execution_status_store = useTestExecutionStatusStore();
</script>



<style lang="scss" scoped>
.fixed-size-textarea-light {
  width: 100%;    /* Fixed width */
  height: 200px;   /* Fixed height */
  overflow: auto;  /* Adds a scrollbar if needed */
  resize: none;    /* Prevents resizing */
  margin-top: 0;
  background-color: rgb(243, 237, 237);
  border: 2px solid rgb(0, 0, 0);
}

.fixed-size-textarea-dark {
  width: 100%;    /* Fixed width */
  height: 200px;   /* Fixed height */
  overflow: auto;  /* Adds a scrollbar if needed */
  resize: none;    /* Prevents resizing */
  margin-top: 0;
  background-color: rgb(22, 21, 21);
  border: 2px solid rgb(111, 107, 107);
}


</style>

