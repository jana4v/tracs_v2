// ESM loader to normalize Windows absolute paths (e.g., I:\foo) into valid file:// URLs
// Works with Node 20/22 using the hooks API. Keep logic minimal and defer to next() when not needed.

import { pathToFileURL } from 'node:url';

// Match Windows absolute paths like C:\ or D:/path/to/file
const drivePathRE = /^[a-zA-Z]:[/\\]/;

export async function resolve(specifier, context, next) {
  // Only attempt to convert if it looks like a Windows absolute path
  // and NOT already a file:// URL (avoid recursion)
  if (drivePathRE.test(specifier) && !specifier.startsWith('file://')) {
    try {
      const url = pathToFileURL(specifier).href;
      return { url, shortCircuit: true };
    } catch (_err) {
      // If conversion fails, let the default resolver handle it
      return next(specifier, context);
    }
  }
  
  return next(specifier, context);
}

export async function load(url, context, next) {
  return next(url, context);
}