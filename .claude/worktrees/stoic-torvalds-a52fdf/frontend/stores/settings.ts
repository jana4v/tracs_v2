export type ExecutionMode = 'simulation' | 'hardware'

export const useSettingsStore = defineStore('settings', {
  state: () => ({
    mode: 'simulation' as ExecutionMode,
    autoValidate: true,
    tmPollInterval: 2000,
    globalProject: 'gsat7r',
    username: 'user1',
  }),

  actions: {
    setMode(mode: ExecutionMode) {
      this.mode = mode
    },

    setAutoValidate(value: boolean) {
      this.autoValidate = value
    },

    setTmPollInterval(interval: number) {
      this.tmPollInterval = interval
    },

    setGlobalProject(project: string) {
      this.globalProject = project
    },

    setUsername(username: string) {
      this.username = username
    },

    savePreferences() {
      const prefs = {
        mode: this.mode,
        autoValidate: this.autoValidate,
        tmPollInterval: this.tmPollInterval,
        globalProject: this.globalProject,
        username: this.username,
      }
      localStorage.setItem('astra-settings', JSON.stringify(prefs))
    },

    loadPreferences() {
      const saved = localStorage.getItem('astra-settings')
      if (saved) {
        const prefs = JSON.parse(saved)
        Object.assign(this, prefs)
      }
    },
  },
})
