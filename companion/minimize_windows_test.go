//go:build windows

package main

// Tests for the pure minimize interception logic of minimize_windows.go: the
// WinEvent filter on the tracked top-level window and the event-driven phase
// transitions, moved unchanged out of main_windows_test.go.

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

func TestMinimizeEventTransitionPrimesThenAcceptsTheReplay(t *testing.T) {
	phase, action := minimizeEventTransition(minimizeIdle, true)
	if phase != minimizePriming || action != minimizePrime {
		t.Fatalf("initial start = (%v, %v), want priming/prime", phase, action)
	}

	phase, action = minimizeEventTransition(phase, false)
	if phase != minimizePriming || action != minimizeNoAction {
		t.Fatalf("internal restore = (%v, %v), want priming/no-action", phase, action)
	}

	phase = minimizeReplaying
	phase, action = minimizeEventTransition(phase, true)
	if phase != minimizeCommitted || action != minimizeAllowReplay {
		t.Fatalf("replayed start = (%v, %v), want committed/allow-replay", phase, action)
	}

	phase, action = minimizeEventTransition(phase, false)
	if phase != minimizeIdle || action != minimizeRestored {
		t.Fatalf("real restore = (%v, %v), want idle/restored", phase, action)
	}
}

func TestDuplicateMinimizeStartDoesNotRestartPriming(t *testing.T) {
	phase, action := minimizeEventTransition(minimizePriming, true)
	if phase != minimizePriming || action != minimizeNoAction {
		t.Fatalf("duplicate start = (%v, %v), want priming/no-action", phase, action)
	}
}
