# Changelog

## 0.0.9

- Release the latched halo on any keystroke while its window is focused,
  so a keyboard-only return into the window clears the halo without a
  mouse click.

## 0.0.8

- Latch a taskbar-triggered halo: moving from the taskbar up into the
  thumbnail miniatures no longer drops it. The halo stays until the next
  mouse click inside that window.

## 0.0.7

- Show the halo whenever the mouse is over the taskbar: the reliable cue
  that the thumbnail previews may be displayed, independent of how the
  shell renders them. The flyout-window detection stays as a secondary cue.

## 0.0.6

- Fix the shell preview detection: the flyouts live in z-bands that a
  sibling walk from an application window never enters, so the host now
  scans them with `EnumWindows`, recognizes the Windows App SDK taskbar
  classes (`XamlExplorerHostIslandWindow_WASDK`, WinUI popup site bridges),
  and ignores surfaces parked with an empty rectangle.

## 0.0.5

- Show the halo while a shell window-picking surface is displayed: the
  taskbar thumbnail previews (hovering the VS Code taskbar icon), Alt+Tab,
  and Task View. The halo now also shows on the focused window in that case,
  so its miniature is identifiable like the others.

## 0.0.4

- Add the `workspaceHalo.rootSynonyms` workspace setting: extra root folder
  names accepted as the workspace root when no root folder name matches the
  workspace name.

## 0.0.3

- Add the Workspace Halo logo as the extension icon.
- Mark the Workspace Halo workspace with its own logo, so the extension shows
  its halo on its own repository.
- Keep the development `.vscode` directory and scratch files out of the VSIX.

## 0.0.2

- Derive saved workspace identity from the `.code-workspace` filename instead
  of the VS Code display label containing the localized `(Workspace)` suffix.
- Report the native host PID and log path in the Workspace Halo output channel.

## 0.0.1

- Add workspace-logo activation and workspace-scoped configuration.
- Add Alt+Tab, occlusion, and double-Shift overlay behavior on Windows 11.
- Bundle the Go/Win32 native host in a Windows x64 VSIX.
