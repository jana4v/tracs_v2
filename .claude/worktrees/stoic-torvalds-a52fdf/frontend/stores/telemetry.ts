export interface TMBankData {
  [mnemonic: string]: any
}

export const useTelemetryStore = defineStore('telemetry', {
  state: () => ({
    banks: {} as Record<string, TMBankData>,
    mnemonics: [] as string[],
    updateInterval: 2000,
    isPolling: false,
    _pollTimer: null as ReturnType<typeof setInterval> | null,
  }),

  getters: {
    getBankData: (state) => (bankId: number): TMBankData => {
      const result: TMBankData = {}
      const prefix = `TM${bankId}.`
      for (const [key, value] of Object.entries(state.banks)) {
        if (key.startsWith(prefix)) {
          result[key.substring(prefix.length)] = value
        }
      }
      return result
    },

    bankIds: (state): number[] => {
      const ids = new Set<number>()
      for (const key of Object.keys(state.banks)) {
        const match = key.match(/^TM(\d+)\./)
        if (match) ids.add(parseInt(match[1]))
      }
      return Array.from(ids).sort()
    },
  },

  actions: {
    setBanks(data: Record<string, any>) {
      this.banks = data
    },

    setMnemonics(mnemonics: string[]) {
      this.mnemonics = mnemonics
    },

    startPolling() {
      if (this.isPolling) return
      this.isPolling = true
    },

    stopPolling() {
      this.isPolling = false
    },
  },
})
