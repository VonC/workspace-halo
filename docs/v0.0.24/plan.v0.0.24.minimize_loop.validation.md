# v0.0.24 minimize_loop implementation tracking and validation

No, it is not implemented.

This document tracks the five steps of
[plan.v0.0.24.minimize_loop.md](plan.v0.0.24.minimize_loop.md): the minimize
code split, the observation model with own-call absorption and session latch,
the interception cap, the acceptance scenarios with documentation and the VSIX
build, and the manual unplug. Steps 1 to 4 are implemented; Step 5, the manual
three-to-one unplug on the committed build, has not started.

> Initial-skeleton note: this first version was written by the `write-plans`
> skill, before any implementation check. Every section that needs a check
> holds the placeholder `_(empty: no check has taken place yet.)_.`, and an
> implementation check replaces it with findings.
>
> Markdown lint note: never leave a space immediately inside an inline code span
> (MD038); write a needed space as the token `[space]`, as in `` `[space]${x}` ``.
> The empty placeholder ends in `)_.` so the line is not pure italic text (MD036).

---

## File-based IO cost clarification for v0.0.24 minimize_loop (implementation)

All implementation work must respect the IO rules of the plan's "File-based IO
cost clarification for v0.0.24 minimize_loop" section:

- No file read, directory scan or persisted state is added; the model, the
  cap and the session latch stay in memory.
- Decisions use `IsIconic` and `GetTickCount64` only.
- `native-host.log` gets one line per edge, own call, matched WinEvent,
  Unsettled entry or resolution, and cap change, never one per quiet tick.
- `getTickCount64` resolves its `kernel32` export once, from the proc `var`
  block.

---

## Complexity Bound Clarification for v0.0.24 minimize_loop (implementation)

The scaling target for all v0.0.24 minimize_loop code paths is:

- **O(1) amortized per hot-loop event**: one `IsIconic` reading, one pure
  transition over a fixed-size state, and at most two own `ShowWindow` calls
  per tick or matched WinEvent.
- **O(n) total per phase**: host start seeds the model from one reading; no
  state grows with the number of minimizes.

Every implemented step is reviewed against this bound in its Performance check
section.

---

## Step 1. Extract the minimize responsibility from main_windows

### Analysis of Step 1 implementation state

Yes. Step 1 has been fully implemented.

The minimize constants, types, `minimizeEventTransition` and
`minimizeTransition` now live in `companion/minimize_windows.go`, the hook,
callback and priming, composition and replay functions in
`companion/minimize_hook_windows.go`, and the three minimize tests in
`companion/minimize_windows_test.go`, all moved verbatim. The Go suite still
reports 33 `--- PASS` lines, `main_windows.go` is at 1846 lines (target
<= 1850) and `main_windows_test.go` at 604 (target <= 650). The two criteria
the first code-review round found unmet are resolved as the plan owner chose:
a `renderOverlay` type change carried in the move commit makes `go vet`
clean, and the plan's
test-file grep now names the minimize-interception symbols and prints
nothing.

### Goal for Step 1

Move the pure minimize code to `companion/minimize_windows.go`, the minimize
Win32 glue to `companion/minimize_hook_windows.go`, and the three minimize
tests to `companion/minimize_windows_test.go`, verbatim and with no behavior
change, so that the test file drops under the 650-line ceiling and
`main_windows.go` drops to at most 1850 lines.

### Step 1 improvement expectations

- `companion/main_windows_test.go` is at most 650 lines and has no minimize
  test left.
- `companion/main_windows.go` is at most 1850 lines and defines none of the
  moved functions.
- The Go suite passes with the same `--- PASS` count as before the move.
- `gofmt -l` prints nothing and `go vet` is clean.

### What was implemented for Step 1

- **Pure minimize file**: `companion/minimize_windows.go` (72 lines, no
  imports) holds `eventSystemMinimizeStart`, `eventSystemMinimizeEnd`,
  `objidWindow`, `childidSelf` and `wineventOutofcontext` (moved out of the
  main `const` block), `minimizePhase` with its four constants,
  `minimizeAction` with its four constants, `minimizeReplayDelayMS`,
  `minimizeEventTransition` and `minimizeTransition`, with their original
  bodies and comments.
- **Win32 glue file**: `companion/minimize_hook_windows.go` (133 lines,
  imports `fmt`, `syscall`, `unsafe`) holds
  `var minimizeWinEventCallback = syscall.NewCallback(minimizeWinEventProc)`,
  `installMinimizeHook`, `minimizeWinEventProc`,
  `restoreTargetForMinimizePriming`, `composeHalo` and
  `replayPendingMinimize`, with their original bodies and doc comments.
- **Main file shrink**: `companion/main_windows.go` went from 2022 to 1846
  lines (1844 after the move, plus 2 comment lines of the vet fix below);
  its `var` block keeps only `activeApp`, its imports are unchanged,
  and the shared declarations (`procIsIconic`, `procShowWindow`,
  `procSetWinEventHook`, `procUnhookWinEvent`, the `sw*` constants) stay in
  place as planned. The callers (`main`, `close`, `tick`, `visibilityState`
  input) are untouched.
- **Test move**: `TestMinimizeTransitionTargetsOnlyTheTrackedTopLevelWindow`,
  `TestMinimizeEventTransitionPrimesThenAcceptsTheReplay` and
  `TestDuplicateMinimizeStartDoesNotRestartPriming` moved verbatim to
  `companion/minimize_windows_test.go` (66 lines, imports `testing`);
  `companion/main_windows_test.go` went from 661 to 604 lines, and its
  imports all stay in use.
- **File headers**: each new file carries a short comment after the package
  clause stating its responsibility; no moved comment was changed.
- **Vet fix, in the move commit**: `go vet` reported
  `possible misuse of unsafe.Pointer` on
  `unsafe.Slice((*byte)(unsafe.Pointer(bits)), w*h*4)` in `renderOverlay`, a
  finding already present on HEAD `6752442` (line 1505, checked in a
  temporary worktree). As the plan owner chose after review round 1,
  `renderOverlay` now declares the `CreateDIBSection` pixel address `bits` as
  `unsafe.Pointer` instead of `uintptr` and slices `(*byte)(bits)` directly,
  with a comment giving the reason. The same address reaches the same slice,
  so rendering is unchanged; the plan records this fix under Step 1. It rides
  in the move commit because both touch `main_windows.go` and a commit group
  stages whole files.
- **Test-file grep amended**: the plan's bare `git grep -n 'minimize'` also
  matched the `minimized` trigger field of `TestVisibilityStatePrecedence`,
  a `visibilityState` test that stays in `main_windows_test.go`. The plan now
  checks the minimize-interception symbols,
  `minimize(Transition|EventTransition|Idle|Priming|Replaying|Committed)` or
  `eventSystemMinimize`; it prints nothing on `main_windows_test.go` and
  finds 17 lines in `minimize_windows_test.go`.
