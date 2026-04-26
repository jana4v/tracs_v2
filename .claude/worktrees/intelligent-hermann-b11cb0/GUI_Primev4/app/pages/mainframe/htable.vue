<template>
    <div class="ht-theme-main-dark-auto mt-50" style="width: 600px;">
        <hot-table :data="tableData" :rowHeaders="true"
        :contextMenu="contextMenu"
        :mergeCells="mergeCells"
        :colHeaders="true"
        :columns="columns"
        
            licenseKey="non-commercial-and-evaluation"></hot-table>
    </div>
</template>

<script setup>
import { ref } from "vue";
import { HotTable } from "@handsontable/vue3";
import { registerAllModules } from "handsontable/registry";
import SelectEditor from '~/components/Htable/pe';
import "handsontable/styles/handsontable.min.css";
import "handsontable/styles/ht-theme-main.min.css";

// Register Handsontable's modules
registerAllModules();
const contextMenu = ['row_above', 'row_below', 'remove_row', 'clear_column','mergeCells'];
const mergeCells = ref({});
// Reactive data
const tableData = ref([
    ["", "Rome", "Volvo", "Toyota", "Honda"],
    ["2016", 10, 11, 12, 13],
    ["2017", 20, 11, 14, 13],
    ["2018", 30, 15, 12, 13],
]);
const columns =  [
{ type: 'text', editor: SelectEditor },// Use the custom editor for the first column
{ type: 'text', editor: "htableSelectEditor" },// Use the registered custom editor for the second column
    {  },
    {},
    {},
    {},
  ];
  
 
</script>