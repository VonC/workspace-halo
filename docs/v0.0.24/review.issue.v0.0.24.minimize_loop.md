# Specification review transcript for v0.0.24

- Exchange: specification/issue/v0.0.24/minimize_loop
- Reviewed document: docs/v0.0.24/issue.v0.0.24.minimize_loop.md

This append-only transcript records completed review rounds. Review agents add
new entries through the review-exchange core and do not reread earlier entries
as working context.

## Round 1 by requestor

- Recorded: 2026-09-24T21:52:56+02:00
- Exchange: specification/issue/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/issue.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: unrecorded
- Outcome: request

### Review identity for issue minimize_loop (round 1)

Umbrella draft: none
Reviewed specification: docs/v0.0.24/issue.v0.0.24.minimize_loop.md
Review round: 1

### Requestor assessment for issue minimize_loop (round 1)

The issue describes a loop in the native host's minimize interception, seen
when unplugging monitors (three to one). Late `WINEVENT_OUTOFCONTEXT`
delivery reorders the host's own restore `MinimizeEnd` and replay
`MinimizeStart`. The state machine then resets to Idle and takes its own
replayed minimize for a new one, restoring the window again. The document
records the log evidence (four hosts, 12 to 43 interceptions each, never a
`minimize replay accepted` line), the confirmed rules (at most one interception
per minimize, own events never re-intercepted, a per-window cap), and the gap
to close.

Five open questions cover what an issue still needs settled:

- Q01, scope: split the unconfirmed child-mode `WM_DISPLAYCHANGE` observation
  into its own issue.
- Q02, expected behavior: do not intercept a minimize notified late (about
  500 ms bound), to avoid a visible restore seconds after an unplug.
- Q03, cap behavior: fixed, documented constants, logged in the host log.
- Q04, edge case: a real user restore during the replay window; the actual
  window state wins, so the state never sticks.
- Q05, acceptance: unit tests replaying the logged sequences plus one manual
  unplug checked in `native-host.log`.

I don't see a missing question for an issue document. Implementation choices
(how to measure lateness, where to keep the replay tick) are left to the
design. Every question has at least two options with pros and cons, a
recommendation, and an `Answer to Qxx` line. This is round 1, so no reviewer
wording suggestions exist yet.

Points I'd like the reviewer to challenge:

- whether Q02's late-notification rule conflicts with the confirmed rule that
  every minimize gets its haloed thumbnail in the normal case;
- whether the ~500 ms and 2-in-2 s figures belong in the issue or in the
  design;
- whether the gap list mixes in implementation detail an issue should avoid.

### Change summary for issue minimize_loop (round 1)

Round 1: first review of the new issue document. There is no earlier round to
compare with. The document was written from
`docs/v0.0.24/draft.v0.0.24.minimize_loop.md`, and its five open questions
(Q01 to Q05) were placed with `oqm`.

### Writer response for issue minimize_loop (round 1)

Writer response:

No reviewer feedback exists yet for this first round. Please assess the issue's
scope, confirmed rules, examples and the five open questions. Challenge any
recommendation you would answer differently, and propose missing questions
about scope, expected behavior, edge cases or acceptance.

### Reviewer focus for issue minimize_loop (round 1)

Check for missing questions, assess the existing options and answers, and suggest any clearer wording.

<!-- review-entry-id: request-round-1 -->

### LLM nature completion for reviewer (exchange 1)

Recorded nature: `codex`

Completed artifacts:

- `.reviews/a.review-active.specification.issue.v0.0.24.minimize_loop.md`
- `.reviews/a.review-requested.issue.v0.0.24.minimize_loop.md`

<!-- review-entry-id: llm-nature-completion-reviewer-exchange-1 -->

## Round 1 by reviewer

- Recorded: 2026-09-24T22:02:12+02:00
- Exchange: specification/issue/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/issue.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Outcome: answer

### Reviewer assessment for issue minimize_loop round 1

The observed event order and log counts make the loop credible, and the core outcome is clear: the host must stop restoring the same window repeatedly and leave the intended minimize stable. The draft still contains behavior contradictions and unproven detection assumptions that prevent implementation and acceptance from being unambiguous.

The confirmed rule promises that the host's own events never begin a new interception regardless of arrival order or delay. Gap item 2 and the third concrete example instead classify a start as self-generated only for about one second. A self-generated start after that interval can repeat the loop; a genuine second user minimize inside the interval can be swallowed. The cap limits damage but does not satisfy the stronger rule. The issue must say which guarantee is required, and the design must demonstrate a way to distinguish events or recover safely without relying on a time window alone.

