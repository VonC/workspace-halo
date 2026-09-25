# Specification review transcript for v0.0.24

- Exchange: specification/design-specification/v0.0.24/minimize_loop
- Reviewed document: docs/v0.0.24/design.v0.0.24.minimize_loop.md

This append-only transcript records completed review rounds. Review agents add
new entries through the review-exchange core and do not reread earlier entries
as working context.

## Round 1 by requestor

- Recorded: 2026-09-25T09:02:05+02:00
- Exchange: specification/design-specification/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/design.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: unrecorded
- Outcome: request

### Review identity for design-specification minimize_loop (round 1)

Umbrella draft: none
Reviewed specification: docs/v0.0.24/design.v0.0.24.minimize_loop.md
Review round: 1

### Requestor assessment for design-specification minimize_loop (round 1)

The design answers the settled issue
(`docs/v0.0.24/issue.v0.0.24.minimize_loop.md`). Before a minimize
notification can drive any state, it is classified by origin (own restore,
own replay, separate, ambiguous) and by age (prompt, late, unknown). Origin
comes from a ledger of the host's own `ShowWindow` calls, matched against
`dwmsEventTime`. Only a prompt, separate `MinimizeStart` below the cap can
start an interception. `MinimizeEnd` never triggers an action, it only feeds
reconciliation from `IsIconic`, so a stale End can no longer reset the state
into a loop. The re-prime path is bounded to one per minimize action, and a
fixed cap backs everything up.

The synchronous cross-thread `ShowWindow` premise is stated as timing
evidence, with a validation obligation through logged offsets. The design
also shows why a failed premise degrades to logged skips (lateness, cap) and
never to a loop.

Six design questions are open: strictness of own-event matching (Q01), the
lateness bound value (Q02), where the logic lives (Q03), the re-prime policy
(Q04), the cap constants (Q05), and the settle timeout (Q06). Implementation
steps are left to the plan.

Points I'd like the reviewer to challenge:

- whether the synchronous `ShowWindow` premise and its safe-failure argument
  hold;
- whether "Separate Start while an interception is in flight" (cancel the
  replay and move to Committed) is right;
- whether any ordering remains in which a stale event restores the window.

### Change summary for design-specification minimize_loop (round 1)

Round 1: first review of the new design document, written from the settled
issue. No earlier round exists. Six open questions (Q01 to Q06) were placed
with `oqm`.

### Writer response for design-specification minimize_loop (round 1)

Writer response:

No reviewer feedback exists yet for this first round. Please assess the
design against the settled issue, challenge any recommendation you would
answer differently, and propose missing design questions.

### Reviewer focus for design-specification minimize_loop (round 1)

Check for missing questions, assess the existing options and answers, and suggest any clearer wording.

<!-- review-entry-id: request-round-1 -->

### LLM nature completion for reviewer (exchange 1)

Recorded nature: `codex`

Completed artifacts:

- `.reviews/a.review-active.specification.design-specification.v0.0.24.minimize_loop.md`
- `.reviews/a.review-requested.design-specification.v0.0.24.minimize_loop.md`

<!-- review-entry-id: llm-nature-completion-reviewer-exchange-1 -->

## Round 1 by reviewer

- Recorded: 2026-09-25T09:15:17+02:00
- Exchange: specification/design-specification/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/design.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Outcome: answer

### Reviewer assessment for design-specification minimize_loop round 1

The design identifies the main loop and adds useful origin, age, reconciliation and cap concepts, but several transitions do not yet meet its stated outcomes. These are behavioral gaps, not wording polish.

First, the ledger establishes time proximity, not origin. Cross-thread ShowWindow may establish that its own event was generated during a call, but does not prove that the first same-kind event in that interval was caused by the call. Q01's consume-once rule prevents multiple events being silently called own, yet a coincidental separate event can consume the entry first and be silently misclassified. The later real own event then becomes ambiguous. The issue explicitly requires uncertain origin to take the safe, logged path. The design needs either additional evidence for a positive own classification or an explicit rule that an interval-only match is ambiguous. Its validation proposal logs offsets only for matched events, so it cannot detect a prompt own replay classified as separate; the cap then permits a second interception, contrary to the one-interception acceptance case.

Second, state reconciliation is too narrow. After Replaying times out, `IsIconic` can move the host to Idle because the user restored the window; an arbitrarily late own replay Start then unconditionally moves it to Committed although the window is shown. A lost separate restore End can likewise leave Committed indefinitely, because reconciliation is only scheduled for an arriving End or for Replaying. Those cases contradict the promise that the phase always matches the actual window. A stale own End being ignored cannot repair them.

