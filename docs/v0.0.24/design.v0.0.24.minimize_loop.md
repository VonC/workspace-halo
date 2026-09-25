# Design v0.0.24 -- minimize_loop

Reference issue: [issue.v0.0.24.minimize_loop.md](issue.v0.0.24.minimize_loop.md)

---

## Context for v0.0.24 minimize_loop

Unplugging monitors (three to one) made every VS Code window take turns coming
to the front for about 40 seconds. Each native host kept intercepting its own
replayed minimize: late, reordered WinEvents reset the minimize state machine
to Idle, so the replay's `MinimizeStart` looked like a new user minimize. The
settled issue fixes the behavior contract: the host's identified own events
never re-intercept, late or ambiguous minimizes are skipped and logged, the
pending replay completes the minimize, and a fixed cap backs everything up.

This design stops deriving decisions from which event arrived and in what
order. The host decides from the window's actual minimized state, which it
observes directly, and from what its own calls just did.

## Scope for v0.0.24 minimize_loop

The v0.0.24 outcomes are:

1. Interceptions start only from an observed minimize transition the host did
   not cause, observed promptly, and below the cap.
2. The minimize phase always follows the observed window state, so no
   delivery order, lost event or late event can leave it stale, stuck or
   looping.
3. A per-window cap, which a continuing stream of minimizes can't re-arm, backs
   everything up. Every decision is logged with the data needed to validate
   it.

Everything else is either supporting design context for those outcomes or
explicitly deferred.

### In scope for v0.0.24 minimize_loop

- Observation of the window's minimized state (`IsIconic`) on each tick, in
  each minimize WinEvent callback, and right before and after each of the
  host's own `ShowWindow` calls.
- Edge detection with a provable age bound, and absorption of the host's own
  transitions.
- A redesigned minimize phase model, driven by observations only.
- Removal of the re-prime: an observed minimize during Priming completes the
  action without another restore.
- A session latch that turns interception off after the first uncertain own
  call.
- An amendment of the issue's lateness rule, from notification age to the
  age of the observed minimized-state transition (Q02).
- The per-window cap, with a reset that needs a real quiet interval.
- Decision logging in `native-host.log`, and documentation of the new timing
  constants.
- Unit tests for edge handling, the phase model and the cap, driven by
  synthetic observation sequences.

### Deferred from v0.0.24 minimize_loop to v0.0.25 and beyond

- The child-mode `WM_DISPLAYCHANGE` reception problem, tracked as its own
  follow-up issue.
- Pausing interception after display changes, which depends on that
  follow-up.
- Removing or replacing the activating `SW_RESTORE` fallback of the priming
  restore. This design keeps it, and only makes sure it can't repeat.

---

## Confirmed Technical Facts for v0.0.24 minimize_loop

These facts were confirmed by inspecting the current codebase and the
2026-09-24 host logs before writing this design.

**Single-threaded host**: `main` calls `runtime.LockOSThread()`, and both the
25 ms `WM_TIMER` tick and the `WINEVENT_OUTOFCONTEXT` callback
(`minimizeWinEventProc`) run on that thread while it pumps messages. Observing
and deciding need no locking, and a callback never interleaves with a tick.

**The tick already observes the state**: `tick` calls `IsIconic` on the
target every 25 ms to drive the `minimized` visibility trigger.

**The tick saw the unplug minimizes promptly, the events did not**: in the
unplug logs, the tick logged `visibility=minimized minimized=true` for w4 at
16:43:49.6, w1 at 16:43:52.3 and w2 at 16:43:53.6. Their `minimize
intercepted` lines, driven by the WinEvents, came at 16:43:55 or later. The
state observation was prompt while event delivery lagged by seconds.

**Always-iconic prime**: every logged interception says `restored=true`,
including normal ones (for example 2026-09-24 07:10:38). By the time the
callback runs, Windows has already minimized the window. Starting
interception from an observed iconic state therefore doesn't lose a moment
the event path had.

