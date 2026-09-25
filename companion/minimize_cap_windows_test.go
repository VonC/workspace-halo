//go:build windows

package main

// Tests for the interception cap of minimize_windows.go (v0.0.24 step 3),
// fed with synthetic readings and own-call outcomes: two interceptions within
// the window pass, the third edge trips the cap once, a continuing stream of
// edges (prompt, late or latched) keeps it closed, only a quiet period without
// any edge resumes it, and a Priming cancellation never counts. They reuse
// primedMinimizeModel of minimize_windows_test.go.

import "testing"

// minimizeEdgeAfter observes the window shown at shownAt, then an external
// minimize at at; an intercept is followed by the own restore and the own
// replay, as the controller runs them. The decision reports a cap resume from
// either observation.
func minimizeEdgeAfter(m minimizeModel, shownAt, at uint64) (minimizeModel, minimizeDecision) {
	m, restore := m.observe(false, shownAt)
	m, decision := m.observe(true, at)
	if decision.action == minimizeIntercept {
		m = m.ownCall(false, false, at)
		m = m.ownCall(true, true, at)
	}
	decision.capResumed = decision.capResumed || restore.capResumed
	return m, decision
}

// trippedMinimizeModel intercepts prompt edges at 1020 and 1520, then trips
// the cap with a third prompt edge at 2020.
func trippedMinimizeModel(t *testing.T) minimizeModel {
	t.Helper()
	m := newMinimizeModel(false, 1000)
	var decision minimizeDecision
	for _, at := range []uint64{1020, 1520} {
		m, decision = minimizeEdgeAfter(m, at-20, at)
		if decision.action != minimizeIntercept {
			t.Fatalf("setup edge at %d = %+v, want intercept", at, decision)
		}
	}
	m, decision = minimizeEdgeAfter(m, 2000, 2020)
	if decision.action != minimizeSkip || decision.reason != minimizeReasonCap || !decision.capTripped {
		t.Fatalf("setup third edge = %+v, want skip cap and trip", decision)
	}
	return m
}

func TestMinimizeCapAllowsTwoInterceptionsWithinTheWindow(t *testing.T) {
	m := newMinimizeModel(false, 1000)
	for _, at := range []uint64{1020, 1520} {
		var decision minimizeDecision
		m, decision = minimizeEdgeAfter(m, at-20, at)
		if decision.action != minimizeIntercept || decision.capTripped {
			t.Fatalf("edge at %d = %+v, want intercept", at, decision)
		}
	}
	if m.cap.filled != 2 || m.cap.intercepts != [minimizeCapCount]uint64{1020, 1520} || m.cap.suspended {
		t.Fatalf("cap = %+v, want intercepts 1020 and 1520, not suspended", m.cap)
	}

	// The window is exclusive: a third edge 2000 ms after the first is intercepted.
	m, decision := minimizeEdgeAfter(m, 3000, 3020)
	if decision.action != minimizeIntercept || m.cap.intercepts != [minimizeCapCount]uint64{1520, 3020} {
		t.Fatalf("edge at 3020 = (%+v, %+v), want intercept, oldest dropped", m.cap, decision)
	}
}

func TestMinimizeCapTripsOnTheThirdEdgeWithinTwoSeconds(t *testing.T) {
	m := trippedMinimizeModel(t)
	if !m.cap.suspended || m.cap.lastAttemptAt != 2020 || m.phase != minimizeMinimized || m.haloComposed {
		t.Fatalf("tripped model = %+v, want suspended at 2020, Minimized without halo", m)
	}

	_, decision := minimizeEdgeAfter(m, 2500, 2520)
	if decision.action != minimizeSkip || decision.reason != minimizeReasonCap || decision.capTripped {
		t.Fatalf("fourth edge = %+v, want skip cap, trip reported once", decision)
	}

	// A late third edge keeps its unknown-age reason and does not trip the cap.
	late := newMinimizeModel(false, 1000)
	late, _ = minimizeEdgeAfter(late, 1000, 1020)
	late, _ = minimizeEdgeAfter(late, 1500, 1520)
	late, decision = minimizeEdgeAfter(late, 1600, 2200)
	if decision.reason != minimizeReasonUnknownAge || decision.capTripped || late.cap.suspended {
		t.Fatalf("late third edge = (%+v, %+v), want skip unknown-age, cap not suspended", late.cap, decision)
	}
}

