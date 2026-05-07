import { defineNuxtPlugin } from '#app'
import { install as VueMonacoEditorPlugin } from '@guolao/vue-monaco-editor'
import * as monaco from 'monaco-editor'

export default defineNuxtPlugin((nuxtApp) => {
  nuxtApp.vueApp.use(VueMonacoEditorPlugin, {
    monaco,
  })
})
