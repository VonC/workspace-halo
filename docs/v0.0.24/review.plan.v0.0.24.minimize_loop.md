# Specification review transcript for v0.0.24

- Exchange: specification/plan/v0.0.24/minimize_loop
- Reviewed document: docs/v0.0.24/plan.v0.0.24.minimize_loop.md

This append-only transcript records completed review rounds. Review agents add
new entries through the review-exchange core and do not reread earlier entries
as working context.

## Round 1 by requestor

- Recorded: 2026-09-25T11:44:38+02:00
- Exchange: specification/plan/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: unrecorded
- Outcome: request

### Review identity for plan minimize_loop (round 1)

Umbrella draft: none
Reviewed specification: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
Review round: 1

### Requestor assessment for plan minimize_loop (round 1)

The plan `docs/v0.0.24/plan.v0.0.24.minimize_loop.md` was just written by
`write-plans` from the settled issue and design, then reviewed once by the
writer, which placed eleven open questions (Q01 to Q11) in its `## Open
questions for the v0.0.24 minimize_loop implementation plan` section.

Scope of the questions: implementation details only, as the plan-review rule
requires. They cover the file layout of the split (Q01), whether the move is a
separate behavior-preserving step (Q02), the controller's test seam (Q03), the
tick an own call records (Q04), the intercept path when the own restore
doesn't settle (Q05), the property-test form in Go (Q06), the location of the
acceptance scenarios (Q07), a coverage criterion for the pure model (Q08), the
per-step gate loop in a repository where groundhog exits 9 (Q09), the legacy
`minimize end` log line (Q10), and where the manual unplug evidence is kept
(Q11).

Missing questions: none known to the writer. Design choices already settled
by the design (observation basis, lateness basis, cap values, latch, no
re-prime, file placement of the pure logic) are deliberately not re-asked.

Sufficiency: every question has two or three options with pros and cons, a
recommended option with arguments, and an `Answer to Qxx` line. The reviewer
is asked to check whether any option set misses a credible alternative,
whether any recommendation conflicts with the settled design or issue, and
whether the plan body has an implementation gap that deserves a new question
(for example in the line budgets, the grep checks, the step order, or the
acceptance scenario list).

Reviewer wording suggestions applied: none yet (round 1).

### Change summary for plan minimize_loop (round 1)

First round: the plan and its validation skeleton are new.

- `docs/v0.0.24/plan.v0.0.24.minimize_loop.md` (new): four steps (verbatim
  split, observation model with own-call absorption and session latch, cap,
  acceptance with documentation and the manual unplug), shared command
  checklist and templates, line-budget checkpoints, and the eleven open
  questions.
- `docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md` (new): skeleton with
  goals and expectations per step, every check section still empty.
- `docs/v0.0.24/issue.v0.0.24.minimize_loop.md` and
  `docs/v0.0.24/design.v0.0.24.minimize_loop.md`: one short "File-based IO
  cost clarification" section each, as the write-plans workflow requires in
  all four documents. No other change to the settled documents.

### Writer response for plan minimize_loop (round 1)

Writer response:

No reviewer feedback exists yet; this is the first round.

The writer asks for an independent assessment of the eleven plan questions and
of the plan body. Two repository facts shape several answers and are worth
checking against the tree: `companion/main_windows.go` is 2022 lines and
`companion/main_windows_test.go` is 661 lines (both over the 650-line
ceiling), and the repository has no `check.bat` and no pytest configuration,
so `ghog day` ends with `exit=9` and the Go suite runs through
`scripts\test-companion.ps1`.

### Reviewer focus for plan minimize_loop (round 1)

Check for missing questions, assess the existing options and answers, and suggest any clearer wording.

<!-- review-entry-id: request-round-1 -->

### LLM nature completion for reviewer (exchange 1)

Recorded nature: `codex`

Completed artifacts:

- `.reviews/a.review-active.specification.plan.v0.0.24.minimize_loop.md`
- `.reviews/a.review-requested.plan.v0.0.24.minimize_loop.md`

<!-- review-entry-id: llm-nature-completion-reviewer-exchange-1 -->

## Round 1 by reviewer

- Recorded: 2026-09-25T12:51:49+02:00
- Exchange: specification/plan/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Outcome: answer

### Reviewer assessment for plan minimize_loop round 1

The four-step sequence and the file split are implementable, and the eleven questions cover the main implementation choices. I recommend changes because three selected answers are not reflected in the executable plan, and the cap sequence contradicts its quiet-period requirement. These are behavior and validation changes, not wording polish.

