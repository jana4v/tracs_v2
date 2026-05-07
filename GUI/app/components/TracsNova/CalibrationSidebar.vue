<template>
  <div class="cal-sidebar">
    <nav class="cal-nav">
      <div
        v-for="item in items"
        :key="item.key"
        class="cal-nav-item"
        :class="{ active: activeSection === item.key }"
        @click="select(item.key)"
      >
        <span class="nav-icon"><i :class="item.icon" /></span>
        <span class="nav-label">{{ item.label }}</span>
      </div>
    </nav>
  </div>
</template>

<script lang="ts" setup>
const emit = defineEmits<{
  (e: 'select', section: string): void;
}>();

defineProps<{
  activeSection: string;
}>();

const items = [
  { key: 'uplink', label: 'Uplink', icon: 'pi pi-arrow-up-right' },
  { key: 'downlink', label: 'Downlink', icon: 'pi pi-arrow-down-left' },
  { key: 'tvac_ref', label: 'TVAC Ref', icon: 'pi pi-sliders-v' },
  { key: 'fixed_pad', label: 'Fixed Pad', icon: 'pi pi-minus-circle' },
  { key: 'cal_sg', label: 'Cal SG', icon: 'pi pi-cog' },
];

function select(section: string) {
  emit('select', section);
}
</script>

<style scoped>
.cal-sidebar {
  background-color: #0d1b2e;
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid #1e3050;
  padding-top: 0.75rem;
}

.cal-nav {
  flex: 1;
  overflow-y: auto;
  padding: 0.5rem 0;
}

.cal-nav-item {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.6rem 1rem;
  cursor: pointer;
  color: #94a3b8;
  font-size: 0.9rem;
  transition: background 0.15s, color 0.15s;
  border-left: 3px solid transparent;
}

.cal-nav-item:hover {
  background: #132035;
  color: #e2e8f0;
}

.cal-nav-item.active {
  color: #22d3ee;
  border-left-color: #22d3ee;
  background: #0f2744;
}

.nav-icon {
  font-size: 0.9rem;
  color: #22d3ee;
}

.nav-label {
  flex: 1;
}

/* ── Light theme overrides ──────────────────────────────────────────────── */
html:not(.dark) .cal-sidebar {
  background-color: var(--p-surface-0);
  border-right-color: var(--p-content-border-color);
}
html:not(.dark) .cal-nav-item {
  color: var(--p-text-color);
}
html:not(.dark) .cal-nav-item:hover {
  background: var(--p-surface-100);
  color: var(--p-text-color);
}
html:not(.dark) .cal-nav-item.active {
  background: var(--p-primary-50);
  color: var(--p-primary-color);
  border-left-color: var(--p-primary-color);
}
</style>
