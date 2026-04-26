<script setup lang="ts">
const editorStore = useEditorStore()
const collapsed = ref(false)
</script>

<template>
  <div
    v-if="editorStore.problems.length > 0"
    class="border-t border-[var(--astra-border)] bg-[var(--astra-surface)]"
  >
    <!-- Header -->
    <div
      class="flex items-center justify-between px-3 py-1.5 cursor-pointer hover:bg-[var(--astra-border)]/30"
      @click="collapsed = !collapsed"
    >
      <div class="flex items-center gap-2 text-sm">
        <i :class="collapsed ? 'pi pi-chevron-right' : 'pi pi-chevron-down'" class="text-xs" />
        <span>Problems</span>
        <Tag severity="danger" :value="String(editorStore.errorCount)" v-if="editorStore.errorCount" class="text-xs" />
        <Tag severity="warn" :value="String(editorStore.warningCount)" v-if="editorStore.warningCount" class="text-xs" />
      </div>
    </div>

    <!-- Problem List -->
    <div v-if="!collapsed" class="max-h-40 overflow-auto">
      <div
        v-for="(problem, idx) in editorStore.problems"
        :key="idx"
        class="flex items-center gap-2 px-3 py-1 text-xs hover:bg-[var(--astra-border)]/30 cursor-pointer"
      >
        <i
          :class="problem.severity === 'error' ? 'pi pi-times-circle text-[var(--astra-error)]' : 'pi pi-exclamation-triangle text-[var(--astra-warning)]'"
        />
        <span class="text-[var(--astra-text)]/50 w-16">Line {{ problem.line_number }}</span>
        <span class="text-[var(--astra-text)]">{{ problem.message }}</span>
        <span v-if="problem.suggestion" class="text-[var(--astra-accent)] ml-auto">{{ problem.suggestion }}</span>
      </div>
    </div>
  </div>
</template>
