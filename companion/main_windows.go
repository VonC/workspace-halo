//go:build windows

package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	wsPopup         = uint32(0x80000000)
	wsChild         = uint32(0x40000000)
	wsExLayered     = uint32(0x00080000)
	wsExTransparent = uint32(0x00000020)
	wsExToolWindow  = uint32(0x00000080)
	wsExNoActivate  = uint32(0x08000000)

	wmDestroy       = uint32(0x0002)
	wmClose         = uint32(0x0010)
	wmEraseBkgnd    = uint32(0x0014)
	wmNCHitTest     = uint32(0x0084)
	wmMouseActivate = uint32(0x0021)
	wmTimer         = uint32(0x0113)

	htTransparent = ^uintptr(0)
	maNoActivate  = uintptr(3)

	swHide           = 0
	swShowNoActivate = 4

	swpNoSize        = uint32(0x0001)
	swpNoMove        = uint32(0x0002)
	swpNoZOrder      = uint32(0x0004)
	swpNoActivate    = uint32(0x0010)
	swpFrameChanged  = uint32(0x0020)
	swpNoOwnerZOrder = uint32(0x0200)

	gwHwndPrev = uint32(3)
	gwlExStyle = int32(-20)
	gwlStyle   = int32(-16)

	ulwAlpha            = uint32(0x00000002)
	acSrcOver           = byte(0x00)
	acSrcAlpha          = byte(0x01)
	lwaColorKey         = uint32(0x00000001)
	srcCopy             = uint32(0x00CC0020)
	transparentColorKey = uint32(0x00010203)

	biRGB        = uint32(0)
	dibRGBColors = uint32(0)

	transparentBkMode  = 1
	antialiasedQuality = 4
	defaultCharset     = 1
	defaultPitch       = 0

	vkShift = uintptr(0x10)
	vkMenu  = uintptr(0x12)
	vkTab   = uintptr(0x09)

	processQueryLimitedInformation = uint32(0x1000)
	dwmwaCloaked                   = uint32(14)
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	dwmapi   = syscall.NewLazyDLL("dwmapi.dll")

	procRegisterClassExW              = user32.NewProc("RegisterClassExW")
	procCreateWindowExW               = user32.NewProc("CreateWindowExW")
	procDefWindowProcW                = user32.NewProc("DefWindowProcW")
	procDestroyWindow                 = user32.NewProc("DestroyWindow")
	procShowWindow                    = user32.NewProc("ShowWindow")
	procSetWindowPos                  = user32.NewProc("SetWindowPos")
	procSetWindowLongPtrW             = user32.NewProc("SetWindowLongPtrW")
	procSetParent                     = user32.NewProc("SetParent")
	procSetLayeredWindowAttributes    = user32.NewProc("SetLayeredWindowAttributes")
	procGetDC                         = user32.NewProc("GetDC")
	procReleaseDC                     = user32.NewProc("ReleaseDC")
	procGetWindowRect                 = user32.NewProc("GetWindowRect")
	procGetClientRect                 = user32.NewProc("GetClientRect")
	procGetForegroundWindow           = user32.NewProc("GetForegroundWindow")
	procGetWindow                     = user32.NewProc("GetWindow")
	procGetWindowLongPtrW             = user32.NewProc("GetWindowLongPtrW")
	procIsWindow                      = user32.NewProc("IsWindow")
	procIsWindowVisible               = user32.NewProc("IsWindowVisible")
	procIsIconic                      = user32.NewProc("IsIconic")
	procGetWindowThreadProcessId      = user32.NewProc("GetWindowThreadProcessId")
	procGetClassNameW                 = user32.NewProc("GetClassNameW")
	procGetWindowTextW                = user32.NewProc("GetWindowTextW")
	procGetAsyncKeyState              = user32.NewProc("GetAsyncKeyState")
	procGetLastInputInfo              = user32.NewProc("GetLastInputInfo")
	procSetTimer                      = user32.NewProc("SetTimer")
	procKillTimer                     = user32.NewProc("KillTimer")
	procGetMessageW                   = user32.NewProc("GetMessageW")
	procTranslateMessage              = user32.NewProc("TranslateMessage")
	procDispatchMessageW              = user32.NewProc("DispatchMessageW")
	procPostMessageW                  = user32.NewProc("PostMessageW")
	procPostQuitMessage               = user32.NewProc("PostQuitMessage")
	procUpdateLayeredWindow           = user32.NewProc("UpdateLayeredWindow")
	procSetProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")

	procCreateCompatibleDC    = gdi32.NewProc("CreateCompatibleDC")
	procDeleteDC              = gdi32.NewProc("DeleteDC")
	procCreateDIBSection      = gdi32.NewProc("CreateDIBSection")
	procSelectObject          = gdi32.NewProc("SelectObject")
	procDeleteObject          = gdi32.NewProc("DeleteObject")
	procCreateFontW           = gdi32.NewProc("CreateFontW")
	procSetTextColor          = gdi32.NewProc("SetTextColor")
	procSetBkMode             = gdi32.NewProc("SetBkMode")
	procGetTextExtentPoint32W = gdi32.NewProc("GetTextExtentPoint32W")
	procTextOutW              = gdi32.NewProc("TextOutW")
	procBitBlt                = gdi32.NewProc("BitBlt")

	procGetModuleHandleW           = kernel32.NewProc("GetModuleHandleW")
	procOpenProcess                = kernel32.NewProc("OpenProcess")
	procCloseHandle                = kernel32.NewProc("CloseHandle")
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
	procSetLastError               = kernel32.NewProc("SetLastError")

	procDwmGetWindowAttribute = dwmapi.NewProc("DwmGetWindowAttribute")
)

