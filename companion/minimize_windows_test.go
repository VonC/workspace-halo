//go:build windows

package main

// Tests for the pure minimize interception logic of minimize_windows.go: the
// WinEvent filter on the tracked top-level window, and the v0.0.24
// observation model fed with synthetic (iconic, now) readings and own-call
// outcomes. The event-order tests went with the event-order transition.
//
// v0.0.24 step 3 took this file past 550 lines, so the fuzz target
// FuzzMinimizeModelObservations moved to minimize_fuzz_windows_test.go and
// the interception cap tests went to minimize_cap_windows_test.go, which
// reuses primedMinimizeModel below.

import "testing"

func TestMinimizeTransitionTargetsOnlyTheTrackedTopLevelWindow(t *testing.T) {
	const target = uintptr(0x1234)
	tests := []struct {
		name              string
		event             uint32
		hwnd              uintptr
		idObject, idChild int32
		wantMatched       bool
		wantStarting      bool
	}{
		{"start", eventSystemMinimizeStart, target, objidWindow, childidSelf, true, true},
		{"end", eventSystemMinimizeEnd, target, objidWindow, childidSelf, true, false},
		{"other window", eventSystemMinimizeStart, 0x5678, objidWindow, childidSelf, false, false},
		{"child object", eventSystemMinimizeStart, target, objidWindow, 1, false, false},
		{"other event", 0x0003, target, objidWindow, childidSelf, false, false},
	}
	for _, test := range tests {
		matched, starting := minimizeTransition(test.event, test.hwnd, target, test.idObject, test.idChild)
		if matched != test.wantMatched || starting != test.wantStarting {
			t.Errorf(
				"%s: minimizeTransition() = (%t, %t), want (%t, %t)",
				test.name, matched, starting, test.wantMatched, test.wantStarting,
			)
		}
	}
}

// primedMinimizeModel returns a model intercepted at 1020 and restored as
// expected at 1030, so it is Priming with the replay due at 1105.
func primedMinimizeModel(t *testing.T) minimizeModel {
	t.Helper()
	m := newMinimizeModel(false, 1000)
	m, decision := m.observe(true, 1020)
	if decision.action != minimizeIntercept {
		t.Fatalf("setup intercept = %v, want intercept", decision.action)
	}
	return m.ownCall(false, false, 1030)
}

func TestMinimizeModelSeedsPhaseFromTheFirstReading(t *testing.T) {
	shown := newMinimizeModel(false, 1000)
	if shown.phase != minimizeShown || shown.iconic || shown.lastShownAt != 1000 {
		t.Fatalf("shown seed = %+v, want Shown since 1000", shown)
	}

	iconic := newMinimizeModel(true, 1000)
	if iconic.phase != minimizeMinimized || !iconic.iconic || iconic.haloComposed {
		t.Fatalf("iconic seed = %+v, want Minimized without halo", iconic)
	}
	next, decision := iconic.observe(true, 1025)
	if decision.edge != minimizeNoEdge || decision.action != minimizeNoAction || next != iconic {
		t.Fatalf("iconic seed observation = (%+v, %+v), want no edge and no change", next, decision)
	}
}

func TestMinimizeModelInterceptsAPromptShownToIconicEdge(t *testing.T) {
	m, decision := newMinimizeModel(false, 1000).observe(true, 1020)

	if decision.edge != minimizeShownToIconic || decision.action != minimizeIntercept {
		t.Fatalf("decision = %+v, want shown->iconic intercept", decision)
	}
	if decision.ageBound != 20 || decision.reason != "" {
		t.Fatalf("age bound = %d reason %q, want 20 and no reason", decision.ageBound, decision.reason)
	}
	if m.phase != minimizeMinimized || !m.iconic {
		t.Fatalf("model = %+v, want Minimized until the own restore", m)
	}
}

func TestMinimizeModelSkipsAnEdgeWhoseAgeBoundExceedsTheLatenessBound(t *testing.T) {
	tests := []struct {
		name       string
		at         uint64
		wantAction minimizeAction
		wantReason string
	}{
		{"stalled 3.4 s", 4400, minimizeSkip, minimizeReasonUnknownAge},
		{"bound 501", 1501, minimizeSkip, minimizeReasonUnknownAge},
		{"bound 500", 1500, minimizeIntercept, ""},
	}
	for _, test := range tests {
		m, decision := newMinimizeModel(false, 1000).observe(true, test.at)
		if decision.action != test.wantAction || decision.reason != test.wantReason {
			t.Errorf("%s: decision = %+v, want %v %q", test.name, decision, test.wantAction, test.wantReason)
		}
		if m.phase != minimizeMinimized || m.haloComposed {
			t.Errorf("%s: model = %+v, want Minimized without halo", test.name, m)
		}
	}
}

