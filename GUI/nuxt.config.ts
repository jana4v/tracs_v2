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
    '@fortawesome/fontawesome-free/css/all.css',
    'handsontable/styles/handsontable.css',
    'handsontable/styles/ht-theme-main.css',
    'handsontable/styles/ht-icons-main.css',
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
    optimizeDeps: {
      exclude: ['monaco-editor'],
    },
    plugins: [
      {
        // monaco-editor's marked.js has a sourceMappingURL pointing to a file
        // that is not shipped in the npm package. Vite reads the reference at
        // load time, so strip the comment in a pre-load hook to avoid the WARN.
        name: 'monaco-sourcemap-fix',
        enforce: 'pre' as const,
        async load(id: string) {
          if (id.includes('monaco-editor') && id.replace(/\\/g, '/').includes('/marked/marked.js')) {
            const { readFile } = await import('node:fs/promises')
            const code = await readFile(id.split('?')[0], 'utf-8')
            return {
              code: code.replace(/\/\/# sourceMappingURL=\S+\.map\b/g, ''),
              map: null,
            }
          }
        },
      },
    ],
    build: {
      rollupOptions: {
        // @univerjs packages declare react/react-dom as peer deps but this app
        // doesn't use React — mark them external so Rollup doesn't try to bundle them.
        // Catch react, react-dom, react-dom/client, react/jsx-runtime, etc.
        external: (id) => id === 'react' || id.startsWith('react-dom') || id.startsWith('react/'),
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
    '@nuxt/image',
    '@nuxt/fonts',
    '@primevue/nuxt-module',
    '@unocss/nuxt',
    ['@vite-pwa/nuxt', {
      registerType: 'autoUpdate',
      manifest: {
        name: 'TRACS-Nova',
        short_name: 'TRACS-Nova',
        description: 'RF Automated Checkout Software V2',
        display: 'standalone',
        orientation: 'landscape',
        start_url: '/',
        scope: '/',
        theme_color: '#0f172a',
        background_color: '#0f172a',
        icons: [
          {
            src: 'pwa-64x64.png',
            sizes: '64x64',
            type: 'image/png',
          },
          {
            src: 'pwa-192x192.png',
            sizes: '192x192',
            type: 'image/png',
          },
          {
            src: 'pwa-512x512.png',
            sizes: '512x512',
            type: 'image/png',
            purpose: 'any',
          },
          {
            src: 'maskable-icon-512x512.png',
            sizes: '512x512',
            type: 'image/png',
            purpose: 'maskable',
          },
        ],
      },
      workbox: {
        // Cache the app shell (HTML entry + JS/CSS chunks) for offline use
        // Only glob JS/CSS — avoids workbox warnings in dev where those files don't exist yet
        globPatterns: ['**/*.{js,css,html}'],
        navigateFallback: '/',
        cleanupOutdatedCaches: true,
        runtimeCaching: [
          {
            // Serve icons/images from cache-first (they rarely change)
            urlPattern: /\.(?:png|svg|ico|woff2)$/,
            handler: 'CacheFirst',
            options: {
              cacheName: 'tracs-assets-cache',
              expiration: { maxEntries: 60, maxAgeSeconds: 30 * 24 * 60 * 60 },
            },
          },
          {
            // Cache API responses from the TRACS-Nova backend with network-first
            urlPattern: /^https?:\/\/.*\/api\/.*$/,
            handler: 'NetworkFirst',
            options: {
              cacheName: 'tracs-api-cache',
              expiration: {
                maxEntries: 100,
                maxAgeSeconds: 60 * 60, // 1 hour
              },
              networkTimeoutSeconds: 10,
            },
          },
        ],
      },
      devOptions: {
        enabled: true,
        type: 'module',
        suppressWarnings: true,
      },
    }],
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
    baseURL: '/',
    // Inject PWA manifest + meta into the STATIC HTML shell (required for ssr:false)
    // useHead() in components runs client-side only — Chrome's PWA scanner needs it upfront
    head: {
      title: 'TRACS-Nova',
      meta: [
        { name: 'description', content: 'TRACS-Nova - RF Automated Checkout Software' },
        { name: 'theme-color', content: '#0f172a' },
        { name: 'mobile-web-app-capable', content: 'yes' },
        { name: 'apple-mobile-web-app-capable', content: 'yes' },
        { name: 'apple-mobile-web-app-status-bar-style', content: 'black-translucent' },
        { name: 'apple-mobile-web-app-title', content: 'TRACS-Nova' },
      ],
      link: [
        { rel: 'manifest', href: '/manifest.webmanifest' },
        { rel: 'icon', type: 'image/svg+xml', href: '/icon.svg' },
        { rel: 'apple-touch-icon', href: '/apple-touch-icon.png' },
      ],
    },
  },

  nitro: {
    output: {
      // Static files land in GoLang New/webserver/dist/web/gui after `nuxt generate`.
      publicDir: '../GoLang New/dist/web/gui',
    },
    prerender: {
      crawlLinks: false,
      routes: ['/sitemap.xml'],
    },
    devProxy: {
      '/iam/api/v1': {
        target: 'http://localhost:21005',
        changeOrigin: true,
      },
      '/api/go/v1': {
        target: 'http://localhost:21000',
        changeOrigin: true,
      },
    },
  },
  
  
});