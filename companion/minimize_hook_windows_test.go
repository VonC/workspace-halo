//go:build windows

package main

// Controller tests for minimize_hook_windows.go: a minimizeController driven
// through a scripted fakeMinimizeWindow and a fake clock, with its decision
// log written to a buffer. They check the v0.0.24 own-call handling: one
// restore per interception, the single activating fallback, no composition
// after an unsettled restore, own calls stamped after their after-reading,
// late events that only log, and the session latch after an unsettled replay.
// v0.0.24 step 3 adds the interception cap log lines: the trip on the third
// edge within 2 s, and the resume after 5 s without any edge.

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"
)

// fakeMinimizeWindow is a scripted target window. A showWindow command
// advances the fake clock by showDelayMS, then applies its state change at
// once, or stores it when the command is listed in deferred.
type fakeMinimizeWindow struct {
	iconic        bool
	clock         *uint64
	showDelayMS   uint64
	deferred      map[uintptr]bool
	pending       bool
	pendingIconic bool
	commands      []uintptr
	compositions  int
	composeErr    error
}

func (w *fakeMinimizeWindow) isIconic() bool {
	return w.iconic
}

func (w *fakeMinimizeWindow) showWindow(cmd uintptr) {
	w.commands = append(w.commands, cmd)
	*w.clock += w.showDelayMS
	if w.deferred[cmd] {
		w.pending = true
		w.pendingIconic = cmd == swMinimize
		return
	}
	w.iconic = cmd == swMinimize
}

func (w *fakeMinimizeWindow) composeHalo() error {
	w.compositions++
	return w.composeErr
}

// applyDeferred applies the last deferred command, as a late own call would.
func (w *fakeMinimizeWindow) applyDeferred() {
	if w.pending {
		w.iconic = w.pendingIconic
		w.pending = false
	}
}

func (w *fakeMinimizeWindow) count(cmd uintptr) int {
	n := 0
	for _, command := range w.commands {
		if command == cmd {
			n++
		}
	}
	return n
}

type minimizeControllerFixture struct {
	controller *minimizeController
	window     *fakeMinimizeWindow
	output     *bytes.Buffer
	clock      *uint64
}

// newMinimizeControllerFixture seeds a controller on a shown window at 1000.
func newMinimizeControllerFixture() minimizeControllerFixture {
	clock := uint64(1000)
	window := &fakeMinimizeWindow{clock: &clock, deferred: map[uintptr]bool{}}
	var output bytes.Buffer
	controller := newMinimizeController(window, log.New(&output, "", 0), func() uint64 { return clock })
	return minimizeControllerFixture{controller, window, &output, &clock}
}

// externalMinimize minimizes the window from outside and observes at at.
func (f minimizeControllerFixture) externalMinimize(at uint64) {
	*f.clock = at
	f.window.iconic = true
	f.controller.observe(at)
}

// externalRestore restores the window from outside and observes at at.
func (f minimizeControllerFixture) externalRestore(at uint64) {
	*f.clock = at
	f.window.iconic = false
	f.controller.observe(at)
}

// tickUntil runs 25 ms ticks while the next one is at or before until.
func (f minimizeControllerFixture) tickUntil(until uint64) {
	for *f.clock+25 <= until {
		*f.clock += 25
		f.controller.observe(*f.clock)
	}
}

func (f minimizeControllerFixture) requireLog(t *testing.T, lines ...string) {
	t.Helper()
	for _, line := range lines {
		if !strings.Contains(f.output.String(), line) {
			t.Fatalf("log lacks %q:\n%s", line, f.output.String())
		}
	}
}

func (f minimizeControllerFixture) forbidLog(t *testing.T, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if strings.Contains(f.output.String(), fragment) {
			t.Fatalf("log holds %q:\n%s", fragment, f.output.String())
		}
	}
}

