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
  be told apart from the host's own events, it is notified promptly enough
  (Q02), and the window is below the cap (Q03).
- When a notification's origin is ambiguous, the host does not restore the
  window just to compose a halo. It leaves the actual window state as it is
  and logs the skipped interception.
- The design must justify any method used to attribute an event's origin
  (Q06). Matching an event's generation time to the interval of a host call is
  timing evidence, not proof of origin on its own.
- One minimize action leads to at most one interception, and the window ends
  up minimized. The one exception is a real user restore, handled as Q04
  decides.
- The host's minimize state always returns to a state consistent with the
  actual window, even when an event is missing or reordered. It never stays
  stuck in an intermediate state.
- A minimize notified promptly, with events arriving in order, still gets a
  thumbnail with its halo when its origin is distinguishable and the
  per-window cap has not applied.
- A per-window cap on interceptions is kept as a last-resort backstop, not as
  the mechanism that ends the loop. Q03 settles its form.

## Required behaviors to close the gap for `minimize_loop`

1. Handling its own events: a notification the host identifies as caused by
   its own restore or replay is never intercepted, whatever its delay or
   order. A notification of ambiguous origin is not intercepted either: the
   window state stays as it is, and the skip is logged (Q06).
2. Keeping user minimizes eligible: a separate user or system minimize, even
   soon after a replay, gets the normal interception when it can be told apart
   from the host's events and passes the Q02 and Q03 rules.
3. Respecting a real user restore: the outcome follows Q04, and the host's
   state is reconciled with the actual window so it never stays stuck.
4. Handling late notifications: a minimize notified too late for a
   non-disruptive halo capture is not intercepted. The window stays minimized
   and the reason is logged (Q02). The lateness criterion must rest on a
   trustworthy event-time basis, with a defined fallback when that basis is
   unavailable or ambiguous.
5. Capping as a backstop: interceptions per window are bounded, and the host
   logs when the bound applies (Q03).
6. Acceptance evidence: unit tests on the recorded and counterexample
   sequences, plus one manual unplug judged per minimize action (Q05).

The exact thresholds (the lateness bound, the cap count, its time window and
the suspension duration) are examples here. The design fixes them.

## Related work outside minimize_loop

After the unplug, no host logged `display topology=internal`. In child mode,
the overlay may never receive `WM_DISPLAYCHANGE`, which would disable topology
tracking after startup. This is not confirmed and does not block this issue.
Q01 moves it to a separate follow-up issue.

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
- The user restores the window after the replayed minimize -> the window
  stays restored, and the host returns to its idle state.
- A minimize is notified seconds after Windows performed it, as after the
  unplug -> it is not intercepted. The window stays minimized without a
  halo on its thumbnail, and the reason is logged.
- A minimize's notification time can't be established -> the fallback
  decided in Q02 applies, and it is logged.
- An unexpected sequence still repeats interceptions for one window -> the
  cap stops them, the window minimizes without the halo, and the host logs
  it.

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

## Open questions for the v0.0.24 minimize_loop issue

### Q01: Scope of the child-mode display-change observation

Question description: the "Related work outside minimize_loop" section notes
that child-mode overlays (`WS_CHILD`) may never receive `WM_DISPLAYCHANGE`, so
topology tracking and the Duplicate-mode suppression of the `occluded` trigger
would stop after startup. This is unconfirmed, and nothing in this issue's
required behaviors depends on it any more. Should this issue also fix the
display-change reception, move it to a separate follow-up issue, or keep it as
a note only?

#### BBQ for Q01

A smoke detector goes off every time someone grills, and while checking it you
find the doorbell wire may also be cut. You can fix both in one visit, open a
second work order for the doorbell, or just note it on the invoice. In this
picture: the smoke detector going off is the minimize interception loop, the
possibly cut doorbell wire is the missing `WM_DISPLAYCHANGE` in child mode, the
second work order is a separate follow-up issue, and the invoice note is the
"Related work outside minimize_loop" section.

#### Options for Q01

- Option A: fix the display-change reception in this issue as well.
  - pro: one effort covers both problems seen during the same unplug.
  - con: mixes two separate problems. Duplicate-mode suppression has nothing to
    do with minimize.
  - con: the observation is unconfirmed, so this issue's completion would
    depend on an investigation.
- Option B: track it in a separate follow-up issue, and keep only a short
  related-work pointer here.
  - pro: this issue stays about ending the minimize loop, and its acceptance is
    clear.
  - pro: the follow-up can confirm the observation first.
  - con: two efforts to run, review and merge.
