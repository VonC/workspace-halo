//go:build windows

package main

// The pure minimize interception logic: the WinEvent identifiers the host
// watches, the filter that maps a WinEvent callback to the tracked top-level
// window, and the observation model that decides interceptions. It holds no
// Win32 call, so every transition is testable without a window; the hook, the
// controller and their callers live in minimize_hook_windows.go.
//
// v0.0.24 replaces the event-order state machine (its event transition and
// the idle, replaying and committed phases) by minimizeModel, driven by
// IsIconic readings and ticks only. A late, lost, duplicated or reordered
// event can only cause an extra observation of the present state, which finds
// no edge. The model absorbs the transitions of the host's own ShowWindow
// calls through their after-readings, skips a shown-to-iconic edge whose age
// bound exceeds minimizeLatenessBoundMS, cancels a pending replay without any
// restore when the window is minimized again during Priming (no re-prime), and
// latches interception off for the host session at the first own call whose
// outcome stayed unsettled past minimizeSettleTimeoutMS.
//
// v0.0.24 step 3 adds the per-window interception cap, minimizeCap: an edge
// that would be the third interception within minimizeCapWindowMS is skipped
// with reason cap and suspends interception. While suspended, every external
// shown-to-iconic edge moves the quiet timer, whatever its skip reason, so
// only minimizeCapQuietMS without any such edge resumes it: a continuing
// stream of minimizes can't re-arm the cap. An edge during Priming is never
// counted as an interception.

const (
	eventSystemMinimizeStart = uint32(0x0016)
	eventSystemMinimizeEnd   = uint32(0x0017)
	objidWindow              = int32(0)
	childidSelf              = int32(0)
	wineventOutofcontext     = uint32(0x0000)
)

// minimizePhase is the interception phase of the tracked window. Shown and
// Priming imply a shown last reading, Minimized an iconic one; Unsettled holds
// either until its deadline.
type minimizePhase uint8

const (
	minimizeShown minimizePhase = iota
	minimizePriming
	minimizeMinimized
	minimizeUnsettled
)

// minimizeEdge is the change between the previous reading and the observed one.
type minimizeEdge uint8

const (
	minimizeNoEdge minimizeEdge = iota
	minimizeShownToIconic
	minimizeIconicToShown
)

var minimizeEdgeNames = [...]string{"none", "shown->iconic", "iconic->shown"}

func (e minimizeEdge) String() string {
	return minimizeEdgeNames[e]
}

// minimizeAction is what the caller of minimizeModel.observe must do, or log.
type minimizeAction uint8

const (
	minimizeNoAction minimizeAction = iota
	minimizeIntercept
	minimizeCancelReplay
	minimizeSkip
	minimizeRestoreHonored
	minimizeAbsorbed
	minimizeSettled
)

var minimizeActionNames = [...]string{
	"none",
	"intercept",
	"cancel-replay",
	"skip",
	"restore-honored",
	"absorbed",
	"settled",
}

func (a minimizeAction) String() string {
	return minimizeActionNames[a]
}

// Skip reasons of a shown-to-iconic edge in phase Shown.
const (
	minimizeReasonUnknownAge = "unknown-age"
	minimizeReasonLatched    = "latched"
	minimizeReasonCap        = "cap"
)

const (
	minimizeReplayDelayMS   = uint64(75)
	minimizeLatenessBoundMS = uint64(500)
	minimizeSettleTimeoutMS = uint64(1000)
)

// The interception cap: at most minimizeCapCount interceptions within
// minimizeCapWindowMS, and a suspension that only minimizeCapQuietMS without
// any external shown-to-iconic edge ends.
const (
	minimizeCapCount    = 2
	minimizeCapWindowMS = uint64(2000)
	minimizeCapQuietMS  = uint64(5000)
)

// minimizeCap holds the last minimizeCapCount interception ticks, oldest
// first, with their fill count, the suspension, and the tick of the latest
// external shown-to-iconic edge seen while suspended.
type minimizeCap struct {
	intercepts    [minimizeCapCount]uint64
	filled        int
	suspended     bool
	lastAttemptAt uint64
}

