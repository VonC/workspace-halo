# Changelog

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
