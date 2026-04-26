@echo off
setlocal enabledelayedexpansion

REM ============================================================
REM  build.bat  —  Build all Mainframe Go services
REM  Run from:  GoLang New\
REM  Output:    GoLang New\bin\*.exe
REM ============================================================

set ROOT=%~dp0
set BIN=%ROOT%bin
set FAILED=0

echo.
echo ============================================================
echo   Mainframe ^— Build All Services
echo   Output: %BIN%
echo ============================================================
echo.

if not exist "%BIN%" mkdir "%BIN%"

REM Must run from workspace root so Go workspace (go.work) is active
cd /d "%ROOT%"

call :build  iam
call :build  gateway
call :build  ingest
call :build  storage
call :build  limiter
call :build  simulator
call :build  comparator
call :build  chainmon
call :build  umacs-tc
call :build  umacs-tc-emulator
call :build  launcher

echo.
if %FAILED%==0 (
    echo [OK] All services built successfully.
) else (
    echo [ERROR] One or more services failed to build.
    exit /b 1
)
exit /b 0


REM ── subroutine :build <service-name> ────────────────────────
:build
set SVC=%~1
set CMD_DIR=%ROOT%%SVC%\cmd

if not exist "%CMD_DIR%" (
    echo [SKIP]  %SVC%  ^(no cmd\ directory^)
    exit /b 0
)

echo|set /p="[BUILD] %-20s" "%SVC%"
go build -o "%BIN%\%SVC%.exe" "./%SVC%/cmd" 2>&1

if errorlevel 1 (
    echo   FAILED
    set FAILED=1
) else (
    echo   OK  ^-^>  bin\%SVC%.exe
)
exit /b 0
