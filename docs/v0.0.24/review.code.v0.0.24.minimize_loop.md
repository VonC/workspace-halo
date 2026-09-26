# Code review transcript for v0.0.24

- Exchange: code/code/v0.0.24/minimize_loop
- Reviewed document: docs/v0.0.24/plan.v0.0.24.minimize_loop.md

This append-only transcript records completed review rounds. Review agents add
new entries through the review-exchange core and do not reread earlier entries
as working context.

## Round 1 by requestor - Step 1

- Recorded: 2026-09-25T15:04:06+02:00
- Exchange: code/code/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: unrecorded
- Implementation step: 1
- Outcome: request

### Review identity for step 1 minimize_loop (round 1)

Umbrella draft: none
Implementation plan: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
Implementation step: 1
Review round: 1

### Code review evidence for step 1 minimize_loop (round 1)

request_index_tree: a3d1de99588d263459a4ab2fce3a8511bd0c7bef
resolved_validation_set:

- ghog day (sources: project)
- powershell -NoProfile -ExecutionPolicy Bypass -File scripts\test-companion.ps1 (sources: plan)
- npm test (sources: plan)
- Push-Location companion; gofmt -l .; go vet ./...; Pop-Location (sources: plan)
- git grep -nE 'func (minimizeEventTransition|minimizeTransition)' -- companion/main_windows.go (sources: plan)
- git grep -nE 'func .*(MinimizeHook|WinEventProc|Priming|composeHalo|PendingMinimize)' -- companion/main_windows.go (sources: plan)

commit_plan_result:

```text
state: valid
ready: true
group 1: refactor(minimize_loop): split minimize out of main
group 1 path: companion/minimize_windows.go
group 1 path: companion/minimize_hook_windows.go
group 1 path: companion/main_windows.go
group 1 path: companion/minimize_windows_test.go
group 1 path: companion/main_windows_test.go
group 2: docs(minimize_loop): record step 1 validation
group 2 path: docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md
staged path: companion/main_windows.go
staged path: companion/main_windows_test.go
staged path: companion/minimize_hook_windows.go
staged path: companion/minimize_windows.go
staged path: companion/minimize_windows_test.go
staged path: docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md
```

### Requestor assessment for step 1 minimize_loop (round 1)

Step 1 is fully implemented: it is a verbatim, behavior-neutral move of the
minimize responsibility out of `companion/main_windows.go` and
`companion/main_windows_test.go` into three new files.

- **Tests**: `go test -v` gives 33 `--- PASS` lines before the move (HEAD
  `6752442`) and 33 after, with no `--- FAIL`. `ghog day` ends at `exit=9`
  ("not a pytest project", its complete verdict here);
  `scripts\test-companion.ps1` prints `ok  workspace-halo/companion` and exits
  0; `npm test` exits 0 (8 pass).
- **Static checks**: `gofmt -l .` prints nothing. The three
  `main_windows.go` completion greps print nothing. `go vet ./...` reports
  one finding, `main_windows.go:1327:33: possible misuse of unsafe.Pointer`,
  on the unchanged `unsafe.Slice((*byte)(unsafe.Pointer(bits)), w*h*4)`
  rendering line; `go vet` on HEAD, run in a temporary worktree, reports the
  same finding at `main_windows.go:1505:33`, so the move introduced none.
- **Test-file grep variance**: `git grep -n 'minimize' --
  companion/main_windows_test.go` still prints lines 209, 217, 226 and 235:
  the `minimized` trigger field of `TestVisibilityStatePrecedence`, which
  tests `visibilityState` in `main_windows.go`. The plan keeps that test in
  place while the file stays under 650 lines (it is at 604) and names it only
  as the fallback when the file stays above 650; no minimize test remains.
- **Line budgets**: `main_windows.go` 2022 to 1844 (target at most 1850),
  `main_windows_test.go` 661 to 604 (target at most 650),
  `minimize_windows.go` 72, `minimize_hook_windows.go` 133,
  `minimize_windows_test.go` 66.
- **Coverage**: no coverage gate exists for Go; statically,
  `minimize_windows.go` is at 100% (every branch of `minimizeTransition` and
  `minimizeEventTransition` is in the moved tests).
  `minimize_hook_windows.go` has no unit test, as before; the plan keeps it
  evidence-only and Step 2 adds its controller tests.
- **Architecture**: the pure file has no import and no `proc*` reference; the
  hook file depends on it and on the shared proc declarations, never the
  reverse.
- **Performance**: no body changed, so no computation is added on the tick or
  callback path.
- **Feature integrity**: the same functions run from the same callers and
  write the same `native-host.log` lines.

### Implementation report for step 1 minimize_loop (round 1)

- `companion/minimize_windows.go` (new, no imports): the five WinEvent
  identifiers moved out of the main `const` block
  (`eventSystemMinimizeStart`, `eventSystemMinimizeEnd`, `objidWindow`,
  `childidSelf`, `wineventOutofcontext`), `minimizePhase` and its four
  constants, `minimizeAction` and its four constants,
  `minimizeReplayDelayMS`, `minimizeEventTransition` and
  `minimizeTransition`, with unchanged bodies and comments.
- `companion/minimize_hook_windows.go` (new, imports `fmt`, `syscall`,
  `unsafe`): `var minimizeWinEventCallback =
  syscall.NewCallback(minimizeWinEventProc)`, `installMinimizeHook`,
  `minimizeWinEventProc`, `restoreTargetForMinimizePriming`, `composeHalo`
  and `replayPendingMinimize`, with unchanged bodies and doc comments.
- `companion/main_windows.go`: those declarations removed; the `var` block
  keeps only `activeApp`; imports unchanged; the shared `proc*` and `sw*`
  declarations and all callers (`main`, `close`, `tick`) untouched.
- `companion/minimize_windows_test.go` (new, imports `testing`):
  `TestMinimizeTransitionTargetsOnlyTheTrackedTopLevelWindow`,
  `TestMinimizeEventTransitionPrimesThenAcceptsTheReplay` and
  `TestDuplicateMinimizeStartDoesNotRestartPriming`, moved verbatim.
- `companion/main_windows_test.go`: the three tests removed; every import
  still used.
- Each new file has a short comment after the package clause stating its
  responsibility; that is the only text not moved from the old files.
- `docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md`: Step 1 recorded
  as fully implemented with its evidence and the two variances above; the
  document status stays `No, it is not implemented.` since Steps 2 to 4 are
  pending.

### Change summary for step 1 minimize_loop (round 1)

Staged paths:

- `M companion/main_windows.go`
- `M companion/main_windows_test.go`
- `A companion/minimize_hook_windows.go`
- `A companion/minimize_windows.go`
- `A companion/minimize_windows_test.go`
- `M docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md`

Root `a.commit` groups (formatted by `wac`, `commit-plan-check` ready):

1. `refactor(minimize_loop): split minimize out of main`: the five
   `companion/` files, one commit for the verbatim move as plan Q02 requires.
2. `docs(minimize_loop): record step 1 validation`: the validation plan.

### Writer response for step 1 minimize_loop (round 1)

Writer response:

No earlier feedback: this is round 1 of the Step 1 code review. The writer
asks the reviewer to confirm in particular that the moved code is
byte-identical to the removed code, apart from the new file-header comments,
and to judge the two recorded variances (the pre-existing `go vet` finding and
the test-file `minimize` grep matching `TestVisibilityStatePrecedence`).

### Reviewer focus for step 1 minimize_loop (round 1)

Check the exact plan step, staged implementation, test evidence, repaired path inventory, and a.commit accuracy.

<!-- review-entry-id: request-step-1-round-1 -->

### LLM nature completion for reviewer (exchange 1)

Recorded nature: `codex`

Completed artifacts:

- `.reviews/a.review-active.code.code.v0.0.24.minimize_loop.md`
- `.reviews/a.review-requested.code.v0.0.24.minimize_loop.md`

<!-- review-entry-id: llm-nature-completion-reviewer-exchange-1 -->

## Round 1 by reviewer - Step 1

- Recorded: 2026-09-25T15:37:18+02:00
- Exchange: code/code/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Implementation step: 1
- Outcome: answer

### Assessed index identity for step 1 minimize_loop (exchange 1) (round 1)

Baseline index tree: a3d1de99588d263459a4ab2fce3a8511bd0c7bef

Assessed index tree: 2af4a969625d810a9c97e616b4c6f98f735c9335

### Implementation check for step 1 minimize_loop (exchange 1) (round 1)

Result:

No. Step 1 has NOT been fully implemented under the plan's literal completion criteria. The staged diff shows the pure declarations, Win32 hook and callback, and three named tests moved verbatim; bodies and comments are unchanged. Host call sites remain intact. The writer reports 33 Go test PASS lines before and after, companion and npm tests green, and line counts 1844/604/72/133/66 within targets. The pure file has no Win32 import or call; the moved tests cover its transition and filter statements. The two failing criteria are a clean go vet result and an empty test-file minimize grep.

Validation plan effects:

Updated only the exact Step 1 validation rows in docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md: changed the verdict to the required No sentence and added a Missing work for Step 1 section. This is review metadata only; no implementation code, test, acceptance behavior, or commit grouping changed. The introductory theme sentence still says Step 1 is implemented; the writer should align it when resolving the criteria because reviewer mode cannot edit outside the Step 1 rows.

### Pre-repair mandatory checks and coverage for step 1 minimize_loop (exchange 1) (round 1)

Requestor evidence: ghog day exit 9 ("not a pytest project") as the plan's expected verdict; scripts/test-companion.ps1 exit 0; npm test exit 0 with 8 passes; 33 Go PASS lines before and after; gofmt empty; three main_windows.go grep checks empty. go vet ./... reports possible misuse of unsafe.Pointer at main_windows.go:1327; the requestor reports the same warning on HEAD at old line 1505. The literal test-file grep reports four matches in TestVisibilityStatePrecedence. I did not rerun the requestor's validation set or measure coverage, per reviewer rules. Static inspection finds the moved pure transition and filter statements covered by the moved tests; legacy Win32 glue remains untested under the plan's evidence-only treatment.

### Resolved validation set and sources for step 1 minimize_loop (exchange 1) (round 1)

The request's typed resolver set parses and has six ordered entries: project default ghog day; plan commands powershell -NoProfile -ExecutionPolicy Bypass -File scripts\test-companion.ps1, npm test, Push-Location companion; gofmt -l .; go vet ./...; Pop-Location, and two main_windows.go grep commands. There is no .review-validation project override. Each source label and command matches the current resolver inputs. These commands are requestor-owned and were not rerun. The plan's additional literal test-file grep is a completion criterion outside the six-entry resolver set and is assessed from requestor evidence.

### Resolver drift and direction for step 1 minimize_loop (exchange 1) (round 1)

No resolver drift: the current project default and plan additions produce the same six commands in the same order and with the same source labels as the request. No request additions were declared.

### Repository state around validation for step 1 minimize_loop (exchange 1) (round 1)

The live request-time index tree matched a3d1de99588d263459a4ab2fce3a8511bd0c7bef before assessment. The assessed index tree is 2af4a969625d810a9c97e616b4c6f98f735c9335 after the attributable Step 1 validation-plan update. Umbrella is not applicable and its digest remained unchanged. The before/after validation-state comparison reports only the permitted validation-plan path and resulting `<index>` change; there is no other tracked or untracked validation side effect in the nine-path set. No reviewer evidence command was needed because the exact move is inspectable and the requestor supplied executed test results.

### Repair inventory for step 1 minimize_loop (exchange 1) (round 1)

Repairs made:

- docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md — review-metadata-only Step 1 verdict and missing-work update; non-substantive under the review protocol.

Paths staged:

- docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md — only reviewer-authored path staged; attribution was verified against the pre-repair index blob.

### Commit plan assessment for step 1 minimize_loop (exchange 1) (round 1)

Independent commit-plan-check.bat --format json returned state valid, ready true, no diagnostics after the review metadata edit. The six staged paths remain grouped as five companion files under refactor(minimize_loop): split minimize out of main, then the Step 1 validation plan under docs(minimize_loop): record step 1 validation. The subjects, membership, and order remain accurate; a.commit needs no amendment. git diff --cached --check is clean. This mechanical result does not cure the two failed completion criteria.

### Findings and boundaries for step 1 minimize_loop (exchange 1) (round 1)

Unresolved findings:

- Step 1 requires a clean go vet result, but the reported vet run warns on main_windows.go:1327. The requestor's HEAD comparison indicates a pre-existing rendering warning; the current plan has no baseline exception.
- Step 1 requires the test-file minimize grep to print nothing, but the requestor reports four matches in unrelated TestVisibilityStatePrecedence. The intended three minimize-interception tests did move; the literal check is too broad for that intent.
- The validation document's introductory theme sentence says Step 1 is implemented while its Step 1 verdict now says No under the current criteria.

Boundary-crossing work:

- Fixing the unsafe.Pointer rendering warning would touch code outside the verbatim Step 1 move. A baseline-aware plan criterion is an alternative for the plan owner; reviewer mode cannot change the plan.
- Moving TestVisibilityStatePrecedence merely to satisfy the broad grep would enlarge the agreed three-test move. A declaration-specific criterion is an alternative for the plan owner.

### Writer instructions for step 1 minimize_loop (exchange 1) (round 1)

Resolve both literal Step 1 gates before the next review: obtain a clean go vet result through an authorized, separately scoped fix or amend the plan to accept the documented baseline warning while prohibiting new ones; make the test-file grep accurately check for remaining minimize-interception tests or authorize moving the unrelated visibility test. Record the chosen checks and results, update the validation plan's Step 1 verdict and introductory theme sentence consistently, then request another review round. Keep the code move and test count evidence intact.

### Decision rationale for step 1 minimize_loop (exchange 1) (round 1)

Changes requested. Identity, staged scope, byte-preserving move, line budgets, Go/npm test evidence, static pure-code coverage, and commit grouping are satisfactory. The readiness floor is incomplete because two explicit completion criteria fail. The reviewer made only an attributable validation-plan metadata update, not a substantive repair. No commit is authorized.

### Final reviewer decision for step 1 minimize_loop (exchange 1) (round 1)

Decision: changes-requested. The writer must address the concrete instructions and publish another review round. This advisory answer does not authorize a commit.

<!-- review-entry-id: answer-step-1-round-1 -->

## Round 2 by requestor - Step 1

- Recorded: 2026-09-25T16:31:45+02:00
- Exchange: code/code/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Implementation step: 1
- Outcome: request

### Review identity for step 1 minimize_loop (round 2)

Umbrella draft: none
Implementation plan: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
Implementation step: 1
Review round: 2

### Code review evidence for step 1 minimize_loop (round 2)

request_index_tree: 49799b6bfb58327b69e829351f11e57705eb232c
resolved_validation_set:

- ghog day (sources: project)
- powershell -NoProfile -ExecutionPolicy Bypass -File scripts\test-companion.ps1 (sources: plan)
- npm test (sources: plan)
- Push-Location companion; gofmt -l .; go vet ./...; Pop-Location (sources: plan)
- git grep -nE 'func (minimizeEventTransition|minimizeTransition)' -- companion/main_windows.go (sources: plan)
- git grep -nE 'func .*(MinimizeHook|WinEventProc|Priming|composeHalo|PendingMinimize)' -- companion/main_windows.go (sources: plan)
- git grep -nE 'minimize(Transition|EventTransition|Idle|Priming|Replaying|Committed)|eventSystemMinimize' -- companion/main_windows_test.go (sources: plan)

commit_plan_result:

```text
state: valid
ready: true
group 1: docs(minimize_loop): amend step 1 completion checks
group 1 path: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
group 2: refactor(minimize_loop): split minimize out of main
group 2 path: companion/minimize_windows.go
group 2 path: companion/minimize_hook_windows.go
group 2 path: companion/main_windows.go
group 2 path: companion/minimize_windows_test.go
group 2 path: companion/main_windows_test.go
group 3: docs(minimize_loop): record step 1 review transcript
group 3 path: docs/v0.0.24/review.code.v0.0.24.minimize_loop.md
group 4: docs(minimize_loop): record step 1 validation
group 4 path: docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md
staged path: companion/main_windows.go
staged path: companion/main_windows_test.go
staged path: companion/minimize_hook_windows.go
staged path: companion/minimize_windows.go
staged path: companion/minimize_windows_test.go
staged path: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
staged path: docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md
staged path: docs/v0.0.24/review.code.v0.0.24.minimize_loop.md
```

### Requestor assessment for step 1 minimize_loop (round 2)

Step 1 is fully implemented, and the two completion criteria that round 1
found unmet are now met, as the plan owner (the human) chose between the
alternatives round 1 named.

- **go vet**: the owner chose to fix the finding rather than accept a
  baseline. `renderOverlay` declares the `CreateDIBSection` pixel address
  `bits` as `unsafe.Pointer` instead of `uintptr` and builds the slice as
  `unsafe.Slice((*byte)(bits), w*h*4)`, with a two-line comment giving the
  reason. `go vet ./...` now exits 0 with no finding. The same address
  reaches the same slice, so rendering is unchanged.
- **Commit placement of the fix**: the owner first chose a separate
  `fix(companion)` commit, but `a.commit` stages whole files per group and
  rejects a path in two groups, and a `main_windows.go`-only commit would not
  build. The owner then chose to fold the change into the move commit, where a
  dedicated What bullet names it. The plan's Step 1 `main_windows.go` entry
  now states this.
- **Test-file grep**: the owner chose to amend the plan. The Step 1
  completion block now runs
  `git grep -nE "$pat|eventSystemMinimize" -- companion/main_windows_test.go`
  with
  `$pat = 'minimize(Transition|EventTransition|Idle|Priming|Replaying|Committed)'`,
  and explains why `TestVisibilityStatePrecedence` stays. It prints nothing on
  `main_windows_test.go` and matches 17 lines of `minimize_windows_test.go`,
  so it still detects any minimize test that has not moved.
- **Tests after the fix**: `go test -v` gives 33 `--- PASS` lines; `gofmt -l`
  prints nothing; `ghog day` ends at `exit=9`; `scripts\test-companion.ps1`
  exits 0; `npm test` exits 0 (8 pass).
- **Line budgets**: `main_windows.go` 1846 (target at most 1850; the vet
  comment added 2 lines), `main_windows_test.go` 604, `minimize_windows.go`
  72, `minimize_hook_windows.go` 133, `minimize_windows_test.go` 66. The
  Step 2 baseline in the plan now reads about 1846.
- **Validation plan**: Step 1 verdict back to the exact Yes sentence, the
  missing-work section removed since both items are closed, the introductory
  sentence consistent ("Step 1 is implemented"), and the evidence, architecture,
  performance and feature-integrity sections updated for the vet fix.
- **Coverage, architecture, performance**: unchanged from round 1; the vet fix
  stays inside the rendering adapter and adds no computation.
- **Markdown**: markdownlint is clean on the validation plan; on the plan it
  reports only the MD013 table-row findings of the decision table, which were
  there before.

### Implementation report for step 1 minimize_loop (round 2)

Round 2 changes on top of the round 1 move:

- `companion/main_windows.go`: in `renderOverlay`, `var bits uintptr` becomes
  `var bits unsafe.Pointer` with a comment giving the reason, and
  `unsafe.Slice((*byte)(unsafe.Pointer(bits)), w*h*4)` becomes
  `unsafe.Slice((*byte)(bits), w*h*4)`. The `CreateDIBSection` call passes
  `uintptr(unsafe.Pointer(&bits))` as before.
- `docs/v0.0.24/plan.v0.0.24.minimize_loop.md`: the Step 1 `main_windows.go`
  entry adds the `bits` type change and why it rides in the move commit; the
  Step 1 test-file grep becomes the symbol-specific check, with its
  rationale; the Step 2 `main_windows.go` baseline becomes about 1846.
- `docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md`: Step 1 verdict
  Yes, missing-work section removed, evidence and checks updated for the vet
  fix and the amended grep.
- `docs/v0.0.24/review.code.v0.0.24.minimize_loop.md`: the exchange's
  versioned transcript, now staged for its own commit.

The round 1 move itself is unchanged: same five `companion/` files, same
moved bodies and comments.

### Change summary for step 1 minimize_loop (round 2)

Staged paths:

- `M companion/main_windows.go`
- `M companion/main_windows_test.go`
- `A companion/minimize_hook_windows.go`
- `A companion/minimize_windows.go`
- `A companion/minimize_windows_test.go`
- `M docs/v0.0.24/plan.v0.0.24.minimize_loop.md`
- `M docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md`
- `A docs/v0.0.24/review.code.v0.0.24.minimize_loop.md`

Root `a.commit` groups (formatted by `wac`, `commit-plan-check` ready):

1. `docs(minimize_loop): amend step 1 completion checks`: the plan.
2. `refactor(minimize_loop): split minimize out of main`: the five
   `companion/` files, including the `renderOverlay` vet fix in its own What
   bullet.
3. `docs(minimize_loop): record step 1 review transcript`: the transcript.
4. `docs(minimize_loop): record step 1 validation`: the validation plan.

### Writer response for step 1 minimize_loop (round 2)

Writer response:

All three round 1 findings are accepted, and none is disputed.

- Vet finding: fixed in code, as the plan owner chose; `go vet` is clean.
- Test-file grep: the plan owner amended the plan to a symbol-specific
  check; it prints nothing.
- Validation plan: the reviewer's Step 1 metadata edit was superseded by
  the writer's update now that both criteria are met. The verdict is back to
  the exact Yes sentence, the missing-work section it added is removed
  because its two items are done, and the introductory sentence agrees with
  the verdict.

One deviation from the owner's first choice: the vet fix is not a separate
`fix(companion)` commit. `a.commit` cannot put `main_windows.go` in two groups,
so the owner chose to fold it into the move commit with its own What bullet.

### Reviewer focus for step 1 minimize_loop (round 2)

Check the exact plan step, staged implementation, test evidence, repaired path inventory, and a.commit accuracy.

<!-- review-entry-id: request-step-1-round-2 -->

## Round 2 by reviewer - Step 1

- Recorded: 2026-09-25T16:35:15+02:00
- Exchange: code/code/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Implementation step: 1
- Outcome: answer

### Assessed index identity for step 1 minimize_loop (exchange 1) (round 2)

Baseline index tree: 49799b6bfb58327b69e829351f11e57705eb232c

Assessed index tree: 49799b6bfb58327b69e829351f11e57705eb232c

### Implementation check for step 1 minimize_loop (exchange 1) (round 2)

Result:

Yes. Step 1 has been fully implemented. The staged move preserves the minimize functions and three named tests, while the owner-authorized rendering fix replaces a uintptr pixel address with unsafe.Pointer and uses that same address to construct the same slice. The four Step 1 file targets are met: main_windows.go 1846, main_windows_test.go 604, pure file 72, hook file 133, test file 66. The plan now explicitly includes the rendering fix and narrows the test-file grep to minimize-interception symbols. The requestor reports clean gofmt and go vet, unchanged 33 Go PASS results, companion and npm tests green, and the amended grep empty on the old test file. Static inspection finds the pure file free of Win32 calls and its transitions/filter statements exercised by the moved tests; the existing Win32 adapter test gap is deferred by the plan. No implementation gap remains for Step 1.

Validation plan effects:

No reviewer edit in round 2. The writer's staged validation plan has the exact Yes sentence for Step 1, no Missing work section, accurate line counts, the clean vet result, the amended grep evidence, and a consistent introductory theme sentence. Steps 2 to 4 remain pending, so the document-level No status remains correct.

### Pre-repair mandatory checks and coverage for step 1 minimize_loop (exchange 1) (round 2)

Requestor-run evidence after the fix: ghog day exit 9 as the plan expects for this non-pytest project; scripts/test-companion.ps1 exit 0; npm test exit 0 with 8 passes; Go test count 33 PASS before and after; gofmt empty; go vet ./... exit 0; all four Step 1 completion greps empty on their old files. The new test-file pattern matches the moved tests, so it is not vacuous. I did not rerun the requestor-owned validation set or measure coverage, per reviewer rules. Static inspection of the moved pure file and its tests finds all transition/filter statements exercised. No reviewer focused command was needed to settle a remaining code risk.

### Resolved validation set and sources for step 1 minimize_loop (exchange 1) (round 2)

The typed seven-entry set parses. Sources: project default ghog day; plan additions scripts/test-companion.ps1, npm test, combined gofmt/go vet command, two main_windows.go grep commands, and the new symbol-specific main_windows_test.go grep command. There is no .review-validation override or request addition. The source labels, order, and command strings match the current request and amended plan. These are requestor-owned validation commands and were not rerun by the reviewer.

### Resolver drift and direction for step 1 minimize_loop (exchange 1) (round 2)

No drift in either direction: the current resolver's project default and amended plan additions reproduce the seven request entries with identical source labels and order.

### Repository state around validation for step 1 minimize_loop (exchange 1) (round 2)

The request-time and assessed index trees both equal 49799b6bfb58327b69e829351f11e57705eb232c. The umbrella is not applicable and its digest remained unchanged. Validation-state before/after comparison over all eight staged paths and three known ghog artifacts is acceptable, with no tracked or untracked path change. No reviewer path was edited or staged. The default Git whitespace check flags CR characters on newly added lines of the existing CRLF plan file; the CRLF-aware check passes without whitespace findings.

### Repair inventory for step 1 minimize_loop (exchange 1) (round 2)

Repairs made: None.

Paths staged: None.

### Commit plan assessment for step 1 minimize_loop (exchange 1) (round 2)

Independent commit-plan-check.bat --format json returned state valid, ready true, no diagnostics, and all eight staged paths accounted for. Group order and subjects are accurate: amend Step 1 plan criteria; move the five companion files with the explicitly documented vet fix; record the review transcript; record Step 1 validation. a.commit needs no amendment. The mechanical check does not itself authorize a commit.

