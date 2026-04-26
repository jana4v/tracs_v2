<template>
  <div id="jana" style="width: 800px; height: 400px">
    <div class="ma-2 flex grow">
    <span class="text-2xl text-primary mr-1 font-italic font-bold" >Filter: </span>
    <InputText class="w-3/4" type="text" @update:modelValue=onQuickFilterChanged() v-model="quickFilter" />
    <Button class="w-3/4 ml-2" label="Save" icon="i-mdi:content-save" @click="addRows" autofocus />
  </div>
    <ag-grid-vue
      ref="ag_grid"
      style="width: 100%; height: 100%"
      :defaultColDef="defaultColDef"
      :columnDefs="colDefs"
      :rowData="rowData"
      :rowNumbers="false"
      :theme="colorModeStore.currentMode === 'dark'?themeQuartz.withPart(colorSchemeDarkBlue):themeQuartz.withPart(colorSchemeLightCold)"
      @grid-ready="onGridReady"
      :cellSelection="cellSelection"
      :rowSelection="rowSelection"
      :rowGroupPanelShow="rowGroupPanelShow"
      :aggFuncs="aggFuncs"
      :getRowId="getRowId"
      :getContextMenuItems="getContextMenuItems"
    >
    </ag-grid-vue>

    <Dialog
      v-model:visible="showDialog"
      header="Add Rows"
      :modal="true"
      :style="{ width: '30vw' }"
    >
      <div class="p-fluid">
        <label for="rowCount">Number of Rows:</label>
        <InputNumber id="rowCount" v-model="rowCountInput" :min="1" />
      </div>
      <template #footer>
        <Button
          label="Cancel"
          icon="pi pi-times"
          @click="showDialog = false"
          class="p-button-text"
        />
        <Button label="Add" icon="pi pi-check" @click="addRows" autofocus />
      </template>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
import { ModuleRegistry } from "ag-grid-community";
import { AllEnterpriseModule } from "ag-grid-enterprise";
import TriStateCheckboxRenderer from "@/components/AgGrid/CellRenderers/TriStateCheckboxRenderer";
import type {
  ColDef,
  GridApi,
  GridReadyEvent,
  CellSelectionOptions,
  RowSelectionOptions,
  GetContextMenuItemsParams,
  DefaultMenuItem,
  MenuItemDef
} from "ag-grid-community";
import {
  colorSchemeDarkBlue,
  colorSchemeLightCold,
  themeQuartz,
} from "ag-grid-community";
import { AgGridVue } from "ag-grid-vue3";
import { ref, shallowRef } from "vue";
import { v4 as uuidv4 } from "uuid";
import { useColorModeStore } from '~/stores/colorMode';

ModuleRegistry.registerModules([AllEnterpriseModule]);
const colorModeStore = useColorModeStore();
const rowGroupPanelShow = ref<string>("always");

// State for the dialog
const showDialog = ref(false);
const selectedRowIndex = ref(null); // Index where rows will be added
const rowCountInput = ref(1); // Number of rows to add
const quickFilter = ref("");
// Function to open the dialog
const openAddRowsDialog = (insertIndex:any) => {
  selectedRowIndex.value = insertIndex; // Store the insertion index
  showDialog.value = true; // Show the dialog
};

const getContextMenuItems = (params: GetContextMenuItemsParams) => {
  const result: (DefaultMenuItem | MenuItemDef)[] = [
    {
      // custom item
      name: "Delete Selected Rows",
      action: () => {
        const selectedRows = gridApi.value?.getSelectedRows();
        if (selectedRows?.length === 0) {
          alert("No rows selected to delete.");
          return;
        }

        // Remove the selected rows
        gridApi.value?.applyTransaction({
          remove: selectedRows,
        });
      },
      cssClasses: ["red"],
    },

    {
      name: "Add Rows Here",
      action: async () => {
        const selectedNode = gridApi.value?.getSelectedNodes()[0];
        let insertIndex = rowData.value.length; // Default to end of grid

        if (selectedNode) {
          insertIndex = selectedNode?.rowIndex; // Insert at selected row
        }
        openAddRowsDialog(insertIndex);
      },
      cssClasses: ["green"],
    },
    "separator", // Add a separator
    "autoSizeAll",
    "expandAll",
    "contractAll",
    "copy", // Copy the selected cell's value
    "copyWithHeaders", // Copy the selected cell's value with headers
    "paste", // Paste the copied value
    "export", // Export the grid data
  ];
  return result;
};

// Function to handle adding rows
const addRows = () => {
  const rowCount = rowCountInput.value;

  if (isNaN(rowCount) || rowCount <= 0) {
    alert("Please enter a valid number of rows.");
    return;
  }

  // Create new rows
  const newRows = Array.from({ length: rowCount }, (_, index) => ({
    id: (rowData.value.length + index + 1).toString(), // Auto-generate unique IDs
    make: "", // Default value for 'make'
    model: "", // Default value for 'model'
    price: null, // Default value for 'price'
    electric: false, // Default value for 'electric'
  }));

  // Add the new rows at the specified index
  gridApi.value?.applyTransaction({
    add: newRows,
    addIndex: selectedRowIndex.value,
  });

  // Close the dialog
  showDialog.value = false;
};

