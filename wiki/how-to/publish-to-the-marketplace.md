# Publish to the Visual Studio Marketplace

Goal: turn the locally built `workspace-halo-win32-x64.vsix` into the public
Marketplace listing under the `vonc` publisher. The whole loop runs in a
browser with a plain Microsoft account: no Azure subscription, no Azure
DevOps organization, and no Personal Access Token.

## Create the publisher once

1. Sign in to the
   [Marketplace management page](https://marketplace.visualstudio.com/manage)
   with the Microsoft account that owns the listing.
2. **Create publisher**. Its identifier must equal the `publisher` field of
   `package.json`, here `vonc`, and cannot be renamed afterwards. Everything
   else — description, logo, links, the "Verified domain" section — is
   optional and editable later.

The publisher profile describes the person; the extension page is generated
from the repository (`README.md` body, `displayName`, `description`, `icon`),
so keep the profile person-centric.

## Why not the PAT route

The classic `vsce publish -p <token>` flow costs more than it returns for a
personal account:

- A Personal Access Token can only be minted from inside an Azure DevOps
  organization (profile page: <https://aex.dev.azure.com/me>, which never
  bounces to the Azure portal the way `dev.azure.com` does for org-less
  accounts). Since 2025, creating a **new** organization requires linking an
  active Azure subscription, which drags a personal account through the
  credit-card signup funnel.
- Azure DevOps retires global PATs on 2026-12-01; Microsoft steers automated
  publishing toward Entra ID workload identity federation instead.

Manual upload needs none of that. Revisit automation only if releases become
frequent, and then through Entra ID rather than a PAT.

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
one. Build with `build.bat`, then upload from the browser on the
[management page](https://marketplace.visualstudio.com/manage):

- First publish: **New extension > Visual Studio Code**, drop
  `workspace-halo-win32-x64.vsix`.
- Later releases: the **...** menu on the Workspace Halo row > **Update**,
  same file.

Because only a `win32-x64` target is published, the Marketplace offers the
extension to Windows x64 VS Code alone; other platforms do not see it as
installable, which matches reality.

## After the upload

The dashboard row shows **Verifying...** while the Marketplace runs its
virus scan, usually a few minutes. When it flips to a check mark, the
listing is live at
`https://marketplace.visualstudio.com/items?itemName=vonc.workspace-halo`
and the extension is installable from the VS Code Extensions view. A
rejected upload arrives as an email with the reason. Later releases repeat
the same loop: bump the version, rebuild, re-sign the host, upload. A
trusted Windows runner should perform the release builds and retain logs and
checksums, as the [packaging guide](build-and-package-the-vsix.md) already
recommends.
