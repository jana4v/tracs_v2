<script setup lang="ts">
const props = defineProps<{ params: any }>()

const priorityLabel = computed(() => props.params.data?.priority_label || 'low')
const priorityNum = computed(() => props.params.data?.priority || 0)

const badgeClass = computed(() => {
  switch (priorityLabel.value) {
    case 'high': return 'bg-red-500/20 text-red-400 border-red-500/40'
    case 'medium': return 'bg-amber-500/20 text-amber-400 border-amber-500/40'
    case 'low': return 'bg-emerald-500/20 text-emerald-400 border-emerald-500/40'
    default: return 'bg-gray-500/20 text-gray-400 border-gray-500/40'
  }
})
</script>

<template>
  <div class="flex items-center h-full">
    <span
      class="px-2.5 py-0.5 text-xs font-semibold rounded-full border capitalize"
      :class="badgeClass"
    >
      {{ priorityLabel }}
    </span>
  </div>
</template>