// closed reports whether an interception at now must be skipped: the cap is
// suspended, or minimizeCapCount interceptions all happened within the last
// minimizeCapWindowMS.
func (c minimizeCap) closed(now uint64) bool {
	return c.suspended || (c.filled == minimizeCapCount && now-c.intercepts[0] < minimizeCapWindowMS)
}

// recordIntercept records the tick of an interception decided at now.
func (c minimizeCap) recordIntercept(now uint64) minimizeCap {
	copy(c.intercepts[:], c.intercepts[1:])
	c.intercepts[minimizeCapCount-1] = now
	if c.filled < minimizeCapCount {
		c.filled++
	}
	return c
}

// recordAttempt records an external shown-to-iconic edge at now against a
// closed cap: it moves the quiet timer to now and suspends the cap, reporting
// whether this edge tripped it.
func (c minimizeCap) recordAttempt(now uint64) (minimizeCap, bool) {
	tripped := !c.suspended
	c.suspended = true
	c.lastAttemptAt = now
	return c, tripped
}

// maybeResume ends a suspension, and clears the interception history, once
// minimizeCapQuietMS passed since the latest recorded attempt.
func (c minimizeCap) maybeResume(now uint64) (minimizeCap, bool) {
	if !c.suspended || now-c.lastAttemptAt < minimizeCapQuietMS {
		return c, false
	}
	return minimizeCap{}, true
}

// minimizeModel is the whole interception state of one window: the phase, the
// last IsIconic reading, the tick of the last shown reading, the pending
// replay, the pending own-call expectation, the session latch and the
// interception cap.
type minimizeModel struct {
	phase          minimizePhase
	iconic         bool
	lastShownAt    uint64
	replayAt       uint64
	haloComposed   bool
	expectIconic   bool
	settleDeadline uint64
	latched        bool
	cap            minimizeCap
}

// minimizeDecision reports one observation: the edge found, its age bound (for
// a shown-to-iconic edge), the action, the skip reason, whether this
// observation resolved an Unsettled phase and latched interception off, and
// whether it tripped or resumed the interception cap.
type minimizeDecision struct {
	edge       minimizeEdge
	ageBound   uint64
	action     minimizeAction
	reason     string
	latchedNow bool
	capTripped bool
	capResumed bool
}

// newMinimizeModel seeds the model from the first reading: a shown window is
// Shown since now, an iconic one is Minimized without a halo and is never
// intercepted, because its minimize was not observed.
func newMinimizeModel(iconic bool, now uint64) minimizeModel {
	if iconic {
		return minimizeModel{phase: minimizeMinimized, iconic: true}
	}
	return minimizeModel{phase: minimizeShown, lastShownAt: now}
}

// observe applies one IsIconic reading taken at now. An Unsettled phase past
// its deadline is resolved first, from the last reading it held, and latches
// interception off; a suspended cap then resumes when its quiet period has
// passed; the edge against that last reading is then handled with the rules
// of the current phase, so the resulting phase always follows the new reading.
func (m minimizeModel) observe(iconic bool, now uint64) (minimizeModel, minimizeDecision) {
	var decision minimizeDecision
	if m.phase == minimizeUnsettled && now >= m.settleDeadline {
		m.phase = minimizeShown
		if m.iconic {
			m.phase = minimizeMinimized
		}
		m.haloComposed = false
		m.latched = true
		decision.action = minimizeSettled
		decision.latchedNow = true
	}
	m.cap, decision.capResumed = m.cap.maybeResume(now)
	switch {
	case iconic == m.iconic:
		if !iconic {
			m.lastShownAt = now
		}
		return m, decision
	case iconic:
		decision.edge = minimizeShownToIconic
		decision.ageBound = now - m.lastShownAt
		m.iconic = true
		return m.shownToIconic(decision, now)
	default:
		decision.edge = minimizeIconicToShown
		m.iconic = false
		m.lastShownAt = now
		return m.iconicToShown(decision)
	}
}