type point struct{ X, Y int32 }
type size struct{ CX, CY int32 }
type rect struct{ Left, Top, Right, Bottom int32 }

func (r rect) width() int             { return int(r.Right - r.Left) }
func (r rect) height() int            { return int(r.Bottom - r.Top) }
func (r rect) equals(other rect) bool { return r == other }

type wndClassEx struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type msg struct {
	Hwnd     uintptr
	Message  uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       point
	LPrivate uint32
}

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type rgbQuad struct{ Blue, Green, Red, Reserved byte }
type bitmapInfo struct {
	Header bitmapInfoHeader
	Colors [1]rgbQuad
}
type blendFunction struct{ BlendOp, BlendFlags, SourceConstantAlpha, AlphaFormat byte }
type lastInputInfo struct{ CbSize, DwTime uint32 }

type config struct {
	name           string
	logo           image.Image
	color          color.NRGBA
	borderWidth    int
	borderStyle    string
	fontFamily     string
	fontWeight     int
	shadow         bool
	logPath        string
	allowAnyWindow bool
	targetOverride uintptr
	waitForVSCode  time.Duration
	windowMode     string
	startupWarning string
}

type application struct {
	cfg              config
	target           uintptr
	overlay          uintptr
	targetRect       rect
	renderedRect     rect
	visible          bool
	visibilityReason string
	manualVisible    bool
	altTabVisible    bool
	shiftDown        bool
	lastShiftRelease uint64
	lastManualInput  uint32
	logger           *log.Logger
	logFile          *os.File
}

var activeApp *application

func main() {
	runtime.LockOSThread()
	cfg, err := parseFlags()
	if err != nil {
		fatalf("configuration error: %v", err)
	}
	logger, logFile, err := openLogger(cfg.logPath)
	if err != nil {
		fatalf("open log: %v", err)
	}
	defer logFile.Close()
	if cfg.startupWarning != "" {
		logger.Printf("warning: %s", cfg.startupWarning)
	}

	setPerMonitorDPIAware()
	target, err := acquireTarget(cfg, logger)
	if err != nil {
		fatalf("select target window: %v", err)
	}

	app := &application{cfg: cfg, target: target, logger: logger, logFile: logFile}
	activeApp = app
	if err := app.createOverlay(); err != nil {
		fatalf("create overlay: %v", err)
	}
	defer app.close()

	logger.Printf("bound to hwnd=0x%X class=%q title=%q", target, windowClass(target), windowTitle(target))
	logger.Printf("overlay hwnd=0x%X name=%q mode=%s", app.overlay, cfg.name, cfg.windowMode)

	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	go func() { <-interrupts; procPostMessageW.Call(app.overlay, uintptr(wmClose), 0, 0) }()

	if err := app.run(); err != nil {
		fatalf("message loop: %v", err)
	}
}

