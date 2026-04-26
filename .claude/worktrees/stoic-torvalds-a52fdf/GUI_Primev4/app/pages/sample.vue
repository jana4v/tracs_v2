<template>

</template>

<script setup>

const { $indexedDB } = useNuxtApp();
const data = ref(null);

const fetchData = async () => {
  data.value = await $indexedDB.getDataByKey(1);
};

</script>