<script>
import { defineComponent, reactive, toRefs } from 'vue'
import HandSonTbl from '@/components/HandsonTable/HandSonTbl.vue'

export default defineComponent({
  name: 'SpuriousSpecDataRenderer',
  components: {
    HandSonTbl,
  },
  props: ['params'],
  setup(props) {
    // console.log("Cell Renderer called");
    const ROW_HEIGHT = 32
    // eslint-disable-next-line vue/no-setup-props-destructure
    const { params } = props
    const data = reactive({
      tbl_settings: {

        colHeaders: ['Offset(MHz)', 'Value(dBc)'],
        readOnly: true,
        contextMenu: false,
        className: 'handson-cell-text-center',
        columns: [
          {
            type: 'numeric',
            data: 'offset',
            width: 200,

          },
          {
            type: 'numeric',
            data: 'value',
            width: 100,

          },
        ],
      },
      spurious: [],
      counter: 0,
    })

    function get_row_height() {
      const renderer_name = 'SpuriousSpecDataRenderer'
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
      console.log(row_height)
      return row_height
    }

    // params.api.getRowNode(0).rowHeight
    if (params != undefined) {
      if (params.value != undefined) {
        try {
          data.spurious = params.value
          const height = get_row_height()
          chanageRowHeight(height + 64)
        }
        catch (error) {
          console.log(error)
        }
      }
    }
    function chanageRowHeight(row_height) {
      params.node.setRowHeight(row_height)
      setTimeout(() => params.api.onRowHeightChanged(), 100)
    }

    const getValue = () => {
      return data.spurious
    }
    function handleDblClick(event) {
      event.stopPropagation()
    }

    return {
      ...toRefs(data),
      getValue,
      props,
      handleDblClick,

    }
  },
})
</script>

<template>
  <div class="my-2 p-2" style="width:300px;">
    <HandSonTbl :key="counter" :data="spurious" :hot-settings="tbl_settings" />
  </div>
</template>

<style>

</style>
