# How the host finds its VS Code window

The extension spawns the native host from inside a VS Code window, but the
extension API never reveals the native handle (HWND) of that window. The host
must therefore identify its window on its own, and it must never adopt the
wrong one: a halo on the wrong window would be worse than no halo.

## First strategy: an unambiguous window title

At startup the host enumerates the visible top-level windows of `Code.exe` and
keeps those whose title ends with an exact workspace segment: either
`<name> - Visual Studio Code` or `<name> (Workspace) - Visual Studio Code`,
optionally preceded by a file name. If exactly one window matches, the host
binds to it immediately and reports `workspace-halo-bound startup`.

This is the common case, and it works before the window has ever been focused,
so the activation halo can appear as soon as the extension is ready. A
customized `window.title`, a missing title, or two windows with the same
workspace name all make the match ambiguous, and ambiguity fails closed into
the second strategy rather than guessing.

## Second strategy: the focus handshake

When titles cannot decide, the extension and the host run a two-phase
handshake over the host's standard input and output:

1. The extension sends `focus <token>` whenever its own window is focused,
   and `blur <token>` when it is not. Each signal carries a fresh token.
2. On a focus signal, the host proposes the current foreground window, but
   only if it belongs to `Code.exe`. It answers
   `workspace-halo-candidate <token>`.
3. The extension confirms with `confirm <token>` only while its own window is
   still focused and the token is still the latest one.
4. The host rechecks that the same HWND is still the foreground window, then
   binds and reports `workspace-halo-bound <token>`.

A stale token, a focus change between proposal and confirmation, or a
foreground window that is not stable VS Code each abort the round with
`workspace-halo-rejected`, and the next focus event starts a new round.

## Why the double check matters

During a session restore, several VS Code windows start their extensions at
almost the same time, and each one spawns its own host. Without the handshake,
a host launched by window A could observe window B in the foreground and adopt
it. The token matching, the focus condition on both sides, and the final
foreground recheck ensure that a host binds only to the window whose extension
launched it. The full message set is listed in
[native host command line](../reference/native-host-command-line.md).
