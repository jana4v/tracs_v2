import type {
  DefaultMenuItem,
  GetContextMenuItemsParams,
  GridApi,
  MenuItemDef,
} from 'ag-grid-community'

function get_context_menu(gridApi: GridApi, isAddRowsAllowed: boolean, isDeleteRowsAllowed: boolean) {
  const getContextMenuItems = (params: GetContextMenuItemsParams) => {
    const result: (DefaultMenuItem | MenuItemDef)[] = [
      {
        // custom item
        name: 'Delete Selected Rows',
        action: () => {
          const selectedRows = gridApi.getSelectedRows()
          if (selectedRows?.length === 0) {
            alert('No rows selected to delete.')
            return
          }

          // Remove the selected rows
          gridApi.applyTransaction({
            remove: selectedRows,
          })
        },
        cssClasses: ['red'],
      },

      {
        name: 'Add Rows Here',
        action: async () => {
          const selectedNode = gridApi.getSelectedNodes()[0]
          let insertIndex = rowData.value.length // Default to end of grid

          if (selectedNode) {
            insertIndex = selectedNode?.rowIndex // Insert at selected row
          }
          openAddRowsDialog(insertIndex)
        },
        cssClasses: ['green'],
      },
      'separator', // Add a separator
      'expandAll',
      'contractAll',
      'copy', // Copy the selected cell's value
      'copyWithHeaders', // Copy the selected cell's value with headers
      'paste', // Paste the copied value
      'export', // Export the grid data
    ]
    return result
  }
}
