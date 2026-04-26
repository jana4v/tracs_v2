<script setup lang="ts">
useHead({
  title: 'GEO RF Applications',
  meta: [
    { name: 'description', content: 'GEO RF Applications - Professional RF Management System' },
    { name: 'viewport', content: 'width=device-width, initial-scale=1' },
  ],
})

// Global error handling
const handleError = (error: any) => {
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