- **Validation evidence**: `go test -v` gives 33 `--- PASS` lines before and
  after the move and after the vet fix; `gofmt -l` prints nothing;
  `go vet ./...` exits 0 with no finding; the three `main_windows.go`
  completion greps and the amended test-file grep print nothing; `ghog day`
  ends at `exit=9` ("not a pytest project"); `scripts\test-companion.ps1`
  prints `ok  workspace-halo/companion` and exits 0; `npm test` exits 0
  (8 pass).

### New types or classes introduced for Step 1

- No new type: `minimizePhase` and `minimizeAction` only moved files.
- `companion/minimize_windows.go`, `companion/minimize_hook_windows.go` and
  `companion/minimize_windows_test.go`: new files in `package main` that
  hold the moved minimize code and tests, completed with functions only.

### Architecture check for Step 1

- **Pure versus Win32 split**: `minimize_windows.go` has no import and no
  `proc*` reference, so the minimize decision logic is now a Win32-free
  domain file, as design Q03 and plan Q01 require; the Win32 calls sit with
  their callers in `minimize_hook_windows.go`, which acts as the adapter.
- **Boundary direction**: the hook file depends on the pure file and on the
  shared proc declarations of `main_windows.go`; the pure file depends on
  nothing. No new dependency points from the pure logic to Win32.
- **Shared adapter state**: the glue still reads and writes
  `application.minimizeState` and `minimizeReplayAt` and uses the global
  `activeApp`, as before; Step 2 replaces that state with
  `minimizeController`, so this is carried over unchanged, not introduced.
- **Vet fix placement**: the `bits` type change stays inside
  `renderOverlay`, the rendering adapter that owns the `CreateDIBSection`
  call; it adds no dependency.
- **Girth**: `main_windows.go` remains at 1846 lines, over the 650-line
  ceiling; the plan defers its remaining rendering, taskbar and
  target-acquisition split to a later effort.

Yes, there is something to address: the girth of `main_windows.go`, outside
this step's scope and deferred by the plan; no DDD-Hexagonal violation is
introduced by the move or the vet fix.

### Performance check for Step 1

- **No new `O(n^2)` or `O(n log n)` path**: the step moves code between
  files of the same package without changing any moved body, and the vet fix
  only changes a variable's type, so no computation is added.
- **Hot-path bound**: the tick and WinEvent callback paths run the same
  statements as before.
- **Startup or background path**: `minimizeWinEventCallback` is still built
  once at package initialization; Go orders package-level initialization by
  dependency, not by file, so moving it changes nothing.
- **Vet fix cost**: the `bits` type change alters no statement count; the
  render path does the same work.
- **Plan-bound alignment**: the step stays within the plan's O(1) per event
  bound, and adds no file IO.

No, there is no performance issue that needs to be addressed for Step 1.

### Unit test coverage check for Step 1

The repository has no pytest suite and no coverage gate; the Go tests run
under `scripts\test-companion.ps1` without a coverage threshold, so coverage
here is established statically from the tests' branches.

- **`minimize_windows.go`**: covered at 100% of its statements by
  `minimize_windows_test.go`. `minimizeTransition` has the filter mismatch,
  start, end and other-event branches in its table;
  `minimizeEventTransition` has the Idle start, Replaying start and default
  start (duplicate in Priming), Priming end and other end branches.
- **`minimize_hook_windows.go`**: Win32 glue with no unit test, before or
  after the move; the plan (Q08) keeps it evidence-only, and Step 2 adds its
  controller tests through a fake window.
- **`main_windows_test.go`**: legacy test file impacted by the removal of the
  three moved tests; `main_windows.go` stays below 100%, as before, because
  most of it is Win32 rendering, message-loop and target-acquisition code.
  The plan defers that file to a later split effort.

Yes, there is a unit-tested class below 100% that needs completing for
Step 1: `main_windows.go` stays below 100% (legacy, unchanged by the move,
deferred by the plan), while the new `minimize_windows.go` is at 100%.

### Feature integrity for Step 1

- **Existing feature behavior**: no moved body changed, the same functions are
  called from the same places, and the Go suite passes with the same 33
  results, so minimize interception, rendering and occlusion behave as
  before.
- **Reporting or diagnostics**: every `native-host.log` line of the minimize
  path (`minimize intercepted`, `minimize replay ...`, `minimize end`,
  `minimize re-primed ...`, the render errors) is emitted by the same moved
  code.
- **Rendering**: the vet fix keeps the same pixel address and the same
  `w*h*4` slice, so the overlay bitmap is written exactly as before.
- **Compatibility or rollout note**: no TypeScript change; `npm test` stays
  green.

No existing feature or reporting capability appears impaired.

---

## Step 2. Observation model, own-call absorption and session latch

### Analysis of Step 2 implementation state

Yes. Step 2 has been fully implemented.

The event-order state machine is gone: `companion/minimize_windows.go` now
holds the pure `minimizeModel` (phases Shown, Priming, Minimized and
Unsettled, 500 ms lateness bound, 1 s settle timeout, session latch, no
re-prime), and `companion/minimize_hook_windows.go` holds the
`minimizeController` that runs it on every tick and every matched minimize
WinEvent through the three-method `minimizeWindow` seam, logging every
decision. `main_windows.go` is wired to the controller and stays at 1846
lines. Every planned test is present and passes, the pure-model coverage gate
finds no uncovered block, and the six completion greps give the expected
output. Review round 1 asked for a post-latch edge older than 500 ms to
report `latched`. The writer keeps `unknown-age` for that edge, because plan
Step 3 fixes the precedence as `unknown-age`, then `latched`, then `cap`,
following the order of the design's target-behavior pseudo-code, and either
reason skips the edge. `TestMinimizeModelLatchedSkipsEveryLaterEdge` now
asserts that case, so the precedence is tested. One planned test assertion
could not be written as worded, because it contradicts the design's Priming
rule; the test checks the same stamping fact through the cancel-replay line
instead (see "What was implemented for Step 2").

### Goal for Step 2

Replace the event-order state machine with a pure `minimizeModel` driven by
`IsIconic` readings and ticks (phases Shown, Priming, Minimized and Unsettled,
500 ms lateness bound, 1 s settle timeout, session latch, no re-prime), run by
a `minimizeController` on every tick and every matched minimize WinEvent
through a three-method window seam, with every decision logged.

### Step 2 improvement expectations

- Late, duplicated, lost or reordered minimize WinEvents cause no restore;
  each only logs `minimize event: start|end age=<n>ms` and one observation.
- A prompt shown-to-iconic edge is intercepted once and replayed, with the
  kept `minimize intercepted`, `minimize replay requested after halo
  composition` and `minimize replay accepted with composed halo` lines.
- An edge with an age bound over 500 ms is skipped with reason
  `unknown-age`; an edge during Priming cancels the replay with no restore.
