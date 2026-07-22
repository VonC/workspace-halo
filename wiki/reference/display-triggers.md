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
| 4 | `taskbar-hover` | shown | The cursor is inside a taskbar window (`Shell_TrayWnd` or `Shell_SecondaryTrayWnd`), even while the window is focused | The cursor leaves the taskbar |
| 5 | `minimized` | shown | The window is minimized, or its minimize interception is in progress | The window is restored |
| 6 | `focused` | hidden | The window is foreground and no higher state applies | Focus is lost |
| 7 | `occluded` | shown | Another window partially covers this one | The overlap ends |
| 8 | `hidden` | hidden | No state applies | A state above applies |

Because `focused` ranks above `occluded`, a focused window never shows an
overlap halo; the reasoning is in
[why the halo shows only on triggers](../explanation/why-the-halo-shows-only-on-triggers.md).

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

## Timing constants

| Constant | Value | Role |
| --- | --- | --- |
| Poll tick | 25 ms | Evaluation cadence of all triggers |
| Double-Shift window | 400 ms | Maximal delay between the two Shift releases |
| Minimize replay delay | 75 ms | Delay before replaying an intercepted minimize with the halo composed |
| Refresh debounce | 150 ms | Extension-side debounce before re-evaluating conditions |
