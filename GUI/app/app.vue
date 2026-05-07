<script setup lang="ts">
useHead({
  title: 'TRACS-Nova',
  meta: [
    { name: 'description', content: 'TRACS-Nova - RF Automated Checkout Software' },
    { name: 'viewport', content: 'width=device-width, initial-scale=1' },
    { name: 'theme-color', content: '#0f172a' },
  ],
  link: [
    { rel: 'icon', type: 'image/svg+xml', href: '/icon.svg' },
    { rel: 'apple-touch-icon', href: '/apple-touch-icon.png' },
    { rel: 'manifest', href: '/manifest.webmanifest' },
  ],
})

// Global error handling
function handleError(error: any) {
  console.error('Global error:', error)
  // You can add more sophisticated error handling here
  // Such as sending to error tracking service
}

// Register global error handler
onErrorCaptured(handleError)
</script>

<template>
  <NuxtLayout>
    <NuxtPage />
  </NuxtLayout>
</template>

<style lang='scss'>
@use 'App.scss';
</style>
