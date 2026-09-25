# v0.0.24 minimize_loop implementation tracking and validation

No, it is not implemented.

This document tracks the four steps of
[plan.v0.0.24.minimize_loop.md](plan.v0.0.24.minimize_loop.md): the minimize
code split, the observation model with own-call absorption and session latch,
the interception cap, and the acceptance scenarios with documentation and the
manual unplug. None of them has started.

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

Not started. Step 1 is not implemented because `companion/minimize_windows.go`
and `companion/minimize_hook_windows.go` don't exist yet, and the minimize code
and its three tests are still in `companion/main_windows.go` (2022 lines) and
`companion/main_windows_test.go` (661 lines).

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

_(empty: no check has taken place yet.)_.

### New types or classes introduced for Step 1

_(empty: no check has taken place yet.)_.

### Architecture check for Step 1

_(empty: no check has taken place yet.)_.

### Performance check for Step 1

_(empty: no check has taken place yet.)_.

### Unit test coverage check for Step 1

_(empty: no check has taken place yet.)_.

### Feature integrity for Step 1

_(empty: no check has taken place yet.)_.

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
