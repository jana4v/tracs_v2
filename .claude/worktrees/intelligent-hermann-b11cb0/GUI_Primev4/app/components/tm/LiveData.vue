<script setup>
import { ref,watch,computed } from 'vue';
import { ModuleRegistry } from "ag-grid-community";
import { AllEnterpriseModule } from "ag-grid-enterprise";
import { AgGridVue } from "ag-grid-vue3";
import { useColorModeStore } from "~/stores/colorMode";
import {
  colorSchemeDarkBlue,
  colorSchemeLightCold,
  themeQuartz,
} from "ag-grid-community";

ModuleRegistry.registerModules([AllEnterpriseModule]);
const colorModeStore = useColorModeStore();
const gridApi = ref(AgGridVue);
const { $wamp } = useNuxtApp();
const ag_grid = ref(null);
const is_dark_theme=ref(false);
const rowData = ref([]);

const props = defineProps({
    data_source: String,
    table_height: String,
    columnDefs: Object
});


const dynamicStyles = computed(() => ({
  width: '100%',
  height: props.table_height
 }));

const onGridReady = (params) => {
    gridApi.value = params;
    params.api.sizeColumnsToFit();
    $wamp.subscribe('tm1', (args) => {
       
        let _data = args[0];

        _data.forEach(data => {
          //   console.log(data);
    let mnemonic = data[0];
    let value = data[1];

    // Find the row with the matching mnemonic
    let existingRow = rowData.value.find(row => row.mnemonic === mnemonic);

    if (existingRow) {
      // Update the value of the existing row
      existingRow.value = value;
    } else {
      // Add a new row if the mnemonic does not exist
      rowData.value = [...rowData.value, { mnemonic, value }];
    }
       });
});
};

const filter_text = ref('');
const onFilterTextBoxChanged = () => {
      gridApi.value.api.setGridOption(
        "quickFilterText",filter_text.value
      );
    };

</script>

<template>

    <div class="title">Source:{{ props.data_source }}</div>
    <div class="mx-3 mb-2">
        <span>Quick Filter: </span>
        <InputText v-model="filter_text" v-on:input="onFilterTextBoxChanged()"  />
    </div>
    <ag-grid-vue :style="dynamicStyles" 
                ref="ag_grid" 
                :theme="colorModeStore.currentMode === 'dark' ? themeQuartz.withPart(colorSchemeDarkBlue): themeQuartz.withPart(colorSchemeLightCold)" 
                :columnDefs="props.columnDefs" 
                :rowData="rowData" 
                @grid-ready="onGridReady">
    </ag-grid-vue>
      
</template>

<style lang="scss" scoped>
.title{
    font-size: large;
    margin: 4px;
    background-color: var(--surface-300);
}
</style>