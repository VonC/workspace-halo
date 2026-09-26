//go:build windows

package main

// Acceptance scenarios for the v0.0.24 minimize interception (step 4): one
// sub-test per row of the design's acceptance table, the click and the restore
// around the replay, and a four-window replay of the 2026-09-24 unplug
// timeline, all run through minimizeController by the scenario runner of
// minimize_scenario_windows_test.go. Each scenario is a script of external
// window actions, stalls and event lags, and asserts the restore attempts, the
// interceptions, the edges, the final phase and the decisive log lines. The
// acceptance tests were split out of minimize_hook_windows_test.go, as the
// plan's split guidance asks, to keep that file under the line ceiling.

import (
	"strings"
	"testing"
)

// minimizeAcceptanceCase is one row of the design's acceptance table: a
// script of external actions and event lags, and the expected restore
// attempts, interceptions, edges, final phase and decisive log lines.
type minimizeAcceptanceCase struct {
	name        string
	lags        []uint64
	steps       []minimizeScenarioStep
	until       uint64
	restores    int
	intercepted int
	edges       int
	phase       minimizePhase
	decisive    []string
	forbidden   []string
	check       func(t *testing.T, s *minimizeScenario)
}

// unsettledReplaySteps defers the own replay of an interception at 1010, so
// its after-reading is shown and the model latches at 2100.
func unsettledReplaySteps(extra ...minimizeScenarioStep) []minimizeScenarioStep {
	steps := []minimizeScenarioStep{
		{at: 1000, kind: minimizeStepDefer, value: uint64(swMinimize)},
		scenarioMinimize(1010),
	}
	return append(steps, extra...)
}

// capSteps minimizes at 1010 and 1800 with a restore in between, both
// intercepted by the ticks at 1025 and 1800.
func capSteps(extra ...minimizeScenarioStep) []minimizeScenarioStep {
	steps := []minimizeScenarioStep{scenarioMinimize(1010), scenarioRestore(1500), scenarioMinimize(1800)}
	return append(steps, extra...)
}

// capStreamSteps trips the cap at 2600, then restores and minimizes every
// second for 20 s, the last edge at 22600.
func capStreamSteps() []minimizeScenarioStep {
	steps := capSteps(scenarioRestore(2300), scenarioMinimize(2600))
	for k := uint64(1); k <= 20; k++ {
		steps = append(steps, scenarioRestore(2100+1000*k), scenarioMinimize(2600+1000*k))
	}
	return steps
}

const minimizeLatchedLine = "minimize interception disabled: own call unsettled (expected=iconic observed=shown)"