The rule that every minimize ends minimized conflicts with Q04's chosen outcome that a user restore during replay remains restored. State the minimize outcome with that explicit user-restore exception. Moreover, checking the actual window state after replay cannot preserve a restore that happens before a pending SW_MINIMIZE call; the pending replay would minimize it again. The requirement must say when a real restore cancels pending replay and what later reconciliation must observe, while leaving the implementation mechanism to design.

The late-notification exception in Q02 is reasonable, but the draft has not established how notification age can be known reliably or what happens if it cannot. The observed 3 to 6 seconds are inferred from log chronology; an arrival timestamp alone cannot measure event age. State an observable outcome and require a trustworthy event-time basis, or choose another detectable criterion. A nominal 500 ms value and the proposed circuit-breaker numbers are illustrative until evidence or a design fixes them.

Q01 is a valid scope decision. The unconfirmed child-mode display-change observation should be tracked separately, while gap item 5 and the dependent implementation proposal leave this issue. Q05's manual pass criterion should be per minimize action, since legitimate repeated user minimizes in one session may yield more than one interception per window over time.

### Question verdicts for issue minimize_loop round 1

Q01 — Choose B. The display-change reception is unconfirmed and affects topology tracking beyond this minimize loop. Keep a short related-work pointer, remove gap item 5 and do not make this issue's completion depend on that investigation. A loses a focused acceptance boundary; C risks losing a real regression.

Q02 — Choose B as an outcome, conditional on establishing trustworthy event age or an equivalent detectable late-notification condition. Prompt normal minimizes retain halo composition; late notifications do not restore a window solely to add a halo and are logged. A preserves the disruptive late pop; C depends on Q01 and suppresses unrelated prompt minimizes. Add a decision for unavailable or ambiguous timing evidence; the current 500 ms example is not itself a proven acceptance threshold.

Q03 — Choose A for fixed, documented and logged protection. It is a safety bound, not a user-facing tuning need. Define whether the limit means two allowed interceptions and suspension on the third, when its window starts, and when it resets; leave exact durations to design unless the issue has evidence for specific values. B creates configuration and transport work; C creates a new reporting path for a rare condition.

Q04 — Choose B for respecting a real user restore and recovering from missing or reordered events, but its current answer is incomplete. If a user restores before replay, a later SW_MINIMIZE must not undo that action; if restore occurs after replay, state reconciliation must reflect the restored window. An actual-state read only after replay does not guarantee the first case. A drops the user's action; C permits stuck state.

Q05 — Choose B with clearer evidence. Test the recorded in-order and reordered sequences, a replay start delayed beyond any proposed recognition window, a prompt separate user minimize, a user restore on either side of replay, missing-event recovery, and breaker reset. For a real unplug, require no repeated interception cycle per minimize action and no unsolicited foreground restore; collect the relevant host-log evidence. A misses the real trigger; C adds product test machinery without demonstrated need.

Missing decision — How is a host-generated replay distinguished from a separate minimize when events can be reordered and arbitrarily late? Option A requires durable causal attribution or equivalent state reconciliation, preserving the confirmed guarantee but needing design proof. Option B uses a bounded timing heuristic with the breaker as fallback, simpler but requiring the issue to weaken its no-reinterception guarantee and explicitly allow one extra interception. Recommend A, or revise the confirmed rule and acceptance if the platform cannot support A. The present draft implicitly promises A while specifying B.

### Requested changes for issue minimize_loop round 1

Requested changes:

Substantive revision is required. In particular, the unbounded own-event guarantee conflicts with the one-second heuristic, and the chosen user-restore behavior is not ensured by a post-replay state check. Settle those contracts, the late-event measurement/fallback, and the related acceptance cases before another review round.

Suggested replacement for the core outcome: "A host-generated restore or replay must not start another interception, including when its notifications arrive out of order or late. A separate user minimize remains eligible for normal interception. If the user restores during a pending interception, that restore remains effective and the host's state returns to a state consistent with the actual window. When a notification is demonstrably too late for a nondisruptive halo capture, the host leaves the window minimized, skips interception and records the reason." Use this wording only if the implementation can support the strong own-event guarantee; otherwise document the weaker bound and fallback explicitly.

### Writer instructions for issue minimize_loop round 1

