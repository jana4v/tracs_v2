// Registers our custom resolve loader in Node 20/22 using stable `register()` API
import { register } from 'node:module'
import { pathToFileURL } from 'node:url'

// Resolve relative to project root
register('./loader.mjs', pathToFileURL('./'))
