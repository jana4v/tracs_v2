<script lang="ts">
import { defineComponent, ref } from 'vue';

export default defineComponent({
  name: 'multiLineTextRenderer',
   props: ['params'],
  setup(props) {
    const values = ref([]);
    const ROW_HEIGHT = 42;
    // eslint-disable-next-line vue/no-setup-props-destructure
    let { params } = props;
    console.log(params);

   function get_row_height() {
    const renderer_name = 'multiLineTextRenderer';
    //let row_index = params.node.rowIndex;
    let row_index = parseInt(params.node.id);
    let col_defs = params.api.getColumnDefs();
    let row_data = params.api.getRowNode(row_index).data;
    let columns_with_renderer = col_defs.filter(
      (col: any) => { 
        if(col.cellRenderer != undefined){
        return col.cellRenderer.name == renderer_name
        } else {
          return false; 
        }
      }
    );
    let column_fields_with_renderer = columns_with_renderer.map(
      (col: any) => col.field
    );

    let row_height = 0;
    console.log(row_data);
    Object.keys(row_data).forEach((key: any) => {
      if (column_fields_with_renderer.includes(key)) {
        let data = row_data[key].split(',');
        if (row_height < ROW_HEIGHT * data.length) {
          row_height = ROW_HEIGHT * data.length;
        }
      }
    });
    console.log(row_height);
    return row_height;
    }
    
    //params.api.getRowNode(0).rowHeight
    if (params != undefined) {
      if (params.value != undefined) {
        try {
            values.value = params.value.split(',');
            chanageRowHeight(get_row_height());
            
        } catch (error) {
          console.log(error);
        }
      }
    }
    function chanageRowHeight(row_height: number) {
      params.node.setRowHeight(row_height);
      setTimeout(() => params.api.onRowHeightChanged(), 100);
      
      
    }

    const getValue = () => {
      return values.value;
    };
    function handleDblClick(event: Event) {
      event.stopPropagation();
    }

    return {
      getValue,
      props,
      handleDblClick,
      values,
    };
  },
});
</script>
<template>
  <div>
    <div v-for="val in values" :key="val">
      {{ val }}
    </div>
  </div>
</template>