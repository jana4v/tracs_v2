<script setup>
const props = defineProps({
    testProcedure: String,
    readonly: {
        type: Boolean,
        default: true
    },
    rows: {
        type: Number,
        default: 15
    },
    cols: {
        type: Number,
        default: 66
    }
})
const emit = defineEmits(['update:testProcedure'])

const modelValue = computed({
  get: () => props.testProcedure,
  set: (val) => emit('update:testProcedure', val)
})
</script>

<template>
    <div>
        <h3>Procedure</h3>
        <Textarea class="preserve-whitespace" spellcheck="false"  :readonly="props.readonly" v-model="modelValue"  variant="filled" :rows="props.rows" :cols="props.cols" />
        <!-- <textarea
            class="preserve-whitespace"
            :readonly="props.readonly"
            spellcheck="false"
            v-model="modelValue"
            :value="props.testProcedure"
            rows="15"
            cols="66"
        
        ></textarea> -->
    </div>
</template>

<style lang="scss" scoped>
.preserve-whitespace {
    white-space: pre-wrap; /* Preserve whitespace and wrap text */
    font-family: monospace; /* Use a monospaced font to align spaces and tabs */
    font-size: xx-large;
}
</style>
