@echo off
REM Windows startup script with ESM fix
set NODE_OPTIONS=--experimental-specifier-resolution=node --loader=tsx/esm
pnpm dev