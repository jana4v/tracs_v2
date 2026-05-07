<script setup lang="ts">
import type {
  CellSelectionOptions,
  ColDef,
  DefaultMenuItem,
  GetContextMenuItemsParams,
  GridApi,
  GridReadyEvent,
  MenuItemDef,
  RowSelectionOptions,
} from 'ag-grid-community'
import { colorSchemeDarkBlue, colorSchemeLightCold, ModuleRegistry, themeQuartz } from 'ag-grid-community'
import { AllEnterpriseModule } from 'ag-grid-enterprise'
import { AgGridVue } from 'ag-grid-vue3'
import { debounce } from 'lodash'
import { v4 as uuidv4 } from 'uuid'
import { ref, shallowRef } from 'vue'
import { useColorModeStore } from '~/stores/colorMode'

const props = withDefaults(
  defineProps<{
    table_configuration_func: Function
    table_name: string
    table_height: string
    table_width: string
    app_name: string
    row_group_panel_show?: string
    get_url?: string
    save_url?: string
    delete_url?: string
    use_local_data?: boolean
    force_load_data_from_back_end?: number
    enable_column_auto_size?: boolean
    show_filter?: boolean
  }>(),
  {
    table_name: '',
    table_height: '50vh',
    table_width: '100%',
    app_name: 'cmacs',
    row_group_panel_show: 'always',
    get_url: '',
    save_url: '',
    delete_url: '',
    use_local_data: false,
    force_load_data_from_back_end: 0,
    enable_column_auto_size: true,
    show_filter: true,
  },
)
const emit = defineEmits(['cell-value-changed', 'grid-ready'])
const { $dbUtils } = useNuxtApp()
ModuleRegistry.registerModules([AllEnterpriseModule])
const colorModeStore = useColorModeStore()
const rowGroupPanelShow = ref<string>('always')

const dynamicStyles = computed(() => ({
  width: '100%',
  height: `calc(${props.table_height} - 1rem)`,
}))

const row_selection_type = ref('multiple')
const gridApi = shallowRef<GridApi | null>(null)
const rowData = ref([{}])
const show_banner = ref(false)
const DefaultDef = ref({})
const ToolBarConf = ref({})
const ColDefs = ref([])
const aggFuncs = ref({})
const banner_severity = ref('success')
const banner_msg = ref('')
const ag_grid = ref<InstanceType<typeof AgGridVue> | null>(null)
const data = ref([])
const defaultColDef = ref<ColDef>({})
// State for the dialog
const showDialog = ref(false)
const selectedRowIndex = ref(null) // Index where rows will be added
const rowCountInput = ref(1) // Number of rows to add
const quickFilter = ref('')
// Function to open the dialog
function openAddRowsDialog(insertIndex: any) {
  selectedRowIndex.value = insertIndex // Store the insertion index
  showDialog.value = true // Show the dialog
}
const is_save_allowed = ref(false)
const BANNER_TIMEOUT_ERROR = 5000 // 5 seconds for error messages
const BANNER_TIMEOUT_SUCCESS = 3000 // 3 seconds for success messages

watch(
  () => props.table_name,
  async (newValue, oldValue) => {
    await loadTable()
  },
)

watch(
  () => props.force_load_data_from_back_end,
  (newValue, oldValue) => {
    if (newValue != oldValue) {
      loadTableFromBackend()
    }
  },
)

function getContextMenuItems(params: GetContextMenuItemsParams) {
  const data = props.table_configuration_func(`${props.table_name}`)

  const conf = data.ToolBarConf
  console.log(conf)
  const delete_option = {
    // custom item
    name: 'Delete Selected Rows',
    action: () => {
      deleteSelectedRows()
    },
    cssClasses: ['red'],
  }

  const add_option = {
    name: 'Add Rows Here',
    action: async () => {
      const selectedNode = gridApi.value?.getSelectedNodes()[0]
      let insertIndex = rowData.value.length // Default to end of grid

      if (selectedNode) {
        insertIndex = selectedNode?.rowIndex // Insert at selected row
      }
      openAddRowsDialog(insertIndex)
    },
    cssClasses: ['green'],
  }

  const result: (DefaultMenuItem | MenuItemDef)[] = [
    'separator', // Add a separator
    'expandAll',
    'contractAll',
    'copy', // Copy the selected cell's value
    'copyWithHeaders', // Copy the selected cell's value with headers
    'paste', // Paste the copied value
    'export', // Export the grid data
  ]

  if (conf.is_removeRowsAllowed) {
    result.splice(0, 0, delete_option)
  }
  if (conf.is_addRowsAllowed) {
    result.splice(0, 0, add_option)
  }

  return result
}

