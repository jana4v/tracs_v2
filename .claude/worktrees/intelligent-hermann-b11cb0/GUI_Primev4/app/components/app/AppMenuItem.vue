<template>
    <li :class="{ 'layout-root-menuitem': root, 'active-menuitem': isActiveMenu }">
      <div v-if="root && item.visible !== false" class="layout-menuitem-root-text">
        {{ item.label }}
      </div>
      <a
        v-if="(!item.to || item.items) && item.visible !== false"
        :href="item.url"
        @click="itemClick($event)"
        :class="item.class"
        :target="item.target"
        tabindex="0"
      >
        <i :class="item.icon" class="layout-menuitem-icon"></i>
        <span class="layout-menuitem-text">{{ item.label }}</span>
        <i v-if="item.items" class="pi pi-fw pi-angle-down layout-submenu-toggler"></i>
      </a>
      <Transition v-if="item.items && item.visible !== false" name="layout-submenu">
        <ul v-show="isActiveMenu" class="layout-submenu">
          <app-menu-item
            v-for="(child, i) in item.items"
            :key="i"
            :item="child"
            :root="false"
          />
        </ul>
      </Transition>
    </li>
  </template>
  
  <script setup>
  import { ref } from 'vue';
  
  const props = defineProps({
    item: {
      type: Object,
      required: true
    },
    root: {
      type: Boolean,
      default: false
    }
  });
  
  // State to track whether the submenu is active/open
  const isActiveMenu = ref(false);
  
  // Handle item click
  const itemClick = (event) => {
    if (props.item.items) {
      isActiveMenu.value = !isActiveMenu.value; // Toggle submenu visibility
      event.preventDefault(); // Prevent default link behavior
    }
  };
  </script>
  
  <style scoped>
  /* Transition for submenu */
  .layout-submenu-enter-active,
  .layout-submenu-leave-active {
    transition: opacity 0.3s ease, transform 0.3s ease;
  }
  .layout-submenu-enter-from,
  .layout-submenu-leave-to {
    opacity: 0;
    transform: translateY(-10px);
  }
  </style>