func TestMinimizeModelAbsorbsItsOwnRestoreAndReplay(t *testing.T) {
	m := primedMinimizeModel(t)
	if m.phase != minimizePriming || m.iconic || m.lastShownAt != 1030 || m.replayAt != 1105 {
		t.Fatalf("after own restore = %+v, want Priming shown since 1030, replay at 1105", m)
	}
	m, decision := m.observe(false, 1055)
	if decision.edge != minimizeNoEdge || m.phase != minimizePriming {
		t.Fatalf("observation after own restore = (%+v, %+v), want no edge", m, decision)
	}

	m = m.ownCall(true, true, 1110)
	if m.phase != minimizeMinimized || !m.iconic || !m.haloComposed || m.replayAt != 0 {
		t.Fatalf("after own replay = %+v, want Minimized with halo", m)
	}
	m, decision = m.observe(true, 1135)
	if decision.edge != minimizeNoEdge || m.phase != minimizeMinimized {
		t.Fatalf("observation after own replay = (%+v, %+v), want no edge", m, decision)
	}
}

func TestMinimizeModelRepeatedReadingsAreNotEdges(t *testing.T) {
	m := primedMinimizeModel(t)
	for _, now := range []uint64{1040, 1040, 1060, 1090} {
		next, decision := m.observe(false, now)
		want := m
		want.lastShownAt = now
		if decision != (minimizeDecision{}) || next != want {
			t.Fatalf("repeat at %d = (%+v, %+v), want only lastShownAt=%d", now, next, decision, now)
		}
		m = next
	}
}

func TestMinimizeModelReplayIsDueOnlyWhileShownInPriming(t *testing.T) {
	m := primedMinimizeModel(t)
	if m.replayDue(1104) {
		t.Fatal("replay due at 1104, want not before restore tick + 75")
	}
	if !m.replayDue(1105) || !m.replayDue(1200) {
		t.Fatal("replay not due from 1105, want due at restore tick + 75")
	}

	cancelled, _ := m.observe(true, 1080)
	if cancelled.replayDue(1200) {
		t.Fatal("replay due while iconic, want only while shown in Priming")
	}
	if newMinimizeModel(false, 1000).replayDue(5000) {
		t.Fatal("replay due in Shown, want only in Priming")
	}
}

func TestMinimizeModelEdgeDuringPrimingCancelsTheReplayWithoutRestore(t *testing.T) {
	m, decision := primedMinimizeModel(t).observe(true, 1080)

	if decision.edge != minimizeShownToIconic || decision.action != minimizeCancelReplay {
		t.Fatalf("decision = %+v, want shown->iconic cancel-replay", decision)
	}
	if decision.ageBound != 50 {
		t.Fatalf("age bound = %d, want 50 from the own restore reading", decision.ageBound)
	}
	if m.phase != minimizeMinimized || !m.haloComposed || m.replayAt != 0 {
		t.Fatalf("model = %+v, want Minimized with halo and no replay", m)
	}
}

func TestMinimizeModelHonorsARestoreFromMinimized(t *testing.T) {
	m := primedMinimizeModel(t).ownCall(true, true, 1110)

	m, decision := m.observe(false, 4000)

	if decision.edge != minimizeIconicToShown || decision.action != minimizeRestoreHonored {
		t.Fatalf("decision = %+v, want iconic->shown restore-honored", decision)
	}
	if m.phase != minimizeShown || m.iconic || m.haloComposed || m.lastShownAt != 4000 {
		t.Fatalf("model = %+v, want Shown since 4000", m)
	}
	if m.showsMinimizedTrigger() {
		t.Fatal("Shown keeps the minimized trigger, want it off")
	}
}

