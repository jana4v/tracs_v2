
<script >
import { defineComponent, toRefs, reactive, ref } from 'vue';
import HandSonTbl from '@/components/HandsonTable/HandSonTbl.vue';
export default defineComponent({
  name: 'SpuriousSpecDataCellEditor',
  components: {
    HandSonTbl,
  },
  props: ['params'],
  setup(props) {
    const dropdownRef = ref(null);
    const spurious = ref([[, ]]);
    // eslint-disable-next-line vue/no-setup-props-destructure
    let { params } = props;
    const style = ref({ "width": '100%' });
    style.value = { "width": params.eGridCell.style.width };
    const data = reactive({
      tbl_settings: {
        height: 250,
        width: '300px',
        stretchH: 'all',
        colHeaders: ["Offset(MHz)", "Value(dBc)"],
        readOnly: false,
        columns: [
          {
            type: 'numeric',
            data:'offset',
            width:200
          },
          {
            type: 'numeric',
            data:'value',
            width:100
            
          },
        ]
        
      },
      
      counter: 0
    });

    if (params != undefined) {
      if (params.value != undefined) {
        spurious.value = params.value;
        // params.value.forEach(spur => {
        //   spur_data.push([spur.offset, spur.value])
        // });
      }
      

    }
   
    const getValue = () => {
      return spurious.value;
    };

    const getGui = () => {

    };
    const afterGuiAttached = () => {
    };

    const isPopup = () => {
      // and we could leave this method out also, false is the default
      return false;
    };


    const close_editor = () => {
      params.stopEditing();
    }

    function onKeyDown(event) {
      const key = event.key;
      if (key == 'Enter') {
        event.preventDefault();
        event.stopPropagation();
      }
    }

    return {
      ...toRefs(data),
      getValue,
      style,
      props,
      dropdownRef,
      afterGuiAttached,
      isPopup,
      close_editor,
      getGui,
      onKeyDown,
      spurious
    };
  },
  mounted() {
    // nextTick(() => {
    //   this.$refs.container.focus();
    // });
    //console.log(this.$refs.dropdownRef);
    //this.$refs.dropdownRef.showPopup('');
  },
});
</script>

<template>
  <div style="width:400px;height:400px">
    <div class="grid">
      <div class="col-12">
        <div class="m-1 text-xl w-10 font-semibold text-primary">Add Spurious detected during {{ params.test_phase }}
        </div>
        <div class="m-1 card">
          <HandSonTbl @keydown="onKeyDown" :data="spurious" :hotSettings="tbl_settings" :key="counter"></HandSonTbl>
          <Button class="mt-1 mx-1" @click="close_editor" label="close" icon="pi pi-check" />
        </div>
      </div>
    </div>

  </div>
</template>
<style>
.center-cell {
  text-align: center;
  vertical-align: middle;
}
</style>
