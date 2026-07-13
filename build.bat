@echo off

REM ******************************************************************
REM Script Name:  build.bat
REM Description:  Build entry point for the Workspace Halo extension.
REM
REM               It sets up the environment (senv.bat), installs the
REM               npm dependencies when they are missing, runs the test
REM               suite as the build gate, then packages the VSIX.
REM
REM               The VSIX build itself chains, through npm:
REM                 - build:native  the Go/Win32 host (needs Go on PATH)
REM                 - compile       tsc --noEmit, then the esbuild bundle
REM                 - vsce package  the win32-x64 VSIX
REM
REM Parameters:
REM    (none)          full build: deps, test gate, VSIX
REM    notest          skip the test gate
REM
REM Usage:
REM    build.bat
REM    build.bat notest
REM
REM Return Value: 0 - Success, non-zero - build failed
REM ******************************************************************

for %%i in ("%~dp0") do SET "build_dir=%%~fi"
set "build_dir=%build_dir:~0,-1%"

call <NUL "%build_dir%\senv.bat"
if errorlevel 1 goto:build_failed

REM The native host is Go, and it is part of vscode:prepublish: without Go the
REM VSIX build fails halfway through. Say so now rather than after npm ci.
where go >NUL 2>&1
if errorlevel 1 (
  echo FATAL: Go not found on PATH: the native host cannot be built.
  goto:build_failed
)

pushd "%build_dir%"

if not exist "%build_dir%\node_modules" (
  echo INFO: node_modules is missing, installing from package-lock.json
  call npm ci
  if errorlevel 1 (
    echo FATAL: npm ci failed.
    goto:build_failed_popd
  )
)

if /i "%~1"=="notest" (
  echo INFO: skipping the test gate ^(notest^)
  goto:build_package
)

echo INFO: ----------------------------------------
echo INFO: Test gate for '%PRJ_DIR_NAME%'
echo INFO: ----------------------------------------
call npm test
if errorlevel 1 (
  echo FATAL: the test gate failed, the VSIX is not packaged.
  goto:build_failed_popd
)

:build_package
echo INFO: ----------------------------------------
echo INFO: Packaging the VSIX for '%PRJ_DIR_NAME%'
echo INFO: ----------------------------------------
call npm run package:vsix
if errorlevel 1 (
  echo FATAL: the VSIX packaging failed.
  goto:build_failed_popd
)

popd

for %%f in ("%build_dir%\*.vsix") do echo OK: Packaged %%~nxf ^(%%~zf bytes^)
call:build_unset
exit /b 0

:build_failed_popd
popd

:build_failed
call:build_unset
exit /b 1

::##################################################
::  CLEANUP
::##################################################

:build_unset
set "build_dir="
goto:eof
