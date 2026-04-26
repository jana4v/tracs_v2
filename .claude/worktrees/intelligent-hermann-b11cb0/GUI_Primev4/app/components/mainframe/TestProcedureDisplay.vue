<script setup>
const props = defineProps({
  testProcedure: String,
})
</script>

<template>
    <div>
        <h3>Procedure</h3>
        <Textarea class="preserve-whitespace" spellcheck="false" :modelValue="props.testProcedure" variant="filled" rows="15" cols="66" />
    </div>
</template>

<style lang="scss" scoped>
.preserve-whitespace {
    white-space: pre-wrap; /* Preserve whitespace and wrap text */
    font-family: monospace; /* Use a monospaced font to align spaces and tabs */
    font-size: xx-large;
}
</style>