1. Resolve the self-generated event question explicitly. Align the confirmed rule, gap, examples and acceptance criteria on either a guarantee that remains true after arbitrary delay or a stated bounded guarantee with a defined fallback. Do not treat a one-second clock check as proof of event identity.
2. Reconcile the general minimized outcome with a real user restore. Specify the expected result for a restore before the pending replay and after it, including cancellation or reconciliation so the user's action survives and the state cannot remain stuck.
3. Keep Q02's chosen late-event behavior only with a clear, measurable lateness criterion and fallback for unknown timing. Describe 500 ms, two interceptions, two seconds and suspension duration as examples unless confirmed requirements justify exact thresholds.
4. Apply Q01 consistently: remove the display-change-dependent gap item from this issue and track the unconfirmed reception problem in a separate issue or explicit follow-up. Keep only a short related-work reference here.
5. Rewrite the numbered implementation gap as required behaviors and acceptance outcomes. Move specific state transitions, replay ticks, and proposed function-level fixes to the design, while retaining the observed code references as context.
6. Expand Q05's acceptance to cover the counterexamples in the verdicts. Express the manual log criterion per minimize action and record the no-pop result alongside the absence of a repeated cycle.

### Final reviewer decision for issue minimize_loop round 1

Decision: changes-requested. The writer should apply the concrete instructions and publish another automated review round.

<!-- review-entry-id: answer-round-1 -->

## Round 2 by requestor

- Recorded: 2026-09-24T22:07:43+02:00
- Exchange: specification/issue/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/issue.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Outcome: request

### Review identity for issue minimize_loop (round 2)

Umbrella draft: none
Reviewed specification: docs/v0.0.24/issue.v0.0.24.minimize_loop.md
Review round: 2

### Requestor assessment for issue minimize_loop (round 2)

The round 1 answer was right on its main points. The confirmed rule promised a
guarantee for any delay, but the gap specified a 1 s heuristic. Q04's outcome
was not ensured by a post-replay check alone. Q02 had no event-time basis. The
revised issue now aligns rules, required behaviors, examples and questions on
one contract.

Questions: the missing decision became Q06 (how the host recognizes its own
restore and replay), recommending causal attribution by the event generation
time. Q02 now uses the same basis and defines a fail-safe fallback for an
unknown or ambiguous age. Q03 defines trip and reset semantics. Q05 lists every
counterexample from the verdicts and judges the manual unplug per minimize
action. Every threshold is described as an example that the design fixes. I
don't see a remaining missing question for an issue document.

One point departs from the round 1 verdict: Q04 now recommends option A, not
B. The writer response explains why.

### Change summary for issue minimize_loop (round 2)

Changes since round 1 in `docs/v0.0.24/issue.v0.0.24.minimize_loop.md`:

- Evidence: the 3 to 6 s delay is now described as inferred from log
  chronology, since the host does not record event generation time.
- Confirmed rule: rewritten from the suggested replacement. The own-event
  guarantee holds for any delay or order (Q06). A separate user or system
  minimize stays eligible. One minimize action leads to at most one
  interception, except for a real user restore (Q04). The state is always
  reconciled with the actual window. The halo is kept for prompt in-order
  minimizes. The cap is a backstop (Q03).
- The numbered implementation gap is replaced by "Required behaviors to close
  the gap", stated as behaviors and acceptance outcomes with no function-level
  fixes. Thresholds are marked as examples for the design to fix.
- The display-change observation is reduced to a short "Related work outside
  minimize_loop" pointer, and the dependent gap item is removed (Q01).
- Examples: the 1 s window example is replaced by an arbitrarily late replay
  that is still recognized. Added a prompt separate user minimize after a
  replay, a restore after the replay, a late notification, an unknown-age
  notification, and the cap as backstop.
- Code references: `minimizeWinEventProc` is noted to discard `idEventThread`
  and `dwmsEventTime`. The display-change references are marked as related
  work only.
- Open questions: Q01 to Q05 revised per the verdicts, Q06 added.

### Writer response for issue minimize_loop (round 2)

Writer response:

Accepted instructions:

1. Self-generated events: added Q06 on the guarantee and its basis. The
   recommendation is causal attribution: a notification is the host's own when
   the event generation time Windows supplies (`dwmsEventTime`, discarded
   today) falls within the host's restore or replay call interval. That holds
   for arbitrary delays. An ambiguous case fails safe (no restore, logged), and
   reconciliation with the actual window stays the safety net. The design must
   prove the timing premise. The confirmed rule, required behaviors and
   examples no longer rely on a clock check at arrival.
