<template>
  <div class="content">
    <AppName appname="TRACS-Nova - Test Results" />
    <div class="coming-soon">
      <i class="pi pi-check-square" />
      <p>Test Results workspace</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { initMenu } from '@/composables/tracsNova/SideNav';

definePageMeta({
  title: 'TRACS-Nova Test Results',
});

initMenu(2);
</script>

<style scoped>
.coming-soon {
  min-height: 70vh;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  gap: 0.75rem;
  color: #64748b;
}

.coming-soon .pi {
  font-size: 2.5rem;
  color: #22d3ee;
}
</style>
