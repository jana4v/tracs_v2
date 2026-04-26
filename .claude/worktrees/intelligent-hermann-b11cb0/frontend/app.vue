<script setup lang="ts">
const colorMode = useColorMode()
const themeClass = computed(() => colorMode.value)

useHead(() => ({
  htmlAttrs: { class: themeClass.value },
  bodyAttrs: { class: themeClass.value },
}))
</script>

<template>
  <NuxtLayout>
    <NuxtPage />
  </NuxtLayout>
  <Toast position="top-right" />
</template>
