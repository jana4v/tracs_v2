@echo off
setlocal enabledelayedexpansion

REM ============================================================
REM  launch.bat  —  Start all Mainframe services via launcher
REM  Run from:   GoLang New\
REM
REM  Usage:
REM    launch.bat              — use pre-built bin\launcher.exe
REM    launch.bat --build      — build everything first, then launch
REM    launch.bat --run        — run from source (go run, no build)
REM ============================================================

set ROOT=%~dp0
set BIN=%ROOT%bin
set LAUNCHER_EXE=%BIN%\launcher.exe
set LAUNCHER_CFG=%ROOT%launcher\config.yaml

REM Parse argument
set MODE=exe
if "%~1"=="--build" set MODE=build
if "%~1"=="--run"   set MODE=run

echo.
echo ============================================================
echo   Mainframe Launcher  [mode: %MODE%]
echo ============================================================
echo.

REM ── --build mode: build all first, then run exe ──────────────
if "%MODE%"=="build" (
    echo [STEP 1/2] Building all services...
    call "%ROOT%build.bat"
    if errorlevel 1 (
        echo [ERROR] Build failed. Aborting launch.
        exit /b 1
    )
    echo.
    echo [STEP 2/2] Starting launcher...
    goto :run_exe
)

REM ── --run mode: go run (no binary needed) ────────────────────
if "%MODE%"=="run" (
    echo [INFO] Starting via go run (source mode)...
    echo [INFO] Config: %LAUNCHER_CFG%
    echo [INFO] Press Ctrl+C to stop all services.
    echo.
    cd /d "%ROOT%launcher"
    go run ./cmd --config "%LAUNCHER_CFG%"
    exit /b %errorlevel%
)

REM ── default: run pre-built exe ───────────────────────────────
:run_exe
if not exist "%LAUNCHER_EXE%" (
    echo [ERROR] %LAUNCHER_EXE% not found.
    echo         Run:  launch.bat --build    to build first
    echo         Run:  launch.bat --run      to run from source
    exit /b 1
)

echo [INFO] Starting: %LAUNCHER_EXE%
echo [INFO] Config:   %LAUNCHER_CFG%
echo [INFO] Press Ctrl+C to stop all services.
echo.
"%LAUNCHER_EXE%" --config "%LAUNCHER_CFG%"
exit /b %errorlevel%