func parseFlags() (config, error) {
	var cfg config
	var logoPath, colorValue, hwndValue string
	flag.StringVar(&cfg.name, "name", "", "workspace name displayed in the overlay")
	flag.StringVar(&logoPath, "logo", "", "PNG logo path")
	flag.StringVar(&colorValue, "color", "#ff2d55", "shared border and text color")
	flag.IntVar(&cfg.borderWidth, "border-width", 12, "border width in pixels")
	flag.StringVar(&cfg.borderStyle, "border-style", "solid", "solid, double, dashed, or dotted")
	flag.StringVar(&cfg.fontFamily, "font", "Segoe UI", "installed Windows font family")
	flag.IntVar(&cfg.fontWeight, "font-weight", 700, "Windows font weight")
	flag.BoolVar(&cfg.shadow, "shadow", true, "draw an automatic dark text shadow")
	flag.StringVar(&cfg.logPath, "log", filepath.Join(os.TempDir(), "workspace-halo-companion.log"), "companion log file")
	flag.BoolVar(&cfg.allowAnyWindow, "allow-any-window", false, "allow binding to a non-Code.exe foreground window for diagnostics")
	flag.StringVar(&hwndValue, "hwnd", "", "optional target HWND in decimal or 0x-prefixed hexadecimal")
	flag.DurationVar(&cfg.waitForVSCode, "wait-for-vscode", time.Minute, "time to wait for a stable VS Code window to become foreground")
	flag.StringVar(&cfg.windowMode, "window-mode", "owned", "overlay attachment mode: owned or child")
	flag.StringVar(&cfg.startupWarning, "startup-warning", "", "warning copied from the VS Code extension into the native-host log")
	flag.Parse()

	if strings.TrimSpace(cfg.name) == "" {
		return cfg, errors.New("--name is required")
	}
	if logoPath == "" {
		return cfg, errors.New("--logo is required")
	}
	file, err := os.Open(logoPath)
	if err != nil {
		return cfg, fmt.Errorf("open logo: %w", err)
	}
	defer file.Close()
	cfg.logo, err = png.Decode(file)
	if err != nil {
		return cfg, fmt.Errorf("decode PNG logo: %w", err)
	}
	cfg.color, err = parseColor(colorValue)
	if err != nil {
		return cfg, err
	}
	if cfg.borderWidth < 1 {
		return cfg, errors.New("--border-width must be positive")
	}
	switch cfg.borderStyle {
	case "solid", "double", "dashed", "dotted":
	default:
		return cfg, fmt.Errorf("unsupported --border-style %q", cfg.borderStyle)
	}
	if cfg.fontWeight < 1 || cfg.fontWeight > 1000 {
		return cfg, errors.New("--font-weight must be between 1 and 1000")
	}
	if cfg.windowMode != "owned" && cfg.windowMode != "child" {
		return cfg, fmt.Errorf("unsupported --window-mode %q", cfg.windowMode)
	}
	if hwndValue != "" {
		parsed, err := strconv.ParseUint(hwndValue, 0, 64)
		if err != nil {
			return cfg, fmt.Errorf("parse --hwnd: %w", err)
		}
		cfg.targetOverride = uintptr(parsed)
	}
	return cfg, nil
}

func parseColor(value string) (color.NRGBA, error) {
	v := strings.TrimPrefix(strings.TrimSpace(value), "#")
	if len(v) == 3 {
		v = strings.Repeat(v[0:1], 2) + strings.Repeat(v[1:2], 2) + strings.Repeat(v[2:3], 2)
	}
	if len(v) != 6 {
		return color.NRGBA{}, fmt.Errorf("color %q must use #RGB or #RRGGBB", value)
	}
	n, err := strconv.ParseUint(v, 16, 32)
	if err != nil {
		return color.NRGBA{}, fmt.Errorf("parse color %q: %w", value, err)
	}
	return color.NRGBA{R: byte(n >> 16), G: byte(n >> 8), B: byte(n), A: 255}, nil
}

func openLogger(path string) (*log.Logger, *os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, nil, err
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, err
	}
	return log.New(io.MultiWriter(file, os.Stderr), "workspace-halo: ", log.Ldate|log.Ltime|log.Lmicroseconds), file, nil
}

