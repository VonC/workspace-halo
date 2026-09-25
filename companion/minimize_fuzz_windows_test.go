//go:build windows

package main

// The property test of the pure minimize model of minimize_windows.go:
// FuzzMinimizeModelObservations decodes bytes into readings and own-call
// outcomes, then checks the phase, intercept and Unsettled invariants and,
// since v0.0.24 step 3, the interception cap invariants through
// minimizeFuzzCapTrace. Its f.Add seeds run under plain go test. It moved out
// of minimize_windows_test.go when step 3 took that file past 550 lines.

import "testing"

// minimizeFuzzStep encodes one fuzz observation: the ticks elapsed since the
// previous one (in 25 ms units, 0 to 63), the reading, and whether an own call
// made during that step ends on the opposite of its expected state.
func minimizeFuzzStep(ticks25 uint8, iconic, ownMismatch bool) byte {
	b := ticks25 << 2
	if ownMismatch {
		b |= 2
	}
	if iconic {
		b |= 1
	}
	return b
}

func minimizeFuzzSeed(steps ...byte) []byte {
	return append([]byte{0}, steps...)
}

func FuzzMinimizeModelObservations(f *testing.F) {
	s := minimizeFuzzStep
	// Recorded reordered sequence: a prompt minimize, the replay, then the
	// late events re-observing the unchanged iconic window for seconds.
	f.Add(minimizeFuzzSeed(s(1, false, false), s(1, true, false), s(1, false, false),
		s(1, false, false), s(1, false, false), s(1, true, false), s(63, true, false),
		s(63, true, false), s(40, true, false)))
	// Stalled thread: a minimize first observed 3.4 s after the last shown reading.
	f.Add(minimizeFuzzSeed(s(1, false, false), s(63, true, false), s(62, true, false)))
	// Unsettled replay that applies late, after the deadline, then more edges.
	f.Add(minimizeFuzzSeed(s(1, true, false), s(3, false, true), s(20, false, false),
		s(30, false, false), s(1, true, false), s(4, false, false), s(1, true, false)))
	// User restore during Priming, then a second minimize.
	f.Add(minimizeFuzzSeed(s(1, true, false), s(1, false, false), s(1, true, false),
		s(2, false, false), s(1, true, false)))
	// Unsettled restore: the window stays iconic.
	f.Add(minimizeFuzzSeed(s(1, true, true), s(10, true, false), s(40, true, false),
		s(1, false, false), s(1, true, false)))
	// Minimize, restore, minimize within 2 s.
	f.Add(minimizeFuzzSeed(s(1, true, false), s(4, false, false), s(10, false, false),
		s(1, true, false), s(4, false, false)))
	// Edges every second for 20 s.
	f.Add(minimizeFuzzSeed(s(40, true, false), s(1, false, false), s(40, true, false),
		s(1, false, false), s(40, true, false), s(1, false, false), s(40, true, false)))
	// Host start on an iconic window, then restore and a prompt minimize.
	f.Add([]byte{1, s(2, true, false), s(2, false, false), s(1, true, false), s(3, false, false)})
	// Late events during Priming: repeated shown readings until the replay.
	f.Add(minimizeFuzzSeed(s(1, true, false), s(0, false, false), s(1, false, false),
		s(1, false, false), s(1, false, false), s(1, true, false)))
	// Cap trip: three intercept cycles within 375 ms, a late and a prompt edge
	// while suspended, then 6.3 s of shown readings and a resumed intercept.
	f.Add(minimizeFuzzSeed(s(1, true, false), s(3, false, false), s(1, false, false),
		s(1, true, false), s(3, false, false), s(1, false, false), s(1, true, false),
		s(1, false, false), s(40, true, false), s(1, false, false), s(1, true, false),
		s(1, false, false), s(63, false, false), s(63, false, false), s(63, false, false),
		s(63, false, false), s(1, true, false)))
	// Cap stream: edges every second for 10 s right after a trip.
	f.Add(minimizeFuzzSeed(s(1, true, false), s(3, false, false), s(1, false, false),
		s(1, true, false), s(3, false, false), s(1, false, false), s(1, true, false),
		s(39, false, false), s(1, true, false), s(39, false, false), s(1, true, false),
		s(39, false, false), s(1, true, false), s(39, false, false), s(1, true, false),
		s(39, false, false), s(1, true, false), s(63, true, false), s(63, true, false),
		s(63, true, false), s(63, true, false)))

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 {
			return
		}
		now := uint64(1000)
		m := newMinimizeModel(data[0]&1 == 1, now)
		var trace minimizeFuzzCapTrace
		for i, b := range data[1:] {
			now += uint64(b>>2) * 25
			prev := m
			next, decision := m.observe(b&1 == 1, now)
			m = next
			checkMinimizeFuzzObservation(t, i, prev, m, decision, now)
			trace.check(t, i, prev, m, decision, now)
			mismatch := b&2 == 2
			if decision.action == minimizeIntercept {
				// Exactly one own restore follows every intercept.
				m = m.ownCall(false, mismatch, now)
				checkMinimizeFuzzPhase(t, i, m)
			}
			if m.replayDue(now) {
				m = m.ownCall(true, !mismatch, now)
				checkMinimizeFuzzPhase(t, i, m)
			}
		}
	})
}