func TestMinimizeCapStaysClosedDuringAContinuingStream(t *testing.T) {
	tripped := trippedMinimizeModel(t)
	latched := tripped
	latched.latched = true
	streams := []struct {
		name         string
		m            minimizeModel
		promptReason string
	}{
		{"unlatched", tripped, minimizeReasonCap},
		{"latched", latched, minimizeReasonLatched},
	}
	for _, stream := range streams {
		m := stream.m
		for k := uint64(1); k <= 20; k++ {
			at := 2020 + k*1000
			shownAt, want := at-20, stream.promptReason
			if k%2 == 0 {
				shownAt, want = at-900, minimizeReasonUnknownAge
			}
			var decision minimizeDecision
			m, decision = minimizeEdgeAfter(m, shownAt, at)
			if decision.action != minimizeSkip || decision.reason != want || decision.capResumed {
				t.Fatalf("%s: edge at %d = %+v, want skip %s and no resume", stream.name, at, decision, want)
			}
			if !m.cap.suspended || m.cap.lastAttemptAt != at {
				t.Fatalf("%s: cap after edge at %d = %+v, want suspended with the attempt at %d", stream.name, at, m.cap, at)
			}
		}
	}
}

func TestMinimizeCapResumesAfterFiveSecondsWithoutAnyEdge(t *testing.T) {
	const last = uint64(2020)
	tripped := trippedMinimizeModel(t)

	m, decision := tripped.observe(true, last+4999)
	if decision.capResumed || !m.cap.suspended {
		t.Fatalf("at last + 4999 = (%+v, %+v), want still suspended", m.cap, decision)
	}
	m, decision = m.observe(true, last+5000)
	if !decision.capResumed || m.cap != (minimizeCap{}) {
		t.Fatalf("at last + 5000 = (%+v, %+v), want resumed with a cleared history", m.cap, decision)
	}

	// A prompt edge at last + 4999 is skipped and moves the quiet timer.
	m, decision = minimizeEdgeAfter(tripped, last+4990, last+4999)
	if decision.reason != minimizeReasonCap || m.cap.lastAttemptAt != last+4999 {
		t.Fatalf("edge at last + 4999 = (%+v, %+v), want skip cap, attempt moved", m.cap, decision)
	}
	m, decision = m.observe(true, last+9998)
	if decision.capResumed || !m.cap.suspended {
		t.Fatalf("at last + 9998 = (%+v, %+v), want still suspended", m.cap, decision)
	}
	if _, decision = m.observe(true, last+9999); !decision.capResumed {
		t.Fatalf("at last + 9999 = %+v, want resumed", decision)
	}

	// A prompt edge at last + 5000 is intercepted after the resume.
	m, decision = minimizeEdgeAfter(tripped, last+4990, last+5000)
	if !decision.capResumed || decision.action != minimizeIntercept || m.cap.filled != 1 {
		t.Fatalf("edge at last + 5000 = (%+v, %+v), want resumed then intercepted", m.cap, decision)
	}
}

func TestMinimizeCapIgnoresPrimingCancellations(t *testing.T) {
	m, decision := primedMinimizeModel(t).observe(true, 1080)
	if decision.action != minimizeCancelReplay || m.cap.filled != 1 {
		t.Fatalf("cancellation = (%+v, %+v), want cancel-replay not counted", m.cap, decision)
	}

	// Counted as an interception, the cancellation would make this edge the third.
	m, decision = minimizeEdgeAfter(m, 1200, 1220)
	if decision.action != minimizeIntercept || m.cap.filled != 2 {
		t.Fatalf("edge at 1220 = (%+v, %+v), want intercept", m.cap, decision)
	}
}
