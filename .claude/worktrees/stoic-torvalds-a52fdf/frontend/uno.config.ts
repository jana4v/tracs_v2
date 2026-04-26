import { defineConfig, presetUno, presetAttributify, presetIcons } from 'unocss'

export default defineConfig({
  presets: [
    presetUno(),
    presetAttributify(),
    presetIcons({
      scale: 1.2,
      cdn: 'https://esm.sh/',
    }),
  ],
  theme: {
    colors: {
      astra: {
        bg: '#0b0f19',
        surface: '#111827',
        border: '#263245',
        text: '#e2e8f0',
        accent: '#22d3ee',
        success: '#10b981',
        warning: '#f59e0b',
        error: '#f87171',
        keyword: '#38bdf8',
        command: '#22c55e',
        variable: '#60a5fa',
      },
    },
  },
  shortcuts: {
    'panel-card': 'bg-astra-surface/85 border border-astra-border/80 rounded-2xl shadow-xl shadow-black/30 backdrop-blur-sm',
    'text-muted': 'text-astra-text/60',
  },
})
