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
   A linked Git worktree is the exception: its folder name does not matter
   when its main working tree holds the same workspace file (see
   [show the branch of a Git worktree](show-the-branch-of-a-git-worktree.md)).
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

## Check a minimized window shown without its halo

When a window was minimized but its taskbar thumbnail shows no halo, the host
most likely let that minimize through on purpose:

1. Open the window's `native-host.log` (its path is in the **Workspace Halo**
   output channel) and find the `minimize edge: shown->iconic` line at the
   time of the minimize.
2. Read its `action`. `action=intercept` means the halo was composed; if the
   thumbnail still lacks it, look for an `own call unsettled` line after it.
   `action=skip` means the minimize went through without the halo, and its
   `reason` says why:
   - `unknown-age`: the host could not prove the minimize was prompt, as after
     a monitor unplug or a busy moment;
   - `latched`: an earlier restore or replay stayed unsettled, so interception
     is off until the host restarts; reload the window to rearm it;
   - `cap`: minimizes came too fast; the cap reopens after a quiet period.
3. When no `minimize edge` line exists at that time, check the binding above:
   an unbound host observes nothing.

The rules behind each reason are in
[display triggers](../reference/display-triggers.md#minimize-interception-rules),
and why the host prefers a missing halo to a window coming back is in
[how the overlay stays inside its window](../explanation/how-the-overlay-stays-inside-its-window.md#interception-follows-the-observed-window-not-the-events).

## Read the logs

Open **View > Output** and select **Workspace Halo**:

- a `Tracking <name>` line confirms the activation conditions are met and
  shows the root, the logo path (`logo=none` without a logo file), and the
  displayed branch (`branch=none` without a branch line);
- `Native host started (pid=...)` and the `Native host log:` line locate the
  host and its `native-host.log`;
- startup errors, binding rejections, and host exits are reported here.

An independent process check from PowerShell:

```powershell
Get-Process workspace-halo-host
```

One process per haloed window is expected. The log locations and formats are
detailed in [logs and processes](../reference/logs-and-processes.md).
