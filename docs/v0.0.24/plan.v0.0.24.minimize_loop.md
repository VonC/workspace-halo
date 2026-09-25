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
  event: start|end age=<n>ms` and then calls `observe(now)`.
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

## Open questions for the v0.0.24 minimize_loop implementation plan

### Q01: Where the minimize Win32 glue lives after the split

Question description: design Q03 puts the pure observation, phase and cap
logic in `companion/minimize_windows.go` and keeps the Win32 calls with their
callers. Today those callers (`installMinimizeHook`, `minimizeWinEventProc`,
the own restore and replay, `composeHalo`) sit in `companion/main_windows.go`,
which is 2022 lines, far over the 650-line ceiling that forbids growth in
place. The plan moves them, with their Win32 calls, to a second new file,
`companion/minimize_hook_windows.go`. Which file layout should Step 1 create?

#### BBQ for Q01

A workshop keeps its one overloaded toolbox and wants to add a new set of
drill bits. It can put the bits in a new case and leave the drill in the old
box, move the drill and its bits into one new case, or split them into a case
for the bits and a case for the drill with its cord. In this picture: the
overloaded toolbox is `main_windows.go`, the drill bits are the pure model,
the drill with its cord is the Win32 glue (hook, callback, own calls), and the
two cases are `minimize_windows.go` and `minimize_hook_windows.go`.

#### Options for Q01

- Option A: two new files, the pure model in `minimize_windows.go` and the
  glue in `minimize_hook_windows.go`.
  - pro: `main_windows.go` shrinks by about 180 lines and never grows again.
  - pro: the pure file has no Win32 reference, which a grep check can prove.
  - con: one more file to open when following a minimize from event to
    decision.
- Option B: only `minimize_windows.go` is new; the glue stays in
  `main_windows.go` and is rewritten there.
  - pro: follows the design's "Win32 calls stay with their callers" wording
    most literally.
  - con: grows or rewrites code in place in a file over the ceiling, which
    the plan policy forbids.
- Option C: one new file holding both the pure model and the glue.
  - pro: a single minimize file.
  - con: mixes the pure model with Win32 calls, so the purity check of the
    model can no longer be a file-level grep, and the file heads for 550 lines
    once the cap lands.

#### Recommended option for Q01 (with arguments for this choice)

Option A: it keeps the design's split (pure logic apart, Win32 calls with
their callers, the callers moving with them) and honors the line policy
without touching the rest of `main_windows.go`. Both files stay well under
550 lines through Step 4.

#### Answer to Q01: option A (with reason why it must be accepted as the answer)

Option A: it is the only layout that both keeps the model provably pure and
stops the over-limit file from growing.

### Q02: A separate verbatim move step, or fold the move into the rewrite

Question description: Step 1 moves the minimize code and its three tests
verbatim, then Step 2 deletes much of what was moved
(`minimizeEventTransition`, `replayPendingMinimize`, two tests) and rewrites
the rest. The move could instead happen inside Step 2, moving only the code
that survives. Should Step 1 stay a standalone behavior-preserving move?

#### BBQ for Q02

Movers can carry the old furniture to the new flat first and sort it there,
or sort in the old flat and carry only what stays. In this picture: the old
flat is `main_windows.go` and `main_windows_test.go`, the new flat is the two
new minimize files and their test file, the furniture thrown away is
`minimizeEventTransition`, `replayPendingMinimize` and their two tests, and
sorting is the Step 2 rewrite.

#### Options for Q02

- Option A: keep Step 1 as a verbatim move with its own commit, then rewrite
  in Step 2.
  - pro: the move diff is reviewable as "no behavior change", and the rewrite
    diff then shows only real changes in small files.
  - pro: `main_windows_test.go` drops under 650 lines before any new test is
    written.
  - con: about 60 moved lines are deleted one step later.
- Option B: fold the move into Step 2 and move only the surviving code.
  - pro: no line is moved and then deleted.
  - con: one large diff mixes relocation and new behavior, which is harder to
    review and to bisect.

#### Recommended option for Q02 (with arguments for this choice)

Option A: a pure move is cheap to verify (same test count, same behavior) and
makes the Step 2 diff read as the behavior change it is. The throwaway is
small.

#### Answer to Q02: option A (with reason why it must be accepted as the answer)

Option A: it separates relocation from behavior, so each step's review and
any later bisect points at one kind of change.

### Q03: How the controller reaches the window in tests

Question description: the controller must read `IsIconic`, call `ShowWindow`
and compose the halo, and the Step 2 and Step 4 tests must drive it without a
real window. The plan uses a three-method `minimizeWindow` interface
(`isIconic`, `showWindow`, `composeHalo`) implemented by `*application`, with
a scripted fake in tests. Which seam should the controller use?

#### BBQ for Q03

A flight simulator can plug the real cockpit controls into a socket shared
with the training rig, hard-wire them, or let each instrument take a separate
cable. In this picture: the socket is the `minimizeWindow` interface, the
real controls are `*application` with `procIsIconic` and `procShowWindow`,
the training rig is the scripted fake window, and the separate cables are
per-call function fields.

#### Options for Q03

- Option A: a `minimizeWindow` interface implemented by `*application` and by
  a test fake.
  - pro: one small, named contract; the fake can script deferred own calls
    for the Unsettled cases.
  - con: an interface with a single production implementation.
- Option B: three function fields on the controller (`isIconic func() bool`
  and so on).
  - pro: no interface type.
  - con: three fields to wire in `main` and in every test; a partially wired
    controller compiles.
- Option C: no seam; test only the pure model, and rely on the manual unplug
  for the glue.
  - pro: least code.
  - con: the own-call absorption, the `SW_RESTORE` fallback and the log lines
    go untested, and the recorded timeline replay of Step 4 becomes
    impossible.

#### Recommended option for Q03 (with arguments for this choice)

Option A: the glue holds the riskiest logic (own-call absorption, fallback,
Unsettled), and an interface lets the acceptance scenarios run four
controllers against scripted windows. It is idiomatic Go and costs a few
lines.

#### Answer to Q03: option A (with reason why it must be accepted as the answer)

Option A: it makes the controller testable end to end with one explicit
contract, which the Step 4 acceptance scenarios need.

### Q04: Which tick an own call records

Question description: `ShowWindow` on another thread's window returns only
once that thread applied the change, so it can take noticeable time.
`model.ownCall` sets `lastShownAt`, `replayAt` (+75 ms) and the settle
deadline (+1000 ms) from the tick it receives. Passing it the observation's
`now`, read before the call, would leave `lastShownAt` too old after a slow
call: the next prompt edge could be misjudged `unknown-age`, and the replay
would come early. Step 2 now passes the tick read after the effective
after-reading, as option A proposes. Which tick should an own call record?

#### BBQ for Q04

A courier stamps a parcel's receipt time. He can reuse the time he read on
the clock before queuing at the counter, or read the clock again once the
clerk hands the parcel back. In this picture: the queue is the blocking
cross-thread `ShowWindow`, the stamp is the tick given to `model.ownCall`,
and the receipt time drives `lastShownAt`, the replay time and the settle
deadline.

#### Options for Q04

- Option A: read `getTickCount64()` after the after-reading of each own call
  and pass that tick to `model.ownCall`.
  - pro: `lastShownAt`, `replayAt` and the deadline start when the state was
    actually observed, so a slow call never inflates an age bound.
  - con: one extra `GetTickCount64` call per own call (negligible).
- Option B: reuse the observation's `now`.
  - pro: one tick value per observation, simpler tests.
  - con: a slow `ShowWindow` makes the next edge look older than it is and
    shortens the 75 ms replay delay.
- Option C: let the caller pass both ticks (before and after) and let the
  model choose.
  - pro: full information in the model.
  - con: widens the pure API for no decision that needs the before tick.

#### Recommended option for Q04 (with arguments for this choice)

Option A: the design defines `lastShownAt` as the last observation that found
the window shown, and the after-reading is that observation. The tests stay
simple because the fake clock is under test control.

#### Answer to Q04: option A (with reason why it must be accepted as the answer)

Option A: it keeps every timestamp aligned with the reading it belongs to,
so a slow own call can't cause a false `unknown-age` skip.

### Q05: What an intercept does when the own restore doesn't settle

Question description: on an intercept, the controller runs the own restore.
If the restore's after-reading is still iconic, the model enters Unsettled
and no replay will follow. The alternatives being decided are whether the
halo is still composed, what the kept `minimize intercepted` line reports,
and whether the attempt counts for the cap. Step 2 now follows option A:
compose only on a shown after-reading, log `restored=false replay=none`
otherwise, and count the attempt. How should the controller finish an
intercept whose restore didn't settle?

#### BBQ for Q05

A photographer asks a sitter to open the curtains before a portrait. If the
curtains stay shut, he can still set up the lights, pack up at once, or note
the attempt in the session log either way. In this picture: opening the
curtains is the own restore, the lights are `composeHalo`, packing up is
entering Unsettled without a replay, and the session log is the kept
`minimize intercepted` line plus the cap's interception count.

#### Options for Q05

- Option A: compose the halo only when the after-reading is shown; always log
  `minimize intercepted` with the after state; count the attempt for the cap.
  - pro: no rendering work for a window that stays minimized, and the log
    line shows at a glance that the restore didn't settle.
  - pro: the cap still sees every restore the host attempted.
  - con: one more branch in the intercept path.
- Option B: always compose, as today.
  - pro: identical code path in every case.
  - con: renders a halo into an iconic window, where it is not captured, and
    hides the failure in the kept log line.
- Option C: don't count an unsettled intercept for the cap.
  - pro: the cap counts only interceptions that worked.
  - con: an attempted restore is exactly what the cap exists to bound.

#### Recommended option for Q05 (with arguments for this choice)

Option A: it keeps the cap an honest count of restores attempted and avoids
pointless rendering, and the latch already takes over from there.

#### Answer to Q05: option A (with reason why it must be accepted as the answer)

Option A: it makes the unsettled intercept visible in the log and counted by
the cap, with no wasted composition.

### Q06: The form of the property test

Question description: the write-plans convention asks whether a
property-based test is needed. Go has no pytest-hypothesis; its options are
native fuzz targets with a seed corpus (seeds run under plain `go test`),
`testing/quick`, or table tests only. The plan chooses
`FuzzMinimizeModelObservations` with at least eight seeds. Which form should
the property test take?

#### BBQ for Q06

A lock maker tests a new lock with a fixed ring of known tricky keys that
every inspector reruns, with a machine that cuts random keys each time, or
with only the keys from the manual. In this picture: the fixed ring is the
fuzz seed corpus, the random cutter is `testing/quick`, the manual's keys are
the table tests, and the lock's promises are the model invariants (phase
matches reading, no intercept unless prompt and unlatched, Unsettled never
outlives its deadline).

#### Options for Q06

- Option A: a native fuzz target with a seed corpus, run as seeds by the gate;
  a developer can run `go test -fuzz` locally to explore.
  - pro: deterministic in the gate, and exploration finds new seeds that can
    be committed as regressions.
  - con: the byte decoder needs a little care.
- Option B: `testing/quick` with random sequences.
  - pro: random exploration on every run.
  - con: non-deterministic gate, and a failure is hard to replay.
- Option C: table tests only.
  - pro: simplest.
  - con: no invariant check across arbitrary interleavings, which is where
    the loop came from.

#### Recommended option for Q06 (with arguments for this choice)

Option A: the bug was an interleaving nobody tabled. Invariants checked over
seeded and explorable sequences guard that class, and the gate stays
deterministic.

#### Answer to Q06: option A (with reason why it must be accepted as the answer)

Option A: it gives property coverage of arbitrary observation orders with a
deterministic, replayable gate.

### Q07: Where the acceptance scenarios live

Question description: Step 4 adds `TestMinimizeAcceptanceCases`, the
click-during-priming and restore-after-replay scenarios, the four-window
recorded unplug replay and a scenario runner. The plan puts them in
`companion/minimize_hook_windows_test.go` (expected about 430 lines), with a
split to `minimize_acceptance_windows_test.go` only past 550. Where should
they go?

#### BBQ for Q07

A workshop keeps its road-test logs in the same binder as its bench-test
sheets until the binder gets thick, or opens a road-test binder from the
start. In this picture: the bench-test sheets are the controller unit tests,
the road-test logs are the acceptance scenarios, the first binder is
`minimize_hook_windows_test.go`, and the second is
`minimize_acceptance_windows_test.go`.

#### Options for Q07

- Option A: in `minimize_hook_windows_test.go`, splitting only past 550 lines.
  - pro: one fewer file, and the fake window and scenario runner are shared
    with the controller tests without cross-file helpers.
  - con: the file mixes unit and acceptance levels.
- Option B: a dedicated `minimize_acceptance_windows_test.go` from the start.
  - pro: acceptance level clearly separated.
  - con: one more file, and the fake window must be shared across two test
    files of the package.

#### Recommended option for Q07 (with arguments for this choice)

Option A: the projected size stays under the risk band, the helpers are
shared naturally, and the plan already names the split if it grows.

#### Answer to Q07: option A (with reason why it must be accepted as the answer)

Option A: it fits the plan's aim of fewer files, with a defined split
trigger.

### Q08: Coverage criterion for the new code

Question description: the repository has no coverage gate, and groundhog
can't measure Go. The plan collects `go tool cover -func` output as evidence
and, per option A, also runs a pure-model coverage gate over the profile as
a Step 2 and Step 3 completion criterion. The validation plan has a "Unit
test coverage check" per step. Should a coverage level be a completion
criterion?

#### BBQ for Q08

A bridge inspector can require that every bolt of the new span be checked,
check every bolt of the whole bridge, or just note which bolts were checked.
In this picture: the new span is `minimize_windows.go` (the pure model), the
whole bridge is the `companion` package, and noting is coverage as evidence
only.

#### Options for Q08

- Option A: 100% statement coverage of `minimize_windows.go` (and of
  `minimize_cap_windows.go` if the Step 3 split happens) as a Step 2 and
  Step 3 completion criterion, enforced by the plan's "pure-model coverage
  gate" script over the coverage profile; the controller file reported as
  evidence only.
  - pro: the pure model is fully testable, so full coverage is cheap and
    catches untested branches of the decision logic.
  - pro: the gate is a repeatable script that fails on any uncovered block,
    not a manual reading of `go tool cover` output.
  - con: an extra criterion to check per step.
- Option B: evidence only, as the plan stands.
  - pro: no new rule for this repository.
  - con: an untested model branch can pass the gate unnoticed.
- Option C: 100% of both new production files.
  - pro: strongest guarantee.
  - con: the `*application` implementation of the seam and the hook
    installation need a real window, so 100% is not reachable in unit tests.

#### Recommended option for Q08 (with arguments for this choice)

Option A: the model holds every decision, is pure, and is cheap to cover
fully; the glue is exercised through the fake and reported without an
unreachable target.

#### Answer to Q08: option A (with reason why it must be accepted as the answer)

Option A: it sets a reachable, meaningful bar exactly where the decisions
are made.

### Q09: The per-step gate loop

Question description: the plan's shared gate loop is `ghog day` (expected
`exit=9`, since this is not a pytest project), then
`scripts\test-companion.ps1`, then `npm test`. No TypeScript changes in this
effort, and `build.bat` runs both test gates and packages the VSIX. What
should the per-step gate be?

#### BBQ for Q09

A kitchen checks each dish before it leaves: the head chef's quick look that
here only confirms there is nothing for him, the sauce tasting, the dessert
tasting, or the full service rehearsal. In this picture: the quick look is
`ghog day` at `exit=9`, the sauce is the Go suite, the dessert is `npm test`
(untouched this time), and the rehearsal is `build.bat`.

#### Options for Q09

- Option A: `ghog day`, then `scripts\test-companion.ps1`, then `npm test`,
  every step; `build.bat` once in Step 4.
  - pro: follows the workflow's groundhog rule and the project's own gate;
    `npm test` proves the TypeScript side stays green at a few seconds' cost.
  - con: `ghog day` contributes nothing but its `exit=9` verdict here.
- Option B: `scripts\test-companion.ps1` only per step; `npm test` and
  `build.bat` in Step 4.
  - pro: fastest loop.
  - con: departs from the workflow's groundhog-first rule.
- Option C: `build.bat` every step.
  - pro: the exact release gate each time.
  - con: packages a VSIX per step for no extra signal.

#### Recommended option for Q09 (with arguments for this choice)

Option A: it keeps the shared workflow contract (groundhog first, then the
project's own tests, as groundhog's exit 9 asks) at negligible cost, and
leaves packaging to the step that needs the VSIX.

#### Answer to Q09: option A (with reason why it must be accepted as the answer)

Option A: it satisfies the workflow and the project gate with the least
wasted work.

### Q10: The legacy `minimize end` and re-prime log lines

Question description: the design keeps the existing interception lines so
older logs remain comparable. The plan keeps `minimize intercepted`,
`minimize replay requested after halo composition` and `minimize replay
accepted with composed halo`. It replaces `minimize end` (logged on a
`MinimizeEnd` event) with `minimize edge: iconic->shown
action=restore-honored`, and the re-prime lines disappear with the re-prime.
Should the host still write `minimize end`?

#### BBQ for Q10

A station keeps its departure announcements the same so regular travelers
recognize them. One old announcement, "train arrived", was read out whenever
a signal box sent a message, and the signal boxes are no longer used. In this
picture: the kept announcements are the three kept interception lines, "train
arrived" is `minimize end`, the signal box messages are `MinimizeEnd`
events, and the new sighting from the platform is the `restore-honored` edge
line.

#### Options for Q10

- Option A: drop `minimize end`; the restore is logged by the edge line only.
  - pro: no line suggests an event still drives a decision.
  - pro: `minimize event: end` lines already record every `MinimizeEnd` with
    its age.
  - con: a grep for `minimize end` over old and new logs no longer lines up.
- Option B: also write `minimize end` next to the `restore-honored` edge
  line.
  - pro: old greps keep working.
  - con: two lines for one fact, and the old line's meaning (event received)
    has changed.

#### Recommended option for Q10 (with arguments for this choice)

Option A: the acceptance evidence counts `minimize intercepted` and `replay
accepted`, which stay; `minimize end` described the event path that this
effort removes, and its information survives in the event and edge lines.

#### Answer to Q10: option A (with reason why it must be accepted as the answer)

Option A: it keeps the lines that evidence depends on and avoids a line whose
meaning no longer holds.

### Q11: Where the manual unplug evidence is kept

Question description: Step 4 ends with the manual three-to-one unplug. Its
`native-host.log` lines from four windows are the acceptance evidence, and
the issue asks to keep them. The plan copies the minimize lines into the
Step 4 section of the validation plan. Where should they be kept, and how
much of them?

#### BBQ for Q11

After a crash test, a lab can staple the full sensor dump to the report,
staple the relevant excerpts per dummy, or keep the dump in a drawer that is
emptied every week. In this picture: the full dump is the four whole
`native-host.log` files, the excerpts are each window's minimize lines, the
report is the validation plan, and the drawer is a git-ignored `a.*` scratch
file.

#### Options for Q11

- Option A: per-window excerpts of the minimize lines (`minimize edge`,
  `minimize event`, own calls, interception, cap and latch lines), with
  their timestamps and a window label, in the Step 4 section of the
  validation plan, in a `text` fence, judged by a per-action table (one row
  per external minimize edge).
  - pro: the evidence is committed next to the verdict it supports, and short
    enough to read.
  - con: the excerpt is a selection, not the raw file.
- Option B: the full logs committed under `docs/v0.0.24/`.
  - pro: complete raw evidence.
  - con: large, and the logs carry local paths and window titles that don't
    belong in the repository.
- Option C: a git-ignored `a.*` file only.
  - pro: nothing extra committed.
  - con: the evidence the issue asks to keep is lost with the working tree.

#### Recommended option for Q11 (with arguments for this choice)

Option A: it keeps the issue's evidence with the implementation check that
judges it, trims it to what the verdict needs, and keeps local details out of
the repository.

#### Answer to Q11: option A (with reason why it must be accepted as the answer)

Option A: committed, focused and privacy-safe evidence, judged per minimize
action where the validation plan records it.
