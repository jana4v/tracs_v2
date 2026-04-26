<template>
  <nav class="topnav-overlay">
    <div class="topnav-container">
      <!-- Left side: Logo + Menu Toggle + App Selector -->
      <div class="topnav-left">
        <!-- Menu Toggle -->
        <button
          @click="visible = !visible"
          class="menu-toggle-btn"
          title="Toggle Menu"
        >
          <i class="i-mdi-menu" />
        </button>

        <!-- Logo -->
        <router-link to="/" class="topnav-logo">
          <img src="/isro_logo.svg" alt="logo" />
          <span class="topnav-brand">SCG</span>
        </router-link>

        <!-- App Selector Dropdown -->
        <div class="app-selector" ref="appDropdownRef">
          <button @click="toggleAppDropdown" class="app-selector-btn">
            <i class="i-mdi-apps" />
            <span>{{ selectedApp.name }}</span>
            <i class="i-mdi-chevron-down" />
          </button>

          <div v-show="showAppDropdown" class="app-dropdown">
            <div class="dropdown-header">Available Apps</div>
            <button
              v-for="app in apps"
              :key="app.id"
              @click="selectApp(app)"
              :class="['app-item', { 'app-item--active': selectedApp.id === app.id }]"
            >
              <i :class="app.icon" />
              <div class="app-item-content">
                <div class="app-item-name">{{ app.name }}</div>
                <div class="app-item-desc">{{ app.description }}</div>
              </div>
              <i v-if="selectedApp.id === app.id" class="pi pi-check" />
            </button>
          </div>
        </div>
      </div>

      <!-- Right side: Theme Toggle + User Menu -->
      <div class="topnav-right">
        <AppColorMode class="theme-toggle" />
        <Button icon="pi pi-sign-out" severity="danger" size="small" />
      </div>
    </div>
  </nav>
</template>

<script setup lang="ts">
const visible = useState<boolean>("visible", () => true);
const router = useRouter();
const route = useRoute();

const showAppDropdown = ref(false);
const appDropdownRef = ref<HTMLElement | null>(null);

interface App {
  id: string;
  name: string;
  description: string;
  icon: string;
  route: string;
}

const apps = ref<App[]>([
  {
    id: "telecommand",
    name: "Telecommand",
    description: "Satellite command interface",
    icon: "pi pi-wifi",
    route: "/tc",
  },
  // {
  //   id: "papert",
  //   name: "PAPERT",
  //   description: "Presentation generator",
  //   icon: "pi pi-palette",
  //   route: "/papert",
  // },
  // {
  //   id: "mainframe",
  //   name: "Mainframe Utils",
  //   description: "Mainframe utilities",
  //   icon: "pi pi-server",
  //   route: "/mainframe",
  // },
  // {
  //   id: "test-eval",
  //   name: "Test & Evaluation",
  //   description: "Testing and evaluation tools",
  //   icon: "pi pi-check-square",
  //   route: "/TestAndEvaluation",
  // },
]);

const selectedApp = computed(() => {
  return apps.value.find((app: App) => route.path.startsWith(app.route)) || apps.value[0];
});

function toggleAppDropdown() {
  showAppDropdown.value = !showAppDropdown.value;
}

function selectApp(app: App) {
  showAppDropdown.value = false;
  router.push(app.route);
}

function handleClickOutside(event: MouseEvent) {
  if (appDropdownRef.value && !appDropdownRef.value.contains(event.target as Node)) {
    showAppDropdown.value = false;
  }
}

onMounted(() => {
  document.addEventListener("click", handleClickOutside);
});

onBeforeUnmount(() => {
  document.removeEventListener("click", handleClickOutside);
});
</script>
<style lang="scss" scoped>
.topnav-overlay {
  position: fixed;
  width: 100%;
  top: 0;
  left: 0;
  z-index: 2000;
  background: var(--p-content-background);
  border-bottom: 1px solid var(--p-surface-border);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.topnav-container {
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 4rem;
  padding: 0 1.5rem;
}

.topnav-left {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.topnav-right {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.menu-toggle-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 2.5rem;
  height: 2.5rem;
  padding: 0;
  background: transparent;
  border: 1px solid var(--p-surface-border);
  border-radius: 8px;
  color: var(--p-primary-color);
  cursor: pointer;
  transition: all 0.2s ease;

  i {
    font-size: 1.5rem;
  }

  &:hover {
    background: var(--p-primary-color);
    color: white;
    border-color: var(--p-primary-color);
    transform: scale(1.05);
  }
}

.topnav-logo {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  text-decoration: none;
  padding: 0.5rem 1rem;
  border-radius: 8px;
  transition: all 0.2s ease;

  img {
    height: 2.5rem;
  }

  &:hover {
    background: color-mix(in srgb, var(--p-primary-color), transparent 95%);
  }
}

.topnav-brand {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--p-primary-color);
  letter-spacing: 0.5px;
}

.app-selector {
  position: relative;
}

.app-selector-btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: var(--p-content-background);
  border: 1px solid var(--p-surface-border);
  border-radius: 8px;
  color: var(--p-primary-color);
  font-size: 1rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;

  i:first-child {
    font-size: 1.5rem;
    color: var(--p-primary-color);
  }

  i:last-child {
    font-size: 1.25rem;
    margin-left: 0.25rem;
    color: var(--p-primary-color);
  }

  &:hover {
    border-color: var(--p-primary-color);
    background: color-mix(in srgb, var(--p-primary-color), transparent 95%);
  }
}

.app-dropdown {
  position: absolute;
  left: 0;
  top: calc(100% + 0.5rem);
  min-width: 300px;
  background: var(--p-content-background);
  border: 1px solid var(--p-surface-border);
  border-radius: 12px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
  padding: 0.5rem;
  z-index: 1000;
  overflow: hidden;
}

.dropdown-header {
  padding: 0.75rem 1rem;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--p-text-secondary-color);
  border-bottom: 1px solid var(--p-surface-border);
  margin-bottom: 0.5rem;
}

.app-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  width: 100%;
  padding: 0.75rem 1rem;
  background: transparent;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
  text-align: left;

  i:first-child {
    font-size: 1.25rem;
    color: var(--p-primary-color);
    flex-shrink: 0;
  }

  i.pi-check {
    font-size: 1rem;
    color: var(--p-primary-color);
    margin-left: auto;
  }

  &:hover {
    background: color-mix(in srgb, var(--p-primary-color), transparent 95%);
  }

  &.app-item--active {
    background: color-mix(in srgb, var(--p-primary-color), transparent 90%);

    .app-item-name {
      color: var(--p-primary-color);
      font-weight: 600;
    }
  }
}

.app-item-content {
  flex: 1;
}

.app-item-name {
  font-weight: 500;
  color: var(--p-text-color);
  margin-bottom: 0.125rem;
}

.app-item-desc {
  font-size: 0.75rem;
  color: var(--p-text-secondary-color);
}

.theme-toggle {
  margin-left: 0.5rem;
}

/* Dark mode is automatically handled by PrimeVue CSS variables */
</style>
