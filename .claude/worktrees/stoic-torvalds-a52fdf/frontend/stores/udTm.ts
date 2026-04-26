import type { UdTmRow, UdTmVersion, UdTmChange } from '~/types/astra'

export const useUdTmStore = defineStore('udTm', {
  state: () => ({
    rows: [] as UdTmRow[],
    originalRows: [] as UdTmRow[],
    versions: [] as UdTmVersion[],
    loading: false,
    saving: false,
    latestVersion: 0,
  }),

  getters: {
    isDirty(state): boolean {
      return JSON.stringify(state.rows) !== JSON.stringify(state.originalRows)
    },

    totalRowCount(state): number {
      return state.rows.length
    },

    /** Returns duplicate mnemonics (non-empty only) */
    duplicateMnemonics(state): Set<string> {
      const seen = new Map<string, number>()
      for (const r of state.rows) {
        if (r.mnemonic) {
          seen.set(r.mnemonic, (seen.get(r.mnemonic) || 0) + 1)
        }
      }
      const dupes = new Set<string>()
      for (const [m, count] of seen) {
        if (count > 1) dupes.add(m)
      }
      return dupes
    },
  },

  actions: {
    async loadRows(project?: string) {
      const api = useAstraApi()
      this.loading = true
      try {
        const result = await api.getUdTm(project)
        this.latestVersion = result.latest_version || 0
        this.rows = (result.rows || []).map((r, i) => ({
          row_number: r.row_number || i + 1,
          mnemonic: r.mnemonic || '',
          value: r.value || '',
          range: r.range || '',
          limit: r.limit || '',
          tolerance: r.tolerance || '',
        }))
        this.originalRows = JSON.parse(JSON.stringify(this.rows))
      } catch (e) {
        console.warn('[UD_TM] Failed to load rows:', e)
        this.rows = []
        this.originalRows = []
        this.latestVersion = 0
      } finally {
        this.loading = false
      }
    },

    addRows(count: number) {
      const startNum = this.rows.length + 1
      for (let i = 0; i < count; i++) {
        this.rows.push({
          row_number: startNum + i,
          mnemonic: '',
          value: '',
          range: '',
          limit: '',
          tolerance: '',
        })
      }
    },

    deleteRow(index: number) {
      this.rows.splice(index, 1)
      this.renumberRows()
    },

    renumberRows() {
      this.rows.forEach((r, i) => { r.row_number = i + 1 })
    },

    /** Check if mnemonics are unique. Returns true if valid. */
    validateMnemonics(): boolean {
      return this.duplicateMnemonics.size === 0
    },

    computeChanges(): UdTmChange[] {
      const changes: UdTmChange[] = []

      const origMnemonics = this.originalRows.map(r => r.mnemonic).filter(Boolean)
      const newMnemonics = this.rows.map(r => r.mnemonic).filter(Boolean)

      // Deleted rows
      const deletedMnemonics = origMnemonics.filter(m => !newMnemonics.includes(m))
      for (const mnemonic of deletedMnemonics) {
        const orig = this.originalRows.find(r => r.mnemonic === mnemonic)
        if (orig) {
          changes.push({ type: 'deleted', row_number: orig.row_number, mnemonic })
        }
      }

      // Added rows
      const addedMnemonics = newMnemonics.filter(m => !origMnemonics.includes(m))
      for (const mnemonic of addedMnemonics) {
        const row = this.rows.find(r => r.mnemonic === mnemonic)
        if (row) {
          changes.push({ type: 'added', row_number: row.row_number, mnemonic })
        }
      }

      // Modified rows
      const sharedMnemonics = newMnemonics.filter(m => origMnemonics.includes(m))
      for (const mnemonic of sharedMnemonics) {
        const orig = this.originalRows.find(r => r.mnemonic === mnemonic)
        const curr = this.rows.find(r => r.mnemonic === mnemonic)
        if (!orig || !curr) continue
        for (const field of ['value', 'range', 'limit', 'tolerance'] as const) {
          if (orig[field] !== curr[field]) {
            changes.push({
              type: 'modified',
              row_number: curr.row_number,
              mnemonic,
              field,
              old_value: orig[field],
              new_value: curr[field],
            })
          }
        }
        if (orig.row_number !== curr.row_number) {
          changes.push({
            type: 'reordered',
            row_number: curr.row_number,
            mnemonic,
            old_value: String(orig.row_number),
            new_value: String(curr.row_number),
          })
        }
      }

      return changes
    },

    async save(project: string, createdBy: string, changeMessage: string = '') {
      const api = useAstraApi()
      this.saving = true
      try {
        this.renumberRows()
        const changes = this.computeChanges()
        const result = await api.saveUdTm(this.rows, project, createdBy, changeMessage, changes)
        if (result.saved) {
          this.latestVersion = result.version || this.latestVersion + 1
          this.originalRows = JSON.parse(JSON.stringify(this.rows))
        }
        return result
      } finally {
        this.saving = false
      }
    },

    async loadVersions(project?: string) {
      const api = useAstraApi()
      try {
        const result = await api.getUdTmVersions(project)
        this.versions = result.versions || []
      } catch (e) {
        console.warn('[UD_TM] Failed to load versions:', e)
        this.versions = []
      }
    },
  },
})