- Option C: keep the pointer as a note only, with no follow-up.
  - pro: no extra work now.
  - con: a likely regression of the Duplicate-mode protection stays unfixed and
    untracked.

#### Recommended option for Q01 (with arguments for this choice)

Option B: the loop is fully explained by late and reordered events, and none
of this issue's required behaviors needs display-change notices. The child-mode
problem affects another trigger and must be confirmed first. A separate issue
keeps each fix verifiable and still tracks the possible regression.

#### Answer to Q01: option B (with reason why it must be accepted as the answer)

Option B: it keeps this issue focused and its completion independent of an
unconfirmed investigation, without losing track of the possible regression.

### Q02: Minimizes notified late, and how lateness is known

Question description: after the unplug, each host intercepted its window 3 to
6 seconds after logging it as minimized. That delay is inferred from the log
chronology, because the host does not record when Windows generated each
event. Restoring a window that long after its minimize makes it visibly pop
back. Each WinEvent notification does carry the time Windows generated it
(`dwmsEventTime`), which the host discards today. That gives timing input to
estimate a notification's age when it arrives. The design must validate its
time base, precision and wrap behavior before relying on any bound. What
should happen to a minimize notified late, and what applies when its age
cannot be established?

#### BBQ for Q02

A photographer notices a missing name tag on a class photo. Noticed right
away, a two-second retake is fine. Noticed an hour later, calling the student
back disrupts everyone. The time printed on the photo, not the time the
photographer looks at it, tells how old the shot is. In this picture: the
class photo is the minimized window's thumbnail, the name tag is the halo, the
retake is the restore, compose and replay cycle, the time printed on the photo
is the event generation time (`dwmsEventTime`), and the moment the photographer
looks is when the notification reaches the host.

#### Options for Q02

- Option A: always intercept, however late the notification is.
  - pro: every minimized thumbnail gets its halo.
  - con: seconds after an unplug, each affected window visibly reappears and
    minimizes again.
- Option B: intercept only when the notification's age, measured from the time
  Windows generated the event, is under a bound (the design fixes the value,
  500 ms is an example). Past the bound, the window stays minimized without a
  halo and the reason is logged. When the age cannot be established (for
  example a missing, future or wrapped timestamp), do not intercept and log
  it.
  - pro: no window reappears seconds later, which removes the visible pop.
  - pro: the criterion rests on a time Windows records, not on log order,
    once the design has validated its clock and wrap behavior.
  - pro: the fallback fails safe: at worst a bare thumbnail, never a pop.
  - con: thumbnails of windows minimized by an unplug have no halo until their
    next minimize.
- Option C: skip interception for a fixed period after any display
  configuration change.
  - pro: also avoids the visible pop.
  - con: depends on display-change notices, which Q01 moves out of scope.
  - con: also skips prompt, legitimate minimizes during that period.

#### Recommended option for Q02 (with arguments for this choice)

Option B: lateness is what makes an interception disruptive. The event
generation time can provide a basis for measuring age once the design
validates its clock, precision and wrap behavior. Its fallback never restores
a window on uncertain evidence. Normal minimizes, notified within
milliseconds, keep their halo.

#### Answer to Q02: option B (with reason why it must be accepted as the answer)

Option B: it removes the after-unplug pop on a measurable basis, fails safe
when the age is unknown, and doesn't depend on the display-change work moved
out by Q01.

### Q03: Form and semantics of the interception cap

Question description: the cap is a last-resort backstop, not the mechanism
that ends the loop. What does it count, when does it trip and reset, and is it
fixed or configurable?

#### BBQ for Q03

A revolving door has a jam sensor. After a given number of jams within a few
seconds, it stops turning and simply opens so people can walk through, then
resumes once no jam has happened for a while. The threshold can be fixed at the
factory or adjustable by the building manager. In this picture: the jam sensor
is the per-window interception counter, opening the door is letting a minimize
through without a halo, resuming is the counter's reset after a quiet period,
the building manager is the user through a `workspaceHalo` setting, and the
maintenance log is `native-host.log`.

#### Options for Q03

- Option A: fixed documented constants. A window gets at most N interceptions
  within a sliding time window. A further minimize in that window goes through
  without interception, and the host logs that the cap applied. Counting
  restarts once the window has had no interception for a quiet period. The
  design fixes N and both durations (2 interceptions within 2 s and a few
  seconds of quiet are examples). The values are documented with the other
  timing constants in `wiki/reference/display-triggers.md`.
  - pro: no new setting to explain, and it behaves the same everywhere.
  - pro: trip and reset are defined, so they can be tested.
  - con: a user who quickly minimizes, restores and minimizes again can lose
    the halo on a later minimize.