Third, a separate Start generated before the host's priming restore can be delivered while Priming. The target rule cancels the replay and moves to Committed on that Start, despite the host having already shown the window. The user's minimize is left undone. Check the actual iconic state before deciding to cancel the replay; if the window is shown, complete the pending replay according to the settled issue. A genuine second minimize during Priming must also be counted as its own action rather than silently completing the first one without an acceptance rule.

Fourth, the cap's 5 s quiet period is measured since the last interception. Once suspended, interceptions stop, so a continuing stream of skipped Starts automatically meets that condition and the cap rearms every 5 s. A residual loop can therefore restore the window twice every cycle indefinitely. Quiet must be defined in terms of attempted Starts or cap hits, or another condition that a persistent stream cannot satisfy. The cap limits rate as written; it does not stop a continuing loop.

Finally, Q06 says a late own Start could no longer matter after a timeout tied to the lateness bound, but the target transition handles an own replay Start regardless of age. The timeout and late-event behavior must be designed together so delayed notifications cannot regress a reconciled phase.

### Question verdicts for design-specification minimize_loop round 1

Q01 — B is better than A or C, but is insufficient as the origin rule. Consuming an entry prevents a second silent match; it does not prove the first match came from the host. Revise the options to distinguish a positively supported own match from an interval-only ambiguous match, and log the latter. The chosen answer must state what happens when a prompt own Start fails to match the ledger.

Q02 — A is a reasonable provisional bound because it separates the observed seconds-long delay from expected prompt delivery. The design has no measured normal event ages yet, so validate the 500 ms value with real logged ages and keep the unknown-age fallback. B risks more lost halos; C permits a more visible late restore.

Q03 — Choose B. Keep the pure classifier, transition and cap logic in a focused file in the same package; leave Win32 actions with their current caller.

Q04 — A can preserve the animation-race retry, but only if the retry does not undo a distinguishable new minimize and the cap is checked before a second restore. The current iconic-only re-prime trigger does not establish why the window is iconic. B loses the recovery path; C allows repeated restores up to the cap.

Q05 — None of the choices is safe with the stated reset rule. Five seconds since the last interception is guaranteed during suspension even if skipped Starts continue, so A periodically rearms a residual loop. Keep the two-in-two-seconds backstop if desired, but require a quiet period without attempted or capped Starts, or an equivalent reset condition that cannot pass during a continuing stream. Reassess the options and answer on that basis.

Q06 — B is acceptable only if an own Start delivered after reconciliation cannot blindly move the state back to Committed. The present rationale is false because own Starts bypass the age skip. Reconcile late events against `IsIconic`, and run reconciliation for Committed when a restore End is lost. A fixed timeout does not solve that alone; C increases races.

Missing decision — What happens to a distinguishable separate Start generated before priming restore but delivered during Priming, or to another Start generated during Priming? The existing transition cancels replay and declares Committed without checking `IsIconic`, which can leave a shown window after a minimize request. Recommend checking the actual state and completing the pending replay when the host's restore has left it shown; specify whether a genuinely new action is counted separately.

Missing decision — How does Committed recover when a user restore End is never delivered, and how are late own events handled after timeout reconciliation? Recommend periodic or event-triggered actual-state reconciliation across the settled phases, with late own events unable to overwrite a newer actual-state observation.

### Requested changes for design-specification minimize_loop round 1

Requested changes:

Another review round is required. The current design can misattribute a separate event as its own, regress a reconciled phase on a late own Start, cancel replay while the window is still shown, and rearm the cap during a continuous stream. Each can defeat a confirmed outcome or an acceptance case.

Suggested safety invariant: "Before an event changes the minimize phase, compare its classification with the current `IsIconic` state and the most recent completed action. A timing match alone does not establish origin. When origin is uncertain, leave the actual state intact and log the skip. A settled phase is reconciled on a bounded schedule so missing Ends cannot leave it stale. Suspension resets only after a real quiet interval with no attempted Starts." The design may choose a different implementation that proves the same outcomes.

### Writer instructions for design-specification minimize_loop round 1

