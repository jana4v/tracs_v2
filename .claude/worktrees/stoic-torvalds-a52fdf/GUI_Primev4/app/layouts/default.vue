<script setup lang='ts'>
const collapsed = useState<boolean>('collapsed', () => false)
const isOnMobile = useState<boolean>('isOnMobile', () => false)
const visible = useState<boolean>("visible", () => true); // Made visible by default
const side_nav_config = useState<any>("side_nav_config");
const lock_side_nav = useState<boolean>("lock_side_nav", () => false);
const side_nav_width = useState("side_nav_width", () => '250px');
const isDrawerVisible = computed(() => {
  return visible.value || lock_side_nav.value;
});

</script>

<template>
  <div class="">
    <ConfirmDialog />
    <Toast />
    <!-- <AppSidebar /> -->

    <!-- <div id="workspace" :class="[{ collapsed }, { mobile: isOnMobile }]"> -->
    <AppTopbar></AppTopbar>
    <AppSideNav></AppSideNav>
      <div :class="{'mt-8':isDrawerVisible && side_nav_config?.show_side_nav,' mt-8':!(isDrawerVisible && side_nav_config?.show_side_nav) }"
      :style="{ marginLeft: isDrawerVisible && side_nav_config?.show_side_nav ? `${side_nav_width}` : '0' }"
      >
        <div class="ma-4 pt-4">
          <slot />
        </div>
        
      </div>
    <!-- </div> -->
  </div>
</template>

<style lang="scss">

</style>