- Option B: the same semantics, with N and the durations as user settings.
  - pro: an unusual workflow can tune it.
  - con: an extension setting has to reach the host, which is more surface for
    a safety net most users never hit.
- Option C: option A plus a warning in the extension's output channel when the
  cap trips.
  - pro: the trip is visible without opening the host log.
  - con: adds a channel from the host to the extension for a rare event.

#### Recommended option for Q03 (with arguments for this choice)

Option A: a backstop should be predictable and testable, not tuned. Defined
trip and reset semantics make it testable, and the host log already records
every visibility and minimize decision.

#### Answer to Q03: option A (with reason why it must be accepted as the answer)

Option A: it adds a bounded, testable protection with the least surface. The
rare lost halo during rapid repeated minimizes is acceptable for a guard whose
job is to make an endless loop impossible.

### Q04: User actions while a replay is pending

Question description: an interception restores the window so the halo can be
composed, then replays the minimize shortly after. A user restore before the
replay cannot happen as a restore: the host has already shown the window. The
user can still act on the window in that interval, for example by clicking into
it. After the replay, the user can restore the window normally. Which actions
cancel the pending replay, and what must the host's state reflect afterwards?

#### BBQ for Q04

A valet takes a car to the garage, but first drives it past the entrance for a
quick photo, then parks it. If the owner taps the window during that photo
pass, the valet can finish parking as asked, or give the keys back. Once the car
is parked, the owner can always ask for it again. In this picture: taking the
car to the garage is the user's minimize, the photo pass is the interval between
the host's priming restore and its replay, tapping the window is the user
clicking into the window, giving the keys back is cancelling the replay, and
asking for the parked car is a real restore after the replay.

#### Options for Q04

- Option A: the pending replay always completes the user's minimize. A restore
  after the replay is honored. In every case the host's state is reconciled
  with the actual window, so it never stays stuck.
  - pro: the host never takes its own activating restore (the `SW_RESTORE`
    fallback) for a user choice.
  - pro: the interval is short, because late notifications are not
    intercepted (Q02).
  - con: a user who clicks into the window during that interval sees it
    minimize anyway.
- Option B: user activation of the window during the interval, not caused by
  the host, cancels the pending replay. The window stays shown and the
  cancellation is logged. A restore after the replay is honored, and the state
  is reconciled in every case.
  - pro: a deliberate user click during the interval is respected.
  - con: it needs to tell user activation apart from the host's own activating
    fallback. A mistake would leave a window shown that the user asked to
    minimize.
- Option C: leave actions during the interval undefined.
  - pro: no extra requirement.
  - con: allows a stuck state, the kind of silent fault this issue is about.

#### Recommended option for Q04 (with arguments for this choice)

Option A: the user's last explicit request in that interval is the minimize,
and the window is visible only because the host showed it. Completing the
replay honors that request, and it can't be confused by the host's own
activation. Reconciling with the actual window state keeps any later restore
effective and prevents a stuck state.

#### Answer to Q04: option A (with reason why it must be accepted as the answer)

Option A: it keeps the user's minimize request, honors every real restore after
it, and guarantees recovery. It avoids the attribution risk of option B for an
interval that Q02 keeps short.

### Q05: Acceptance evidence for the minimize loop fix

Question description: the loop only shows up during a real display
reconfiguration, which unit tests cannot produce. What evidence should close
this issue, and per what unit is the manual result judged?

#### BBQ for Q05

A mechanic fixes a rattle that only appears on cobblestones. A bench test with
recorded vibrations proves the part holds, and a drive over real cobblestones
proves the car is quiet. In this picture: the bench test is the unit tests over
the recorded and counterexample event sequences, the recorded vibrations are
the event orders from the 2026-09-24 logs, the cobblestone drive is a real
three-to-one monitor unplug, and a quiet car is at most one interception per
minimize action, with no window coming forward on its own.

#### Options for Q05

- Option A: unit tests only.
  - pro: fast, repeatable, and part of the normal test run.
  - con: doesn't show that the real Windows timing is covered.