1. Rework origin classification and Q01 so a call-interval match is not treated as sufficient proof that the event is the host's. Explain what additional evidence supports a positive own classification, or route interval-only matches through the ambiguous, logged skip. Cover a coincidental separate event that matches first and a prompt own replay Start that matches no ledger entry.
2. Make state reconciliation robust after the Replaying timeout and while Committed. A late own Start must not move a restored window back to Committed, and a missing restore End must not leave Committed stuck. Add both sequences to the acceptance table.
3. Revisit the separate Start transition in Priming/Replaying. Use the actual iconic state before cancelling a pending replay, and specify how a genuinely new minimize action in flight is counted. Add a test where the Start was generated before priming restore but delivered while Priming.
4. Change the cap reset so continued skipped Starts cannot rearm it. Update Q05, the target behavior and the acceptance cases with a persistent-stream scenario lasting beyond the proposed quiet period.
5. Align Q04's re-prime rule with origin and cap decisions before any additional restore. Align Q06's timeout rationale with the actual handling of an own Start delivered after timeout.

### Final reviewer decision for design-specification minimize_loop round 1

Decision: changes-requested. The writer should apply the concrete instructions and publish another automated review round.

<!-- review-entry-id: answer-round-1 -->

## Round 2 by requestor

- Recorded: 2026-09-25T09:20:30+02:00
- Exchange: specification/design-specification/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/design.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Outcome: request

### Review identity for design-specification minimize_loop (round 2)

Umbrella draft: none
Reviewed specification: docs/v0.0.24/design.v0.0.24.minimize_loop.md
Review round: 2

### Requestor assessment for design-specification minimize_loop (round 2)

The round 1 answer found five behavior gaps, and all five were real. Rather
than patch each one with a guard, the design now bases decisions on a
different signal. The host observes the window's minimized state (`IsIconic`)
on each tick, in each callback, and right before and after each own
`ShowWindow` call. It takes the after-reading as its baseline, so its own
transitions are absorbed, and any later edge is external by construction.
Events only trigger an extra observation. The unplug logs support the switch:
the tick saw the minimizes seconds before the WinEvents arrived.

The questions were revised to match: Q01 is now the decision basis, Q02
flags a deliberate departure from the issue's wording on the age basis, and
Q04 to Q06 were reworked per the verdicts. I see no remaining missing design
question.

### Change summary for design-specification minimize_loop (round 2)

Changes since round 1 in `docs/v0.0.24/design.v0.0.24.minimize_loop.md`:

- Context, scope and target behavior: decisions come from observed `IsIconic`
  edges, not from classified events. The own-call ledger, the origin rule and
  the event age rule are removed.
- New "State Observation" section: observation points, absorption of own
  transitions, the Unsettled path for an own call whose after-reading
  mismatches, the host-clock age bound (`now - lastShownAt`), and missed
  edges.
- Confirmed facts: added that the tick already reads `IsIconic`, and that in
  the unplug logs the tick saw the minimizes at 16:43:49 to 16:43:53 while
  the events arrived at 16:43:55 or later. `dwmsEventTime` is now logged only.
- Phase model: Shown, Priming, Minimized (halo flag), Unsettled. Replaying is
  removed because the replay's after-reading settles it at once. Priming
  keeps the replay unless an external edge occurs. A pre-priming Start
  delivered during Priming is a no-op observation.
- Re-prime: once, only within a 250 ms race window, cap open, counted.
- Cap: suspension ends only after 5 s with no external minimize edge,
  skipped ones included. Unsettled resolutions count toward the cap.
- Acceptance table: added late own Start after a restore, lost restore End,
  pre-priming Start delivered in Priming, own call mismatch, a stalled thread,
  and a 20 s persistent stream.
- Questions: Q01 (decision basis) and Q02 (age basis and bound) rewritten, Q03
  kept, Q04 to Q06 revised.

### Writer response for design-specification minimize_loop (round 2)

Writer response:

All five instructions are addressed, with no disagreement. The main change
answers several of them at once.

1. Origin: the design no longer infers event origin. Own transitions are
   absorbed by reading `IsIconic` right after each synchronous own call, so
   a coincidental separate event can't be misattributed. A prompt own replay
   Start that matches nothing has no effect, because events carry no
   decision. The one premise left (synchronous `ShowWindow`) is checked on
   every own call, and a mismatch takes the Unsettled path: no interception,
   logged, counted by the cap (Q06).
2. Reconciliation: the phase follows observations on every tick. A late own
   Start after a restore finds the window shown and changes nothing. A lost
   restore End is covered by the next tick. Both sequences are in the
   acceptance table.
3. Priming: a Start generated before priming but delivered during Priming
   only triggers an observation. The window is shown, so the replay
   proceeds. An external minimize edge during Priming is either the bounded
   re-prime (race window) or a completed minimize with the halo composed,
   which is not counted as a new interception because nothing is restored.
   The acceptance table has the pre-priming case.
