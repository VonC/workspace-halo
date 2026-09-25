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
)

const (
	minimizeReplayDelayMS   = uint64(75)
	minimizeLatenessBoundMS = uint64(500)
	minimizeSettleTimeoutMS = uint64(1000)
)

// minimizeModel is the whole interception state of one window: the phase, the
// last IsIconic reading, the tick of the last shown reading, the pending
// replay, the pending own-call expectation and the session latch.
type minimizeModel struct {
	phase          minimizePhase
	iconic         bool
	lastShownAt    uint64
	replayAt       uint64
	haloComposed   bool
	expectIconic   bool
	settleDeadline uint64
	latched        bool
}

// minimizeDecision reports one observation: the edge found, its age bound (for
// a shown-to-iconic edge), the action, the skip reason, and whether this
// observation resolved an Unsettled phase and latched interception off.
type minimizeDecision struct {
	edge       minimizeEdge
	ageBound   uint64
	action     minimizeAction
	reason     string
	latchedNow bool
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
// interception off; the edge against that last reading is then handled with
// the rules of the current phase, so the resulting phase always follows the
// new reading.
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
		return m.shownToIconic(decision)
	default:
		decision.edge = minimizeIconicToShown
		m.iconic = false
		m.lastShownAt = now
		return m.iconicToShown(decision)
	}
}

// shownToIconic handles an observed minimize. In Shown it is intercepted only
// when proven prompt and not latched; the skip reason follows the design's
// target-behavior order, unknown-age before latched, so a late edge reports
// unknown-age even after the latch. The phase is Minimized until the own
// restore the caller performs next. In Priming it cancels the replay with the
// halo already composed, and never restores again.
func (m minimizeModel) shownToIconic(decision minimizeDecision) (minimizeModel, minimizeDecision) {
	switch m.phase {
	case minimizeUnsettled:
		if m.expectIconic {
			decision.action = minimizeAbsorbed
		}
		return m, decision
	case minimizePriming:
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
	default:
		decision.action = minimizeIntercept
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
