# The minimize that kept coming back

## Behavior history for minimize_loop

- 0.0.16 composited the halo into the tracked VS Code window at minimize start,
  so the minimized Alt+Tab and taskbar thumbnails keep it.
- 0.0.17 replaced that with interception: the host catches the first minimize
  transition, restores the window, composes and flushes the child halo, then
  replays the minimize after a short settling delay (75 ms) so DWM captures a
  window that already shows its halo.
- 0.0.20 restored this composition and replay path, and stopped one workspace's
  halo from triggering another's through occlusion checks. That fixed an
  earlier loop between relocated windows, but only for the `occluded` trigger.
  Nothing bounds the minimize interception itself.

## Current behavior before v0.0.24

1. The host hooks `EVENT_SYSTEM_MINIMIZESTART` and `EVENT_SYSTEM_MINIMIZEEND`
   for the target process with `WINEVENT_OUTOFCONTEXT`, so the events arrive
   asynchronously through the host's message loop.
2. On a `MinimizeStart` in state Idle, `minimizeEventTransition` moves to
   Priming. `restoreTargetForMinimizePriming` restores the window, first with
   `SW_SHOWNOACTIVATE`, then with an activating `SW_RESTORE` if the window is
   still minimized. That restore queues a `MinimizeEnd`.
3. A `MinimizeEnd` in Priming is ignored, on the assumption that it is the one
   caused by the host's own restore.
4. After `minimizeReplayDelayMS` (75 ms), `replayPendingMinimize` moves to
   Replaying and calls `ShowWindow(SW_MINIMIZE)`, which queues a
   `MinimizeStart`.
5. In Replaying, a `MinimizeStart` moves to Committed (logged
   `minimize replay accepted with composed halo`). A `MinimizeEnd` moves back
   to Idle (logged `minimize end`) as if the user had restored the window.
6. When the events arrive late, the `MinimizeEnd` from step 2 lands in
   Replaying, not in Priming. It resets the state to Idle, and the
   `MinimizeStart` from step 4 then arrives in Idle and is taken for a new user
   minimize. The cycle restarts at step 2 and never ends.

## Current side effects before v0.0.24 for minimize_loop

- Each cycle restores the window, which brings it to the front. With several
  VS Code windows looping, they take turns coming forward, several times per
  second.
- Each cycle adds `minimize intercepted`, `minimize replay requested after halo
  composition` and `minimize end` lines to `native-host.log`, and never
  `minimize replay accepted with composed halo`.
- The restore and minimize calls from all the windows keep the system busy, so
  events stay late and the loop keeps itself going.
- No rate limit, time window or circuit breaker caps interceptions per window.

## Observed trigger and gap analysis for minimize_loop

Unplugging the video cable, going from three monitors to the laptop screen
alone, triggers the loop. The Windows 11 setting "Minimize windows when a
monitor is disconnected" minimizes the windows that were on the removed
monitors. During that display reconfiguration, WinEvent delivery takes longer
than the 75 ms replay delay.

The state machine assumes each event it causes arrives before its next step:
the restore's `MinimizeEnd` before the replay, and the replay's `MinimizeStart`
before any other `MinimizeEnd`. Nothing enforces that ordering. The host also
cannot tell its own replayed minimize from a new one, and nothing limits
repeated interceptions.

Evidence, from
`%APPDATA%\Code\logs\20260923T222904\window{1..4}\exthost\vonc.workspace-halo\native-host.log`
on 2026-09-24:

1. 16:43:49 to 16:43:53: Windows minimizes w4, w1 and w2 on its own. The hosts
   log `visibility=minimized minimized=true` with no `minimize intercepted`
   line yet.
2. 16:43:55: each host intercepts (`minimize intercepted: restored=true`) and
   restores its window, 3 to 6 seconds after it logged the window as
   minimized. That delay is inferred from the log chronology: the host does
   not record when Windows generated each event.
3. 16:43:55 to 16:44:35: each host repeats `intercepted -> replay requested ->
   minimize end -> intercepted` every 0.3 to 1 s. The counts are 43, 40, 41 and
   12 interceptions for w1 to w4. `minimize replay accepted with composed halo`
   never appears.

The state machine cycles through these transitions:

| Step | State | Event or action | Result |
| --- | --- | --- | --- |
| 1 | Idle | Late `MinimizeStart` | Priming; the restore queues a `MinimizeEnd` (E1) |
| 2 | Priming | Replay after 75 ms | Replaying; `SW_MINIMIZE` queues a `MinimizeStart` (S2) |
| 3 | Replaying | E1 arrives | Idle, logged `minimize end` |
| 4 | Idle | S2 arrives | Priming again, logged `minimize intercepted` |

## Confirmed rule for `minimize_loop`

- A notification identified as caused by the host's own restore or replay
  never starts another interception, whatever its delivery order or delay.
