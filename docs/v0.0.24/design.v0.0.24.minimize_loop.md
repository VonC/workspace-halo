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
  age of the observed minimized-state transition, approved at the design
  review gate and recorded in the issue.
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
The issue's rule is amended, as approved at the design review gate: the
lateness bound applies to the
age of the observed minimized-state transition. That is the age the user
sees, since a late restore is disruptive because the window visibly
minimized long ago. Decisions don't need notifications at all. The issue
records the amendment in its "Amendment from the v0.0.24 design" section.
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

## File-based IO cost clarification for v0.0.24 minimize_loop

Every decision reads `IsIconic` and `GetTickCount64`, in-process Win32 calls,
and updates a fixed-size in-memory model (phase, last reading, `lastShownAt`,
cap history of at most 2 ticks, latch). Host start seeds that model from one
reading: there is no file or metadata to load. Nothing is persisted, and the
latch ends with the host process. The only file IO is appending lines to
`native-host.log` for each edge, own call, WinEvent, Unsettled change and cap
change; a tick that finds no edge writes nothing.

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

## Design decisions for minimize_loop

| Question | Decision | Integrated in | Rejected alternatives |
| --- | --- | --- | --- |
| Q01 | Decide only from observed `IsIconic` edges, after absorbing the expected transitions of the host's own calls. WinEvents only trigger an extra observation. An uncertain own call disables interception | Context; target behavior; state observation; phase model | Classified events with an own-call ledger (timing proves no origin, and late events regress phases); a hybrid needing event agreement (brings back delivery order) |
| Q02 | Measure lateness as the age of the observed minimized-state transition, bounded by `now - lastShownAt` within 500 ms. A larger bound is `unknown-age` and skipped. `dwmsEventTime` is logged only. This amends the issue's notification-age rule, approved at the review gate | Scope; age bound of an observed edge; logging; issue amendment section | Keep the notification-age rule with a validated `dwmsEventTime` (interception would wait on lagging events); a lead-time measurement spike (no general guarantee) |
| Q03 | Put the pure observation, phase and cap logic in a new `companion/minimize_windows.go` in the same package, tested in `companion/minimize_windows_test.go`. The Win32 calls stay with their callers | Phase model (transitions are a pure function) | Grow `companion/main_windows.go`; a separate Go package (API boundary for a single caller) |
| Q04 | No re-prime. A minimize observed during Priming cancels the replay and moves to Minimized with the halo composed, with no restore and no cap count | Target behavior; external minimize during Priming; removed re-prime; acceptance cases | One re-prime in a 250 ms race window (restores a fresh user minimize); re-prime bounded only by the cap |
| Q05 | Cap: a third interception within 2 s is skipped and starts suspension. Suspension ends only after 5 s with no external minimize edge, skipped ones included | Interception cap; acceptance cases | One interception per 2 s (a quick second minimize loses its halo); a fixed 5 s suspension from the last interception (a continuing stream rearms it) |
| Q06 | The first Unsettled resolution (own call's after-reading differs from its expected state, past the 1 s deadline) latches interception off for the host session. Later edges are skipped with reason `latched` until the host restarts | Absorbing own transitions; uncertain outcome of an own call; target behavior; acceptance cases | Trip the cap and latch on a second resolution within 60 s (expires before a late own call applies); stay Unsettled with no deadline (can stick, and misattributes an unrelated edge) |
