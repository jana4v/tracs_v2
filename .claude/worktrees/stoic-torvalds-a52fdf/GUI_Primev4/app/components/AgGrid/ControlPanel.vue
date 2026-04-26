<script setup lang="ts">
type toolbar_status_type = {
    is_addRowsAllowed: Boolean,
    is_removeRowsAllowed: Boolean,
    is_saveAllowed: Boolean,
};

const props = defineProps({
    toolbar: {
        type: Object as () => toolbar_status_type,
    }
});
const row_count = ref(2);

</script>
   
<template>
    <div class="flex justify-content-start">
        <span class="p-buttonset">

            <Button v-if="$props.toolbar?.is_addRowsAllowed" label="Add" @click="$emit('addRow', row_count)"
                icon="pi pi-plus" />
            <input v-if="$props.toolbar?.is_addRowsAllowed" class="w-1 h-full text-center" unstyled="true"
                v-model="row_count" inputId="stacked-buttons" />
            <Button v-if="$props.toolbar?.is_addRowsAllowed" disabled label="Rows" />

            <Button v-if="$props.toolbar?.is_removeRowsAllowed" class="xml-2" severity="warning"
                label="Delete Selected Rows" @click="$emit('deleteSelectedRows')" icon="pi pi-trash" />
            <Button v-if="$props.toolbar?.is_saveAllowed" class="xml-2" severity="success" label="Save"
                @click="$emit('saveData', true)" icon="pi pi-save" />
        </span>
    </div>
</template>


<style lang="scss" scoped></style>
