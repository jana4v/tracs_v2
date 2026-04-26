// ASTRA TypeScript type definitions

export interface TMRef {
  bank: number
  mnemonic: string
  full_ref: string
  description: string
  data_type: 'string' | 'number' | 'boolean'
  unit?: string
  range_min?: number | null
  range_max?: number | null
  enum_values?: string[]
  subsystem: string
}

export interface EnrichedTMRef {
  _id: string
  subsystem: string
  cdbPidNo: string
  cdbMnemonic: string
  type: 'ANALOG' | 'BINARY' | 'DECIMAL' | string
  processingType: string
  samplingRate: number | string
  range: number[] | string[] | string
  resolutionA1: number | string
  offsetA0: number | string
  unit: string
  tolerance: number | string
  digitalStatus: string
  description: string
  [key: string]: any
}

export interface TCRef {
  command: string
  full_ref: string
  description: string
  parameters: TCParameter[]
  subsystem: string
  category?: string
}

export interface TCParameter {
  name: string
  type: string
  required: boolean
  default?: any
}

export interface SCORef {
  command: string
  full_ref: string
  description: string
  parameters: { name: string; type: string }[]
  subsystem: string
  category?: string
}

// Procedure types - ISO 8601 timestamps (UTC)
export interface Procedure {
  _id: string
  test_name: string
  project: string
  description: string
  tags: string[]
  latest_version: number
  latest_content?: string
  /** First ~200 chars of content for list preview */
  preview?: string
  is_deleted: boolean
  created_by?: string
  updated_by?: string
  created_at: string
  updated_at: string
  deleted_at?: string | null
  deleted_by?: string
}

export interface ProcedureVersion {
  _id: string
  procedure_id: string
  version: number
  content: string
  project: string
  created_by: string
  created_at: string
  /** Optional reason for this version (save API). */
  change_message?: string
}

export interface ValidationError {
  file: string
  line_number: number
  line_text: string
  message: string
  suggestion: string
  severity: 'error' | 'warning'
}

export interface ExecutionResult {
  test_name: string
  status: 'passed' | 'failed' | 'aborted' | 'error'
  duration_seconds: number
  log_entries: number
}

export interface StepState {
  session_id: string
  test_name: string
  line_number: number
  line_text: string
  status: 'ready' | 'running' | 'paused' | 'completed' | 'failed' | 'error'
  variables: Record<string, any>
  call_stack: string[]
  output: string
}

export interface LogEntry {
  timestamp: string
  message: string
  type: 'ok' | 'error' | 'warning' | 'info'
}

export interface TestResult {
  _id?: string
  test_name: string
  status: 'passed' | 'failed' | 'aborted' | 'error'
  duration_seconds: number
  started_at: string
  completed_at: string
  mode: 'simulation' | 'hardware'
  log_entries: TestLogEntry[]
  error?: string | null
  variables_snapshot: Record<string, any>
}

export interface TestLogEntry {
  timestamp: string
  line_number: number
  statement: string
  result: string
  status: 'passed' | 'failed' | 'skipped'
}

export interface WebSocketEvent {
  type: 'connected' | 'step_update' | 'test_complete' | 'tm_update' | 'alert' | 'runner_update'
  data: any
}

// === Procedure Runner (parallel execution) ===

export interface RunnerStartResponse {
  success: boolean
  run_id: string
  procedure: string
  status: string
}

export interface RunnerControlResponse {
  success: boolean
  run_id: string
  status: string
}

export type RunStatus = 'running' | 'paused' | 'completed' | 'failed' | 'aborted'

export interface ProcedureRunStatus {
  run_id: string
  procedure: string
  status: RunStatus
  started_at: string
  finished_at: string | null
  error: string | null
  current_line?: number
  total_lines?: number
  log_entries?: number
  duration?: number
}

export interface SchedulerEntry {
  procedure_name: string
  priority: number
  run_id: string | null
  status: RunStatus | 'pending' | 'queued'
}

// === AG Grid Scheduler ===

export type SchedulerPriority = 'high' | 'medium' | 'low'

export interface SchedulerGridRow {
  id: string
  procedure_name: string
  priority: number
  priority_label: SchedulerPriority
  status: RunStatus | 'pending' | 'queued'
  progress: number
  operator: string
  start_time: string | null
  end_time: string | null
  run_id: string | null
  queue_position: number | null
  procedure_content: string
  current_line: number
  total_lines: number
  error: string | null
  step_mode: boolean
  test_phase: string
}

export interface SatelliteConfig {
  test_phase: string
  [key: string]: string
}

export interface TestPhaseResponse {
  test_phase: string
}

// === User Defined Telemetry (UD_TM) — single flat table per project ===

export interface UdTmRow {
  row_number: number
  mnemonic: string
  value: string
  range: string
  limit: string
  tolerance: string
}

export interface UdTmDocument {
  _id: string
  project: string
  latest_version: number
  rows: UdTmRow[]
  updated_at: string
}

export interface UdTmVersion {
  _id: string
  ud_tm_id: string
  version: number
  rows: UdTmRow[]
  created_by: string
  created_at: string
  change_message: string
  changes: UdTmChange[]
}

export interface UdTmChange {
  type: 'added' | 'modified' | 'deleted' | 'reordered'
  row_number: number
  mnemonic: string
  field?: string
  old_value?: string
  new_value?: string
}

// === TM Simulator ===
//
export interface SimulatorMnemonic {
  mnemonic: string
  value: string
  range?: string[]
  type?: string
}

export interface SimulatorStatus {
  config: Record<string, string>
  mnemonic_count: number
}

export interface SimulatorUpdateItem {
  mnemonic: string
  value: string
}

// === Background Scheduler ===

export type BackgroundStatus = 'running' | 'idle' | 'failed' | 'stopped'
export type BackgroundScheduleType = 'interval' | 'event'

export interface BackgroundScheduleInfo {
  type: BackgroundScheduleType
  // interval
  interval_seconds?: number
  // event-driven
  condition?: string
  poll_interval?: number
  // shared
  restart_on_failure: boolean
  max_consecutive_failures: number
}

export interface BackgroundEntry {
  proc_name: string
  schedule: BackgroundScheduleInfo
  status: BackgroundStatus
  last_run_at: string | null
  next_run_at: string | null
  last_result: string
  total_runs: number
  error_count: number
  consecutive_failures: number
  last_error: string | null
}

export interface BackgroundRegisterRequest {
  proc_name: string
  schedule_type: BackgroundScheduleType
  interval_seconds?: number
  condition?: string
  poll_interval?: number
  restart_on_failure?: boolean
  max_consecutive_failures?: number
}

export interface BackgroundListResponse {
  entries: BackgroundEntry[]
  count: number
}

export interface BackgroundControlResponse {
  success: boolean
  proc_name?: string
  started?: number
  stopped?: number
}

export interface TmMnemonic {
  _id: string
  subsystem: string
  type: string
  processingType: string
  range?: string[]
  tolerance: number
  unit: string
  digitalStatus: string
  cdbMnemonic: string
  enableComparison: boolean
  enableLimit: boolean
  enableStorage: boolean
}
