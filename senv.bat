@echo off

REM ******************************************************************
REM Script Name:  senv.bat
REM Description:  minimal environment setup for the Workspace Halo
REM               VS Code extension (Node for the TypeScript bundle,
REM               Go for the Win32 native host).
REM
REM Parameters:
REM none
REM
REM Usage:
REM First script to be called to setup the environment for the project.
REM   senv.bat     apply the environment (skipped when already applied)
REM   fsenv        clear the guard and apply it again (doskey alias)
REM   i             run install.bat (doskey alias)
REM   cdp           cd back to the project root (doskey alias)
REM   pwiki         serve wiki\ as a local website via ..\llm-shared (doskey alias)
REM
REM Node: selected with 'switchnode 22' when that launcher is on PATH
REM (it picks the latest node22 in %PRGS%\nodes). Without switchnode, any
REM Node 22+ already on PATH is used, so a plain 'npm install' setup works
REM as well.
REM Go: expected on PATH, only for the native host (scripts\build-native-host.ps1).
REM Optional: %HOME%\.npmrc (registry, proxy, ...) picked up for npm when
REM HOME is defined by the global senv.
REM Optional: senv.local.bat at the project root, to override any variable.
REM
REM Return Value: 0 - Success, 1 - Error
REM
REM ******************************************************************

for %%i in ("%~dp0") do SET "PRJ_DIR=%%~fi"
set "PRJ_DIR=%PRJ_DIR:~0,-1%"
for %%i in ("%PRJ_DIR%") do SET "PRJ_DIR_NAME=%%~nxi"

REM Project aliases are defined before the guard check on purpose, so even a
REM run skipped by the guard (fresh VSCode console inheriting the variable)
REM installs them. fsenv clears the guard and forces this senv to run again.
doskey fsenv=set "NO_MORE_SENV_%PRJ_DIR_NAME%=" ^& "%~f0" force $*
doskey i=call "%PRJ_DIR%\install.bat" $*
doskey cdp=cd /d "%PRJ_DIR%"

REM Shared tooling lives in the sibling llm-shared checkout. Resolve it here
REM so pwiki works even when llm-shared's own senv never ran in this console.
for %%i in ("%~dp0..\llm-shared") do set "LLM_SHARED_DIR=%%~fi"
REM pwiki goes through the mds.ps1 launcher, not a bare 'python': this senv
REM puts no Python on PATH (Node/Go project) and the launcher self-locates
REM llm-shared's bundled venv Python. PowerShell also stops on Ctrl-C without
REM cmd's "Terminate batch job (Y/N)?" question. The wiki's serve_docs.ini
REM mounts docs\poc.md next to the pages (the root README cannot be mounted:
REM it would collide with wiki\README.md, which becomes the site index).
doskey pwiki=powershell -ExecutionPolicy Bypass -File "%LLM_SHARED_DIR%\bin\mds.ps1" "%PRJ_DIR%\wiki" $*$tif not errorlevel 1 echo %PRJ_DIR_NAME%: wiki server stopped

REM The guard variable is inherited by child processes (a VSCode terminal
REM opened from an activated shell), while PATH changes may have been undone
REM there: say so instead of exiting silently. Recovery in a fresh VSCode
REM terminal, in this order (the global senv resets PATH before rebuilding it,
REM so it must run first):
REM   hsenv         always run %%HOME%%\bin\senv.bat via %%USERPROFILE%%\senv.bat
REM   .\senv.bat    installs the project fsenv alias even when the guard skips
REM   fsenv         clears the project guard and re-runs this senv.bat
if defined NO_MORE_SENV_%PRJ_DIR_NAME% (
  echo INFO: senv already applied for '%PRJ_DIR_NAME%' in this environment, skipping
  exit /b 0
)

REM ---------------------------------------------------------------
REM Node 22
REM ---------------------------------------------------------------
REM switchnode exits with errorlevel 1 even when it succeeds, so its exit
REM code says nothing: NODE_HOME\node.exe is the only reliable check.
where switchnode >NUL 2>&1
if errorlevel 1 goto:node_from_path
pushd "%PRJ_DIR%"
call switchnode 22
popd
if not defined NODE_HOME goto:node_from_path
if exist "%NODE_HOME%\node.exe" goto:node_ready

:node_from_path
where node >NUL 2>&1
if not errorlevel 1 goto:node_ready
echo FATAL: no Node found. Install Node 22 or later and put it on PATH,
echo        or provide the 'switchnode' launcher with a node22 in "%%PRGS%%\nodes".
exit /b 1

:node_ready
for /f "delims=" %%v in ('node -v') do set "NODE_V=%%v"
echo INFO: Using Node %NODE_V%
set "NODE_V="