4. Cap: the quiet period now counts every external minimize edge, skipped
   ones included, so a persistent stream keeps the cap closed. A 20 s stream
   case is in the table, and Q05 was revised.
5. Q04 now bounds the re-prime by race window, single use and cap. Q06 now
   covers the Unsettled deadline, where a late own Start can't regress
   anything any more.

One point for the human, not a disagreement with you: the host-clock age
bound departs from the issue's wording, which named the event generation time
as the basis. Q02 records that choice explicitly.

### Reviewer focus for design-specification minimize_loop (round 2)

Check for missing questions, assess the existing options and answers, and suggest any clearer wording.

<!-- review-entry-id: request-round-2 -->

## Round 2 by reviewer

- Recorded: 2026-09-25T09:24:30+02:00
- Exchange: specification/design-specification/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/design.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Outcome: answer

### Reviewer assessment for design-specification minimize_loop round 2

The state-observation redesign removes the round-1 event-order and stale-phase failure modes. A delayed, reordered, duplicated, or lost WinEvent no longer makes a decision by itself. The proposed cap reset also addresses a continuing stream of observed minimize edges.

Two substantive decisions remain. First, an observed shown-to-iconic edge in Priming does not identify its cause. It can be a late completion of the original animation or a separate user minimize. Q04 option A restores in both cases, so its statement that it never undoes the user's minimize is false. The acceptance case at 100 ms proves a retry, but does not cover the separate-user-action case or the manual-unplug foreground guarantee. Second, Q02 option A deliberately replaces the settled issue's validated event-generation-time basis with an observation-clock upper bound. The request explicitly calls this a departure. The design cannot claim that every settled issue guarantee holds while that conflict is unresolved.

There is also a precision issue in the lateness decision. `now - lastShownAt` is an upper bound on the edge's age. When the bound exceeds 500 ms, the edge may have happened just before the observation; its age is unknown, not proven late. Conservative skipping fits the stated fail-safe behavior, but the target behavior, log reason, and stalled-thread acceptance case must say `unknown-age` rather than `late` unless another signal proves lateness.

### Question verdicts for design-specification minimize_loop round 2

- Q01: Option A is the best basis for removing event delivery order from decisions, provided the design limits its claim to observed edges and resolves Q02's cross-document conflict. It does not establish the origin of every later edge; it establishes only that the edge was not observed as the immediate result of an own call.
- Q02: I cannot endorse option A as a settled answer while the issue requires a validated event-generation-time basis. Choose an issue-consistent option, or obtain an explicit amendment of that earlier decision before consolidating this design. If option A is approved, describe the host-clock value as an upper bound that proves promptness only when at most 500 ms and otherwise yields unknown age. Keep `dwmsEventTime` comparison in the logs.
- Q03: Option B. A same-package file keeps the pure phase and cap logic testable without a new API boundary.
- Q04: Choose option B under the evidence currently documented. A shown-to-iconic edge during Priming cannot distinguish animation completion from a fresh user minimize, so cancelling replay and accepting the iconic state avoids restoring the latter. If the writer retains option A, the question and acceptance criteria must explicitly accept one extra restore of a genuine new minimize and establish that this still meets the settled foreground behavior; the current claim that option A never undoes it must be removed.
- Q05: Option A. Recording skipped external edges in the quiet-period calculation prevents a continuing observed stream from rearming the cap.
- Q06: Option A as a bounded uncertainty path, with narrower language: counting an Unsettled resolution helps trip the cap during a burst, but a 1 s timeout and 2 s rolling window alone do not prove that an arbitrarily late own call can never be interpreted as an external edge. Document the remaining premise and test a post-timeout own-call transition.
- Missing decision: whether a genuine new minimize during the first 250 ms of Priming may be restored once. This is implicit in Q04 option A but contradicts its stated consequence, so make it explicit in Q04 rather than adding a separate question.

### Requested changes for design-specification minimize_loop round 2

Requested changes:

1. Resolve Q04's origin ambiguity: either use no re-prime on an observed minimize during Priming, or explicitly accept and validate that a genuine new user minimize can be restored once. Remove the current false assurance that option A never undoes that action.
2. Reconcile Q02's host-clock decision with the settled issue's event-time requirement before treating the design as ready for consolidation.
3. State that `now - lastShownAt > 500 ms` proves only unknown age, and make the skip reason and stalled-thread acceptance case match.
4. Qualify Q06's claim that a finite cap always prevents an arbitrarily late own-call transition from feeding another interception; add a post-timeout transition case.

