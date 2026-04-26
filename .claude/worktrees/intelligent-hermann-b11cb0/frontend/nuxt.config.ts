import Aura from '@primevue/themes/aura'

export default defineNuxtConfig({
  compatibilityDate: '2025-01-01',

  modules: [
    '@unocss/nuxt',
    '@pinia/nuxt',
    '@primevue/nuxt-module',
    '@nuxtjs/color-mode',
  ],

  colorMode: {
    preference: 'dark',
    fallback: 'dark',
    classSuffix: '',
    storage: 'local',
    storageKey: 'nuxt-color-mode',
  },

  ssr: false,

  primevue: {
    options: {
      theme: {
        preset: Aura,
        options: {
          darkModeSelector: '.dark',
          lightModeSelector: '.light',
        },
      },
    },
  },

  runtimeConfig: {
    public: {
      apiBase: 'http://localhost:8080/api/v1',
      wsUrl: 'ws://localhost:8080/ws',
      lspUrl: 'ws://localhost:3001',
      simulatorBase: 'http://localhost:8091',
      agGridLicenseKey: process.env.AG_GRID_LICENSE_KEY || '',
    },
  },

  css: [
    'primeicons/primeicons.css',
    '~/assets/css/main.css',
    'ag-grid-community/styles/ag-grid.css',
    'ag-grid-community/styles/ag-theme-alpine.css',
  ],

  vite: {
    optimizeDeps: {
      include: ['monaco-editor'],
    },
  },

  devtools: { enabled: true },
})