**Event order drives the current transitions**:
`minimizeEventTransition(phase, starting)` treats any `MinimizeEnd` in
Replaying as a user restore, and any `MinimizeStart` in Idle as a new
minimize. That is the transition the loop exploits. Its tests
(`TestMinimizeEventTransitionPrimesThenAcceptsTheReplay`,
`TestDuplicateMinimizeStartDoesNotRestartPriming`) cover in-order sequences
only.

**Uncapped re-prime**: `replayPendingMinimize` re-primes whenever the window
is iconic at replay time, with no count or cap.

**Visibility coupling**: `tick` treats `minimized != 0 || a.minimizeState !=
minimizeIdle` as the `minimized` trigger, so a phase that sticks in a
non-idle value also keeps the halo shown.

**Discarded event metadata**: `minimizeWinEventProc` ignores `idEventThread`
and `dwmsEventTime`. This design logs `dwmsEventTime`'s age for evidence, but
no decision uses it.

---

## Current Behavior for v0.0.24 minimize_loop

```txt
MinimizeStart (any origin) --Idle--> Priming: restore window, compose halo,
                                     replay at +75 ms
MinimizeEnd              --Priming--> ignored (assumed: own restore)
tick at +75 ms           --Priming--> if iconic: re-prime (uncapped)
                                      else: Replaying, SW_MINIMIZE
MinimizeStart            --Replaying--> Committed ("replay accepted")
MinimizeEnd              --Replaying/Committed--> Idle ("minimize end")

Late delivery: own restore End reaches Replaying -> Idle, then own replay
Start reaches Idle -> Priming again: endless loop.
```

## Target Behavior for v0.0.24 minimize_loop

```txt
observe()  -- on every tick, and at once in every minimize WinEvent callback
  iconic := IsIconic(target); now := GetTickCount64()
  phase Unsettled past its deadline (checked on every observation, edge or
    not): phase from iconic, latch interception off for the host session, log
  if iconic == lastIconic: record lastObservedAt; done (no edge)
  edge (not the immediate result of an own call; own transitions are absorbed):
    shown -> iconic:
      phase Shown:
        age bound := now - lastShownAt
        bound > lateness bound  -> skip "unknown-age", Minimized(no halo)
        cap closed or latched   -> skip "cap" or "latched", Minimized(no halo)
        else                    -> intercept: ownRestore(), Priming
      phase Priming (the window was shown by the host):
        cancel replay, Minimized(halo composed), log (no restore)
    iconic -> shown: phase Shown (a restore honored), log

ownRestore() / ownReplay():
  read IsIconic before, call ShowWindow, read IsIconic after,
  set lastIconic from the after reading (the own transition is absorbed)
  after reading != expected -> phase Unsettled(expected, deadline), log

tick at replay time, phase Priming, window still shown:
  ownReplay() -> Minimized(halo composed)
```

Events carry no decision. A late, lost, duplicated or reordered event can only
cause an extra observation of the current state, which finds no edge.

---

## State Observation for v0.0.24 minimize_loop

### Observation points

The host reads `IsIconic` at four kinds of points, all on its single thread:

- every 25 ms tick, as it already does;
- every minimize WinEvent callback for the target, so a prompt event starts
  the interception without waiting for the next tick;
- immediately before each own `ShowWindow` call;
- immediately after each own `ShowWindow` call returns.

Each observation records the reading and the tick (`GetTickCount64`). The host
keeps the last reading (`lastIconic`) and the tick of the last observation
that found the window shown (`lastShownAt`).

### Absorbing the host's own transitions

After an own call, the host takes the after-reading as the new `lastIconic`.
A transition its own call produced is therefore never seen as an edge by
later observations. An edge observed later is not the immediate result of an
own call. While every own call has settled as expected, it can only come from
outside. The one exception is an own call whose outcome was uncertain
(below), which could apply late and show up as an edge. Once that happens,
interception is latched off, so such an edge is never intercepted. No event
origin needs to be inferred.