- A separate minimize by the user or by Windows stays eligible for normal
  interception, even shortly after a replay, when three conditions hold: it can
  be told apart from the host's own events, it is notified within the lateness
  bound, and the window is below the interception cap.
- When a notification's origin is ambiguous, the host does not restore the
  window just to compose a halo. It leaves the actual window state as it is
  and logs the skipped interception.
- The design must choose and validate the method used to attribute an event's
  origin. Matching an event's generation time (`dwmsEventTime`) to the
  interval of a host call is a candidate method. It is timing evidence, not
  proof of origin on its own.
- A minimize notified later than a fixed bound after Windows generated it is
  not intercepted. The window stays minimized without a halo on its thumbnail,
  and the reason is logged. The age is measured from the event generation
  time once the design has validated its clock, precision and wrap behavior.
  When the age can't be established (a missing, future or wrapped timestamp),
  the host doesn't intercept and logs it.
- One minimize action leads to at most one interception, and the window ends
  up minimized. While a replay is pending, the window is visible only because
  the host showed it, so the replay always completes the user's minimize,
  even if the user clicks into the window in that interval. A real restore
  after the replay is honored.
- The host's minimize state always returns to a state consistent with the
  actual window, even when an event is missing or reordered. It never stays
  stuck in an intermediate state.
- A minimize notified promptly, with events arriving in order, still gets a
  thumbnail with its halo when its origin is distinguishable and the
  per-window cap has not applied.
- A per-window cap on interceptions is kept as a last-resort backstop, not as
  the mechanism that ends the loop. A window gets at most N interceptions
  within a sliding time window. A further minimize in that window goes through
  without interception, and the host logs that the cap applied. Counting
  restarts once the window has had no interception for a quiet period. The
  cap is fixed, not a user setting, and is reported in `native-host.log` only.

## Required behaviors to close the gap for `minimize_loop`

1. Handling its own events: a notification the host identifies as caused by
   its own restore or replay is never intercepted, whatever its delay or
   order. A notification of ambiguous origin is not intercepted either: the
   window state stays as it is, and the skip is logged.
2. Keeping user minimizes eligible: a separate user or system minimize, even
   soon after a replay, gets the normal interception when it can be told apart
   from the host's events, is within the lateness bound, and the window is
   below the cap.
3. Completing the pending replay and honoring later restores: the replay
   completes the minimize the user asked for, a restore after the replay keeps
   the window restored, and the host's state is reconciled with the actual
   window so it never stays stuck.
4. Handling late notifications: a minimize notified past the lateness bound,
   or whose age can't be established, is not intercepted. The window stays
   minimized and the reason is logged.
5. Capping as a backstop: interceptions per window follow the sliding-window
   cap with its quiet-period reset, and the host logs when the cap applies.
6. Acceptance evidence: unit tests cover the in-order and reordered recorded
   sequences, a replay notification delayed beyond any timing bound, a prompt
   separate user minimize after a replay, user actions before and after the
   replay, recovery from a missing event, late and unknown-age notifications,
   a notification of ambiguous origin, and cap trip and reset. One manual
   unplug from three monitors to one, with four VS Code windows open, is
   judged per minimize action: the host log shows no repeated interception
   cycle, and no window makes an unsolicited delayed or repeated return to the
   foreground after the unplug. The relevant log lines are kept as evidence.

The design fixes the exact thresholds and documents them with the other
timing constants in `wiki/reference/display-triggers.md`: the lateness bound
(500 ms is an example), the cap count and its time window (2 interceptions
within 2 s is an example), and the quiet period that resets it.

## Related work outside minimize_loop

After the unplug, no host logged `display topology=internal`. In child mode,
the overlay may never receive `WM_DISPLAYCHANGE`, which would disable topology
tracking, including the Duplicate-mode suppression of the `occluded` trigger,
after startup. This is unconfirmed and does not block this issue. It is
tracked as a separate follow-up issue, which must confirm the observation
first.

## Concrete examples for `minimize_loop`

- The events arrive in order: the user's `MinimizeStart`, then the
  restore's `MinimizeEnd`, then the replay's `MinimizeStart` -> one
  interception, `minimize replay accepted with composed halo`, and a
  thumbnail with its halo. This is the same as today.
- The events are reordered: the restore's `MinimizeEnd` arrives after the
  replay was requested, and the replay's `MinimizeStart` arrives after that
  -> both are recognized as the host's own. One interception, and the window
  stays minimized.
- The replay's `MinimizeStart` arrives arbitrarily late, long after the host
  would otherwise be back in its idle state, and the host can identify it as
  its own -> it is never intercepted.
- The user minimizes the window again shortly after a completed replay, and
  the host can tell that minimize apart from its own events -> it is a new
  action and is intercepted normally.
