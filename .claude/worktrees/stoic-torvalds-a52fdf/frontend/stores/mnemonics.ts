import type { TMRef, TCRef, SCORef } from '~/types/astra'

export const useMnemonicsStore = defineStore('mnemonics', {
  state: () => ({
    tmMnemonics: [] as TMRef[],
    tcMnemonics: [] as TCRef[],
    scoCommands: [] as SCORef[],
    loading: false,
  }),

  getters: {
    tmByBank: (state) => (bank: number) =>
      state.tmMnemonics.filter(tm => tm.bank === bank),

    tmBanks: (state): number[] => {
      const banks = new Set(state.tmMnemonics.map(tm => tm.bank))
      return Array.from(banks).sort()
    },

    allRefs: (state): string[] => [
      ...state.tmMnemonics.map(tm => tm.full_ref),
      ...state.tcMnemonics.map(tc => tc.full_ref),
      ...state.scoCommands.map(sco => sco.full_ref),
    ],
  },

  actions: {
    setTMMnemonics(mnemonics: TMRef[]) {
      this.tmMnemonics = mnemonics
    },

    setTCMnemonics(mnemonics: TCRef[]) {
      this.tcMnemonics = mnemonics
    },

    setSCOCommands(commands: SCORef[]) {
      this.scoCommands = commands
    },

    setLoading(loading: boolean) {
      this.loading = loading
    },
  },
})