// Function to handle adding rows
function addRows() {
  const rowCount = rowCountInput.value
  if (isNaN(rowCount) || rowCount <= 0) {
    alert('Please enter a valid number of rows.')
    return
  }
  // Create new rows
  const newRows = Array.from({ length: rowCount }, (_, index) => ({
    id: uuidv4() + (rowData.value.length + index + 1).toString(), // Auto-generate unique IDs
  }))
  // Add the new rows at the specified index
  gridApi.value?.applyTransaction({
    add: newRows,
    addIndex: selectedRowIndex.value,
  })
  // Close the dialog
  showDialog.value = false
}

function onQuickFilterChanged() {
  gridApi.value!.setGridOption('quickFilterText', quickFilter.value)
}

const cellSelection = ref<boolean | CellSelectionOptions>({
  handle: { mode: 'fill', direction: 'y' },
})

const rowSelection = ref<boolean | RowSelectionOptions>({
  mode: 'multiRow',
  checkboxes: true,
  headerCheckbox: true,
  enableClickSelection: true,
  enableSelectionWithoutKeys: true,
})

// Add unique UUIDs to each row
rowData.value = rowData.value.map(row => ({
  id: uuidv4(), // Generate a unique UUID
  ...row,
}))

function getRowId(params: any) {
  if (!params.data) {
    console.warn('getRowId called with undefined data:', params)
    return null // Return a fallback value or handle as needed
  }
  return params.data.id
}

async function onGridReady(params: GridReadyEvent) {
  gridApi.value = params.api
  emit('grid-ready')
  await loadTable()
}

async function init_table() {
  const data = props.table_configuration_func(`${props.table_name}`)
  DefaultDef.value = data.DefaultDef
  ColDefs.value = data.ColDefs
  ToolBarConf.value = data.ToolBarConf
  // console.log(ToolBarConf.value);
  rowData.value = []
  is_save_allowed.value = ToolBarConf.value.is_saveAllowed
}

const GRID_UPDATE_DELAY = 1 // Delay in milliseconds for grid updates

async function loadTable() {
  try {
    await init_table()
    // Clear any existing banner messages
    show_banner.value = false

    let tbl_data = null

    // Attempt to load data from the local database
    if (props.use_local_data) {
      tbl_data = await $dbUtils.get(props.table_name)
    }

    // If no data is found locally, load data from the backend
    if (tbl_data == null) {
      await loadTableFromBackend()
      return
    }

    // Update the grid with the local data
    setTimeout(() => {
      try {
        // gridApi.value?.applyTransaction({
        //   add: tbl_data,
        //   addIndex: selectedRowIndex.value,
        // });
        rowData.value = tbl_data
        // Auto-size columns and refresh cells if required
        if (props.enable_column_auto_size) {
          gridApi.value?.autoSizeAllColumns()
          gridApi.value?.refreshCells()
        }
      }
      catch (error) {
        console.error('Failed to update grid with local data:', error)

        banner_severity.value = 'error'
        banner_msg.value = 'Failed to load data into the grid'
        show_banner.value = true
        setTimeout(() => (show_banner.value = false), BANNER_TIMEOUT_ERROR)
      }
    }, GRID_UPDATE_DELAY)
  }
  catch (error: any) {
    // Handle any errors during the load process
    console.error('Failed to load table data:', error)

    banner_severity.value = 'error'
    banner_msg.value = error.message || 'Failed to load data'
    show_banner.value = true
    setTimeout(() => (show_banner.value = false), BANNER_TIMEOUT_ERROR)
  }
}
async function getUpdatedRows() {
  const data: any = []
  gridApi.value?.forEachNode((node: any) => {
    if (node.data != undefined && node.group == false) {
      if (Object.keys(node.data).length != 0) {
        data.push(node.data)
      }

      // console.log(node.data);
      // if (node.data.changed) {
      //   node.data.changed = false;
      //   data.push(node.data);
      // }
    }
  })
  return data
}
async function loadTableFromBackend() {
  try {
    // Clear existing data
    rowData.value = []
    // Construct the URL for fetching data
    const get_url = props.get_url
      ? props.get_url
      : `com.${props.app_name}.${props.table_name.replaceAll('#', '.')}.get`

    // Fetch data from the backend
    // const res = await useAPIFetch(get_url, { method: "get" });

    rpc(get_url, []).then(
      async (res) => {
        if (res?.error == null) {
          const tbl_data = res.data
          if (tbl_data.length === 0) {
            throw new Error('No data received from the backend')
          }

          // Save data to the local database if required
          if (props.use_local_data) {
            await $dbUtils.add(props.table_name, tbl_data)
          }
          rowData.value = tbl_data
          // Auto-size columns and refresh cells if required
          if (props.enable_column_auto_size) {
            setTimeout(() => {
              gridApi.value?.autoSizeAllColumns()
              gridApi.value?.refreshCells()
            }, 100) // Small delay to ensure the grid processes the transaction
          }
        }
        else {
          throw new Error(res.error || 'Failed to fetch data from the backend')
        }
      },
    )

    // Handle errors in the response
  }
  catch (error: any) {
    // Handle any errors during the load process
    console.error('Failed to load table data:', error)

    banner_severity.value = 'error'
    banner_msg.value = error.message || 'Failed to get data from Database'
    show_banner.value = true
    setTimeout(() => (show_banner.value = false), BANNER_TIMEOUT_ERROR)
  }
}

