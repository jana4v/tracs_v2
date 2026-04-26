import type { SimulatorMnemonic, SimulatorUpdateItem, TmMnemonic } from '~/types/astra'

export const useSimulatorStore = defineStore('simulator', {
  state: () => ({
    mnemonics: [] as SimulatorMnemonic[],
    originalValues: {} as Record<string, string>,
    loading: false,
    loadError: null as string | null,
    simulating: false,
    simulatorStatus: null as { config: Record<string, string>; mnemonic_count: number } | null,
    subsystems: [] as string[],
    selectedSubsystems: [] as string[],
    searchQuery: '' as string,
  }),

  getters: {
    changedMnemonics(): SimulatorUpdateItem[] {
      const changes: SimulatorUpdateItem[] = []
      for (const m of this.mnemonics) {
        if (m.value !== this.originalValues[m.mnemonic]) {
          changes.push({ mnemonic: m.mnemonic, value: m.value })
        }
      }
      console.log('changedMnemonics:', changes.length, changes)
      return changes
    },

    hasChanges(): boolean {
      return this.changedMnemonics.length > 0
    },

    changeCount(): number {
      return this.changedMnemonics.length
    },

    filteredMnemonics(): SimulatorMnemonic[] {
      let result = this.mnemonics
      if (this.searchQuery) {
        const q = this.searchQuery.toLowerCase()
        result = result.filter(m =>
          m.mnemonic.toLowerCase().includes(q),
        )
      }
      return result
    },
  },

  actions: {
    async loadMnemonicsWithRange() {
      this.loading = true
      this.loadError = null
      try {
        const api = useAstraApi()
        const allMnemonics = await api.getSimulatorMnemonics()
        
        if (this.selectedSubsystems.length === 0) {
          this.mnemonics = []
          this.originalValues = {}
          return
        }

        const filteredMnemonics = allMnemonics.filter((m: TmMnemonic) => 
          this.selectedSubsystems.includes(m.subsystem)
        )

        const values = await api.getSimulatorValues(this.selectedSubsystems)
        const valuesMap = new Map(values.map((v: SimulatorUpdateItem) => [v.mnemonic, v.value]))

        this.mnemonics = filteredMnemonics.map((m: TmMnemonic) => ({
          mnemonic: m.cdbMnemonic,
          value: valuesMap.get(m.cdbMnemonic) || '',
          range: m.range,
          type: m.type,
        }))

        this.originalValues = {}
        for (const m of this.mnemonics) {
          this.originalValues[m.mnemonic] = m.value
        }
      } catch (e: any) {
        this.loadError = e?.message || 'Failed to load mnemonics from simulator'
        this.mnemonics = []
      } finally {
        this.loading = false
      }
    },

    async loadValues() {
      this.loading = true
      this.loadError = null
      try {
        const api = useAstraApi()
        if (this.selectedSubsystems.length === 0) {
          this.mnemonics = []
          this.originalValues = {}
          return
        }
        const values = await api.getSimulatorValues(this.selectedSubsystems)

        this.mnemonics = values.map((item: SimulatorUpdateItem) => ({
          mnemonic: item.mnemonic,
          value: item.value,
        }))

        this.originalValues = {}
        for (const m of this.mnemonics) {
          this.originalValues[m.mnemonic] = m.value
        }
      } catch (e: any) {
        this.loadError = e?.message || 'Failed to load mnemonics from simulator'
        this.mnemonics = []
      } finally {
        this.loading = false
      }
    },

    async loadSubsystems() {
      try {
        const api = useAstraApi()
        const subsystems = await api.getSimulatorSubsystems()
        this.subsystems = subsystems
      } catch {
        this.subsystems = []
      }
    },

    updateValue(mnemonicName: string, value: string) {
      const mnemonic = this.mnemonics.find(m => m.mnemonic === mnemonicName)
      if (mnemonic) {
        mnemonic.value = value
        console.log('updateValue:', mnemonicName, value)
      }
    },

    resetAllValues() {
      for (const m of this.mnemonics) {
        m.value = this.originalValues[m.mnemonic] || ''
      }
    },

    async loadStatus() {
      try {
        const api = useAstraApi()
        this.simulatorStatus = await api.getSimulatorStatus()
      } catch {
        this.simulatorStatus = null
      }
    },
  },
})
