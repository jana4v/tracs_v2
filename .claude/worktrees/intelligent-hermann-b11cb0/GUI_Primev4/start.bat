@echo off
echo Starting Nuxt with Windows ESM fix...
set NODE_OPTIONS=--experimental-specifier-resolution=node --no-warnings
pnpm run dev