The cap must record every external shown-to-iconic edge before a skip can return for unknown age or another reason. In lines 723-725 the continuing stream includes unknown-age edges and must remain capped; lines 747-750 instead place the unknown-age and latch checks ahead of the cap path. A stream of old edges could therefore let the cap resume after five seconds even though edges continued. State the order of attempt bookkeeping, resume, and skip-reason precedence, and make the stream test assert the reset clock and reason for each edge.

Q04 chooses a fresh tick after an own call, but lines 603-609 still pass the earlier observation tick into model.ownCall. A slow ShowWindow can make the replay deadline or age bound too early. Q05 chooses to compose only after a confirmed shown reading, but lines 592-600 still compose and log replay-in=75ms after an unsuccessful restore that enters Unsettled. The controller path, log contract, and tests need to match those answers.

Q08 chooses 100% statement coverage of the pure model as a Step 2 and Step 3 completion criterion, while lines 299-307 and the step completion criteria only collect coverage evidence. Supply a repeatable per-file check, including how a split minimize cap file is included, or choose evidence-only and align the question answer. The listed Go coverage output by itself does not enforce the proposed threshold.

The Step 4 manual test is a useful final check. Its verdict should distinguish an interception from the possible two ShowWindow commands in the fallback, and its evidence should identify each external minimize action by window and time so the one-interception claim can be judged from selected log excerpts.

### Question verdicts for plan minimize_loop round 1

- Q01: Choose A. Two files keep the pure model separate and reduce the over-limit main file.
- Q02: Choose A. The standalone move makes the behavior-preserving step reviewable.
- Q03: Choose A. The interface is needed for controller-level scenarios and has a clear contract.
- Q04: Choose A. A fresh tick after the final after-reading aligns replay and settle deadlines with observed state. The body must implement it.
- Q05: Choose A. Compose only on confirmed shown state, log the failed attempt without promising a replay, and count that attempt toward the cap. The body must implement it.
- Q06: Choose A. Native fuzz seeds give deterministic gate coverage and optional exploration; specify meaningful state and transition invariants.
- Q07: Choose A. The test file is projected below the split threshold and can share the fake window.
- Q08: Choose A if the plan adds a real per-file 100% statement-coverage check for the pure model and any later extracted cap file. The current evidence command does not establish this answer.
- Q09: Choose A. The expected groundhog exit 9 and both project suites are explicitly accounted for.
- Q10: Choose A. The new event and observed-edge lines preserve the useful facts without changing the meaning of the legacy line.
- Q11: Choose A. Focused per-window excerpts can support the manual verdict while avoiding unrelated local log contents; include timestamps or action markers for traceability.

Missing question: the cap's bookkeeping order and skip-reason precedence are not resolved by Q01-Q11. Add a question only if the settled design leaves that order open; otherwise write the required order directly into Step 3. The continuing unknown-age stream must never reset the cap's quiet timer.

### Requested changes for plan minimize_loop round 1

Requested changes:

- Reconcile Step 3 cap bookkeeping with every-edge quiet reset and the unknown-age stream acceptance case.
- Reconcile Step 2 own-call timestamps and unsuccessful-restore work/logs with Q04 and Q05.
- Make Q08's selected coverage gate verifiable in Step 2 and Step 3, or change its answer consistently.
- Tighten the Step 4 evidence and fallback wording so the manual verdict can be checked.

### Writer instructions for plan minimize_loop round 1

1. In Step 3, define a transition order that records every external shown-to-iconic edge for the quiet timer even when it is unknown-age, latched, or capped. Specify whether a currently suspended edge reports cap or another skip reason, and make tests at 4999/5000 ms and during the twenty-second unknown-age stream prove that no premature resume occurs.
2. Apply Q04 in the Step 2 ownRestore and ownReplay descriptions: read a fresh tick after the effective ShowWindow after-reading and pass it to model.ownCall. Add a fake-clock test where ShowWindow takes longer than 75 ms and where the next edge remains prompt relative to the after-reading.
3. Apply Q05 in the Step 2 controller description and tests: after an unsuccessful own restore, do not compose the halo or claim replay-in=75ms; log the attempted interception and observed state, enter Unsettled, and count the attempted restore in Step 3's cap.
4. Apply Q08 by stating an executable per-file statement-coverage gate for Step 2 and Step 3, including a cap file if the planned split occurs. If the project does not intend a hard gate, revise Q08's answer to evidence-only and remove the 100% recommendation.
5. In Step 4, describe the manual evidence fields that associate each intercepted line with its window and external minimize action. Clarify that the assertion limits interception attempts, while the one allowed attempt can issue a single SW_RESTORE fallback.

### Final reviewer decision for plan minimize_loop round 1

Decision: changes-requested. The writer should apply the concrete instructions and publish another automated review round.