func minimizeAcceptanceCases() []minimizeAcceptanceCase {
	return []minimizeAcceptanceCase{
		{
			name: "normal path intercepts once and replays", lags: []uint64{0},
			steps: []minimizeScenarioStep{scenarioMinimize(1020)}, until: 1500,
			restores: 1, intercepted: 1, edges: 1, phase: minimizeMinimized,
			decisive: []string{
				"minimize event: start age=0ms",
				"minimize edge: shown->iconic age<=20ms action=intercept",
				"minimize intercepted: restored=true replay-in=75ms",
				"minimize replay accepted with composed halo",
			},
		},
		{
			name: "events in order 3 s late find no edge", lags: []uint64{3000},
			steps: []minimizeScenarioStep{scenarioMinimize(1010)}, until: 4200,
			restores: 1, intercepted: 1, edges: 1, phase: minimizeMinimized,
			decisive: []string{
				"minimize edge: shown->iconic age<=25ms action=intercept",
				"minimize replay accepted with composed halo",
				"minimize event: start age=3000ms",
				"minimize event: end age=3000ms",
				"minimize event: start age=3000ms",
			},
		},
		{
			name: "events reordered 3 s late find no edge", lags: []uint64{3000, 3600, 3300},
			steps: []minimizeScenarioStep{scenarioMinimize(1010)}, until: 5000,
			restores: 1, intercepted: 1, edges: 1, phase: minimizeMinimized,
			decisive: []string{
				"minimize edge: shown->iconic age<=25ms action=intercept",
				"minimize event: start age=3000ms",
				"minimize event: start age=3300ms",
				"minimize event: end age=3600ms",
			},
		},
		{
			name: "replay start delivered after a restore keeps Shown", lags: []uint64{0, 0, 900, 0},
			steps: []minimizeScenarioStep{scenarioMinimize(1010), scenarioRestore(1500)}, until: 2100,
			restores: 1, intercepted: 1, edges: 2, phase: minimizeShown,
			decisive: []string{
				"minimize edge: shown->iconic age<=10ms action=intercept",
				"own replay: before=shown after=iconic",
				"minimize event: end age=0ms",
				"minimize edge: iconic->shown action=restore-honored",
				"minimize event: start age=900ms",
			},
		},
		{
			name: "lost restore end still ends Shown", lags: []uint64{0, 0, 0, minimizeEventLost},
			steps: []minimizeScenarioStep{scenarioMinimize(1010), scenarioRestore(1500)}, until: 1600,
			restores: 1, intercepted: 1, edges: 2, phase: minimizeShown,
			decisive: []string{
				"minimize edge: shown->iconic age<=10ms action=intercept",
				"minimize replay accepted with composed halo",
				"minimize edge: iconic->shown action=restore-honored",
			},
			check: func(t *testing.T, s *minimizeScenario) {
				if got := strings.Count(s.outputs[0].String(), "minimize event: end"); got != 1 {
					t.Fatalf("delivered end events = %d, want only the own restore's", got)
				}
			},
		},
		{
			name: "start delivered during Priming lets the replay proceed", lags: []uint64{40, 0, 0},
			steps: []minimizeScenarioStep{scenarioMinimize(1010)}, until: 1200,
			restores: 1, intercepted: 1, edges: 1, phase: minimizeMinimized,
			decisive: []string{
				"minimize edge: shown->iconic age<=25ms action=intercept",
				"minimize event: start age=40ms",
				"minimize replay accepted with composed halo",
			},
		},
		{
			name: "late animation 100 ms after the restore cancels the replay", lags: []uint64{0},
			steps: []minimizeScenarioStep{
				scenarioMinimize(1010),
				{at: 1011, kind: minimizeStepStall, value: 99},
				scenarioMinimize(1110),
			},
			until: 1300, restores: 1, intercepted: 1, edges: 2, phase: minimizeMinimized,
			decisive: []string{
				"minimize edge: shown->iconic age<=10ms action=intercept",
				"minimize edge: shown->iconic age<=100ms action=cancel-replay",
			},
			forbidden: []string{"minimize replay requested"},
			check: func(t *testing.T, s *minimizeScenario) {
				if !s.controllers[0].model.haloComposed || s.windows[0].count(swMinimize) != 0 {
					t.Fatalf("model %+v, commands %v, want halo composed and no replay", s.controllers[0].model, s.windows[0].commands)
				}
			},
		},
		{
			name: "user minimize within 250 ms of the restore is not re-primed", lags: []uint64{0},
			steps: []minimizeScenarioStep{scenarioMinimize(1010), scenarioMinimize(1070)}, until: 1200,
			restores: 1, intercepted: 1, edges: 2, phase: minimizeMinimized,
			decisive:  []string{"minimize edge: shown->iconic age<=20ms action=cancel-replay"},
			forbidden: []string{"minimize replay requested"},
			check: func(t *testing.T, s *minimizeScenario) {
				if s.controllers[0].model.cap.filled != 1 {
					t.Fatalf("cap = %+v, want only the interception counted", s.controllers[0].model.cap)
				}
			},
		},
		{
			name: "replay after-reading shown latches at the deadline", lags: []uint64{0},
			steps: unsettledReplaySteps(), until: 2200,
			restores: 1, intercepted: 1, edges: 1, phase: minimizeShown,
			decisive: []string{
				"own replay: before=shown after=shown",
				"own call unsettled: expected=iconic observed=shown",
				minimizeLatchedLine,
			},
			check: func(t *testing.T, s *minimizeScenario) {
				if !s.controllers[0].model.latched {
					t.Fatal("interception not latched off")
				}
			},
		},
		{
			name: "replay applying at 2.5 s is skipped latched", lags: []uint64{0},
			steps: unsettledReplaySteps(minimizeScenarioStep{at: 3600, kind: minimizeStepApplyDeferred}),
			until: 3700, restores: 1, intercepted: 1, edges: 2, phase: minimizeMinimized,
			decisive: []string{minimizeLatchedLine, "minimize edge: shown->iconic age<=25ms action=skip reason=latched"},
		},
		{
			name: "replay applying at 7 s is skipped latched", lags: []uint64{0},
			steps: unsettledReplaySteps(minimizeScenarioStep{at: 8100, kind: minimizeStepApplyDeferred}),
			until: 8200, restores: 1, intercepted: 1, edges: 2, phase: minimizeMinimized,
			decisive: []string{minimizeLatchedLine, "minimize edge: shown->iconic age<=25ms action=skip reason=latched"},
		},
		{
			name: "unrelated minimize 90 s after the latch is skipped latched", lags: []uint64{0},
			steps: unsettledReplaySteps(scenarioMinimize(92100)), until: 92200,
			restores: 1, intercepted: 1, edges: 2, phase: minimizeMinimized,
			decisive: []string{minimizeLatchedLine, "minimize edge: shown->iconic age<=25ms action=skip reason=latched"},
			check: func(t *testing.T, s *minimizeScenario) {
				if s.controllers[0].model.haloComposed {
					t.Fatal("a latched minimize reports a composed halo")
				}
			},
		},
		{
			name: "a new host after a latch intercepts again", lags: []uint64{0},
			steps: unsettledReplaySteps(
				minimizeScenarioStep{at: 3000, kind: minimizeStepRestart},
				scenarioMinimize(3510),
			),
			until: 3700, restores: 2, intercepted: 2, edges: 2, phase: minimizeMinimized,
			decisive: []string{
				minimizeLatchedLine,
				"minimize edge: shown->iconic age<=10ms action=intercept",
				"minimize replay accepted with composed halo",
			},
			check: func(t *testing.T, s *minimizeScenario) {
				if s.controllers[0].model.latched {
					t.Fatal("the new host starts latched")
				}
			},
		},
		{
			name: "a 3.4 s old shown reading is skipped unknown-age", lags: []uint64{0},
			steps: []minimizeScenarioStep{
				{at: 1000, kind: minimizeStepStall, value: 3400},
				scenarioMinimize(4390),
			},
			until: 4500, restores: 0, intercepted: 0, edges: 1, phase: minimizeMinimized,
			decisive: []string{
				"minimize event: start age=10ms",
				"minimize edge: shown->iconic age<=3400ms action=skip reason=unknown-age",
			},
		},
		{
			name:  "minimize restore minimize within 2 s intercepts twice",
			steps: capSteps(), until: 2000,
			restores: 2, intercepted: 2, edges: 3, phase: minimizeMinimized,
			decisive: []string{
				"minimize edge: shown->iconic age<=25ms action=intercept",
				"minimize edge: iconic->shown action=restore-honored",
				"minimize edge: shown->iconic age<=25ms action=intercept",
				"minimize replay accepted with composed halo",
			},
			forbidden: []string{"reason=cap", "minimize event"},
		},
		{
			name:  "a third edge within 2 s trips the cap",
			steps: capSteps(scenarioRestore(2300), scenarioMinimize(2600)), until: 2700,
			restores: 2, intercepted: 2, edges: 5, phase: minimizeMinimized,
			decisive: []string{
				"minimize edge: shown->iconic age<=25ms action=skip reason=cap",
				"minimize interception suspended: 2 intercepts in 2000ms",
			},
		},
		{
			name:  "edges every second for 20 s keep the cap closed until 5 s of quiet",
			steps: capStreamSteps(), until: 27599,
			restores: 2, intercepted: 2, edges: 45, phase: minimizeMinimized,
			decisive:  []string{"minimize interception suspended: 2 intercepts in 2000ms"},
			forbidden: []string{"resumed"},
			check: func(t *testing.T, s *minimizeScenario) {
				if got := strings.Count(s.outputs[0].String(), "reason=cap"); got != 21 || !s.controllers[0].model.cap.suspended {
					t.Fatalf("capped edges = %d, cap %+v, want 21 and still suspended", got, s.controllers[0].model.cap)
				}
				s.run(t, nil, 27600)
				s.requireOrderedLog(t, 0, "minimize interception resumed after 5000ms quiet")
				s.run(t, []minimizeScenarioStep{scenarioRestore(27700), scenarioMinimize(28000)}, 28200)
				s.requireOrderedLog(t, 0,
					"minimize interception resumed after 5000ms quiet",
					"minimize edge: shown->iconic age<=25ms action=intercept",
					"minimize replay accepted with composed halo",
				)
				if s.windows[0].count(swShowNoActivate) != 3 {
					t.Fatalf("commands = %v, want a third restore once the cap rearmed", s.windows[0].commands)
				}
			},
		},
	}
}

