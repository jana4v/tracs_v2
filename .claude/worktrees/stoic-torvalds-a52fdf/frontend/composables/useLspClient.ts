// LSP Client composable for connecting Monaco editor to ASTRA Language Server
// Uses monaco-languageclient with WebSocket transport

import type * as Monaco from 'monaco-editor'
import { MonacoLanguageClient } from 'monaco-languageclient'
import type { MessageTransports } from 'vscode-languageclient'
import { CloseAction, ErrorAction } from 'vscode-languageclient'
import 'vscode/localExtensionHost'
import { initEnhancedMonacoEnvironment, initServices } from 'monaco-languageclient/vscode/services'
import { toSocket, WebSocketMessageReader, WebSocketMessageWriter } from 'vscode-ws-jsonrpc'

// Global state to ensure services are only initialized once
let globalServicesInitialized = false
let globalServicesInitPromise: Promise<void> | null = null

export function useLspClient() {
  const config = useRuntimeConfig()
  const lspUrl = config.public.lspUrl
  const toast = useToast()

  const isConnected = ref(false)
  let ws: WebSocket | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let languageClient: MonacoLanguageClient | null = null
  let shouldReconnect = true
  let isConnecting = false
  let connectionAttempts = 0
  let lastConnectionAttempt = 0

  function createLanguageClient(transports: MessageTransports) {
    return new MonacoLanguageClient({
      name: 'ASTRA Language Client',
      clientOptions: {
        documentSelector: [{ language: 'astra' }],
        errorHandler: {
          error: () => ({ action: ErrorAction.Continue }),
          closed: () => ({ action: CloseAction.DoNotRestart }),
        },
      },
      connectionProvider: {
        get: async () => transports,
      },
    })
  }

  async function ensureServicesReady(): Promise<boolean> {
    // Check if already initialized globally AND VSCode API is ready
    if (globalServicesInitialized) {
      const env = initEnhancedMonacoEnvironment()
      if (env.vscodeApiInitialised) {
        return true
      }
      // Services initialized but VSCode API not ready - need to wait
      console.log('[LSP] Services initialized but VSCode API not ready, waiting...')
    }

    // Prevent multiple simultaneous initialization attempts
    if (globalServicesInitPromise) {
      try {
        await globalServicesInitPromise
      } catch (e) {
        console.error('[LSP] Services initialization failed:', e)
        globalServicesInitPromise = null
        return false
      }
    }

    // Start initialization if not already started
    if (!globalServicesInitPromise && !globalServicesInitialized) {
      console.log('[LSP] Initializing Monaco services...')
      globalServicesInitPromise = initServices({ 
        caller: 'astra-frontend',
        enableExtHostWorker: true 
      })
        .then(() => {
          console.log('[LSP] Monaco services initialized')
        })
        .catch((error: any) => {
          const message = String(error?.message || error)
          if (message.includes('Services are already initialized')) {
            console.log('[LSP] Monaco services already initialized')
          } else {
            console.error('[LSP] Failed to initialize Monaco services:', error)
            globalServicesInitPromise = null
            throw error
          }
        })

      try {
        await globalServicesInitPromise
        globalServicesInitialized = true
      } catch {
        return false
      }
    }

    // CRITICAL: Always wait for VSCode API to be ready, even if services are initialized
    const env = initEnhancedMonacoEnvironment()
    if (!env.vscodeApiInitialised) {
      console.log('[LSP] Waiting for VSCode API to initialize...')
      const ready = await new Promise<boolean>((resolve) => {
        const start = Date.now()
        const tick = () => {
          const current = initEnhancedMonacoEnvironment()
          if (current.vscodeApiInitialised) {
            console.log('[LSP] VSCode API is now ready ✓')
            resolve(true)
            return
          }
          if (Date.now() - start > 15000) {
            console.error('[LSP] VSCode API initialization timeout after 15s')
            resolve(false)
            return
          }
          setTimeout(tick, 100)
        }
        tick()
      })
      return ready
    }

    console.log('[LSP] VSCode API already ready ✓')
    return true
  }

  async function connect(_monaco?: typeof Monaco) {
    // Prevent rapid reconnection attempts
    const now = Date.now()
    if (now - lastConnectionAttempt < 2000) {
      console.log('[LSP] Skipping connection attempt - too soon after last attempt')
      return
    }
    lastConnectionAttempt = now

    if (ws?.readyState === WebSocket.OPEN || isConnecting) {
      console.log('[LSP] Already connected or connecting')
      return
    }

    // Ensure Monaco services are ready before connecting
    if (!globalServicesInitialized) {
      console.log('[LSP] Ensuring Monaco services are ready...')
      try {
        const ready = await ensureServicesReady()
        if (!ready) {
          console.warn('[LSP] Monaco services/VSCode API not ready, will retry later')
          scheduleReconnect(10000) // Longer delay for service initialization
          return
        }
      } catch (e) {
        console.error('[LSP] Service initialization failed:', e)
        scheduleReconnect(10000)
        return
      }
    } else {
      // Services were initialized before, but double-check VSCode API is still ready
      const env = initEnhancedMonacoEnvironment()
      if (!env.vscodeApiInitialised) {
        console.warn('[LSP] VSCode API was ready but is not now, re-initializing...')
        globalServicesInitialized = false
        const ready = await ensureServicesReady()
        if (!ready) {
          console.error('[LSP] Failed to re-initialize VSCode API')
          scheduleReconnect(10000)
          return
        }
      }
    }

    shouldReconnect = true
    isConnecting = true
    connectionAttempts++

    console.log(`[LSP] Connecting to language server (attempt ${connectionAttempts})...`)

    try {
      ws = new WebSocket(lspUrl)

      ws.onopen = () => {
        console.log('[LSP] WebSocket opened')
        isConnecting = false
        connectionAttempts = 0 // Reset on successful connection

        // Triple-check VSCode API is ready (should always be true if we got here)
        const env = initEnhancedMonacoEnvironment()
        if (!env.vscodeApiInitialised) {
          console.error('[LSP] CRITICAL: VSCode API not ready after WebSocket open - this should not happen!')
          console.error('[LSP] Environment state:', JSON.stringify(env))
          ws?.close()
          // Don't schedule reconnect immediately - something is wrong
          shouldReconnect = false
          isConnected.value = false
          try {
            toast.add({ 
              severity: 'error', 
              summary: 'LSP Error', 
              detail: 'Monaco editor services failed to initialize. Please reload the page.', 
              life: 10000 
            })
          } catch {
            // Toast might not be available
          }
          return
        }

        isConnected.value = true
        console.log('[LSP] Creating language client...')

        try {
          const socket = toSocket(ws as WebSocket)
          const reader = new WebSocketMessageReader(socket)
          const writer = new WebSocketMessageWriter(socket)
          const transports: MessageTransports = { reader, writer }

          languageClient = createLanguageClient(transports)
          
          languageClient.start()
            .then(() => {
              console.log('[LSP] ✓ Language client started successfully')
            })
            .catch((err) => {
              console.error('[LSP] Language client start failed:', err)
            })

          reader.onClose(() => {
            console.log('[LSP] Reader closed')
            languageClient?.stop().catch(() => {})
          })
        } catch (err) {
          console.error('[LSP] Failed to create language client:', err)
          ws?.close()
        }
      }

      ws.onclose = (event) => {
        console.log(`[LSP] WebSocket closed (code: ${event.code}, reason: ${event.reason})`)
        isConnecting = false
        isConnected.value = false
        
        if (languageClient) {
          languageClient.stop().catch(() => {})
          languageClient = null
        }
        
        ws = null
        
        if (shouldReconnect) {
          // Exponential backoff for reconnection
          const delay = Math.min(5000 * Math.pow(1.5, Math.min(connectionAttempts, 5)), 30000)
          console.log(`[LSP] Will reconnect in ${delay}ms`)
          scheduleReconnect(delay)
        }
      }

      ws.onerror = (error) => {
        console.error('[LSP] WebSocket error:', error)
        isConnecting = false
      }
    } catch (e) {
      console.error('[LSP] Connection failed:', e)
      isConnecting = false
      if (shouldReconnect) {
        scheduleReconnect(5000)
      }
    }
  }

  function disconnect() {
    console.log('[LSP] Disconnecting...')
    shouldReconnect = false
    
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    
    if (languageClient) {
      languageClient.stop().catch(() => {})
      languageClient = null
    }
    
    if (ws) {
      ws.close()
      ws = null
    }
    
    isConnected.value = false
    isConnecting = false
    connectionAttempts = 0
  }

  function scheduleReconnect(delay: number = 5000) {
    if (reconnectTimer || !shouldReconnect) {
      return
    }
    
    console.log(`[LSP] Scheduling reconnect in ${delay}ms`)
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      connect()
    }, delay)
  }

  return {
    isConnected: readonly(isConnected),
    connect,
    disconnect,
  }
}