func acquireTarget(cfg config, logger *log.Logger) (uintptr, error) {
	if cfg.targetOverride != 0 {
		if cfg.allowAnyWindow {
			return cfg.targetOverride, nil
		}
		path, err := processImagePath(cfg.targetOverride)
		if err != nil {
			return 0, fmt.Errorf("identify --hwnd target: %w", err)
		}
		if !strings.EqualFold(filepath.Base(path), "Code.exe") {
			return 0, fmt.Errorf("--hwnd belongs to %q, not stable VS Code (Code.exe)", path)
		}
		return cfg.targetOverride, nil
	}

	if cfg.allowAnyWindow {
		target := foregroundWindow()
		if target == 0 {
			return 0, errors.New("no foreground window is available")
		}
		return target, nil
	}

	deadline := time.Now().Add(cfg.waitForVSCode)
	logger.Printf("waiting up to %s; focus the stable VS Code window to identify", cfg.waitForVSCode)
	for {
		target := foregroundWindow()
		if target != 0 {
			path, err := processImagePath(target)
			if err == nil && strings.EqualFold(filepath.Base(path), "Code.exe") {
				return target, nil
			}
		}
		if cfg.waitForVSCode <= 0 || time.Now().After(deadline) {
			return 0, errors.New("no stable VS Code window became foreground before the wait expired")
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func fatalf(format string, values ...any) {
	fmt.Fprintf(os.Stderr, "Workspace Halo: "+format+"\n", values...)
	os.Exit(1)
}

func setPerMonitorDPIAware() {
	// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 is the signed pseudo-handle -4.
	procSetProcessDpiAwarenessContext.Call(^uintptr(3))
}

func (a *application) createOverlay() error {
	className, _ := syscall.UTF16PtrFromString("WorkspaceHaloOverlay")
	title, _ := syscall.UTF16PtrFromString("Workspace Halo Overlay")
	hInstance, _, _ := procGetModuleHandleW.Call(0)
	wc := wndClassEx{
		CbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		LpfnWndProc:   syscall.NewCallback(windowProc),
		HInstance:     hInstance,
		LpszClassName: className,
	}
	result, _, callErr := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if result == 0 && callErr != syscall.Errno(0) {
		return callErr
	}
	exStyle := wsExLayered | wsExTransparent | wsExNoActivate
	style := wsPopup
	parent := a.target
	if a.cfg.windowMode == "child" {
		parent = 0
	} else {
		exStyle |= wsExToolWindow
	}
	hwnd, _, callErr := procCreateWindowExW.Call(
		uintptr(exStyle), uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(title)), uintptr(style),
		0, 0, 0, 0, parent, 0, hInstance, 0,
	)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowExW: %w", callErr)
	}
	a.overlay = hwnd
	if a.cfg.windowMode == "child" {
		procSetLastError.Call(0)
		previousStyle, _, styleErr := procSetWindowLongPtrW.Call(hwnd, signedInt32Argument(gwlStyle), uintptr(wsChild))
		if previousStyle == 0 && styleErr != syscall.Errno(0) {
			return fmt.Errorf("SetWindowLongPtrW(WS_CHILD): %w", styleErr)
		}
		procSetLastError.Call(0)
		previousParent, _, parentErr := procSetParent.Call(hwnd, a.target)
		if previousParent == 0 && parentErr != syscall.Errno(0) {
			return fmt.Errorf("SetParent: %w", parentErr)
		}
		procSetWindowPos.Call(
			hwnd, 0,
			0, 0, 0, 0,
			uintptr(swpNoMove|swpNoSize|swpNoZOrder|swpNoActivate|swpFrameChanged),
		)
		layeredResult, _, layeredErr := procSetLayeredWindowAttributes.Call(hwnd, uintptr(transparentColorKey), 255, uintptr(lwaColorKey))
		if layeredResult == 0 {
			return fmt.Errorf("SetLayeredWindowAttributes: %w", layeredErr)
		}
	}
	if timer, _, callErr := procSetTimer.Call(hwnd, 1, 25, 0); timer == 0 {
		return fmt.Errorf("SetTimer: %w", callErr)
	}
	return nil
}

func windowProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	switch message {
	case wmTimer:
		if activeApp != nil {
			if err := activeApp.tick(); err != nil {
				activeApp.logger.Printf("tick error: %v", err)
			}
		}
		return 0
	case wmNCHitTest:
		return htTransparent
	case wmMouseActivate:
		return maNoActivate
	case wmEraseBkgnd:
		return 1
	case wmClose:
		procDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	default:
		result, _, _ := procDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
		return result
	}
}

