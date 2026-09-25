//go:build windows

package main

// The Win32 side of minimize interception: the WinEvent hook on the target
// process, its out-of-context callback, the priming restore, the halo
// composition and the replayed minimize run from the polling tick. The pure
// transitions these functions apply live in minimize_windows.go.

import (
	"fmt"
	"syscall"
	"unsafe"
)

var minimizeWinEventCallback = syscall.NewCallback(minimizeWinEventProc)

// installMinimizeHook watches the target process for the system's pre-minimize
// notification. The first transition is restored, the child halo is composed,
// and minimization is replayed after DWM has presented the halo for several
// frames.
func (a *application) installMinimizeHook() error {
	var processID uint32
	threadID, _, callErr := procGetWindowThreadProcessId.Call(
		a.target,
		uintptr(unsafe.Pointer(&processID)),
	)
	if threadID == 0 || processID == 0 {
		return fmt.Errorf("GetWindowThreadProcessId: %w", callErr)
	}
	hook, _, callErr := procSetWinEventHook.Call(
		uintptr(eventSystemMinimizeStart),
		uintptr(eventSystemMinimizeEnd),
		0,
		minimizeWinEventCallback,
		uintptr(processID),
		0,
		uintptr(wineventOutofcontext),
	)
	if hook == 0 {
		return fmt.Errorf("SetWinEventHook: %w", callErr)
	}
	a.minimizeHook = hook
	return nil
}

func minimizeWinEventProc(_, event, hwnd, idObject, idChild, _, _ uintptr) uintptr {
	app := activeApp
	if app == nil {
		return 0
	}
	matched, starting := minimizeTransition(
		uint32(event),
		hwnd,
		app.target,
		int32(idObject),
		int32(idChild),
	)
	if !matched {
		return 0
	}

	next, action := minimizeEventTransition(app.minimizeState, starting)
	app.minimizeState = next
	switch action {
	case minimizePrime:
		wasMinimized := app.restoreTargetForMinimizePriming()
		if err := app.composeHalo(); err != nil {
			app.logger.Printf("minimize prime render error: %v", err)
		}
		app.minimizeReplayAt = getTickCount64() + minimizeReplayDelayMS
		app.logger.Printf(
			"minimize intercepted: restored=%t replay-in=%dms",
			wasMinimized,
			minimizeReplayDelayMS,
		)
	case minimizeAllowReplay:
		app.minimizeReplayAt = 0
		if err := app.composeHalo(); err != nil {
			app.logger.Printf("minimize replay compose error: %v", err)
		}
		app.logger.Printf("minimize replay accepted with composed halo")
	case minimizeRestored:
		app.minimizeReplayAt = 0
		app.logger.Printf("minimize end")
	}
	return 0
}

func (a *application) restoreTargetForMinimizePriming() bool {
	minimized, _, _ := procIsIconic.Call(a.target)
	procShowWindow.Call(a.target, swShowNoActivate)
	stillMinimized, _, _ := procIsIconic.Call(a.target)
	if stillMinimized != 0 {
		procShowWindow.Call(a.target, swRestore)
	}
	return minimized != 0
}

func (a *application) composeHalo() error {
	if err := a.updateGeometryAndBitmap(); err != nil {
		return err
	}
	if err := a.show(); err != nil {
		return err
	}
	return flushDwm()
}

func (a *application) replayPendingMinimize(now uint64) {
	if a.minimizeState != minimizePriming || now < a.minimizeReplayAt {
		return
	}

	minimized, _, _ := procIsIconic.Call(a.target)
	if minimized != 0 {
		a.restoreTargetForMinimizePriming()
		if err := a.composeHalo(); err != nil {
			a.logger.Printf("minimize re-prime render error: %v", err)
		}
		a.minimizeReplayAt = now + minimizeReplayDelayMS
		a.logger.Printf("minimize re-primed after original transition completed")
		return
	}

	if err := a.composeHalo(); err != nil {
		a.logger.Printf("minimize replay render error: %v", err)
	}
	a.minimizeState = minimizeReplaying
	a.minimizeReplayAt = 0
	a.logger.Printf("minimize replay requested after halo composition")
	procShowWindow.Call(a.target, swMinimize)
}