func TestMinimizeControllerInterceptsOnceAndReplays(t *testing.T) {
	f := newMinimizeControllerFixture()

	f.externalMinimize(1020)
	if f.window.count(swShowNoActivate) != 1 || f.window.iconic || f.window.compositions != 1 {
		t.Fatalf("after intercept: commands %v, iconic %t, compositions %d", f.window.commands, f.window.iconic, f.window.compositions)
	}
	if f.controller.model.phase != minimizePriming {
		t.Fatalf("phase = %d, want Priming", f.controller.model.phase)
	}
	f.tickUntil(1300)

	if f.window.count(swShowNoActivate) != 1 || f.window.count(swRestore) != 0 || f.window.count(swMinimize) != 1 {
		t.Fatalf("commands = %v, want one restore and one replay", f.window.commands)
	}
	if !f.window.iconic || f.controller.model.phase != minimizeMinimized || !f.controller.model.haloComposed {
		t.Fatalf("model = %+v, want Minimized with halo", f.controller.model)
	}
	f.requireLog(t,
		"minimize edge: shown->iconic age<=20ms action=intercept",
		"own restore: before=iconic after=shown fallback=false",
		"minimize intercepted: restored=true replay-in=75ms",
		"minimize replay requested after halo composition",
		"own replay: before=shown after=iconic",
		"minimize replay accepted with composed halo",
	)
	if strings.Count(f.output.String(), "minimize intercepted") != 1 {
		t.Fatalf("log holds more than one interception:\n%s", f.output.String())
	}
}

func TestMinimizeControllerLateEventsCauseNoRestore(t *testing.T) {
	f := newMinimizeControllerFixture()
	f.externalMinimize(1020)
	// A MinimizeStart generated before the priming restore, delivered in Priming.
	f.controller.onEvent(true, 40, 1030)
	f.tickUntil(1300)
	commands := len(f.window.commands)

	for i, starting := range []bool{false, true, true, false} {
		*f.clock += 750
		f.controller.onEvent(starting, 3000+uint32(i)*100, *f.clock)
	}
	f.tickUntil(*f.clock + 1000)

	if len(f.window.commands) != commands || f.window.count(swShowNoActivate) != 1 {
		t.Fatalf("commands = %v, want no command after the replay", f.window.commands)
	}
	f.requireLog(t, "minimize event: start age=40ms", "minimize event: end age=3000ms", "minimize event: start age=3200ms")
	if strings.Count(f.output.String(), "minimize edge") != 1 {
		t.Fatalf("late events found edges:\n%s", f.output.String())
	}
}

func TestMinimizeControllerUsesTheActivatingFallbackOnlyOnce(t *testing.T) {
	f := newMinimizeControllerFixture()
	f.window.deferred[swShowNoActivate] = true

	f.externalMinimize(1020)
	f.tickUntil(1300)
	f.controller.onEvent(true, 3000, 1400)
	f.tickUntil(3000)

	if f.window.count(swRestore) != 1 || f.window.count(swShowNoActivate) != 1 || f.window.count(swMinimize) != 1 {
		t.Fatalf("commands = %v, want one fallback restore and one replay", f.window.commands)
	}
	f.requireLog(t,
		"own restore: before=iconic after=shown fallback=true",
		"minimize intercepted: restored=true replay-in=75ms",
		"minimize replay accepted with composed halo",
	)
}

func TestMinimizeControllerUnsettledRestoreComposesNothing(t *testing.T) {
	f := newMinimizeControllerFixture()
	f.window.deferred[swShowNoActivate] = true
	f.window.deferred[swRestore] = true

	f.externalMinimize(1020)
	f.tickUntil(3000)

	if f.window.compositions != 0 || f.window.count(swMinimize) != 0 || f.window.count(swRestore) != 1 {
		t.Fatalf("commands = %v, compositions %d, want no composition and no replay", f.window.commands, f.window.compositions)
	}
	f.requireLog(t,
		"own restore: before=iconic after=iconic fallback=true",
		"own call unsettled: expected=shown observed=iconic",
		"minimize intercepted: restored=false replay=none",
		"minimize interception disabled: own call unsettled (expected=shown observed=iconic)",
	)
	f.forbidLog(t, "replay-in=75ms", "minimize replay requested")
	text := f.output.String()
	if strings.Index(text, "own call unsettled") > strings.Index(text, "restored=false") {
		t.Fatalf("interception line before the unsettled line:\n%s", text)
	}
}