func checkMinimizeFuzzObservation(t *testing.T, i int, prev, m minimizeModel, decision minimizeDecision, now uint64) {
	t.Helper()
	checkMinimizeFuzzPhase(t, i, m)
	if decision.action == minimizeIntercept {
		if prev.phase != minimizeShown || prev.latched || decision.edge != minimizeShownToIconic ||
			decision.ageBound > minimizeLatenessBoundMS {
			t.Fatalf("step %d: intercept from %+v with %+v", i, prev, decision)
		}
	}
	if m.phase == minimizeUnsettled && now >= m.settleDeadline {
		t.Fatalf("step %d: Unsettled survives its deadline %d at %d", i, m.settleDeadline, now)
	}
	if prev.latched && !m.latched {
		t.Fatalf("step %d: latch released", i)
	}
}

// minimizeFuzzCapTrace follows the cap from outside the model: the ticks of
// the last minimizeCapCount intercepts, oldest first, and of the latest
// external shown-to-iconic edge.
type minimizeFuzzCapTrace struct {
	intercepts []uint64
	lastEdgeAt uint64
}

// check asserts the cap invariants of one observation: no resume less than
// minimizeCapQuietMS after the latest external edge, no intercept while
// suspended, no three intercepts within minimizeCapWindowMS, and a suspended
// cap timing its quiet period from the latest external edge.
func (trace *minimizeFuzzCapTrace) check(t *testing.T, i int, prev, m minimizeModel, decision minimizeDecision, now uint64) {
	t.Helper()
	if decision.capResumed && now-trace.lastEdgeAt < minimizeCapQuietMS {
		t.Fatalf("step %d: resumed at %d, edge at %d", i, now, trace.lastEdgeAt)
	}
	if decision.edge == minimizeShownToIconic && decision.action != minimizeAbsorbed {
		trace.lastEdgeAt = now
	}
	if decision.action == minimizeIntercept {
		if prev.cap.suspended && !decision.capResumed {
			t.Fatalf("step %d: intercept while suspended: %+v", i, prev.cap)
		}
		if len(trace.intercepts) == minimizeCapCount && now-trace.intercepts[0] < minimizeCapWindowMS {
			t.Fatalf("step %d: third intercept at %d after %v", i, now, trace.intercepts)
		}
		trace.intercepts = append(trace.intercepts, now)
		if len(trace.intercepts) > minimizeCapCount {
			trace.intercepts = trace.intercepts[1:]
		}
	}
	if m.cap.suspended && m.cap.lastAttemptAt != trace.lastEdgeAt {
		t.Fatalf("step %d: suspended cap attempt at %d, latest edge at %d", i, m.cap.lastAttemptAt, trace.lastEdgeAt)
	}
}

func checkMinimizeFuzzPhase(t *testing.T, i int, m minimizeModel) {
	t.Helper()
	switch m.phase {
	case minimizeMinimized:
		if !m.iconic {
			t.Fatalf("step %d: Minimized with a shown reading: %+v", i, m)
		}
	case minimizeShown, minimizePriming:
		if m.iconic {
			t.Fatalf("step %d: %d with an iconic reading: %+v", i, m.phase, m)
		}
	}
}
