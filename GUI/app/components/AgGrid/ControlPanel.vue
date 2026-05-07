<script setup lang="ts">
interface toolbar_status_type {
  is_addRowsAllowed: boolean
  is_removeRowsAllowed: boolean
  is_saveAllowed: boolean
}

const props = defineProps({
  toolbar: {
    type: Object as () => toolbar_status_type,
  },
})
const row_count = ref(2)
</script>

<template>
  <div class="flex justify-content-start">
    <span class="p-buttonset">

      <Button
        v-if="$props.toolbar?.is_addRowsAllowed" label="Add" icon="pi pi-plus"
        @click="$emit('addRow', row_count)"
      />
      <input
        v-if="$props.toolbar?.is_addRowsAllowed" v-model="row_count" class="w-1 h-full text-center"
        unstyled="true" inputId="stacked-buttons"
      >
      <Button v-if="$props.toolbar?.is_addRowsAllowed" disabled label="Rows" />

      <Button
        v-if="$props.toolbar?.is_removeRowsAllowed" class="xml-2" severity="warning"
        label="Delete Selected Rows" icon="pi pi-trash" @click="$emit('deleteSelectedRows')"
      />
      <Button
        v-if="$props.toolbar?.is_saveAllowed" class="xml-2" severity="success" label="Save"
        icon="pi pi-save" @click="$emit('saveData', true)"
      />
    </span>
  </div>
</template>

<style lang="scss" scoped></style>
