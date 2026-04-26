import type {
  ProcedureRunStatus,
  RunStatus,
  SchedulerGridRow,
  SchedulerPriority,
  Procedure,
} from '~/types/astra'

function derivePriorityLabel(priority: number): SchedulerPriority {
  if (priority <= 3) return 'high'
  if (priority <= 6) return 'medium'
  return 'low'
}

export const useSchedulerStore = defineStore('scheduler', {
  state: () => ({
    rows: [] as SchedulerGridRow[],
    isExecuting: false,
    currentPriority: null as number | null,
    availableProcedures: [] as Procedure[],
    testPhase: '' as string,
    loading: false,
  }),

  getters: {
    priorityGroups: (state): number[] => {
      return [...new Set(state.rows.map(r => r.priority))].sort((a, b) => a - b)
    },

    activeRunCount: (state): number => {
      return state.rows.filter(r =>
        r.status === 'running' || r.status === 'paused' || r.status === 'queued',
      ).length
    },

    rowsByPriority: (state): Record<number, SchedulerGridRow[]> => {
      const groups: Record<number, SchedulerGridRow[]> = {}
      for (const row of state.rows) {
        if (!groups[row.priority]) groups[row.priority] = []
        groups[row.priority].push(row)
      }
      return groups
    },

    allCompleted: (state): boolean => {
      if (state.rows.length === 0) return false
      return state.rows.every(r =>
        r.status === 'completed' || r.status === 'failed' || r.status === 'aborted',
      )
    },

    completedCount: (state): number =>
      state.rows.filter(r => r.status === 'completed').length,

    failedCount: (state): number =>
      state.rows.filter(r => r.status === 'failed').length,

    pendingCount: (state): number =>
      state.rows.filter(r => r.status === 'pending').length,
  },

  actions: {
    addRow(procedureName: string, priority: number = 1, operator: string = '') {
      if (this.rows.some(r => r.procedure_name === procedureName)) return

      this.rows.push({
        id: `${procedureName}-${Date.now()}`,
        procedure_name: procedureName,
        priority,
        priority_label: derivePriorityLabel(priority),
        status: 'pending',
        progress: 0,
        operator,
        start_time: null,
        end_time: null,
        run_id: null,
        queue_position: null,
        procedure_content: '',
        current_line: 0,
        total_lines: 0,
        error: null,
        step_mode: false,
        test_phase: this.testPhase,
      })

      this.recalculateQueuePositions()
    },

    removeRow(procedureName: string) {
      const row = this.rows.find(r => r.procedure_name === procedureName)
      if (row && (row.status === 'running' || row.status === 'paused')) return
      this.rows = this.rows.filter(r => r.procedure_name !== procedureName)
      this.recalculateQueuePositions()
    },

    setPriority(procedureName: string, priority: number) {
      const row = this.rows.find(r => r.procedure_name === procedureName)
      if (row) {
        row.priority = priority
        row.priority_label = derivePriorityLabel(priority)
      }
    },

    setOperator(procedureName: string, operator: string) {
      const row = this.rows.find(r => r.procedure_name === procedureName)
      if (row) row.operator = operator
    },

    toggleStepMode(procedureName: string) {
      const row = this.rows.find(r => r.procedure_name === procedureName)
      if (row) row.step_mode = !row.step_mode
    },

    setProcedureContent(procedureName: string, content: string) {
      const row = this.rows.find(r => r.procedure_name === procedureName)
      if (row) row.procedure_content = content
    },

    handleRunnerUpdate(data: {
      run_id: string
      procedure: string
      status: string
      current_line?: number
      total_lines?: number
      duration?: number
      error?: string
      started_at?: string
      finished_at?: string
      log_entries?: number
      [key: string]: any
    }) {
      const row = this.rows.find(r => r.run_id === data.run_id)
      if (!row) return

      row.status = data.status as RunStatus
      if (data.current_line !== undefined) row.current_line = data.current_line
      if (data.total_lines !== undefined) row.total_lines = data.total_lines
      if (data.error) row.error = data.error
      if (data.started_at) row.start_time = data.started_at
      if (data.finished_at) row.end_time = data.finished_at

      if (row.total_lines > 0 && row.current_line > 0) {
        row.progress = Math.round((row.current_line / row.total_lines) * 100)
      }

      if (['completed', 'failed', 'aborted'].includes(data.status)) {
        row.queue_position = null
        if (data.finished_at) row.end_time = data.finished_at
        if (data.status === 'completed') row.progress = 100
      }
    },

    setRunId(procedureName: string, runId: string) {
      const row = this.rows.find(r => r.procedure_name === procedureName)
      if (row) {
        row.run_id = runId
        row.status = 'running'
        row.start_time = new Date().toISOString()
        row.queue_position = null
      }
    },

    setEntryStatus(procedureName: string, status: SchedulerGridRow['status']) {
      const row = this.rows.find(r => r.procedure_name === procedureName)
      if (row) row.status = status
    },

    recalculateQueuePositions() {
      const pending = this.rows
        .filter(r => r.status === 'pending')
        .sort((a, b) => a.priority - b.priority)
      pending.forEach((row, index) => {
        row.queue_position = index + 1
      })
    },

    resetAll() {
      for (const row of this.rows) {
        row.run_id = null
        row.status = 'pending'
        row.progress = 0
        row.start_time = null
        row.end_time = null
        row.current_line = 0
        row.error = null
      }
      this.isExecuting = false
      this.currentPriority = null
      this.recalculateQueuePositions()
    },

    clearAll() {
      this.rows = []
      this.isExecuting = false
      this.currentPriority = null
    },

    async restoreFromBackend() {
      const api = useAstraApi()
      try {
        const { runs } = await api.runnerList()
        for (const run of runs) {
          const existingRow = this.rows.find(r => r.run_id === run.run_id)
          if (existingRow) {
            this.handleRunnerUpdate({
              run_id: run.run_id,
              procedure: run.procedure,
              status: run.status,
              current_line: run.current_line,
              total_lines: run.total_lines,
              started_at: run.started_at,
              finished_at: run.finished_at,
              error: run.error,
              duration: run.duration,
            })
          } else if (!['completed', 'failed', 'aborted'].includes(run.status)) {
            this.rows.push({
              id: `restored-${run.run_id}`,
              procedure_name: run.procedure,
              priority: 1,
              priority_label: 'medium',
              status: run.status as any,
              progress: run.total_lines ? Math.round(((run.current_line || 0) / run.total_lines) * 100) : 0,
              operator: '',
              start_time: run.started_at,
              end_time: run.finished_at,
              run_id: run.run_id,
              queue_position: null,
              procedure_content: '',
              current_line: run.current_line || 0,
              total_lines: run.total_lines || 0,
              error: run.error,
              step_mode: false,
              test_phase: '',
            })
          }
        }
      } catch (e) {
        console.warn('[Scheduler] Failed to restore from backend:', e)
      }
    },
  },
})
