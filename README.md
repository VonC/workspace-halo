# Workspace Halo

Workspace Halo makes Visual Studio Code workspaces immediately recognizable on
Windows 11. It draws a configurable border, the workspace name, and its logo
over the VS Code window when Windows shows Alt+Tab, when the window is covered,
or when you press Shift twice.

The overlay is part of the tracked VS Code window, so Windows includes it in the
window's Alt+Tab thumbnail. It never floats above unrelated applications.

## Features

- Shows the halo as soon as Alt+Tab is pressed, even when the VS Code window is
  fully visible on another monitor.
- Shows the halo while an unfocused VS Code window is even partially obscured.
- Shows the halo on demand with a double press and release of Shift.
- Hides an on-demand halo on the next mouse or keyboard interaction.
- Fits the workspace name to the window, up to one third of its height.
- Fits or enlarges the logo into a square up to one third of the window height.
- Supports solid, double, dashed, and dotted borders.
- Uses `peacock.color` when Peacock has a workspace-scoped color; otherwise it
  uses `workspaceHalo.color`.
- Watches the logo and workspace settings and restarts the renderer when they
  change.
- Has no visible UI, commands, status-bar item, or native process when the exact
  workspace logo is absent.

Workspace Halo supports stable desktop VS Code on Windows 11. It is packaged as
a UI extension and is not intended for web-only VS Code, vscode.dev, or running
the UI itself on another operating system.

## Install

Install the platform-specific `workspace-halo-win32-x64.vsix` file:

1. Open the Extensions view in VS Code.
2. Open the `...` menu and choose **Install from VSIX...**.
3. Select the VSIX and reload VS Code if requested.

The VSIX already contains the compiled TypeScript extension and the Windows
x64 native host. End users do not need Go, Node.js, Visual Studio, or the source
tree.

The equivalent command-line installation is:

```powershell
code --install-extension .\workspace-halo-win32-x64.vsix
```

The included native executable is currently unsigned. A corporate release
should Authenticode-sign `workspace-halo-host.exe` before packaging if endpoint
policy requires signed binaries.

## Activate a workspace

Assume the VS Code workspace is named `my-project` and the workspace folder
named `my-project` contains its `.vscode` directory. Add this exact file:

```text
my-project/
|-- .vscode/
|   |-- my-project.logo.png
|   `-- settings.json
`-- ...
```

The filename comparison is case-sensitive and preserves spaces. For a workspace
named `My Project`, the required path is `.vscode/My Project.logo.png`.

For a multi-root workspace, only the root folder whose VS Code folder name
exactly matches the workspace name is inspected. Other root folders are ignored.
If the matching root or exact PNG file is absent, Workspace Halo remains
visually inactive and does not start its native host.

If several `*.logo.png` files exist in the selected `.vscode` directory, the
exact workspace logo is still used. A warning is written to the developer
console, the **Workspace Halo** output channel, and the native-host log. Remove
the extra logo files to clear it.

## Configure

Put settings in the matching workspace folder's `.vscode/settings.json`.
Workspace and workspace-folder values are honored; global user values are
intentionally ignored.

```json
{
  "workspaceHalo.color": "#ff2d55",
  "workspaceHalo.borderWidth": 12,
  "workspaceHalo.borderMotif": "solid",
  "workspaceHalo.fontFamily": "Segoe UI",
  "workspaceHalo.fontWeight": 700,
  "workspaceHalo.textShadow": true
}
```

| Setting | Default | Meaning |
| --- | --- | --- |
| `workspaceHalo.color` | `#ff2d55` | Shared border and workspace-name color. |
| `workspaceHalo.borderWidth` | `12` | Border width from 1 to 64 pixels. |
| `workspaceHalo.borderMotif` | `solid` | `solid`, `double`, `dashed`, or `dotted`. |
| `workspaceHalo.fontFamily` | `Segoe UI` | An installed Windows font family. |
| `workspaceHalo.fontWeight` | `700` | Windows font weight from 1 to 1000. |
| `workspaceHalo.textShadow` | `true` | Draw a dark shadow behind the name. |

A valid workspace-scoped `peacock.color` value takes priority over
`workspaceHalo.color`. The border and name always share the resulting color.

## How it works

VS Code process command lines do not reliably expose the workspace represented
by a particular top-level window. Workspace Halo therefore has two small parts:

- The TypeScript extension uses the VS Code API to identify its own workspace,
  select the exact logo, read workspace-scoped settings, and launch one host for
  that VS Code window.
- The bundled Go/Win32 host binds to that focused stable VS Code window, detects
  Alt+Tab, double-Shift, focus, and geometric occlusion, and renders a child
  overlay. Making it a child is what lets Windows include the halo in the
  Alt+Tab thumbnail without placing it over other applications.

The host is a build artifact, not a separate user-installed component. Closing
the VS Code window or deactivating the extension stops its host process.

## Logs and troubleshooting

Open **View > Output** and select **Workspace Halo** to see extension and host
messages. The native host also writes `native-host.log` in the extension's VS
Code log-storage directory for that window.

If no halo appears:

- Confirm the workspace name and matching root-folder name are identical.
- Confirm `.vscode/<workspace name>.logo.png` exists with matching case.
- Confirm the image is a readable PNG.
- Focus the intended stable VS Code window after activation so its host can bind.
- Check the Workspace Halo output channel for startup or binding errors.

## Build and package

Development requires Go, Node.js, and PowerShell, but not Visual Studio. From the
repository root:

```powershell
npm install
npm test
npm run package:vsix
```

`package:vsix` first compiles the Go host to
`bin/win32-x64/workspace-halo-host.exe`, bundles the TypeScript extension to
`dist/extension.js`, and then creates `workspace-halo-win32-x64.vsix` with
`@vscode/vsce`.

The same platform-specific VSIX can be distributed directly inside an
organization or uploaded as a Windows x64 target when publishing through the VS
Code Marketplace. Release automation should build the executable on a trusted
Windows runner, sign it when required, package the VSIX, and retain both build
logs and checksums.

Run the native and TypeScript checks separately with:

```powershell
go test ./companion/...
npm run check-types
npm test
```

The original Win32 integration experiment and acceptance results are recorded
in `docs/poc.md` in the source repository.
