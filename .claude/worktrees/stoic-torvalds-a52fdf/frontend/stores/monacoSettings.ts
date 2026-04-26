export type EditorTheme = 'ASTRA-dark' | 'ASTRA-light'

export const useMonacoSettings = defineStore('monacoSettings', {
  state: () => ({
    editorFontSize: 30,
    editorTheme: 'ASTRA-dark' as EditorTheme,
  }),

  actions: {
    setEditorFontSize(size: number) {
      this.editorFontSize = size
    },

    setEditorTheme(theme: EditorTheme) {
      this.editorTheme = theme
    },

    savePreferences() {
      const prefs = {
        editorFontSize: this.editorFontSize,
        editorTheme: this.editorTheme,
      }
      localStorage.setItem('monaco-settings', JSON.stringify(prefs))
    },

    loadPreferences() {
      const saved = localStorage.getItem('monaco-settings')
      if (saved) {
        const prefs = JSON.parse(saved)
        Object.assign(this, prefs)
      }
    },
  },
})
