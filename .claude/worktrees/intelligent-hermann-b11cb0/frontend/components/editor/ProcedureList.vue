<script setup lang="ts">
const api = useAstraApi()
const proceduresStore = useProceduresStore()
const editorStore = useEditorStore()
const executionStore = useExecutionStore()

onMounted(async () => {
  await fetchProcedures()
})

async function fetchProcedures() {
  proceduresStore.setLoading(true)
  try {
    const data = await api.getProcedures()
    proceduresStore.setVersionedList(data.procedures)
  } catch (e) {
    console.error('Failed to fetch procedures:', e)
  } finally {
    proceduresStore.setLoading(false)
  }
}

async function loadProcedure(testName: string) {
  try {
    const proc = await api.getProcedure(testName)
    const content = proc.latest_content ?? ''
    editorStore.setContent(content)
    editorStore.setFileName(`${testName}.tst`)
    editorStore.setTestName(testName)
    editorStore.markClean()
    proceduresStore.setSelectedProcedure(testName, null)
    executionStore.addLog(`Loaded procedure: ${testName}`, 'ok')
  } catch (e: any) {
    executionStore.addLog(`Failed to load procedure: ${e.message}`, 'error')
  }
}
</script>

<template>
  <div class="flex flex-col h-full">
    <div class="flex items-center justify-between px-3 py-2 border-b border-[var(--astra-border)]">
      <span class="text-sm font-medium">Procedures</span>
      <Button icon="pi pi-refresh" size="small" text severity="secondary" @click="fetchProcedures" />
    </div>

    <div class="flex-1 overflow-auto">
      <div
        v-for="proc in proceduresStore.versionedList"
        :key="proc._id"
        class="flex items-center gap-2 px-3 py-2 text-sm cursor-pointer hover:bg-[var(--astra-border)]/30 transition-colors"
        :class="{ 'bg-[var(--astra-accent)]/10 text-[var(--astra-accent)]': proceduresStore.selectedTestName === proc.test_name }"
        @click="loadProcedure(proc.test_name)"
      >
        <i class="pi pi-file text-xs" />
        <span>{{ proc.test_name }}</span>
        <span class="text-xs text-muted">v{{ proc.latest_version }}</span>
      </div>

      <div v-if="proceduresStore.versionedList.length === 0 && !proceduresStore.loading" class="text-center text-muted text-xs py-4">
        No procedures loaded
      </div>
    </div>
  </div>
</template>
