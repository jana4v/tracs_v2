<script setup lang="ts">
const settingsStore = useSettingsStore()
const monacoSettings = useMonacoSettings()
const { isConnected } = useWebSocket()
const route = useRoute()

// Theme - use colorMode directly
const colorMode = useColorMode()
const themeClass = computed(() => colorMode.value)

// Load preferences on app start
onMounted(() => {
  settingsStore.loadPreferences()
  monacoSettings.loadPreferences()
  // Sync UI theme with Monaco settings on load
  if (monacoSettings.editorTheme === 'ASTRA-light') {
    colorMode.preference = 'light'
    currentThemeIndex.value = 1
  }
})

// Menu items
const menuItems = ref([
  { label: 'Dashboard', icon: 'pi pi-home', to: '/' },
  { label: 'Procedures', icon: 'pi pi-file-edit', to: '/procedures' },
  { label: 'Execute', icon: 'pi pi-bolt', to: '/execute' },
  { label: 'Scheduler', icon: 'pi pi-calendar-clock', to: '/scheduler' },
  { label: 'Editor', icon: 'pi pi-code', to: '/editor' },
  { label: 'Results', icon: 'pi pi-chart-bar', to: '/results' },
  { label: 'TM Monitor', icon: 'pi pi-wave-pulse', to: '/tm-monitor' },
  { label: 'UD TM', icon: 'pi pi-list-check', to: '/ud-tm' },
  { label: 'Background', icon: 'pi pi-sync', to: '/background' },
  { label: 'Simulator', icon: 'pi pi-sliders-h', to: '/simulator' },
  { label: 'Mnemonics', icon: 'pi pi-database', to: '/mnemonics' },
  { label: 'Settings', icon: 'pi pi-cog', to: '/settings' },
])

// Sidebar toggle
const sidebarCollapsed = ref(false)

const modeOptions = [
  { label: 'Simulation', value: 'simulation' },
  { label: 'Hardware', value: 'hardware' },
]

// Theme toggle - default to dark mode
const themes = ['dark', 'light', 'auto']
const currentThemeIndex = ref(0) // 0 = dark (default)

const currentTheme = computed(() => themes[currentThemeIndex.value])

function toggleTheme() {
  currentThemeIndex.value = (currentThemeIndex.value + 1) % themes.length
  colorMode.preference = currentTheme.value
  // Sync Monaco editor theme with UI theme
  if (currentTheme.value === 'light') {
    monacoSettings.setEditorTheme('ASTRA-light')
  } else {
    monacoSettings.setEditorTheme('ASTRA-dark')
  }
  monacoSettings.savePreferences()
}

// Get current page name
const currentPageName = computed(() => {
  const title = route.meta.title as string || 'Dashboard'
  return title
})
</script>

<template>
  <div class="app-layout" :class="themeClass">
    <!-- Top Navigation -->
    <header class="topbar">
      <div class="topbar-left">
        <button class="menu-btn" @click="sidebarCollapsed = !sidebarCollapsed">
          <i class="pi pi-bars" />
        </button>
        <div class="brand">
          <i class="pi pi-bolt brand-icon" />
          <span class="brand-text">APEX</span>
        </div>
      </div>

      <div class="topbar-right">
        <button class="icon-btn" @click="toggleTheme">
          <i v-if="currentTheme === 'dark'" class="pi pi-moon" />
          <i v-else-if="currentTheme === 'light'" class="pi pi-sun" />
          <i v-else class="pi pi-desktop" />
        </button>

        <Tag :severity="isConnected ? 'success' : 'danger'" :value="isConnected ? 'Connected' : 'Disconnected'" />

        <SelectButton v-model="settingsStore.mode" :options="modeOptions" option-label="label" option-value="value" :allow-empty="false" />
      </div>
    </header>

    <!-- Sidebar -->
    <aside class="sidebar" :class="{ collapsed: sidebarCollapsed }">
      <div class="sidebar-header">
        <i class="pi pi-bolt sidebar-logo" />
        <span class="sidebar-title">APEX</span>
      </div>

      <nav class="sidebar-nav">
        <NuxtLink v-for="item in menuItems" :key="item.to" :to="item.to" class="nav-link" :class="{ active: route.path === item.to }">
          <i :class="item.icon" />
          <span>{{ item.label }}</span>
        </NuxtLink>
      </nav>

      <div class="sidebar-footer">
        <Button label="Account" icon="pi pi-user" class="w-full" outlined size="small" />
        <Button label="Logout" icon="pi pi-sign-out" class="w-full" severity="danger" text size="small" />
      </div>
    </aside>

    <!-- Main Content -->
    <main class="main-content" :class="{ 'sidebar-open': !sidebarCollapsed }">
      <div class="page-header">
        <span class="text-muted">APEX</span>
        <i class="pi pi-angle-right text-xs mx-1" />
        <span>{{ currentPageName }}</span>
      </div>
      <div class="page-body">
        <slot />
      </div>
    </main>
  </div>
</template>

<style lang="scss" scoped>
/* Default: Dark Mode */
.app-layout {
  min-height: 100vh;
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--astra-bg, #0b0f19);
}