This rests on one premise: `ShowWindow` on a window owned by another thread
doesn't return until the owning thread has applied the change. That is what
separates it from `ShowWindowAsync`. The premise is checked on every own
call, because the after-reading must equal the expected state (shown after a
restore, iconic after a replay). A mismatch doesn't prove anything about
origin, so it takes the uncertain path below and never triggers an
interception.

### Uncertain outcome of an own call

When the after-reading differs from the expected state, the host can't tell
whether its call hasn't applied yet or someone acted in between. It enters
the phase Unsettled, which records the expected state and a deadline (the
settle timeout). While Unsettled, no edge starts an interception: an edge
toward the expected state is absorbed as the late-applied own call, and any
other edge is logged. At the deadline, the phase is set from the current
reading (Minimized without a halo guarantee, or Shown). This path is logged
with both readings. It fails safe: no restore happens on an uncertain state.

An own call that applies after the deadline would be observed as an edge,
at any delay. A suspension or a time-windowed count can expire before it
applies. So the first Unsettled resolution latches interception off for the
rest of the host session, logged as `minimize interception disabled: own
call unsettled (expected=iconic observed=shown)`. From then on every
shown-to-iconic edge is skipped with reason `latched`, whenever the delayed
own call applies. A failed premise therefore costs at most the one restore
that was already in flight, and no interception follows until the host
restarts (a reload, a settings change or a new VS Code window starts a new
host). Minimizes still work normally while latched; only thumbnail halos are
lost.

### Age bound of an observed edge

A shown-to-iconic edge happened after `lastShownAt`, so its age is at most
`now - lastShownAt`. When that bound is at most the lateness bound (500 ms),
the edge is proven prompt. Normal ticks keep it near 25 ms. When the bound is
larger, because the host's thread was stalled or observations were delayed,
it proves nothing: the edge may have happened just before the observation.
Its age is then unknown, and the edge is skipped with reason `unknown-age`, as
the issue requires for an age that can't be established.

This bounds the age of the minimized-state transition, not the age of the
minimize notification the issue's lateness rule names. The design doesn't
establish that a notification can't be generated before `lastShownAt`, for
example at the start of a minimize animation that sets the iconic state
later. So it doesn't claim the bound measures the notification's age.
Instead, it amends the issue's rule (Q02): the lateness bound applies to the
age of the observed minimized-state transition. That is the age the user
sees, since a late restore is disruptive because the window visibly
minimized long ago. Decisions don't need notifications at all. The amendment
is recorded in the issue when this design is consolidated.
`dwmsEventTime` ages are logged next to the bound, as evidence for any later
revision.

### Missed edges

An external restore followed by an external minimize, both between two
observations, produces no edge and is not intercepted. With a normal 25 ms
tick, that pair is faster than Windows' own minimize and restore animations.
If the thread was stalled long enough to miss it, the minimize would have been
late anyway. The only cost is a thumbnail without a halo, the same fail-safe
result.

---

## Minimize Phase Model for v0.0.24 minimize_loop

### Phases

- Shown: the window is not minimized and no interception is in flight.
- Priming: the host restored the window and composed the halo. The replay is
  due at a tick, and the replay deadline is part of it.
- Minimized: the window is minimized. A flag records whether the halo was
  composed (after a replay) or not (after a skip).
- Unsettled: an own call's outcome didn't match its expected state. It holds
  the expected state and a deadline.

Replaying disappears as a waiting phase. The replay call is synchronous, so
its after-reading settles the result at once: iconic moves to Minimized with
the halo composed, anything else moves to Unsettled.

### Transitions are a pure function

A pure function takes the phase, the observation (reading, edge, age bound),
the cap state and the tick, and returns the next phase and one action:
intercept, replay, cancel replay, skip with a reason, or none. All
Win32 calls stay in the callers. The visibility trigger keeps treating every
phase except Shown as `minimized`.

