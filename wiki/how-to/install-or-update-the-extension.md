# Install or update the extension

Goal: get Workspace Halo into VS Code, from the Visual Studio Marketplace
(the normal path) or from a locally built
`workspace-halo-<version>-<commit>[-dirty]-win32-x64.vsix`
(development, audit, or a rebuilt version). Either way the package contains
the compiled extension and the Windows x64 native host; whoever installs it
needs neither Go nor Node.js.

## Install from the Marketplace

Open
[Workspace Halo on the Visual Studio Marketplace](https://marketplace.visualstudio.com/items?itemName=vonc.workspace-halo)
and choose **Install**, search for `Workspace Halo` in the Extensions view, or
run:

```bat
code --install-extension vonc.workspace-halo
```

Only Windows x64 VS Code sees the listing (the published target is
`win32-x64`), and VS Code keeps Marketplace extensions updated
automatically.

## Install a locally built VSIX

Build `workspace-halo-<version>-<commit>[-dirty]-win32-x64.vsix` from the
repository first, as described in
[build and package the VSIX](build-and-package-the-vsix.md). Then:

1. Press **Ctrl+Shift+P** to open VS Code's Command Palette.
2. Run the **Extensions: Install from VSIX...** command.
3. Select the matching
   `workspace-halo-<version>-<commit>[-dirty]-win32-x64.vsix` and reload VS
   Code if requested.

### Install from the command line

To install or replace the local build from the command line, run:

```bat
code --install-extension .\workspace-halo-<version>-<commit>-win32-x64.vsix --force
```

Replace `<version>` and `<commit>` with the values printed by `build.bat`.

`--force` replaces an installed copy even when the version number did not
change, which is exactly what a rebuilt VSIX needs.

### Install from the source tree

From the repository root, after building (see
[build and package the VSIX](build-and-package-the-vsix.md)):

```bat
install.bat
```

`install.bat` targets the project's VS Code installation at
`%PRGS%\vscodes\current\bin\code.cmd` and applies the same `--force`
installation. It reads `dist\build-provenance.json` and selects the exact clean
or dirty artifact recorded by the last build. Once `senv.bat` has initialized
the project command prompt, the `i` doskey alias runs it too.

## Mind the unsigned host executable

The bundled `workspace-halo-host.exe` is currently unsigned. It runs without
issue on the maintainer's corporate laptop, but that does not guarantee that
Smart App Control or another managed endpoint policy will allow it everywhere.
The distinction between the Marketplace package signature and executable
Authenticode signing, along with the release-build ordering requirement, is
covered in [sign the native host for free](sign-the-native-host-for-free.md).
