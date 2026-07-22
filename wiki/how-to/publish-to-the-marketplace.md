# Publish to the Visual Studio Marketplace

Goal: turn the locally built `workspace-halo-win32-x64.vsix` into a public
Marketplace listing. Today the extension is not published; users build and
install the VSIX themselves. Publishing removes that step.

## Create the publisher once

1. Sign in to [Azure DevOps](https://dev.azure.com) with a Microsoft account
   and create an organization if you have none; the Marketplace uses it only
   for authentication.
2. Create a Personal Access Token from your Azure DevOps profile: set
   **Organization** to "All accessible organizations" and the custom scope to
   **Marketplace > Manage**. Store it like a password.
3. Create the publisher on the
   [Marketplace management page](https://marketplace.visualstudio.com/manage).
   Its identifier must equal the `publisher` field of `package.json`, here
   `vonc`.

## Check the manifest before the first publish

The Marketplace page is generated from the repository: `package.json` already
carries the `publisher`, `displayName`, `description`, `repository`,
`homepage`, `bugs`, MIT `license`, and `icon` fields, and the README becomes
the listing body. Inspect what would ship with:

```bat
npx vsce ls
```

`@vscode/vsce` is a dev dependency of this project, so `npx` finds it after
`npm ci`.

## Sign before you ship

The Marketplace signs the VSIX container itself and VS Code verifies that
signature at install time. The bundled `workspace-halo-host.exe` is a
separate concern: Windows endpoint trust wants an Authenticode signature on
the executable. Sign it before packaging, as described in
[sign the native host for free](sign-the-native-host-for-free.md).

## Publish the platform-specific VSIX

Every publish needs a `version` in `package.json` higher than the published
one. Then either publish from the source tree, letting `vsce` rebuild through
`vscode:prepublish` (full toolchain required):

```bat
npx vsce publish --target win32-x64 -p <token>
```

or publish the VSIX that `build.bat` already produced:

```bat
npx vsce publish --packagePath workspace-halo-win32-x64.vsix -p <token>
```

or upload that same file manually on the
[management page](https://marketplace.visualstudio.com/manage) with
**New extension > Visual Studio Code**.

Because only a `win32-x64` target is published, the Marketplace offers the
extension to Windows x64 VS Code alone; other platforms do not see it as
installable, which matches reality.

## After the upload

The Marketplace runs a virus scan before the listing goes live, so allow a
short delay. Later releases repeat the same loop: bump the version, rebuild,
re-sign the host, publish. A trusted Windows runner should perform the
release builds and retain logs and checksums, as the
[packaging guide](build-and-package-the-vsix.md) already recommends.