### Priming, the pending replay and user actions

While Priming, the window is shown because the host showed it. A click into
the window doesn't create an edge and doesn't change the phase. At the replay
tick, if the window is still shown, the replay completes the minimize the
user asked for, as the issue decided.

A `MinimizeStart` generated before the priming restore but delivered during
Priming only triggers an observation. The window is shown, so there is no
edge, and the replay goes on as planned.

### External minimize during Priming

A shown-to-iconic edge during Priming means something minimized the window
the host had shown. The observation can't tell whether that was the original
minimize animation completing late or a fresh user minimize. Either way, the
host never restores again: it cancels the pending replay and moves to
Minimized with the halo composed. Nothing is restored, so this is not an
interception for the cap.

A fresh user minimize in that interval therefore ends in the state the replay
would have produced, with no extra restore. The only cost of the late
animation case is a thumbnail that may miss the halo, if the minimize won
the race against the halo composition. That is the same fail-safe result as
a skipped interception.

### Restores and lost events

An iconic-to-shown edge in Minimized moves to Shown and is logged as a
restore. It comes from the tick, not from a `MinimizeEnd`, so a lost End can't
leave Minimized stale: the next tick sees the shown window. A late own event
can't regress a phase either, because events only trigger an observation of
the present state.

### Removed re-prime

The old iconic-at-replay-time re-prime, which restored the window again with
no bound, is removed. The "external minimize during Priming" rule above
replaces it. Each interception performs exactly one restore.

---

## Interception Cap for v0.0.24 minimize_loop

### Counting and trip

Each interception (priming restore) records its tick. When an external
shown-to-iconic edge in Shown would be intercepted, and 2 interceptions
already happened within the last 2 s, the cap applies. The edge is skipped
with reason `cap`, and the host enters suspension.

### Suspension and reset

While suspended, every external shown-to-iconic edge is skipped with reason
`cap`, and its tick is recorded as an attempt. Suspension ends only after
5 s with no attempt, meaning no external minimize edge at all, not merely no
interception. A continuing stream of minimizes keeps the cap closed however
long it lasts. The values are fixed constants, not settings, and are
documented in `wiki/reference/display-triggers.md`: 2 interceptions, a 2 s
window, a 5 s quiet period.

### Why these values

A normal minimize costs one interception. A user who minimizes, restores and
minimizes again within 2 s gets both halos. Only a third interception within
2 s, which no normal usage produces, trips the cap. Because only a real 5 s
pause rearms it, a residual loop is stopped, not rate-limited.

---

## Logging and Documentation for v0.0.24 minimize_loop

### Decision log lines

Each external edge logs one line with its direction, age bound and decision,
for example `minimize edge: shown->iconic age<=25ms action=intercept`,
`minimize edge: shown->iconic age<=3150ms action=skip reason=unknown-age`,
or
`minimize edge: iconic->shown action=restore-honored`. Own calls log their
before and after readings. A mismatch logs `own call unsettled:
expected=iconic observed=shown`. Minimize WinEvents log their `dwmsEventTime`
age, to compare it with the observation bound. The cap logs
`minimize interception suspended: 2 intercepts in 2000ms` when it trips and
`minimize interception resumed after 5000ms quiet` when it rearms. The
existing interception lines stay, so older logs remain comparable.

### Documentation updates

`wiki/reference/display-triggers.md` gains the lateness bound, the settle
timeout and the cap values in its timing constants table, and describes the
session latch.
`wiki/explanation/how-the-overlay-stays-inside-its-window.md` explains that
interception now starts from an observed minimize, and that minimizes of
unknown age, uncertain own calls, and capped or latched minimizes are let
through without the halo, and why. `CHANGELOG.md`
records the fix under 0.0.24.

---

## Acceptance Cases for v0.0.24 minimize_loop

