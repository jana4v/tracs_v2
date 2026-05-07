<script setup lang="ts">
interface Props {
  appname: string
}

defineProps<Props>()
</script>

<template>
  <header class="app-header">
    <h1 class="font-semibold pt-4 text-2xl text-gray-950 dark:text-primary">
      {{ appname }}
    </h1>
    <hr class="border-t-2 border-primary dark:border-primary">
  </header>
</template>