### Writer instructions for design-specification minimize_loop round 2

Revise Q04 and the Priming transition table to handle the indistinguishable fresh-user-minimize case honestly; option B is my recommendation. Add an acceptance sequence in which the user starts a second minimize within 250 ms of the priming restore, and state whether any extra restore occurs. Reconcile Q02 with the settled issue's event-time decision through an issue-consistent design or an explicit earlier-decision amendment. Correct the age-bound wording and log labels so a bound above 500 ms means unknown age, and narrow Q06's cap guarantee to what its time windows establish. Return the revised design for another automated review round.

### Final reviewer decision for design-specification minimize_loop round 2

Decision: changes-requested. The writer should apply the concrete instructions and publish another automated review round.

<!-- review-entry-id: answer-round-2 -->

## Round 3 by requestor

- Recorded: 2026-09-25T09:30:46+02:00
- Exchange: specification/design-specification/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/design.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Outcome: request

### Review identity for design-specification minimize_loop (round 3)

Umbrella draft: none
Reviewed specification: docs/v0.0.24/design.v0.0.24.minimize_loop.md
Review round: 3

### Requestor assessment for design-specification minimize_loop (round 3)

Round 2 agreed with the observation basis (Q01), the file layout (Q03) and the
cap reset (Q05). It asked for four changes. Three are applied as asked:

- Q04 now takes option B. An observed minimize during Priming cancels the
  replay with no restore, and the re-prime is removed from the design.
- An age bound over 500 ms now yields `unknown-age`, in the target behavior,
  the log label and the stalled-thread case.
- Q06 is strengthened, not just qualified. An Unsettled resolution trips the
  cap at once, and a second one within 60 s latches interception off for the
  host session. Post-deadline and latch cases are in the acceptance table.

On Q02 I keep option A and explain why it is consistent with the issue,
instead of switching or amending. The writer response has the argument. The
disagreement flag is set, and the human confirms Q02 at the gate.

### Change summary for design-specification minimize_loop (round 3)

Changes since round 2 in `docs/v0.0.24/design.v0.0.24.minimize_loop.md`:

- Target behavior: the "late" skip is now "unknown-age". The re-prime branch
  in Priming is replaced by cancelling the replay with no restore. The
  Unsettled deadline now trips the cap, and a second deadline within 60 s
  latches interception off.
- In scope: the bounded re-prime item is replaced by its removal, and the
  session latch is added.
- State Observation: the age bound paragraph states that a bound over 500 ms
  proves nothing (reason `unknown-age`). It adds why the host-clock bound is
  an upper bound of the issue's generation-time age. The Unsettled paragraph
  now describes tripping the cap, the 60 s latch, and why counting alone only
  slowed a late-applied own call.
- Phase model: "External minimize during Priming" never restores and is not
  an interception. "Bounded re-prime" became "Removed re-prime". The Priming
  phase no longer carries a re-prime flag.
- Cap, logging and documentation text updated (no re-prime, `unknown-age`,
  latch window).
- Acceptance table: the 100 ms case now expects no restore. Added a second
  user minimize within 250 ms (no extra restore, not counted), an own replay
  applied 2.5 s after its call (skipped during suspension), and a second
  Unsettled within 60 s (latched). The stalled-thread case says
  `unknown-age`.
- Questions: Q02 restated as the host-clock bound of the generation-time
  age, with an explicit amendment alternative (option C). Q04 now recommends
  B and drops the false assurance. Q06 now recommends immediate cap trip plus
  the session latch.
- Line endings normalized to LF (`oqm` had written CRLF).

### Writer response for design-specification minimize_loop (round 3)

Writer response:

1. Q04: accepted as recommended. Option B: no re-prime, and the claim that
   option A never undoes a user minimize is gone. The case of a second user
   minimize within 250 ms of the priming restore is in the acceptance table,
   with no extra restore.
2. Q02: disagreement, with reasons. The issue's rule is about the age from
   the generation time: prompt within the bound is interceptable, and an age
   that can't be established is not. It named `dwmsEventTime` as a candidate
   basis "once validated", not as the only admissible measurement. The
   notification for an observed edge is generated after the last observation
   that found the window shown. So the generation-time age is at most `now -
   lastShownAt`. A bound within 500 ms therefore proves the issue's prompt
   case, and every other case is the issue's "can't be established" case.
   The design applies the issue's rules to the issue's quantity, measured by
   an upper bound that can only err toward skipping. Q02 now also lists
   amending the issue (option C) as an alternative, and the human decides at
   the gate. If you still read the issue as requiring the timestamp itself,
   please say which issue rule option A breaks.
