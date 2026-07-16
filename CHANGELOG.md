# Changelog

## Unreleased

- Restrict halo display to extension activation, real window overlap, Alt+Tab,
  and mouse taskbar hovering; remove minimized-window, Task View, shell-flyout,
  and double-Shift display paths.
- Render the activation halo immediately and dismiss it on a focused click or
  focus gain. Alt+Tab and taskbar-hover triggers outrank focus; focus suppresses
  ambient overlap halos.
- Ignore every Workspace Halo overlay during occlusion checks, preventing
  relocated VS Code windows from triggering one another indefinitely when a
  multi-monitor desktop is consolidated onto one monitor.

## 0.0.19

- Start the native host as soon as the extension is ready and bind immediately
  when exactly one stable VS Code window title matches the workspace name.
- Keep the focus-confirmation handshake as the fallback for customized or
  ambiguous window titles.

## 0.0.18

- Use DWM extended-frame bounds for occlusion checks so invisible resize borders
  from maximized windows on adjacent monitors do not count as real overlap.
- Let fully exposed VS Code windows hide their halos immediately after Alt+Tab,
  while preserving halos for windows with genuine visible overlap.

## 0.0.17

- Intercept the first minimize transition, restore it without activation, compose
  and flush the child halo, then replay minimization after a short settling
  interval so DWM captures the already-visible halo.
- Remove direct GDI painting into the VS Code surface, which could leave a halo
  visible after the window was restored and fully exposed.

## 0.0.16

- Composite the halo directly into the tracked VS Code window at minimize
  start, making it part of the DWM snapshot used by minimized Alt+Tab and
  taskbar thumbnails. This replaces reliance on child-overlay composition,
  which Windows 11 omits from minimized thumbnails on affected systems.
- Keep the child overlay for normal visible-window policy and explicit
  double-Shift display.

## 0.0.15

- Keep the child overlay surface composed while its halo is hidden by using
  `DWMWA_CLOAK` instead of `SW_HIDE`. Uncloaking and flushing that retained
  surface lets minimized Alt+Tab and taskbar thumbnails display the halo.

## 0.0.14

- Paint the halo on Windows' minimize-start event, before DWM captures the
  minimized window surface, so Alt+Tab and taskbar miniatures retain it.
  Minimized windows then keep the halo active.
- Remove the focus-loss and preview latch: a fully visible, non-minimized
  window hides its halo as soon as no Alt+Tab, taskbar-preview, or
  double-Shift trigger is active.
- Bind each native host through a two-phase focus handshake with its launching
  VS Code extension window. The host rechecks the proposed foreground HWND at
  confirmation time and rejects stale proposals, preventing rapid window
  switches and session restoration from attaching one workspace's halo to
  another VS Code window.
- Add `install.bat` to force-install the packaged VSIX with the project's
  `%PRGS%\vscodes\current\bin\code.cmd`, plus the `i` project environment
  alias.

## 0.0.13

- Re-arm the latch when the window loses the focus: a window the user just
  left keeps its halo until the next click inside it.
- Keep the pill and the name at least `workspaceHalo.pillMargin` pixels
  (default 50) away from the left and right window edges; the font fitting
  accounts for the pill cap overhang.
- Default the pill opacity to 100 (solid), still configurable.
- Alternate opaque black segments in the border, `workspaceHalo.borderSegment`
  pixels long (default 50); the black runs are painted, never transparent,
  and 0 restores the continuous `borderMotif` rendering.

## 0.0.12

- Tint the pill with the name color: one eighth of the text color mixed
  into its higher-contrast pole, so the pill visibly carries the hue. The
  pure pole stays as the fallback when the tint would drop the contrast
  under the 3:1 WCAG large-text minimum.

## 0.0.11

- Hug the pill to the ink of the name: trim the font's internal leading,
  keep only a sliver below the descent, and shrink the horizontal pad. The
  pill no longer inflates from the text cell height.
- Align the black-or-white crossover on the exact WCAG break-even
  luminance.

## 0.0.10

- Draw a rounded pill behind the workspace name, black or white by the WCAG
  contrast computation against the name color, at a configurable opacity
  (`workspaceHalo.namePill`, `workspaceHalo.pillOpacity`, default 80). The
  color-keyed overlay has no alpha blending, so the opacity is rendered as
  an ordered dither.

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