### Findings and boundaries for step 1 minimize_loop (exchange 1) (round 2)

Unresolved findings: None.

Boundary-crossing work: None.

### Writer instructions for step 1 minimize_loop (exchange 1) (round 2)

No changes requested for Step 1. This commit-ready recommendation is advisory; the requestor and human owner retain commit authority. Keep the four declared commit groups when seeking that authority.

### Decision rationale for step 1 minimize_loop (exchange 1) (round 2)

Recommend commit-ready. Exact identity and index tree match; Step 1 implementation and amended completion gates are satisfied; requestor validation is green under the plan's stated ghog exit convention; static coverage reveals no new Step 1 gap; staged scope is attributable; no findings or reviewer repairs remain; and a.commit passes its independent check. The owner-authorized code fix was submitted by the writer before this review round, so this reviewer made no substantive repair.

### Final reviewer decision for step 1 minimize_loop (exchange 1) (round 2)

Decision: commit-ready (advisory). The evidence floor is complete, but this recommendation does not authorize a commit; authority remains at the durable human gate.

<!-- review-entry-id: answer-step-1-round-2 -->

## Round 2 by human - Step 1 - human-confirmation

- Recorded: 2026-09-25T16:44:31+02:00
- Exchange: code/code/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Implementation step: 1
- Outcome: human-confirmation

Human choice: Commit
Outcome: continue-owning-workflow

<!-- review-entry-id: human-confirmation-round-2 -->

## Round 1 by requestor - Step 2

- Recorded: 2026-09-25T18:13:36+02:00
- Exchange: code/code/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: unrecorded
- Implementation step: 2
- Outcome: request

### Review identity for step 2 minimize_loop (round 1)

Umbrella draft: none
Implementation plan: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
Implementation step: 2
Review round: 1

### Code review evidence for step 2 minimize_loop (round 1)

request_index_tree: edcde4e1122dbca37d3a3f3ab5a409861c849d97
resolved_validation_set:

- ghog day (sources: project)
- powershell -NoProfile -ExecutionPolicy Bypass -File scripts\test-companion.ps1 (sources: plan)
- npm test (sources: plan)
- Push-Location companion; gofmt -l .; go vet ./...; Pop-Location (sources: plan)
- git grep --untracked -nE 'minimizeEventTransition|PendingMinimize' -- companion (sources: plan)
- git grep --untracked -n 'ForMinimizePriming' -- companion (sources: plan)
- git grep --untracked -nE 'minimize(Replaying|Committed|Idle)' -- companion (sources: plan)
- git grep --untracked -nE 'minimize(State|ReplayAt)' -- companion (sources: plan)
- git grep -nE 'procShowWindow|procIsIconic' -- companion/minimize_windows.go (sources: plan)
- git grep -nE 'NewProc\(.GetTickCount64.\)' -- companion/main_windows.go (sources: plan)
- pure-model coverage gate over a.cover.minimize.out (plan command templates) (sources: plan)

commit_plan_result:

```text
state: valid
ready: true
group 1: fix(minimize_loop): drive minimize from IsIconic
group 1 path: companion/minimize_windows.go
group 1 path: companion/minimize_hook_windows.go
group 1 path: companion/main_windows.go
group 1 path: companion/minimize_windows_test.go
group 1 path: companion/minimize_hook_windows_test.go
group 2: docs(minimize_loop): record step 2 validation
group 2 path: docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md
staged path: companion/main_windows.go
staged path: companion/minimize_hook_windows.go
staged path: companion/minimize_hook_windows_test.go
staged path: companion/minimize_windows.go
staged path: companion/minimize_windows_test.go
staged path: docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md
```

### Requestor assessment for step 2 minimize_loop (round 1)

Step 2 is fully implemented; the implementation check wrote the exact Yes
sentence into the validation plan.

- **Tests**: `go test -v` gives 51 `--- PASS` lines (33 before, minus the two
  deleted event-order tests, plus 20 new). `FuzzMinimizeModelObservations`
  runs its nine `f.Add` seeds under plain `go test`; no fuzzing session was
  run, as the plan says.
- **Static checks**: `gofmt -l` prints nothing and `go vet ./...` exits 0. The
  first five Step 2 completion greps print nothing, and the sixth finds one
  line, `companion/main_windows.go:161`, in the `kernel32` proc block.
- **Gate**: `ghog day` ends at `exit=9` ("not a pytest project"),
  `scripts\test-companion.ps1` prints `ok  workspace-halo/companion` and exits
  0, and `npm test` exits 0 (8 pass).
- **Coverage**: the plan's pure-model coverage gate over
  `a.cover.minimize.out` passes, so `minimize_windows.go` has no zero-count
  block. In `minimize_hook_windows.go`, every controller method is at 100%;
  the five Win32 adapter functions (`installMinimizeHook`,
  `minimizeWinEventProc`, `isIconic`, `showWindow`, `composeHalo`) are at 0%,
  which is evidence-only under plan Q08.
- **Architecture**: the pure model has no import and no `proc*` reference.
  The controller reaches the window only through the `minimizeWindow` port,
  and `*application` is its Win32 adapter. The controller and adapter share
  one file, as plan Q01 chose. The global `activeApp` is carried over, with
  a new nil guard on the controller.
- **Performance**: every transition is constant-size. A tick now makes two
  `IsIconic` calls (the controller's observation and the existing visibility
  reading). `GetTickCount64` is resolved once from the proc block, and quiet
  ticks write no log line.
- **Feature integrity**: the kept interception lines are unchanged. The
  `minimize end` and re-prime lines are replaced by the edge and event lines,
  as plan Q10 decided. The `minimized` visibility trigger stays on in every
  phase except Shown.
- **Line budgets**: `main_windows.go` stays at 1846 (mandatory: no growth;
  advisory net -1, actual net 0). The four minimize files are at 277, 251,
  401 and 286 lines, above their advisory estimates but below 550, so no split
  applies; the validation plan records the variance.
- **One planned assertion deviates**: the plan asks
  `TestMinimizeControllerStampsOwnCallsAfterTheAfterReading` for an external
  minimize 30 ms after the restore's after-reading to be "intercepted with
  `age<=30ms`". After that restore the model is in Priming, where design Q04
  and this plan's own Priming test say an edge cancels the replay and is never
  intercepted. So the test asserts the stamping on the
  `minimize edge: shown->iconic age<=30ms action=cancel-replay` line, and
  checks that neither `unknown-age` nor `age<=150ms` appears. It also asserts
  `replayAt` = after-reading tick + 75 and the unsettled deadline =
  after-reading tick + 1000, with a 120 ms `ShowWindow`.

### Implementation report for step 2 minimize_loop (round 1)

Step 2 changes on top of the Step 1 split:

- `companion/minimize_windows.go`: deletes `minimizeEventTransition`, the
  Idle, Replaying and Committed phases and the old actions. Adds the phases
  Shown, Priming, Minimized and Unsettled, `minimizeEdge` and the redefined
  `minimizeAction` (each with a log name), `minimizeLatenessBoundMS = 500`,
  `minimizeSettleTimeoutMS = 1000`, `minimizeModel`, `minimizeDecision`,
  `newMinimizeModel`, `observe` (resolve an expired Unsettled from the last
  reading and latch, then handle the edge with the phase rules),
  `ownCall`, `replayDue`, `showsMinimizedTrigger` and `minimizeReadingName`.
  `minimizeTransition` is unchanged apart from its doc comment.
- `companion/minimize_hook_windows.go`: adds the `minimizeWindow` interface
  and `minimizeController` (`newMinimizeController`, `observe`, `onEvent`,
  `logEdge`, `intercept`, `replay`, `ownRestore`, `ownReplay`,
  `settleOwnCall`). Own calls read before and after, take `clock()` after the
  after-reading, and log their outcome. An intercept composes the halo only
  when the restore settled shown. `minimizeWinEventProc` reads
  `dwmsEventTime` and calls `onEvent`. `*application` gains `isIconic` and
  `showWindow`. `restoreTargetForMinimizePriming` and `replayPendingMinimize`
  are deleted.
- `companion/main_windows.go`: `minimize *minimizeController` replaces
  `minimizeState` and `minimizeReplayAt`. `main` builds the controller before
  `installMinimizeHook`, and `tick` calls `a.minimize.observe(now)` and feeds
  `showsMinimizedTrigger()` to `visibilityState`. `procGetTickCount64` joins
  the proc block. The file stays at 1846 lines.
- `companion/minimize_windows_test.go`: keeps the filter test, removes the two
  event-order tests, and adds the ten planned model tests, an
  unsettled-restore test, a naming test, and the fuzz target with nine seeds.
- `companion/minimize_hook_windows_test.go` (new): `fakeMinimizeWindow`, a
  controller fixture with a fake clock and a buffered logger, the six planned
  controller tests, and a test for the render-error, unknown-age and
  restore-honored log lines.
- `docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md`: Step 2 recorded as
  fully implemented, with its evidence and checks; the summary sentence now
  says Steps 1 and 2 are implemented.

### Change summary for step 2 minimize_loop (round 1)

Staged paths:

- `M companion/main_windows.go`
- `M companion/minimize_hook_windows.go`
- `A companion/minimize_hook_windows_test.go`
- `M companion/minimize_windows.go`
- `M companion/minimize_windows_test.go`
- `M docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md`

Root `a.commit` groups (formatted by `wac`, `commit-plan-check` ready):

1. `fix(minimize_loop): drive minimize from IsIconic`: the five `companion/`
   files. Code and tests stay together, because the old tests call the
   deleted `minimizeEventTransition` and would not build alone.
2. `docs(minimize_loop): record step 2 validation`: the validation plan.

### Writer response for step 2 minimize_loop (round 1)

Writer response:

First round of the Step 2 review: no earlier feedback to accept or dispute.
The one known deviation from the plan's wording (the stamping test asserts
the cancel-replay edge line rather than an interception) is explained in the
assessment for the reviewer to judge.

### Reviewer focus for step 2 minimize_loop (round 1)

Check the exact plan step, staged implementation, test evidence, repaired path inventory, and a.commit accuracy.

<!-- review-entry-id: request-step-2-round-1 -->

## Round 1 by reviewer - Step 2

- Recorded: 2026-09-25T19:02:26+02:00
- Exchange: code/code/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Implementation step: 2
- Outcome: answer

### Assessed index identity for step 2 minimize_loop (exchange 1) (round 1)

Baseline index tree: edcde4e1122dbca37d3a3f3ab5a409861c849d97

Assessed index tree: a47cd55e8d326aab6ee27c0da55a1341d4f78107

### Implementation check for step 2 minimize_loop (exchange 1) (round 1)

Result:

No. Step 2 has NOT been fully implemented.

The model and controller match most Step 2 wiring, and the Win32 seam, constant-time transitions, and stated coverage gate are present. In `minimizeModel.shownToIconic`, the age-bound check precedes the session-latch check. A post-latch edge older than 500 ms therefore reports `reason=unknown-age` instead of the design's required `reason=latched`. `TestMinimizeModelLatchedSkipsEveryLaterEdge` only gives the model a shown reading 20 ms before each edge, so it cannot catch that case. No further architecture or performance defect was found in the staged step.

Validation plan effects:

Updated only the Step 2 validation rows: changed the Step 2 verdict to the exact No sentence, corrected the description of the skip-reason priority, and added concrete missing work for the model and its latch test. This is review metadata, not a substantive implementation repair. The document-level status remains No because Steps 3 and 4 are pending.

### Pre-repair mandatory checks and coverage for step 2 minimize_loop (exchange 1) (round 1)

Requestor evidence says the Go suite, gofmt, go vet, completion greps, pure-model coverage gate, companion test script, and npm test passed. `ghog day` returned the plan's expected exit 9 for this non-pytest project. The reviewer did not rerun the requestor validation set or coverage. Static inspection found the missing post-latch late-edge assertion despite the claimed 100% statement coverage.

### Resolved validation set and sources for step 2 minimize_loop (exchange 1) (round 1)

The request lists 11 commands: the project `ghog day` default, the plan's companion test script, npm test, Go format/vet, six Step 2 completion greps, and the pure-model coverage gate. The current Step 2 plan and shared command templates still declare this same set; there is no request-only addition.

### Resolver drift and direction for step 2 minimize_loop (exchange 1) (round 1)

No drift. The request and current plan have the same 11 validation commands and sources; none was added, removed, or reordered.

### Repository state around validation for step 2 minimize_loop (exchange 1) (round 1)

Request-time and baseline index tree: `edcde4e1122dbca37d3a3f3ab5a409861c849d97`. Assessed index tree after the attributable Step 2 validation-plan update: `a47cd55e8d326aab6ee27c0da55a1341d4f78107`. Validation-state comparison shows only the reviewed Step 2 validation-plan path and the corresponding index change; the existing unstaged review transcript is protocol output. Umbrella digest is not applicable and did not change. No implementation or test file was edited by the reviewer.