| Scenario | Expected outcome | Reason |
| --- | --- | --- |
| User minimizes; next tick observes iconic 20 ms after a shown reading | One interception, replay, Minimized with the halo | Normal path |
| Events for that minimize delivered in any order, or 3 s late | No effect: each triggers an observation that finds no edge | Events carry no decision |
| Own replay's `MinimizeStart` delivered after the window was restored | No effect: the observation finds the window shown, the phase stays Shown | Late events can't regress a phase |
| Restore `MinimizeEnd` lost while Minimized | Next tick observes shown, phase Shown | Observation, not events, drives restores |
| `MinimizeStart` generated before priming, delivered during Priming | Window shown, no edge, the replay proceeds | Pending replay completes the minimize |
| Window minimized again 100 ms after the priming restore (late animation) | Replay cancelled, Minimized with the halo composed, no restore | No re-prime |
| User starts a second minimize within 250 ms of the priming restore | Replay cancelled, Minimized, no extra restore, not counted by the cap | A fresh user minimize is never undone |
| Replay's after-reading is shown, not iconic | Unsettled, no interception; at the deadline, phase from the reading, interception latched off, logged | Uncertain own outcome fails safe |
| That own replay applies 2.5 s after its call (after the deadline) | Edge skipped, reason `latched` | The first Unsettled resolution latches interception off |
| That own replay applies 7 s after its call (after any 5 s quiet period) | Edge skipped, reason `latched` | The latch doesn't expire |
| An unrelated minimize 90 s after the latch | Skipped, reason `latched`; the window minimizes without the halo | The latch lasts for the host session |
| Host restarted after a latch (reload or settings change) | Interception active again | The latch is per host session |
| Iconic edge with `lastShownAt` 3.4 s old (stalled thread) | Skipped, reason `unknown-age` | A bound over 500 ms proves no age |
| Minimize, restore and minimize within 2 s | Two interceptions, both halos | Cap silent in normal use |
| Third external minimize edge within 2 s | Skipped, reason `cap`, suspension logged | Cap trips |
| External minimize edges every second for 20 s | Cap stays closed for the whole stream, then rearms 5 s after the last edge | Reset needs a real quiet interval |
| Manual unplug, three to one monitors, four windows | At most one interception per minimize action, no unsolicited delayed or repeated return to the foreground | Real trigger acceptance |

## Open questions for the v0.0.24 minimize_loop design

### Q01: What minimize decisions are based on

Question description: round 1 classified each WinEvent by origin, matching its
generation time against a ledger of the host's own calls. A timing match
establishes proximity, not origin: a coincidental separate event can match
first, and a late own event can regress a phase already reconciled. The
revised design instead observes the window's minimized state (`IsIconic`) on
each tick, in each callback, and around each own call. It absorbs the host's
own transitions. A later edge is not the immediate result of an own call.
While every own call has settled as expected, it comes from outside. After an
uncertain own call it could be that call applying late, which Q06 handles by
latching interception off. Which basis should the design use?

#### BBQ for Q01

A doorman wants to know who has left a building. He can read the exit log
slips that arrive by internal mail, sometimes days late and out of order. He
can watch the door himself and note that he holds it open for the staff he
escorts out. Or he can do both and trust the slips when they agree with what
he saw. In this picture: the exit log slips are the WinEvents, internal mail
is out-of-context delivery, watching the door is observing `IsIconic`,
holding the door for the escorted staff is absorbing the host's own
transitions, and the staff are the host's own restore and replay.

#### Options for Q01

- Option A: observed state edges. Decisions come only from `IsIconic` edges
  observed after expected own-call transitions have been absorbed; uncertain
  own calls disable interception. Events only trigger an extra observation.
  - pro: late, lost, duplicated or reordered events have no effect on
    decisions.
  - pro: the host's own transitions are absorbed when they settle as
    expected, so no origin inference is needed. An uncertain own call
    latches interception off (Q06), so its delayed edge is never
    intercepted.
  - pro: the unplug logs show the tick saw the minimizes seconds before the
    events arrived.
  - con: two external transitions between two observations (normally 25 ms
    apart) are missed, which leaves a thumbnail without a halo.
