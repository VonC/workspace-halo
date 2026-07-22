# Troubleshoot a missing halo

Goal: find out why a workspace shows no halo. Work through the checks in
order; each one corresponds to a way the extension deliberately stays
inactive or a way the host fails to bind.

## Check the platform first

Workspace Halo only runs on Windows 11 with stable desktop VS Code
(`Code.exe`). VS Code Insiders, vscode.dev, web VS Code, and remote UI on
another operating system never show a halo.

## Check the two names match exactly

1. The workspace name: for a saved workspace it is the `.code-workspace`
   file name without its suffix; for a plain opened folder it is the folder
   name.
2. One root folder name must equal that workspace name, character for
   character (case-sensitive, spaces preserved), or be listed in
   `workspaceHalo.rootSynonyms` as described in
   [accept a differently named root folder](accept-a-differently-named-root-folder.md).
3. The logo must be `.vscode\<workspace name>.logo.png` inside that root
   folder, with the same exact case.

While any of these is false the extension is silent by design: no halo, no
host process, no output.

## Check the logo file itself

The file must be a readable PNG; the host decodes it at startup and exits on
failure. Also remove any other `*.logo.png` files from the same `.vscode`
directory: the exact one still wins, but a warning is raised until the extras
are gone.

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
  shows the root and logo paths;
- `Native host started (pid=...)` and the `Native host log:` line locate the
  host and its `native-host.log`;
- startup errors, binding rejections, and host exits are reported here.

An independent process check from PowerShell:

```powershell
Get-Process workspace-halo-host
```

One process per haloed window is expected. The log locations and formats are
detailed in [logs and processes](../reference/logs-and-processes.md).