<!-- review-entry-id: answer-round-1 -->

## Round 2 by requestor

- Recorded: 2026-09-25T12:58:35+02:00
- Exchange: specification/plan/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Outcome: request

### Review identity for plan minimize_loop (round 2)

Umbrella draft: none
Reviewed specification: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
Review round: 2

### Requestor assessment for plan minimize_loop (round 2)

All four round 1 requested changes were accepted and applied to the plan body,
so the executable plan now matches the recommended answers of Q04, Q05 and
Q08, and Step 3 states the cap's bookkeeping order.

Missing questions: none. The cap's bookkeeping order and skip-reason
precedence were written directly into Step 3, as the reviewer allowed,
because the design's target-behavior pseudo-code already fixes the reason
order (`unknown-age`, then `latched`, then `cap`) and its cap section fixes
that every external edge counts as an attempt; no new design choice was
needed.

Sufficiency: the eleven questions keep their options; Q08 option A now names
the enforcing gate script, and Q11 option A now names the timestamps, window
labels and per-action table. All reviewer verdicts agree with the
recommended options.

Reviewer wording suggestions applied: yes, the reviewer's five writer
instructions are applied as described in the change summary.

### Change summary for plan minimize_loop (round 2)

Changes to `docs/v0.0.24/plan.v0.0.24.minimize_loop.md` since round 1:

- Command templates: the coverage profile commands use the space-separated
  flag form (`-coverprofile ..\a.cover.minimize.out`), because Windows
  PowerShell 5.1 splits the `=` form and `go test` fails with `setup failed`;
  both forms were run on this tree. A new "pure-model coverage gate" script
  reads the profile and throws when any block of `minimize_windows.go` (or
  `minimize_cap_windows.go` after a Step 3 split) has a zero count, or when
  none is present. The hook file stays evidence-only (Q08).
- Step 2, controller: a `clock func() uint64` field; every own call takes
  `after := clock()` after its effective after-reading and passes it to
  `model.ownCall`, so `lastShownAt`, `replayAt` and the settle deadline
  follow the reading (Q04). An intercept composes the halo and logs
  `restored=true replay-in=75ms` only when the restore's after-reading is
  shown; otherwise it composes nothing, logs `minimize intercepted:
  restored=false replay=none` after `own call unsettled`, and still counts
  for the cap (Q05). `own restore` lines carry `fallback=<bool>`.
- Step 2, tests: `TestMinimizeControllerUnsettledRestoreComposesNothing` and
  `TestMinimizeControllerStampsOwnCallsAfterTheAfterReading` (a 120 ms
  `ShowWindow` on a fake clock, the replay due at after-tick + 75, and an
  edge 30 ms later still prompt). The coverage gate is a completion
  criterion.
- Step 3: a four-stage order within one observation (Unsettled resolution
  and `maybeResume`; `recordAttempt` for every external shown-to-iconic edge
  while suspended, before any skip returns; reason precedence
  `unknown-age`, `latched`, `cap`, with the trip on the first closed edge;
  `recordIntercept` at the decision). The stream test now alternates prompt
  and `unknown-age` edges, then repeats with a latched model, and asserts
  `lastAttemptAt` and the reason per edge. The resume test covers T + 4999
  and T + 5000 with and without an edge. Two fuzz invariants were added on
  the quiet timer. The coverage gate is a completion criterion.
- Step 4: the manual assertion limits interception attempts (one
  `minimize intercepted` and one `own restore` per external minimize), and
  states that one attempt may carry a single `SW_RESTORE` fallback. The
  rollout adds per-window timestamped excerpts (w1 to w4, line kinds and
  time range listed) and a per-action judgment table (window, edge time, age
  bound, decision, attempts, own restores after the accepted replay,
  verdict), which the completion criterion now uses.
- Line estimates of `minimize_hook_windows_test.go` raised by about 50
  lines per step; still below 550 at about 480.
- Open questions: Q08 option A names the gate script; Q11 option A names the
  timestamps, window labels and per-action table.

`docs/v0.0.24/plan.v0.0.24.minimize_loop.validation.md`: the Step 2, Step 3
and Step 4 expectations match the above.

### Writer response for plan minimize_loop (round 2)

Writer response:

I accept every round 1 finding; there is no disagreement.

1. Cap order: accepted. Attempt bookkeeping now runs before any skip
   returns, so a stream of `unknown-age` or `latched` edges keeps the cap
   suspended. For the reason reported while suspended, I kept the design's
   target-behavior precedence (`unknown-age`, then `latched`, then `cap`)
   instead of reporting `cap` for every edge. The reason is only a log
   label, the design's cap-section sentence still holds for every edge that
   has no earlier reason, and no edge is intercepted while suspended. The
   tests assert both the reason and the reset clock per edge.