- Option B: classified events with the own-call ledger, as in round 1.
  - pro: reacts to each individual event.
  - con: the timing match can't prove origin, and late events can regress a
    reconciled phase unless many extra guards are added.
- Option C: a hybrid. State edges decide, and events must agree before an
  interception.
  - pro: two independent signals.
  - con: the event half brings back the delivery-order problems, and
    disagreement needs yet another fail-safe rule.

#### Recommended option for Q01 (with arguments for this choice)

Option A: it removes event order from the decisions, which is the root cause
of the loop. It needs no origin inference, and its only failure mode, a
missed double transition within one tick, is harmless and faster than Windows'
own animations.

#### Answer to Q01: option A (with reason why it must be accepted as the answer)

Option A: it makes every issue guarantee hold by construction instead of by
guards, and the logs confirm the observation is the more prompt signal.

### Q02: Basis and value of the lateness bound

Question description: the issue settles that a minimize notified later than a
fixed bound after Windows generated the notification is not intercepted, and
that a minimize whose age can't be established is not intercepted either. The
host observes edges with its own clock, so it can bound the age of the
minimized-state transition by `now - lastShownAt`. It cannot bound the age of
the notification: nothing established here says a `MinimizeStart` can't be
generated before `lastShownAt`, for example at the start of an animation that
sets the iconic state later. The design needs either an issue-consistent basis
or an explicit amendment of the issue's rule. Which should it be, and is
500 ms the right value?

#### BBQ for Q02

A night guard checks a gate every few minutes. When he finds it open, he knows
it opened after his last round. But the visitor may have rung the bell well
before that, and the bell's own clock has never been checked. The house rule
says "don't chase visitors who rang more than a minute ago". The guard can
follow the bell's clock, keep waiting for the bell's record before acting, or
ask the owner to change the rule to "don't chase gates opened more than a
minute ago". In this picture: the gate opening is the minimized-state
transition, the guard's rounds are the host's observations, ringing the bell
is the notification's generation, the bell's clock is `dwmsEventTime`, and the
owner changing the rule is the human amending the issue.

#### Options for Q02

- Option A: amend the issue's rule to the age of the observed minimized-state
  transition, bounded by `now - lastShownAt` within 500 ms. A larger bound is
  `unknown-age` and skipped. `dwmsEventTime` is logged only. The amendment is
  recorded in the issue when this design is consolidated.
  - pro: the bound is proven from the host's own observations, with nothing
    external to validate.
  - pro: it measures what the user sees: a restore is disruptive because the
    window visibly minimized long ago, not because a notification was old.
  - pro: decisions stay independent of notifications, whose delivery caused
    the loop.
  - con: it changes a settled issue rule, which needs explicit human
    agreement.
- Option B: keep the issue's rule. Intercept an observed edge only once a
  `MinimizeStart` for it arrives with a validated `dwmsEventTime` age within
  500 ms.
  - pro: literally consistent with the issue, with no amendment.
  - con: interception waits for notifications, which lag by seconds in the
    very situation this issue fixes, so those minimizes are all skipped.
  - con: brings back the timestamp validation (clock, precision, wrap) and
    ties a decision to event delivery again.
- Option C: run a measurement spike first. Log both the notification
  generation time and the observed transition, establish a maximum lead time,
  and subtract it from the 500 ms budget.
  - pro: could keep the issue's wording with evidence.
  - con: a lead time measured on one machine is not a guarantee, and it
    delays the fix for a rule the user doesn't observe directly.

#### Recommended option for Q02 (with arguments for this choice)

Option A: the transition age is what makes a late restore visible, and the
host can prove it. The issue's notification-age wording came from an earlier,
event-driven stage of the analysis. The amendment keeps the issue's intent
(no late restores, skip anything unestablished) on a basis that needs no
unproven ordering.

#### Answer to Q02: option A (with reason why it must be accepted as the answer)

