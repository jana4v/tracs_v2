<script>
import { defineComponent, reactive, ref, toRefs } from 'vue'
import HandSonTbl from '@/components/HandsonTable/HandSonTbl.vue'

export default defineComponent({
  name: 'HandsonTableCellEditor',
  components: {
    HandSonTbl,
  },
  props: ['params'],
  setup(props) {
    const table_data = ref([{}])
    // eslint-disable-next-line vue/no-setup-props-destructure
    const { params } = props
    const style = ref({ width: '100%' })
    style.value = params.style
    const data = reactive({
      tbl_settings: params.tbl_settings,
      counter: 0,
    })

    if (params != undefined) {
      if (params.value != undefined) {
        table_data.value = params.value
      }
    }

    const getValue = () => {
      return table_data.value
    }

    const getGui = () => {

    }
    const afterGuiAttached = () => {
    }

    const isPopup = () => {
      return false
    }

    const close_editor = () => {
      params.stopEditing()
    }

    function onKeyDown(event) {
      const key = event.key
      if (key == 'Enter') {
        event.preventDefault()
        event.stopPropagation()
      }
    }

    return {
      ...toRefs(data),
      getValue,
      style,
      props,
      afterGuiAttached,
      isPopup,
      close_editor,
      getGui,
      onKeyDown,
      table_data,
    }
  },
  mounted() {

  },
})
</script>

<template>
  <div :style="style">
    <div class="grid">
      <div class="col-12">
        <div class="m-1 text-xl w-10 font-semibold text-primary">
          {{ params.editor_header }}
        </div>
        <div class="m-1 card">
          <HandSonTbl :key="counter" :data="table_data" :hot-settings="tbl_settings" @keydown="onKeyDown" />
          <Button class="mt-1 mx-1" label="close" icon="pi pi-check" @click="close_editor" />
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
