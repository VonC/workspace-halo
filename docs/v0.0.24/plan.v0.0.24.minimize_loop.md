# v0.0.24 minimize_loop implementation plan -- observed edges instead of event order

The native host stops deciding from WinEvent order and decides from the
window's observed minimized state, in four steps: a split, a pure observation
model, the cap, and acceptance.

- **Split first**: the minimize code leaves the 2022-line
  `companion/main_windows.go` and the 661-line
  `companion/main_windows_test.go` unchanged in behavior, so every later change
  lands in small files.
- **Pure model, thin Win32 glue**: a pure observation, phase and cap model in
  `companion/minimize_windows.go`, driven by a controller whose Win32 calls go
  through a three-method window seam that tests can fake.
- **Evidence everywhere**: every decision is one `native-host.log` line, and
  the final step replays the recorded unplug timeline through the controller
  before the manual three-to-one unplug.

> Markdown lint note: never leave a space immediately inside an inline code span
> (MD038); when a snippet starts or ends with a space, write that space as the
> literal token `[space]`, as in `` `[space]${x}` ``. End any line that would be
> only italic text with a period after the closing underscore (MD036).

## Plan goal for v0.0.24 minimize_loop

Implement the v0.0.24 minimize_loop behavior described in
[design.v0.0.24.minimize_loop.md](design.v0.0.24.minimize_loop.md) and
[issue.v0.0.24.minimize_loop.md](issue.v0.0.24.minimize_loop.md), in four
ordered steps.

- **Step 1 goal**: move the minimize code out of `main_windows.go` and its
  tests out of `main_windows_test.go`, with no behavior change, bringing the
  test file under the 650-line ceiling.
- **Step 2 goal**: replace the event-order state machine with the pure
  observation model (edges, age bound, own-call absorption, Unsettled, session
  latch, no re-prime) and the controller that runs it on every tick and every
  minimize WinEvent.
- **Step 3 goal**: add the per-window interception cap (2 in 2 s, 5 s quiet
  reset counted on every external minimize edge) to the model, with its log
  lines.
- **Step 4 goal**: acceptance tests that replay the design's acceptance cases
  and the recorded unplug timeline through the controller, the documentation
  and changelog updates, the VSIX build, and the manual three-to-one unplug.

---

## Scope anchors for v0.0.24 minimize_loop plan

This plan targets the design's three outcomes:

1. Interceptions start only from an observed shown-to-iconic edge the host did
   not cause, proven prompt (age bound at most 500 ms), not latched and below
   the cap.
2. The minimize phase always follows the observed window state: no delivery
   order, lost event or late event can leave it stale, stuck or looping.
3. A per-window cap that a continuing stream of minimizes can't re-arm backs
   everything up, and every decision is logged with its data.

The following are explicitly **in scope** for this plan:

- `IsIconic` observation on each 25 ms tick, in each matched minimize WinEvent
  callback, and before and after each own `ShowWindow` call.
- The four phases Shown, Priming, Minimized and Unsettled, the 1 s settle
  timeout, and the session latch.
- Removal of `minimizeEventTransition`, the Replaying and Committed phases and
  the re-prime in `replayPendingMinimize`.
- The cap constants and suspension logic.
- Decision log lines, `dwmsEventTime` age logging, wiki and changelog updates.
- Unit, fuzz-seeded property and controller-level acceptance tests, plus the
  manual unplug.

The following are explicitly **deferred** to v0.0.25 and beyond:

- The child-mode `WM_DISPLAYCHANGE` reception follow-up issue, and any pause of
  interception after display changes.
- Removing or replacing the activating `SW_RESTORE` fallback of the priming
  restore (kept, only made non-repeatable).
- Bringing `companion/main_windows.go` itself under the 650-line ceiling
  (Step 1 extracts the minimize responsibility only; the remaining rendering,
  taskbar and target-acquisition responsibilities need their own split effort).

---

## Complexity Bound Clarification for v0.0.24

The scaling target for all v0.0.24 minimize_loop code paths is:

- **O(1) amortized per hot-loop event**: each observation (tick or WinEvent)
  costs one `IsIconic` call, one pure model transition over a fixed-size state
  (the cap keeps at most 2 interception ticks), and at most two own
  `ShowWindow` calls with their `IsIconic` readings.
- **O(n) total per phase**: host start seeds the model with one `IsIconic`
  reading; there is no loading phase and no history that grows with the
  number of minimizes.

No v0.0.24 code path should introduce `O(n^2)` or `O(n log n)` cost on the
tick or callback path. Any new path that breaks this bound must be called out
as a defect before merge.

---

## File-based IO cost clarification for v0.0.24 minimize_loop

- No file read, directory scan or persisted state is added: the model, the
  cap and the session latch live in memory in the single-threaded host, and
  the latch ends with the host process.
- Decisions use `IsIconic` and `GetTickCount64` only, which are in-process
  Win32 calls, not file IO.
- The only file IO is appending lines to the existing `native-host.log`: one
  line per external edge, own call, matched minimize WinEvent, Unsettled
  entry or resolution, and cap trip or resume. No line is written on a tick
  that observes no edge, so a steady window writes nothing.
- There is no loading phase: host start seeds the model from one `IsIconic`
  reading, an in-memory step with no metadata to load.
- `getTickCount64` currently builds a new `kernel32` lazy proc on every call;
  Step 2 hoists it into the existing proc `var` block, so the extra calls this
  plan adds (callback, own calls) don't each resolve the export again.

---

## Confirmed technical facts for v0.0.24 plan viability

These facts come from direct inspection of the tree at commit `1c6346a` on
branch `minimize_loop`.

**Files over the 650-line repository limit** (must be split by
responsibility):

- `companion/main_windows.go`: **2022 lines** -- Step 1 extracts the minimize
  responsibility (constants at lines 90-94, types and `minimizeEventTransition`
  at lines 175-211, `minimizeWinEventCallback` at line 362, and lines 833-965
  from `installMinimizeHook` to `replayPendingMinimize`) into two new files.
  Later steps must not grow it; the rest of the file is out of this effort's
  scope.
- `companion/main_windows_test.go`: **661 lines** -- Step 1 moves the three
  minimize tests (lines 274-329) out, which brings it to about 604 lines.

**Files in the 550-through-650 risk band** (avoid growth where practical):

- None among the files this plan touches.

**Files below 550 and safe to extend** (current lines, expected additions):

- `wiki/reference/display-triggers.md`: 52 -- about +25 (constants rows,
  interception rules, session latch); advisory.
- `wiki/explanation/how-the-overlay-stays-inside-its-window.md`: 45 -- about
  +30 (observed-minimize explanation and the let-through cases); advisory.
- `wiki/reference/logs-and-processes.md`: 46 -- about +8 (the new minimize
  log lines); advisory.
- `CHANGELOG.md`: 211 -- about +8 (the 0.0.24 entry); advisory.