func TestMinimizeModelUnsettledOwnCallResolvesAtTheDeadlineAndLatches(t *testing.T) {
	// Arrange: the own replay at 1110 leaves the window shown.
	m := primedMinimizeModel(t).ownCall(true, false, 1110)
	if m.phase != minimizeUnsettled || !m.expectIconic || m.settleDeadline != 2110 || m.replayAt != 0 {
		t.Fatalf("after unsettled replay = %+v, want Unsettled until 2110", m)
	}
	if !m.showsMinimizedTrigger() {
		t.Fatal("Unsettled drops the minimized trigger, want it on")
	}

	// Act and assert: edges before the deadline are absorbed or only logged.
	steps := []struct {
		iconic     bool
		at         uint64
		wantAction minimizeAction
	}{
		{true, 1500, minimizeAbsorbed},
		{false, 1600, minimizeNoAction},
		{false, 1700, minimizeNoAction},
	}
	for _, step := range steps {
		var decision minimizeDecision
		m, decision = m.observe(step.iconic, step.at)
		if decision.action != step.wantAction || decision.latchedNow || m.phase != minimizeUnsettled {
			t.Fatalf("at %d: (%+v, %+v), want %v while Unsettled", step.at, m, decision, step.wantAction)
		}
	}

	m, decision := m.observe(false, 2110)
	if decision.action != minimizeSettled || !decision.latchedNow || decision.edge != minimizeNoEdge {
		t.Fatalf("deadline decision = %+v, want settled and latched", decision)
	}
	if m.phase != minimizeShown || !m.latched {
		t.Fatalf("deadline model = %+v, want Shown from the reading and latched", m)
	}
}

func TestMinimizeModelUnsettledRestoreAbsorbsTheLateRestore(t *testing.T) {
	m := newMinimizeModel(false, 1000)
	m, _ = m.observe(true, 1020)
	m = m.ownCall(false, true, 1030)
	if m.phase != minimizeUnsettled || m.expectIconic || !m.iconic {
		t.Fatalf("after unsettled restore = %+v, want Unsettled expecting shown", m)
	}

	m, decision := m.observe(false, 1300)
	if decision.edge != minimizeIconicToShown || decision.action != minimizeAbsorbed {
		t.Fatalf("late restore = %+v, want absorbed", decision)
	}
	m, decision = m.observe(true, 1400)
	if decision.edge != minimizeShownToIconic || decision.action != minimizeNoAction {
		t.Fatalf("minimize while expecting shown = %+v, want only logged", decision)
	}

	m, decision = m.observe(false, 2030)
	if !decision.latchedNow || decision.edge != minimizeIconicToShown {
		t.Fatalf("deadline with restore = %+v, want latched then iconic->shown", decision)
	}
	if decision.action != minimizeRestoreHonored || m.phase != minimizeShown || !m.latched {
		t.Fatalf("deadline with restore = (%+v, %+v), want restore-honored to Shown", m, decision)
	}
}

func TestMinimizeModelLatchedSkipsEveryLaterEdge(t *testing.T) {
	m := primedMinimizeModel(t).ownCall(true, false, 1110)
	m, _ = m.observe(false, 2110)

	for _, at := range []uint64{4610, 9110, 92110} {
		m, _ = m.observe(false, at-20)
		var decision minimizeDecision
		m, decision = m.observe(true, at)
		if decision.action != minimizeSkip || decision.reason != minimizeReasonLatched {
			t.Fatalf("edge at %d = %+v, want skip latched", at, decision)
		}
		if m.phase != minimizeMinimized || m.haloComposed {
			t.Fatalf("model at %d = %+v, want Minimized without halo", at, m)
		}
		m, _ = m.observe(false, at+500)
	}

	// An edge whose age bound exceeds the lateness bound keeps the unknown-age
	// reason after the latch too: the plan's precedence (unknown-age, then
	// latched, then cap) follows the design's target-behavior order, and
	// either reason skips the edge.
	m, decision := m.observe(true, 92610+minimizeLatenessBoundMS+1)
	if decision.action != minimizeSkip || decision.reason != minimizeReasonUnknownAge || !m.latched {
		t.Fatalf("late edge after the latch = (%+v, %+v), want skip unknown-age, still latched", m, decision)
	}
}

func TestMinimizeModelNamesItsEdgesActionsAndStates(t *testing.T) {
	edges := map[minimizeEdge]string{
		minimizeNoEdge:        "none",
		minimizeShownToIconic: "shown->iconic",
		minimizeIconicToShown: "iconic->shown",
	}
	for edge, want := range edges {
		if got := edge.String(); got != want {
			t.Errorf("edge %d = %q, want %q", edge, got, want)
		}
	}
	actions := map[minimizeAction]string{
		minimizeNoAction:       "none",
		minimizeIntercept:      "intercept",
		minimizeCancelReplay:   "cancel-replay",
		minimizeSkip:           "skip",
		minimizeRestoreHonored: "restore-honored",
		minimizeAbsorbed:       "absorbed",
		minimizeSettled:        "settled",
	}
	for action, want := range actions {
		if got := action.String(); got != want {
			t.Errorf("action %d = %q, want %q", action, got, want)
		}
	}
	if minimizeReadingName(true) != "iconic" || minimizeReadingName(false) != "shown" {
		t.Error("state names, want iconic and shown")
	}
}