.topbar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 3.5rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 1rem;
  background: var(--astra-surface);
  border-bottom: 1px solid var(--astra-border);
  z-index: 100;
  color: var(--astra-text);
}

.topbar-left {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.topbar-right {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.menu-btn {
  width: 2.25rem;
  height: 2.25rem;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: 1px solid var(--topbar-border, #1e293b);
  border-radius: 6px;
  color: var(--topbar-text, #e2e8f0);
  cursor: pointer;
  transition: all 0.2s;

  &:hover {
    background: var(--astra-accent, #22d3ee);
    border-color: var(--astra-accent, #22d3ee);
    color: #0b0f19;
  }
}

.brand {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.brand-icon {
  font-size: 1.5rem;
  color: var(--astra-accent, #22d3ee);
}

.brand-text {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--astra-accent, #22d3ee);
}

.icon-btn {
  width: 2.25rem;
  height: 2.25rem;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  color: var(--topbar-text, #e2e8f0);
  cursor: pointer;
  border-radius: 6px;

  &:hover {
    background: var(--topbar-hover, #1e293b);
  }
}

.sidebar {
  position: fixed;
  top: 3.5rem;
  left: 0;
  bottom: 0;
  width: 220px;
  background: #3a4954;
  border-right: 1px solid #2a3942;
  display: flex;
  flex-direction: column;
  transition: transform 0.3s ease;
  z-index: 99;
  overflow-y: auto;

  &.collapsed {
    transform: translateX(-100%);
  }
}

.sidebar-header {
  padding: 1rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  border-bottom: 1px solid #2a3942;
}

.sidebar-logo {
  font-size: 1.25rem;
  color: var(--astra-accent, #22d3ee);
}

.sidebar-title {
  font-size: 1rem;
  font-weight: 700;
  color: var(--astra-accent, #22d3ee);
  text-transform: uppercase;
  letter-spacing: 1px;
}

.sidebar-nav {
  flex: 1;
  padding: 0.5rem;
}

.nav-link {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.625rem 0.75rem;
  margin-bottom: 0.25rem;
  border-radius: 6px;
  color: #e2e8f0;
  text-decoration: none;
  transition: all 0.2s;

  &:hover {
    background: rgba(255, 255, 255, 0.1);
    transform: translateX(4px);
  }

  &.active {
    background: var(--apex-accent, #22d3ee);
    color: #0b0f19;

    i {
      color: #0b0f19;
    }
  }

  i {
    font-size: 1rem;
    color: #b0bec5;
  }
}

.sidebar-footer {
  padding: 0.75rem;
  border-top: 1px solid #2a3942;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.main-content {
  flex: 1;
  margin-top: 3.5rem;
  margin-left: 0;
  transition: margin-left 0.3s ease;
  display: flex;
  flex-direction: column;
  overflow: hidden;

  &.sidebar-open {
    margin-left: 220px;
  }
}

.page-header {
  flex-shrink: 0;
  padding: 0.75rem 1.5rem;
  background: var(--page-header-bg, #1e293b);
  border-bottom: 1px solid var(--page-header-border, #334155);
  display: flex;
  align-items: center;
  font-size: 0.875rem;
  color: var(--sidebar-text, #e2e8f0);
}

.text-muted {
  color: var(--sidebar-muted, #94a3b8);
}

.page-body {
  flex: 1;
  padding: 1.5rem;
  overflow: auto;
}

/* Light Mode Overrides - applied via class on app-layout */
.app-layout.light {
  background: var(--astra-bg, #f8fafc);
}

.app-layout.light .topbar {
  background: var(--astra-surface);
  border-bottom-color: var(--astra-border);
  color: var(--astra-text);
}

.app-layout.light .menu-btn {
  border-color: #cbd5e1;
  color: #1e293b;
}

.app-layout.light .menu-btn:hover {
  background: #0891b2;
  border-color: #0891b2;
  color: white;
}

.app-layout.light .brand-icon,
.app-layout.light .brand-text {
  color: #0891b2;
}

.app-layout.light .icon-btn {
  color: #1e293b;
}

.app-layout.light .icon-btn:hover {
  background: #f1f5f9;
}

.app-layout.light .sidebar {
  background: #f8fafc;
  border-right-color: #e2e8f0;
}

.app-layout.light .sidebar-header,
.app-layout.light .sidebar-footer {
  border-color: #e2e8f0;
}

.app-layout.light .sidebar-logo,
.app-layout.light .sidebar-title {
  color: #0891b2;
}

.app-layout.light .nav-link {
  color: #1e293b;
}

.app-layout.light .nav-link:hover {
  background: #f1f5f9;
}

.app-layout.light .nav-link.active {
  background: #0891b2;
  color: #ffffff;
}

.app-layout.light .nav-link.active i {
  color: #ffffff;
}

.app-layout.light .nav-link i {
  color: #64748b;
}

.app-layout.light .page-header {
  background: #f8fafc;
  border-bottom-color: #e2e8f0;
  color: #1e293b;
}

.app-layout.light .text-muted {
  color: #64748b;
}
</style>