**What does not exist yet (all new for v0.0.24)**:

- `companion/minimize_windows.go` (pure minimize model).
- `companion/minimize_hook_windows.go` (Win32 hook, callback, controller and
  window seam).
- `companion/minimize_windows_test.go` (model, cap and fuzz-seeded property
  tests).
- `companion/minimize_hook_windows_test.go` (controller tests with a fake
  window, and the acceptance scenarios).

**Other confirmed technical facts that affect plan shape**:

- **Go package layout**: `companion` is one `package main` (module
  `workspace-halo/companion`, `go 1.26`, toolchain go1.26.4), and every source
  starts with `//go:build windows`. New files use the `_windows.go` suffix and
  the same build line. Go has no `__init__.py`: nothing to create or update
  for package registration.
- **Test naming adaptation**: the `tests\unit\...\test_filename_tdd.py`
  convention is Python-only. Go tests live beside their code as
  `<file>_test.go` in `package main`, as the design (Q03) names them. The
  property test is a native Go fuzz target whose `f.Add` seed corpus runs
  under plain `go test`, so it needs no fuzzing run to gate.
- **Test gate**: the repository has no `check.bat` and no pytest
  configuration. `ghog day` skips the check step, then stops with `exit=9`
  ("not a pytest project"), which is its complete verdict here. The project's
  own gate is `scripts\test-companion.ps1` (`go test -buildvcs=false ./...`
  with `GOCACHE` under `.gocache`) and `npm test` (types and TypeScript unit
  tests), both called by `build.bat`.
- **Line-count metric**: no big-file gate exists for Go or Markdown in this
  repository; this plan applies the write-plans bands to every touched file
  with the physical line count `(Get-Content -LiteralPath <file>).Count`,
  which matches `wc -l` on these files.
- **Single thread**: `main` calls `runtime.LockOSThread()`; the `WM_TIMER`
  tick (`SetTimer` 25 ms) and the out-of-context WinEvent callback both run on
  that thread while `run` pumps messages, so the controller needs no locking.
- **Shared Win32 declarations stay put**: `procIsIconic`, `procShowWindow`,
  `procSetWinEventHook`, `procUnhookWinEvent` and the `sw*` constants are
  shared with rendering and occlusion code and stay in `main_windows.go`;
  `swShowNoActivate` and `procIsIconic` are also used at lines 1385, 1447 and
  1513.
- **Callers to rewire**: `tick` calls `a.replayPendingMinimize(now)` (line
  974) and feeds `a.minimizeState != minimizeIdle` into `visibilityState`
  (line 1021); `close` unhooks `a.minimizeHook` (line 823); `main` installs the
  hook after the first render (line 410).
- **Callback parameters**: `minimizeWinEventProc` discards its sixth and
  seventh parameters (`idEventThread`, `dwmsEventTime`); `dwmsEventTime` is a
  32-bit `GetTickCount` value, so its age is `uint32(now) - dwmsEventTime` in
  wrapping 32-bit arithmetic.
- **Baseline green**: `scripts\test-companion.ps1` passes on the current tree.

---

## Current test-tree validation snapshot for v0.0.24 minimize_loop

Existing test packages that v0.0.24 must not break:

- `companion/main_windows_test.go` -- 661 lines, 33 tests; after Step 1 it
  keeps its 30 non-minimize tests unchanged and holds no minimize test.
- `test/model.test.ts` -- 85 lines, run by `npm test`; no TypeScript change is
  planned, it only has to stay green.

New test files to create for v0.0.24:

- `companion/minimize_windows_test.go`.
- `companion/minimize_hook_windows_test.go`.

---

## Runtime file note for v0.0.24 minimize_loop plan

- `.gocache/` -- Go build cache from `scripts\test-companion.ps1`, already
  ignored by `/.gocache/`.
- `a.ghog.log`, `a.ghog.status` -- groundhog reports, ignored by `a.*`.
- `a.cover.minimize.out` -- the coverage profile of the validation command
  below, ignored by `a.*`.
- `companion/testdata/fuzz/` -- only written when `go test -fuzz` finds a
  failure; this plan never runs a fuzzing session, so nothing is expected
  there. If a failing input is ever saved, commit it as a regression seed.

---

## Step 0 perf-gate decision for v0.0.24 minimize_loop

No Step 0 is planned. The pytest `timeout` plus `xfail` pattern has no Go
equivalent (Go has no expected-failure marker), and the new code is a pure
constant-size transition per observation with no IO to time. The timing that
matters is behavioral, not CPU: the 500 ms lateness bound, the 75 ms replay
delay, the 1 s settle timeout and the cap windows. Those are driven by
synthetic tick values in the Step 2 and Step 3 unit tests and in the Step 4
acceptance scenarios, so they are deterministic and need no wall-clock gate.

---

## Shared execution command checklist for all v0.0.24 minimize_loop steps

Apply this checklist for every numbered step, filling in the step-specific
files from the step's line-budget checkpoint.

1. Count lines before edits on all step files with the line-count template
   below, and compare with the step's recorded baseline.
2. Write the step's tests first, as listed under the step implementation
   section, and run the targeted tests to see them fail for the expected
   reason (missing symbol or wrong behavior).
3. Implement the step, then run the targeted tests until they pass.
4. Run the step's grep checks and the format and vet checks.
5. Run the shared gate loop until it is green in one pass.
6. Count lines after edits and compare them with the step line-budget
   checkpoint.
7. If any Go file exceeds 650 lines after edits, or `main_windows.go` grows
   past its step ceiling, stop and apply the step's split guidance before
   committing.
8. If a file exceeds only an advisory estimate while staying at or below 650,
   record the variance in the validation plan without failing the step.

---

## Ready-to-run command templates for all v0.0.24 minimize_loop steps

Run from the repository root in PowerShell, substituting the step's files and
test names.

- Line count before and after, once per step file:
  `(Get-Content -LiteralPath <file>).Count`.
- Targeted tests (Go has no `ghog single`; this mirrors
  `scripts\test-companion.ps1` with a `-run` filter; the steps below write
  only the `-run` pattern):

  ```powershell
  $env:GOCACHE = "$PWD\.gocache"
  Push-Location companion
  go test -buildvcs=false -run '<TestRegex>' ./...
  Pop-Location
  ```

- Format and vet checks:
  `Push-Location companion; gofmt -l .; go vet ./...; Pop-Location`
  (`gofmt -l` must print nothing).
- Grep checks: `git grep --untracked -nE '<pattern>' -- <path>` with the
  step's patterns (`rg` is not installed here; `--untracked` also searches
  the new files before they are committed).