### Repair inventory for step 2 minimize_loop (exchange 1) (round 1)

Repairs made:

- docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md — review metadata only: recorded No and the missing model/test work. No substantive code, test, acceptance, or commit-group repair was made.

Paths staged:

- companion/main_windows.go
- companion/minimize_hook_windows.go
- companion/minimize_hook_windows_test.go
- companion/minimize_windows.go
- companion/minimize_windows_test.go
- docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md

### Commit plan assessment for step 2 minimize_loop (exchange 1) (round 1)

Independent `commit-plan-check.bat --format json` returned `state=valid`, `ready=true`, six staged paths, two ordered groups, and no mechanical diagnostics, both before and after the validation-plan update. Group 1 covers the five code/test paths; Group 2 covers the validation plan. The Group 2 body currently says Step 2 is fully implemented and the gates are green, which is inaccurate while this finding is open. The writer must refresh that body after the implementation and validation record are corrected; mechanical readiness alone does not establish semantic accuracy.

### Findings and boundaries for step 2 minimize_loop (exchange 1) (round 1)

Unresolved findings:

- After the session latch, a shown-to-iconic edge with age bound over 500 ms reports `unknown-age` because `shownToIconic` tests the age before `m.latched`. The design explicitly requires every post-latch shown-to-iconic edge to report `latched`, regardless of delay. The current latch test checks only age 20 ms, so this behavior is untested.

Boundary-crossing work: None.

### Writer instructions for step 2 minimize_loop (exchange 1) (round 1)

In `companion/minimize_windows.go`, prioritize the session latch over the age bound when choosing the skip reason for a shown-to-iconic edge in Shown. Extend `TestMinimizeModelLatchedSkipsEveryLaterEdge` with a post-latch edge older than 500 ms, while retaining an unlatched late-edge test for `unknown-age`. Re-run the Step 2 validation set, update the Step 2 validation rows to Yes only after the behavior and test pass, correct the Group 2 `a.commit` body, and publish a new review round.

### Decision rationale for step 2 minimize_loop (exchange 1) (round 1)

Changes requested. Exact identity and request-time index match passed. The staged implementation is incomplete against the explicit post-latch logging rule, and an unresolved finding remains. Requestor validation evidence and static coverage do not exercise the missing case. The reviewer-authored validation-plan metadata patch was attributable and staged, with no boundary crossing. Commit grouping is mechanically ready, but its Group 2 description is semantically stale. The six-part readiness floor is incomplete, so commit-ready is not warranted.

### Final reviewer decision for step 2 minimize_loop (exchange 1) (round 1)

Decision: changes-requested. The writer must address the concrete instructions and publish another review round. This advisory answer does not authorize a commit.

<!-- review-entry-id: answer-step-2-round-1 -->

## Round 2 by requestor - Step 2

- Recorded: 2026-09-25T19:10:40+02:00
- Exchange: code/code/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Implementation step: 2
- Outcome: request

### Review identity for step 2 minimize_loop (round 2)

Umbrella draft: none
Implementation plan: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
Implementation step: 2
Review round: 2

### Code review evidence for step 2 minimize_loop (round 2)

request_index_tree: 5b3def582558e149983cafc33698f66ba8357dba
resolved_validation_set:

- ghog day (sources: project)
- powershell -NoProfile -ExecutionPolicy Bypass -File scripts\test-companion.ps1 (sources: plan)
- npm test (sources: plan)
- Push-Location companion; gofmt -l .; go vet ./...; Pop-Location (sources: plan)
- git grep --untracked -nE 'minimizeEventTransition|PendingMinimize' -- companion (sources: plan)
- git grep --untracked -n 'ForMinimizePriming' -- companion (sources: plan)
- git grep --untracked -nE 'minimize(Replaying|Committed|Idle)' -- companion (sources: plan)
- git grep --untracked -nE 'minimize(State|ReplayAt)' -- companion (sources: plan)
- git grep -nE 'procShowWindow|procIsIconic' -- companion/minimize_windows.go (sources: plan)
- git grep -nE 'NewProc\(.GetTickCount64.\)' -- companion/main_windows.go (sources: plan)
- pure-model coverage gate over a.cover.minimize.out (plan command templates) (sources: plan)

commit_plan_result:

```text
state: valid
ready: true
group 1: fix(minimize_loop): drive minimize from IsIconic
group 1 path: companion/minimize_windows.go
group 1 path: companion/minimize_hook_windows.go
group 1 path: companion/main_windows.go
group 1 path: companion/minimize_windows_test.go
group 1 path: companion/minimize_hook_windows_test.go
group 2: docs(minimize_loop): record step 2 validation
group 2 path: docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md
staged path: companion/main_windows.go
staged path: companion/minimize_hook_windows.go
staged path: companion/minimize_hook_windows_test.go
staged path: companion/minimize_windows.go
staged path: companion/minimize_windows_test.go
staged path: docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md
```

### Requestor assessment for step 2 minimize_loop (round 2)

Step 2 is fully implemented under the plan's skip-reason precedence, and the
round 1 test gap is closed.

- **Precedence**: `unknown-age`, then `latched` (then `cap` in Step 3), as
  plan Step 3 item 3 and the design's target-behavior pseudo-code order it.
  A post-latch late edge is skipped as `unknown-age`, a post-latch prompt edge
  as `latched`; neither is intercepted.
- **Tests after the repair**: `go test -v` gives 51 `--- PASS` lines (the new
  case extends an existing test). `gofmt -l` prints nothing, `go vet ./...`
  exits 0, and the targeted `-run 'Minimize'` run passes.
- **Gate**: `ghog day` ends at `exit=9`, `scripts\test-companion.ps1` exits 0,
  and `npm test` exits 0 (8 pass).
- **Coverage**: the pure-model coverage gate over a fresh
  `a.cover.minimize.out` passes; the controller methods stay at 100%, and the
  five Win32 adapter functions stay evidence-only under plan Q08.
- **Static checks**: the six Step 2 completion greps give the same output as
  round 1; markdownlint is clean on the validation plan.
- **Architecture, performance, feature integrity**: unchanged from round 1;
  the repair adds one test case and one comment.
- **Planned-assertion deviation** (unchanged from round 1): the stamping test
  checks `age<=30ms` on the cancel-replay line, because an edge in Priming is
  never intercepted.

### Implementation report for step 2 minimize_loop (round 2)

Round 2 changes on top of the round 1 implementation:

- `companion/minimize_windows_test.go`: `TestMinimizeModelLatchedSkipsEveryLaterEdge`
  ends with a post-latch shown-to-iconic edge whose bound is 501 ms, asserting
  `minimizeSkip`, `minimizeReasonUnknownAge` and a kept latch, with a comment
  giving the precedence (410 lines).
- `companion/minimize_windows.go`: the `shownToIconic` doc comment states the
  skip-reason order, unknown-age before latched; no statement changed
  (279 lines).
- `docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md`: Step 2 verdict
  restored to Yes, reviewer missing-work section removed, analysis paragraph
  records the round 1 finding and the decision, model-test bullet and line
  counts refreshed.

The round 1 model, controller, wiring and other tests are unchanged.

### Change summary for step 2 minimize_loop (round 2)

Staged paths:

- `M companion/main_windows.go`
- `M companion/minimize_hook_windows.go`
- `A companion/minimize_hook_windows_test.go`
- `M companion/minimize_windows.go`
- `M companion/minimize_windows_test.go`
- `M docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md`

Root `a.commit` groups (formatted by `wac`, `commit-plan-check` ready):

1. `fix(minimize_loop): drive minimize from IsIconic`: the five `companion/`
   files. Code and tests stay together, because the old tests call the
   deleted `minimizeEventTransition` and would not build alone.
2. `docs(minimize_loop): record step 2 validation`: the validation plan.

### Writer response for step 2 minimize_loop (round 2)

Writer response:

Round 1 raised one finding: after the latch, a shown-to-iconic edge with an
age bound over 500 ms reports `unknown-age` instead of `latched`. The writer
accepts the test gap and disagrees with the reordering.

- **Disagreement on the precedence**: the skip-reason order stays
  `unknown-age`, then `latched`. The design's Target Behavior pseudo-code
  lists `bound > lateness bound -> skip "unknown-age"` before
  `cap closed or latched -> skip "cap" or "latched"`. Plan Step 3, item 3,
  fixes the precedence explicitly: "`unknown-age` when the bound exceeds
  500 ms; else `latched`; else `cap`", "which follows the design's
  target-behavior order". Its `TestMinimizeCapStaysClosedDuringAContinuingStream`
  feeds a latched model the same stream and expects old edges to report
  `unknown-age`. Moving the latch first would contradict that settled plan
  and the Step 3 test. The design prose "every shown-to-iconic edge is skipped
  with reason `latched`, whenever the delayed own call applies" describes an
  own call that applies late while the host keeps ticking, so its edge is
  prompt and does report `latched` (covered by
  `TestMinimizeControllerDeferredReplayLatches`). Either reason skips the
  edge, so no interception differs.
- **Test gap accepted**: `TestMinimizeModelLatchedSkipsEveryLaterEdge` now
  also feeds a post-latch edge with a 501 ms bound and asserts `unknown-age`
  with the latch kept, which pins the precedence. The `shownToIconic` doc
  comment now states the order.
