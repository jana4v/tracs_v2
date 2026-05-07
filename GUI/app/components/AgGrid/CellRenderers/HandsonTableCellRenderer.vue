<script>
import { defineComponent, reactive, ref, toRefs } from 'vue'
import HandSonTbl from '@/components/HandsonTable/HandSonTbl.vue'

function formatCompactTableData(value) {
  if (!Array.isArray(value)) {
    return ''
  }

  return value
    .map((row) => {
      if (Array.isArray(row)) {
        return row
          .filter(cell => cell !== null && cell !== undefined && `${cell}`.trim() !== '')
          .map(cell => `${cell}`.trim())
          .join(',')
      }

      if (row && typeof row === 'object') {
        const offset = row.offset ?? row.Offset ?? row[0]
        const cellValue = row.value ?? row.Value ?? row[1]
        return [offset, cellValue]
          .filter(cell => cell !== null && cell !== undefined && `${cell}`.trim() !== '')
          .map(cell => `${cell}`.trim())
          .join(',')
      }

      return `${row}`.trim()
    })
    .filter(rowText => rowText !== '')
    .join('; ')
}

export default defineComponent({
  name: 'HandsonTableCellRenderer',
  components: {
    HandSonTbl,
  },
  props: ['params'],
  setup(props) {
    const ROW_HEIGHT = 25
    // eslint-disable-next-line vue/no-setup-props-destructure
    const { params } = props
    const style = ref({ width: '100%' })
    style.value = params.style
    const data = reactive({
      tbl_settings: params.tbl_settings,
      table_data: [],
      compact_text: '',
      counter: 0,
      is_data_exist: true,
    })

    function get_row_height() {
      const renderer_name = 'HandsonTableCellRenderer'
      // let row_index = params.node.rowIndex;
      const row_index = Number.parseInt(params.node.id)
      const col_defs = params.api.getColumnDefs()
      const row_data = params.api.getRowNode(row_index).data
      const columns_with_renderer = col_defs.filter(
        (col) => {
          if (col.cellRenderer != undefined) {
            return col.cellRenderer.name == renderer_name
          }
          else {
            return false
          }
        },
      )
      const column_fields_with_renderer = columns_with_renderer.map(
        col => col.field,
      )

      let row_height = 0
      // console.log(row_data);
      Object.keys(row_data).forEach((key) => {
        if (column_fields_with_renderer.includes(key)) {
          const data = row_data[key]
          if (row_height < ROW_HEIGHT * data.length) {
            row_height = ROW_HEIGHT * data.length
          }
        }
      })
      return row_height
    }

    // params.api.getRowNode(0).rowHeight
    if (params != undefined) {
      if (params.value != undefined) {
        try {
          data.table_data = params.value
          data.is_data_exist = params.value.length > 0
          if (params.compactText === true) {
            data.compact_text = formatCompactTableData(params.value)
            changeRowHeight(48)
          }
          else {
            const height = get_row_height()
            changeRowHeight(height + 25 * 2)
          }
        }
        catch (error) {
          console.log(error)
        }
      }
      else {
        data.is_data_exist = false
      }
    }
    function changeRowHeight(row_height) {
      params.node.setRowHeight(row_height)
      setTimeout(() => params.api.onRowHeightChanged(), 100)
    }

    const getValue = () => {
      return data.table_data
    }
    function handleDblClick(event) {
      event.stopPropagation()
    }

    return {
      ...toRefs(data),
      getValue,
      props,
      handleDblClick,
      style,

    }
  },
})
</script>

<template>
  <div class="my-2 p-2" :style="style">
    <div v-if="params.compactText === true" class="compact-table-text">
      {{ compact_text }}
    </div>
    <HandSonTbl v-else-if="is_data_exist" :key="counter" :data="table_data" :hot-settings="tbl_settings" />
  </div>
</template>

<style>
.compact-table-text {
  white-space: normal;
  line-height: 1.4;
  text-align: left;
}

</style>