REM ---------------------------------------------------------------
REM Go, for the native host only
REM ---------------------------------------------------------------
where go >NUL 2>&1
if errorlevel 1 (
  echo WARNING: Go not found on PATH: 'npm run build:native' cannot build the native host.
) else (
  for /f "tokens=3" %%v in ('go version') do set "GO_V=%%v"
)
if defined GO_V echo INFO: Using Go %GO_V%
set "GO_V="

REM ---------------------------------------------------------------
REM Local node_modules binaries first in %PATH%
REM ---------------------------------------------------------------
REM Only if not already present, to avoid duplicates on multiple senv.bat
REM calls from the same command prompt. findstr bug: in a /C: literal on the
REM command line, backslashes in long strings must be doubled to match, hence
REM the %NPM_BIN:\=\\% substitution (a plain ";%NPM_BIN%;" needle never
REM matches and the entry would be re-added on every fsenv).
set "NPM_BIN=%PRJ_DIR%\node_modules\.bin"
echo ;%PATH%; | findstr /C:";%NPM_BIN:\=\\%;" >NUL 2>&1
if errorlevel 1 (
  set "PATH=%NPM_BIN%;%PATH%"
)
set "NPM_BIN="

REM ---------------------------------------------------------------
REM npm user config
REM ---------------------------------------------------------------
REM npm reads its user config from %%HOME%%\.npmrc when HOME is set (the
REM global senv defines HOME). Pin it explicitly so node tools that resolve
REM the home folder from USERPROFILE instead of HOME still pick the same
REM file. Without HOME, npm falls back to %%USERPROFILE%%\.npmrc, which is
REM the normal case outside the global senv: stay silent about it.
if defined HOME (
  if exist "%HOME%\.npmrc" set "NPM_CONFIG_USERCONFIG=%HOME%\.npmrc"
)

REM Keep package-lock.json portable across public and mirrored npm registries.
REM Git stores canonical registry.npmjs.org URLs; checkout smudges them back to
REM the registry selected by npm config for local installs. The helper resolves
REM that registry dynamically, so no private hostname is persisted in Git config.
set "PRJ_DIR_UNIX=%PRJ_DIR:\=/%"
set "EXPECTED_NPM_LOCK_FILTER_VERSION=1"
set "configured_npm_lock_filter_version="
set "NPM_LOCK_MIRROR_PATH=/repository/public-npm/"
for /f "tokens=* delims=" %%i in ('git -C "%PRJ_DIR%" config filter."npm-lock-public".version 2^>nul') do set "configured_npm_lock_filter_version=%%i"
if not "%configured_npm_lock_filter_version%"=="%EXPECTED_NPM_LOCK_FILTER_VERSION%" (
  echo INFO: Configuring npm-lock-public Git content filter
  git -C "%PRJ_DIR%" config filter.npm-lock-public.smudge "bash %PRJ_DIR_UNIX%/scripts/npm-lock-smudge.sh"
  if errorlevel 1 (
    echo FATAL: unable to configure the npm-lock-public smudge filter
    exit /b 1
  )
  git -C "%PRJ_DIR%" config filter.npm-lock-public.clean "sed -E 's#https://[^[:space:]]+%NPM_LOCK_MIRROR_PATH%#https://registry.npmjs.org/#g'"
  if errorlevel 1 (
    echo FATAL: unable to configure the npm-lock-public clean filter
    exit /b 1
  )
  git -C "%PRJ_DIR%" config filter.npm-lock-public.version "%EXPECTED_NPM_LOCK_FILTER_VERSION%"
  if errorlevel 1 (
    echo FATAL: unable to record the npm-lock-public filter version
    exit /b 1
  )
) else (
  echo INFO: npm-lock-public Git content filter already configured
)
set "EXPECTED_NPM_LOCK_FILTER_VERSION="
set "configured_npm_lock_filter_version="
set "NPM_LOCK_MIRROR_PATH="

if exist "%PRJ_DIR%\senv.local.bat" (
  REM Can override variables from senv.bat
  echo INFO: Loading local environment variables from '%PRJ_DIR%\senv.local.bat'
  call "%PRJ_DIR%\senv.local.bat"
)

REM Set project-specific flag when done.
REM Next call to senv.bat will be skipped.
set "NO_MORE_SENV_%PRJ_DIR_NAME%=true "
set "PRJ_DIR_UNIX="

echo OK: Environment initialized for project '%PRJ_DIR_NAME%'

REM 'exit /b 0' is load-bearing, not decoration: neither 'echo' nor 'set'
REM resets ERRORLEVEL in cmd, so the 1 returned by the findstr PATH probe
REM above (and by switchnode, which fails even when it succeeds) would leak
REM out of this script and make the caller believe senv.bat failed.
exit /b 0
