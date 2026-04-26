<script setup lang="ts">
const props = defineProps<{ params: any }>()

const progress = computed(() => props.params.data?.progress || 0)
const currentLine = computed(() => props.params.data?.current_line || 0)
const totalLines = computed(() => props.params.data?.total_lines || 0)
const status = computed(() => props.params.data?.status || 'pending')

const barColor = computed(() => {
  if (status.value === 'completed') return 'bg-emerald-500'
  if (status.value === 'failed' || status.value === 'aborted') return 'bg-red-500'
  if (status.value === 'paused') return 'bg-amber-500'
  return 'bg-cyan-500'
})

const showBar = computed(() =>
  ['running', 'paused', 'completed', 'failed', 'aborted'].includes(status.value),
)
</script>

<template>
  <div class="flex items-center gap-2 h-full w-full pr-2">
    <template v-if="showBar">
      <div class="flex-1 h-2 bg-slate-700/60 rounded-full overflow-hidden">
        <div
          :class="barColor"
          class="h-full rounded-full transition-all duration-300"
          :style="{ width: `${progress}%` }"
        />
      </div>
      <span class="text-xs text-slate-400 tabular-nums min-w-[3rem] text-right">
        {{ currentLine }}/{{ totalLines }}
      </span>
    </template>
    <template v-else>
      <span class="text-xs text-slate-500">--</span>
    </template>
  </div>
</template>