func TestMinimizeAcceptanceCases(t *testing.T) {
	for _, tc := range minimizeAcceptanceCases() {
		t.Run(tc.name, func(t *testing.T) {
			s := newMinimizeScenario(1, tc.lags)
			s.run(t, tc.steps, tc.until)
			text := s.outputs[0].String()
			model := s.controllers[0].model

			if got := s.windows[0].count(swShowNoActivate); got != tc.restores {
				t.Fatalf("restores = %d, want %d:\n%s", got, tc.restores, text)
			}
			if got := strings.Count(text, "minimize intercepted:"); got != tc.intercepted {
				t.Fatalf("interceptions = %d, want %d:\n%s", got, tc.intercepted, text)
			}
			if got := strings.Count(text, "minimize edge:"); got != tc.edges {
				t.Fatalf("edges = %d, want %d:\n%s", got, tc.edges, text)
			}
			if model.phase != tc.phase || model.iconic != s.windows[0].iconic {
				t.Fatalf("model = %+v, window iconic %t, want phase %d on the observed state", model, s.windows[0].iconic, tc.phase)
			}
			s.requireOrderedLog(t, 0, tc.decisive...)
			for _, fragment := range tc.forbidden {
				if strings.Contains(text, fragment) {
					t.Fatalf("log holds %q:\n%s", fragment, text)
				}
			}
			if tc.check != nil {
				tc.check(t, s)
			}
		})
	}
}