- **Reviewer validation metadata reversed**: the Step 2 verdict is back to the
  exact Yes sentence and the reviewer's missing-work section is removed,
  because the writer keeps the plan's precedence. The analysis paragraph
  records the round 1 finding and this decision. The reviewer's clearer
  wording of the skip order in "What was implemented for Step 2" ("even after
  the latch") is kept, since it describes the code accurately.
- **Commit plan**: groups and subjects are unchanged. Group 2's "fully
  implemented" body is accurate again under the writer's decision, so
  `a.commit` needs no amendment; `commit-plan-check` is still ready.

### Reviewer focus for step 2 minimize_loop (round 2)

Check the exact plan step, staged implementation, test evidence, repaired path inventory, and a.commit accuracy.

<!-- review-entry-id: request-step-2-round-2 -->

## Round 2 by reviewer - Step 2

- Recorded: 2026-09-25T19:16:46+02:00
- Exchange: code/code/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Implementation step: 2
- Outcome: answer

### Assessed index identity for step 2 minimize_loop (exchange 1) (round 2)

Baseline index tree: 5b3def582558e149983cafc33698f66ba8357dba

Assessed index tree: 6eeaabc8ce68987844dee66c5d7b92d7a707f19c

### Implementation check for step 2 minimize_loop (exchange 1) (round 2)

Result:

Yes. Step 2 is fully implemented. The pure minimizeModel and Win32 adapter satisfy the Step 2 plan, including phase transitions, age bounds, latch behavior, one composition before replay, logging, and the planned test migration. The prior round's requested post-latch skip reason was incorrect: plan Step 3 item 3 and the design target-behavior sequence give unknown-age precedence over latched. A new test asserts that a post-latch edge over 500 ms skips as unknown-age while the latch remains set. No current implementation gap remains.

Validation plan effects:

The exact Step 2 validation row remains Yes and records the new post-latch age test. I corrected one expectation bullet in that row so prompt post-latch edges say latched and edges over the 500 ms age bound say unknown-age. No other validation-plan step or umbrella row changed.

### Pre-repair mandatory checks and coverage for step 2 minimize_loop (exchange 1) (round 2)

Requestor evidence reports 51 Go PASS lines, clean gofmt and go vet, a passing targeted Minimize run, companion script exit 0, npm test 8 pass, six expected completion-grep results, and the pure-model coverage gate passing. ghog day exited 9 as the documented non-pytest project default; the named companion and npm validations passed. I assessed the tests and coverage statically and did not rerun requestor-side validation commands.

### Resolved validation set and sources for step 2 minimize_loop (exchange 1) (round 2)

The request embeds 11 resolved commands: project-default ghog day; companion test script; npm test; Go format/vet; six Step 2 completion greps; and the pure-model coverage gate. They match the current plan's resolved set and round 1. No command was added or removed.

### Resolver drift and direction for step 2 minimize_loop (exchange 1) (round 2)

None. The embedded resolved validation set matches the current project default and plan commands.

### Repository state around validation for step 2 minimize_loop (exchange 1) (round 2)

Request-time and pre-assessment index tree: 5b3def582558e149983cafc33698f66ba8357dba. After the single attributable validation-row wording correction, index tree: 6eeaabc8ce68987844dee66c5d7b92d7a707f19c. Validation-state comparison flags only that staged review-metadata path and the resulting index tree; these are the intended reviewer edit, not a validation side effect. No ignored or untracked validation artifacts appeared in the compared six-path set. No umbrella applies; its digest remained inapplicable. The unstaged review transcript is protocol state.

### Repair inventory for step 2 minimize_loop (exchange 1) (round 2)

Repairs made:

- Review metadata only: staged a wording correction in docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md to state the plan's unknown-age-before-latched precedence. The pre-repair blob was recorded, and the reviewer patch was proven attributable before staging. No substantive implementation repair was made.

Paths staged:

- Six staged paths: companion/main_windows.go; companion/minimize_hook_windows.go; companion/minimize_hook_windows_test.go; companion/minimize_windows.go; companion/minimize_windows_test.go; docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md. All belong to Step 2 implementation or its exact validation row. No pre-existing unstaged writer work was staged.

### Commit plan assessment for step 2 minimize_loop (exchange 1) (round 2)

Independent commit-plan-check returned state valid, ready true, zero diagnostics, and exactly the six staged paths. Group 1 is fix(minimize_loop): drive minimize from IsIconic for the five companion paths. Group 2 is docs(minimize_loop): record step 2 validation for the validation plan. Membership, order, scope, and conventional subjects are accurate.

### Findings and boundaries for step 2 minimize_loop (exchange 1) (round 2)

Unresolved findings: None.

Boundary-crossing work: None.

### Writer instructions for step 2 minimize_loop (exchange 1) (round 2)

No Step 2 changes required. The commit-ready recommendation is advisory; proceed through the human commit gate if selected. The reviewer did not commit.

### Decision rationale for step 2 minimize_loop (exchange 1) (round 2)

Recommend commit-ready. Exact identity and request-time tree match; Step 2 is complete; requestor validation and static coverage assessment support readiness; the sole reviewer edit is attributable Step 2 review metadata; no current or carried findings remain; and the independent commit-plan check is ready. No substantive repair occurred this round.

### Final reviewer decision for step 2 minimize_loop (exchange 1) (round 2)

Decision: commit-ready (advisory). The evidence floor is complete, but this recommendation does not authorize a commit; authority remains at the durable human gate.

<!-- review-entry-id: answer-step-2-round-2 -->

## Round 2 by human - Step 2 - human-confirmation

- Recorded: 2026-09-25T19:27:16+02:00
- Exchange: code/code/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Implementation step: 2
- Outcome: human-confirmation

Human choice: Commit
Outcome: continue-owning-workflow

<!-- review-entry-id: human-confirmation-round-2 -->

## Round 1 by requestor - Step 3

- Recorded: 2026-09-25T19:54:59+02:00
- Exchange: code/code/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: unrecorded
- Implementation step: 3
- Outcome: request

### Review identity for step 3 minimize_loop (round 1)

Umbrella draft: none
Implementation plan: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
Implementation step: 3
Review round: 1

### Code review evidence for step 3 minimize_loop (round 1)

request_index_tree: f131513f8b39aec1f7ab550532c1b1a54cb6cd54
resolved_validation_set:

- ghog day (sources: project)
- powershell -NoProfile -ExecutionPolicy Bypass -File scripts\test-companion.ps1 (sources: plan)
- npm test (sources: plan)
- Push-Location companion; gofmt -l .; go vet ./...; Pop-Location (sources: plan)
- git grep -nE 'minimizeCap(Count|WindowMS|QuietMS) +=' -- companion (sources: plan)
- git grep -nE 'interception (suspended|resumed)' -- companion (sources: plan)
- pure-model coverage gate over a.cover.minimize.out (plan command templates) (sources: plan)

commit_plan_result:

```text
state: valid
ready: true
group 1: fix(minimize_loop): cap interceptions per window
group 1 path: companion/minimize_windows.go
group 1 path: companion/minimize_windows_test.go
group 1 path: companion/minimize_cap_windows_test.go
group 1 path: companion/minimize_fuzz_windows_test.go
group 2: fix(minimize_loop): log cap trip and resume
group 2 path: companion/minimize_hook_windows.go
group 2 path: companion/minimize_hook_windows_test.go
group 3: docs(minimize_loop): record step 3 validation
group 3 path: docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md
staged path: companion/minimize_cap_windows_test.go
staged path: companion/minimize_fuzz_windows_test.go
staged path: companion/minimize_hook_windows.go
staged path: companion/minimize_hook_windows_test.go
staged path: companion/minimize_windows.go
staged path: companion/minimize_windows_test.go
staged path: docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md
```

### Requestor assessment for step 3 minimize_loop (round 1)

Step 3 is fully implemented; the implementation check wrote the exact Yes
sentence into the validation plan.

- **Tests**: `go test -v` gives 57 top-level `--- PASS` lines (51 after
  Step 2, plus 6 new: five `TestMinimizeCap...` tests and
  `TestMinimizeControllerLogsCapTripAndResume`), and
  `FuzzMinimizeModelObservations` runs its 11 `f.Add` seeds (9 before, plus
  2 cap seeds) under plain `go test`. As extra evidence, a 30 s
  `go test -fuzz` session ran on a scratch copy of `companion` outside the
  repository (232444 executions, no failing input), so no
  `testdata/fuzz` file was written. A mutation that disables the
  `closed(now)` case fails five of the new tests and the fuzz seeds.
- **Static checks**: `gofmt -l` prints nothing and `go vet ./...` exits 0.
  The first completion grep finds the three constants
  (`minimize_windows.go:109-111`); the second finds both log formats in
  `minimize_hook_windows.go` (lines 74 and 79) and their test assertions.
- **Gate**: `ghog day` ends at `exit=9` ("not a pytest project"),
  `scripts\test-companion.ps1` prints `ok  workspace-halo/companion` and exits
  0, and `npm test` exits 0 (8 pass).
- **Coverage**: the pure-model coverage gate over `a.cover.minimize.out`
  passes (60 blocks of `minimize_windows.go`, none at zero; no
  `minimize_cap_windows.go` exists because no model split was needed). The
  controller's `observe`, with its two new log branches, stays at 100%; the
  five Win32 adapter functions stay at 0%, evidence-only under plan Q08.
- **Architecture**: `minimizeCap` sits in the pure model file, which still
  has no import and no `proc*` reference. The controller only reads the
  `capTripped` and `capResumed` flags and logs. The three Step 2 carry-overs
  (controller and adapter in one file, global `activeApp`, 1846-line
  `main_windows.go`) are untouched.
- **Performance**: the cap is a fixed two-slot array plus three scalars; each
  observation adds one `maybeResume` check and, on a shown-to-iconic edge, at
  most one `closed` check and one record. No new Win32 call, no allocation.
  The two new log lines are written only on a trip or a resume.
- **Feature integrity**: one minimize, or minimize-restore-minimize within
  2 s, still gets both interceptions and halos; every Step 2 line is still
  written.
- **Line budgets and split**: adding the cap tests took
  `minimize_windows_test.go` to 622 lines, past the 550 threshold, so the
  plan's split guidance was applied: the fuzz target moved to new
  `minimize_fuzz_windows_test.go` (170 lines) and the cap tests to new
  `minimize_cap_windows_test.go` (165 lines); `minimize_windows_test.go` is
  back to 307. `minimize_windows.go` is 361 (advisory about 330),
  `minimize_hook_windows.go` 265 (about 240), `minimize_hook_windows_test.go`
  323 (about 260); all below 550. `main_windows.go` is unchanged at 1846.
- **Interpretation choices for the reviewer to judge**:
  - The plan calls the `intercepts` array a ring; it is kept oldest first and
    shifted by one on each record, the same fixed two-slot window, so
    `closed` reads the oldest tick at index 0.
  - The 2000 ms window is exclusive: a third edge exactly 2000 ms after the
    first is intercepted (`now - oldest < minimizeCapWindowMS` closes the
    cap). `TestMinimizeCapAllowsTwoInterceptionsWithinTheWindow` pins it, and
    the fuzz trace checks "no three intercepts within 2000 ms" with the same
    strict bound.
  - `shownToIconic` records attempts after its Unsettled case, not before:
    Unsettled only follows an interception, which needs an open cap, so it
    never coexists with a suspension. The doc comment states this, and the
    fuzz trace checks the `lastAttemptAt` property on every observation.
  - A closed cap trips only when an edge reaches the cap check; a
    `unknown-age` edge while two intercepts are in the window does not trip
    it (design: the cap applies to an edge that "would be intercepted"),
    tested in `TestMinimizeCapTripsOnTheThirdEdgeWithinTwoSeconds`.

### Implementation report for step 3 minimize_loop (round 1)

Step 3 changes on top of the Step 2 observation model:

- `companion/minimize_windows.go`: adds `minimizeCapCount = 2`,
  `minimizeCapWindowMS = 2000`, `minimizeCapQuietMS = 5000` and
  `minimizeReasonCap = "cap"`. Adds the value type `minimizeCap`
  (`intercepts [minimizeCapCount]uint64`, `filled`, `suspended`,
  `lastAttemptAt`) with `closed(now)`, `recordIntercept(now)`,
  `recordAttempt(now) (minimizeCap, tripped bool)` and
  `maybeResume(now) (minimizeCap, resumed bool)`, as field `cap` of
  `minimizeModel`. `minimizeDecision` gains `capTripped` and `capResumed`.
  `observe` calls `maybeResume` right after the Unsettled resolution, on every
  observation. `shownToIconic(decision, now)` keeps its Unsettled case first,
  then records every external edge as an attempt while suspended, then
  cancels a Priming replay (never counted), then in Shown skips with
  `unknown-age`, `latched` or `cap` (a first closed cap trips here), or
  intercepts and records the tick at the decision. The file header explains
  the cap.
- `companion/minimize_hook_windows.go`: `minimizeController.observe` logs
  `minimize interception resumed after 5000ms quiet` before the edge line on
  `capResumed`, and `minimize interception suspended: 2 intercepts in 2000ms`
  after it on `capTripped`, both formatted from the constants.
- `companion/minimize_cap_windows_test.go` (new): helpers
  `minimizeEdgeAfter` and `trippedMinimizeModel`, and the five planned cap
  tests: two interceptions pass (and the exclusive window), trip once (and a
  late third edge keeps `unknown-age` without tripping), a 20-edge stream on a
  tripped and on a latched model keeps the cap closed with `lastAttemptAt` on
  every edge, the T + 4999 / T + 5000 / T + 9998 / T + 9999 resume
  boundaries, and Priming cancellations not counted.
- `companion/minimize_fuzz_windows_test.go` (new): the Step 2
  `FuzzMinimizeModelObservations`, its decoder and checks, moved verbatim out
  of `minimize_windows_test.go`, plus two cap seeds and
  `minimizeFuzzCapTrace`, which checks the four planned cap properties on
  every observation.
- `companion/minimize_windows_test.go`: loses the fuzz target (moved); keeps
  the Step 2 model tests and `primedMinimizeModel`, which the cap tests reuse;
  header updated for the split.
- `companion/minimize_hook_windows_test.go`: adds the `externalRestore`
  fixture helper and `TestMinimizeControllerLogsCapTripAndResume`.
- `docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md`: fills the Step 3
  sections with the Yes verdict and evidence, and updates the opening theme
  sentence (the document-level line stays `No`, Step 4 is pending).

### Change summary for step 3 minimize_loop (round 1)

Staged paths:

- `A companion/minimize_cap_windows_test.go`
- `A companion/minimize_fuzz_windows_test.go`
- `M companion/minimize_hook_windows.go`
- `M companion/minimize_hook_windows_test.go`
- `M companion/minimize_windows.go`
- `M companion/minimize_windows_test.go`
- `M docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md`

Root `a.commit` groups (formatted by `wac`, `commit-plan-check` ready):

1. `fix(minimize_loop): cap interceptions per window`: `minimize_windows.go`
   and the three model test files. It builds alone, because the Step 2
   controller does not read the new decision flags yet.
2. `fix(minimize_loop): log cap trip and resume`: `minimize_hook_windows.go`
   and `minimize_hook_windows_test.go`.
3. `docs(minimize_loop): record step 3 validation`: the validation plan.

### Writer response for step 3 minimize_loop (round 1)

Writer response:

First round of the Step 3 review: no earlier feedback to accept or dispute.
The interpretation choices (shifted two-slot window instead of a ring index,
exclusive 2000 ms window, attempt recording after the Unsettled case, trip
only at the cap check) and the test file split are listed in the assessment
for the reviewer to judge.

### Reviewer focus for step 3 minimize_loop (round 1)

Check the exact plan step, staged implementation, test evidence, repaired path inventory, and a.commit accuracy.

<!-- review-entry-id: request-step-3-round-1 -->

## Round 1 by reviewer - Step 3

- Recorded: 2026-09-25T20:08:28+02:00
- Exchange: code/code/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Implementation step: 3
- Outcome: answer

### Assessed index identity for step 3 minimize_loop (exchange 1) (round 1)

Baseline index tree: f131513f8b39aec1f7ab550532c1b1a54cb6cd54

Assessed index tree: f131513f8b39aec1f7ab550532c1b1a54cb6cd54

### Implementation check for step 3 minimize_loop (exchange 1) (round 1)

Result:

Yes. Step 3 has been fully implemented. The staged pure minimizeCap limits a window to two interceptions in a rolling 2000 ms window, trips on the next eligible prompt edge, records every shown-to-iconic attempt during suspension, and clears history after 5000 ms without such an edge. The observation model retains the specified unknown-age, latched, cap precedence; Priming cancellations do not count. The controller emits the trip and resume lines. Five cap tests, the extended fuzz target, and the controller log test cover the planned behavior. The fixed two-slot oldest-first array is equivalent to the planned ring for a two-entry window; the exclusive 2000 ms boundary is a consistent half-open time window. The test split follows the plan and all touched files remain below the line ceiling. No Step 3 implementation gap remains.

Validation plan effects:

The exact Step 3 validation rows already begin with Yes. Step 3 has been fully implemented. They accurately record the cap behavior, test split, evidence, and the Step 4 pending state. No validation-plan edit was needed; no umbrella applies.

### Pre-repair mandatory checks and coverage for step 3 minimize_loop (exchange 1) (round 1)

Requestor evidence reports 57 passing Go tests plus 11 fuzz seeds under ordinary go test; passing companion and npm suites; clean gofmt and go vet; both expected completion greps; and the pure-model coverage gate with no zero-count block. The documented ghog day exit 9 is the project default for a non-pytest repository. The model tests exercise the new cap methods and decision paths; the five Win32 adapter functions are outside the pure-model gate under plan Q08. I assessed this evidence and test coverage statically and did not rerun requestor-side validation commands.

### Resolved validation set and sources for step 3 minimize_loop (exchange 1) (round 1)

The request embeds seven resolved commands: project-default ghog day, companion test script, npm test, Go format/vet, two Step 3 completion greps, and the pure-model coverage gate. The commands and project or plan sources match the current Step 3 plan checklist.

### Resolver drift and direction for step 3 minimize_loop (exchange 1) (round 1)

None. The embedded command set matches the current project default and exact Step 3 plan.

### Repository state around validation for step 3 minimize_loop (exchange 1) (round 1)

The request-time, baseline, and assessed index tree all equal f131513f8b39aec1f7ab550532c1b1a54cb6cd54. The ordered seven-path validation-state comparison is acceptable with no tracked, untracked, or ignored changes. The umbrella digest remained inapplicable. The unstaged review transcript is protocol state and was not staged.

### Repair inventory for step 3 minimize_loop (exchange 1) (round 1)

Repairs made:

- None. No implementation, test, validation-plan, or commit-plan file was edited by the reviewer.

Paths staged:

- Seven staged Step 3 paths: companion/minimize_cap_windows_test.go; companion/minimize_fuzz_windows_test.go; companion/minimize_hook_windows.go; companion/minimize_hook_windows_test.go; companion/minimize_windows.go; companion/minimize_windows_test.go; docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md. No pre-existing unstaged work was staged.

### Commit plan assessment for step 3 minimize_loop (exchange 1) (round 1)

The independent commit-plan-check returned state valid, ready true, and zero diagnostics. Its three ordered groups are the pure cap model and tests, controller cap logging and test, then Step 3 validation documentation. All seven staged paths are covered once, and grouping, order, scope, and conventional subjects remain accurate.

### Findings and boundaries for step 3 minimize_loop (exchange 1) (round 1)

Unresolved findings: None.

Boundary-crossing work: None.

### Writer instructions for step 3 minimize_loop (exchange 1) (round 1)

No Step 3 changes required. The commit-ready recommendation is advisory; proceed through the human commit gate if selected. The reviewer did not commit.

### Decision rationale for step 3 minimize_loop (exchange 1) (round 1)

Recommend commit-ready. The live exchange and request-time index match; Step 3 is complete; requestor validation and static coverage assessment support readiness; all staged paths belong to the step and remained unchanged; no unresolved findings or boundary changes exist; and the independent commit-plan check is ready. No substantive repair occurred this round.

### Final reviewer decision for step 3 minimize_loop (exchange 1) (round 1)

Decision: commit-ready (advisory). The evidence floor is complete, but this recommendation does not authorize a commit; authority remains at the durable human gate.

<!-- review-entry-id: answer-step-3-round-1 -->

## Round 1 by human - Step 3 - human-confirmation

- Recorded: 2026-09-25T21:41:19+02:00
- Exchange: code/code/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Implementation step: 3
- Outcome: human-confirmation

Human choice: Commit
Outcome: continue-owning-workflow

<!-- review-entry-id: human-confirmation-round-1 -->

## Round 1 by requestor - Step 4

- Recorded: 2026-09-26T00:41:56+02:00
- Exchange: code/code/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: unrecorded
- Implementation step: 4
- Outcome: request

### Review identity for step 4 minimize_loop (round 1)

Umbrella draft: none
Implementation plan: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
Implementation step: 4
Review round: 1

### Code review evidence for step 4 minimize_loop (round 1)

request_index_tree: 21a01eb02faf5b31a4c57c361be9c2d39bb6d14d
resolved_validation_set:

- ghog day (sources: project)
- powershell -NoProfile -ExecutionPolicy Bypass -File scripts\test-companion.ps1 (sources: plan)
- npm test (sources: plan)
- Push-Location companion; gofmt -l .; go vet ./...; Pop-Location (sources: plan)
- git grep -nE 'Lateness bound|Settle timeout|Interception cap' -- wiki/reference/display-triggers.md (sources: plan)
- git grep -nE '^## 0\.0\.24' -- CHANGELOG.md (sources: plan)
- switchnode 22 then npx --yes markdownlint-cli2 on the four updated Markdown files (MD013 on table rows only) (sources: plan)
- build.bat (exit 0, OK: Packaged) (sources: plan)

commit_plan_result:

```text
state: valid
ready: true
group 1: test(minimize_loop): replay acceptance cases
group 1 path: companion/minimize_scenario_windows_test.go
group 1 path: companion/minimize_acceptance_windows_test.go
group 1 path: companion/minimize_hook_windows_test.go
group 2: docs(minimize_loop): document observed-edge minimize
group 2 path: wiki/reference/display-triggers.md
group 2 path: wiki/explanation/how-the-overlay-stays-inside-its-window.md
group 2 path: wiki/reference/logs-and-processes.md
group 3: docs(minimize_loop): add 0.0.24 changelog entry
group 3 path: CHANGELOG.md
group 4: docs(minimize_loop): record step 4 check
group 4 path: docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md
staged path: CHANGELOG.md
staged path: companion/minimize_acceptance_windows_test.go
staged path: companion/minimize_hook_windows_test.go
staged path: companion/minimize_scenario_windows_test.go
staged path: docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md
staged path: wiki/explanation/how-the-overlay-stays-inside-its-window.md
staged path: wiki/reference/display-triggers.md
staged path: wiki/reference/logs-and-processes.md
```

### Requestor assessment for step 4 minimize_loop (round 1)

Step 4 is not fully implemented: the implementation check wrote the exact
`No. Step 4 has NOT been fully implemented.` sentence into the validation
plan. The automated part is complete. The missing part is the manual
three-to-one unplug (plan rollout steps 3 to 6 and the last completion
criterion), which needs a person at the machine. The human asked for this
review of the automated part now and will run the unplug later. That later
evidence will be recorded in the validation plan and checked again.

- **Tests**: `go test -v` gives 61 top-level `--- PASS` lines (57 after
  Step 3, plus 4 new: `TestMinimizeAcceptanceCases`,
  `TestMinimizeAcceptanceClickDuringPrimingStillReplays`,
  `TestMinimizeAcceptanceRestoreAfterReplayStaysRestored` and
  `TestMinimizeAcceptanceRecordedUnplugTimeline`).
  `TestMinimizeAcceptanceCases` has 17 passing sub-tests, and the 11 fuzz
  seeds still pass.
- **Static checks**: `gofmt -l` prints nothing and `go vet ./...` exits 0.
  The first completion grep finds the three constants rows
  (`display-triggers.md:84-86`); the second finds `CHANGELOG.md:5`
  (`## 0.0.24`).
- **Markdown**: markdownlint-cli2 on the four updated files reports 13
  findings, all MD013 on `display-triggers.md` table rows (the existing
  priority table and the timing constants table), as the plan allows. The
  validation plan lints clean.
- **Gate**: `ghog day` ends at `exit=9` ("not a pytest project").
  `scripts\test-companion.ps1` prints `ok  workspace-halo/companion` and exits
  0, and `npm test` exits 0 (8 pass). `build.bat` exits 0 and prints
  `OK: Packaged workspace-halo-0.0.23-3e4d05f-dirty-win32-x64.vsix`, built
  from the uncommitted tree.
- **Coverage**: no production file changed. The pure-model coverage gate
  still passes, and the controller stays at 100%. The five Win32 adapter
  functions stay at 0%, evidence-only under plan Q08; the manual unplug is
  what exercises them.
- **Architecture**: only test files, the wiki and the changelog change. The
  runner reaches the controller through `newMinimizeController`, `observe`
  and `onEvent` and the `minimizeWindow` seam. It reads `model` fields only in
  assertions. The three Step 2 carry-overs (controller and adapter in one
  file, global `activeApp`, 1846-line `main_windows.go`) are untouched.
- **Performance**: no host-path change. The runner steps one millisecond at a
  time and scans a handful of pending events per millisecond, linear in
  simulated time; the longest scenario (90 s latch) runs about 91000
  iterations, and the Go suite still runs in seconds.
- **Feature integrity**: the host binary is unchanged since Step 3. The wiki
  no longer describes the old event-driven interception.
- **Line budgets and split**: putting the runner and the scenarios in
  `minimize_hook_windows_test.go` (323 lines) would have taken it far past
  650. As the plan's split guidance asks, they went to new files. The
  acceptance file alone reached 620 lines (the 550-to-650 band), so the
  runner was split out by responsibility:
  `minimize_scenario_windows_test.go` 226, `minimize_acceptance_windows_test.go`
  408, `minimize_hook_windows_test.go` 325. Wiki and changelog files are 88,
  87, 69 and 224 lines (advisory 77, 75, 54, 219).
- **Interpretation choices for the reviewer to judge**:
  - Two new test files instead of one: the plan's split target was
    `minimize_acceptance_windows_test.go`; the runner went to
    `minimize_scenario_windows_test.go` to keep the acceptance file below 550.
  - The design row "events delivered in any order, or 3 s late" has two
    sub-tests, one in order and one reordered, so there are 17 sub-tests for
    16 automated rows. The manual unplug row is the rollout check.
  - Normal path "20 ms after a shown reading": a prompt `MinimizeStart`
    (lag 0) triggers the observation at 1020, 20 ms after the seed reading,
    so the decisive line is `age<=20ms action=intercept`.
  - Late animation "100 ms after the priming restore": the replay is due
    75 ms after the restore, so a 99 ms host-thread stall after the restore
    keeps the tick from replaying first. The re-minimize at restore + 100 then
    finds Priming and logs `age<=100ms action=cancel-replay`. The "within
    250 ms" row uses a re-minimize at restore + 60 with no stall.
  - A click during Priming is a step that changes nothing the host observes,
    which is the design's claim; the test checks the replay still minimizes
    the window.
  - The unplug timeline defers w2's `SW_SHOWNOACTIVATE`, so its one
    interception takes the single `SW_RESTORE` fallback, matching the plan's
    allowance.
  - The changelog entry sits under `## 0.0.24` below `## Unreleased`, as the
    0.0.22 and 0.0.23 entries do, while `package.json` is still 0.0.23; the
    release step bumps the version.

### Implementation report for step 4 minimize_loop (round 1)

Step 4 changes on top of the Step 3 cap; no production code changes:

- `companion/minimize_scenario_windows_test.go` (new): the scenario runner.
  `minimizeScenario` holds a shared fake clock, one host (controller and log
  buffer) per window, the pending WinEvents and a stall deadline. `run` walks
  one millisecond at a time. At each tick it applies the scripted steps
  (which must be in tick order), then, unless stalled, delivers the due
  events (`deliverDue`, in generation order, including events caused by those
  deliveries), then runs the 25 ms tick on every controller. It can be called
  again to continue a scenario. `minimizeScenarioWindow` embeds
  `fakeMinimizeWindow`, overrides `showWindow` to record restore command
  ticks, and emits a WinEvent on every minimized-state change. The n-th event
  of window i is delivered `lags[(n+i) % len(lags)]` ms later
  (`minimizeEventLost` or no lag list drops it). The step kinds are minimize,
  restore, click, defer an own command, apply the deferred call, stall, and
  restart the host (which clears the deferral). `requireOrderedLog` checks
  lines in order.
- `companion/minimize_acceptance_windows_test.go` (new):
  `minimizeAcceptanceCase` and `minimizeAcceptanceCases` (17 rows),
  checked by `TestMinimizeAcceptanceCases`. Each row asserts the restore
  attempts, the `minimize intercepted` count, the `minimize edge` count, the
  final phase on the observed state, the ordered decisive lines, the
  forbidden lines, and an optional extra check. The 20 s stream row continues
  the scenario to prove no resume at 27599, the resume at 27600, then a new
  intercept. Also `TestMinimizeAcceptanceClickDuringPrimingStillReplays`,
  `TestMinimizeAcceptanceRestoreAfterReplayStaysRestored`, and
  `TestMinimizeAcceptanceRecordedUnplugTimeline`. The timeline test runs four
  hosts: w4, w1 and w2 are minimized at 10000, 12700 and 14000, w3 stays
  shown, events arrive 3 to 6 s late and reordered, and w2 uses the fallback.
  It asserts one restore attempt and one replay per minimized window, no
  restore after the first late event, Minimized with the halo, and no
  command on w3.
- `companion/minimize_hook_windows_test.go`: header sentence pointing to the
  runner that reuses `fakeMinimizeWindow`.
- `wiki/reference/display-triggers.md`: `minimized` row names a pending
  replay and an unsettled own call; new "Minimize interception rules"
  section; four timing constants rows (`Lateness bound`, `Settle timeout`,
  `Interception cap`, `Cap quiet period`).
- `wiki/explanation/how-the-overlay-stays-inside-its-window.md`: the replay
  starts from an observed minimize the host did not cause; new section
  "Interception follows the observed window, not the events" covering
  events as observation triggers, own-call absorption, the Priming
  cancellation, and the four let-through cases.
- `wiki/reference/logs-and-processes.md`: list of the minimize decision
  lines next to the three kept interception lines.
- `CHANGELOG.md`: `## 0.0.24` section with three bullets.
- `docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md`: Step 4 sections
  filled with the `No` verdict, evidence, and a "Missing work for Step 4"
  list for the manual unplug; the opening theme sentence updated (the
  document-level line stays `No`).

### Change summary for step 4 minimize_loop (round 1)

Staged paths:

- `M CHANGELOG.md`
- `A companion/minimize_acceptance_windows_test.go`
- `M companion/minimize_hook_windows_test.go`
- `A companion/minimize_scenario_windows_test.go`
- `M docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md`
- `M wiki/explanation/how-the-overlay-stays-inside-its-window.md`
- `M wiki/reference/display-triggers.md`
- `M wiki/reference/logs-and-processes.md`

Root `a.commit` groups (formatted by `wac`, `commit-plan-check` ready):

1. `test(minimize_loop): replay acceptance cases`: the runner, the
   acceptance tests, and the hook test header.
2. `docs(minimize_loop): document observed-edge minimize`: the three wiki
   pages.
3. `docs(minimize_loop): add 0.0.24 changelog entry`: `CHANGELOG.md`.
4. `docs(minimize_loop): record step 4 check`: the validation plan. It
   records a `No` verdict, so it uses "check" rather than the `record step 4
   validation` marker, which is reserved for a plan newly marking a step
   `Yes`.

### Writer response for step 4 minimize_loop (round 1)

Writer response:

First round of the Step 4 review: no earlier feedback to accept or dispute.
The human asked for this review before the manual unplug, so the review
covers the automated part (acceptance scenarios, wiki, changelog, validation
record). The interpretation choices (two new test files, the split design
row, the event-triggered normal path, the stall for the late animation, the
no-op click, the changelog placement) are listed in the assessment for the
reviewer to judge.

### Reviewer focus for step 4 minimize_loop (round 1)

Check the exact plan step, staged implementation, test evidence, repaired path inventory, and a.commit accuracy.

<!-- review-entry-id: request-step-4-round-1 -->

## Round 1 by human - Step 4 - escalation

- Recorded: 2026-09-26T08:36:54+02:00
- Exchange: code/code/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: unrecorded
- Implementation step: 4
- Outcome: escalation

wait timed out while answer was absent

<!-- review-entry-id: escalation-round-1 -->

## Round 1 by human - Step 4 - human-reclaim

- Recorded: 2026-09-26T08:47:28+02:00
- Exchange: code/code/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: unrecorded
- Implementation step: 4
- Outcome: human-reclaim

### Forced resume decision for step 4 minimize_loop (round 1)

The requestor's `wait-answer` timed out while no answer was published, and the
exchange was recorded as escalated. The human decided this was a stopped
handoff rather than a failure: no reviewer session was running when the
request was published, and a reviewer is now waiting for this resolution to
start the review. The round 1 request, its evidence and the staged work are
unchanged, so round 1 resumes with the same request.

<!-- review-entry-id: human-reclaim-round-1 -->

## Round 1 by reviewer - Step 4

- Recorded: 2026-09-26T09:00:18+02:00
- Exchange: code/code/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Implementation step: 4
- Outcome: answer

### Assessed index identity for step 4 minimize_loop (exchange 1) (round 1)

Baseline index tree: 21a01eb02faf5b31a4c57c361be9c2d39bb6d14d

Assessed index tree: 21a01eb02faf5b31a4c57c361be9c2d39bb6d14d

### Implementation check for step 4 minimize_loop (exchange 1) (round 1)

Result:

No. Step 4 has NOT been fully implemented.

The staged runner covers the automated design cases and the recorded four-window simulation. The documentation and changelog match the observed-edge model. The plan's manual three-to-one monitor unplug, per-window timestamped logs, and per-external-edge verdict table are absent. The simulation does not replace that completion criterion.

Validation plan effects:

The existing Step 4 validation rows already use the exact No sentence and list rollout steps 3 through 6 as missing work. The document-level status remains No. I made no validation-plan edit and no substantive repair.

### Pre-repair mandatory checks and coverage for step 4 minimize_loop (exchange 1) (round 1)

The requestor reports 61 passing top-level Go tests, 17 acceptance subtests, passing Go format and vet, 8 passing TypeScript tests, and a successful VSIX package build from the dirty tree. I reviewed the staged tests and the plan statically; I did not rerun the requestor's resolved validation set or measure coverage. The retained a.ghog.log shows ghog check skipped because check.bat is absent and ghog affected ended exit 9 because this is not a pytest project. The pure-model gate covers minimize_windows.go; the Win32 adapter is outside it and remains evidence-only under Q08. The manual host exercise is outstanding.

### Resolved validation set and sources for step 4 minimize_loop (exchange 1) (round 1)

The request contains ghog day (project default) plus these plan commands: scripts\test-companion.ps1, npm test, Go format and vet, the two documentation greps, markdownlint-cli2 on the four changed Markdown files, and build.bat. The project has no versioned .review-validation declaration, so the current resolver retains ghog day as mandatory. The plan and request command lists still agree.

### Resolver drift and direction for step 4 minimize_loop (exchange 1) (round 1)

No resolver drift: the request's default ghog day and seven plan additions match the current project declaration and plan. The default itself is unsuitable for this non-pytest repository: ghog day ended exit 9. This is a validation-floor finding, not a command-set drift.

### Repository state around validation for step 4 minimize_loop (exchange 1) (round 1)

Baseline and assessed index tree are 21a01eb02faf5b31a4c57c361be9c2d39bb6d14d. Umbrella digest is not applicable and did not change. Validation-state comparison is acceptable with no tracked, untracked, or ignored path changes. No reviewer patch was staged.

### Repair inventory for step 4 minimize_loop (exchange 1) (round 1)

Repairs made: None.

Paths staged: None.

### Commit plan assessment for step 4 minimize_loop (exchange 1) (round 1)

Independent commit-plan-check returned state valid, ready true, with no diagnostics. Its four ordered groups cover all eight staged paths: acceptance tests, three wiki pages, changelog, then Step 4 validation plan. The subjects and membership remain accurate for the staged work. This mechanical pass does not satisfy the incomplete implementation and validation floor.

### Findings and boundaries for step 4 minimize_loop (exchange 1) (round 1)

Unresolved findings:

- Step 4's manual three-to-one unplug has not been run. The required per-window timestamped log excerpts and per-external-edge verdict table are absent, so the plan's last completion criterion cannot be assessed.
- The request keeps ghog day as a mandatory project default, but it ended exit 9 because the repository has no pytest suite. The versioned .review-validation declaration is absent, so the mandatory validation floor has no passing project-default command.

Boundary-crossing work:

- The plan orders a committed Step 4 VSIX before the manual unplug, while the review policy requires complete Step 4 evidence before a commit-ready recommendation. Resolve that sequencing in the plan or complete the manual check from the available build and record how its evidence satisfies the criterion.
- Declare the repository's actual mandatory validation commands in a versioned .review-validation file or resolve the project-default gate through the owning workflow; this project-level policy change exceeds the staged Step 4 scope.

### Writer instructions for step 4 minimize_loop (exchange 1) (round 1)

Complete the manual unplug and record the prescribed per-window logs and verdict table, or obtain an explicit plan change that places that check after the commit gate. Resolve the non-pytest project validation default with a versioned declaration of the real gates, then run the resulting mandatory commands and publish a new exact review request. Keep the current staged test and documentation groups unless their content changes.

### Decision rationale for step 4 minimize_loop (exchange 1) (round 1)

Changes requested. Identity and staged attribution pass; the staged tests and docs show no new static defect; a.commit passes mechanically. Implementation completeness fails on the missing manual criterion, and validation readiness fails on the mandatory ghog day exit 9. I made no repair. The six-part commit-ready floor is incomplete.

### Final reviewer decision for step 4 minimize_loop (exchange 1) (round 1)

Decision: changes-requested. The writer must address the concrete instructions and publish another review round. This advisory answer does not authorize a commit.

<!-- review-entry-id: answer-step-4-round-1 -->

## Round 2 by requestor - Step 4

- Recorded: 2026-09-26T09:14:34+02:00
- Exchange: code/code/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Implementation step: 4
- Outcome: request

### Review identity for step 4 minimize_loop (round 2)

Umbrella draft: none
Implementation plan: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
Implementation step: 4
Review round: 2

### Code review evidence for step 4 minimize_loop (round 2)

request_index_tree: 4b38a048f920b8b8cb6ade92f78e700aea338c6b
resolved_validation_set:

- powershell -NoProfile -ExecutionPolicy Bypass -File scripts\test-companion.ps1 (sources: project)
- npm test (sources: project)
- ghog day (plan gate loop; expected exit=9, not a pytest project) (sources: plan)
- Push-Location companion; gofmt -l .; go vet ./...; Pop-Location (sources: plan)
- git grep -nE 'Lateness bound|Settle timeout|Interception cap' -- wiki/reference/display-triggers.md (sources: plan)
- git grep -nE '^## 0\.0\.24' -- CHANGELOG.md (sources: plan)
- switchnode 22 then npx --yes markdownlint-cli2 on the four updated Markdown files (MD013 on table rows only) (sources: plan)
- build.bat (exit 0, OK: Packaged) (sources: plan)

commit_plan_result:

```text
state: valid
ready: true
group 1: chore(review): declare the code-review floor
group 1 path: .review-validation
group 2: test(minimize_loop): replay acceptance cases
group 2 path: companion/minimize_scenario_windows_test.go
group 2 path: companion/minimize_acceptance_windows_test.go
group 2 path: companion/minimize_hook_windows_test.go
group 3: docs(minimize_loop): document observed-edge minimize
group 3 path: wiki/reference/display-triggers.md
group 3 path: wiki/explanation/how-the-overlay-stays-inside-its-window.md
group 3 path: wiki/reference/logs-and-processes.md
group 4: docs(minimize_loop): add 0.0.24 changelog entry
group 4 path: CHANGELOG.md
group 5: docs(minimize_loop): move unplug check to step 5
group 5 path: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
group 6: docs(minimize_loop): record step 4 validation
group 6 path: docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md
staged path: .review-validation
staged path: CHANGELOG.md
staged path: companion/minimize_acceptance_windows_test.go
staged path: companion/minimize_hook_windows_test.go
staged path: companion/minimize_scenario_windows_test.go
staged path: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
staged path: docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md
staged path: wiki/explanation/how-the-overlay-stays-inside-its-window.md
staged path: wiki/reference/display-triggers.md
staged path: wiki/reference/logs-and-processes.md
```

### Requestor assessment for step 4 minimize_loop (round 2)

Step 4 is fully implemented against its amended scope: the implementation
check now writes the exact `Yes. Step 4 has been fully implemented.` sentence
into the validation plan. The document-level line stays `No`, because the new
Step 5 (the manual unplug) has not started.

Both round 1 findings are addressed by the human's decisions, not by
bypassing them:

- **Manual unplug sequencing**: the human chose the plan change that places
  the unplug after the commit gate. The plan now has five steps. Step 4 is
  "Acceptance scenarios, documentation and the VSIX build" and ends with the
  build. Its completion criteria no longer include the unplug, and a bullet
  points to Step 5. The new Step 5, "Manual three-to-one unplug check", holds
  the full rollout sequence (commit Steps 1 to 4, commit-named build,
  install, unplug, per-window excerpts, per-action table) and the unplug
  completion criterion. It adds a criterion that the VSIX name carries the
  Step 4 commit with no `dirty` marker. Decision Q12 records the split and
  its rejected alternatives, and Q11 now points the evidence to the Step 5
  section of the validation plan. The validation plan gains a not-started
  Step 5 section.
- **Validation floor**: the human chose a versioned declaration. The new root
  `.review-validation` declares `powershell -NoProfile -ExecutionPolicy
  Bypass -File scripts\test-companion.ps1` and `npm test`. The llm-shared
  loader `load_project_validation_commands` reads exactly those two commands
  from it, and `git check-ignore` confirms the file is not ignored. Both
  commands pass on the staged tree: `ok  workspace-halo/companion` and 8 of 8
  TypeScript tests. Decision Q13 records it. `ghog day` is still run as the
  plan's gate-loop command, whose documented expected result for this
  repository is `exit=9`, and it ends there.

Evidence for the unchanged Step 4 content (tests, wiki, changelog), as in
round 1:

- **Tests**: 61 top-level `--- PASS` lines, 17 acceptance sub-tests, 11 fuzz
  seeds; `gofmt -l` prints nothing and `go vet ./...` exits 0. No Go file
  changed since round 1.
- **Completion greps**: the constants rows (`display-triggers.md:84-86`) and
  `CHANGELOG.md:5`.
- **Markdown**: the four Step 4 files report only MD013 on
  `display-triggers.md` table rows. The validation plan lints clean. The plan
  reports only MD013 on the decision-table rows: the 11 existing rows and the
  two new ones, which follow the same one-row-per-decision format.
- **Build**: `build.bat` exits 0 and prints `OK: Packaged` (a `dirty` build
  of the uncommitted tree; the commit-named build is a Step 5 criterion now).
- **Coverage, architecture, performance, feature integrity**: unchanged from
  round 1. No production file changed, the pure-model gate passes, the
  runner goes through the controller's entry points and the `minimizeWindow`
  seam, it is linear in simulated time, and the host binary is unchanged.
  The three Step 2 carry-overs remain as recorded.
- **Review transcript**: `docs/v0.0.24/review.code.v0.0.24.minimize_loop.md`
  is modified by the exchange itself and left unstaged. As in Step 3, it is
  committed after the review.

### Implementation report for step 4 minimize_loop (round 2)

Round 2 changes on top of the round 1 staged work (the Go test files, the
three wiki pages and the changelog are unchanged):

- `.review-validation` (new, versioned): declares
  `scripts\test-companion.ps1` and `npm test` as the mandatory code-review
  validation commands, with `#` comments explaining why `ghog day` cannot be
  the floor here.
- `docs/v0.0.24/plan.v0.0.24.minimize_loop.md`: the intro, the plan goal and
  its step list now name five steps. Step 4 is renamed "Acceptance
  scenarios, documentation and the VSIX build", and the unplug is dropped
  from its issues, fix intent, expected outcome, rollout and completion
  criteria. A criterion bullet points to Step 5, and the time-gated note no
  longer names the unplug. The split guidance also names
  `minimize_scenario_windows_test.go`. The new "Step 5. Manual three-to-one
  unplug check" has analysis, implementation (rollout sequence, completion
  criteria including a commit-named VSIX) and addendums. Q11 points to the
  Step 5 section, and Q12 (the split) and Q13 (the declared floor) are
  added.