// shownToIconic handles an observed minimize at now. In Shown it is
// intercepted only when proven prompt, not latched and below the cap; the skip
// reason follows the design's target-behavior order, unknown-age, then
// latched, then cap, so a late edge reports unknown-age even after the latch.
// A closed cap not yet suspended trips here, and an interception records its
// tick at the decision, so one whose own restore ends Unsettled still counts.
// The phase is Minimized until the own restore the caller performs next. In
// Priming it cancels the replay with the halo already composed, never restores
// again, and is not counted as an interception. Every external edge moves the
// quiet timer of a suspended cap before any skip; Unsettled only follows an
// interception, so it never coexists with a suspended cap.
func (m minimizeModel) shownToIconic(decision minimizeDecision, now uint64) (minimizeModel, minimizeDecision) {
	if m.phase == minimizeUnsettled {
		if m.expectIconic {
			decision.action = minimizeAbsorbed
		}
		return m, decision
	}
	if m.cap.suspended {
		m.cap, _ = m.cap.recordAttempt(now)
	}
	if m.phase == minimizePriming {
		m.phase = minimizeMinimized
		m.haloComposed = true
		m.replayAt = 0
		decision.action = minimizeCancelReplay
		return m, decision
	}
	m.phase = minimizeMinimized
	m.haloComposed = false
	decision.action = minimizeSkip
	switch {
	case decision.ageBound > minimizeLatenessBoundMS:
		decision.reason = minimizeReasonUnknownAge
	case m.latched:
		decision.reason = minimizeReasonLatched
	case m.cap.closed(now):
		decision.reason = minimizeReasonCap
		m.cap, decision.capTripped = m.cap.recordAttempt(now)
	default:
		decision.action = minimizeIntercept
		m.cap = m.cap.recordIntercept(now)
	}
	return m, decision
}

// iconicToShown handles an observed restore: honored from Minimized, absorbed
// as a late own restore while Unsettled on a shown expectation.
func (m minimizeModel) iconicToShown(decision minimizeDecision) (minimizeModel, minimizeDecision) {
	if m.phase == minimizeUnsettled {
		if !m.expectIconic {
			decision.action = minimizeAbsorbed
		}
		return m, decision
	}
	m.phase = minimizeShown
	m.haloComposed = false
	decision.action = minimizeRestoreHonored
	return m, decision
}

// ownCall records the after-reading of an own ShowWindow call, taken at now:
// the reading becomes the last one, so the own transition is never seen as an
// edge. An expected shown state starts Priming with the replay due at
// now + minimizeReplayDelayMS; an expected iconic state completes the replay
// with the halo composed; any mismatch is Unsettled until
// now + minimizeSettleTimeoutMS.
func (m minimizeModel) ownCall(expectIconic, afterIconic bool, now uint64) minimizeModel {
	m.iconic = afterIconic
	if !afterIconic {
		m.lastShownAt = now
	}
	m.replayAt = 0
	switch {
	case expectIconic != afterIconic:
		m.phase = minimizeUnsettled
		m.expectIconic = expectIconic
		m.settleDeadline = now + minimizeSettleTimeoutMS
		m.haloComposed = false
	case expectIconic:
		m.phase = minimizeMinimized
		m.haloComposed = true
	default:
		m.phase = minimizePriming
		m.replayAt = now + minimizeReplayDelayMS
	}
	return m
}

// replayDue reports whether the pending replay must run at now: only in
// Priming, while the window is shown, from the replay tick on.
func (m minimizeModel) replayDue(now uint64) bool {
	return m.phase == minimizePriming && !m.iconic && now >= m.replayAt
}

// showsMinimizedTrigger keeps the halo's minimized visibility trigger on in
// every phase except Shown.
func (m minimizeModel) showsMinimizedTrigger() bool {
	return m.phase != minimizeShown
}

// minimizeReadingName names a reading in the decision log lines.
func minimizeReadingName(iconic bool) string {
	if iconic {
		return "iconic"
	}
	return "shown"
}

// minimizeTransition filters WinEvent callbacks to the top-level target and
// tells the two system events apart. A matched event carries no decision: it
// is logged with its age and triggers one observation.
func minimizeTransition(event uint32, hwnd, target uintptr, idObject, idChild int32) (matched, starting bool) {
	if hwnd != target || idObject != objidWindow || idChild != childidSelf {
		return false, false
	}
	switch event {
	case eventSystemMinimizeStart:
		return true, true
	case eventSystemMinimizeEnd:
		return true, false
	default:
		return false, false
	}
}
