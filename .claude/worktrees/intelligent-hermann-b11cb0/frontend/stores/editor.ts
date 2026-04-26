import type { ValidationError } from '~/types/astra'

export const useEditorStore = defineStore('editor', {
  state: () => ({
    content: `TEST_NAME example-test
PRE_TEST_REQ TM1.xyz_sts == "on" AND TM1.abc > 20
SEND START_RW
WAIT 5

# Inline Julia code
adjusted = TM1.abc + 10
println("Adjusted value: ", adjusted)

IF adjusted > 50
    SEND START_RW
ELSE
    ALERT_MSG "Value too low"
END

FOR i IN 1 TO 3
    SEND RAMP_RW_$(i)
    WAIT 2
END

CHECK TM1.RW_SPEED <= 100
`,
    fileName: null as string | null,
    testName: null as string | null,
    isDirty: false,
    problems: [] as ValidationError[],
  }),

  getters: {
    hasProblems: (state) => state.problems.length > 0,
    errorCount: (state) => state.problems.filter(p => p.severity === 'error').length,
    warningCount: (state) => state.problems.filter(p => p.severity === 'warning').length,
  },

  actions: {
    setContent(content: string) {
      this.content = content
      this.isDirty = true
    },

    setTestName(name: string) {
      this.testName = name
    },

    setFileName(name: string | null) {
      this.fileName = name
    },

    setProblems(problems: ValidationError[]) {
      this.problems = problems
    },

    clearProblems() {
      this.problems = []
    },

    markClean() {
      this.isDirty = false
    },

    reset() {
      this.content = ''
      this.fileName = null
      this.testName = null
      this.isDirty = false
      this.problems = []
    },
  },
})
