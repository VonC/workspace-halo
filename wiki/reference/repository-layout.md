# Repository layout

The source tree of the extension, its native host, and their build scripts.

## Directories

| Path | Content |
| --- | --- |
| `src/` | The TypeScript extension: `extension.ts` (controller, host lifecycle, handshake) and `model.ts` (pure decision functions: name derivation, root and logo selection, color resolution) |
| `test/` | `model.test.ts`, the Node test suite of the pure model |
| `companion/` | The Go/Win32 native host: `main_windows.go` and its test suite |
| `scripts/` | PowerShell and shell helpers used by the builds |
| `bin/win32-x64/` | The compiled `workspace-halo-host.exe` packaged into the VSIX |
| `dist/` | The bundled `extension.js` produced by esbuild |
| `images/` | The extension icon, its transparent variant, and the README screenshot |
| `docs/` | `poc.md`, the original Win32 integration experiment and its acceptance results |
| `wiki/` | This Diátaxis documentation set |

## Entry-point files

| File | Role |
| --- | --- |
| `package.json` | Extension manifest: `ui` extension kind, `onStartupFinished` activation, the `workspaceHalo.*` configuration schema, and the npm scripts |
| `esbuild.mjs` | Bundles `src/extension.ts` to `dist/extension.js` |
| `build.bat` | Full build: environment, dependencies, test gate, VSIX packaging; `build.bat notest` skips the gate |
| `install.bat` | Installs the packaged VSIX into `%PRGS%\vscodes\current` with `--force`; the `i` doskey alias runs it |
| `senv.bat` | Initializes the project command prompt and its doskey aliases |
| `workspace-halo-<version>-win32-x64.vsix` | The packaged platform-specific extension |

## npm scripts

| Script | Effect |
| --- | --- |
| `build:native` | Runs `scripts/build-native-host.ps1`: `go build` of the host, stripped, into `bin/win32-x64/` |
| `check-types` | `tsc --noEmit` |
| `compile` | Type check, then the esbuild bundle |
| `test` | Type check, then the Node test suite under `test/` |
| `vscode:prepublish` | `build:native` then `compile`, chained automatically by packaging |
| `package:vsix` | `vsce package --target win32-x64` producing the VSIX |

## Helper scripts

| Script | Role |
| --- | --- |
| `scripts/build-native-host.ps1` | Builds the packaged host executable with a repository-local Go cache |
| `scripts/build-companion.ps1` | Builds the standalone diagnostic companion `bin/workspace-halo-companion.exe` |
| `scripts/test-companion.ps1` | Runs the Go test suite of the host |
| `scripts/install-dev-dependencies.ps1` | Runs `npm install` with an explicit Node.js home |
| `scripts/run-npm-task.ps1` | Runs one allow-listed npm task with an explicit Node.js home |
| `scripts/npm-lock-smudge.sh` | Git filter keeping `package-lock.json` portable across npm registry mirrors |