func TestMinimizeControllerStampsOwnCallsAfterTheAfterReading(t *testing.T) {
	f := newMinimizeControllerFixture()
	f.window.showDelayMS = 120

	f.externalMinimize(1020)

	if f.controller.model.lastShownAt != 1140 || f.controller.model.replayAt != 1215 {
		t.Fatalf("model = %+v, want shown at 1140 and replay at 1215", f.controller.model)
	}
	// In Priming an external minimize cancels the replay (no re-prime); its
	// age bound starts at the restore's after-reading, not at the observation.
	f.externalMinimize(1170)
	f.requireLog(t, "minimize edge: shown->iconic age<=30ms action=cancel-replay")
	f.forbidLog(t, "unknown-age", "age<=150ms")

	unsettled := newMinimizeControllerFixture()
	unsettled.window.showDelayMS = 120
	unsettled.window.deferred[swShowNoActivate] = true
	unsettled.window.deferred[swRestore] = true
	unsettled.externalMinimize(1020)
	if unsettled.controller.model.settleDeadline != 2260 {
		t.Fatalf("settle deadline = %d, want after-reading tick 1260 + 1000", unsettled.controller.model.settleDeadline)
	}
}

func TestMinimizeControllerDeferredReplayLatches(t *testing.T) {
	f := newMinimizeControllerFixture()
	f.window.deferred[swMinimize] = true

	f.externalMinimize(1020)
	f.tickUntil(1100)
	f.requireLog(t, "own replay: before=shown after=shown", "own call unsettled: expected=iconic observed=shown")
	f.forbidLog(t, "minimize replay accepted")
	f.tickUntil(2600)
	f.window.applyDeferred()
	f.tickUntil(2650)

	f.requireLog(t,
		"minimize interception disabled: own call unsettled (expected=iconic observed=shown)",
		"minimize edge: shown->iconic age<=25ms action=skip reason=latched",
	)
	if f.window.count(swShowNoActivate) != 1 || !f.controller.model.latched {
		t.Fatalf("commands = %v, latched %t, want one restore and the latch", f.window.commands, f.controller.model.latched)
	}
}

func TestMinimizeControllerLogsCapTripAndResume(t *testing.T) {
	f := newMinimizeControllerFixture()
	for _, at := range []uint64{1020, 1220} {
		f.externalMinimize(at)
		f.tickUntil(at + 80)
		f.externalRestore(at + 180)
	}

	f.externalMinimize(1420)
	f.requireLog(t,
		"minimize edge: shown->iconic age<=20ms action=skip reason=cap",
		"minimize interception suspended: 2 intercepts in 2000ms",
	)
	f.externalRestore(1500)
	f.tickUntil(6419)
	f.forbidLog(t, "resumed")
	f.tickUntil(6450)
	f.requireLog(t, "minimize interception resumed after 5000ms quiet")
	f.externalMinimize(6450)

	if f.window.count(swShowNoActivate) != 3 || f.controller.model.cap.suspended {
		t.Fatalf("commands = %v, cap %+v, want the capped edge left alone", f.window.commands, f.controller.model.cap)
	}
	if strings.Count(f.output.String(), "interception suspended") != 1 {
		t.Fatalf("log reports more than one trip:\n%s", f.output.String())
	}
}

func TestMinimizeControllerLogsRenderErrorsSkipsAndHonoredRestores(t *testing.T) {
	f := newMinimizeControllerFixture()
	f.window.composeErr = errors.New("render failed")

	f.externalMinimize(1020)
	f.tickUntil(1100)
	f.window.iconic = false
	f.controller.onEvent(false, 12, 1200)
	*f.clock = 5000
	f.externalMinimize(5000)

	f.requireLog(t,
		"minimize prime render error: render failed",
		"minimize replay render error: render failed",
		"minimize event: end age=12ms",
		"minimize edge: iconic->shown action=restore-honored",
		"minimize edge: shown->iconic age<=3800ms action=skip reason=unknown-age",
	)
	if f.window.count(swShowNoActivate) != 1 {
		t.Fatalf("commands = %v, want the skipped minimize left alone", f.window.commands)
	}
}