- Option B: unit tests covering the in-order and reordered recorded sequences,
  a replay notification delayed beyond any timing bound, a prompt separate user
  minimize after a replay, user actions before and after the replay (Q04),
  recovery from a missing event, late and unknown-age notifications (Q02), a
  notification of ambiguous origin (Q06), and cap trip and reset (Q03). Plus
  one manual unplug from three monitors to one with four VS Code windows
  open. For each minimize action, the host log shows no repeated interception
  cycle, and no window makes an unsolicited delayed or repeated return to the
  foreground after the unplug. The relevant log lines are kept as evidence.
  - pro: covers the logic, the counterexamples and the real trigger.
  - pro: the per-action criterion doesn't penalize legitimate repeated user
    minimizes in the same session.
  - con: needs one manual session with the docking hardware.
- Option C: option B plus a diagnostic host flag that delays event handling on
  purpose.
  - pro: late delivery can be reproduced without unplugging.
  - con: adds a test-only feature to the shipped host.

#### Recommended option for Q05 (with arguments for this choice)

Option B: the unit tests lock down every ordering and counterexample the
required behaviors name, and one real unplug confirms nothing else loops or
pops. The per-action log criterion makes the manual check objective.

#### Answer to Q05: option B (with reason why it must be accepted as the answer)

Option B: the bug came from a real unplug, so closing it needs that scenario,
judged per minimize action. Option C's diagnostic flag isn't needed once the
tests cover the recorded and counterexample sequences.

### Q06: How the host treats its own events and events of ambiguous origin

Question description: the host's own restore and replay must never start
another interception, whatever the delay or order of their notifications. A
separate user minimize should stay eligible. A clock check on arrival time (for
example "within 1 s of the replay") can't deliver that: a replay notified later
loops again, and a real user minimize inside the interval is swallowed.
Matching an event's generation time (`dwmsEventTime`) to the interval of a
host call is a candidate method. It is timing evidence, not proof of origin: an
unrelated event can be generated in the same interval, timestamps can share a
millisecond, and a host-caused event could be generated outside the measured
interval. Which behavior does this issue require, and what happens when origin
is uncertain?

#### BBQ for Q06

A mailroom receives letters days late and out of order. It wants to spot its
own returned mail. The postmark date matching a day it posted letters is a
strong hint, but someone else may have posted on that day too. For an envelope
it can't place with confidence, the mailroom can guess, or set it aside and
leave the desk as it is. In this picture: the letters are minimize
notifications, the mailroom's own mail is the events caused by the host's
restore and replay, the postmark is the event generation time
(`dwmsEventTime`), the posting days are the intervals of the host's calls,
guessing is intercepting an event of uncertain origin, and setting the envelope
aside is skipping the interception and logging it.

#### Options for Q06

- Option A: safety outcome with an explicit uncertainty exception. A
  notification the host identifies as its own never triggers an interception.
  A separate minimize the host can tell apart stays eligible. A notification of
  ambiguous origin is not intercepted: the window state stays as it is, and the
  skip is logged. The attribution method is left to the design, which must
  validate it. Matching the event generation time to the host's call interval
  is one candidate, and only counts as evidence once validated.
  - pro: the loop can't repeat, even for arbitrarily late events the host can
    identify, and uncertainty never causes a visible restore.
  - pro: states honestly that some separate minimizes may lose their halo when
    their origin is ambiguous.
  - con: how many minimizes turn out ambiguous depends on the attribution
    method the design validates.
- Option B: bounded guarantee. The host treats a notification arriving within
  a fixed interval after its replay as its own, and relies on the cap for
  anything later.
  - pro: simple, and needs no attribution method.
  - con: late replays can still restore the window once more, and a real
    minimize inside the interval is swallowed.
- Option C: no event attribution. While the host has a restore or replay in
  flight, it ignores every notification, then decides only from the actual
  window state.
  - pro: independent of event timing.
  - con: a notification delivered after the in-flight period ends is still
    unattributed, so the arbitrarily late case is not covered.

#### Recommended option for Q06 (with arguments for this choice)

Option A: it keeps the goal (the host's own events never re-trigger) and
fails safe on uncertainty (no restore, logged), without claiming a proof the
issue can't give. The design picks and validates the attribution method,
starting with the event-time interval candidate. Option B gives up the
guarantee for late replays, and option C doesn't cover the late case that
caused the loop.

#### Answer to Q06: option A (with reason why it must be accepted as the answer)

Option A: it's the only option that removes the loop for every event the host
can identify while never restoring a window on uncertain evidence. It also
leaves the attribution proof to the design, where it belongs.
