# Why a native host runs beside the extension

Workspace Halo is two cooperating programs: a regular TypeScript VS Code
extension, and a small Go/Win32 executable called the native host. This page
explains why one program is not enough.

## The gap between a workspace and its window

The VS Code extension API tells an extension everything about its own
workspace: its name, its root folders, its settings. It tells it nothing about
the top-level Windows window that displays that workspace. The reverse is also
true at the operating-system level: VS Code process command lines do not
reliably expose which workspace a particular `Code.exe` window represents, so
an external program cannot map windows to workspaces on its own either.

Drawing a halo needs both sides at once: the workspace identity (name, logo,
colors) and the physical window (position, focus, minimization, overlap,
Alt+Tab). No single vantage point has both.

## Two parts, one small responsibility each

The extension keeps the workspace side:

- it derives the logical workspace name and selects the exact
  `<name>.logo.png` file;
- it reads the workspace-scoped settings, including the Peacock color;
- it launches one native host for its own window and passes everything as
  command-line flags;
- it watches the logo file and the settings, and restarts the host when its
  configuration fingerprint changes.

The host keeps the window side:

- it identifies the one `Code.exe` window it belongs to (see
  [how the host finds its window](how-the-host-finds-its-window.md));
- it renders the border, name, and logo as a child overlay of that window;
- it watches double-Shift, Alt+Tab, taskbar hovering, focus, minimization,
  and geometric occlusion to decide visibility.

## A build artifact, not an installed service

The host ships inside the VSIX as `bin/win32-x64/workspace-halo-host.exe`. It
is not a separately installed component and it never outlives its window:
closing the VS Code window or deactivating the extension stops the process.
There is exactly one host process per VS Code window whose workspace has its
exact logo; a window without the logo starts no process, contributes no UI, no
command, and no status-bar item.

This keeps the cost model simple: unmarked workspaces pay nothing, and each
marked workspace pays one tiny always-idle process whose only job is to know
where its window is and when it deserves a halo.