func (a *application) run() error {
	var message msg
	for {
		result, _, callErr := procGetMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if int32(result) == -1 {
			return callErr
		}
		if result == 0 {
			return nil
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
	}
}

func (a *application) close() {
	if a.overlay != 0 {
		procKillTimer.Call(a.overlay, 1)
		procDestroyWindow.Call(a.overlay)
	}
}

func (a *application) tick() error {
	if ok, _, _ := procIsWindow.Call(a.target); ok == 0 {
		a.logger.Printf("target window closed")
		procPostMessageW.Call(a.overlay, uintptr(wmClose), 0, 0)
		return nil
	}

	focused := foregroundWindow() == a.target
	a.pollDoubleShift(focused)
	a.pollAltTab()
	if a.manualVisible {
		if !focused || lastInputTime() != a.lastManualInput {
			a.manualVisible = false
		}
	}

	minimized, _, _ := procIsIconic.Call(a.target)
	visible, _, _ := procIsWindowVisible.Call(a.target)
	occluded := false
	if !focused && minimized == 0 && visible != 0 {
		occluded = a.hasOccludingWindow()
	}

	desired := a.manualVisible || a.altTabVisible || occluded
	reason := "hidden"
	if a.manualVisible {
		reason = "double-shift"
	} else if a.altTabVisible {
		reason = "alt-tab"
	} else if occluded {
		reason = "occluded"
	}
	if desired {
		if err := a.updateGeometryAndBitmap(); err != nil {
			return err
		}
		a.show()
	} else {
		a.hide()
	}
	if reason != a.visibilityReason {
		a.logger.Printf("visibility=%s focused=%t minimized=%t", reason, focused, minimized != 0)
		a.visibilityReason = reason
	}
	return nil
}

func (a *application) pollAltTab() {
	altDown := asyncKeyDown(vkMenu)
	tabState := asyncKeyState(vkTab)
	if altDown && (tabState&0x8000 != 0 || tabState&1 != 0) {
		if !a.altTabVisible {
			a.logger.Printf("alt-tab gesture")
		}
		a.altTabVisible = true
	}
	if !altDown {
		a.altTabVisible = false
	}
}

func (a *application) pollDoubleShift(focused bool) {
	down := asyncKeyDown(vkShift)
	if down == a.shiftDown {
		return
	}
	a.shiftDown = down
	if down || !focused {
		return
	}
	now := getTickCount64()
	if a.lastShiftRelease != 0 && now-a.lastShiftRelease <= 400 {
		a.manualVisible = true
		a.lastManualInput = lastInputTime()
		a.lastShiftRelease = 0
		a.logger.Printf("double-shift gesture")
		return
	}
	a.lastShiftRelease = now
}

func (a *application) updateGeometryAndBitmap() error {
	r, err := a.overlayRect()
	if err != nil {
		return err
	}
	if r.width() <= 0 || r.height() <= 0 {
		return errors.New("target window has an empty rectangle")
	}
	a.targetRect, _ = windowRect(a.target)
	insertAfter := uintptr(0)
	positionFlags := swpNoActivate | swpNoOwnerZOrder
	left, top := r.Left, r.Top
	if a.cfg.windowMode == "owned" {
		positionFlags |= swpNoZOrder
	} else {
		left, top = 0, 0
	}
	procSetWindowPos.Call(
		a.overlay, insertAfter,
		uintptr(uint32(left)), uintptr(uint32(top)), uintptr(uint32(r.width())), uintptr(uint32(r.height())),
		uintptr(positionFlags),
	)
	if !r.equals(a.renderedRect) || (a.cfg.windowMode == "child" && !a.visible) {
		if err := a.renderOverlay(r); err != nil {
			return err
		}
		a.renderedRect = r
	}
	return nil
}

func (a *application) overlayRect() (rect, error) {
	if a.cfg.windowMode == "child" {
		return clientRect(a.target)
	}
	return windowRect(a.target)
}

func (a *application) show() {
	if !a.visible {
		procShowWindow.Call(a.overlay, swShowNoActivate)
		a.visible = true
	}
	a.placeDirectlyAboveTarget()
}

func (a *application) hide() {
	if a.visible {
		procShowWindow.Call(a.overlay, swHide)
		a.visible = false
	}
}

func (a *application) placeDirectlyAboveTarget() {
	if a.cfg.windowMode == "child" {
		procSetWindowPos.Call(
			a.overlay, 0,
			0, 0, 0, 0,
			uintptr(swpNoMove|swpNoSize|swpNoActivate|swpNoOwnerZOrder),
		)
		return
	}
	above, _, _ := procGetWindow.Call(a.target, uintptr(gwHwndPrev))
	if above == a.overlay {
		above, _, _ = procGetWindow.Call(a.overlay, uintptr(gwHwndPrev))
	}
	procSetWindowPos.Call(
		a.overlay, above,
		0, 0, 0, 0,
		uintptr(swpNoMove|swpNoSize|swpNoActivate|swpNoOwnerZOrder),
	)
}

func (a *application) hasOccludingWindow() bool {
	targetRect, err := windowRect(a.target)
	if err != nil {
		return false
	}
	for candidate, _, _ := procGetWindow.Call(a.target, uintptr(gwHwndPrev)); candidate != 0; candidate, _, _ = procGetWindow.Call(candidate, uintptr(gwHwndPrev)) {
		if candidate == a.overlay || isIgnoredShellWindow(candidate) {
			continue
		}
		isVisible, _, _ := procIsWindowVisible.Call(candidate)
		isMinimized, _, _ := procIsIconic.Call(candidate)
		if isVisible == 0 || isMinimized != 0 || isCloaked(candidate) {
			continue
		}
		candidateRect, err := windowRect(candidate)
		if err == nil && rectanglesIntersect(targetRect, candidateRect) {
			return true
		}
	}
	return false
}

func isIgnoredShellWindow(hwnd uintptr) bool {
	class := windowClass(hwnd)
	switch class {
	case "Progman", "WorkerW", "Shell_TrayWnd", "Shell_SecondaryTrayWnd", "tooltips_class32":
		return true
	default:
		return false
	}
}

func rectanglesIntersect(a, b rect) bool {
	return a.Left < b.Right && a.Right > b.Left && a.Top < b.Bottom && a.Bottom > b.Top
}

func isCloaked(hwnd uintptr) bool {
	var cloaked uint32
	result, _, _ := procDwmGetWindowAttribute.Call(hwnd, uintptr(dwmwaCloaked), uintptr(unsafe.Pointer(&cloaked)), unsafe.Sizeof(cloaked))
	return result == 0 && cloaked != 0
}

func (a *application) renderOverlay(r rect) error {
	w, h := r.width(), r.height()
	canvas := image.NewNRGBA(image.Rect(0, 0, w, h))
	drawScaledLogo(canvas, a.cfg.logo, a.cfg.borderWidth)
	drawBorder(canvas, a.cfg.color, a.cfg.borderWidth, a.cfg.borderStyle)

	hdc, _, callErr := procCreateCompatibleDC.Call(0)
	if hdc == 0 {
		return fmt.Errorf("CreateCompatibleDC: %w", callErr)
	}
	defer procDeleteDC.Call(hdc)
	info := bitmapInfo{Header: bitmapInfoHeader{
		Size: uint32(unsafe.Sizeof(bitmapInfoHeader{})), Width: int32(w), Height: -int32(h),
		Planes: 1, BitCount: 32, Compression: biRGB, SizeImage: uint32(w * h * 4),
	}}
	var bits uintptr
	bitmap, _, callErr := procCreateDIBSection.Call(hdc, uintptr(unsafe.Pointer(&info)), uintptr(dibRGBColors), uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bitmap == 0 {
		return fmt.Errorf("CreateDIBSection: %w", callErr)
	}
	defer procDeleteObject.Call(bitmap)
	previous, _, _ := procSelectObject.Call(hdc, bitmap)
	defer procSelectObject.Call(hdc, previous)
	pixels := unsafe.Slice((*byte)(unsafe.Pointer(bits)), w*h*4)
	copyCanvasToBGRA(pixels, canvas)
	if err := drawWorkspaceName(hdc, pixels, w, h, a.cfg); err != nil {
		return err
	}

	if a.cfg.windowMode == "child" {
		applyTransparentColorKey(pixels)
		procShowWindow.Call(a.overlay, swShowNoActivate)
		windowDC, _, callErr := procGetDC.Call(a.overlay)
		if windowDC == 0 {
			return fmt.Errorf("GetDC: %w", callErr)
		}
		defer procReleaseDC.Call(a.overlay, windowDC)
		result, _, callErr := procBitBlt.Call(windowDC, 0, 0, uintptr(w), uintptr(h), hdc, 0, 0, uintptr(srcCopy))
		if result == 0 {
			return fmt.Errorf("BitBlt: %w", callErr)
		}
		return nil
	}

	destination := point{X: r.Left, Y: r.Top}
	source := point{}
	dimensions := size{CX: int32(w), CY: int32(h)}
	blend := blendFunction{BlendOp: acSrcOver, SourceConstantAlpha: 255, AlphaFormat: acSrcAlpha}
	result, _, callErr := procUpdateLayeredWindow.Call(
		a.overlay, 0,
		uintptr(unsafe.Pointer(&destination)), uintptr(unsafe.Pointer(&dimensions)), hdc,
		uintptr(unsafe.Pointer(&source)), 0, uintptr(unsafe.Pointer(&blend)), uintptr(ulwAlpha),
	)
	if result == 0 {
		return fmt.Errorf("UpdateLayeredWindow: %w", callErr)
	}
	return nil
}

func applyTransparentColorKey(pixels []byte) {
	for index := 0; index < len(pixels); index += 4 {
		if pixels[index+3] == 0 {
			// DIB byte order is B, G, R, A. COLORREF 0x00010203 is R=3, G=2, B=1.
			pixels[index], pixels[index+1], pixels[index+2] = 1, 2, 3
		}
	}
}

func drawScaledLogo(dst *image.NRGBA, src image.Image, borderWidth int) {
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	padding := borderWidth + 8
	maxSide := h / 3
	if maxSide > w-2*padding {
		maxSide = w - 2*padding
	}
	if maxSide > h-2*padding {
		maxSide = h - 2*padding
	}
	if maxSide <= 0 {
		return
	}
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	if sw <= 0 || sh <= 0 {
		return
	}
	scale := float64(maxSide) / float64(max(sw, sh))
	dw := max(1, int(math.Round(float64(sw)*scale)))
	dh := max(1, int(math.Round(float64(sh)*scale)))
	startX, startY := w-padding-dw, h-padding-dh
	for y := 0; y < dh; y++ {
		sy := sb.Min.Y + min(sh-1, y*sh/dh)
		for x := 0; x < dw; x++ {
			sx := sb.Min.X + min(sw-1, x*sw/dw)
			dst.Set(startX+x, startY+y, src.At(sx, sy))
		}
	}
}

func drawBorder(dst *image.NRGBA, c color.NRGBA, width int, style string) {
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	width = min(width, max(1, min(w, h)/2))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if borderPixel(x, y, w, h, width, style) {
				dst.SetNRGBA(x, y, c)
			}
		}
	}
}

