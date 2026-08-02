@echo off

REM ******************************************************************
REM Script Name:  version.bat
REM Description:  Audit a packaged Workspace Halo VSIX.
REM
REM               Checks the embedded package and VSIX versions, full
REM               Git SHA-1 provenance, native-host VCS stamp, dirty
REM               state, filename, local version tag, and SHA-256. A
REM               version-tag audit rejects dirty packages.
REM
REM Parameters:
REM    (none)          check the artifact recorded by the latest build
REM    path-to-vsix    check the specified VSIX independently
REM
REM Usage:
REM    version.bat
REM    version.bat workspace-halo-0.0.21-debc8e2-win32-x64.vsix
REM
REM Return Value: 0 - All identity checks passed, non-zero - check failed
REM ******************************************************************

for %%i in ("%~dp0") do SET "version_dir=%%~fi"
set "version_dir=%version_dir:~0,-1%"

call <NUL "%version_dir%\senv.bat"
if errorlevel 1 goto:version_failed

pushd "%version_dir%"

if "%~1"=="" (
  powershell -NoProfile -ExecutionPolicy Bypass -File scripts\verify-vsix-provenance.ps1 -Independent -RequireClean -RequireVersionTag
) else (
  powershell -NoProfile -ExecutionPolicy Bypass -File scripts\verify-vsix-provenance.ps1 -Vsix "%~1" -Independent -RequireClean -RequireVersionTag
)
if errorlevel 1 goto:version_failed_popd

popd
set "version_dir="
exit /b 0

:version_failed_popd
popd

:version_failed
set "version_dir="
exit /b 1
