<script>
import { defineComponent, reactive, ref, toRefs } from 'vue'
// import { useAPIFetch } from '@/composables/restApi';
export default defineComponent({
  name: 'DropDown',
  components: {
    // executionStatus: executionStatus,
  },
  props: ['params'],
  setup(props) {
    const dropdownRef = ref(null)
    // eslint-disable-next-line vue/no-setup-props-destructure
    const { params } = props
    const style = ref({ width: '100%' })
    style.value = { width: params.eGridCell.style.width }
    const data = reactive({
      options: [],
      option: {},
    })
    const show_filter = ref(true)
    if (params.show_filter != undefined) {
      show_filter.value = params.show_filter
    }

    console.log(params)
    const getValue = () => {
      return data.option?.value || params.value.name
    }

    const getGui = () => {

    }
    const afterGuiAttached = () => {
      // useAPIFetch(params.url).then((res) => {
      //   if (res.error.value == null && res.data.value.length > 0) {
      //     data.options = res.data.value.map(opt => ({ value: opt}));
      //     data.option = data.options.find(item => item.value === params.value);
      //   }
      // });

      rpc(params.url, []).then(
        async (res) => {
          if (res?.error == null) {
            data.options = res.data.map(opt => ({ value: opt }))
            data.option = data.options.find(item => item.value === params.value)
          }
        },
      )
    }
    const removeEditor = () => {

    }
    const isPopup = () => {
      // and we could leave this method out also, false is the default
      return false
    }
    const ValueChanged = () => {
      setTimeout(() => params.api.redrawRows(), 100)
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
      show_filter,
    }
  },
  mounted() {
    // nextTick(() => {
    //   this.$refs.container.focus();
    // });
    // console.log(this.$refs.dropdownRef);
    // this.$refs.dropdownRef.showPopup('');
  },
})
</script>

<template>
  <Select
    ref="dropdownRef" v-model="option" :options="options" :filter="show_filter" option-label="value"
    :placeholder="props.params.placeholder" :style="style" @hide="ValueChanged"
  />
</template>