2. General outcome versus user restore: the rule now states the minimized
   outcome with the Q04 exception, plus reconciliation so the state never
   sticks.
3. Q02: lateness is now measured as the event's age from its generation time,
   with a fail-safe fallback for a missing, future or wrapped timestamp. 500
   ms, 2 interceptions, 2 s and the suspension duration are examples.
4. Q01: gap item 5 is removed, and only a short related-work pointer remains.
5. The gap is rewritten as required behaviors and acceptance outcomes.
   Function-level fixes are left to the design, and code references stay as
   context.
6. Q05 now covers every listed counterexample and judges the manual unplug per
   minimize action, including the no-pop result.

Disagreement on Q04: the verdict asked that a user restore before the pending
replay not be undone. In that interval the window is not minimized, because
the host's priming restore has already shown it, so no user restore can occur.
The only user action there is interacting with the shown window, for example
clicking into it. Q04 now spells out these cases. It recommends option A: the
replay completes the user's own minimize request, a restore after the replay
is honored, and the state is reconciled in every case. Option B (user
activation cancels the replay) stays available. It needs activation caused by
the host (the `SW_RESTORE` fallback activates the window) to be told apart
from activation by the user, which risks leaving a window shown that the user
asked to minimize. Please challenge this if you see a real user restore that
can happen before the replay.

### Reviewer focus for issue minimize_loop (round 2)

Check for missing questions, assess the existing options and answers, and suggest any clearer wording.

<!-- review-entry-id: request-round-2 -->

## Round 2 by reviewer

- Recorded: 2026-09-24T22:09:36+02:00
- Exchange: specification/issue/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/issue.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Outcome: answer

### Reviewer assessment for issue minimize_loop round 2

The revision resolves the late-arrival contradiction in the original wording, separates the unconfirmed display-change work, defines a conservative unknown-age outcome, and adds the missing acceptance counterexamples. Q04's new choice is reasonable: the host has already shown the window during priming, so a user click before replay is activation rather than a restore; completing the pending minimize is an explicit, testable policy.

One substantive decision remains incorrect. Q06 calls a generation timestamp inside the host's ShowWindow call interval "causal attribution." The timestamp establishes when an event was generated, not who caused it. A separate user or system event may be generated in the same interval, timestamps may share a millisecond, and a host-caused event may be generated outside the measured call interval. The document itself says the design must prove the premise. Until then, Option A cannot claim that every separate minimize stays eligible while every own event is suppressed for arbitrary delay. The ambiguous-event fallback also skips a separate minimize, contrary to the unconditional confirmed rule and required behavior 2.

Keep the behavior goal and conservative fallback, but state the tradeoff honestly. A separate minimize is eligible when it can be distinguished from host-generated events and passes the late-notification and cap rules. If attribution is ambiguous, the host lets the current window state stand, does not restore solely to compose the halo, and logs the skipped interception. The design must validate any proposed attribution method, including the timestamp interval premise; an interval match alone must not be described as proof of origin. If exact attribution cannot be established, the issue must allow the conservative skip rather than promise normal interception for every separate minimize.

### Question verdicts for issue minimize_loop round 2

Q01 — Agree with B. The separate follow-up and short pointer keep this issue's acceptance focused.

Q02 — Agree with B, including the no-restore fallback when event age is unavailable or ambiguous. The design must handle the timestamp's clock and wrap behavior before using a chosen bound.

Q03 — Agree with A. The sliding count, trip on the next minimize, quiet-period reset, logging and design-owned exact values are sufficiently specified.

Q04 — Agree with the revised A. During priming the host has already shown the window, so the cited click is not a distinct restore. Completing the replay honors the pending minimize; a restore after replay must still win and the state must reconcile.

Q05 — Agree with B. The expanded unit sequences and per-action manual criterion cover the reported failure and relevant counterexamples. Add an ambiguous-attribution case alongside unknown age to test the conservative fallback.

Q06 — Disagree with A as written. Event generation time within a call interval is timing evidence, not causal identity; simultaneous unrelated events or a host-caused event generated outside that interval are not ruled out. Choose the strong safety outcome with an explicit uncertainty exception: recognized host events never trigger interception; distinguishable separate minimizes remain eligible; ambiguous events are not intercepted and are logged. Require design evidence for any claimed attribution, including the interval method. B still allows a visible repeat, and C does not cover arbitrarily late delivery.

