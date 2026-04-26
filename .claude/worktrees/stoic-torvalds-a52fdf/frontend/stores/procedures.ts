import type { Procedure, ProcedureVersion, ValidationError } from '~/types/astra'

export const useProceduresStore = defineStore('procedures', {
  state: () => ({
    loading: false,

    // procedure list
    versionedList: [] as Procedure[],

    // Current editor state (primary pane)
    selectedTestName: null as string | null,
    selectedVersion: null as number | null,
    currentContent: '',
    originalContent: '',
    currentProject: '',
    currentCreatedBy: '',
    availableVersions: [] as ProcedureVersion[],

    // Second pane (side-by-side comparison)
    secondTestName: null as string | null,
    secondVersion: null as number | null,
    secondContent: '',
    secondAvailableVersions: [] as ProcedureVersion[],
    dualPaneMode: false,

    // Validation state
    problems: [] as ValidationError[],
    isValidated: false,
    saving: false,
  }),

  getters: {
    isDirty: (state) => state.currentContent !== state.originalContent,
    hasProblems: (state) => state.problems.length > 0,
    errorCount: (state) => state.problems.filter(p => p.severity === 'error').length,
  },

  actions: {
    setLoading(loading: boolean) {
      this.loading = loading
    },

    setVersionedList(list: Procedure[]) {
      this.versionedList = list
    },

    setCurrentContent(content: string) {
      this.currentContent = content
    },

    setOriginalContent(content: string) {
      this.originalContent = content
    },

    setSelectedProcedure(testName: string | null, version: number | null = null) {
      this.selectedTestName = testName
      this.selectedVersion = version
      this.isValidated = false
      this.problems = []
    },

    setAvailableVersions(versions: ProcedureVersion[]) {
      this.availableVersions = versions
    },

    setSecondPane(testName: string | null, version: number | null = null) {
      this.secondTestName = testName
      this.secondVersion = version
    },

    setSecondContent(content: string) {
      this.secondContent = content
    },

    setSecondAvailableVersions(versions: ProcedureVersion[]) {
      this.secondAvailableVersions = versions
    },

    toggleDualPane() {
      this.dualPaneMode = !this.dualPaneMode
      if (!this.dualPaneMode) {
        this.secondTestName = null
        this.secondVersion = null
        this.secondContent = ''
        this.secondAvailableVersions = []
      }
    },

    setProblems(problems: ValidationError[]) {
      this.problems = problems
    },

    clearProblems() {
      this.problems = []
    },

    setValidated(valid: boolean) {
      this.isValidated = valid
    },

    resetEditor() {
      this.selectedTestName = null
      this.selectedVersion = null
      this.currentContent = ''
      this.originalContent = ''
      this.currentProject = ''
      this.currentCreatedBy = ''
      this.availableVersions = []
      this.problems = []
      this.isValidated = false
    },
  },
})