3. Accepted: a bound over 500 ms proves only an unknown age. The target
   behavior, the log reason and the stalled-thread case now say
   `unknown-age`.
4. Accepted and strengthened. Counting alone only rate-limits, as you noted.
   An Unsettled resolution now trips the cap at once, and a second one
   within 60 s latches interception off for the session, so no delay can
   produce a loop. The acceptance table has the post-deadline transition at
   2.5 s and the latch case.

### Reviewer focus for design-specification minimize_loop (round 3)

Check for missing questions, assess the existing options and answers, and suggest any clearer wording.

<!-- review-entry-id: request-round-3 -->

## Round 3 by reviewer

- Recorded: 2026-09-25T09:32:52+02:00
- Exchange: specification/design-specification/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/design.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Outcome: answer

### Reviewer assessment for design-specification minimize_loop round 3

Round 3 resolves the Priming ambiguity by removing re-prime, and correctly labels an observation interval above 500 ms as unknown age. Q03 and Q05 remain adequately settled.

Q02 still lacks the proof needed to use the host-clock interval as an upper bound on the issue's notification-generation age. `lastShownAt` is the last observation of `IsIconic == false`. That observation precedes the iconic state change, but the design does not establish that it precedes `EVENT_SYSTEM_MINIMIZESTART` generation. The text itself says generation is after it "give or take the short start of the minimize animation". If a notification starts an animation before a shown observation and the state changes later, `now - lastShownAt` can be small while the notification is old. For example, generation at t=0, shown observation at t=1000 ms, iconic observation at t=1020 ms gives a 20 ms host bound but a 1020 ms notification age. Calling the animation start short is not a bound. The unplug logs show delayed delivery and prompt state detection, not the relative generation and `IsIconic` transition times. Option A might be valid for the age of the state transition, but it is not yet proven for the settled issue's notification age.

Q06's new protection still does not establish the stated at-most-one-extra-restore guarantee at any delay. An Unsettled resolution at t=1 s suspends interception, but the cap resumes after 5 s with no edge. A delayed own replay at t=7 s is then observed as a prompt shown-to-iconic edge and can be intercepted. If the next Unsettled resolution is more than 60 s after the first, the latch never engages. The 2.5 s acceptance case tests only an edge during suspension; it does not cover the stated arbitrary-delay guarantee. Q01's statement that later edges are external by construction also conflicts with Q06's acknowledged post-deadline own edge.

### Question verdicts for design-specification minimize_loop round 3

- Q01: Option A remains preferable for state-driven decisions and immunity to WinEvent delivery order. Qualify the origin claim: an edge observed after an own call can still be a delayed effect of that call after an Unsettled deadline, as Q06 explicitly acknowledges. It is an unclassified observed edge unless the own-call completion premise is established.
- Q02: I would choose option A only after the design establishes that notification generation cannot precede `lastShownAt`, or provides a validated maximum lead time and subtracts it from the 500 ms budget. The present `IsIconic` observations bound the iconic state-change age, not necessarily the notification-generation age named by the issue. If that ordering cannot be established, use the issue-consistent validated event-time path for interception or explicitly amend the issue to make state-transition age the rule. The timestamp can remain a comparison log only after the basis is reconciled. The current option A answer is not yet justified.
- Q03: Option B remains sound; same-package pure logic and tests are proportionate.
- Q04: Option B is now sound. The observed edge cannot distinguish the two causes, and cancelling the replay avoids an extra restore of a fresh user minimize.
- Q05: Option A remains sound for a continuing observed minimize stream, with skipped edges extending the quiet period.
- Q06: None of the current options proves the answer's any-delay, session-wide bound. Option A as written allows the first suspension to expire after 5 s of quiet and the 60 s latch window to expire before a second Unsettled resolution. Add an option that keeps interception disabled until the delayed own transition is resolved, or that latches on the first unresolved own call if the safety guarantee must be absolute. If the intended protection is best-effort, narrow the answer and acceptance claim accordingly and show how the settled issue's fixed cap suffices.
- Missing decision: what establishes the temporal ordering between notification generation and the last shown `IsIconic` observation. Put this evidence or a conservative fallback directly into Q02.

### Requested changes for design-specification minimize_loop round 3

Requested changes:

1. Prove or replace Q02's claim that `now - lastShownAt` bounds notification-generation age; observation of the shown state alone only bounds the later iconic state transition, and the design admits animation-start uncertainty.
2. Repair Q06's claimed any-delay guarantee: a delayed own edge after the 5 s quiet reset can be intercepted, and a second Unsettled resolution after 60 s does not latch. Add those two acceptance timelines or narrow the claimed guarantee to a contract the issue permits.
3. Reword Q01's origin assertion so it agrees with Q06's delayed-own-call path.

### Writer instructions for design-specification minimize_loop round 3

Establish Q02's event-generation-time bound with evidence for the required ordering, or choose a decision basis that can enforce the issue's age rule. Remove the unproven "give or take" qualification from a purported proof; if a lead time is possible, bound it and account for it explicitly. Rework Q06 so an own transition after the 5 s suspension and after the 60 s latch window cannot create the repeated interception the answer says is impossible, or state the narrower protection honestly if the settled contract permits it. Add acceptance sequences at 7 s and beyond 60 s. Qualify Q01's "external by construction" language for Unsettled calls. Return a revised design for another automated review round.

### Final reviewer decision for design-specification minimize_loop round 3

Decision: changes-requested. The writer should apply the concrete instructions and publish another automated review round.

<!-- review-entry-id: answer-round-3 -->

## Round 4 by requestor

- Recorded: 2026-09-25T09:35:50+02:00
- Exchange: specification/design-specification/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/design.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Outcome: request

### Review identity for design-specification minimize_loop (round 4)

Umbrella draft: none
Reviewed specification: docs/v0.0.24/design.v0.0.24.minimize_loop.md
Review round: 4

### Requestor assessment for design-specification minimize_loop (round 4)

Round 3 settled Q03, Q04 and Q05, and its three remaining points were all
correct. They are now applied without disagreement:

- Q02: I no longer claim that the host-clock bound measures the notification's
  age. The design says it bounds the minimized-state transition. The ordering
  between a notification's generation and the state change is not
  established, so the unproven "give or take" argument is gone. Q02 now
  recommends explicitly amending the issue's lateness rule to the transition
  age (option A). It also lists the issue-consistent event-time path (option
  B) and a measurement spike (option C). The human's choice at the gate is
  the amendment decision, and consolidation records it in the issue.
- Q06: the first Unsettled resolution now latches interception off for the
  host session, so a delayed own call is skipped at any delay. The 7 s and
  after-90 s cases are in the acceptance table.
- Q01: the origin claim now says an edge is not the immediate result of an
  own call. It comes from outside while every own call has settled as
  expected, and after an uncertain call the latch covers it.

### Change summary for design-specification minimize_loop (round 4)

Changes since round 3 in `docs/v0.0.24/design.v0.0.24.minimize_loop.md`:

- Scope: the latch now engages after the first uncertain own call. Added the
  amendment of the issue's lateness rule to the transition age (Q02).
- Target behavior: an edge is labelled "not the immediate result of an own
  call". The Unsettled deadline now latches interception off for the
  session.
- Absorbing own transitions: the paragraph is requalified. Edges come from
  outside only while own calls settle as expected, and after an uncertain
  call the latch prevents any interception.
- Uncertain outcome: a first-resolution latch replaces the cap trip and the
  60 s window. It explains why expiring protections fail at arbitrary
  delays, and that the latch lasts until the host restarts.
- Age bound: the generation-time claim and its "give or take" argument are
  replaced by the transition-age statement and the amendment, to be recorded
  in the issue at design consolidation.
- Documentation line: the latch is described instead of a latch window.
- Acceptance: the 2.5 s case expects `latched`. Added the 7 s delayed own
  replay, an unrelated minimize 90 s after the latch, and host restart
  re-enabling interception. The 60 s second-resolution case is removed.
- Questions: Q01 qualified, Q02 rewritten (amend, event-time path, spike), and
  Q06 rewritten (first-resolution latch, time-windowed variant, no-deadline
  variant).

### Writer response for design-specification minimize_loop (round 4)

Writer response:

1. Q02: accepted. Your counterexample (generation at 0 ms, shown observation
   at 1000 ms, iconic observation at 1020 ms) shows the bound doesn't cover
   notification age without an ordering proof, and I don't have one. The
   design now bounds only the transition age and recommends amending the
   issue's rule to it, which you listed as acceptable. The alternatives you
   named are kept as options B (validated event time) and C (a lead-time
   spike), and the human decides at the gate.
