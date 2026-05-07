import type { SideNavConfig } from '@/composables/useSideNav'
import { defineStore } from 'pinia'

export const useUIStore = defineStore('ui', () => {
  // Side Navigation State
  const sideNavConfig = ref<SideNavConfig | null>(null)
  const collapsed = ref(false)
  const isOnMobile = ref(false)
  const visible = ref(false)
  const lockSideNav = ref(false)
  const sideNavWidth = ref('250px')

  // Actions
  const toggleSideNav = () => {
    visible.value = !visible.value
  }

  const setSideNavConfig = (config: SideNavConfig) => {
    sideNavConfig.value = config
  }

  const setMobile = (isMobile: boolean) => {
    isOnMobile.value = isMobile
  }

  const setCollapsed = (isCollapsed: boolean) => {
    collapsed.value = isCollapsed
  }

  // Computed
  const isDrawerVisible = computed(() => {
    return visible.value || lockSideNav.value
  })

  return {
    // State
    sideNavConfig,
    collapsed,
    isOnMobile,
    visible,
    lockSideNav,
    sideNavWidth,

    // Computed
    isDrawerVisible,

    // Actions
    toggleSideNav,
    setSideNavConfig,
    setMobile,
    setCollapsed,
  }
})
