import { defineStore } from 'pinia'
import type { BackgroundEntry, BackgroundRegisterRequest } from '~/types/astra'

export const useBackgroundStore = defineStore('background', () => {
  const api = useAstraApi()

  const entries = ref<BackgroundEntry[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  // ── Getters ──────────────────────────────────────────────

  const runningCount = computed(() => entries.value.filter(e => e.status === 'running').length)
  const idleCount = computed(() => entries.value.filter(e => e.status === 'idle').length)
  const failedCount = computed(() => entries.value.filter(e => e.status === 'failed').length)
  const stoppedCount = computed(() => entries.value.filter(e => e.status === 'stopped').length)
  const totalCount = computed(() => entries.value.length)

  const intervalEntries = computed(() =>
    entries.value.filter(e => e.schedule.type === 'interval'),
  )
  const eventEntries = computed(() =>
    entries.value.filter(e => e.schedule.type === 'event'),
  )

  // ── Actions ───────────────────────────────────────────────

  async function fetchList() {
    loading.value = true
    error.value = null
    try {
      const res = await api.backgroundList()
      entries.value = res.entries
    }
    catch (e: any) {
      error.value = e?.message ?? 'Failed to load background procedures'
    }
    finally {
      loading.value = false
    }
  }

  async function register(req: BackgroundRegisterRequest) {
    await api.backgroundRegister(req)
    await fetchList()
  }

  async function remove(procName: string) {
    await api.backgroundRemove(procName)
    entries.value = entries.value.filter(e => e.proc_name !== procName)
  }

  async function startOne(procName: string) {
    await api.backgroundStart(procName)
    const entry = entries.value.find(e => e.proc_name === procName)
    if (entry) entry.status = 'running'
  }

  async function stopOne(procName: string) {
    await api.backgroundStop(procName)
    const entry = entries.value.find(e => e.proc_name === procName)
    if (entry) entry.status = 'stopped'
  }

  async function startAll() {
    await api.backgroundStartAll()
    await fetchList()
  }

  async function stopAll() {
    await api.backgroundStopAll()
    await fetchList()
  }

  /** Called by the WebSocket handler when a `background_update` event arrives. */
  function handleWsUpdate(data: BackgroundEntry) {
    const idx = entries.value.findIndex(e => e.proc_name === data.proc_name)
    if (idx >= 0) {
      entries.value[idx] = { ...entries.value[idx], ...data }
    }
    else {
      entries.value.push(data)
    }
  }

  return {
    entries,
    loading,
    error,
    runningCount,
    idleCount,
    failedCount,
    stoppedCount,
    totalCount,
    intervalEntries,
    eventEntries,
    fetchList,
    register,
    remove,
    startOne,
    stopOne,
    startAll,
    stopAll,
    handleWsUpdate,
  }
})