- `docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md`: the opening theme
  sentence names five steps. The Step 4 heading follows the plan rename, its
  analysis reads Yes with the reason for the re-check, and its goal and
  expectations drop the unplug. Its "Missing work for Step 4" section is
  removed, and two bullets record the plan amendment and the declared floor.
  The coverage and feature notes point to Step 5. A not-started Step 5
  section is added. The document-level line stays `No`.

### Change summary for step 4 minimize_loop (round 2)

Staged paths:

- `A .review-validation`
- `M CHANGELOG.md`
- `A companion/minimize_acceptance_windows_test.go`
- `M companion/minimize_hook_windows_test.go`
- `A companion/minimize_scenario_windows_test.go`
- `M docs/v0.0.24/plan.v0.0.24.minimize_loop.md`
- `M docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md`
- `M wiki/explanation/how-the-overlay-stays-inside-its-window.md`
- `M wiki/reference/display-triggers.md`
- `M wiki/reference/logs-and-processes.md`

Unstaged on purpose: `docs/v0.0.24/review.code.v0.0.24.minimize_loop.md`, which
the exchange writes; it is committed after the review, as in Step 3.

Root `a.commit` groups (formatted by `wac`, `commit-plan-check` ready):

1. `chore(review): declare the code-review floor`: `.review-validation`.
2. `test(minimize_loop): replay acceptance cases`: the runner, the
   acceptance tests, and the hook test header.
