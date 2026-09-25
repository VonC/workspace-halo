# v0.0.24 minimize_loop implementation tracking and validation

No, it is not implemented.

This document tracks the four steps of
[plan.v0.0.24.minimize_loop.md](plan.v0.0.24.minimize_loop.md): the minimize
code split, the observation model with own-call absorption and session latch,
the interception cap, and the acceptance scenarios with documentation and the
manual unplug. Step 1 is implemented; Steps 2 to 4 have not started.

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

Not started. Step 2 is not implemented because the host still decides from
event order through `minimizeEventTransition`, still re-primes in
`replayPendingMinimize`, and has no `minimizeModel` or `minimizeController`.

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
  resolution latches interception off, and later edges are skipped with
  reason `latched`.
- An own restore that leaves the window iconic composes nothing and logs
  `minimize intercepted: restored=false replay=none`; every own call is
  stamped with the tick taken after its after-reading.
- The pure-model coverage gate finds no uncovered block in
  `minimize_windows.go`.
- No reference to `minimizeEventTransition`, `replayPendingMinimize`,
  `restoreTargetForMinimizePriming`, `minimizeState` or `minimizeReplayAt`
  remains, and `main_windows.go` does not grow.

### What was implemented for Step 2

_(empty: no check has taken place yet.)_.

### New types or classes introduced for Step 2

_(empty: no check has taken place yet.)_.

### Architecture check for Step 2

_(empty: no check has taken place yet.)_.

### Performance check for Step 2

_(empty: no check has taken place yet.)_.

### Unit test coverage check for Step 2

_(empty: no check has taken place yet.)_.

### Feature integrity for Step 2

_(empty: no check has taken place yet.)_.

---

## Step 3. Per-window interception cap with quiet-period reset

### Analysis of Step 3 implementation state

Not started. Step 3 is not implemented because no `minimizeCap`, cap constants
or cap log lines exist yet.

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

_(empty: no check has taken place yet.)_.

### New types or classes introduced for Step 3

_(empty: no check has taken place yet.)_.

### Architecture check for Step 3

_(empty: no check has taken place yet.)_.

### Performance check for Step 3

_(empty: no check has taken place yet.)_.

### Unit test coverage check for Step 3

_(empty: no check has taken place yet.)_.

### Feature integrity for Step 3

_(empty: no check has taken place yet.)_.

---

## Step 4. Acceptance scenarios, documentation and the unplug check

### Analysis of Step 4 implementation state

Not started. Step 4 is not implemented because no acceptance scenario exists,
the wiki still describes the event-driven interception with only the 75 ms
replay delay, `CHANGELOG.md` has no 0.0.24 section, and no manual unplug has
been run with the new host.

### Goal for Step 4

Replay every design acceptance case and the recorded four-window unplug
timeline through `minimizeController` with scripted fake windows and late,
reordered events; update `wiki/reference/display-triggers.md`,
`wiki/explanation/how-the-overlay-stays-inside-its-window.md`,
`wiki/reference/logs-and-processes.md` and `CHANGELOG.md`; build the VSIX and
run the manual three-to-one unplug with four VS Code windows, keeping the log
lines as evidence.

### Step 4 improvement expectations

- `TestMinimizeAcceptanceCases` has one passing sub-test per design acceptance
  row, and the recorded unplug timeline shows at most one restore per window
  and minimize action.
- The timing constants table lists the lateness bound, the settle timeout and
  the cap values, and the explanation page covers the let-through cases.
- `build.bat` exits 0, and markdownlint reports nothing on the four updated
  Markdown files beyond MD013 on table rows.
- The manual unplug log shows no repeated interception cycle and no
  unsolicited delayed or repeated return to the foreground, judged per
  external minimize action in a table (window, edge timestamp, age bound,
  decision, interception attempts, own restores after the accepted replay,
  verdict) backed by per-window timestamped log excerpts; one attempt may
  include a single `SW_RESTORE` fallback.

### What was implemented for Step 4

_(empty: no check has taken place yet.)_.

### New types or classes introduced for Step 4

_(empty: no check has taken place yet.)_.

### Architecture check for Step 4

_(empty: no check has taken place yet.)_.

### Performance check for Step 4

_(empty: no check has taken place yet.)_.

### Unit test coverage check for Step 4

_(empty: no check has taken place yet.)_.

### Feature integrity for Step 4

_(empty: no check has taken place yet.)_.
