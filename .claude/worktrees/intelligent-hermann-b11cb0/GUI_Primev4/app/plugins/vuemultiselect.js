// plugins/vue-multiselect.js
import { defineNuxtPlugin } from '#app'
import Multiselect from 'vue-multiselect'
import 'vue-multiselect/dist/vue-multiselect.css'

export default defineNuxtPlugin((nuxtApp) => {
  nuxtApp.vueApp.component('Multiselect', Multiselect)
})