func borderPixel(x, y, w, h, width int, style string) bool {
	distance := min(min(x, w-1-x), min(y, h-1-y))
	if distance < 0 || distance >= width {
		return false
	}
	switch style {
	case "double":
		band := max(1, width/3)
		return distance < band || distance >= width-band
	case "dashed", "dotted":
		coordinate := x
		if y >= width && y < h-width {
			coordinate = y
		}
		if style == "dashed" {
			period := max(4, width*5)
			return coordinate%period < max(2, width*3)
		}
		period := max(2, width*2)
		return coordinate%period < max(1, width)
	default:
		return true
	}
}

func copyCanvasToBGRA(dst []byte, src *image.NRGBA) {
	for y := 0; y < src.Bounds().Dy(); y++ {
		for x := 0; x < src.Bounds().Dx(); x++ {
			c := src.NRGBAAt(x, y)
			i := (y*src.Bounds().Dx() + x) * 4
			dst[i] = byte(uint16(c.B) * uint16(c.A) / 255)
			dst[i+1] = byte(uint16(c.G) * uint16(c.A) / 255)
			dst[i+2] = byte(uint16(c.R) * uint16(c.A) / 255)
			dst[i+3] = c.A
		}
	}
}

func drawWorkspaceName(hdc uintptr, pixels []byte, w, h int, cfg config) error {
	text, _ := syscall.UTF16FromString(cfg.name)
	textLength := len(text) - 1
	if textLength <= 0 {
		return nil
	}
	face, _ := syscall.UTF16PtrFromString(cfg.fontFamily)
	maxHeight := max(1, h/3)
	maxWidth := max(1, w-2*(cfg.borderWidth+16))
	fontSize := findFontSize(hdc, face, text, textLength, cfg.fontWeight, maxWidth, maxHeight)
	font := createFont(fontSize, cfg.fontWeight, face)
	if font == 0 {
		return errors.New("CreateFontW failed")
	}
	defer procDeleteObject.Call(font)
	previous, _, _ := procSelectObject.Call(hdc, font)
	defer procSelectObject.Call(hdc, previous)
	var measured size
	procGetTextExtentPoint32W.Call(hdc, uintptr(unsafe.Pointer(&text[0])), uintptr(textLength), uintptr(unsafe.Pointer(&measured)))
	x := (w - int(measured.CX)) / 2
	y := (h - int(measured.CY)) / 2

	// Give fully transparent pixels a sentinel RGB value. GDI does not update
	// the alpha channel, so the sentinel lets us identify anti-aliased glyph
	// pixels even when the requested text color is black.
	for i := 0; i < len(pixels); i += 4 {
		if pixels[i+3] == 0 {
			pixels[i], pixels[i+1], pixels[i+2] = 1, 2, 3
		}
	}
	procSetBkMode.Call(hdc, transparentBkMode)
	if cfg.shadow {
		procSetTextColor.Call(hdc, uintptr(colorRef(color.NRGBA{R: 32, G: 32, B: 32, A: 255})))
		procTextOutW.Call(hdc, uintptr(uint32(x+3)), uintptr(uint32(y+3)), uintptr(unsafe.Pointer(&text[0])), uintptr(textLength))
	}
	procSetTextColor.Call(hdc, uintptr(colorRef(cfg.color)))
	procTextOutW.Call(hdc, uintptr(uint32(x)), uintptr(uint32(y)), uintptr(unsafe.Pointer(&text[0])), uintptr(textLength))
	for i := 0; i < len(pixels); i += 4 {
		if pixels[i+3] == 0 {
			if pixels[i] == 1 && pixels[i+1] == 2 && pixels[i+2] == 3 {
				pixels[i], pixels[i+1], pixels[i+2] = 0, 0, 0
			} else {
				pixels[i+3] = 255
			}
		}
	}
	return nil
}