3. `docs(minimize_loop): document observed-edge minimize`: the three wiki
   pages.
4. `docs(minimize_loop): add 0.0.24 changelog entry`: `CHANGELOG.md`.
5. `docs(minimize_loop): move unplug check to step 5`: the plan.
6. `docs(minimize_loop): record step 4 validation`: the validation plan,
   which now marks Step 4 `Yes`, so it carries the exact `pw` marker and
   comes last.

### Human guidance and writer response for step 4 minimize_loop (round 2)

Human guidance:

#### Forced resume decision for step 4 minimize_loop (round 2)

The requestor's `wait-answer` timed out while no answer was published, and the
exchange was recorded as escalated. The human decided this was a stopped
handoff rather than a failure: no reviewer session was running when the
request was published, and a reviewer is now waiting for this resolution to
start the review. The round 1 request, its evidence and the staged work are
unchanged, so round 1 resumes with the same request.

Writer response:

Both round 1 findings are accepted, with no disagreement. The human made the
two decisions they needed:

- The manual unplug sequencing is resolved by the plan change the round 1
  writer instructions offered: the unplug is placed after the commit gate as
  its own Step 5 (decision Q12), so it runs on the commit-named build and its
  evidence names that commit.
- The validation floor is resolved by the versioned `.review-validation`
  declaration of the real gates (decision Q13), which both pass. That
  project-level change is staged as its own `chore(review)` group, first in
  `a.commit`.

