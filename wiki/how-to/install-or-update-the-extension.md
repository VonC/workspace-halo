# Install or update the extension

Goal: get Workspace Halo into VS Code, from the Visual Studio Marketplace
(the normal path) or from a locally built `workspace-halo-win32-x64.vsix`
(development, audit, or a rebuilt version). Either way the package contains
the compiled extension and the Windows x64 native host; whoever installs it
needs neither Go nor Node.js.

## Install from the Marketplace

The extension is published at
<https://marketplace.visualstudio.com/items?itemName=vonc.workspace-halo>.
Search "Workspace Halo" in the Extensions view and install, or run:

```bat
code --install-extension vonc.workspace-halo
```

Only Windows x64 VS Code sees the listing (the published target is
`win32-x64`), and VS Code keeps Marketplace extensions updated
automatically.

## Install a local build from the Extensions view

The VSIX comes from a build of this repository (see
[build and package the VSIX](build-and-package-the-vsix.md)).

1. Open the Extensions view in VS Code.
2. Open the `...` menu and choose **Install from VSIX...**.
3. Select `workspace-halo-win32-x64.vsix` and reload VS Code if requested.

## Install a local build from the command line

```bat
code --install-extension .\workspace-halo-win32-x64.vsix --force
```

`--force` replaces an installed copy even when the version number did not
change, which is exactly what a rebuilt VSIX needs.

## Install from the source tree

From the repository root, after building (see
[build and package the VSIX](build-and-package-the-vsix.md)):

```bat
install.bat
```

`install.bat` targets the project's VS Code installation at
`%PRGS%\vscodes\current\bin\code.cmd` and applies the same `--force`
installation. Once `senv.bat` has initialized the project command prompt, the
`i` doskey alias runs it too.

## Mind the unsigned host executable

The bundled `workspace-halo-host.exe` is currently unsigned. If your endpoint
policy requires signed binaries, Authenticode-sign the executable before
packaging, as noted in the packaging guide.