### Requested changes for issue minimize_loop round 2

Requested changes:

Another round is needed because Q06 promises causal attribution from a timestamp interval without evidence that the interval uniquely identifies the host's action. This is a behavior contract, not wording polish: the current rule says every separate minimize remains eligible, while the documented ambiguous fallback necessarily skips some separate minimizes.

Suggested replacement: "A notification identified as caused by the host's restore or replay never starts another interception, regardless of delivery order or delay. A separate minimize remains eligible when it can be distinguished from host-generated events and passes the late-notification and cap rules. If origin is ambiguous, the host does not restore the window solely to compose a halo, leaves the actual window state intact, and logs the skipped interception. The design must justify any event-attribution method; matching an event's generation time to a host-call interval is timing evidence, not by itself proof of origin."

### Writer instructions for issue minimize_loop round 2

1. Revise Q06's Option A, recommendation and answer so a matching event-time interval is a candidate method subject to validation, not an assertion of causal proof. State the desired behavior and the conservative ambiguous-event outcome independently of that method.
2. Qualify the confirmed rule and required behavior that a separate minimize stays eligible: it must be distinguishable from host-generated events, prompt enough under Q02, and below the Q03 cap. When origin is ambiguous, do not restore merely to add a halo; retain the actual window state and log the skipped interception.
3. Add an ambiguous-origin example and unit acceptance case. Keep the existing arbitrarily late host-event example as the target for events the host can identify, and state what happens when it cannot identify one with confidence.
4. In the code references and Q02, retain `dwmsEventTime` as available timing input without claiming that it alone identifies the event's actor. Ask the design to validate time-base, precision, callback timing and wrap behavior for the intended uses.

### Final reviewer decision for issue minimize_loop round 2

Decision: changes-requested. The writer should apply the concrete instructions and publish another automated review round.

<!-- review-entry-id: answer-round-2 -->

## Round 3 by requestor

- Recorded: 2026-09-24T22:13:16+02:00
- Exchange: specification/issue/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/issue.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Outcome: request

### Review identity for issue minimize_loop (round 3)

Umbrella draft: none
Reviewed specification: docs/v0.0.24/issue.v0.0.24.minimize_loop.md
Review round: 3

### Requestor assessment for issue minimize_loop (round 3)

The round 2 answer agreed with Q01 to Q05, including the revised Q04, and
found one substantive error: Q06 presented an event-time interval match as
causal attribution. That is timing evidence, not proof of origin, and the
unconditional eligibility rule contradicted the ambiguous-origin fallback.
Both are now fixed. Q06 recommends the safety outcome with an explicit
uncertainty exception and leaves the attribution method to the design to
validate. The confirmed rule and required behaviors now qualify when a
separate minimize is eligible. An ambiguous-origin example and unit case are
added. I see no remaining missing question.

### Change summary for issue minimize_loop (round 3)

Changes since round 2 in `docs/v0.0.24/issue.v0.0.24.minimize_loop.md`:

- Confirmed rule: adopted the suggested replacement. Identified own
  notifications never start an interception. A separate minimize is eligible
  when it can be told apart from host events, is prompt enough (Q02), and the
  window is below the cap (Q03). An ambiguous origin means no restore just for
  a halo, the actual window state kept, and a logged skip. The design must
  justify any attribution method, and an event-time interval match is timing
  evidence only.
- Required behaviors 1 and 2: rewritten with the ambiguous-origin outcome and
  the eligibility conditions.
- Examples: the arbitrarily late replay example now applies to events the host
  can identify. The repeated user minimize example requires the host to tell
  it apart. Added an ambiguous-origin example.
- Code references: `dwmsEventTime` is described as timing input that does not
  identify the actor, and the design must validate its time base, precision,
  callback timing and wrap behavior.
- Q02: the event time is described as timing input to validate, not as a
  settled measurement.
- Q05: an ambiguous-origin unit case is added.
- Q06: rewritten. Option A is now the safety outcome with the uncertainty
  exception, and the interval match is a candidate method subject to design
  validation.

### Writer response for issue minimize_loop (round 3)

Writer response:

All four round 2 instructions are accepted and applied, with no remaining
disagreement. The Q04 disagreement from round 1 is resolved by your
agreement with the revised option A.

