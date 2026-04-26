<script setup lang="ts">
const props = defineProps<{ params: any }>()

const status = computed(() => props.params.data?.status || 'pending')
const queuePosition = computed(() => props.params.data?.queue_position)

const statusConfig = computed(() => {
  switch (status.value) {
    case 'pending':
      return {
        icon: 'pi pi-clock',
        color: 'text-slate-400',
        label: 'Pending',
        spin: false,
      }
    case 'queued':
      return { icon: 'pi pi-hourglass', color: 'text-blue-400', label: 'Queued', spin: false }
    case 'running':
      return { icon: 'pi pi-spinner', color: 'text-cyan-400', label: 'Running', spin: true }
    case 'paused':
      return { icon: 'pi pi-pause-circle', color: 'text-amber-400', label: 'Paused', spin: false }
    case 'completed':
      return { icon: 'pi pi-check-circle', color: 'text-emerald-400', label: 'Completed', spin: false }
    case 'failed':
      return { icon: 'pi pi-times-circle', color: 'text-red-400', label: 'Failed', spin: false }
    case 'aborted':
      return { icon: 'pi pi-ban', color: 'text-orange-400', label: 'Aborted', spin: false }
    default:
      return { icon: 'pi pi-circle', color: 'text-slate-500', label: status.value, spin: false }
  }
})
</script>

<template>
  <div class="flex items-center gap-2 h-full">
    <i
      :class="[statusConfig.icon, statusConfig.color, { 'pi-spin': statusConfig.spin }]"
      class="text-sm"
    />
    <span class="text-xs font-medium" :class="statusConfig.color">{{ statusConfig.label }}</span>
    <span
      v-if="status === 'pending' && queuePosition"
      class="text-xs text-slate-500"
    >
      #{{ queuePosition }}
    </span>
  </div>
</template>