func findFontSize(hdc uintptr, face *uint16, text []uint16, textLength, weight, maxWidth, maxHeight int) int {
	low, high, best := 1, maxHeight, 1
	for low <= high {
		candidate := (low + high) / 2
		font := createFont(candidate, weight, face)
		if font == 0 {
			high = candidate - 1
			continue
		}
		previous, _, _ := procSelectObject.Call(hdc, font)
		var measured size
		procGetTextExtentPoint32W.Call(hdc, uintptr(unsafe.Pointer(&text[0])), uintptr(textLength), uintptr(unsafe.Pointer(&measured)))
		procSelectObject.Call(hdc, previous)
		procDeleteObject.Call(font)
		if int(measured.CX) <= maxWidth && int(measured.CY) <= maxHeight {
			best = candidate
			low = candidate + 1
		} else {
			high = candidate - 1
		}
	}
	return best
}

func createFont(pixelHeight, weight int, face *uint16) uintptr {
	font, _, _ := procCreateFontW.Call(
		uintptr(uint32(int32(-pixelHeight))), 0, 0, 0, uintptr(weight),
		0, 0, 0, defaultCharset, 0, 0, antialiasedQuality, defaultPitch,
		uintptr(unsafe.Pointer(face)),
	)
	return font
}

func colorRef(c color.NRGBA) uint32 { return uint32(c.R) | uint32(c.G)<<8 | uint32(c.B)<<16 }

