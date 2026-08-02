# Build and package the VSIX

Goal: produce `workspace-halo-<version>-win32-x64.vsix` from the source tree.
Development requires Go, Node.js 22 or later, and PowerShell, but not Visual
Studio. When Node and Go are already on `PATH`, `build.bat` works directly
and no senv install is needed; both that setup and the portable senv
toolchain are covered in
[set up the build toolchain](set-up-the-build-toolchain.md).

## Run the full build

From the repository root:

```bat
build.bat
```

The script chains, and stops at the first failure:

1. `senv.bat` initializes the project environment.
2. `npm ci` installs the locked dependencies when `node_modules` is missing.
3. The test gate runs `scripts\test-companion.ps1` (the Go tests of the
   native host) then `npm test` (type check plus the TypeScript tests).
4. `npm run package:vsix` compiles the Go host to
   `bin\win32-x64\workspace-halo-host.exe`, bundles the extension to
   `dist\extension.js`, and packages the VSIX with `@vscode/vsce`.

`build.bat notest` skips the test gate and goes straight to packaging.

## Run the checks individually

```powershell
powershell -ExecutionPolicy Bypass -File scripts/test-companion.ps1
npm run check-types
npm test
```

The npm script names behind the build are listed in
[repository layout](../reference/repository-layout.md).

## Distribute the result

The same platform-specific VSIX can be shared directly inside an
organization, or uploaded as a Windows x64 target when publishing through the
VS Code Marketplace, as described in
[publish to the Marketplace](publish-to-the-marketplace.md). A corporate
release should build on a trusted Windows runner, Authenticode-sign
`workspace-halo-host.exe` when endpoint policy requires signed binaries (see
[sign the native host for free](sign-the-native-host-for-free.md)), package
the VSIX, and retain build logs and checksums. Installation of the packaged
file is covered in
[install or update the extension](install-or-update-the-extension.md).