2. Q06: accepted. The expiring suspension and the 60 s window did fail at 7 s
   and after 60 s, as you showed. The first uncertain own call now latches
   interception off until the host restarts, which bounds the consequence at
   any delay. The 7 s and after-90 s cases are in the table.
3. Q01: accepted. The "external by construction" wording is gone. The claim
   now holds only while own calls settle as expected, with the latch covering
   the delayed own-call path.

### Reviewer focus for design-specification minimize_loop (round 4)

Check for missing questions, assess the existing options and answers, and suggest any clearer wording.

<!-- review-entry-id: request-round-4 -->

## Round 4 by reviewer

- Recorded: 2026-09-25T09:37:57+02:00
- Exchange: specification/design-specification/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/design.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Outcome: answer

### Reviewer assessment for design-specification minimize_loop round 4

The revised design now gives a coherent state-observation path through normal minimizes, delayed and reordered WinEvents, Priming, uncertain own calls, and sustained minimize streams. The Q02 age basis is no longer presented as proof of notification age: it is explicitly a proposed amendment to the issue, to be decided by the human at consolidation. Q06's first-resolution session latch removes the 5 s and 60 s expiry gaps identified in round 3. No further substantive design question is missing.

The human should approve Q02's issue amendment before implementation follows this design. Consolidation must record the approved transition-age rule in the issue. If the human rejects that amendment, Q02 must be revisited; this convergence recommendation does not itself change the issue.

### Question verdicts for design-specification minimize_loop round 4

- Q01: Option A. Observed state edges remove event-delivery order from decisions. Keep the qualified wording that an edge after an uncertain own call is not classified by origin but is protected by the latch.
- Q02: Option A, subject to explicit human approval and the matching issue amendment during consolidation. It measures minimized-state transition age, not notification age. The 500 ms upper bound proves promptness when it is small; a larger bound is unknown and skipped. Option B preserves the old issue rule but reintroduces event dependency; option C does not provide a general guarantee from one measurement.
- Q03: Option B. Same-package pure logic and tests are proportionate.
- Q04: Option B. No re-prime avoids restoring a fresh user minimize that is indistinguishable from the animation race.
- Q05: Option A. Skipped observed edges extend the quiet period, so a continuing stream does not rearm the cap.
- Q06: Option A. The first Unsettled resolution latches interception off for the host session, covering delayed own transitions after any finite timeout or quiet interval.
- No missing decision remains for the selected design. The human's explicit Q02 choice is the outstanding authority step.

### Convergence evidence for design-specification minimize_loop round 4

Covered wording:

- In the Unsettled after-reading acceptance row, replace `cap tripped` with `interception latched off at the deadline`.
- In target pseudocode, evaluate the Unsettled deadline even when `iconic == lastIconic`; an unchanged reading must not bypass settlement.
- In Q01 option A, replace `edges the host did not cause` with `observed edges after expected own-call transitions have been absorbed; uncertain own calls disable interception`.

Convergence rationale:

The former substantive gaps have explicit answers and acceptance timelines: Priming never re-primes an ambiguous minimize, a long observation interval is unknown age, the age basis is a named human amendment choice, and an uncertain own call disables later interception for the session. The remaining corrections are wording and pseudocode placement that preserve the chosen behavior. Recommend convergence and consolidation, conditioned on the human's Q02 amendment decision.

### Writer instructions for design-specification minimize_loop round 4

Consolidate after the human confirms Q02. Record the approved transition-age rule in the issue before using the design for implementation. During consolidation, correct the stale acceptance row that says an Unsettled replay outcome trips the cap; it now latches interception off at the deadline. In the target pseudocode, make the Unsettled deadline check independent of the `iconic == lastIconic` early return, so unchanged observations still resolve and latch. Qualify Q01 option A's shorthand "edges the host did not cause" to match its detailed uncertain-call exception.

### Final reviewer decision for design-specification minimize_loop round 4

Decision: convergence-recommended. This recommendation is advisory; consolidation is not confirmed and remains at the durable human gate.

<!-- review-entry-id: answer-round-4 -->

## Round 4 by human - human-confirmation

- Recorded: 2026-09-25T11:11:04+02:00
- Exchange: specification/design-specification/v0.0.24/minimize_loop
- Umbrella: none
- Reviewed document: docs/v0.0.24/design.v0.0.24.minimize_loop.md
- Requestor LLM nature: claude
- Reviewer LLM nature: codex
- Outcome: human-confirmation

Human choice: Consolidate
Outcome: continue-owning-workflow

<!-- review-entry-id: human-confirmation-round-4 -->
