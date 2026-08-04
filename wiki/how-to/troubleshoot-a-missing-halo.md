# Troubleshoot a missing halo

Goal: find out why a workspace shows no halo. Work through the checks in
order; each one corresponds to a way the extension deliberately stays
inactive or a way the host fails to bind.

## Check the platform first

Workspace Halo only runs on Windows 11 with stable desktop VS Code
(`Code.exe`). VS Code Insiders, vscode.dev, web VS Code, and remote UI on
another operating system never show a halo.

## Check the saved workspace file

1. The workspace must be saved on disk: the workspace name is the
   `.code-workspace` file name without its suffix. A plain opened folder or
   an untitled workspace never shows a halo.
2. One root folder name must equal that workspace name, character for
   character (case-sensitive, spaces preserved), or be listed in
   `workspaceHalo.rootSynonyms` as described in
   [accept a differently named root folder](accept-a-differently-named-root-folder.md).
3. The workspace file must sit inside that root folder as
   `.vscode\<workspace name>.code-workspace`, with the same exact case.

While any of these is false the extension is silent by design: no halo, no
host process, no output.

## Check the optional logo file

A logo is not required: without one the halo shows the border and the name
only. To display a logo, the file must be
`.vscode\<workspace name>.logo.png` inside the matched root folder, matching
the workspace name exactly (case included), and a readable PNG; the host
decodes it at startup and exits on failure. When `*.logo.png` files exist
but the exact name is missing, or extras sit beside it, a warning lists them
until they are renamed or removed.

## Check the binding

If the extension is active but no halo appears, the host may not have bound
to its window:

1. Focus the VS Code window once and leave it focused for a moment; the
   focus handshake needs it (see
   [how the host finds its window](../explanation/how-the-host-finds-its-window.md)).
2. If you customized `window.title`, the title match cannot work; the
   handshake becomes the only path, so focusing the window matters even more.

## Read the logs

Open **View > Output** and select **Workspace Halo**:

- a `Tracking <name>` line confirms the activation conditions are met and
  shows the root and the logo path (`logo=none` without a logo file);
- `Native host started (pid=...)` and the `Native host log:` line locate the
  host and its `native-host.log`;
- startup errors, binding rejections, and host exits are reported here.

An independent process check from PowerShell:

```powershell
Get-Process workspace-halo-host
```

One process per haloed window is expected. The log locations and formats are
detailed in [logs and processes](../reference/logs-and-processes.md).
