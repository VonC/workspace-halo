# Sign the native host for free

Goal: give `workspace-halo-host.exe` an Authenticode signature without buying
a certificate. Unsigned, the executable triggers SmartScreen friction and
fails endpoint policies that require signed binaries; the VSIX signature
added by the Marketplace does not cover the executable inside it.

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
4. Once accepted, insert the SignPath signing step between the native build
   and `vsce package`, so the VSIX embeds the signed executable.

## Paid or partial alternatives

- **Azure Trusted Signing**: Microsoft's signing service on a small monthly
  subscription, with identity validation and `signtool` integration.
  Inexpensive and well integrated with SmartScreen, but not free.
- **Certum Open Source Code Signing**: a low-cost certificate aimed at
  open-source authors, used with the classic `signtool` flow.
- **Self-signed certificate**: free, but only meaningful for internal
  distribution where the organization deploys the certificate to its trust
  store; the public still sees an unknown publisher.

## Where signing fits the build

Sign the executable after the native build and before packaging, so the VSIX
always embeds a signed host. With a certificate usable by `signtool`:

```bat
signtool sign /fd SHA256 /tr <timestamp-url> /td SHA256 bin\win32-x64\workspace-halo-host.exe
```

Always timestamp (`/tr`): the signature then outlives the certificate's own
validity. With SignPath, this local `signtool` call is replaced by the
signing request in CI. Release builds belong on a trusted Windows runner
with retained logs and checksums, per the
[packaging guide](build-and-package-the-vsix.md); publishing the result is
covered in [publish to the Marketplace](publish-to-the-marketplace.md).
