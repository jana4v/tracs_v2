<template>
  <q-select
    ref="dropdownRef"
    class="Qselect"
    :clearable="multiple"
    :multiple="multiple"
    :use-input="true"
    :use-chips="multiple"
    standout="bg-primary text-white"
    color="green"
    filled
    outlined
    label-color="black"
    v-model="option"
    :options="filteredOptions"
    @filter="filterFunc"
    @popup-hide="removeEditor"
    input-debounce="0"
    borderless
  >
    <template v-slot:no-option>
      <q-item>
        <q-item-section class="text-grey">No Results</q-item-section>
      </q-item>
    </template>
  </q-select>
</template>
<script>
import { defineComponent, toRefs, reactive, ref, nextTick } from 'vue';
import * as restApi from 'components/ApiHandler/RestApi';

export default defineComponent({
  name: 'logicalConditionEditor',
  components: {
    // executionStatus: executionStatus,
  },
  props: ['params'],
  setup(props) {
    const dropdownRef = ref(null);
    // eslint-disable-next-line vue/no-setup-props-destructure
    let { params } = props;

    //const { multiple } = params;
    const data = reactive({
      filteredOptions: [''],
      options: [],
      option: null,
      multiple: false,
    });

    if (params != undefined) {
      if (params.value != undefined) data.option = params.value.split(',');
      data.multiple = params.multiple;
    }
    const getValue = () => {
      if (Array.isArray(data.option)) return data.option.join(',');
      return data.option;
    };

    const getGui = () => {
      //rpc(params.rpc_name, params.rpc_args.split(','), rpc_Cb_for_load_data);
    };
    const afterGuiAttached = () => {
      console.log(params);
      restApi.getData(params.url, params.node.data).then((res) => {
        console.log(res);
        console.log('-------------------------------------');
        if (res.status == 200 && res.data.length > 0) {
          data.options = res.data;

          console.log(res);
        }
        dropdownRef.value.showPopup();
      });

      //console.log(dropdownRef.value);
    };
    const removeEditor = () => {
      setTimeout(() => params.api.redrawRows(), 100);
      // console.log(params.api);
    };
    const isPopup = () => {
      // and we could leave this method out also, false is the default
      return false;
    };
    function filterFunc(val, update) {
      if (val === '') {
        update(() => {
          data.filteredOptions = data.options;
        });
      } else {
        update(() => {
          let filter = val.toLowerCase();
          data.filteredOptions = data.options.filter(
            (v) => v.toLowerCase().indexOf(filter) > -1
          );
        });
      }
    }

    return {
      ...toRefs(data),
      filterFunc,
      getValue,
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
