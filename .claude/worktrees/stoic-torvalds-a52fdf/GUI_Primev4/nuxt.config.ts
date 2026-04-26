import Aura from '@primeuix/themes/aura';
import MyPreset from "./CustomThemes";
import pkg from './package.json' with { type: 'json' };

export default defineNuxtConfig({
  // Nuxt 4 compatibility
  future: {
    compatibilityVersion: 4,
  },
  
  // Directory configuration for Nuxt 4
  srcDir: 'app/',
  
  css: [
    '@fortawesome/fontawesome-free/css/all.css'
  ],

  compatibilityDate: '2024-07-04',

  ssr: false,
  devtools: { enabled: true },

  runtimeConfig: {
    public: {
      APP_VERSION: pkg.version,
      APP_NAME: pkg.name,
      // eslint-disable-next-line node/prefer-global/process
      APP_MODE: process.env?.NODE_ENV,
    },
  },

  // Performance optimizations
  vite: {
    build: {
      rollupOptions: {
        output: {
          manualChunks: {
            vendor: ['vue', 'pinia'],
            primevue: ['primevue'],
            charts: ['ag-charts-vue3', 'ag-grid-vue3'],
            utils: ['lodash', 'uuid'],
          },
        },
      },
    },
  },

  modules: [
    '@pinia/nuxt',
    '@nuxt/content',
    '@vueuse/nuxt',
    '@nuxt/test-utils/module',
    '@nuxt/image',
    '@nuxt/fonts',
    '@primevue/nuxt-module',
    '@unocss/nuxt'
  ],

  i18n: {
    langDir: 'locales',
    defaultLocale: 'en',
    strategy: 'no_prefix',
    locales: [
      { code: 'en', file: 'en.json', name: 'English' },
    ],
    vueI18n: './vue-i18n.options.ts',
  },

  content: {
    highlight: {
      theme: {
        // Default theme (same as single string)
        default: 'github-light',
        // Theme used if `html.dark`
        dark: 'github-dark',
      }
    }
  },
  
  primevue: {
    autoImport: true,
    options: {
      theme: {
        preset: MyPreset,
        options: {
          darkModeSelector: '.dark',
        },
      },
      ripple: true,
    },
    importPT: { as: 'Aura', from: '@primeuix/themes/aura' },
    components: {
      prefix: '',
    },
  },

  build: {
    transpile: ['nuxt', 'primevue'],
    
  },

  sourcemap: {
    client: 'hidden', // Better for debugging while still being production-friendly
    server: false,
  },
  app: {
    baseURL: '', // Set the base URL for your app
  },
  
  
});