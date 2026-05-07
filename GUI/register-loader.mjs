// Registers our custom resolve loader in Node 20/22 using stable `register()` API
import { register } from 'node:module'
import { pathToFileURL } from 'node:url'

// Resolve relative to project root
register('./loader.mjs', pathToFileURL('./'))

// Nitro's dev proxy (httpxy) doesn't handle the case where the browser cancels
// an in-flight proxied request — the upstream response keeps streaming into a
// closed socket and surfaces as an unhandled rejection that crashes the dev
// server. Swallow only those benign disconnect codes; let everything else throw.
const BENIGN_SOCKET_CODES = new Set(['ECONNABORTED', 'ECONNRESET', 'EPIPE'])

function getBenignSocketCode(reason) {
  if (!reason || typeof reason !== 'object')
    return undefined

  const candidates = [
    reason.code,
    reason.cause?.code,
    reason.error?.code,
  ]

  for (const code of candidates) {
    if (code && BENIGN_SOCKET_CODES.has(code))
      return code
  }

  const message = [
    reason.message,
    reason.cause?.message,
    reason.error?.message,
  ].find(Boolean)

  if (typeof message === 'string') {
    for (const code of BENIGN_SOCKET_CODES) {
      if (message.includes(code))
        return code
    }
  }

  return undefined
}

process.on('unhandledRejection', (reason) => {
  const code = getBenignSocketCode(reason)
  if (code)
    return
  throw reason
})