- An own call with an unexpected after-reading enters Unsettled; its
  resolution latches interception off. Later prompt edges are skipped with
  reason `latched`; edges over the 500 ms age bound retain `unknown-age`.
- An own restore that leaves the window iconic composes nothing and logs
  `minimize intercepted: restored=false replay=none`; every own call is
  stamped with the tick taken after its after-reading.
- The pure-model coverage gate finds no uncovered block in
  `minimize_windows.go`.
- No reference to `minimizeEventTransition`, `replayPendingMinimize`,
  `restoreTargetForMinimizePriming`, `minimizeState` or `minimizeReplayAt`
  remains, and `main_windows.go` does not grow.

### What was implemented for Step 2

- **Pure observation model**: `companion/minimize_windows.go` (279 lines, no
  import, no `proc*` reference) deletes the Idle, Replaying and Committed
  phases, the old `minimizeAction` values and `minimizeEventTransition`. It
  adds the phases `minimizeShown`, `minimizePriming`, `minimizeMinimized`
  and `minimizeUnsettled`, the constants `minimizeLatenessBoundMS = 500` and
  `minimizeSettleTimeoutMS = 1000` (`minimizeReplayDelayMS = 75` kept), and
  `minimizeModel` with `newMinimizeModel`, `observe`, `ownCall`,
  `replayDue` and `showsMinimizedTrigger`. `observe` first resolves an
  expired Unsettled phase from the last reading it held and latches. It then
  handles the edge against that reading with the current phase's rules, so
  the phase always follows the new reading, and it records `lastShownAt` on
  every shown reading. A shown-to-iconic edge in Shown is skipped as
  `unknown-age` when its bound exceeds 500 ms, even after the latch; otherwise
  it is skipped as `latched` when latched, or intercepted. In Priming it
  cancels the replay with the halo composed. In Unsettled it is absorbed when
  it moves toward the expected state, and otherwise it is only logged. An
  iconic-to-shown edge is honored from Minimized.
- **Own-call absorption**: `ownCall(expectIconic, afterIconic, now)` takes the
  after-reading as the last reading. It starts Priming with
  `replayAt = now + 75`, completes Minimized with the halo, or enters
  Unsettled with `settleDeadline = now + 1000` on any mismatch.
- **Controller and window seam**: `companion/minimize_hook_windows.go` (251
  lines, imports `fmt`, `log`, `syscall`, `unsafe`) adds the `minimizeWindow`
  interface (`isIconic`, `showWindow`, `composeHalo`) and `minimizeController`
  (`model`, `window`, `logger`, `clock`). `newMinimizeController` seeds the
  model from one reading at `clock()`. `observe(now)` logs the latch line
  `minimize interception disabled: own call unsettled (expected=<state>
  observed=<state>)` and the `minimize edge` line, intercepts, and runs a due
  replay. `onEvent` logs `minimize event: start|end age=<n>ms` and observes.
  `ownRestore` (`SW_SHOWNOACTIVATE`, one `SW_RESTORE` fallback while still
  iconic) and `ownReplay` (`SW_MINIMIZE`) read before and after, take
  `clock()` after the after-reading, and log `own restore: ...
  fallback=<bool>`, `own replay: ...` and `own call unsettled: ...`. An
  intercept composes the halo and logs `restored=true replay-in=75ms` only
  when the restore settled shown. Otherwise it logs `restored=false
  replay=none` after the unsettled line. The kept replay lines stay, and the
  legacy `minimize end` and re-prime lines are gone.
- **Win32 adapter**: `*application` implements the seam with `isIconic`
  (`procIsIconic`), `showWindow` (`procShowWindow`) and the existing
  `composeHalo`. `minimizeWinEventProc` reads its seventh parameter as
  `dwmsEventTime` and calls `app.minimize.onEvent(starting,
  uint32(now)-uint32(dwmsEventTime), now)`, with a nil guard on the
  controller. `restoreTargetForMinimizePriming` and `replayPendingMinimize`
  are deleted.
- **Wiring in `main_windows.go`**: `minimizeState` and `minimizeReplayAt` are
  replaced by `minimize *minimizeController`. `main` builds it with
  `newMinimizeController(app, logger, getTickCount64)` before
  `installMinimizeHook`. `tick` calls `a.minimize.observe(now)` in place of
  `replayPendingMinimize` and passes
  `minimized != 0 || a.minimize.model.showsMinimizedTrigger()` to
  `visibilityState`. `procGetTickCount64` joins the `kernel32` proc block and
  `getTickCount64` calls it. The file stays at 1846 lines (net 0; the plan's
  advisory estimate was net -1).
- **Model tests**: `companion/minimize_windows_test.go` (410 lines) keeps
  `TestMinimizeTransitionTargetsOnlyTheTrackedTopLevelWindow`, deletes the two
  event-order tests, and adds the ten planned `TestMinimizeModel...` tests.
  Since review round 1, `TestMinimizeModelLatchedSkipsEveryLaterEdge` also
  feeds a post-latch edge with a 501 ms bound and asserts `unknown-age` with
  the latch kept, pinning the plan's skip-reason precedence. It
  also adds `TestMinimizeModelUnsettledRestoreAbsorbsTheLateRestore`, which
  covers an unsettled restore and resolution followed by an edge in the same
  observation, and `TestMinimizeModelNamesItsEdgesActionsAndStates`. It adds
  `FuzzMinimizeModelObservations` with nine `f.Add` seeds: the recorded
  reordered sequence, a stalled thread, an Unsettled replay applied late, a
  restore during Priming, an unsettled restore, minimize-restore-minimize, a
  one-second edge stream, an iconic host start, and late events in Priming.
- **Controller tests**: new `companion/minimize_hook_windows_test.go` (286
  lines) holds `fakeMinimizeWindow` (scripted `iconic`, a `showWindow` that
  advances the fake clock by `showDelayMS` and applies or defers the change,
  a command log, a composition counter and an optional compose error), a
  buffered logger, the six planned `TestMinimizeController...` tests, and
  `TestMinimizeControllerLogsRenderErrorsSkipsAndHonoredRestores`.
