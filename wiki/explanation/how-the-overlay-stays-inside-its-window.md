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
bare thumbnail. The host therefore hooks the system's minimize events for its
target process and, on the first minimize transition:

1. cancels the transition by restoring the window without activating it;
2. composes and flushes the child halo through DWM;
3. replays the minimization after a 75 millisecond delay.

The replayed minimize then captures an already-composed window, so the
thumbnail carries the border, name, and logo. While the window stays
minimized, the halo remains part of its composed image.
