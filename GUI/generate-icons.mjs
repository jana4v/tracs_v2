/**
 * One-time script to generate PWA icons for TRACS Nova.
 * Uses the source GUI's sharp installation (Node-version-compatible).
 * Run: node generate-icons.mjs
 */
import { createRequire } from 'node:module'
import { fileURLToPath } from 'node:url'
import { dirname, resolve } from 'node:path'
import { readFileSync, mkdirSync } from 'node:fs'

const __dirname = dirname(fileURLToPath(import.meta.url))

// Use the working sharp from the source GUI node_modules
const guiSharpPath = resolve(__dirname, '../../GUI/node_modules/sharp/lib/index.js')
const require = createRequire(guiSharpPath)
const sharp = require(guiSharpPath)

const svgPath = resolve(__dirname, 'public/icon.svg')
const svgBuffer = readFileSync(svgPath)
const outDir = resolve(__dirname, 'public')
mkdirSync(outDir, { recursive: true })

const icons = [
  { name: 'favicon.ico',              size: 32  },
  { name: 'pwa-64x64.png',            size: 64  },
  { name: 'pwa-192x192.png',          size: 192 },
  { name: 'pwa-512x512.png',          size: 512 },
  { name: 'maskable-icon-512x512.png',size: 512 },
  { name: 'apple-touch-icon.png',     size: 180 },
]

for (const { name, size } of icons) {
  const outPath = resolve(outDir, name)
  await sharp(svgBuffer)
    .resize(size, size)
    .png()
    .toFile(outPath)
  console.log(`✓ Generated ${name} (${size}x${size})`)
}

console.log('\nAll PWA icons generated in public/')
