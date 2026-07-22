# Set up the build toolchain

Goal: give `build.bat` the tools it needs. The build requires Node.js 22 or
later (with npm), Go for the native host, and PowerShell; Visual Studio is
not needed. The project `senv.bat` accepts either of the two setups below and
fails with a clear message when Node is missing.

## Option 1: Node and Go already on PATH, no senv install

This is the quickest onboarding, verified end to end: with only Node 22, Go,
Git, and the Windows system folders on `PATH`, `build.bat` runs its test
gates and packages the VSIX. Install Node.js 22+ and Go with whatever you
normally use (the official installers, winget, a version manager) and check
both from the shell that will run the build:

```bat
node -v
go version
```

With both answering, `build.bat` works as is: the project `senv.bat` detects
the Node already on `PATH` and skips its portable-toolchain path. Nothing
from senv is required. The plain npm route works in that setup too:
`npm ci` then `npm run package:vsix` produce the same VSIX, without the
test gate that `build.bat` adds.

## Option 2: the portable senv toolchain

[senv](https://github.com/VonC/setupsenv) is a portable, no-admin
development environment for Windows: tools are downloaded and uncompressed
under `%PRGS%`, and nothing outside the session is modified. The minimum
steps from zero to a Workspace Halo build:

1. Clone senv (with its submodule) into `<PRGS>\senv`, for example
   `C:\Public\SOFTWARE\senv`:

   ```cmd
   git clone --recurse-submodules https://github.com/VonC/setupsenv senv
   ```

2. Run `getstarted.bat` once: it creates your private `custom\` repository,
   downloads a minimal tool set, and generates `%USERPROFILE%\senv.bat`.
3. Open a new `CMD` session and type `senv` to activate it.
4. Download and install Node 22 under `%PRGS%\nodes`:

   ```cmd
   dwl node 22
   inst node 22
   ```

5. Same for Go, needed by the native host:

   ```cmd
   dwl go
   inst go
   ```

6. In the workspace-halo checkout, run the project environment then the
   build:

   ```cmd
   senv.bat
   build.bat
   ```

The project `senv.bat` selects Node through senv's `switchnode 22` launcher,
which picks the latest `node22` in `%PRGS%\nodes`, so the session gets the
right version without touching the global `PATH`.

## How the project senv chooses Node

`senv.bat` tries `switchnode 22` first when that launcher is on `PATH`
(option 2), and otherwise uses any Node 22+ already on `PATH` (option 1). Go
is only checked and reported: without it, `build.bat` stops before packaging
with a fatal message, since the VSIX cannot embed the native host.

With the toolchain ready, continue with
[build and package the VSIX](build-and-package-the-vsix.md).