2. Q04 and Q05: accepted and written into the controller description and
   tests as the reviewer instructed.
3. Q08: accepted with a hard gate, not evidence-only. While checking it I
   found that the `=` flag form of the coverage command fails under Windows
   PowerShell 5.1, so the template now uses the space form, verified on the
   current tree.
4. Step 4 evidence: accepted; the fallback wording and the per-action table
   are in the rollout and the completion criteria.

### Reviewer focus for plan minimize_loop (round 2)

Check for missing questions, assess the existing options and answers, and suggest any clearer wording.

<!-- review-entry-id: request-round-2 -->

## Round 2 by reviewer

- Recorded: 2026-09-25T13:00:38+02:00
- Exchange: specification/plan/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Outcome: answer

### Reviewer assessment for plan minimize_loop round 2

I recommend convergence. The Step 3 transition order now updates the quiet timer for every external shown-to-iconic edge while suspended, including edges later classified unknown-age or latched. The 4999/5000 ms and continuing-stream tests cover the boundary that round 1 identified. Step 2 now records a fresh clock tick after the effective own-call reading, avoids composition after an unsettled restore, logs its outcome truthfully, and counts its attempted interception. The Step 2 and Step 3 completion criteria now invoke a repeatable pure-model coverage check. Step 4 supplies per-window excerpts and a per-action judgment table.

No implementation decision is missing. The eleven questions still have materially distinct options and support their selected answers. Remaining edits update question descriptions that refer to the pre-revision plan and remove unnecessary workspace names from committed evidence. Those edits do not change behavior or validation criteria.

### Question verdicts for plan minimize_loop round 2

- Q01 A: keep pure model and Win32 glue in separate files.
- Q02 A: keep the behavior-preserving move as Step 1.
- Q03 A: use the three-method window interface for controller tests.
- Q04 A: stamp each own call after its effective after-reading; the Step 2 body and fake-clock test now agree.
- Q05 A: compose only after a confirmed restore, log an unsettled attempt, and count it; the Step 2 body and tests now agree.
- Q06 A: use deterministic native Go fuzz seeds with optional exploration.
- Q07 A: place acceptance scenarios beside controller tests until the file reaches the stated split threshold.
- Q08 A: require full statement coverage of the pure model; the profile gate and Step 2/3 criteria now enforce it.
- Q09 A: keep the stated groundhog verdict and both project test suites per step.
- Q10 A: replace the old minimize-end line with observed-edge and event lines.
- Q11 A: commit focused, timestamped per-window excerpts with the per-action table.

No additional question is needed. The cap's attempt order and skip-reason precedence are specified directly in Step 3, and its tests cover both.

### Convergence evidence for plan minimize_loop round 2

Covered wording:

- Q04 description currently says the plan passes the observation's `now` into `model.ownCall`; change this to describe the previous risk and state that Step 2 now passes the post-reading tick.
- Q05 description currently says the plan always composes after a restore and does not specify an unsettled outcome; describe this as the alternative being decided, because Step 2 now follows option A.
- Q08 description currently says the plan only collects `go tool cover -func` evidence; say it also runs the pure-model coverage gate in Steps 2 and 3.
- In Step 4 rollout step 5, omit the workspace name from committed log headings unless it is deliberately anonymized. The `w1–w4/window<N>` labels already identify each window for the per-action table and keep Q11's privacy rationale intact.
- In the Step 4 acceptance-test description, clarify that “at most one restore” means one `ownRestore` attempt; that attempt may include the single `SW_RESTORE` fallback documented for the manual check.

Convergence rationale:

The revised implementation sequence, cap quiet-period accounting, own-call timestamps, unsettled outcome, coverage gate, and manual evidence are internally consistent. Each in-scope question has a supported answer. The remaining corrections only bring explanatory wording into line with the revised plan, so a further review round would not resolve a live decision.

### Writer instructions for plan minimize_loop round 2

During consolidation, apply the covered wording edits below. They are editorial corrections to descriptions and evidence labels; the plan is ready to converge without another automated review round.

### Final reviewer decision for plan minimize_loop round 2

Decision: convergence-recommended. This recommendation is advisory; consolidation is not confirmed and remains at the durable human gate.

<!-- review-entry-id: answer-round-2 -->

## Round 2 by human - human-confirmation

- Recorded: 2026-09-25T14:36:32+02:00
- Exchange: specification/plan/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/plan.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Outcome: human-confirmation

Human choice: Consolidate
Outcome: continue-owning-workflow

<!-- review-entry-id: human-confirmation-round-2 -->
