# How the overlay stays inside its window

The halo is a real Win32 window drawn by the native host. Three decisions keep
it glued to its VS Code window without ever getting in the way: it is a child
window, it is click-through, and minimization is briefly intercepted so
thumbnails include it.

## A child window cannot float over other applications

The overlay is created as a `WS_CHILD` window of the tracked VS Code window.
A child moves with its parent, is clipped to it, and shares its place in the
z-order, so the halo can never appear above an unrelated application the way
an always-on-top popup would. Showing and hiding the halo is then just showing
and hiding the child, placed directly above the parent's client area.

Being a child also means transparency works with a color key rather than
per-pixel alpha: pixels painted with the reserved key color become holes. The
renderer marks every fully transparent pixel with that key, so only the
border, the name with its pill, and the logo remain visible. The color key is
binary, which is why the pill opacity setting is rendered by ordered
dithering, as described in
[how colors keep their contrast](how-colors-keep-their-contrast.md).

## Input passes through untouched

The overlay never takes part in input. It declines activation on mouse
interaction, reports every hit test as transparent, and is created with the
no-activate extended style. Clicks land in the editor below exactly as if the
halo did not exist; the halo's own visibility logic runs on a 25 millisecond
timer tick instead of input events.

## Minimization is replayed so thumbnails keep the halo

Windows composes Alt+Tab and taskbar thumbnails from the last frames a window
presented. A window minimized before its halo was ever composed would show a
bare thumbnail. So when the host observes a minimize it did not cause, it:

1. cancels the transition by restoring the window without activating it;
2. composes and flushes the child halo through DWM;
3. replays the minimization after a 75 millisecond delay.

The replayed minimize then captures an already-composed window, so the
thumbnail carries the border, name, and logo. While the window stays
minimized, the halo remains part of its composed image.

## Interception follows the observed window, not the events

The host still hooks the system's minimize events for its target process, but
an event only makes it look at the window: it reads whether the window is
minimized, as it also does on every 25 millisecond tick, and only a change
between two readings can start an interception. The order, lateness or loss of
events therefore changes nothing. Events can arrive seconds late and out of
order, as they did while monitors were unplugged. A late event now finds the
window already in its current state and does nothing. Before this change, a
late event could re-trigger the interception and bring windows back to the
front in a loop.

The host's own restore and replay are checked the same way. `ShowWindow`
returns once the window's thread has applied the change, so the host reads the
window before and after each call and takes the after-reading as the new
state. Its own transitions are never seen as someone else's minimize. Each
interception performs exactly one restore. A minimize observed while the
replay is still pending, whether a late animation or a second user minimize,
cancels the replay and keeps the halo already composed, instead of restoring
the window again.

Some minimizes go through without the halo, on purpose:

- **Unknown age**: when the last reading that found the window shown is more
  than 500 milliseconds old (a stalled host thread), the minimize may have
  happened long ago. Restoring it now would bring back a window the user saw
  minimize, so it is left alone.
- **Uncertain own call**: when a restore or a replay has not produced the
  expected state after one second, the host cannot tell a slow call from
  someone else's action. It restores nothing on that uncertain state and turns
  interception off for the rest of the host session, because the pending call
  could still apply at any later time and look like a new minimize.
- **Capped**: a third interception within two seconds, which normal use never
  produces, trips a cap. It stays closed until five seconds pass with no
  minimize at all, so a residual loop is stopped rather than slowed down.
- **Latched**: once interception is off for the session, every minimize goes
  through until the host restarts.

Each of these costs only the halo in one thumbnail, never an unexpected return
of the window to the foreground. Every decision is logged, as listed in
[logs and processes](../reference/logs-and-processes.md), and the constants
are in [display triggers](../reference/display-triggers.md).