async function saveData() {
  // Validate for errors in the grid
  if (ag_grid.value) {
    const elements = ag_grid.value?.$el.querySelectorAll('.table_cell_error')
    if (elements.length > 0) {
      banner_severity.value = 'error'
      banner_msg.value = 'Please correct errors before saving'
      show_banner.value = true
      setTimeout(() => (show_banner.value = false), BANNER_TIMEOUT_ERROR)
      return
    }
  }

  try {
    // Prepare the data to save
    // const data = rowData.value;
    const data = await getUpdatedRows()
    // Determine the save URL
    const save_url = props.save_url
      ? props.save_url
      : `com.${props.app_name}.${props.table_name.replaceAll('#', '.')}.save`

    rpc(save_url, [data]).then(
      async (res) => {
        if (res?.error == null) {
          banner_severity.value = 'success'
          banner_msg.value = 'Database Updated'
          show_banner.value = true
          setTimeout(() => (show_banner.value = false), BANNER_TIMEOUT_SUCCESS)
        }
        else {
          banner_severity.value = 'error'
          banner_msg.value = error.message || 'Failed to Save to Database'
          show_banner.value = true
          setTimeout(() => (show_banner.value = false), BANNER_TIMEOUT_ERROR)
        }
      },
    )

    // Send the data to the backend
    // const res = await useAPIFetch(save_url, { method: "post", body: data });
    //   useAPIFetch(save_url, { method: "post", body: data })
    //       .then(async (res: any) => {
    //         if (res.error) {
    //           throw new Error(res.error);
    //         } else {
    //           banner_severity.value = "success";
    //           banner_msg.value = "Database Updated";
    //           show_banner.value = true;
    //           setTimeout(() => (show_banner.value = false), BANNER_TIMEOUT_SUCCESS);

    //         }
    //       })
    //       .catch((err) => {
    //         console.error('Unexpected error:', err);
    //       });
  }
  catch (error: any) {
    // Handle any errors during the save process
    console.error('Failed to save data:', error)

    banner_severity.value = 'error'
    banner_msg.value = error.message || 'Failed to Save to Database'
    show_banner.value = true
    setTimeout(() => (show_banner.value = false), BANNER_TIMEOUT_ERROR)
  }
}

async function deleteSelectedRows() {
  const selectedRows = gridApi.value?.getSelectedRows()

  // Check if any rows are selected
  if (!selectedRows || selectedRows.length === 0) {
    alert('No rows selected to delete.')
    return
  }

  // Backup the current data before making changes
  const currentDataBackup = [...rowData.value]

  try {
    // Remove the selected rows from the grid
    gridApi.value?.applyTransaction({
      remove: selectedRows,
    })

    // Determine the save URL
    const save_url = props.save_url
      ? props.save_url
      : `${props.app_name}/${props.table_name.replaceAll('#', '/')}/save`

    console.log(selectedRows)

    useAPIFetch(save_url, { method: 'post', body: rowData.value })
      .then(async (res: any) => {
        if (res.error) {
          throw new Error(res.error)
        }
        else {
          banner_severity.value = 'success'
          banner_msg.value = 'Database Updated'
          show_banner.value = true
          setTimeout(() => (show_banner.value = false), BANNER_TIMEOUT_SUCCESS)
        }
      })
      .catch((err) => {
        console.error('Unexpected error:', err)
      })
  }
  catch (error) {
    // Restore the grid data if deletion fails
    rowData.value = currentDataBackup // Restore the original data
    // Use applyTransaction to reset the grid's data
    gridApi.value?.applyTransaction({
      add: rowData.value, // Add all rows back to the grid
      remove: gridApi.value?.getRenderedNodes().map(node => node.data), // Remove all existing rows
    })

    // Show error message
    banner_severity.value = 'error'
    banner_msg.value
      = 'Failed to delete selected rows from Database. Data restored.'
    show_banner.value = true
    setTimeout(() => (show_banner.value = false), 3000)

    console.error('Error during deletion:', error)
  }
}

