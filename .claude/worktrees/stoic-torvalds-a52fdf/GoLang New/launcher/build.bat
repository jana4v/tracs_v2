@echo off
REM Build launcher and place executable in GoLang New\bin
setlocal enabledelayedexpansion

set OUTPUT_DIR=..\bin
set OUTPUT_FILE=%OUTPUT_DIR%\launcher.exe

REM Create output directory if it doesn't exist
if not exist "%OUTPUT_DIR%" (
    mkdir "%OUTPUT_DIR%"
)

REM Build the launcher
echo Building launcher to %OUTPUT_FILE%...
go build -o "%OUTPUT_FILE%" ./cmd

if errorlevel 1 (
    echo Build failed!
    exit /b 1
) else (
    echo Build successful: %OUTPUT_FILE%
    exit /b 0
)
