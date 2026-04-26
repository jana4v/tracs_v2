<template>
    <div class="mt-50" ref="container" style="height: 400px;width: 1000px;"></div>
    <Button @click="get_data">GET Data</Button>
</template>
   
<script setup>
import { onMounted, onBeforeUnmount, ref, toRaw } from 'vue'

import { createUniver, defaultTheme, LocaleType, merge } from '@univerjs/presets';
import { UniverSheetsCorePreset } from '@univerjs/presets/preset-sheets-core';
import UniverPresetSheetsCoreEnUS from '@univerjs/presets/preset-sheets-core/locales/en-US';

import '@univerjs/presets/lib/styles/preset-sheets-core.css';

const container = ref(null);
const univerAPIRef = ref(null);


onMounted(() => {
    const { univerAPI } = createUniver({
        locale: LocaleType.EN_US,
        locales: {
            [LocaleType.EN_US]: merge(
                {},
                UniverPresetSheetsCoreEnUS,
            ),
        },
        theme: defaultTheme,
        presets: [
            UniverSheetsCorePreset({
                container: container.value,
            }),
        ],
    });

    univerAPI.createWorkbook({ "id": "gyI0JO",
  "sheetOrder": [
    "RSfWjJFv4opmE1JaiRj80"
  ],
  
  "name": "",
  "appVersion": "0.5.0",
  "locale": "enUS",
  "styles": {},
  "sheets": {
      "RSfWjJFv4opmE1JaiRj80": {
      "id": "RSfWjJFv4opmE1JaiRj80",
      "name": "Test",
      "tabColor": "",
      "hidden": 0,
      "rowCount": 10,
      "columnCount": 5,
      "zoomRatio": 1,
      "freeze": {
        "startRow": -1,
        "startColumn": -1,
        "ySplit": 0,
        "xSplit": 0
      },
      "scrollTop": 0,
      "scrollLeft": 0,
      "defaultColumnWidth": 73,
      "defaultRowHeight": 23,
      "mergeData": [],
      cellData: {
    0: {
      0: {
        v: 'A1',
        s: 'random_style_id_1'
      },
      1: {
        v: 'B1',
        s: 'random_style_id_1'
      },
      3: {
        v: 'B1',
        s: 'random_style_id_1'
      },
    },
  },
      "rowData": {},
      "columnData": {
        "0": {
          "w": 125,
          "hd": 0
        },
        "1": {
          "w": 125,
          "hd": 0
        },
        "2": {
          "w": 125,
          "hd": 0
        },
        "3": {
          "w": 125,
          "hd": 0
        },
       
      },
      "showGridlines": 1,
      "rowHeader": {
        "width": 46,
        "hidden": 0
      },
      "columnHeader": {
        "height": 20,
        "hidden": 0
      },
      "selections": [
        "A1"
      ],
      "rightToLeft": 0
    }
  },
  "resources": [
    {
      "name": "SHEET_DEFINED_NAME_PLUGIN",
      "data": ""
    }
  ] });

    univerAPIRef.value = univerAPI;


    


});

const get_data = () => {
    const fWorkbook = univerAPIRef.value.getActiveWorkbook();
const fWorksheet = fWorkbook.getActiveSheet();
const sheetSnapshot = fWorksheet.getSheet().getSnapshot();
console.log(sheetSnapshot);
}
onBeforeUnmount(() => {
    toRaw(univerAPIRef.value)?.dispose();
    univerAPIRef.value = null;
});

</script>