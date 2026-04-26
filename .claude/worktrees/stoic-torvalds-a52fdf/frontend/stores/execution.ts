import type { LogEntry, ExecutionResult, StepState } from '~/types/astra'

export type ExecutionStatus = 'idle' | 'running' | 'stepping' | 'completed' | 'error'

export const useExecutionStore = defineStore('execution', {
  state: () => ({
    status: 'idle' as ExecutionStatus,
    sessionId: null as string | null,
    currentLine: 0,
    runDelayMs: 150,
    variables: {} as Record<string, any>,
    callStack: [] as string[],
    log: [] as LogEntry[],
    result: null as ExecutionResult | null,
  }),

  getters: {
    isRunning: (state) => state.status === 'running' || state.status === 'stepping',
    isStepping: (state) => state.status === 'stepping',
    canStep: (state) => state.status === 'stepping',
  },

  actions: {
    setStatus(status: ExecutionStatus) {
      this.status = status
    },

    setSessionId(id: string | null) {
      this.sessionId = id
    },

    setCurrentLine(line: number) {
      this.currentLine = line
    },

    updateStepState(state: StepState) {
      this.currentLine = state.line_number
      this.variables = state.variables
      this.callStack = state.call_stack
      if (state.output) {
        this.addLog(state.output, 'info')
      }
      if (state.status === 'completed') {
        this.status = 'completed'
      } else if (state.status === 'failed') {
        this.status = 'error'
      }
    },

    setResult(result: ExecutionResult) {
      this.result = result
    },

    addLog(message: string, type: LogEntry['type'] = 'ok') {
      this.log.push({
        timestamp: new Date().toLocaleTimeString(),
        message,
        type,
      })
    },

    clearLog() {
      this.log = []
    },

    reset() {
      this.status = 'idle'
      this.sessionId = null
      this.currentLine = 0
      this.runDelayMs = 150
      this.variables = {}
      this.callStack = []
      this.result = null
    },
  },
})
