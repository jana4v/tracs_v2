<script setup lang="ts">
const telemetryStore = useTelemetryStore()
</script>

<template>
  <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
    <TMBankCard
      v-for="bankId in telemetryStore.bankIds"
      :key="bankId"
      :bank-id="bankId"
      :data="telemetryStore.getBankData(bankId)"
    />
  </div>
</template>
