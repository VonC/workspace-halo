# Build and package the VSIX

Goal: produce a provenance-checked
`workspace-halo-<version>-<commit>[-dirty]-win32-x64.vsix` from the source tree.
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
4. `npm run package:vsix` compiles the VCS-stamped Go host to
   `bin\win32-x64\workspace-halo-host.exe`, bundles the extension to
   `dist\extension.js`, writes `dist\build-provenance.json`, and packages the
   VSIX with `@vscode/vsce`.
5. `scripts\verify-vsix-provenance.ps1` checks the filename, package and VSIX
   versions, provenance JSON, target platform, and native host VCS stamp, then
   prints the artifact SHA-256.

`build.bat notest` skips the test gate and goes straight to packaging and
provenance verification.

A clean tree produces the version and short commit in the filename. A tree
with tracked or untracked source changes adds `-dirty`; the same Boolean is
stored in `extension/dist/build-provenance.json` and the native host reports it
as `vcs.modified` through `go version -m`.

After tagging a release, run `version.bat` to audit the artifact recorded by
the latest build. Pass a VSIX path to audit another artifact:

```bat
version.bat
version.bat workspace-halo-0.0.21-debc8e2-win32-x64.vsix
```

The audit reads the package and VSIX versions, full provenance SHA-1, native
host VCS stamp, dirty state, and filename from the package. It rejects dirty
packages, checks that a local tag named for the embedded version targets the
embedded commit, then prints the file's SHA-256 digest.

## Run the checks individually

```powershell
powershell -ExecutionPolicy Bypass -File scripts/test-companion.ps1
npm run check-types
npm test
powershell -ExecutionPolicy Bypass -File scripts/write-build-provenance.ps1
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