- A `MinimizeStart` arrives that the host cannot confidently attribute, to
  itself or to a separate action -> it is not intercepted. The window stays
  in its actual state, and the ambiguous origin is logged.
- The user clicks into the window while the replay is pending -> the replay
  still minimizes it, as the user asked.
- The user restores the window after the replayed minimize -> the window
  stays restored, and the host returns to its idle state.
- A minimize is notified seconds after Windows performed it, as after the
  unplug -> it is not intercepted. The window stays minimized without a
  halo on its thumbnail, and the reason is logged.
- A minimize's notification age can't be established -> it is not
  intercepted, and the reason is logged.
- An unexpected sequence still repeats interceptions for one window -> the
  cap lets the next minimize through without the halo, the host logs it, and
  normal interception resumes after the quiet period.

## Code references for `minimize_loop`

- `companion/main_windows.go`, `minimizeEventTransition`: the Idle, Priming,
  Replaying and Committed transitions.
- `companion/main_windows.go`, `minimizeWinEventProc`: runs the prime, replay
  and restored actions for each event. It discards the callback's last two
  parameters, `idEventThread` and `dwmsEventTime` (the time Windows generated
  the event). `dwmsEventTime` is timing input the host could use. It doesn't
  identify who caused the event on its own. The design must validate its time
  base, precision, callback timing and wrap behavior for each intended use.
- `companion/main_windows.go`, `restoreTargetForMinimizePriming`: restores the
  window with `SW_SHOWNOACTIVATE`, then falls back to the activating
  `SW_RESTORE`.
- `companion/main_windows.go`, `replayPendingMinimize`: the 75 ms replay and
  its re-prime branch, which has no cap either.
- `companion/main_windows.go`, `installMinimizeHook`: the out-of-context
  WinEvent hook.
- `companion/main_windows.go`, `windowProc` (`wmDisplayChange`) and
  `refreshDisplayTopology`: display-change handling, for the related work
  only.
- `companion/main_windows_test.go`,
  `TestMinimizeEventTransitionPrimesThenAcceptsTheReplay` and
  `TestDuplicateMinimizeStartDoesNotRestartPriming`: tests that cover only
  events arriving in order.
- `src/extension.ts`, `startHost`: starts the host with `--window-mode child`.
- `wiki/explanation/how-the-overlay-stays-inside-its-window.md` and
  `wiki/reference/display-triggers.md`: documentation of the minimize
  interception and its timing constants.

## Requirement clarifications for minimize_loop

| Question | Decision | Integrated in | Rejected alternatives |
| --- | --- | --- | --- |
| Q01 | Track the unconfirmed child-mode `WM_DISPLAYCHANGE` observation as a separate follow-up issue. Keep only a short pointer here, so this issue's completion doesn't depend on that investigation | Related work outside minimize_loop | Fix it in this issue (mixes an unconfirmed topology problem into the loop fix); note only (a likely regression stays untracked) |
| Q02 | Don't intercept a minimize notified past a fixed bound after Windows generated it. The window stays minimized, and the reason is logged. The age comes from the event generation time once the design validates it. When the age can't be established, don't intercept and log it | Confirmed rule; required behavior 4; examples | Always intercept (keeps the visible after-unplug pop); skip after display changes (depends on Q01 and skips prompt minimizes) |
| Q03 | Fixed per-window cap as a backstop: at most N interceptions in a sliding window, the next minimize goes through and is logged, counting restarts after a quiet period. Values fixed by the design and documented, reported in `native-host.log` only | Confirmed rule; required behavior 5; examples | User setting (extension-to-host surface for a rare safety net); output-channel warning (new host-to-extension path for a rare event) |
| Q04 | The pending replay always completes the user's minimize, because the window is visible only because the host showed it. A restore after the replay is honored, and the state is always reconciled with the actual window | Confirmed rule; required behavior 3; examples | User activation cancels the replay (can't safely tell it apart from the host's own activating `SW_RESTORE` fallback); leave it undefined (allows a stuck state) |
| Q05 | Unit tests on the recorded and counterexample sequences, including ambiguous origin, plus one manual 3-to-1 unplug. The unplug is judged per minimize action: no repeated interception cycle, and no unsolicited delayed or repeated return to the foreground | Required behavior 6 | Unit tests only (misses the real trigger); a diagnostic delay flag in the host (test-only feature in the shipped product) |
| Q06 | Identified host events never trigger an interception, distinguishable separate minimizes stay eligible, and ambiguous-origin events are skipped and logged. The design chooses and validates the attribution method. An event-time interval match is a candidate, not proof of origin | Confirmed rule; required behaviors 1 and 2; examples; code references | Arrival-time window plus cap (late replays still restore once, real minimizes can be swallowed); in-flight suppression only (doesn't cover arbitrarily late events) |
