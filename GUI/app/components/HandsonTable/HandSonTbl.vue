<script setup>
import { HotTable } from '@handsontable/vue3'
import { registerAllModules } from 'handsontable/registry'
import 'handsontable/styles/handsontable.css'
import 'handsontable/styles/ht-theme-main.css'
import 'handsontable/styles/ht-icons-main.css'

const props = defineProps({
  data: {
    type: Array,
    default: [],
  },
  hotSettings: { },
})

registerAllModules()

const isDark = useDark()

const defaultColDef = {
  themeName: undefined,
  contextMenu: ['row_above', 'row_below', 'remove_row'],
  rowHeaders: false,
  width: '100%',
  mergeCells: false,
  colHeaders: ['DL Portzzz'],
  licenseKey: 'non-commercial-and-evaluation',
}

const hotTblSettings = computed(() => ({
  ...defaultColDef,
  themeName: isDark.value ? 'ht-theme-main-dark' : 'ht-theme-main',
  ...props.hotSettings,
}))
</script>

<template>
  <div :class="isDark ? 'ht-theme-main-dark' : 'ht-theme-main'">
    <HotTable :class="isDark ? 'ht-theme-main-dark' : 'ht-theme-main'" :settings="hotTblSettings" :data="data" />
  </div>
</template>
