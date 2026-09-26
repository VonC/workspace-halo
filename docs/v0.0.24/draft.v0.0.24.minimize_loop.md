# The minimize that kept coming back

- Type: issue
- Target version: 0.0.24

## Symptom seen when unplugging monitors

When the video cable is unplugged, going from three monitors to the laptop
screen alone, the VS Code windows keep coming to the front in turn. One window
comes forward with its halo, then another, then the first again, several times
per second, for about 40 seconds. Nothing in the host limits how often this can
happen.

## Evidence from the native-host logs

Logs:
`%APPDATA%\Code\logs\20260923T222904\window{1..4}\exthost\vonc.workspace-halo\native-host.log`,
on 2026-09-24.

1. 16:43:49 to 16:43:53: Windows minimizes w4, w1 and w2 on its own. This comes
   from the Windows 11 setting "Minimize windows when a monitor is
   disconnected". The hosts log `visibility=minimized minimized=true` with no
   `minimize intercepted` line yet.
2. 16:43:55: the `EVENT_SYSTEM_MINIMIZESTART` notifications reach the hosts 3 to
   6 seconds late. Each host intercepts (`minimize intercepted: restored=true`)
   and restores its window, which brings it forward.
3. 16:43:55 to 16:44:35: each host repeats `minimize intercepted` -> `minimize
   replay requested after halo composition` -> `minimize end` -> `minimize
   intercepted`, every 0.3 to 1 s. The counts are 43, 40, 41 and 12
   interceptions for w1 to w4. `minimize replay accepted with composed halo`
   never appears during the loop.

Merged extract, all four hosts sorted by time:

```text
w3 16:43:55.001667 minimize intercepted: restored=true replay-in=75ms
w3 16:43:55.102072 minimize replay requested after halo composition
w1 16:43:55.236672 minimize intercepted: restored=true replay-in=75ms
w1 16:43:55.368963 minimize replay requested after halo composition
w2 16:43:55.785177 minimize intercepted: restored=true replay-in=75ms
w2 16:43:55.902646 minimize replay requested after halo composition
w4 16:43:55.952407 minimize intercepted: restored=true replay-in=75ms
w3 16:43:55.959650 minimize end
w1 16:43:55.961691 minimize end
w4 16:43:56.152714 minimize replay requested after halo composition
w3 16:43:56.252891 minimize intercepted: restored=true replay-in=75ms
```

## Root cause in the minimize state machine

`minimizeEventTransition` (`companion/main_windows.go`) assumes each WinEvent
arrives before the 75 ms replay (`minimizeReplayDelayMS`). The hook is
`WINEVENT_OUTOFCONTEXT`, so events are delivered asynchronously. During a
display reconfiguration, with several Electron windows being moved, delivery
takes longer than 75 ms, and the events arrive in the wrong order:

| Step | State | What happens |
| --- | --- | --- |
| 1 | Idle | A late `MinimizeStart` arrives. The host moves to Priming and restores the window, which queues a `MinimizeEnd` (E1). |
| 2 | Priming | 75 ms later, `replayPendingMinimize` calls `ShowWindow(SW_MINIMIZE)`, which queues a `MinimizeStart` (S2). The state becomes Replaying. |
| 3 | Replaying | E1 arrives only now. It is read as a user restore: the host logs `minimize end` and goes back to Idle. |
| 4 | Idle | S2, the host's own replayed minimize, arrives and is read as a new user minimize. Back to step 1. |

Each cycle restores the window (`restoreTargetForMinimizePriming`, with a
`SW_RESTORE` fallback that activates the window), so it comes to the front. The
restore and minimize calls from four windows keep the system busy, so delivery
stays slow and the loop keeps itself going. No rate limit or circuit breaker
stops it.

## Expected behavior after a monitor unplug

- A minimize started by Windows, the user, or the host itself leads to at most
  one interception, and the window ends up minimized.
- The host never restores, and so never brings forward, the same window
  repeatedly because its own restore and replay events arrive late or out of
  order.
- Even if some other ordering problem remains, interceptions per window are
  capped, so no window can take the front more than a couple of times in a short
  period.

## Proposed fix for the minimize loop

1. Fix the ordering: in Replaying, ignore `MinimizeEnd`, the same way Priming
   already ignores the end caused by its own restore. Only a `MinimizeStart`
   moves Replaying to Committed.
2. Recognize the host's own minimize: record the tick when
   `replayPendingMinimize` calls `SW_MINIMIZE`. Any `MinimizeStart` within about
   1 s of it is the replay and is let through, never intercepted again.
3. Add a per-window circuit breaker: after more than about 2 interceptions
   within 2 s, stop intercepting for a few seconds and let the minimize go
   through without the halo. Log it, for example `minimize interception
   suspended: N intercepts in Xms`.
4. Unit tests on `minimizeEventTransition` and the replay logic that replay the
   late-event sequence from the logs (Start, restore End delivered after the
   replay, replay Start) and assert that it ends in Committed with a single
   interception. Also test that the circuit breaker trips.

Optional: pause minimize interception for about 1 s after a display topology
change. This depends on the related observation below.

## Related observation on display-change notices in child mode

After the unplug, no host logged `display topology=internal`. The extension
starts the host with `--window-mode child`. `WM_DISPLAYCHANGE` appears to be
sent only to top-level windows, so the `WS_CHILD` overlay likely never gets it.
If so, topology tracking, including the Duplicate-mode suppression of the
`occluded` trigger, stops after startup in child mode. This still needs
confirming. A fix would receive display-change notices through a hidden
top-level message window. This may be split out as a separate issue.