- Shared gate loop, repeated fix-and-walk until green in one pass:
  1. `& "<LLM_SHARED_DIR>\bin\ghog.bat" day` (`<LLM_SHARED_DIR>` is the
     local llm-shared checkout) -- expected
     closing `exit=9` ("not a pytest project"); any other non-zero exit is a
     groundhog finding to fix first.
  2. `powershell -NoProfile -ExecutionPolicy Bypass -File scripts\test-companion.ps1`
     -- must print `ok  workspace-halo/companion` and exit 0.
  3. `npm test` -- must exit 0 (no TypeScript change is planned; this proves
     it).
- Coverage profile and per-function evidence for the validation plan:

  ```powershell
  $env:GOCACHE = "$PWD\.gocache"
  Push-Location companion
  go test -buildvcs=false -coverprofile ..\a.cover.minimize.out ./...
  go tool cover -func ..\a.cover.minimize.out | Select-String 'minimize'
  Pop-Location
  ```

  Keep the space-separated flag form: Windows PowerShell 5.1 splits a
  `-coverprofile=..\a.cover.minimize.out` argument and `go test` then fails
  with `setup failed`. Both forms were tried on this tree; the space form
  writes the `mode: set` profile the gate below reads.

- Pure-model coverage gate (Steps 2 and 3, per Q08), run from the root right
  after the profile above. A profile line reads
  `<file>:<start>,<end> <statements> <count>`; the gate fails when a block of
  `minimize_windows.go`, or of `minimize_cap_windows.go` if the Step 3 split
  happened, has a zero count, or when no block of those files is present.
  `minimize_hook_windows.go` does not match the pattern and stays
  evidence-only:

  ```powershell
  $prof = Get-Content -LiteralPath a.cover.minimize.out | Select-Object -Skip 1
  $pure = $prof | Where-Object { $_ -match '/minimize_(cap_)?windows\.go:' }
  $miss = $pure | Where-Object { $_.Split(' ')[2] -eq '0' }
  if (-not $pure) { throw 'no pure minimize block in the profile' }
  if ($miss) { $miss; throw 'pure minimize model below 100% statements' }
  ```

---

## Numbered steps for v0.0.24 minimize_loop

### Step 1. Extract the minimize responsibility from main_windows

#### Step 1 -- analysis and intent for the minimize code split

Issues to address:

- `companion/main_windows.go` is 2022 lines, far over the 650-line ceiling;
  the minimize rework must not grow it in place.
- `companion/main_windows_test.go` is 661 lines, over the ceiling, and the
  new minimize tests can't go there.

Fix intent:

- Move the pure minimize code (event constants, phase and action types,
  `minimizeReplayDelayMS`, `minimizeEventTransition`, `minimizeTransition`) to
  `companion/minimize_windows.go`.
- Move the minimize Win32 glue (`minimizeWinEventCallback`,
  `installMinimizeHook`, `minimizeWinEventProc`,
  `restoreTargetForMinimizePriming`, `composeHalo`, `replayPendingMinimize`)
  to `companion/minimize_hook_windows.go`.
- Move `TestMinimizeTransitionTargetsOnlyTheTrackedTopLevelWindow`,
  `TestMinimizeEventTransitionPrimesThenAcceptsTheReplay` and
  `TestDuplicateMinimizeStartDoesNotRestartPriming` to
  `companion/minimize_windows_test.go`.
- Move code verbatim: same names, same bodies, same comments.

Expected outcome:

- The host binary behaves exactly as before; the Go suite passes with the
  same test count.
- `main_windows_test.go` is under 650 lines and `main_windows.go` has shed
  about 180 lines.

Step framing:

- Design link: design Q03 (pure logic in `companion/minimize_windows.go`,
  tested in `companion/minimize_windows_test.go`, Win32 calls with their
  callers); the Win32 callers move with the Win32 calls, as the 650-line
  policy requires for a file over the limit.
- Execution checklist reference: "Shared execution command checklist for all
  v0.0.24 minimize_loop steps", with the "Ready-to-run command templates for
  all v0.0.24 minimize_loop steps".

#### Step 1 -- implementation for the minimize code split

**Files involved**:

- `companion/main_windows.go` (existing, to be updated).
- `companion/minimize_windows.go` (new, to be created).
- `companion/minimize_hook_windows.go` (new, to be created).
- `companion/main_windows_test.go` (existing, to be updated).
- `companion/minimize_windows_test.go` (new, to be created).

**Tests first**:

- No new behavior: the three moved tests are the step's guard. Record the
  test count of `go test -v` (`--- PASS` lines) before the move; it must be
  identical after.

**Classes and behavior**:

- `companion/minimize_windows.go`: `//go:build windows`, `package main`, no
  imports; holds `eventSystemMinimizeStart`, `eventSystemMinimizeEnd`,
  `objidWindow`, `childidSelf`, `wineventOutofcontext` (moved out of the main
  `const` block), `minimizePhase` and its constants, `minimizeAction` and its
  constants, `minimizeReplayDelayMS`, `minimizeEventTransition`,
  `minimizeTransition`.
- `companion/minimize_hook_windows.go`: `//go:build windows`, `package main`,
  imports `fmt`, `syscall`, `unsafe` as needed; holds
  `var minimizeWinEventCallback = syscall.NewCallback(minimizeWinEventProc)`
  and the five moved functions with their doc comments.
- `companion/main_windows.go`: the `var` block at line 360 keeps only
  `activeApp`; imports stay as they are unless the compiler reports one
  unused.
- `companion/main_windows_test.go`: lines 274-329 removed; imports checked for
  any that became unused.

**Completion criteria**:

- Shared gate loop green: `ghog day` at `exit=9`, `scripts\test-companion.ps1`
  and `npm test` at exit 0.
- These commands print nothing:

  ```powershell
  $main = 'companion/main_windows.go'
  git grep -nE 'func (minimizeEventTransition|minimizeTransition)' -- $main
  git grep -nE 'func .*(MinimizeHook|WinEventProc|Priming)' -- $main
  git grep -nE 'func .*(composeHalo|PendingMinimize)' -- $main
  git grep -n 'minimize' -- companion/main_windows_test.go
  ```

- `gofmt -l` prints nothing, `go vet` is clean, and the `--- PASS` count is
  unchanged.

#### Step 1 -- addendums for the minimize code split

Line-budget checkpoint:

- `companion/main_windows.go`: before 2022; over-650 split-required; repository
  ceiling <= 650 not reachable in this effort; target <= 1850 (mandatory
  because extracting the minimize responsibility is this step's goal).
- `companion/main_windows_test.go`: before 661; over-650 split-required; target
  <= 650 (mandatory because bringing it under the ceiling is this step's
  goal); expected about 604 (advisory).
- `companion/minimize_windows.go`: before 0 (new); below-550 safe; ceiling
  <= 650; expected about 70 (advisory).
- `companion/minimize_hook_windows.go`: before 0 (new); below-550 safe;
  ceiling <= 650; expected about 130 (advisory).
- `companion/minimize_windows_test.go`: before 0 (new); below-550 safe;
  ceiling <= 650; expected about 65 (advisory).

Split guidance:

- If `main_windows.go` still exceeds 1850, check that the whole 833-965 range
  and the 175-211 block moved; do not move unrelated rendering or taskbar code
  in this step.
- If `main_windows_test.go` stays above 650, move
  `TestVisibilityStatePrecedence` (lines 207-241, the `minimized` trigger
  precedence table) next to the minimize tests as a second option.

Full workflow timing run readiness:

- Targeted: `go test -run 'Minimize' ./...`; then the shared gate loop.

Time-gated status for Step 1:

- No perf gate is affected (see "Step 0 perf-gate decision for v0.0.24
  minimize_loop").

---

### Step 2. Observation model, own-call absorption and session latch

#### Step 2 -- analysis and intent for the observation model

Issues to address:

- `minimizeEventTransition` decides from event kind and order: a late restore
  `MinimizeEnd` in Replaying resets to Idle, and the late replay
  `MinimizeStart` then re-intercepts, forever.
- `replayPendingMinimize` re-primes without bound when the window is iconic at
  replay time.
- Nothing checks that an own `ShowWindow` call produced the expected state.
- `dwmsEventTime` is discarded, so logs can't compare event lag with state
  observations.

Fix intent:

- Replace the old phases and `minimizeEventTransition` with a pure
  `minimizeModel` whose transitions take the `IsIconic` reading and the tick,
  following the design's target behavior pseudo-code.
- Add a `minimizeController` that owns the model, reads the window through a
  three-method seam, executes the model's action, absorbs its own calls, and
  logs every decision.
- Make every matched minimize WinEvent a plain observation trigger that logs
  its `dwmsEventTime` age.
- Wire the controller into `main`, `tick` and `application`, replacing
  `minimizeState` and `minimizeReplayAt`.

Expected outcome:

- The recorded reordered sequence ends after one interception; late,
  duplicated, lost or reordered events find no edge.
- An edge with an age bound over 500 ms is skipped with reason
  `unknown-age`; an edge during Priming cancels the replay without a restore.
- An own call whose after-reading is unexpected enters Unsettled, and its
  resolution at the 1 s deadline latches interception off for the session.

Step framing:

- Design link: "Target Behavior", "State Observation", "Minimize Phase Model",
  "Decision log lines", design Q01, Q02, Q04 and Q06.
- Execution checklist reference: "Shared execution command checklist for all
  v0.0.24 minimize_loop steps".

#### Step 2 -- implementation for the observation model

**Files involved**:

- `companion/minimize_windows.go` (existing after Step 1, to be updated).
- `companion/minimize_hook_windows.go` (existing after Step 1, to be updated).
- `companion/main_windows.go` (existing, to be updated).
- `companion/minimize_windows_test.go` (existing after Step 1, to be updated).
- `companion/minimize_hook_windows_test.go` (new, to be created).

**Tests first**:

- In `minimize_windows_test.go`, delete
  `TestMinimizeEventTransitionPrimesThenAcceptsTheReplay` and
  `TestDuplicateMinimizeStartDoesNotRestartPriming` (their function goes) and
  keep `TestMinimizeTransitionTargetsOnlyTheTrackedTopLevelWindow`.
- Add table-driven model tests, each feeding synthetic `(iconic, now)`
  observations and own-call outcomes:
  - `TestMinimizeModelSeedsPhaseFromTheFirstReading` (shown seeds Shown with
    `lastShownAt`; iconic seeds Minimized without halo and never intercepts).
  - `TestMinimizeModelInterceptsAPromptShownToIconicEdge` (shown at 1000,
    iconic at 1020: intercept, age bound 20).
  - `TestMinimizeModelSkipsAnEdgeWhoseAgeBoundExceedsTheLatenessBound`
    (shown at 1000, iconic at 4400: skip `unknown-age`, Minimized without
    halo; 500 is still prompt, 501 is not).
  - `TestMinimizeModelAbsorbsItsOwnRestoreAndReplay` (after-readings become
    `lastIconic`, so the next observations find no edge; replay after-reading
    iconic gives Minimized with halo).
  - `TestMinimizeModelRepeatedReadingsAreNotEdges` (duplicate observations,
    as produced by late or duplicated events, change nothing but
    `lastShownAt`).
  - `TestMinimizeModelReplayIsDueOnlyWhileShownInPriming` (due at
    restore tick + 75, not before, not when iconic).
  - `TestMinimizeModelEdgeDuringPrimingCancelsTheReplayWithoutRestore`
    (Minimized with halo composed, action cancel-replay, no intercept).
  - `TestMinimizeModelHonorsARestoreFromMinimized` (iconic to shown: Shown,
    action restore-honored).
  - `TestMinimizeModelUnsettledOwnCallResolvesAtTheDeadlineAndLatches`
    (replay after-reading shown: Unsettled, edges before the deadline absorbed
    or logged, never intercepted; at restore tick + 1000 phase from reading,
    latched).
  - `TestMinimizeModelLatchedSkipsEveryLaterEdge` (edges at +2.5 s, +7 s and
    +90 s after the latch: skip `latched`).
- Add `FuzzMinimizeModelObservations` with at least eight `f.Add` seeds (the
  recorded reordered sequence, a stalled thread, an Unsettled replay, a
  restore during Priming). It decodes the bytes into observations and own-call
  outcomes and checks, after every transition: phase Minimized implies the last
  reading is iconic; phases Shown and Priming imply it is shown; no intercept
  unless the phase was Shown, the edge was shown-to-iconic, the bound was at
  most 500 and the model was not latched; Unsettled never survives an
  observation past its deadline; every intercept is followed by exactly one
  own restore before the next observation.
- In the new `minimize_hook_windows_test.go`, add a `fakeMinimizeWindow`
  (scripted `iconic` state, a `showWindow` that applies or defers the change,
  counters for restores, replays and compositions) and a logger writing to a
  `bytes.Buffer`, then:
  - `TestMinimizeControllerInterceptsOnceAndReplays` (one restore, one
    replay, the log holds `minimize intercepted`, `minimize replay requested
    after halo composition` and `minimize replay accepted with composed
    halo`).
  - `TestMinimizeControllerLateEventsCauseNoRestore` (events after the
    replay only log `minimize event` lines with their age).
  - `TestMinimizeControllerUsesTheActivatingFallbackOnlyOnce` (fake stays
    iconic after `SW_SHOWNOACTIVATE`: one `SW_RESTORE`, and the `own restore`
    line says `fallback=true`; never a second restore).
  - `TestMinimizeControllerUnsettledRestoreComposesNothing` (fake stays
    iconic after both commands: no `composeHalo` call, the log holds
    `minimize intercepted: restored=false replay=none` and `own call
    unsettled: expected=shown observed=iconic`, never `replay-in=75ms`, and
    no replay follows).
  - `TestMinimizeControllerStampsOwnCallsAfterTheAfterReading` (the fake's
    `showWindow` advances the fake clock by 120 ms: `replayAt` is the
    after-reading tick + 75, the settle deadline of an unsettled call is that
    tick + 1000, and an external minimize 30 ms after the after-reading is
    intercepted with `age<=30ms`, not skipped `unknown-age`).
  - `TestMinimizeControllerDeferredReplayLatches` (fake defers the replay:
    `own call unsettled: expected=iconic observed=shown`, then `minimize
    interception disabled: own call unsettled (expected=iconic
    observed=shown)` at the deadline).

**Classes and behavior**:

- `minimizePhase` (in `minimize_windows.go`): `minimizeShown`,
  `minimizePriming`, `minimizeMinimized`, `minimizeUnsettled`; the old
  Idle, Replaying and Committed constants and `minimizeEventTransition` are
  deleted.
- Constants: `minimizeReplayDelayMS = 75` (kept),
  `minimizeLatenessBoundMS = 500`, `minimizeSettleTimeoutMS = 1000`.
- `minimizeModel` (value type): `phase`, `iconic` (last reading),
  `lastShownAt`, `replayAt`, `haloComposed`, `expectIconic`,
  `settleDeadline`, `latched`.
- `minimizeDecision` (value type): `edge` (none, shown-to-iconic,
  iconic-to-shown), `ageBound`, `action` (none, intercept, cancel-replay,
  skip, restore-honored, absorbed, settled), `reason` (`unknown-age`,
  `latched`, later `cap`), and `latchedNow` for the resolution log.
- `newMinimizeModel(iconic bool, now uint64) minimizeModel`: seeds from the
  first reading.
- `(m minimizeModel) observe(iconic bool, now uint64) (minimizeModel, minimizeDecision)`:
  resolves an expired Unsettled first (phase from reading, `latched = true`),
  then detects the edge against `m.iconic`, applies the design's transition
  rules, and records `lastShownAt` on every shown reading.
- `(m minimizeModel) ownCall(expectIconic, afterIconic bool, now uint64) minimizeModel`:
  sets `iconic` from the after-reading (absorption); expected shown and
  shown gives Priming with `replayAt = now + 75`; expected iconic and iconic
  gives Minimized with `haloComposed = true`; any mismatch gives Unsettled
  with `settleDeadline = now + 1000`.
- `(m minimizeModel) replayDue(now uint64) bool` and
  `(m minimizeModel) showsMinimizedTrigger() bool` (every phase except
  Shown).
- `minimizeWindow` interface (in `minimize_hook_windows.go`):
  `isIconic() bool`, `showWindow(cmd uintptr)`, `composeHalo() error`;
  `*application` implements it with `procIsIconic`, `procShowWindow` and the
  existing `composeHalo`.
- `minimizeController`: fields `model`, `window`, `logger`, and `clock func()
  uint64` (`getTickCount64` in production, a fake clock in tests);
  `newMinimizeController(window, logger, clock)` seeds the model from one
  `isIconic` reading at `clock()`. `observe(now uint64)` runs one
  observation, logs the edge line (`minimize edge: shown->iconic age<=20ms
  action=intercept`, `... action=skip reason=unknown-age`,
  `... action=cancel-replay`, `minimize edge: iconic->shown
  action=restore-honored`), then executes the decision:
  - intercept: own restore first. When its after-reading is shown (the model
    is now Priming), call `composeHalo` and log the kept line `minimize
    intercepted: restored=true replay-in=75ms`. When it is still iconic (the
    model is now Unsettled), compose nothing and log `minimize intercepted:
    restored=false replay=none` after the `own call unsettled` line. Both
    cases count as an interception for the Step 3 cap, because the restore
    was attempted.
  - due replay: `composeHalo`, the kept `minimize replay requested after halo
    composition` line, own replay, and `minimize replay accepted with composed
    halo` when the after-reading is iconic.
  `onEvent(starting bool, eventAgeMS uint32, now uint64)` logs `minimize
  event: start|end age=<n>ms` and then calls `observe(now)`. The legacy
  `minimize end` line and the two re-prime lines are no longer written: the
  `restore-honored` edge line and the `minimize event: end` line carry that
  information.
- Own calls (per Q04, every own call is stamped after its effective
  after-reading): `ownRestore()` reads `isIconic` before, calls
  `showWindow(swShowNoActivate)`, reads again, calls `showWindow(swRestore)`
  once if still iconic, reads the after state, takes `after := clock()`, logs
  `own restore: before=<state> after=<state> fallback=<bool>`, and passes
  `after` to `model.ownCall(false, afterIconic, after)`; `ownReplay()` does
  the same with one `showWindow(swMinimize)` and `model.ownCall(true, ...)`,
  logging `own replay: before=<state> after=<state>`. So `lastShownAt`,
  `replayAt` and the settle deadline all start at the tick of the reading
  they belong to, however long `ShowWindow` blocked. A mismatch logs `own
  call unsettled: expected=<state> observed=<state>`.
- `minimizeWinEventProc`: keeps the `minimizeTransition` filter, reads its
  seventh parameter as `dwmsEventTime`, and calls
  `app.minimize.onEvent(starting, uint32(now)-uint32(dwmsEventTime), now)`.
  `restoreTargetForMinimizePriming` and `replayPendingMinimize` are deleted.
- `application` (in `main_windows.go`): `minimizeState` and
  `minimizeReplayAt` are replaced by `minimize *minimizeController`; `main`
  creates it with `newMinimizeController(app, logger, getTickCount64)`
  before `installMinimizeHook`; `tick` calls `a.minimize.observe(now)` where
  it called `replayPendingMinimize`, and passes
  `minimized != 0 || a.minimize.model.showsMinimizedTrigger()` to
  `visibilityState`.
- `getTickCount64`: `procGetTickCount64 = kernel32.NewProc("GetTickCount64")`
  moves into the existing `kernel32` proc block, and the function calls it.

**Completion criteria**:

- Shared gate loop green.
- The first five commands print nothing (the fifth keeps the model pure),
  and the last one finds exactly one line, in the proc `var` block:

  ```powershell
  git grep --untracked -nE 'minimizeEventTransition|PendingMinimize' -- companion
  git grep --untracked -n 'ForMinimizePriming' -- companion
  git grep --untracked -nE 'minimize(Replaying|Committed|Idle)' -- companion
  git grep --untracked -nE 'minimize(State|ReplayAt)' -- companion
  git grep -nE 'procShowWindow|procIsIconic' -- companion/minimize_windows.go
  git grep -nF 'NewProc("GetTickCount64")' -- companion/main_windows.go
  ```

- The fuzz seeds and every test above pass under `scripts\test-companion.ps1`.
- The pure-model coverage gate of the command templates passes:
  `minimize_windows.go` has no uncovered statement block.

#### Step 2 -- addendums for the observation model

Line-budget checkpoint:

- `companion/main_windows.go`: before about 1844 (after Step 1); over-650
  split-required; target <= its post-Step-1 count (mandatory because the file
  is over the limit and must not grow in place); expected net -1 (advisory).
- `companion/minimize_windows.go`: before about 70; below-550 safe; ceiling
  <= 650; expected about 250 (advisory).
- `companion/minimize_hook_windows.go`: before about 130; below-550 safe;
  ceiling <= 650; expected about 230 (advisory).
- `companion/minimize_windows_test.go`: before about 65; below-550 safe;
  ceiling <= 650; expected about 330 (advisory).
- `companion/minimize_hook_windows_test.go`: before 0 (new); below-550 safe;
  ceiling <= 650; expected about 230 (advisory).

Split guidance:

- If `main_windows.go` would grow, move the three tick lines' logic into one
  `minimizeController` method call rather than adding code in `tick`.
- If `minimize_windows_test.go` passes 550, move
  `FuzzMinimizeModelObservations` and its decoder into
  `companion/minimize_fuzz_windows_test.go` (new).

Full workflow timing run readiness:

- Targeted: `go test -run 'Minimize' ./...`; then the shared gate loop.

Time-gated status for Step 2:

- No perf gate is affected; the lateness, replay and settle timings are
  covered by synthetic ticks in the tests above.

---

### Step 3. Per-window interception cap with quiet-period reset

#### Step 3 -- analysis and intent for the interception cap

Issues to address:

- No cap bounds interceptions per window if an unexpected sequence still
  repeats them.
- A time-windowed count alone would re-arm during a continuing stream of
  minimizes.

Fix intent:

- Add a pure `minimizeCap` to the model: it records interception ticks, trips
  when an edge would be the third interception within 2000 ms, and resumes
  only after 5000 ms with no external shown-to-iconic edge at all.
- Log the trip and the resume.

Expected outcome:

- Minimize, restore, minimize within 2 s gives two interceptions.
- A third external minimize edge within 2 s is skipped with reason `cap`.
- Edges every second for 20 s keep the cap closed; it rearms 5 s after the
  last edge.

Step framing:

- Design link: "Interception Cap", design Q05, and the cap rows of the
  acceptance cases.
- Execution checklist reference: "Shared execution command checklist for all
  v0.0.24 minimize_loop steps".

#### Step 3 -- implementation for the interception cap

**Files involved**:

- `companion/minimize_windows.go` (existing, to be updated).
- `companion/minimize_hook_windows.go` (existing, to be updated).
- `companion/minimize_windows_test.go` (existing, to be updated).
- `companion/minimize_hook_windows_test.go` (existing, to be updated).

**Tests first**:

- `TestMinimizeCapAllowsTwoInterceptionsWithinTheWindow`.
- `TestMinimizeCapTripsOnTheThirdEdgeWithinTwoSeconds` (skip `cap`,
  `suspended` set, trip reported once).
- `TestMinimizeCapStaysClosedDuringAContinuingStream` (after the trip,
  external edges every 1000 ms for 20 s, alternating prompt and
  `unknown-age` ones, then a latched model fed the same stream: for each
  edge, assert `lastAttemptAt` equals that edge's tick, the cap is still
  suspended, and the reported reason follows the precedence below, so prompt
  edges report `cap` and old ones `unknown-age`; no `capResumed` is reported
  anywhere in the stream).
- `TestMinimizeCapResumesAfterFiveSecondsWithoutAnyEdge` (with the last edge
  at tick T: an observation without edge at T + 4999 keeps the cap
  suspended; one at T + 5000 resumes it and clears the history; a prompt edge
  at T + 4999 instead is skipped `cap` and moves the quiet timer to
  T + 4999, so resume waits until T + 9999; a prompt edge at T + 5000 is
  intercepted after the resume in the same observation).
- `TestMinimizeCapIgnoresPrimingCancellations` (an edge during Priming is not
  counted as an interception).
- Extend `FuzzMinimizeModelObservations` with the properties: no three
  intercepts within any 2000 ms span; no intercept while suspended; while
  suspended, `lastAttemptAt` equals the tick of the latest external
  shown-to-iconic edge; and a resume never happens less than 5000 ms after
  that edge.
- `TestMinimizeControllerLogsCapTripAndResume` (log holds `minimize
  interception suspended: 2 intercepts in 2000ms` and `minimize interception
  resumed after 5000ms quiet`).

**Classes and behavior**:

- Constants: `minimizeCapCount = 2`, `minimizeCapWindowMS = 2000`,
  `minimizeCapQuietMS = 5000`.
- `minimizeCap` (value type, field `cap` of `minimizeModel`): `intercepts
  [minimizeCapCount]uint64` ring with its fill count, `suspended`,
  `lastAttemptAt`.
- `(c minimizeCap) closed(now uint64) bool`: suspended, or `minimizeCapCount`
  recorded intercepts all within the last `minimizeCapWindowMS`.
- `(c minimizeCap) recordIntercept(now uint64) minimizeCap`,
  `(c minimizeCap) recordAttempt(now uint64) (minimizeCap, bool tripped)`,
  `(c minimizeCap) maybeResume(now uint64) (minimizeCap, bool resumed)`.
- `minimizeModel.observe` runs the cap in this fixed order within one
  observation:
  1. Resolve an expired Unsettled (Step 2), then call `maybeResume(now)`:
     a suspended cap resumes, and clears its intercept history, when
     `now - lastAttemptAt >= minimizeCapQuietMS`.
  2. Detect the edge. For every external shown-to-iconic edge, whatever
     reason it will be skipped for, call `recordAttempt(now)` while the cap
     is suspended, before any skip returns, so an `unknown-age`, `latched`
     or `cap` edge all move the quiet timer.
  3. Choose the decision with this precedence, which follows the design's
     target-behavior order: `unknown-age` when the bound exceeds 500 ms;
     else `latched`; else `cap` when `closed(now)` (suspended, or
     `minimizeCapCount` intercepts within `minimizeCapWindowMS`); a
     `closed` that is not yet suspended trips the cap here, sets
     `suspended`, `lastAttemptAt = now` and `capTripped`; else intercept.
     The design's "every edge skipped with reason `cap`" while suspended
     holds for every edge that has no earlier reason; no edge is ever
     intercepted while suspended.
  4. On intercept, `recordIntercept(now)`. The tick is recorded at the
     decision, so an intercept whose own restore ends Unsettled (Q05) still
     counts.
  An edge during Priming (cancel-replay) is recorded as an attempt when
  suspended but never as an intercept. `minimizeDecision` gains `capTripped`
  and `capResumed`.
- `minimizeController.observe`: logs the two cap lines from those flags.

**Completion criteria**:

- Shared gate loop green.
- The first command finds the three constants and the second finds both log
  formats:

  ```powershell
  git grep -nE 'minimizeCap(Count|WindowMS|QuietMS) +=' -- companion
  git grep -nE 'interception (suspended|resumed)' -- companion
  ```

- The pure-model coverage gate of the command templates passes, including
  `minimize_cap_windows.go` when the split below was applied.

#### Step 3 -- addendums for the interception cap

Line-budget checkpoint:

- `companion/minimize_windows.go`: before about 250; below-550 safe; ceiling
  <= 650; expected about 330 (advisory).
- `companion/minimize_hook_windows.go`: before about 230; below-550 safe;
  ceiling <= 650; expected about 240 (advisory).
- `companion/minimize_windows_test.go`: before about 330; below-550 safe;
  ceiling <= 650; expected about 460 (advisory).
- `companion/minimize_hook_windows_test.go`: before about 230; below-550 safe;
  ceiling <= 650; expected about 260 (advisory).

Split guidance:

- If `minimize_windows.go` passes 550, move `minimizeCap` and its constants to
  `companion/minimize_cap_windows.go` (new).
- If `minimize_windows_test.go` passes 550, apply the Step 2 fuzz split, then
  move the cap tests to `companion/minimize_cap_windows_test.go` (new).

Full workflow timing run readiness:

- Targeted: `go test -run 'MinimizeCap|FuzzMinimize|MinimizeController' ./...`;
  then the shared gate loop.

Time-gated status for Step 3:

- No perf gate is affected; the cap windows are covered by synthetic ticks.

---

### Step 4. Acceptance scenarios, documentation and the unplug check

#### Step 4 -- analysis and intent for acceptance and documentation

Issues to address:

- The unit tests check the model and controller piece by piece; the issue's
  required behavior 6 asks for the recorded in-order and reordered sequences,
  arbitrarily late replay events, user actions around the replay, a missing
  event, late and unknown-age minimizes, ambiguous origin, and cap trip and
  reset, plus one manual unplug.
- The wiki still describes the event-driven interception and lists only the
  75 ms replay delay.

Fix intent:

- Add controller-level acceptance scenarios, one per design acceptance case,
  plus a four-window replay of the 2026-09-24 unplug timeline, all through
  `minimizeController` with scripted fake windows and delayed events.
- Update the two wiki pages, the log reference and the changelog.
- Build the VSIX with `build.bat`, install it, and run the manual
  three-to-one unplug with four VS Code windows.

Expected outcome:

- Every acceptance case of the design has a named passing scenario.
- The manual unplug log shows at most one interception attempt (one
  `minimize intercepted` line, one `own restore` line) per external minimize
  action and window, no repeated cycle, and no delayed return to the
  foreground; the lines are kept as evidence in the validation plan. That
  one attempt may issue two `ShowWindow` commands, `SW_SHOWNOACTIVATE` then a
  single `SW_RESTORE` fallback (`fallback=true`); that is still one
  interception.

Step framing:

- Design link: "Acceptance Cases", "Documentation updates"; issue required
  behavior 6 and the Q05 decision.
- Execution checklist reference: "Shared execution command checklist for all
  v0.0.24 minimize_loop steps".

#### Step 4 -- implementation for acceptance and documentation

**Files involved**:

- `companion/minimize_hook_windows_test.go` (existing, to be updated).
- `wiki/reference/display-triggers.md` (existing, to be updated).
- `wiki/explanation/how-the-overlay-stays-inside-its-window.md` (existing, to
  be updated).
- `wiki/reference/logs-and-processes.md` (existing, to be updated).
- `CHANGELOG.md` (existing, to be updated).

**Tests first**:

- `TestMinimizeAcceptanceCases`, table-driven, one sub-test per design
  acceptance row (normal path; events in any order or 3 s late; replay
  `MinimizeStart` after a restore; lost restore `MinimizeEnd`; `MinimizeStart`
  generated before priming and delivered during Priming; re-minimize 100 ms
  after the priming restore; user minimize within 250 ms of it; replay
  after-reading shown; that replay applying at 2.5 s and at 7 s; an unrelated
  minimize 90 s after the latch; a new controller after a latch; a 3.4 s
  `lastShownAt`; minimize, restore, minimize within 2 s; a third edge within
  2 s; edges every second for 20 s). Each scenario is a script of ticks,
  external window actions and event deliveries, and asserts the restore
  count, the final phase and the decisive log line.
- `TestMinimizeAcceptanceClickDuringPrimingStillReplays` (a click changes
  nothing observable; the replay minimizes the window) and
  `TestMinimizeAcceptanceRestoreAfterReplayStaysRestored`.
- `TestMinimizeAcceptanceRecordedUnplugTimeline`: four controllers with fake
  windows; Windows minimizes w4, w1 and w2 at the recorded offsets (0, 2.7 s
  and 4.0 s) while ticks run every 25 ms; every minimize WinEvent, including
  those caused by own calls, is delivered 3 to 6 s late and reordered. Assert
  at most one `ownRestore` attempt per window and minimize action (that one
  attempt may issue the single `SW_RESTORE` fallback, as in the manual
  check), no restore attempt after the late deliveries, every window ends
  Minimized, and no log holds a second `minimize intercepted` for the same
  action.

**Classes and behavior**:

- A small scenario runner in the test file: a list of `{at, kind, value}`
  steps (tick, external minimize, external restore, deliver event with age,
  defer own call) applied in tick order to one or more controllers.
- `wiki/reference/display-triggers.md`: the timing constants table gains
  lateness bound 500 ms, settle timeout 1000 ms, cap 2 interceptions in
  2000 ms, cap quiet period 5000 ms; the `minimized` row condition mentions a
  pending interception or an unsettled own call; a short section describes
  when an observed minimize is intercepted and the session latch.
- `wiki/explanation/how-the-overlay-stays-inside-its-window.md`: the
  replay section explains that interception now starts from an observed
  minimize the host did not cause, that WinEvents only trigger an observation,
  and why unknown-age, uncertain own call, capped and latched minimizes go
  through without the halo.
- `wiki/reference/logs-and-processes.md`: lists the `minimize edge`,
  `minimize event`, `own restore`, `own replay`, `own call unsettled`,
  `minimize interception disabled`, `suspended` and `resumed` lines next to
  the kept interception lines.
- `CHANGELOG.md`: a `## 0.0.24` section with the fix (no more minimize loop
  after a monitor unplug, thumbnail halo kept for prompt minimizes, skipped
  cases logged).

**Rollout sequence for v0.0.24 minimize_loop**:

1. Commit Steps 1 to 4, each with the shared gate loop green.
2. Run `build.bat` (Go and TypeScript test gates, then the commit-named
   VSIX).
3. Install the VSIX with `install.bat`, open four workspace windows spread
   over three monitors, and reload them so each runs the new host.
4. Unplug the video cable (three monitors to the laptop screen), wait 60 s,
   plug it back.
5. Copy the minimize lines of each window's `native-host.log` (path printed in
   the Workspace Halo output channel) into the Step 4 section of the
   validation plan, in one `text` fence per window headed only by its window
   label (w1 to w4, from the log directory `window<N>`; no workspace name, so
   no local detail is committed), keeping each line's timestamp. Keep only
   `minimize edge`, `minimize event`,
   `own restore`, `own replay`, `own call unsettled`, `minimize intercepted`,
   `minimize replay`, `minimize interception` and `visibility=minimized`
   lines, from 10 s before the unplug to 60 s after it.
6. Judge the evidence per external minimize action in a table with one row
   per `minimize edge: shown->iconic` line: window, edge timestamp, age bound,
   decision (`intercept`, `skip` with its reason, or `cancel-replay`), the
   number of `minimize intercepted` and `own restore` lines up to the next
   external edge of the same window (at most one each; `fallback=true`
   allowed), `own restore` lines after the replay was accepted (must be zero,
   so no delayed return to the foreground), and the verdict.

**Completion criteria**:

- Shared gate loop green, then `build.bat` exits 0 and prints `OK: Packaged`.
- The first command finds the new constants rows and the second finds the
  changelog section:

  ```powershell
  $triggers = 'wiki/reference/display-triggers.md'
  git grep -nE 'Lateness bound|Settle timeout|Interception cap' -- $triggers
  git grep -nE '^## 0\.0\.24' -- CHANGELOG.md
  ```

- `cmd /c "switchnode 22 && npx --yes markdownlint-cli2 wiki/**/*.md CHANGELOG.md"`
  reports no finding on the four updated files except MD013 on table rows,
  which the existing wiki tables already carry (the repository has no
  `.markdownlint.json`, so MD013 applies at its 80-column default).
- Every row of the rollout step 6 table passes: at most one interception
  attempt per external minimize action, no `own restore` after an accepted
  replay, and no `minimize intercepted` line without a matching external
  edge; together this meets the issue's required behavior 6.

#### Step 4 -- addendums for acceptance and documentation

Line-budget checkpoint:

- `companion/minimize_hook_windows_test.go`: before about 260; below-550 safe;
  ceiling <= 650; expected about 480 (advisory).
- `wiki/reference/display-triggers.md`: before 52; below-550 safe; expected
  about 77 (advisory).
- `wiki/explanation/how-the-overlay-stays-inside-its-window.md`: before 45;
  below-550 safe; expected about 75 (advisory).
- `wiki/reference/logs-and-processes.md`: before 46; below-550 safe; expected
  about 54 (advisory).
- `CHANGELOG.md`: before 211; below-550 safe; expected about 219 (advisory).

Split guidance:

- If `minimize_hook_windows_test.go` passes 550, move the scenario runner and
  the acceptance tests to `companion/minimize_acceptance_windows_test.go`
  (new), leaving the controller unit tests in place.

Full workflow timing run readiness:

- Targeted: `go test -run 'MinimizeAcceptance' ./...`; then the shared gate
  loop; then `build.bat`.

Time-gated status for Step 4:

- No perf gate is affected; the manual unplug is judged on log evidence, not
  on timing.

---

## Implementation decisions for v0.0.24 minimize_loop

| Question | Decision | Integrated in | Rejected alternatives |
| --- | --- | --- | --- |
| Q01 | Two new files: the pure model in `companion/minimize_windows.go` and the Win32 glue (hook, callback, controller, window seam) in `companion/minimize_hook_windows.go`, so the pure file can be proven Win32-free and `main_windows.go` shrinks and never grows | Confirmed technical facts; Step 1 | Keep the glue in `main_windows.go` (growth in place over the 650-line ceiling); one file for model and glue (purity no longer checkable per file, heads for 550 lines) |
| Q02 | Step 1 is a standalone verbatim move with its own commit and an unchanged `--- PASS` count; Step 2 then rewrites in the small files | Step 1 | Fold the move into Step 2 (one diff mixes relocation and behavior, harder to review and bisect) |
| Q03 | The controller reaches the window through a three-method `minimizeWindow` interface (`isIconic`, `showWindow`, `composeHalo`) implemented by `*application` and by a scripted test fake | Step 2 classes and tests; Step 4 scenarios | Three function fields (partial wiring compiles); no seam (glue untested, no timeline replay) |
| Q04 | Each own call reads `clock()` after its effective after-reading and passes that tick to `model.ownCall`, so `lastShownAt`, `replayAt` and the settle deadline follow the reading; a fake-clock test covers a 120 ms `ShowWindow` | Step 2 own calls and tests | Reuse the observation's `now` (false `unknown-age`, early replay); pass both ticks to the model (wider pure API for no decision) |
| Q05 | An intercept composes the halo and logs `restored=true replay-in=75ms` only when the restore's after-reading is shown; otherwise it composes nothing, logs `restored=false replay=none` after `own call unsettled`, and still counts for the cap | Step 2 controller and tests; Step 3 cap order | Always compose (renders into an iconic window, hides the failure); don't count an unsettled attempt (the cap exists to bound attempted restores) |
| Q06 | The property test is the native fuzz target `FuzzMinimizeModelObservations` with at least eight `f.Add` seeds run by `go test`, checking phase, intercept, Unsettled and cap invariants | Step 2 and Step 3 tests | `testing/quick` (non-deterministic gate); table tests only (no interleaving coverage) |
| Q07 | Acceptance scenarios live in `companion/minimize_hook_windows_test.go` beside the controller tests, split to `minimize_acceptance_windows_test.go` only past 550 lines | Step 4 files and split guidance | A dedicated acceptance file from the start (one more file, fake shared across files) |
| Q08 | 100% statement coverage of `minimize_windows.go`, and of `minimize_cap_windows.go` after a split, is a Step 2 and Step 3 completion criterion, enforced by the pure-model coverage gate script over the profile; the hook file stays evidence-only | Ready-to-run command templates; Step 2 and Step 3 completion criteria | Evidence only (an untested model branch passes); 100% of both files (the Win32 seam implementation is unreachable in unit tests) |
| Q09 | Per-step gate loop: `ghog day` (expected `exit=9`), then `scripts\test-companion.ps1`, then `npm test`; `build.bat` once in Step 4 | Ready-to-run command templates; every step's completion criteria | Go suite only per step (departs from the groundhog-first rule); `build.bat` every step (a VSIX per step for no signal) |
| Q10 | The legacy `minimize end` line and the re-prime lines are dropped; the `restore-honored` edge line and the `minimize event: end` line carry the facts; the three kept interception lines stay | Step 2 controller; Step 4 log reference | Keep writing `minimize end` (two lines for one fact, with a changed meaning) |
| Q11 | Manual unplug evidence is committed in the Step 4 section of the validation plan as timestamped per-window excerpts labeled w1 to w4 only, judged by a per-action table (one row per external minimize edge) | Step 4 rollout sequence and completion criteria | Full logs committed (large, local details); an ignored `a.*` file only (evidence lost) |