function onQuickFilterChanged() {
  gridApi.value!.setGridOption("quickFilterText", quickFilter.value);
}

const logic = (params: any) => {
  const values = params.values;
  const allTrue = values.every((value:boolean) => value === true);
  const allFalse = values.every((value:boolean) => value === false);
  return allTrue ? true : allFalse ? false : null; // Return null for mixed state
};

const aggFuncs = {
  logic: logic,
};

const gridApi = shallowRef<GridApi<IRow> | null>(null);

const cellSelection = ref<boolean | CellSelectionOptions>({
  handle: { mode: "fill" },
});

const rowSelection = ref<boolean | RowSelectionOptions>({
  mode: "multiRow",
  checkboxes: true,
  headerCheckbox: true,
  enableClickSelection: true,
  enableSelectionWithoutKeys: true,
});
interface IRow {
  make: string;
  model: string;
  price?: number;
  electric?: boolean;
}

const rowData = ref<IRow[]>([
  { make: "Tesla", model: "Model A", price: 64950, electric: true },
  { make: "Tesla", model: "Model A", price: 65950, electric: true },
  { make: "Tesla", model: "Model B", price: 68950, electric: true },
  { make: "Tesla", model: "Model D", price: 69950, electric: true },
  { make: "Ford", model: "A-Series", price: 33850, electric: false },
  { make: "Ford", model: "A-Series", price: 34850, electric: false },
  { make: "Ford", model: "C-Series", price: 35850, electric: false },
  { make: "Ford", model: "D-Series", price: 36850, electric: false },
  { make: "Toyota", model: "Corolla", price: 29600, electric: false },
  { make: "Toyota", model: "Corolla1", price: 30600, electric: false },
  { make: "Toyota", model: "Corolla3", price: 32600, electric: false },
]);

// Add unique UUIDs to each row
rowData.value = rowData.value.map((row) => ({
  id: uuidv4(), // Generate a unique UUID
  ...row,
}));

const defaultColDef = ref<ColDef>({
  width: 150,
  cellStyle: { fontWeight: "bold" },
});
const getRowId = (params) => {
  // Defensive check to ensure params.node exists
  // console.log(params);
  // Check if params.data exists
  if (!params.data) {
    console.warn("getRowId called with undefined data:", params);
    return null; // Return a fallback value or handle as needed
  }

  // Use the 'id' field from the row data as the unique identifier
  return params.data.id;
};
const colDefs = ref<ColDef<IRow>[]>([
  // {
  //   headerName: 'Drag', // Column for the drag handle
  //   rowDrag: true, // Enable row dragging
  //   maxWidth: 50, // Optional: Limit the width of the drag column
  // },
  { field: "make", editable: true, enableRowGroup: true },
  { field: "model", editable: true, enableRowGroup: true },
  { field: "price", editable: true },
  {
    field: "electric",
    editable: true,
    aggFunc: "logic",
    cellRendererSelector: (params: any) => {
      if (params.node.group) {
        return {
          component: TriStateCheckboxRenderer,
        };
      }
      return null;
    },
  },
]);

// Function to add blank rows
const addBlankRows = (count = 1) => {
  if (!gridApi.value) return;

  const newRows = Array.from({ length: count }, (_, index) => ({
    id: (rowData.value.length + index + 1).toString(), // Auto-generate unique IDs
    make: "", // Default value for 'make'
    model: "", // Default value for 'model'
    price: null, // Default value for 'price'
    electric: false, // Default value for 'electric'
  }));

  // Use applyTransaction to add the new rows
  gridApi.value.applyTransaction({
    add: newRows,
  });
};

const addBlankRowsAfterSelected = (count = 1) => {
  if (!gridApi.value) return;

  const selectedNode = gridApi.value.getSelectedNodes()[0];
  console.log(gridApi.value.getSelectedRows());
  if (!selectedNode) {
    alert("Please select a row first");
    return;
  }

  const newRows = Array.from({ length: count }, (_, index) => ({
    id: (rowData.value.length + index + 1).toString(),
    make: "",
    model: "",
    price: null,
    electric: false,
  }));

  gridApi.value.applyTransaction({
    add: newRows,
    addIndex: selectedNode.rowIndex + 1, // Insert after the selected row
  });
};
const colDefsMedalsExcluded = ref<ColDef<IRow>[]>([
  { field: "make" },
  { field: "model" },
]);

const onGridReady = (params: GridReadyEvent) => {
  gridApi.value = params.api;
};

const changeCols = () => {
  gridApi.value!.setGridOption("columnDefs", colDefsMedalsExcluded.value);
};
</script>

<style lang="scss">
.red {
  color: red;
}

.green {
  color: var(--p-primary-color);
}
</style>
