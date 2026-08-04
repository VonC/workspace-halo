# Logs and processes

Every observable trace of Workspace Halo, and where to find it.

## The Workspace Halo output channel

Open **View > Output** and select **Workspace Halo**. The extension logs
there:

- `Tracking <name>: root=..., logo=...` when the activation conditions are
  met (`logo=none` for a workspace without a logo file);
- `Native host log: <path>` with the exact per-window log location;
- `Native host started (pid=...)` and
  `Native host exited (code=..., signal=...)`;
- binding confirmations (`Native host confirmed this VS Code window`);
- the logo warnings (multiple logo files, or none matching the workspace
  name) and any startup errors;
- every non-protocol output line of the host process.

## The native host log file

Each host writes `native-host.log` in the extension's VS Code log-storage
directory for that window; the exact path is printed in the output channel.
Lines are prefixed `workspace-halo:` with date and time, and include the
bound window (`bound to hwnd=...`), every visibility change
(`visibility=<reason>`), trigger events (`alt-tab gesture`, `taskbar hover`,
`double-shift gesture`), and the minimize interception steps.

A host started by hand without `--log` writes to
`%TEMP%\workspace-halo-companion.log` instead.

## The host processes

There is exactly one `workspace-halo-host.exe` process per active VS Code
window whose workspace is saved as `.vscode\<name>.code-workspace`; other
windows start none. An independent check from PowerShell:

```powershell
Get-Process workspace-halo-host
```

Windows Task Manager's **Details** view shows the same processes. Closing a
VS Code window or deactivating the extension stops its host; a host that
dies unexpectedly is restarted by the extension after one second, which the
output channel records.