- **Planned test deviation**: the plan asks
  `TestMinimizeControllerStampsOwnCallsAfterTheAfterReading` for "an external
  minimize 30 ms after the after-reading is intercepted with `age<=30ms`".
  After an own restore the model is in Priming, and there a shown-to-iconic
  edge cancels the replay and is never intercepted (design Q04, and this
  plan's own `TestMinimizeModelEdgeDuringPrimingCancelsTheReplayWithoutRestore`).
  So the test asserts the same stamping fact on the edge line that follows:
  `minimize edge: shown->iconic age<=30ms action=cancel-replay`, with neither
  `unknown-age` nor `age<=150ms` present. It also asserts `replayAt` = 1140 +
  75 and an unsettled deadline of 1260 + 1000 with a 120 ms `ShowWindow`.
- **Validation evidence**: `go test -v` gives 51 `--- PASS` lines (33 - 2
  removed + 20 added); `gofmt -l` prints nothing; `go vet ./...` exits 0. The
  first five completion greps print nothing, and the sixth finds one line
  (`main_windows.go:161`, in the proc `var` block). The pure-model coverage
  gate over `a.cover.minimize.out` passes. `ghog day` ends at `exit=9`
  ("not a pytest project"), `scripts\test-companion.ps1` prints
  `ok  workspace-halo/companion` and exits 0, and `npm test` exits 0 (8 pass).
- **Line-budget variance (advisory)**: `minimize_windows.go` 279 (about 250
  expected), `minimize_hook_windows.go` 251 (about 230),
  `minimize_windows_test.go` 410 (about 330), `minimize_hook_windows_test.go`
  286 (about 230). All stay below 550, so no split applies.

### New types or classes introduced for Step 2

- `minimizeModel`: the pure, value-typed interception state of one window,
  with its transitions `observe`, `ownCall`, `replayDue` and
  `showsMinimizedTrigger`.
- `minimizeDecision`: the result of one observation (edge, age bound, action,
  reason, `latchedNow`).
- `minimizeEdge`: none, shown-to-iconic or iconic-to-shown, with its log name.
- `minimizeAction` (redefined): none, intercept, cancel-replay, skip,
  restore-honored, absorbed or settled, with its log name.
- `minimizeWindow`: the three-method port the controller uses to reach the
  window; `*application` is its Win32 adapter.
- `minimizeController`: the application service that owns the model, executes
  its actions through the port, absorbs its own calls and logs decisions.
- `fakeMinimizeWindow` and `minimizeControllerFixture`: test-only scripted
  window and controller fixture.

### Architecture check for Step 2

- **Pure domain model**: `minimize_windows.go` has no import and no `proc*`
  reference (the fifth completion grep prints nothing). Every decision is a
  value transition over `(iconic, now)` readings, so the domain stays
  Win32-free as design Q03 requires.
- **Port and adapter**: the controller depends only on the `minimizeWindow`
  interface, a `*log.Logger` and a clock function, never on `proc*`
  declarations. The Win32 calls sit in the `*application` adapter methods
  and in the hook callback. Dependencies point from adapter to controller to
  model, never back.
- **Controller placement**: the controller shares `minimize_hook_windows.go`
  with the Win32 adapter methods, as plan Q01 decided. It is an application
  service beside its adapter in one file. The seam keeps them separable, but
  the file mixes two roles.
- **Global state carried over**: the callback still reaches the controller
  through the global `activeApp`, as the pre-existing hook did. The step adds
  a nil guard and no new global.
- **Girth**: `main_windows.go` stays at 1846 lines, over the 650-line ceiling.
  This step does not grow it, and the plan defers its split.

Yes, there is something to address: the controller and the Win32 adapter
share one file (as plan Q01 chose), the global `activeApp` is carried over,
and `main_windows.go` is still oversized (deferred by the plan). No
DDD-Hexagonal violation is introduced.

### Performance check for Step 2

- **No new `O(n^2)` or `O(n log n)` path**: every model transition is a
  constant-size switch over a fixed struct, and no collection grows with the
  number of minimizes or events.
- **Hot-path bound**: each tick and each matched WinEvent costs one `IsIconic`
  reading and one pure transition. An intercept adds at most two
  `ShowWindow` calls with three readings, and a replay adds one `ShowWindow`
  call with two readings. The tick keeps its existing visibility `IsIconic`
  reading, so a tick now makes two constant-time `IsIconic` calls instead
  of one.
- **Startup or background path**: host start seeds the model from one
  reading; `GetTickCount64` is now resolved once from the proc block instead
  of on every call.
- **File IO**: log lines are written only on an edge, an own call, a matched
  WinEvent, an Unsettled change or a latch; a quiet tick writes nothing.
- **Plan-bound alignment**: the step stays within the plan's O(1) per event
  bound.

No, there is no performance issue that needs to be addressed for Step 2.

### Unit test coverage check for Step 2

The repository has no pytest suite and no configured coverage threshold. The
plan's pure-model coverage gate (Q08) measures only `minimize_windows.go`, and
this check reads its result from the profile; the hook file is evidence-only.

- **`minimize_windows.go`**: 100% of its statements, from
  `minimize_windows_test.go`; the gate finds no zero-count block, and
  `go tool cover -func` reports 100% for every function.
- **`minimize_hook_windows.go` controller**: `newMinimizeController`,
  `observe`, `onEvent`, `logEdge`, `intercept`, `replay`, `ownRestore`,
  `ownReplay` and `settleOwnCall` are at 100%, from
  `minimize_hook_windows_test.go`.
- **`minimize_hook_windows.go` Win32 adapter**: `installMinimizeHook`,
  `minimizeWinEventProc`, `isIconic`, `showWindow` and `composeHalo` are at
  0%. They call Win32 on a real window and are unreachable in unit tests, as
  plan Q08 accepts. Every one of them is referenced inside the package: by
  `main`, by the `minimizeWinEventCallback` variable, and through the
  `minimizeWindow` interface.
- **Fuzz invariant on own restores**: "every intercept is followed by exactly
  one own restore" is enforced by the fuzz harness, which applies exactly one
  `ownCall(false, ...)` per intercept before the next observation, rather
  than checked as a model property.
- **`main_windows.go`**: legacy, below 100% as before (Win32 rendering,
  message loop and target acquisition); the plan defers that file.

Yes, there is a unit-tested class below 100% that needs completing for
Step 2: `minimize_hook_windows.go` is below 100% because of its five Win32
adapter functions (evidence-only by plan Q08), and `main_windows.go` stays
below 100% (legacy, deferred). No, none of the top-level symbols of the files
outside the gate is unreferenced.

### Feature integrity for Step 2

- **Existing feature behavior**: a prompt user minimize is still restored
  once, composed with the halo and replayed after 75 ms, so the thumbnail
  halo is kept. The intended changes are that late, duplicated or reordered
  events no longer re-intercept, that there is no re-prime, and that minimizes
  of unknown age or after the latch go through without the halo.
- **Visibility trigger**: the `minimized` trigger stays on in every phase
  except Shown, as the old non-Idle test did, so the halo still shows during
  Priming and Unsettled.
- **Reporting or diagnostics**: the kept lines `minimize intercepted:
  restored=true replay-in=75ms`, `minimize replay requested after halo
  composition` and `minimize replay accepted with composed halo` are written
  as before. `minimize end` and the re-prime line are replaced by the
  `restore-honored` edge line and the `minimize event: end` line, as plan
  Q10 decided. New lines log edges, events with their age, own calls,
  Unsettled outcomes and the latch.
- **Compatibility or rollout note**: the replay's halo is composed right
  before `SW_MINIMIZE`, as before. The second composition that the old code
  made when the replayed `MinimizeStart` arrived is gone, because the tick
  keeps rendering the halo while the `minimized` trigger holds. No
  TypeScript change, and `npm test` stays green.

No existing feature or reporting capability appears impaired.

---

## Step 3. Per-window interception cap with quiet-period reset

### Analysis of Step 3 implementation state

Yes. Step 3 has been fully implemented.

`companion/minimize_windows.go` now holds the pure `minimizeCap` (2
interceptions in 2000 ms, a 5000 ms quiet period) as field `cap` of
`minimizeModel`, run by `observe` in the plan's fixed order: resolve an
expired Unsettled, resume a quiet cap, record every external shown-to-iconic
edge as an attempt while suspended, then skip with the precedence
`unknown-age`, `latched`, `cap` (a first closed cap trips here), or intercept
and record its tick. `minimizeController.observe` logs the trip and resume
lines. Every planned test is present and passes, the fuzz target checks the
four cap properties, the two completion greps find the three constants and
both log formats, and the pure-model coverage gate finds no uncovered block.
`minimize_windows_test.go` went past 550 lines, so the plan's split guidance
was applied: the fuzz target moved to `minimize_fuzz_windows_test.go` and the
cap tests to `minimize_cap_windows_test.go`.

### Goal for Step 3

Add a pure `minimizeCap` to the model: at most 2 interceptions within 2000 ms,
a third external minimize edge skipped with reason `cap` and suspension
started, and a resume only after 5000 ms with no external shown-to-iconic edge
at all, with the trip and the resume logged.

### Step 3 improvement expectations

- Minimize, restore, minimize within 2 s gives two interceptions and both
  halos.
- A third external minimize edge within 2 s is skipped with reason `cap` and
  logs `minimize interception suspended: 2 intercepts in 2000ms`.
- A stream of edges every second for 20 s keeps the cap closed, and it logs
  `minimize interception resumed after 5000ms quiet` 5 s after the last edge.
- An edge during Priming is never counted as an interception.
- Every external shown-to-iconic edge moves the quiet timer while suspended,
  whatever its skip reason (`unknown-age`, `latched` or `cap`), so no resume
  happens less than 5000 ms after the latest edge.
- The pure-model coverage gate still passes, including
  `minimize_cap_windows.go` if the cap was split out.

### What was implemented for Step 3

- **Cap constants and reason**: `companion/minimize_windows.go` (361 lines,
  still no import and no `proc*` reference) adds `minimizeCapCount = 2`,
  `minimizeCapWindowMS = 2000` and `minimizeCapQuietMS = 5000`, and the skip
  reason `minimizeReasonCap = "cap"`.
- **Pure cap value**: `minimizeCap` holds `intercepts
  [minimizeCapCount]uint64` with its fill count `filled`, `suspended` and
  `lastAttemptAt`. The plan names the array a ring; it is kept oldest first
  and shifted by one on each record, which is the same fixed two-slot window
  in O(1) and lets `closed` read the oldest tick at index 0. `closed(now)` is
  suspended, or two recorded interceptions with the oldest less than 2000 ms
  old (the window is exclusive: a third edge exactly 2000 ms after the first
  is intercepted). `recordIntercept(now)` adds a tick. `recordAttempt(now)`
  moves `lastAttemptAt` to `now`, sets `suspended`, and reports `tripped`
  when the cap was not suspended yet. `maybeResume(now)` returns a cleared
  cap and `resumed` once `now - lastAttemptAt >= 5000` while suspended.
- **Observation order**: `observe` calls `maybeResume` right after the
  Unsettled resolution, on every observation with or without an edge.
  `shownToIconic` now takes `now`. After the Unsettled case (an absorbed or
  logged edge), it calls `recordAttempt` for every shown-to-iconic edge while
  suspended, before the Priming cancellation and before any skip, so
  `unknown-age`, `latched` and `cap` edges all move the quiet timer. In Shown
  the precedence is `unknown-age`, then `latched`, then `cap` when
  `closed(now)` (a not-yet-suspended cap trips there and sets `capTripped`),
  else intercept with `recordIntercept(now)` at the decision, so an intercept
  whose restore ends Unsettled still counts. A Priming cancellation is never
  recorded as an interception. `minimizeDecision` gains `capTripped` and
  `capResumed`.
- **Controller log lines**: `companion/minimize_hook_windows.go` (265 lines)
  logs `minimize interception resumed after 5000ms quiet` before the edge
  line on `capResumed`, and `minimize interception suspended: 2 intercepts in
  2000ms` after it on `capTripped`, both formatted from the constants.
- **Cap tests**: new `companion/minimize_cap_windows_test.go` (165 lines) holds
  the helpers `minimizeEdgeAfter` and `trippedMinimizeModel` and the five
  planned tests. `TestMinimizeCapAllowsTwoInterceptionsWithinTheWindow` also
  pins the exclusive window. `TestMinimizeCapTripsOnTheThirdEdgeWithinTwoSeconds`
  checks the trip is reported once and that a late third edge keeps
  `unknown-age` without tripping.
  `TestMinimizeCapStaysClosedDuringAContinuingStream` feeds 20 edges one
  second apart, alternating prompt and `unknown-age`, to the tripped model and
  to a latched copy, and checks `lastAttemptAt`, the suspension, the reason
  precedence and the absence of any resume.
  `TestMinimizeCapResumesAfterFiveSecondsWithoutAnyEdge` covers the T + 4999,
  T + 5000, T + 9998 and T + 9999 boundaries and the resume then intercept in
  one observation. `TestMinimizeCapIgnoresPrimingCancellations` shows that an
  edge after a cancellation is still intercepted as the second one.
- **Fuzz extension**: `minimizeFuzzCapTrace` follows the cap from outside the
  model and checks, on every observation, the four planned properties: no
  three intercepts within 2000 ms, no intercept while suspended (unless it
  resumed in the same observation), `lastAttemptAt` on the latest external
  shown-to-iconic edge while suspended, and no resume less than 5000 ms after
  that edge. Two seeds are added (a cap trip with late and prompt edges then a
  resume, and a one-second edge stream after a trip), so there are 11.
- **Controller test**: `TestMinimizeControllerLogsCapTripAndResume` in
  `companion/minimize_hook_windows_test.go` (323 lines, with a new
  `externalRestore` fixture helper) runs two intercepts, a capped third edge
  with its `reason=cap` edge line and the suspended line, no resume up to
  4980 ms after that edge, the resumed line at 5005 ms, and a new intercept.
- **Test file split (plan split guidance)**: with the cap tests added,
  `minimize_windows_test.go` reached 622 lines, past the 550 threshold. As the
  plan orders, the fuzz target and its decoder moved to new
  `companion/minimize_fuzz_windows_test.go` (170 lines, the Step 2 fuzz
  split), and the cap tests to new `companion/minimize_cap_windows_test.go`.
  `minimize_windows_test.go` is back to 307 lines and keeps the Step 2 model
  tests. `minimize_windows.go` stays at 361 lines, below 550, so no
  `minimize_cap_windows.go` split applies.
- **Validation evidence**: `go test -v` gives 57 top-level `--- PASS` lines
  (51 + 6 added) and 11 fuzz seed passes; `gofmt -l` prints nothing; `go vet
  ./...` exits 0. The first completion grep finds the three constants
  (`minimize_windows.go:109-111`) and the second finds both log formats in
  `minimize_hook_windows.go` (lines 74 and 79) and their test assertions. The
  pure-model coverage gate over `a.cover.minimize.out` passes (60 blocks, none
  at zero). `ghog day` ends at `exit=9` ("not a pytest project"),
  `scripts\test-companion.ps1` prints `ok  workspace-halo/companion` and exits
  0, and `npm test` exits 0 (8 pass). As extra evidence, a mutation that
  disables the `closed(now)` case fails five of the new tests and the fuzz
  seeds. A 30 s `go test -fuzz` session run on a scratch copy of `companion`
  (232444 executions) found no failing input, so nothing was written to the
  repository.
- **Line-budget variance (advisory)**: `minimize_windows.go` 361 (about 330
  expected), `minimize_hook_windows.go` 265 (about 240),
  `minimize_hook_windows_test.go` 323 (about 260). `minimize_windows_test.go`
  was expected at about 460; the split leaves it at 307 plus the two new
  files. Every file stays below 550, and `main_windows.go` is unchanged at
  1846.

### New types or classes introduced for Step 3

- `minimizeCap`: the pure, value-typed interception cap of one window (the
  last two interception ticks, the suspension and the latest attempt tick),
  with `closed`, `recordIntercept`, `recordAttempt` and `maybeResume`.
- `minimizeDecision` (extended): `capTripped` and `capResumed`.
- `minimizeFuzzCapTrace`: test-only external trace of intercept and edge ticks
  for the fuzz cap properties.

### Architecture check for Step 3

- **Pure domain model**: the cap lives in `minimize_windows.go` with the
  model, as design Q03 places the cap logic. The file still has no import and
  no `proc*` reference, and every cap operation is a value transition over
  ticks.
- **Port and adapter**: the controller only reads the two decision flags and
  logs; it takes no cap decision and makes no new Win32 call. No dependency
  points from the model toward the controller or the adapter.
- **Invariant documented in code**: `shownToIconic` records attempts after the
  Unsettled case, because Unsettled only follows an interception and so never
  coexists with a suspended cap; its doc comment says so, and the fuzz trace
  checks the resulting `lastAttemptAt` property.
- **Carried over from Step 2**: the controller still shares
  `minimize_hook_windows.go` with the Win32 adapter (plan Q01), the callback
  still reaches it through the global `activeApp`, and `main_windows.go` is
  still 1846 lines (deferred by the plan). Step 3 does not touch any of them.

Yes, there is something to address: nothing new, but the three Step 2
carry-overs remain (controller and adapter in one file, the global
`activeApp`, the oversized `main_windows.go`). Step 3 introduces no
DDD-Hexagonal violation or smell.

### Performance check for Step 3

- **No new `O(n^2)` or `O(n log n)` path**: `minimizeCap` is a fixed
  two-slot array plus three scalars. `recordIntercept` shifts at most one
  element, and `closed`, `recordAttempt` and `maybeResume` are constant-time
  comparisons.
- **Hot-path bound**: each observation adds one `maybeResume` check and, on a
  shown-to-iconic edge, at most one `closed` check and one record. No new
  Win32 call, allocation or growing history.
- **File IO**: two lines are added, written only when the cap trips or
  resumes; a quiet tick still writes nothing.
- **Test-only cost**: the fuzz trace keeps at most three intercept ticks in a
  slice; it is not on the host path.

No, there is no performance issue that needs to be addressed for Step 3.

### Unit test coverage check for Step 3

The repository has no pytest suite and no configured coverage threshold. The
plan's pure-model coverage gate (Q08) measures only `minimize_windows.go`
(and `minimize_cap_windows.go`, which does not exist since no split was
needed); the hook file is evidence-only.

- **`minimize_windows.go`**: 100% of its statements, from
  `minimize_windows_test.go`, `minimize_cap_windows_test.go` and the seeds of
  `minimize_fuzz_windows_test.go`, the three test files of that one class
  file. The gate finds no zero-count block, and `go tool cover -func` reports
  100% for `closed`, `recordIntercept`, `recordAttempt`, `maybeResume`,
  `observe` and `shownToIconic`, as for every other function.
- **`minimize_hook_windows.go` controller**: `observe`, including its two new
  log branches, stays at 100% with `TestMinimizeControllerLogsCapTripAndResume`,
  and so do the other controller functions.
- **`minimize_hook_windows.go` Win32 adapter**: `installMinimizeHook`,
  `minimizeWinEventProc`, `isIconic`, `showWindow` and `composeHalo` stay at
  0%, unchanged by this step. They call Win32 on a real window, as plan Q08
  accepts, and each is referenced inside the package.
- **`main_windows.go`**: not touched; legacy, below 100%, deferred by the
  plan.

Yes, there is a unit-tested class below 100% that needs completing for
Step 3: as in Step 2, `minimize_hook_windows.go` is below 100% because of its
five Win32 adapter functions (evidence-only by plan Q08), and `main_windows.go`
stays below 100% (legacy, deferred). No, none of the top-level symbols of the
files outside the gate is unreferenced.

### Feature integrity for Step 3

- **Existing feature behavior**: normal use is unchanged. One minimize, or a
  minimize, restore and minimize within 2 s, is intercepted and replayed with
  the halo as in Step 2; only a third interception within 2 s is skipped.
- **Skipped cases**: a capped minimize goes through without the halo, as the
  `unknown-age` and `latched` ones do, and interception comes back only after
  5 s with no external minimize.
- **Reporting or diagnostics**: every Step 2 line is still written; the cap
  adds its `reason=cap` edge lines and the suspended and resumed lines.
- **Compatibility or rollout note**: no TypeScript change, and `npm test`
  stays green. The wiki and changelog updates for the cap belong to Step 4.

No existing feature or reporting capability appears impaired.

---

## Step 4. Acceptance scenarios, documentation and the VSIX build

### Analysis of Step 4 implementation state

Yes. Step 4 has been fully implemented.

Every design acceptance row has a passing controller-level scenario, and the
recorded four-window unplug timeline replays with one restore attempt per
window. The two wiki pages, the log reference and the changelog are updated,
the completion greps and markdownlint pass (only MD013 on table rows), and
`build.bat` packages the VSIX. The first check of this step recorded `No`
because the manual unplug was still part of it. Round 1 of the Step 4 code
review found that the plan installed the committed build for that unplug
while a review can only recommend a commit for a complete step. The plan now
moves the unplug to its own Step 5, after the Step 4 commit (decision Q12),
and this re-check applies the amended Step 4 criteria. The same review round
also led to the versioned `.review-validation` floor (decision Q13).

### Goal for Step 4

Replay every design acceptance case and the recorded four-window unplug
timeline through `minimizeController` with scripted fake windows and late,
reordered events; update `wiki/reference/display-triggers.md`,
`wiki/explanation/how-the-overlay-stays-inside-its-window.md`,
`wiki/reference/logs-and-processes.md` and `CHANGELOG.md`; build the VSIX.

### Step 4 improvement expectations

- `TestMinimizeAcceptanceCases` has one passing sub-test per design acceptance
  row, and the recorded unplug timeline shows at most one restore per window
  and minimize action.
- The timing constants table lists the lateness bound, the settle timeout and
  the cap values, and the explanation page covers the let-through cases.
- `build.bat` exits 0, and markdownlint reports nothing on the four updated
  Markdown files beyond MD013 on table rows.

### What was implemented for Step 4

- **Scenario runner**: new `companion/minimize_scenario_windows_test.go`
  (226 lines) drives one or more `minimizeController`s on a shared fake clock.
  At each millisecond it applies the scripted steps of that tick (external
  minimize, external restore, click, deferred own command, late apply of a
  deferred call, host-thread stall, host restart). Then, unless the thread is
  stalled, it delivers the due WinEvents and runs the 25 ms tick on every
  controller. `minimizeScenarioWindow` wraps `fakeMinimizeWindow` and turns
  every minimized-state change, own calls included, into a WinEvent. The
  n-th event of window i is delivered `lags[(n+i) % len(lags)]` ms later, so
  events can arrive late, reordered or lost (`minimizeEventLost`, or no lag
  list at all). Steps must be in tick order, and `run` can continue a
  scenario for later assertions.
- **Acceptance scenarios**: new `companion/minimize_acceptance_windows_test.go`
  (408 lines). `TestMinimizeAcceptanceCases` has 17 sub-tests, one per
  automated design acceptance row. The "any order, or 3 s late" row has two
  sub-tests, one in order and one reordered. The manual unplug row is the
  rollout check. Each sub-test asserts the restore attempts, the
  `minimize intercepted` lines, the `minimize edge` lines, the final phase on
  the observed state, the decisive log lines in order, and any forbidden
  line. The 20 s edge stream continues past its table checks: it proves no
  resume at 27599, the resume at 27600 (5 s after the last edge), then a new
  intercept. `TestMinimizeAcceptanceClickDuringPrimingStillReplays` and
  `TestMinimizeAcceptanceRestoreAfterReplayStaysRestored` cover the user
  actions around the replay.
  `TestMinimizeAcceptanceRecordedUnplugTimeline` runs four controllers:
  w4, w1 and w2 are minimized at 0, 2.7 s and 4.0 s from 10000, w3 stays
  shown, and every WinEvent arrives 3 to 6 s late and reordered. w2's
  `SW_SHOWNOACTIVATE` never applies, so it uses the single `SW_RESTORE`
  fallback. For each minimized window the test checks one restore attempt,
  one replay, a final Minimized with the halo, one `minimize intercepted` and
  one edge line, and no restore command after the first late event. w3 gets
  no command.
- **Test file split (plan split guidance)**: the runner and the scenarios
  would have taken `minimize_hook_windows_test.go` from 323 to about 950
  lines, so they went to new files as the plan's split guidance asks. The
  acceptance file alone reached 620 lines, in the 550-to-650 risk band, so the
  runner was split out by responsibility. `minimize_hook_windows_test.go`
  (325 lines) only gains a header sentence pointing to the runner that reuses
  `fakeMinimizeWindow`.
- **`wiki/reference/display-triggers.md`** (88 lines): the `minimized` row
  now names a pending replay and an unsettled own call. A new "Minimize
  interception rules" section covers edges, the four interception
  conditions, the skip reasons and their precedence, the Priming
  cancellation, the session latch and the cap. The timing constants table
  gains `Lateness bound` 500 ms, `Settle timeout` 1000 ms,
  `Interception cap` 2 in 2000 ms and `Cap quiet period` 5000 ms.
- **`wiki/explanation/how-the-overlay-stays-inside-its-window.md`**
  (87 lines): the replay section now starts from an observed minimize the host
  did not cause. A new section explains that WinEvents only trigger an
  observation, how own calls are absorbed, why a minimize during Priming
  cancels the replay, and why unknown-age, uncertain own call, capped and
  latched minimizes go through without the halo.
- **`wiki/reference/logs-and-processes.md`** (69 lines): lists the
  `minimize edge`, `minimize event`, `own restore`, `own replay`,
  `own call unsettled`, `minimize interception disabled`, `suspended` and
  `resumed` lines next to the three kept interception lines.
- **`CHANGELOG.md`** (224 lines): a `## 0.0.24` section under `## Unreleased`
  covers the unplug loop fix, one restore per interception with the halo kept
  for prompt minimizes, and the logged let-through cases.
- **Validation evidence**: `go test -v` gives 61 top-level `--- PASS` lines
  (57 + 4), 17 acceptance sub-tests and 11 fuzz seeds. `gofmt -l` prints
  nothing and `go vet ./...` exits 0. The first completion grep finds the
  three constants rows (`display-triggers.md:84-86`), and the second finds
  `CHANGELOG.md:5`. markdownlint-cli2 on the four files reports 13 findings,
  all MD013 on `display-triggers.md` table rows (the priority table and the
  timing constants table), as the plan allows. `ghog day` ends at `exit=9`
  ("not a pytest project"). `scripts\test-companion.ps1` prints
  `ok  workspace-halo/companion`, and `npm test` exits 0 (8 pass). The
  pure-model coverage gate still passes. `build.bat` exits 0 and prints
  `OK: Packaged workspace-halo-0.0.23-3e4d05f-dirty-win32-x64.vsix`, built
  from the uncommitted tree.
- **Line-budget variance (advisory)**: the plan expected
  `minimize_hook_windows_test.go` at about 480 lines; the split leaves it at
  325 plus the two new files (226 and 408), all below 550.
  `display-triggers.md` is 88 (about 77 expected),
  `how-the-overlay-stays-inside-its-window.md` 87 (about 75),
  `logs-and-processes.md` 69 (about 54) and `CHANGELOG.md` 224 (about 219).
- **Plan amendment after code review round 1 (Q12)**: `plan.v0.0.24.minimize_loop.md`
  now has five steps. Step 4 is renamed "Acceptance scenarios, documentation
  and the VSIX build" and ends with the build, and its completion criteria
  drop the manual unplug. The new Step 5, "Manual three-to-one unplug check",
  carries the rollout sequence (commit, commit-named build, install, unplug,
  per-window excerpts, per-action table) and the unplug completion criterion.
  Q11 now points the evidence to the Step 5 section, and the Step 4 split
  guidance names the runner file.
- **Declared review validation floor (Q13)**: new versioned root
  `.review-validation` declares `scripts\test-companion.ps1` and `npm test`,
  one command per line with `#` comments. The llm-shared loader
  `load_project_validation_commands` reads exactly those two commands, so a
  code-review request no longer inherits the built-in `ghog day` floor, which
  always ends at exit 9 in this non-pytest repository.

### New types or classes introduced for Step 4

All new types are test-only and live in `minimize_scenario_windows_test.go`
unless noted:

- `minimizeScenarioKind` and its seven step kinds: the external actions a
  scenario scripts.
- `minimizeScenarioStep`: one action at a tick on one window, with the
  `scenarioMinimize` and `scenarioRestore` shorthands.
- `minimizeScenarioEvent`: a generated WinEvent with its generation and due
  ticks.
- `minimizeScenarioWindow`: a `fakeMinimizeWindow` that emits a WinEvent on
  each minimized-state change and records its restore command ticks and its
  first delivery.
- `minimizeScenario`: the shared clock, the hosts and their logs, the pending
  events, and the stall, with `run`, `apply`, `deliverDue`, `startHost` and
  `requireOrderedLog`.
- `minimizeAcceptanceCase` (in `minimize_acceptance_windows_test.go`): one
  acceptance row and its expected outcome.

No production type is added or changed.

### Architecture check for Step 4

- **No production change**: Step 4 touches only test files, the wiki and the
  changelog, so the pure model, the controller and the Win32 adapter keep the
  Step 3 layout.
- **Tests go through the port**: the runner reaches the controller only
  through its production entry points (`newMinimizeController`, `observe`,
  `onEvent`) and the `minimizeWindow` seam. It reads `model` fields only in
  assertions, as the Step 2 and Step 3 controller tests do. No Win32 call is
  made from a test.
- **Split by responsibility**: the runner (the test harness) and the
  scenarios (the acceptance cases) are separate files, and both reuse the
  one `fakeMinimizeWindow` rather than a copy.
- **Carried over from Step 2**: the controller still shares
  `minimize_hook_windows.go` with the Win32 adapter (plan Q01), the callback
  still reaches it through the global `activeApp`, and `main_windows.go` is
  still 1846 lines (deferred by the plan). Step 4 does not touch any of them.

Yes, there is something to address: nothing new, but the three Step 2
carry-overs remain (controller and adapter in one file, the global
`activeApp`, the oversized `main_windows.go`). Step 4 introduces no
DDD-Hexagonal violation or smell.

### Performance check for Step 4

- **No host-path change**: no production code changed, so the per-tick and
  per-event cost of Step 3 stands.
- **Test-only cost**: the runner steps one millisecond at a time. Each
  millisecond scans the pending events (a handful) and, every 25 ms, observes
  each controller, which is linear in simulated time. The longest scenario
  (the 90 s latch) runs about 91000 iterations. Removing a delivered event
  shifts the short pending slice, and no scenario sorts. The whole Go suite
  still runs in a few seconds.
- **File IO**: none added; the scenario logs go to in-memory buffers.

No, there is no performance issue that needs to be addressed for Step 4.

### Unit test coverage check for Step 4

The repository has no pytest suite and no configured coverage threshold. The
plan's pure-model coverage gate (Q08) measures only `minimize_windows.go`;
the hook file is evidence-only. The new files are controller-level
acceptance tests, which carry no coverage target.

- **`minimize_windows.go`**: unchanged, still at 100% of its statements from
  its three unit test files. The pure-model gate finds no zero-count block.
- **`minimize_hook_windows.go` controller**: unchanged, still at 100% for
  `newMinimizeController`, `observe`, `onEvent`, `logEdge`, `intercept`,
  `replay`, `ownRestore`, `ownReplay` and `settleOwnCall`. The acceptance
  scenarios exercise them again end to end.
- **`minimize_hook_windows.go` Win32 adapter**: `installMinimizeHook`,
  `minimizeWinEventProc`, `isIconic`, `showWindow` and `composeHalo` stay at
  0%, unchanged by this step. They call Win32 on a real window, as plan Q08
  accepts; the Step 5 manual unplug is what exercises them.
- **`main_windows.go`**: not touched; legacy, below 100%, deferred by the
  plan.

Yes, there is a unit-tested class below 100% that needs completing for
Step 4: as in Steps 2 and 3, `minimize_hook_windows.go` is below 100% because
of its five Win32 adapter functions (evidence-only by plan Q08), and
`main_windows.go` stays below 100% (legacy, deferred). No, none of the
top-level symbols of the files outside the gate is unreferenced: every runner
type and helper is used by the acceptance tests.

### Feature integrity for Step 4

- **Existing feature behavior**: no production code changed; the host binary
  behaves as after Step 3, and the existing 57 Go tests and 8 TypeScript
  tests pass.
- **Documentation**: the wiki now describes the observed-edge interception
  that Steps 2 and 3 shipped. It no longer describes the old event-driven
  interception, and it lists every constant and log line the host uses.
- **Reporting or diagnostics**: unchanged; the log reference now documents
  the lines already written since Steps 2 and 3.
- **Compatibility or rollout note**: the changelog entry sits under
  `## 0.0.24` while `package.json` is still 0.0.23; the release step bumps
  the version. The manual unplug evidence belongs to Step 5. The new
  `.review-validation` changes only the code-review floor; `build.bat`,
  `ghog` and the plan's gate loop are unchanged.

No existing feature or reporting capability appears impaired.

---

## Step 5. Manual three-to-one unplug check

### Analysis of Step 5 implementation state

Not started. Step 5 is not implemented because the VSIX has not been built
from the committed Step 4 tree and installed, the three-to-one unplug has not
been run with four windows, and no per-window log excerpt or per-action
verdict table has been recorded.

### Goal for Step 5

Build the VSIX from the committed Step 4 tree, install it, run the manual
three-to-one unplug with four VS Code windows, and record the per-window
timestamped log excerpts and the per-action verdict table here.

### Step 5 improvement expectations

- `build.bat` prints `OK: Packaged` with a VSIX name carrying the Step 4
  commit and no `dirty` marker.
- The manual unplug log shows no repeated interception cycle and no
  unsolicited delayed or repeated return to the foreground, judged per
  external minimize action in a table (window, edge timestamp, age bound,
  decision, interception attempts, own restores after the accepted replay,
  verdict) backed by per-window timestamped log excerpts; one attempt may
  include a single `SW_RESTORE` fallback.

### What was implemented for Step 5

_(empty: no check has taken place yet.)_.

### New types or classes introduced for Step 5

_(empty: no check has taken place yet.)_.

### Architecture check for Step 5

_(empty: no check has taken place yet.)_.

### Performance check for Step 5

_(empty: no check has taken place yet.)_.

### Unit test coverage check for Step 5

_(empty: no check has taken place yet.)_.

### Feature integrity for Step 5

_(empty: no check has taken place yet.)_.
