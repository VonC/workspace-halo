//go:build windows

package main

// The scenario runner of the v0.0.24 minimize acceptance tests (step 4),
// split out of minimize_acceptance_windows_test.go to keep both files below
// the line ceiling. It drives one or more minimizeControllers on a shared
// fake clock. At each millisecond it applies the scripted external steps of
// that tick, then, unless the host thread is stalled, delivers the due
// minimize WinEvents and runs the 25 ms polling tick on every controller.
// Every change of a window's minimized state, own calls included, generates a
// WinEvent whose delivery lag comes from the scenario's lag list, so events
// can arrive late, reordered or never. It reuses fakeMinimizeWindow of
// minimize_hook_windows_test.go.

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

// minimizeEventLost is a delivery lag that drops the event.
const minimizeEventLost = ^uint64(0)

// minimizeScenarioKind is the external action of a scenario step.
type minimizeScenarioKind uint8

const (
	minimizeStepMinimize      minimizeScenarioKind = iota // Windows or the user minimizes the window
	minimizeStepRestore                                   // the user restores the window
	minimizeStepClick                                     // a click activates the shown window
	minimizeStepDefer                                     // own ShowWindow calls with command value no longer apply
	minimizeStepApplyDeferred                             // the last deferred own call applies, late
	minimizeStepStall                                     // the host thread is busy for value ms
	minimizeStepRestart                                   // a new host starts on the window
)

// minimizeScenarioStep is one external action at tick at on window window.
type minimizeScenarioStep struct {
	at     uint64
	window int
	kind   minimizeScenarioKind
	value  uint64
}

func scenarioMinimize(at uint64) minimizeScenarioStep {
	return minimizeScenarioStep{at: at, kind: minimizeStepMinimize}
}

func scenarioRestore(at uint64) minimizeScenarioStep {
	return minimizeScenarioStep{at: at, kind: minimizeStepRestore}
}

// minimizeScenarioEvent is a minimize WinEvent generated at generatedAt and
// due for delivery at dueAt.
type minimizeScenarioEvent struct {
	window      int
	starting    bool
	generatedAt uint64
	dueAt       uint64
}

// minimizeScenarioWindow is a fakeMinimizeWindow that reports every change of
// its minimized state as a WinEvent, and records the tick of each own restore
// command and of its first delivered event.
type minimizeScenarioWindow struct {
	*fakeMinimizeWindow
	scenario        *minimizeScenario
	index           int
	emitted         int
	restoreTicks    []uint64
	firstDeliveryAt uint64
}

func (w *minimizeScenarioWindow) showWindow(cmd uintptr) {
	if cmd != swMinimize {
		w.restoreTicks = append(w.restoreTicks, *w.clock)
	}
	w.change(func() { w.fakeMinimizeWindow.showWindow(cmd) })
}

// change applies a state change and generates the WinEvent of a change of the
// minimized state.
func (w *minimizeScenarioWindow) change(apply func()) {
	was := w.iconic
	apply()
	if w.iconic != was {
		w.scenario.emit(w, w.iconic)
	}
}

// minimizeScenario is the shared clock, the windows with their hosts and logs,
// and the WinEvents not delivered yet.
type minimizeScenario struct {
	clock        uint64
	start        uint64
	next         uint64
	stalledUntil uint64
	lags         []uint64
	windows      []*minimizeScenarioWindow
	controllers  []*minimizeController
	outputs      []*bytes.Buffer
	pending      []minimizeScenarioEvent
}

// newMinimizeScenario starts count hosts on shown windows at 1000. The n-th
// event of window i is delivered lags[(n+i) % len(lags)] ms after it was
// generated; no lag list loses every event.
func newMinimizeScenario(count int, lags []uint64) *minimizeScenario {
	s := &minimizeScenario{clock: 1000, start: 1000, next: 1000, lags: lags}
	for i := 0; i < count; i++ {
		s.windows = append(s.windows, &minimizeScenarioWindow{
			fakeMinimizeWindow: &fakeMinimizeWindow{clock: &s.clock, deferred: map[uintptr]bool{}},
			scenario:           s,
			index:              i,
		})
		s.outputs = append(s.outputs, &bytes.Buffer{})
		s.controllers = append(s.controllers, nil)
		s.startHost(i)
	}
	return s
}

