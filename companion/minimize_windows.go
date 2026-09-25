//go:build windows

package main

// The pure minimize interception logic: the WinEvent identifiers the host
// watches, the phases and actions of the minimize state machine, and the
// filter that maps a WinEvent callback to the tracked top-level window. It
// holds no Win32 call, so every transition is testable without a window; the
// hook and its callers live in minimize_hook_windows.go.

const (
	eventSystemMinimizeStart = uint32(0x0016)
	eventSystemMinimizeEnd   = uint32(0x0017)
	objidWindow              = int32(0)
	childidSelf              = int32(0)
	wineventOutofcontext     = uint32(0x0000)
)

type minimizePhase uint8

const (
	minimizeIdle minimizePhase = iota
	minimizePriming
	minimizeReplaying
	minimizeCommitted
)

type minimizeAction uint8

const (
	minimizeNoAction minimizeAction = iota
	minimizePrime
	minimizeAllowReplay
	minimizeRestored
)

const minimizeReplayDelayMS = uint64(75)

func minimizeEventTransition(phase minimizePhase, starting bool) (minimizePhase, minimizeAction) {
	if starting {
		switch phase {
		case minimizeIdle:
			return minimizePriming, minimizePrime
		case minimizeReplaying:
			return minimizeCommitted, minimizeAllowReplay
		default:
			return phase, minimizeNoAction
		}
	}
	if phase == minimizePriming {
		// Ignore the restore generated while cancelling the first minimize.
		return phase, minimizeNoAction
	}
	return minimizeIdle, minimizeRestored
}

// minimizeTransition filters WinEvent callbacks to the top-level target and
// maps the two system events to the state kept between the event and the next
// polling tick.
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