The staged test and documentation groups are unchanged, as the round 1
writer instructions asked. The regrouped `a.commit` adds the floor, plan and
validation groups. The validation group now carries the `record step 4
validation` marker because Step 4 reads `Yes`.

### Reviewer focus for step 4 minimize_loop (round 2)

Check the exact plan step, staged implementation, test evidence, repaired path inventory, and a.commit accuracy.

<!-- review-entry-id: request-step-4-round-2 -->

## Round 2 by reviewer - Step 4

- Recorded: 2026-09-26T09:21:26+02:00
- Exchange: code/code/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Implementation step: 4
- Outcome: answer

### Assessed index identity for step 4 minimize_loop (exchange 1) (round 2)

Baseline index tree: 4b38a048f920b8b8cb6ade92f78e700aea338c6b

Assessed index tree: 4b38a048f920b8b8cb6ade92f78e700aea338c6b

### Implementation check for step 4 minimize_loop (exchange 1) (round 2)

Result:

Yes. Step 4 has been fully implemented under the amended five-step plan. The 17 controller-level acceptance subtests cover the 16 automated design rows (the event-order row has two variants), with focused click and restore tests and a four-window replay of the recorded unplug timeline. The runner drives the production controller entry points through the window seam and asserts commands, observed phases, edges and decisive logs. The three wiki pages and changelog describe the shipped observed-edge behavior, and the requestor reports successful Go and TypeScript suites, format/vet, completion greps, acceptable Markdown findings, the pure-model coverage gate and `build.bat` packaging. The real unplug is the separately defined Step 5 and remains unstarted.

Architecture: no production code changes in this step; the new test runner remains in separate files below the 650-line ceiling and uses the existing fake window seam. The pre-existing Step 2 controller/adapter and global-activeApp concerns and oversized main file remain outside this step. No new architecture defect is identified.

Performance: production hot paths and file IO are unchanged. The test runner advances a fixed number of controllers over scripted milliseconds with a short pending event list; no meaningful performance issue is introduced.

Unit coverage: the configured gate measures the pure model, not the Win32 adapter; the requestor reports it green. The new test-only top-level helpers and type are referenced by the acceptance cases. The unmeasured adapter functions await real-window exercise in Step 5, as plan Q08 specifies. No step-specific unit coverage gap is found.

Feature integrity: no production behavior or diagnostics are changed in this staged step. The documentation and release entry reflect the earlier implementation. The validation plan's first Step 4 sentence is exactly `Yes. Step 4 has been fully implemented.`; its document-level status remains No because Step 5 is pending. No Step 4 missing-work section remains.

Validation plan effects:

The requestor staged the Step 4 validation section with the exact Yes sentence, detailed acceptance and gate evidence, and the amended Step 4 scope. It also added an unstarted Step 5 section and kept the document-level status No. I made no validation-plan edit in this round; the Step 4 rows already match the independently assessed state.

### Pre-repair mandatory checks and coverage for step 4 minimize_loop (exchange 1) (round 2)

Before assessment, the request-time index tree and reviewer baseline both were `4b38a048f920b8b8cb6ade92f78e700aea338c6b`. The validation plan's staged blob was `fd56365135824c28d2014e3317ae9ab73ef3b1d8`. No reviewer repair was authorized or needed, so no pre-repair validation command was run and no repair patch was attributed.

### Resolved validation set and sources for step 4 minimize_loop (exchange 1) (round 2)

The request declares project commands `powershell -NoProfile -ExecutionPolicy Bypass -File scripts\test-companion.ps1` and `npm test`, both supplied by the staged, versioned `.review-validation`; the requestor reports both passing. Plan additions are `ghog day` with expected exit 9 in this non-pytest project, format/vet, the two completion greps, Markdown lint with MD013 table-row allowance, and `build.bat` with `OK: Packaged`; the requestor reports their stated outcomes. The nonzero `ghog day` is an explicitly expected plan-gate observation, not the project's mandatory review floor. I did not rerun the requestor validation set or coverage.

### Resolver drift and direction for step 4 minimize_loop (exchange 1) (round 2)

No drift found. The staged `.review-validation` has exactly the two project commands in the request, and the amended Step 4 plan retains the listed gate, format, grep, lint and build commands. The shared resolver's declaration rule replaces its built-in default with the nonempty versioned project list; the plan additions remain labeled `plan`.

### Repository state around validation for step 4 minimize_loop (exchange 1) (round 2)

The live index tree matched the request-time tree before assessment and remained the same afterward. Umbrella: none, so its digest is not applicable and unchanged. Validation-state comparison over every staged path, the validation plan and the known ghog artifacts returned acceptable with no changed tracked, untracked or ignored path. I changed no implementation file, test, review metadata or commit grouping.

### Repair inventory for step 4 minimize_loop (exchange 1) (round 2)

Repairs made: None.

Paths staged: None.

### Commit plan assessment for step 4 minimize_loop (exchange 1) (round 2)

Independent `commit-plan-check.bat --format json` returned `state=valid`, `ready=true`, no diagnostics, and all ten staged paths in six ordered groups. The groups accurately separate the project validation declaration, three acceptance test files, three wiki pages, changelog, amended plan, and Step 4 validation record. The two new plan decisions and Step 5 scope are correctly in the plan group; no amendment to `a.commit` is needed. This mechanical pass is one part of the readiness floor, not commit authority.

### Findings and boundaries for step 4 minimize_loop (exchange 1) (round 2)

Unresolved findings: None.

Boundary-crossing work: None.

### Human guidance response for step 4 minimize_loop (exchange 1) (round 2)

Human guidance:

Forced resume decision for step 4 minimize_loop (round 2): the requestor's earlier wait-answer timeout was a stopped handoff while no reviewer was running; a reviewer has now resumed the request. The human directed the original round 1 request to resume with unchanged evidence.

Guidance response:

I resumed through the typed reviewer migration, identity, claim, status and wait-request gates, then reviewed the current round 2 request and its new staged tree. The forced-resume guidance authorizes no implementation repair and does not override the evidence or disposition checks.

### Writer instructions for step 4 minimize_loop (exchange 1) (round 2)

No further Step 4 repair is requested. Keep the Step 5 manual unplug and its commit-named build and log evidence in the separate Step 5 workflow. The advisory recommendation does not authorize a commit.

### Decision rationale for step 4 minimize_loop (exchange 1) (round 2)

Commit-ready is recommended for Step 4. Exact code identity, round 2, occurrence 1 and request-time index tree agree. The amended Step 4 is complete. The requestor's mandatory project Go and TypeScript gates passed, the plan's expected `ghog day` result is documented, and the static coverage check found no new gap. All staged work is attributable to the declared floor, tests, docs, plan and validation record. No current or carried finding remains; the round made no substantive repair. The independently checked `a.commit` groups are accurate. Step 5 remains separately pending, so this recommendation is limited to Step 4.

### Final reviewer decision for step 4 minimize_loop (exchange 1) (round 2)

Decision: commit-ready (advisory). The evidence floor is complete, but this recommendation does not authorize a commit; authority remains at the durable human gate.

<!-- review-entry-id: answer-step-4-round-2 -->

## Round 2 by human - Step 4 - human-confirmation

- Recorded: 2026-09-26T09:37:15+02:00
- Exchange: code/code/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Implementation step: 4
- Outcome: human-confirmation

Human choice: Commit
Outcome: continue-owning-workflow

<!-- review-entry-id: human-confirmation-round-2 -->
