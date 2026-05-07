import { spawn } from 'node:child_process'
import { createRequire } from 'node:module'
import { dirname, resolve } from 'node:path'
import { existsSync } from 'node:fs'
import { fileURLToPath, pathToFileURL } from 'node:url'

const require = createRequire(import.meta.url)
const projectRoot = dirname(fileURLToPath(import.meta.url))

let nuxtBinPath

try {
  const nuxtPackageJsonPath = require.resolve('nuxt/package.json')
  nuxtBinPath = resolve(dirname(nuxtPackageJsonPath), 'bin/nuxt.mjs')
} catch (_error) {
  console.error('Unable to resolve "nuxt/bin/nuxt.mjs". Run "yarn install" in GUI first.')
  process.exit(1)
}

if (!existsSync(nuxtBinPath)) {
  console.error(`Resolved Nuxt package, but CLI entry is missing: ${nuxtBinPath}`)
  process.exit(1)
}

const registerLoaderPath = resolve(projectRoot, 'register-loader.mjs')
const registerLoaderUrl = pathToFileURL(registerLoaderPath).href
const args = ['--import', registerLoaderUrl, nuxtBinPath, ...process.argv.slice(2)]

const child = spawn(process.execPath, args, {
  cwd: projectRoot,
  stdio: 'inherit',
  env: process.env,
})

child.on('exit', (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal)
    return
  }

  process.exit(code ?? 1)
})
