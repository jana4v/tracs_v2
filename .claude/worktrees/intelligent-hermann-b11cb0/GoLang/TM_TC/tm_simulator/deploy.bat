@echo off
REM Get the directory two levels up from the current directory
set DIST_DIR=%~dp0..\..\..\deployment\lib\tm

REM Create the 'dist' directory if it doesn't exist
if not exist "%DIST_DIR%" (
    mkdir "%DIST_DIR%"
)

REM Build the Go project and save the executable in the 'dist' folder
echo Building the Go project...
go build -o "%DIST_DIR%\tm_simulator.exe" .

REM Check if the build was successful
if %ERRORLEVEL% equ 0 (
    echo Build successful! Executable saved to '%DIST_DIR%\tm_simulator.exe'.
) else (
    echo Build failed. Please check the errors above.
)

pause
