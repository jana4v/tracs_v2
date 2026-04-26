<script>
import { defineComponent, toRefs, reactive, ref } from 'vue';
import HandSonTbl from '@/components/HandsonTable/HandSonTbl.vue';

export default defineComponent({
  name: 'HandsonTableCellRenderer',
  components: {
    HandSonTbl,
  },
  props: ['params'],
  setup(props) {
    const ROW_HEIGHT = 25;
    // eslint-disable-next-line vue/no-setup-props-destructure
    let { params } = props;
    const style = ref({ "width": '100%' });
    style.value = params.style;
    const data = reactive({
      tbl_settings: params.tbl_settings,
      table_data: [],
      counter: 0,
      is_data_exist: true
    });

    function get_row_height() {
      const renderer_name = 'HandsonTableCellRenderer';
      //let row_index = params.node.rowIndex;
      let row_index = parseInt(params.node.id);
      let col_defs = params.api.getColumnDefs();
      let row_data = params.api.getRowNode(row_index).data;
      let columns_with_renderer = col_defs.filter(
        (col) => {
          if (col.cellRenderer != undefined) {
            return col.cellRenderer.name == renderer_name
          } else {
            return false;
          }
        }
      );
      let column_fields_with_renderer = columns_with_renderer.map(
        (col) => col.field
      );

      let row_height = 0;
      //console.log(row_data);
      Object.keys(row_data).forEach((key) => {
        if (column_fields_with_renderer.includes(key)) {
          let data = row_data[key];
          if (row_height < ROW_HEIGHT * data.length) {
            row_height = ROW_HEIGHT * data.length;
          }
        }
      });
      return row_height;
    }

    //params.api.getRowNode(0).rowHeight
    if (params != undefined) {
      if (params.value != undefined) {
        try {
          data.table_data = params.value;
          data.is_data_exist = params.value.length > 0;
          console.log(params.value,params.value.length)
          let height = get_row_height();  
          changeRowHeight(height+25*2);

        } catch (error) {
          console.log(error);
        }
      }else{
        data.is_data_exist = false;
      }
    }
    function changeRowHeight(row_height) {
      params.node.setRowHeight(row_height);
      setTimeout(() => params.api.onRowHeightChanged(), 100);


    }

    const getValue = () => {
      return data.table_data;
    };
    function handleDblClick(event) {
      event.stopPropagation();
    }

    return {
      ...toRefs(data),
      getValue,
      props,
      handleDblClick,
      style

    };
  },
});
</script>
<template>
  <div class="my-2 p-2" :style="style">
    <HandSonTbl v-if="is_data_exist" :data="table_data" :hotSettings="tbl_settings" :key="counter"></HandSonTbl>
  </div>
</template>

<style>

</style>