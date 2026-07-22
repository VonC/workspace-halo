# Why the halo shows only on triggers

A halo that is always visible would be decoration. Workspace Halo treats it as
an identification cue instead: it must appear exactly at the moments when you
are deciding which window is which, and stay out of the way while you work in
a window you have already identified.

## The moments that need identification

Those moments are few and recognizable:

- the window has just opened and you want confirmation of what it is
  (activation);
- you are cycling through windows with Alt+Tab;
- the mouse rests on a Windows taskbar, where thumbnail previews appear;
- the window is minimized, so only its thumbnail represents it;
- another window partially covers it, so its own content cannot identify it;
- you explicitly ask, with a double-Shift gesture.

Everything else, in particular a focused window fully in front of you, is a
moment where the halo would only hide code. The exact conditions, and their
precedence, are listed in [display triggers](../reference/display-triggers.md).

## Focus acknowledges, explicit triggers override

The activation halo stays until you acknowledge the window: the first mouse
click while it is focused dismisses it, and a window that starts unfocused is
dismissed as soon as it gains focus. From then on, focus suppresses the
ambient overlap halo: a focused window never shows a halo just because another
window touches its frame.

Explicit triggers keep priority over that suppression. Double-Shift and
taskbar hovering both show the halo of a focused window, because in both cases
you asked for identification. Double-Shift then hides on the very next input,
so the halo never lingers.

## No halo can trigger another halo

Overlap detection ignores every Workspace Halo overlay window. Without that
exclusion, two VS Code windows relocated onto the same monitor (for example
when Windows consolidates the desktop after a monitor disconnect) could each
see the other's overlay as an occluding window, and both halos would flicker
on forever. Excluding halos from the occlusion walk makes the loop impossible:
only real application windows count as cover.
