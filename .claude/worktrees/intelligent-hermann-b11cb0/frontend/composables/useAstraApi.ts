import type {
  ValidationError,
  ExecutionResult,
  StepState,
  TMRef,
  EnrichedTMRef,
  TCRef,
  SCORef,
  TestResult,
  Procedure,
  ProcedureVersion,
  RunnerStartResponse,
  RunnerControlResponse,
  ProcedureRunStatus,
  SatelliteConfig,
  TestPhaseResponse,
  UdTmRow,
  UdTmVersion,
  UdTmChange,
  SimulatorStatus,
  SimulatorUpdateItem,
  BackgroundEntry,
  BackgroundRegisterRequest,
  BackgroundListResponse,
  BackgroundControlResponse,
} from '~/types/astra'

export function useAstraApi() {
  const config = useRuntimeConfig()
  const apiBase = config.public.apiBase as string
  const simulatorBase = (config.public.simulatorBase as string) || 'http://localhost:8091'

  async function apiFetch<T>(path: string, options: RequestInit = {}): Promise<T> {
    const response = await $fetch<T>(`${apiBase}${path}`, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
    } as any)
    return response
  }

  // === Procedure Operations ===

  async function loadProcedure(content: string, filename: string = '<editor>') {
    return apiFetch<{ success: boolean; test_name: string; line_count: number }>('/load', {
      method: 'POST',
      body: JSON.stringify({ content, filename }),
    })
  }

  async function validateProcedure(testName: string) {
    return apiFetch<{ valid: boolean; errors: ValidationError[] }>('/validate', {
      method: 'POST',
      body: JSON.stringify({ test_name: testName }),
    })
  }

  async function runTest(testName: string) {
    return apiFetch<{ success: boolean; status: string; duration: number; log_entries: number }>('/run', {
      method: 'POST',
      body: JSON.stringify({ test_name: testName }),
    })
  }

  // === Step Execution ===

  async function startStepSession(testName: string) {
    return apiFetch<{ success: boolean; session_id: string; state: StepState }>('/step/start', {
      method: 'POST',
      body: JSON.stringify({ test_name: testName }),
    })
  }

  async function stepNext(sessionId: string) {
    return apiFetch<StepState>('/step/next', {
      method: 'POST',
      body: JSON.stringify({ session_id: sessionId }),
    })
  }

  async function stepReset(sessionId: string) {
    return apiFetch<StepState>('/step/reset', {
      method: 'POST',
      body: JSON.stringify({ session_id: sessionId }),
    })
  }

  // === Telemetry ===

  async function getTMValues() {
    return apiFetch<Record<string, any>>('/tm')
  }

  async function getTMMnemonics() {
    return apiFetch<{ mnemonics: string[] }>('/tm/mnemonics')
  }

  // === Mnemonic CRUD (MongoDB) ===

  async function getAllTMMnemonics() {
    return apiFetch<TMRef[]>('/mnemonics/tm')
  }

  async function getTMByBank(bank: number) {
    return apiFetch<TMRef[]>(`/mnemonics/tm/${bank}`)
  }

  async function getAllTCMnemonics() {
    return apiFetch<TCRef[]>('/mnemonics/tc')
  }

  async function getAllSCOCommands() {
    return apiFetch<SCORef[]>('/mnemonics/sco')
  }

  async function getTMSubsystems() {
    return apiFetch<{ subsystems: string[] }>('/telemetry/subsystems')
  }

  async function getTMBySubsystem(subsystem: string) {
    return apiFetch<EnrichedTMRef[]>(`/mnemonics/tm/${encodeURIComponent(subsystem)}`)
  }

  async function addTMMnemonic(mnemonic: TMRef) {
    return apiFetch<{ success: boolean }>('/mnemonics/tm', {
      method: 'POST',
      body: JSON.stringify(mnemonic),
    })
  }

  async function addTCMnemonic(mnemonic: TCRef) {
    return apiFetch<{ success: boolean }>('/mnemonics/tc', {
      method: 'POST',
      body: JSON.stringify(mnemonic),
    })
  }

  async function addSCOCommand(command: SCORef) {
    return apiFetch<{ success: boolean }>('/mnemonics/sco', {
      method: 'POST',
      body: JSON.stringify(command),
    })
  }

  // === Procedures ===

  async function initProcedures() {
    return apiFetch<{ success: boolean; message?: string }>('/procedures/init', {
      method: 'POST',
    })
  }

  async function getProcedures(project?: string, opts?: { limit?: number; offset?: number; tags?: string[]; includeDeleted?: boolean; deletedOnly?: boolean }) {
    const params = new URLSearchParams()
    if (project) params.set('project', project)
    if (opts?.limit != null) params.set('limit', String(opts.limit))
    if (opts?.offset != null) params.set('offset', String(opts.offset))
    if (opts?.tags?.length) params.set('tags', opts.tags.join(','))
    if (opts?.includeDeleted) params.set('include_deleted', 'true')
    if (opts?.deletedOnly) params.set('deleted_only', 'true')
    const query = params.toString() ? `?${params.toString()}` : ''
    return apiFetch<{ procedures: Procedure[] }>(`/procedures${query}`)
  }

  async function restoreProcedure(testName: string, project?: string, restoredBy?: string) {
    const query = project ? `?project=${encodeURIComponent(project)}` : ''
    return apiFetch<{ success: boolean }>(`/procedures/${encodeURIComponent(testName)}/restore${query}`, {
      method: 'POST',
      body: restoredBy != null ? JSON.stringify({ restored_by: restoredBy }) : undefined,
    })
  }

  async function getProcedure(testName: string, project?: string) {
    const query = project ? `?project=${encodeURIComponent(project)}` : ''
    const res = await apiFetch<{ procedure: Procedure }>(`/procedures/${encodeURIComponent(testName)}${query}`)
    return res.procedure
  }

  async function getProcedureVersions(testName: string, project?: string) {
    const query = project ? `?project=${encodeURIComponent(project)}` : ''
    return apiFetch<{ versions: ProcedureVersion[] }>(`/procedures/${encodeURIComponent(testName)}/versions${query}`)
  }

  async function getProcedureVersion(testName: string, version: number, project?: string) {
    const query = project ? `?project=${encodeURIComponent(project)}` : ''
    const res = await apiFetch<{ version: ProcedureVersion }>(`/procedures/${encodeURIComponent(testName)}/versions/${version}${query}`)
    return res.version
  }

  async function saveProcedure(testName: string, content: string, project: string, createdBy: string, description?: string, tags?: string[], changeMessage?: string) {
    return apiFetch<{ saved: boolean; version?: number; reason?: string; message?: string }>('/procedures', {
      method: 'POST',
      body: JSON.stringify({
        test_name: testName,
        content,
        project,
        created_by: createdBy,
        description: description || '',
        tags: tags || [],
        change_message: changeMessage || '',
      }),
    })
  }

  async function deleteProcedure(testName: string, project?: string, deletedBy?: string) {
    const query = project ? `?project=${encodeURIComponent(project)}` : ''
    return apiFetch<{ success: boolean }>(`/procedures/${encodeURIComponent(testName)}${query}`, {
      method: 'DELETE',
      body: deletedBy != null ? JSON.stringify({ deleted_by: deletedBy }) : undefined,
    })
  }

  // === Procedure Runner (parallel execution) ===

  async function runnerStart(testName: string) {
    return apiFetch<RunnerStartResponse>('/runner/start', {
      method: 'POST',
      body: JSON.stringify({ test_name: testName }),
    })
  }

  async function runnerPause(runId: string) {
    return apiFetch<RunnerControlResponse>('/runner/pause', {
      method: 'POST',
      body: JSON.stringify({ run_id: runId }),
    })
  }

  async function runnerResume(runId: string) {
    return apiFetch<RunnerControlResponse>('/runner/resume', {
      method: 'POST',
      body: JSON.stringify({ run_id: runId }),
    })
  }

  async function runnerAbort(runId: string) {
    return apiFetch<RunnerControlResponse>('/runner/abort', {
      method: 'POST',
      body: JSON.stringify({ run_id: runId }),
    })
  }

  async function runnerStatus(runId: string) {
    return apiFetch<ProcedureRunStatus>(`/runner/status/${encodeURIComponent(runId)}`)
  }

  async function runnerList() {
    return apiFetch<{ runs: ProcedureRunStatus[] }>('/runner/list')
  }

  // === Satellite Configuration (Redis) ===

  async function getSatelliteConfig() {
    return apiFetch<SatelliteConfig>('/satellite-config')
  }

  async function setSatelliteConfig(field: string, value: string) {
    return apiFetch<{ success: boolean; field: string; value: string }>('/satellite-config', {
      method: 'POST',
      body: JSON.stringify({ field, value }),
    })
  }

  async function getTestPhase() {
    return apiFetch<TestPhaseResponse>('/satellite-config/test-phase')
  }

  async function setTestPhase(testPhase: string) {
    return apiFetch<{ success: boolean; test_phase: string }>('/satellite-config/test-phase', {
      method: 'POST',
      body: JSON.stringify({ test_phase: testPhase }),
    })
  }

  // === User Defined Telemetry (UD_TM) — single flat table per project ===

  async function getUdTm(project?: string) {
    const query = project ? `?project=${encodeURIComponent(project)}` : ''
    return apiFetch<{ rows: UdTmRow[]; latest_version: number; _id?: string; updated_at?: string }>(`/ud-tm${query}`)
  }

  async function saveUdTm(rows: UdTmRow[], project: string, createdBy: string, changeMessage?: string, changes?: UdTmChange[]) {
    return apiFetch<{ saved: boolean; version?: number; message?: string }>('/ud-tm', {
      method: 'POST',
      body: JSON.stringify({
        rows,
        project,
        created_by: createdBy,
        change_message: changeMessage || '',
        changes: changes || [],
      }),
    })
  }

  async function getUdTmVersions(project?: string) {
    const query = project ? `?project=${encodeURIComponent(project)}` : ''
    return apiFetch<{ versions: UdTmVersion[] }>(`/ud-tm/versions${query}`)
  }

  async function getUdTmVersion(version: number, project?: string) {
    const query = project ? `?project=${encodeURIComponent(project)}` : ''
    return apiFetch<{ version: UdTmVersion }>(`/ud-tm/versions/${version}${query}`)
  }

  // === TM Import ===

  async function uploadTelemetryFile(file: File) {
    // Send as base64 JSON to avoid Genie.jl multipart parser bug
    // (files starting with '--' like .out files trigger false boundary detection)
    const base64 = await new Promise<string>((resolve, reject) => {
      const reader = new FileReader()
      reader.onload = () => resolve((reader.result as string).split(',')[1])
      reader.onerror = reject
      reader.readAsDataURL(file)
    })
    const res = await fetch(`${apiBase}/telemetry/upload`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ filename: file.name, data: base64 }),
    })
    const data = await res.json()
    if (!res.ok || !data.success) {
      throw new Error(data.error || `Upload failed (${res.status})`)
    }
    return data as {
      success: boolean
      filename: string
      stats: { total: number; inserted: number; updated: number; skipped: number; errors: string[] }
    }
  }

  // === Test Results ===

  async function getResults(limit: number = 50) {
    return apiFetch<{ results: TestResult[] }>(`/results?limit=${limit}`)
  }

  async function getResult(id: string) {
    return apiFetch<TestResult>(`/results/${id}`)
  }

  // === TM Simulator (Go service on separate port) ===

  async function getSimulatorMnemonics() {
    const res = await $fetch<any[]>(`${simulatorBase}/simulator/mnemonics`)
    return res
  }

  async function getMnemonicRange(mnemonic: string) {
    const res = await $fetch<{ mnemonic: string; type: string; range: string[] }>(
      `${simulatorBase}/simulator/mnemonic/range?mnemonic=${encodeURIComponent(mnemonic)}`
    )
    return res
  }

  async function getSimulatorValues(subsystems?: string[]) {
    const query = subsystems && subsystems.length > 0
      ? `?subsystem=${encodeURIComponent(subsystems.join(','))}`
      : ''
    const res = await $fetch<SimulatorUpdateItem[]>(`${simulatorBase}/simulator/values${query}`)
    return res
  }

  async function getSimulatorSubsystems() {
    const res = await $fetch<string[]>(`${simulatorBase}/simulator/subsystems`)
    return res
  }

  async function getSimulatorMode() {
    const res = await $fetch<{ mode: string }>(`${simulatorBase}/simulator/mode`)
    return res
  }

  async function setSimulatorMode(mode: string) {
    const res = await $fetch<{ success: boolean; mode: string }>(`${simulatorBase}/simulator/mode`, {
      method: 'PUT',
      body: { mode },
    })
    return res
  }

  async function startSimulator() {
    const res = await $fetch<{ success: boolean; running: boolean }>(`${simulatorBase}/simulator/start`, {
      method: 'POST',
    })
    return res
  }

  async function stopSimulator() {
    const res = await $fetch<{ success: boolean; running: boolean }>(`${simulatorBase}/simulator/stop`, {
      method: 'POST',
    })
    return res
  }

  async function getSimulatorStatus() {
    const res = await $fetch<SimulatorStatus>(`${simulatorBase}/simulator-status`)
    return res
  }

  async function updateSimulatorValues(items: SimulatorUpdateItem[]) {
    const res = await $fetch<{ success: boolean; updated: number }>(`${simulatorBase}/simulator/values`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: items,
    })
    return res
  }

  async function resetSimulator() {
    const res = await $fetch<{ success: boolean; message: string }>(`${simulatorBase}/simulator/reset`, {
      method: 'POST',
    })
    return res
  }

  // === Background Scheduler ===

  async function backgroundRegister(req: BackgroundRegisterRequest) {
    return apiFetch<BackgroundControlResponse>('/background/register', {
      method: 'POST',
      body: JSON.stringify(req),
    })
  }

  async function backgroundRemove(procName: string) {
    return apiFetch<BackgroundControlResponse>('/background/remove', {
      method: 'DELETE',
      body: JSON.stringify({ proc_name: procName }),
    })
  }

  async function backgroundStart(procName: string) {
    return apiFetch<BackgroundControlResponse>('/background/start', {
      method: 'POST',
      body: JSON.stringify({ proc_name: procName }),
    })
  }

  async function backgroundStop(procName: string) {
    return apiFetch<BackgroundControlResponse>('/background/stop', {
      method: 'POST',
      body: JSON.stringify({ proc_name: procName }),
    })
  }

  async function backgroundStartAll() {
    return apiFetch<BackgroundControlResponse>('/background/start-all', { method: 'POST' })
  }

  async function backgroundStopAll() {
    return apiFetch<BackgroundControlResponse>('/background/stop-all', { method: 'POST' })
  }

  async function backgroundList() {
    return apiFetch<BackgroundListResponse>('/background/list')
  }

  async function backgroundStatus(procName: string) {
    return apiFetch<BackgroundEntry>(`/background/status/${encodeURIComponent(procName)}`)
  }

  return {
    loadProcedure,
    validateProcedure,
    runTest,
    startStepSession,
    stepNext,
    stepReset,
    getTMValues,
    getTMMnemonics,
    getAllTMMnemonics,
    getTMByBank,
    getAllTCMnemonics,
    getAllSCOCommands,
    getTMSubsystems,
    getTMBySubsystem,
    addTMMnemonic,
    addTCMnemonic,
    addSCOCommand,
    getResults,
    getResult,
    initProcedures,
    getProcedures,
    getProcedure,
    getProcedureVersions,
    getProcedureVersion,
    saveProcedure,
    deleteProcedure,
    restoreProcedure,
    runnerStart,
    runnerPause,
    runnerResume,
    runnerAbort,
    runnerStatus,
    runnerList,
    getSatelliteConfig,
    setSatelliteConfig,
    getTestPhase,
    setTestPhase,
    getUdTm,
    saveUdTm,
    getUdTmVersions,
    getUdTmVersion,
    uploadTelemetryFile,
    getSimulatorMnemonics,
    getMnemonicRange,
    getSimulatorValues,
    getSimulatorSubsystems,
    getSimulatorStatus,
    getSimulatorMode,
    setSimulatorMode,
    startSimulator,
    stopSimulator,
    updateSimulatorValues,
    resetSimulator,
    backgroundRegister,
    backgroundRemove,
    backgroundStart,
    backgroundStop,
    backgroundStartAll,
    backgroundStopAll,
    backgroundList,
    backgroundStatus,
  }
}
