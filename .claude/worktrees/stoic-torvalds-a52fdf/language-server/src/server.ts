// ASTRA Language Server
// WebSocket-based LSP server for browser Monaco Editor integration

import { WebSocketServer, WebSocket } from 'ws'
import {
  createConnection,
  TextDocuments,
  ProposedFeatures,
  TextDocumentSyncKind,
  type InitializeResult,
} from 'vscode-languageserver/node.js'
import { TextDocument } from 'vscode-languageserver-textdocument'
import { registerDiagnostics } from './providers/diagnostics.js'
import { registerCompletion } from './providers/completion.js'
import { registerHover } from './providers/hover.js'
import { registerDocumentSymbols } from './providers/document-symbols.js'
import { disconnect as disconnectMongo } from './utils/backend-client.js'

const LSP_PORT = parseInt(process.env.LSP_PORT || '3001')

// Create WebSocket server
const wss = new WebSocketServer({ port: LSP_PORT })
console.log(`[ASTRA LSP] Language server listening on ws://localhost:${LSP_PORT}`)

wss.on('connection', (socket: WebSocket) => {
  console.log('[ASTRA LSP] Client connected')

  // Create LSP message reader/writer from WebSocket
  const messageQueue: string[] = []
  let messageHandler: ((msg: string) => void) | null = null

  socket.on('message', (data) => {
    const msg = data.toString()
    if (messageHandler) {
      messageHandler(msg)
    } else {
      messageQueue.push(msg)
    }
  })

  // Custom reader/writer for WebSocket transport
  const reader = {
    listen(callback: (msg: any) => void) {
      messageHandler = (raw: string) => {
        try {
          const msg = JSON.parse(raw)
          callback(msg)
        } catch (e) {
          console.error('[ASTRA LSP] Failed to parse message:', e)
        }
      }
      // Drain queued messages
      while (messageQueue.length > 0) {
        const msg = messageQueue.shift()!
        messageHandler(msg)
      }
    },
    dispose() {},
    onError() { return { dispose() {} } },
    onClose() { return { dispose() {} } },
    onPartialMessage() { return { dispose() {} } },
  }

  const writer = {
    write(msg: any) {
      try {
        if (socket.readyState === WebSocket.OPEN) {
          socket.send(JSON.stringify(msg))
        }
      } catch (e) {
        console.error('[ASTRA LSP] Failed to send message:', e)
      }
      return Promise.resolve()
    },
    end() {},
    dispose() {},
    onError() { return { dispose() {} } },
    onClose() { return { dispose() {} } },
  }

  // Create LSP connection
  const connection = createConnection(ProposedFeatures.all, reader as any, writer as any)
  const documents = new TextDocuments(TextDocument)

  // Initialize
  connection.onInitialize((): InitializeResult => {
    console.log('[ASTRA LSP] Initializing...')
    return {
      capabilities: {
        textDocumentSync: TextDocumentSyncKind.Full,
        completionProvider: {
          triggerCharacters: ['.', ' '],
          resolveProvider: false,
        },
        hoverProvider: true,
        documentSymbolProvider: true,
      },
    }
  })

  connection.onInitialized(() => {
    console.log('[ASTRA LSP] Initialized successfully')
  })

  // Register all providers
  registerDiagnostics(connection, documents)
  registerCompletion(connection, documents)
  registerHover(connection, documents)
  registerDocumentSymbols(connection, documents)

  // Start listening
  documents.listen(connection)
  connection.listen()

  socket.on('close', () => {
    console.log('[ASTRA LSP] Client disconnected')
    connection.dispose()
  })

  socket.on('error', (err) => {
    console.error('[ASTRA LSP] Socket error:', err)
  })
})

// Graceful shutdown
process.on('SIGINT', async () => {
  console.log('[ASTRA LSP] Shutting down...')
  await disconnectMongo()
  wss.close()
  process.exit(0)
})

process.on('SIGTERM', async () => {
  await disconnectMongo()
  wss.close()
  process.exit(0)
})