// startHost starts a host on window i: a fresh controller seeded from one
// reading at the current tick.
func (s *minimizeScenario) startHost(i int) {
	s.controllers[i] = newMinimizeController(s.windows[i], log.New(s.outputs[i], "", 0), func() uint64 { return s.clock })
}

func (s *minimizeScenario) emit(w *minimizeScenarioWindow, starting bool) {
	lag := minimizeEventLost
	if len(s.lags) > 0 {
		lag = s.lags[(w.emitted+w.index)%len(s.lags)]
	}
	w.emitted++
	if lag == minimizeEventLost {
		return
	}
	s.pending = append(s.pending, minimizeScenarioEvent{w.index, starting, s.clock, s.clock + lag})
}

// run continues the scenario through until, applying steps listed in tick
// order: at each millisecond the steps of that tick, then, unless the host
// thread is stalled, the due events and the 25 ms polling tick.
func (s *minimizeScenario) run(t *testing.T, steps []minimizeScenarioStep, until uint64) {
	t.Helper()
	next := 0
	for now := s.next; now <= until; now++ {
		s.clock = now
		for ; next < len(steps) && steps[next].at <= now; next++ {
			if steps[next].at < now {
				t.Fatalf("step %d at %d is out of tick order (now %d)", next, steps[next].at, now)
			}
			s.apply(steps[next], now)
		}
		if now < s.stalledUntil {
			continue
		}
		s.deliverDue(now)
		if now > s.start && (now-s.start)%25 == 0 {
			for _, controller := range s.controllers {
				controller.observe(now)
			}
		}
	}
	if next < len(steps) {
		t.Fatalf("step %d at %d is after the scenario end %d", next, steps[next].at, until)
	}
	s.next = until + 1
}

func (s *minimizeScenario) apply(step minimizeScenarioStep, now uint64) {
	w := s.windows[step.window]
	switch step.kind {
	case minimizeStepMinimize:
		w.change(func() { w.iconic = true })
	case minimizeStepRestore:
		w.change(func() { w.iconic = false })
	case minimizeStepClick:
		// A click activates the shown window: nothing the host observes changes.
	case minimizeStepDefer:
		w.deferred[uintptr(step.value)] = true
	case minimizeStepApplyDeferred:
		w.change(w.applyDeferred)
	case minimizeStepStall:
		s.stalledUntil = now + step.value
	case minimizeStepRestart:
		// The new host's own calls apply normally.
		w.deferred = map[uintptr]bool{}
		s.startHost(step.window)
	}
}

// deliverDue delivers, in generation order, every event due by now, including
// the events of the own calls those deliveries cause.
func (s *minimizeScenario) deliverDue(now uint64) {
	for i := 0; i < len(s.pending); {
		event := s.pending[i]
		if event.dueAt > now {
			i++
			continue
		}
		s.pending = append(s.pending[:i], s.pending[i+1:]...)
		w := s.windows[event.window]
		if w.firstDeliveryAt == 0 {
			w.firstDeliveryAt = now
		}
		s.controllers[event.window].onEvent(event.starting, uint32(now-event.generatedAt), now)
	}
}

// requireOrderedLog checks that the log of window i holds the lines in this
// order.
func (s *minimizeScenario) requireOrderedLog(t *testing.T, i int, lines ...string) {
	t.Helper()
	text := s.outputs[i].String()
	from := 0
	for _, line := range lines {
		at := strings.Index(text[from:], line)
		if at < 0 {
			t.Fatalf("log lacks %q after offset %d:\n%s", line, from, text)
		}
		from += at + len(line)
	}
}
