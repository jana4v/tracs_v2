
<script >
import { defineComponent, toRefs, reactive, ref } from 'vue';
import { useAPIFetch } from '@/composables/restApi';
export default defineComponent({
  name: 'MultiSelectdropDown',
  components: {
    // executionStatus: executionStatus,
  },
  props: ['params'],
  setup(props) {
    const dropdownRef = ref(null);
    // eslint-disable-next-line vue/no-setup-props-destructure
    let { params } = props;
    const style = ref({ "width":'100%'});
    style.value ={ "width":params.eGridCell.style.width};
    const data = reactive({
      options: [],
      selectedOptions: []
    });

    
    const getValue = () => {
      let value="";
      data.selectedOptions.forEach((item) => {
        value += item.value + ",";
      });
      return value.slice(0, -1);
    };

    const getGui = () => {
      
    };
    const afterGuiAttached = () => {

      useAPIFetch(params.url).then((res) => {
        if (res.error.value == null && res.data.value.length > 0) {
          data.options = res.data.value.map(opt => ({ value: opt})); 
           params.value?.split(",").forEach((value) => {
            data.selectedOptions.push(data.options.find(item => item.value === value));
          });
          
        }
      });
    };
    const removeEditor = () => {
      
    };
    const isPopup = () => {
      // and we could leave this method out also, false is the default
      return false;
    };
    const ValueChanged = () => {
      setTimeout(() => params.api.redrawRows(), 100);
    }


    return {
      ...toRefs(data),
      getValue,
      style,
      ValueChanged,
      props,
      dropdownRef,
      afterGuiAttached,
      isPopup,
      removeEditor,
      getGui,
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
      <MultiSelect ref="dropdownRef" v-model="selectedOptions" :options="options" filter optionLabel="value" :placeholder='props.params.placeholder' @hide="ValueChanged" :style=style display="chip">
            <template #footer>
                <div class="py-2 px-3">
                    <b>{{ selectedOptions ? selectedOptions.length : 0 }}</b> item{{ (selectedOptions ? selectedOptions.length : 0) > 1 ? 's' : '' }} selected.
                </div>
            </template>
        </MultiSelect>
</template>