func TestMinimizeAcceptanceClickDuringPrimingStillReplays(t *testing.T) {
	s := newMinimizeScenario(1, []uint64{0})

	s.run(t, []minimizeScenarioStep{
		scenarioMinimize(1010),
		{at: 1050, kind: minimizeStepClick},
	}, 1200)

	w := s.windows[0]
	if w.count(swShowNoActivate) != 1 || w.count(swMinimize) != 1 || !w.iconic {
		t.Fatalf("commands = %v, iconic %t, want one restore and the replay minimizing the window", w.commands, w.iconic)
	}
	if model := s.controllers[0].model; model.phase != minimizeMinimized || !model.haloComposed {
		t.Fatalf("model = %+v, want Minimized with the halo", model)
	}
	s.requireOrderedLog(t, 0, "action=intercept", "minimize replay accepted with composed halo")
}

func TestMinimizeAcceptanceRestoreAfterReplayStaysRestored(t *testing.T) {
	s := newMinimizeScenario(1, []uint64{3000, 4500, 3200, 5000})

	s.run(t, []minimizeScenarioStep{scenarioMinimize(1010), scenarioRestore(1500)}, 8000)

	w := s.windows[0]
	if len(w.commands) != 2 || w.count(swShowNoActivate) != 1 || w.count(swMinimize) != 1 || w.iconic {
		t.Fatalf("commands = %v, iconic %t, want one restore, one replay, and the window left restored", w.commands, w.iconic)
	}
	if s.controllers[0].model.phase != minimizeShown {
		t.Fatalf("model = %+v, want Shown", s.controllers[0].model)
	}
	s.requireOrderedLog(t, 0,
		"minimize replay accepted with composed halo",
		"minimize edge: iconic->shown action=restore-honored",
		"minimize event: start age=3000ms",
		"minimize event: end age=5000ms",
	)
	if got := strings.Count(s.outputs[0].String(), "minimize edge:"); got != 2 {
		t.Fatalf("edges = %d, want the minimize and the restore only:\n%s", got, s.outputs[0].String())
	}
}