// Debounced function to refresh cells and update session storage
const debouncedRefreshCells = debounce(async (api: any) => {
  try {
    api.refreshCells({ force: true }) // Refresh grid cells
    if (props.use_local_data) {
      await $dbUtils.add(props.table_name, rowData.value)
      emit('cell-value-changed') // Emit event
    }
  }
  catch (error) {
    console.error('Error during debounced cell refresh:', error)
  }
}, 1000)

// Handle cell value changes
function cell_value_changed(event: any) {
  if (event?.api) {
    debouncedRefreshCells(event.api) // Call debounced function with API
  }
  else {
    console.warn('API not available in cell value change event')
  }
}

// Counter for tracking state (if needed)
const counter = ref(0)

// Handle column row group changes
function onColumnRowGroupChanged(event: any) {
  try {
    if (!gridApi.value) {
      console.warn('Grid API is not available')
      return
    }

    // Auto-size columns if enabled
    if (props.enable_column_auto_size) {
      gridApi.value?.autoSizeAllColumns()
    }

    // Refresh cells
    gridApi.value?.refreshCells({ force: true })

    // // Optional: Log the grid API for debugging
    // console.debug('Grid API after column row group changed:', gridApi.value.api);
  }
  catch (error) {
    console.error('Error during column row group change:', error)
  }
}

defineExpose({
  init_table,
})
</script>

<template>
  <div id="jana" :style="dynamicStyles">
    <div v-if="props.show_filter" class="ma-2 flex grow">
      <span class="text-2xl text-primary mr-1 font-italic font-bold">Filter:
      </span>
      <InputText v-model="quickFilter" class="w-3/4" type="text" @update:model-value="onQuickFilterChanged()" />
      <Button
        v-if="is_save_allowed" class="w-3/4 ml-2" label="Save" icon="i-mdi:content-save" autofocus
        @click="saveData"
      />
    </div>
    <div v-if="show_banner">
      <Message :severity="banner_severity">
        {{ banner_msg }}
      </Message>
    </div>
    <AgGridVue
      ref="ag_grid" :key="counter" style="width: 100%; height: 100%" :tooltip-show-delay="0"
      :tooltip-hide-delay="2000" :default-col-def="defaultColDef" :column-defs="ColDefs" :row-data="rowData"
      :row-numbers="false" :theme="colorModeStore.currentMode === 'dark'
        ? themeQuartz.withPart(colorSchemeDarkBlue)
        : themeQuartz.withPart(colorSchemeLightCold)
      " :cell-selection="cellSelection" :row-selection="rowSelection" :row-group-panel-show="rowGroupPanelShow"
      :agg-funcs="aggFuncs" :get-row-id="getRowId" :get-context-menu-items="getContextMenuItems"
      :tree-data="false" :on-column-row-group-changed="onColumnRowGroupChanged" @grid-ready="onGridReady"
      :undo-redo-cell-editing="true" :undo-redo-cell-editing-limit="10" @cell-value-changed="cell_value_changed"
    />

    <Dialog v-model:visible="showDialog" header="Add Rows" :modal="true" :style="{ width: '30vw' }">
      <div class="p-fluid">
        <label for="rowCount">Number of Rows:</label>
        <InputNumber id="rowCount" v-model="rowCountInput" :min="1" />
      </div>
      <template #footer>
        <Button label="Cancel" icon="pi pi-times" class="p-button-text" @click="showDialog = false" />
        <Button label="Add" icon="pi pi-check" autofocus @click="addRows" />
      </template>
    </Dialog>
  </div>
</template>

<style lang="scss">
.red {
  color: red;
}

.green {
  color: var(--p-primary-color);
}
</style>