Option A: it enforces the issue's intent with a provable bound. Choosing it at
the review gate is the explicit human amendment, which consolidation then
records in the issue.

### Q03: Where the minimize logic lives

Question description: `companion/main_windows.go` is 2022 lines and holds the
whole host. The design adds observation handling, a new phase model and a cap,
all pure logic apart from the Win32 calls around them. Where should that
logic live?

#### BBQ for Q03

A workshop gets a new precision tool. It can hang it on the already crowded
main wall, give it its own labelled drawer in the same workshop, or build a
separate shed for it. In this picture: the main wall is
`companion/main_windows.go`, the labelled drawer is a new file in the same Go
package, the separate shed is a new Go package, and the precision tool is the
minimize observation, phase and cap logic.

#### Options for Q03

- Option A: add it to `companion/main_windows.go`.
  - pro: no new file.
  - con: grows an already large file, and mixes pure logic with Win32 calls.
- Option B: a new `companion/minimize_windows.go` in the same package, with
  its tests in `companion/minimize_windows_test.go`. The Win32 calls stay with
  their current callers.
  - pro: the pure logic and its tests read as one unit.
  - pro: same package, so no exported API and no build change.
  - con: the minimize flow is split across two files.
- Option C: a separate Go package.
  - pro: strongest isolation.
  - con: needs exported types and a package boundary for code with a single
    caller.

#### Recommended option for Q03 (with arguments for this choice)

Option B: it isolates the part that needs the most tests without adding a
package boundary, and it stops `main_windows.go` from growing further.

#### Answer to Q03: option B (with reason why it must be accepted as the answer)

Option B: it gives the new logic a clear home and test file at no build or API
cost.

### Q04: Minimize observed during Priming

Question description: the old re-prime restored the window whenever it was
iconic again at replay time, with no bound. A shown-to-iconic edge during
Priming can be the original minimize animation completing late, or a fresh
user minimize. The observation can't tell them apart. Should the host restore
again in that case?

#### BBQ for Q04

A photographer thinks the subject blinked and wants a retake. But the
subject may have walked away on purpose, and calling them back undoes their
choice. In this picture: the blink is the original minimize animation
completing late, walking away on purpose is a fresh user minimize, both look
the same to the photographer as a shown-to-iconic edge during Priming, and
calling the subject back for a retake is a re-prime (another restore).

#### Options for Q04

- Option A: re-prime once within a 250 ms race window, cap open, counted.
  - pro: keeps the halo when the animation outlasts the priming restore.
  - con: restores a fresh user minimize once before replaying it, an extra
    visible return to the foreground that the issue wants to avoid.
- Option B: no re-prime. Any observed minimize during Priming cancels the
  replay and moves to Minimized with the halo composed, with no restore.
  - pro: never undoes a fresh user minimize, and each interception performs
    exactly one restore.
  - pro: removes a restore path entirely.
  - con: in the late animation case, the thumbnail may miss the halo if the
    minimize won the race against the halo composition.
- Option C: re-prime on any observed minimize during Priming, bounded only by
  the cap.
  - pro: more attempts at a halo.
  - con: every attempt is a visible restore, including of fresh user
    minimizes.

#### Recommended option for Q04 (with arguments for this choice)

Option B: the two causes can't be told apart, and restoring a fresh user
minimize is worse than a thumbnail missing its halo. Exactly one restore per
interception is also the simplest guarantee to test.

#### Answer to Q04: option B (with reason why it must be accepted as the answer)

Option B: it respects every observed minimize and removes the last restore
path the issue's foreground guarantee would have to argue around.

### Q05: Cap constants and reset condition

Question description: the design trips the cap on a third interception within
2 s. It stays suspended until 5 s pass with no external minimize edge at all,
counting skipped ones. So a continuing stream of minimizes can't rearm it.
Are the constants and the reset condition right?

#### BBQ for Q05

