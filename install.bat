@echo off

REM ******************************************************************
REM Script Name:  install.bat
REM Description:  Install the packaged Workspace Halo VSIX into the
REM               project's current Visual Studio Code installation.
REM
REM               The target command is:
REM                 %%PRGS%%\vscodes\current\bin\code.cmd
REM
REM               The existing extension is replaced with --force so a
REM               rebuilt VSIX can be reinstalled without a version bump.
REM
REM Parameters:
REM    (none)
REM
REM Usage:
REM    install.bat
REM    i              doskey alias installed by senv.bat
REM
REM Return Value: 0 - Success, non-zero - installation failed
REM ******************************************************************

for %%i in ("%~dp0") do SET "install_dir=%%~fi"
set "install_dir=%install_dir:~0,-1%"

call <NUL "%install_dir%\senv.bat"
if errorlevel 1 goto:install_failed

set "install_version="
for /f "delims=" %%v in ('node -p "require('./package.json').version"') do set "install_version=%%v"
if not defined install_version (
  echo FATAL: package.json does not contain a readable version.
  goto:install_failed
)
set "install_vsix=%install_dir%\workspace-halo-%install_version%-win32-x64.vsix"
if not exist "%install_vsix%" (
  echo FATAL: Packaged VSIX not found: "%install_vsix%"
  echo        Run build.bat first.
  goto:install_failed
)

if not defined PRGS (
  echo FATAL: PRGS is not defined; cannot locate the current VS Code installation.
  goto:install_failed
)

set "install_code=%PRGS%\vscodes\current\bin\code.cmd"
if not exist "%install_code%" (
  echo FATAL: VS Code command not found: "%install_code%"
  goto:install_failed
)

echo INFO: ----------------------------------------
echo INFO: Installing Workspace Halo
echo INFO: ----------------------------------------
echo INFO: VSIX: "%install_vsix%"
echo INFO: Code: "%install_code%"

REM VS Code's Node CLI currently emits DEP0169 while refreshing Marketplace
REM metadata after a local VSIX install. Suppress only that upstream warning
REM for this subprocess; preserve the caller's NODE_OPTIONS and every other
REM warning category.
set "install_node_options=%NODE_OPTIONS%"
if defined NODE_OPTIONS (
  set "NODE_OPTIONS=%NODE_OPTIONS% --disable-warning=DEP0169"
) else (
  set "NODE_OPTIONS=--disable-warning=DEP0169"
)
call "%install_code%" --install-extension "%install_vsix%" --force
set "install_result=%errorlevel%"
set "NODE_OPTIONS=%install_node_options%"
if not "%install_result%"=="0" (
  echo FATAL: VS Code extension installation failed.
  goto:install_failed
)

echo OK: Installed workspace-halo-%install_version%-win32-x64.vsix
call:install_unset
exit /b 0

:install_failed
call:install_unset
exit /b 1

::##################################################
::  CLEANUP
::##################################################

:install_unset
set "install_code="
set "install_node_options="
set "install_result="
set "install_vsix="
set "install_dir="
set "install_version="
goto:eof