func signedInt32Argument(value int32) uintptr { return uintptr(uint32(value)) }

func foregroundWindow() uintptr { hwnd, _, _ := procGetForegroundWindow.Call(); return hwnd }
func asyncKeyDown(key uintptr) bool {
	return asyncKeyState(key)&0x8000 != 0
}

func asyncKeyState(key uintptr) uintptr {
	result, _, _ := procGetAsyncKeyState.Call(key)
	return result
}

func lastInputTime() uint32 {
	info := lastInputInfo{CbSize: uint32(unsafe.Sizeof(lastInputInfo{}))}
	result, _, _ := procGetLastInputInfo.Call(uintptr(unsafe.Pointer(&info)))
	if result == 0 {
		return 0
	}
	return info.DwTime
}

func getTickCount64() uint64 {
	proc := kernel32.NewProc("GetTickCount64")
	result, _, _ := proc.Call()
	return uint64(result)
}

func windowRect(hwnd uintptr) (rect, error) {
	var r rect
	result, _, callErr := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	if result == 0 {
		return r, callErr
	}
	return r, nil
}

func clientRect(hwnd uintptr) (rect, error) {
	var r rect
	result, _, callErr := procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	if result == 0 {
		return r, callErr
	}
	return r, nil
}

func windowClass(hwnd uintptr) string {
	buffer := make([]uint16, 256)
	length, _, _ := procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	return syscall.UTF16ToString(buffer[:length])
}

func windowTitle(hwnd uintptr) string {
	buffer := make([]uint16, 1024)
	length, _, _ := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	return syscall.UTF16ToString(buffer[:length])
}

func processImagePath(hwnd uintptr) (string, error) {
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return "", errors.New("GetWindowThreadProcessId returned no process")
	}
	process, _, callErr := procOpenProcess.Call(uintptr(processQueryLimitedInformation), 0, uintptr(pid))
	if process == 0 {
		return "", callErr
	}
	defer procCloseHandle.Call(process)
	buffer := make([]uint16, 1024)
	size := uint32(len(buffer))
	result, _, callErr := procQueryFullProcessImageNameW.Call(process, 0, uintptr(unsafe.Pointer(&buffer[0])), uintptr(unsafe.Pointer(&size)))
	if result == 0 {
		return "", callErr
	}
	return syscall.UTF16ToString(buffer[:size]), nil
}
