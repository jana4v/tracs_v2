// ESM loader to normalize Windows absolute paths (e.g., I:\foo) into valid file:// URLs
// Works with Node 20/22 using the hooks API. Keep logic minimal and defer to next() when not needed.

import { pathToFileURL } from 'node:url';

const drivePathRE = /^[A-Za-z]:[\\/]?/;

export async function resolve(specifier, context, next) {
  if (drivePathRE.test(specifier)) {
    try {
      const url = String(pathToFileURL(specifier));
      return { url, shortCircuit: true };
    } catch {
      // fall through
    }
  }
  return next(specifier, context);
}

export async function load(url, context, next) {
  return next(url, context);
}