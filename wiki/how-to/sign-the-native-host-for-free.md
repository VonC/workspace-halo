# Sign the native host for free

Goal: give `workspace-halo-host.exe` an Authenticode signature without buying
a certificate. Signing is recommended for public releases because it reduces
SmartScreen friction and avoids endpoint policies that require trusted
publishers, but an unsigned host is not universally blocked.

## Understand what is signed

The packaged `workspace-halo-<version>-win32-x64.vsix` does contain the native
host at:

```text
extension/bin/win32-x64/workspace-halo-host.exe
```

The Visual Studio Marketplace scans every published extension for malware and
signs the extension package. VS Code verifies that signature during
installation to check the package's source and integrity. That signature does
not add an Authenticode signature to `workspace-halo-host.exe` or establish a
Windows publisher identity for it. See VS Code's documentation on
[Marketplace protections](https://code.visualstudio.com/docs/configure/extensions/extension-runtime-security#_marketplace-protections)
and [extension signature verification](https://code.visualstudio.com/docs/configure/extensions/extension-marketplace#_the-extension-signature-cannot-be-verified-by-vs-code).

The current Marketplace-installed host reports `NotSigned` when inspected with
`Get-AuthenticodeSignature`, yet it runs successfully on the maintainer's
corporate Windows laptop. Windows occasionally reports on that laptop that it
is sending a file to the cloud for checking. This confirms that some
cloud-backed Windows or endpoint inspection is active, but it does not prove
that `workspace-halo-host.exe` triggered the check or identify whether the
responsible control is SmartScreen, Smart App Control, Defender, or another
corporate security product.

That successful installation is useful evidence that unsigned does not mean
unexecutable. It is not a guarantee for every endpoint:

- SmartScreen can warn about an unknown or unsigned downloaded executable;
- Windows 11 Smart App Control can block an unsigned executable unless it has
  positive reputation, even when it was not downloaded directly;
- App Control for Business, WDAC, AppLocker, or another corporate endpoint
  policy can require an approved signer or an explicit file rule;
- antivirus or EDR products can inspect, quarantine, or block the host
  independently of the Marketplace's package verification.

Microsoft's
[SmartScreen reputation guidance](https://learn.microsoft.com/en-us/windows/apps/package-and-deploy/smartscreen-reputation)
notes that even a newly signed executable can initially show an unknown-app
warning. A consistent trusted signing identity nevertheless lets publisher
reputation carry across releases; every new unsigned binary must build file
reputation from zero. Managed devices can also prevent users from bypassing a
warning, as described in the
[SmartScreen policy settings](https://learn.microsoft.com/windows/security/operating-system-security/virus-and-threat-protection/microsoft-defender-smartscreen/available-settings).

Signing is therefore not required for every user to run Workspace Halo, but it
is the safer release posture and materially improves compatibility with locked
down endpoints.

## The free route: SignPath Foundation

[SignPath Foundation](https://signpath.org) issues free code-signing
certificates to open-source projects. The model fits this repository well:

- the certificate links the published binary to the public repository, and
  the foundation verifies that what gets signed was built from it;
- the private key never leaves SignPath.io's Hardware Security Module, so
  there is no key file or USB token to protect;
- signing integrates into a CI pipeline: the build uploads the artifact,
  SignPath signs it, and the pipeline downloads the signed result.

The practical sequence for Workspace Halo:

1. Keep the repository public with its MIT license (already the case).
2. Add a CI workflow (GitHub Actions, for instance) that runs `build.bat`
   equivalents on a Windows runner and produces
   `bin/win32-x64/workspace-halo-host.exe` as an artifact.
3. Apply on the SignPath Foundation site with the repository.
4. Once accepted, make signing the last operation that writes the native host
   before VSIX creation, so the VSIX embeds the signed executable.

## Paid or partial alternatives

- **Microsoft Artifact Signing** (formerly Trusted Signing): Microsoft's
  signing service on a small monthly subscription, with identity validation
  and `signtool` integration.
  Inexpensive and well integrated with SmartScreen, but not free.
- **Certum Open Source Code Signing**: a low-cost certificate aimed at
  open-source authors, used with the classic `signtool` flow.
- **Self-signed certificate**: free, but only meaningful for internal
  distribution where the organization deploys the certificate to its trust
  store; the public still sees an unknown publisher.

## Where signing fits the build

Sign the executable after the final native build and before the VSIX archive is
created. With a certificate usable by `signtool`:

```bat
signtool sign /fd SHA256 /tr <timestamp-url> /td SHA256 bin\win32-x64\workspace-halo-host.exe
```

Always timestamp (`/tr`): the signature then outlives the certificate's own
validity. With SignPath, this local `signtool` call is replaced by the
signing request in CI.

There is an important lifecycle trap in this repository: `vsce package`
automatically runs the `vscode:prepublish` script, and the current script runs
`build:native`, overwriting `bin/win32-x64/workspace-halo-host.exe`. Signing an
existing host and then running the unchanged packaging command therefore loses
the signature. A release pipeline must integrate signing after that final
`build:native` operation—for example, by adding a release-only signing task
between `build:native` and `compile` in the prepublish lifecycle—and verify the
signature immediately before creating the VSIX.

Release builds belong on a trusted Windows runner with retained logs and
checksums, per the
[packaging guide](build-and-package-the-vsix.md); publishing the result is
covered in [publish to the Marketplace](publish-to-the-marketplace.md).
