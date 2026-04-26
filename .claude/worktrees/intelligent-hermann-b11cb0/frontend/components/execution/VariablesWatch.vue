<script setup lang="ts">
const executionStore = useExecutionStore()

const variables = computed(() => {
  return Object.entries(executionStore.variables).map(([key, value]) => ({
    name: key,
    value: JSON.stringify(value),
    type: typeof value,
  }))
})
</script>

<template>
  <div class="h-full overflow-auto">
    <DataTable
      :value="variables"
      striped-rows
      class="text-xs"
      :pt="{ root: { class: 'text-xs' } }"
    >
      <Column field="name" header="Variable" style="width: 40%">
        <template #body="{ data }">
          <span class="font-mono text-[var(--astra-variable)]">{{ data.name }}</span>
        </template>
      </Column>
      <Column field="value" header="Value" style="width: 40%">
        <template #body="{ data }">
          <span class="font-mono">{{ data.value }}</span>
        </template>
      </Column>
      <Column field="type" header="Type" style="width: 20%">
        <template #body="{ data }">
          <Tag :value="data.type" severity="secondary" class="text-xs" />
        </template>
      </Column>
      <template #empty>
        <div class="text-center text-muted py-4 text-xs">
          No variables to display. Start step execution to see runtime variables.
        </div>
      </template>
    </DataTable>
  </div>
</template>