1. Q06: Option A, its recommendation and its answer now treat the event-time
   interval as a candidate method that the design must validate. The desired
   behavior and the ambiguous-origin outcome are stated independently of any
   method.
2. The confirmed rule and required behavior 2 now qualify eligibility: it
   must be distinguishable, prompt under Q02, and below the Q03 cap. An
   ambiguous origin means no restore just for a halo, the actual state kept,
   and a logged skip.
3. An ambiguous-origin example and Q05 unit case are added. The arbitrarily
   late host-event example now applies to events the host can identify.
4. The code references and Q02 now present `dwmsEventTime` as timing input
   only, with its time base, precision, callback timing and wrap behavior left
   for the design to validate.

### Reviewer focus for issue minimize_loop (round 3)

Check for missing questions, assess the existing options and answers, and suggest any clearer wording.

<!-- review-entry-id: request-round-3 -->

## Round 3 by reviewer

- Recorded: 2026-09-24T22:14:43+02:00
- Exchange: specification/issue/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/issue.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Outcome: answer

### Reviewer assessment for issue minimize_loop round 3

The current issue gives a coherent behavior contract for the observed loop. It separates the unconfirmed display-change concern, protects late notifications and uncertain event origin without restoring the window, preserves normal halo capture when conditions allow, bounds residual repeats, respects the selected replay and restore policy, and identifies both automated and real-hardware acceptance evidence. Q06 now leaves the event-attribution method and its proof to the design. No missing in-scope question remains.

The remaining edits are wording alignment only. The general prompt-minimize halo sentence should acknowledge the attribution and cap conditions already stated above it. Q02's recommendation should say that event generation time is timing input to be validated, not already a direct and reliable age measure. Q05's manual criterion should describe the delayed or repeated foreground pop the issue aims to prevent, since a prompt normal interception can briefly show the window during priming.

### Question verdicts for issue minimize_loop round 3

Q01 — Agree with B: track the unconfirmed display-change issue separately and retain a short pointer.

Q02 — Agree with B: skip late or unmeasurable notifications and log the reason, using a bound the design validates. Adjust the recommendation's certainty about the event timestamp.

Q03 — Agree with A: fixed, logged cap with trip and reset semantics; the design sets exact values.

Q04 — Agree with A: the pending replay completes the minimize, a later real restore is honored, and state reconciliation prevents an intermediate state from sticking.

Q05 — Agree with B: the expanded sequence tests and per-action real unplug evidence are appropriate. Clarify that the visual check concerns unsolicited delayed or repeated foreground restores.

Q06 — Agree with revised A: identified host events do not trigger interception, distinguishable separate events remain eligible under Q02 and Q03, and ambiguous events are skipped and logged. The design validates any attribution method. No additional question is needed.

### Convergence evidence for issue minimize_loop round 3

Covered wording:

Suggested wording for the confirmed rule: "A prompt minimize with in-order notifications gets a haloed thumbnail when its origin is distinguishable and the per-window cap has not applied."

Suggested Q02 recommendation phrase: "The event generation time can provide a basis for measuring age once the design validates its clock, precision and wrap behavior."

Suggested Q05 manual phrase: "For each minimize action, the host log shows no repeated interception cycle, and no window makes an unsolicited delayed or repeated return to the foreground after the unplug."

Convergence rationale:

All six questions have distinct options and defensible chosen answers. The safety outcome, uncertainty fallback, per-action cap and manual evidence are consistent across the issue. The remaining changes only make three existing statements reflect qualifications already present elsewhere in the document; they do not require a new decision or another automated review round. Convergence is advisory, with consolidation reserved for the human choice.

### Writer instructions for issue minimize_loop round 3

The issue is ready for consolidation after wording polish. Align the prompt-minimize halo sentence with the existing attribution and cap qualifications; soften Q02's claim of reliable direct age measurement until the design validates the time base; and state the manual no-pop criterion as no unsolicited delayed or repeated foreground restore after the unplug. Preserve the decisions and examples otherwise.

### Final reviewer decision for issue minimize_loop round 3

Decision: convergence-recommended. This recommendation is advisory; consolidation is not confirmed and remains at the durable human gate.

<!-- review-entry-id: answer-round-3 -->

## Round 3 by human - human-confirmation

- Recorded: 2026-09-24T22:52:09+02:00
- Exchange: specification/issue/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/issue.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Outcome: human-confirmation

Human choice: Consolidate
Outcome: continue-owning-workflow

<!-- review-entry-id: human-confirmation-round-3 -->
