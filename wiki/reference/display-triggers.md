# Display triggers

Once the host is bound to its window, a 25 millisecond tick evaluates the
states below in strict priority order; the first matching state wins and its
reason is logged in `native-host.log` as `visibility=<reason>` whenever it
changes.

## Priority table

| Priority | Reason | Halo | Condition | Ends when |
| --- | --- | --- | --- | --- |
| 1 | `double-shift` | shown | Two Shift press-and-release gestures within 400 ms while the window is focused | The next input anywhere (system last-input change) or focus loss |
| 2 | `activation` | shown | The host has just bound; shown immediately, even unfocused | First mouse-button press while focused, or gaining focus when initially unfocused |
| 3 | `alt-tab` | shown | Alt is held and Tab has been pressed, on any monitor | Alt is released |
| 4 | `taskbar-hover` | shown | The cursor is inside a taskbar window (`Shell_TrayWnd` or `Shell_SecondaryTrayWnd`), even while the window is focused; outside Duplicate mode, the miniature band above that taskbar, a taskbar-owned composition child, or a classic top-level thumbnail flyout keeps the trigger active | The cursor leaves the taskbar and its miniature region |
| 5 | `minimized` | shown | The window is minimized, a minimize interception waits for its replay, or an own restore or replay has not settled yet | The window is restored, with no interception or own call pending |
| 6 | `focused` | hidden | The window is foreground and no higher state applies | Focus is lost |
| 7 | `occluded` | shown | Another window partially covers this one | The overlap ends |
| 8 | `hidden` | hidden | No state applies | A state above applies |

Because `focused` ranks above `occluded`, a focused window never shows an
overlap halo; the reasoning is in
[why the halo shows only on triggers](../explanation/why-the-halo-shows-only-on-triggers.md).
The `occluded` trigger is also suspended while Windows reports the Win+P
**Duplicate** (clone) topology. Other explicit triggers remain available in
that mode, but thumbnail-flyout hover does not extend `taskbar-hover` beyond
the taskbar itself.

## Occlusion detection details

A window counts as cover only if all of the following hold:

- it sits above the target in the z-order;
- it is visible, not minimized, and not cloaked by DWM;
- its DWM visible frame intersects the target's DWM visible frame
  (extended frame bounds, which exclude the invisible resize borders that
  straddle adjacent monitors);
- its window class is not one of the ignored classes:
  `WorkspaceHaloOverlay` (any halo overlay), `Progman`, `WorkerW`,
  `Shell_TrayWnd`, `Shell_SecondaryTrayWnd`, `tooltips_class32`.

Ignoring `WorkspaceHaloOverlay` prevents one workspace's halo from making
another workspace's halo appear.

## Minimize interception rules

Minimize interception does not act on the system's minimize events. Each
tick, and each minimize event of the target, reads whether the window is
minimized (`IsIconic`); only a change between two readings, an edge, can
start an interception. The host absorbs the changes its own restore and
replay calls produce, so those never count as edges.

A shown-to-iconic edge is intercepted only when all of the following hold:

- no interception is already in progress for the window;
- the last reading that found the window shown is at most the lateness bound
  old, which proves the minimize prompt;
- interception is not latched off for the host session;
- the interception cap is not closed.

Otherwise the minimize goes through without the halo, and `native-host.log`
records the skip reason: `unknown-age`, `latched` or `cap`, in that order of
precedence. A minimize observed while an interception waits for its replay
cancels the replay and keeps the composed halo, without another restore.

The session latch covers an own call whose outcome stays unsettled: when a
restore or a replay has not produced its expected state by the settle
timeout, interception stays off until the host restarts (a window reload, a
settings change, or a new VS Code window). Minimizes keep working; only their
thumbnail halo is lost.

The cap trips when an edge would be the third interception within the cap
window, and from then on it skips every shown-to-iconic edge. Only a quiet
period with no external minimize at all, skipped ones included, rearms it, so
a continuing stream of minimizes keeps it closed.

## Timing constants

| Constant | Value | Role |
| --- | --- | --- |
| Poll tick | 25 ms | Evaluation cadence of all triggers |
| Double-Shift window | 400 ms | Maximal delay between the two Shift releases |
| Minimize replay delay | 75 ms | Delay before replaying an intercepted minimize with the halo composed |
| Lateness bound | 500 ms | Maximal age of the last shown reading for a minimize to be intercepted |
| Settle timeout | 1000 ms | Delay for an own restore or replay to reach its expected state before interception is latched off |
| Interception cap | 2 in 2000 ms | Interceptions per window within the cap window before the cap trips |
| Cap quiet period | 5000 ms | Time without any external minimize before a tripped cap rearms |
| Refresh debounce | 150 ms | Extension-side debounce before re-evaluating conditions |
