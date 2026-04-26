<template>
    <div> 
        <hot-table ref="hotTable" :settings="hotTblSettings" :data="data"></hot-table>
    </div>
</template>
  
<script setup>
import { HotTable } from '@handsontable/vue3';
import { registerAllModules } from 'handsontable/registry';
import 'handsontable/dist/handsontable.full.css';


//const hotTable = ref(null);
// register Handsontable's modules
registerAllModules();

const props = defineProps({
    //data: [],
    data: {
        type: Array,
        default: []
    },
    hotSettings: {  }
});

// console.log("FROM HANDSON_TABLE START");
// console.log(props.hotSettings.readOnly);
// console.log("FROM HANDSON_TABLE END");

const defaultColDef = {
    
    contextMenu: ['row_above', 'row_below', 'remove_row'],
    rowHeaders: false,
    width: '100%',
    mergeCells: false,
    colHeaders: ["DL Portzzz"],
    licenseKey: 'non-commercial-and-evaluation'

}

const hotTblSettings = ref({...defaultColDef, ...props.hotSettings})

const test = () => {

    // hotTable.value.hotInstance.loadData([['new', 'data']]);
    
    console.log(hotSettings.value.data)
}

</script>

  