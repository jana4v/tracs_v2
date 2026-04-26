<template>
    <div class="card">
        <div class="flex align-items-center justify-content-between">
            <h5 class="mb-2">Down Link Ports</h5>
            <Button icon="pi pi-plus" severity="success" rounded aria-label="Add" @click="addNewRow" />
        </div>
        <DataTable :value="rows">
            <Column field="data" header="Data">
                <template #body="slotProps">
                    <div class="flex align-items-center justify-content-between">
                        <div>
                            <InputText v-if="slotProps.data.editing" v-model="slotProps.data.data" />
                            <span v-else>{{ slotProps.data.data }}</span>
                        </div>
                        <div>
                            <Button icon="pi pi-pencil" severity="success" text rounded aria-label="Edit"
                                v-if="!slotProps.data.editing" @click="() => editRow(slotProps.data)" />
                            <Button icon="pi pi-check" severity="success" text rounded aria-label="Save"
                                v-if="slotProps.data.editing" @click="() => saveRow(slotProps.data)" />
                            <Button icon="pi pi-trash" severity="success" text rounded aria-label="Delete"
                                class="p-button-danger" @click="() => deleteRow(slotProps.data)" />
                        </div>
                    </div>
                </template>
            </Column>

        </DataTable>
    </div>
</template>
  
<script setup>
import { ref } from 'vue';


const rows = ref([]);

const addNewRow = () => {
    rows.value.push({
        id: Date.now(), // unique id for each new row
        data: 'Hello',
        editing: true
    });
};
const editRow = (row) => {
    row.editing = true;
};
const saveRow = (row) => {
    row.editing = false;
};
const deleteRow = (row) => {
    rows.value = rows.value.filter(r => r !== row);
};
</script>
  