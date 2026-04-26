import type { WebSocketEvent } from '~/types/astra'

export function useWebSocket() {
  const config = useRuntimeConfig()
  const wsUrl = config.public.wsUrl

  const isConnected = ref(false)
  const ws = ref<WebSocket | null>(null)
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let reconnectDelay = 1000

  const executionStore = useExecutionStore()
  const telemetryStore = useTelemetryStore()
  const schedulerStore = useSchedulerStore()

  function connect() {
    if (ws.value?.readyState === WebSocket.OPEN) return

    try {
      ws.value = new WebSocket(wsUrl)

      ws.value.onopen = () => {
        isConnected.value = true
        reconnectDelay = 1000
        console.log('[WS] Connected to ASTRA backend')
      }

      ws.value.onmessage = (event) => {
        try {
          const msg: WebSocketEvent = JSON.parse(event.data)
          handleEvent(msg)
        } catch (e) {
          console.error('[WS] Failed to parse message:', e)
        }
      }

      ws.value.onclose = () => {
        isConnected.value = false
        console.log('[WS] Disconnected, reconnecting...')
        scheduleReconnect()
      }

      ws.value.onerror = (error) => {
        console.error('[WS] Error:', error)
      }
    } catch (e) {
      console.error('[WS] Connection failed:', e)
      scheduleReconnect()
    }
  }

  function disconnect() {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    if (ws.value) {
      ws.value.close()
      ws.value = null
    }
    isConnected.value = false
  }

  function scheduleReconnect() {
    if (reconnectTimer) return
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      reconnectDelay = Math.min(reconnectDelay * 2, 30000)
      connect()
    }, reconnectDelay)
  }

  function handleEvent(event: WebSocketEvent) {
    switch (event.type) {
      case 'step_update':
        executionStore.updateStepState(event.data)
        break

      case 'test_complete':
        executionStore.setResult(event.data)
        executionStore.setStatus('completed')
        executionStore.addLog(
          `Test ${event.data.test_name}: ${event.data.status} (${event.data.duration?.toFixed(2)}s)`,
          event.data.status === 'passed' ? 'ok' : 'error',
        )
        break

      case 'tm_update':
        telemetryStore.setBanks(event.data)
        break

      case 'alert':
        executionStore.addLog(event.data.message, 'warning')
        break

      case 'runner_update':
        schedulerStore.handleRunnerUpdate(event.data)
        break

      case 'connected':
        console.log('[WS] Server greeting:', event.data.message)
        break
    }
  }

  function send(data: any) {
    if (ws.value?.readyState === WebSocket.OPEN) {
      ws.value.send(JSON.stringify(data))
    }
  }

  return {
    isConnected: readonly(isConnected),
    connect,
    disconnect,
    send,
  }
}
