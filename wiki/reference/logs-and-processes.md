# Logs and processes

Every observable trace of Workspace Halo, and where to find it.

## The Workspace Halo output channel

Open **View > Output** and select **Workspace Halo**. The extension logs
there:

- `Tracking <name>: root=..., logo=..., branch=...` when the activation
  conditions are met (`logo=none` for a workspace without a logo file,
  `branch=none` without a branch line);
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

The minimize interception writes one line per decision:

- `minimize edge: shown->iconic age<=<n>ms action=<action>` for an observed
  minimize, with `intercept`, `cancel-replay`, or `skip` followed by
  `reason=unknown-age`, `reason=latched` or `reason=cap`;
  `minimize edge: iconic->shown action=restore-honored` for an observed
  restore;
- `minimize event: start|end age=<n>ms` for each minimize notification of the
  window, with the age of the notification;
- `own restore: before=<state> after=<state> fallback=<bool>` and
  `own replay: before=<state> after=<state>` for the host's own calls, and
  `own call unsettled: expected=<state> observed=<state>` when one did not
  reach its expected state;
- `minimize interception disabled: own call unsettled (...)` when interception
  is latched off for the host session;
- `minimize interception suspended: 2 intercepts in 2000ms` and
  `minimize interception resumed after 5000ms quiet` when the cap trips and
  rearms;
- next to them, the lines already written by earlier versions:
  `minimize intercepted: restored=... replay-in=...`,
  `minimize replay requested after halo composition` and
  `minimize replay accepted with composed halo`.

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