A fuse trips on the third surge within two seconds. It can rearm after a fixed
delay whatever happens, or only once the line has been completely quiet for a
while. In this picture: the fuse is the interception cap, a surge is an
external minimize edge, tripping is suspension, and "completely quiet" is 5 s
with no external minimize edge, skipped ones included.

#### Options for Q05

- Option A: 2 interceptions within 2 s trips it. It rearms after 5 s with no
  external minimize edge, skipped ones included.
  - pro: a minimize, restore and minimize within 2 s keeps both halos.
  - pro: a continuing stream never rearms it, so a residual loop is stopped.
- Option B: 1 interception within 2 s trips it, with the same reset.
  - pro: at most one visible restore in any 2 s.
  - con: a quick second minimize loses its halo.
- Option C: 2 within 2 s, then a fixed 5 s suspension measured from the last
  interception.
  - pro: simple timer.
  - con: skipped minimizes don't count, so a continuing stream rearms it every
    5 s and the loop resumes.

#### Recommended option for Q05 (with arguments for this choice)

Option A: normal use never reaches a third interception within 2 s, and
counting skipped edges toward the quiet period is what makes the cap stop a
loop instead of pacing it.

#### Answer to Q05: option A (with reason why it must be accepted as the answer)

Option A: it stays invisible in normal use and can't be rearmed by the loop it
exists to stop.

### Q06: Handling an uncertain own call

Question description: when an own call's after-reading doesn't match its
expected state, the phase becomes Unsettled until a 1 s deadline, then follows
the current reading. If the own call applies after that deadline, at any
delay, its transition shows up as an observed edge. A suspension or a count
over a time window can expire before it applies: a delayed own replay at 7 s
would be intercepted once a 5 s quiet period has passed, and a second
uncertain call after 60 s would escape a 60 s latch window. What should an
Unsettled resolution do?

#### BBQ for Q06

A courier rings, gets no answer, and leaves. The door may open any time later,
and the house would think a stranger came. The depot can pause the route for a
while, stop it after two misses within an hour, or stop it for good on the
first miss until the route is set up again. In this picture: ringing is an
own `ShowWindow` call, no answer is an after-reading that doesn't match, the
door opening later is the own call applying after the deadline, pausing the
route is a cap suspension, two misses within an hour is a time-windowed latch,
and stopping until the route is set up again is latching interception off
until the host restarts.

#### Options for Q06

- Option A: the first Unsettled resolution latches interception off for the
  host session, logged. Every later shown-to-iconic edge is skipped with
  reason `latched`, whenever the delayed own call applies.
  - pro: an absolute bound at any delay: the only restore is the one already
    in flight, and nothing is intercepted afterwards.
  - pro: minimizes keep working; only thumbnail halos are lost until the host
    restarts (reload, settings change or a new window).
  - con: a single uncertain call disables halos in thumbnails for that window
    for the rest of the session.
- Option B: trip the cap at once, and latch on a second resolution within
  60 s.
  - pro: tolerates a one-off glitch.
  - con: a delayed own call after the 5 s quiet period is intercepted, and a
    second resolution after 60 s never latches, so there is no any-delay
    bound.
- Option C: stay Unsettled until an edge toward the expected state is
  observed, with no deadline.
  - pro: absorbs the delayed own call whenever it comes.
  - con: if the call never applies, interception stays blocked with the phase
    out of step with the window, and an unrelated edge in the same direction
    is taken for the own call.

#### Recommended option for Q06 (with arguments for this choice)

Option A: an uncertain own call means the one premise the design relies on
has failed. Stopping interception until the next host start is the only
option that bounds the consequence at any delay, and it costs thumbnail halos
only. It also keeps Q01's claim true: once an own call is uncertain, no
observed edge can start an interception.

#### Answer to Q06: option A (with reason why it must be accepted as the answer)

Option A: it turns the design's one open premise into at most the restore
already in flight. The 2.5 s, 7 s and after-90 s acceptance cases cover it.