// TestMinimizeAcceptanceRecordedUnplugTimeline replays the 2026-09-24 unplug:
// Windows minimizes w4, w1 and w2 at 0, 2.7 s and 4.0 s while w3 stays shown,
// every minimize WinEvent, own calls included, arrives 3 to 6 s late and
// reordered, and w2's non-activating restore does not apply, so its one
// interception takes the SW_RESTORE fallback.
func TestMinimizeAcceptanceRecordedUnplugTimeline(t *testing.T) {
	s := newMinimizeScenario(4, []uint64{5200, 3000, 6000, 4100, 3500, 5700})

	s.run(t, []minimizeScenarioStep{
		{at: 1000, window: 1, kind: minimizeStepDefer, value: uint64(swShowNoActivate)},
		{at: 10000, window: 3, kind: minimizeStepMinimize},
		{at: 12700, window: 0, kind: minimizeStepMinimize},
		{at: 14000, window: 1, kind: minimizeStepMinimize},
	}, 25000)

	for _, i := range []int{0, 1, 3} {
		w, model, text := s.windows[i], s.controllers[i].model, s.outputs[i].String()
		fallbacks := 0
		if i == 1 {
			fallbacks = 1
		}
		if w.count(swShowNoActivate) != 1 || w.count(swRestore) != fallbacks || w.count(swMinimize) != 1 {
			t.Fatalf("w%d commands = %v, want one restore attempt and one replay", i+1, w.commands)
		}
		if model.phase != minimizeMinimized || !model.haloComposed || !w.iconic {
			t.Fatalf("w%d model = %+v, want Minimized with the halo", i+1, model)
		}
		if strings.Count(text, "minimize intercepted") != 1 || strings.Count(text, "minimize edge:") != 1 {
			t.Fatalf("w%d intercepted more than once:\n%s", i+1, text)
		}
		if w.firstDeliveryAt == 0 || strings.Count(text, "minimize event:") != 3 {
			t.Fatalf("w%d late events not all delivered:\n%s", i+1, text)
		}
		for _, at := range w.restoreTicks {
			if at >= w.firstDeliveryAt {
				t.Fatalf("w%d restore at %d after the first late event at %d:\n%s", i+1, at, w.firstDeliveryAt, text)
			}
		}
	}
	s.requireOrderedLog(t, 1, "own restore: before=iconic after=shown fallback=true")
	s.requireOrderedLog(t, 3,
		"minimize replay accepted with composed halo",
		"minimize event: end age=3500ms",
		"minimize event: start age=4100ms",
		"minimize event: start age=5700ms",
	)
	if w3 := s.windows[2]; len(w3.commands) != 0 || s.controllers[2].model.phase != minimizeShown {
		t.Fatalf("w3 commands = %v, model %+v, want the shown window left alone", w3.commands, s.controllers[2].model)
	}
}
