//go:build windows

package main

// The Win32 side of minimize interception: the WinEvent hook on the target
// process, its out-of-context callback, the halo composition, and the
// minimizeController that runs the pure minimizeModel of minimize_windows.go
// on every polling tick and every matched minimize WinEvent.
//
// v0.0.24: decisions come from observed IsIconic edges, not from WinEvent
// order. The controller reaches the window only through the three-method
// minimizeWindow seam, implemented by *application and by a scripted fake in
// the tests. Each own ShowWindow call reads the window before and after, and
// is stamped with the tick taken after its after-reading, so the replay tick,
// the settle deadline and the age bound follow that reading however long
// ShowWindow blocked. A WinEvent only logs its dwmsEventTime age and triggers
// one observation; the re-prime and the legacy "minimize end" line are gone,
// and every decision writes one native-host.log line.

import (
	"fmt"
	"log"
	"syscall"
	"unsafe"
)

var minimizeWinEventCallback = syscall.NewCallback(minimizeWinEventProc)

// minimizeWindow is the window seam of minimizeController.
type minimizeWindow interface {
	isIconic() bool
	showWindow(cmd uintptr)
	composeHalo() error
}

// minimizeController owns the minimize model of one window, executes its
// actions through the window seam, absorbs its own calls, and logs every
// decision.
type minimizeController struct {
	model  minimizeModel
	window minimizeWindow
	logger *log.Logger
	clock  func() uint64
}

// newMinimizeController seeds the model from one reading taken at clock().
func newMinimizeController(window minimizeWindow, logger *log.Logger, clock func() uint64) *minimizeController {
	return &minimizeController{
		model:  newMinimizeModel(window.isIconic(), clock()),
		window: window,
		logger: logger,
		clock:  clock,
	}
}

// observe runs one observation at now, logs its decision, intercepts a prompt
// external minimize, and runs the pending replay once it is due.
func (c *minimizeController) observe(now uint64) {
	settling := c.model
	next, decision := c.model.observe(c.window.isIconic(), now)
	c.model = next
	if decision.latchedNow {
		c.logger.Printf(
			"minimize interception disabled: own call unsettled (expected=%s observed=%s)",
			minimizeReadingName(settling.expectIconic),
			minimizeReadingName(settling.iconic),
		)
	}
	c.logEdge(decision)
	if decision.action == minimizeIntercept {
		c.intercept()
	}
	if c.model.replayDue(now) {
		c.replay()
	}
}

// onEvent logs a matched minimize WinEvent with its age, then observes.
func (c *minimizeController) onEvent(starting bool, eventAgeMS uint32, now uint64) {
	kind := "end"
	if starting {
		kind = "start"
	}
	c.logger.Printf("minimize event: %s age=%dms", kind, eventAgeMS)
	c.observe(now)
}

func (c *minimizeController) logEdge(decision minimizeDecision) {
	switch decision.edge {
	case minimizeShownToIconic:
		reason := ""
		if decision.reason != "" {
			reason = " reason=" + decision.reason
		}
		c.logger.Printf(
			"minimize edge: %s age<=%dms action=%s%s",
			decision.edge,
			decision.ageBound,
			decision.action,
			reason,
		)
	case minimizeIconicToShown:
		c.logger.Printf("minimize edge: %s action=%s", decision.edge, decision.action)
	}
}

// intercept restores the window once; the halo is composed and the replay
// scheduled only when the restore's after-reading is shown. An unsettled
// restore composes nothing and schedules no replay.
func (c *minimizeController) intercept() {
	if !c.ownRestore() {
		c.logger.Printf("minimize intercepted: restored=false replay=none")
		return
	}
	if err := c.window.composeHalo(); err != nil {
		c.logger.Printf("minimize prime render error: %v", err)
	}
	c.logger.Printf("minimize intercepted: restored=true replay-in=%dms", minimizeReplayDelayMS)
}

// replay composes the halo, replays the minimize and accepts it when the
// after-reading is iconic.
func (c *minimizeController) replay() {
	if err := c.window.composeHalo(); err != nil {
		c.logger.Printf("minimize replay render error: %v", err)
	}
	c.logger.Printf("minimize replay requested after halo composition")
	if c.ownReplay() {
		c.logger.Printf("minimize replay accepted with composed halo")
	}
}

// ownRestore restores without activation, falls back once to the activating
// SW_RESTORE when the window is still iconic, and reports whether the window
// ended shown as expected.
func (c *minimizeController) ownRestore() bool {
	before := c.window.isIconic()
	c.window.showWindow(swShowNoActivate)
	after := c.window.isIconic()
	fallback := after
	if fallback {
		c.window.showWindow(swRestore)
		after = c.window.isIconic()
	}
	at := c.clock()
	c.logger.Printf(
		"own restore: before=%s after=%s fallback=%t",
		minimizeReadingName(before),
		minimizeReadingName(after),
		fallback,
	)
	return c.settleOwnCall(false, after, at)
}

// ownReplay minimizes the window and reports whether it ended iconic as
// expected.
func (c *minimizeController) ownReplay() bool {
	before := c.window.isIconic()
	c.window.showWindow(swMinimize)
	after := c.window.isIconic()
	at := c.clock()
	c.logger.Printf(
		"own replay: before=%s after=%s",
		minimizeReadingName(before),
		minimizeReadingName(after),
	)
	return c.settleOwnCall(true, after, at)
}

func (c *minimizeController) settleOwnCall(expectIconic, afterIconic bool, at uint64) bool {
	c.model = c.model.ownCall(expectIconic, afterIconic, at)
	if afterIconic != expectIconic {
		c.logger.Printf(
			"own call unsettled: expected=%s observed=%s",
			minimizeReadingName(expectIconic),
			minimizeReadingName(afterIconic),
		)
		return false
	}
	return true
}

// installMinimizeHook watches the target process for the system's minimize
// notifications. Each matched notification triggers one observation of the
// target; the controller decides from the observed state.
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

// minimizeWinEventProc hands each matched minimize WinEvent of the target to
// the controller with its age. dwmsEventTime is a 32-bit GetTickCount value,
// so the age is computed in wrapping 32-bit arithmetic.
func minimizeWinEventProc(_, event, hwnd, idObject, idChild, _, dwmsEventTime uintptr) uintptr {
	app := activeApp
	if app == nil || app.minimize == nil {
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
	now := getTickCount64()
	app.minimize.onEvent(starting, uint32(now)-uint32(dwmsEventTime), now)
	return 0
}

func (a *application) isIconic() bool {
	iconic, _, _ := procIsIconic.Call(a.target)
	return iconic != 0
}

func (a *application) showWindow(cmd uintptr) {
	procShowWindow.Call(a.target, cmd)
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
