//go:build windows

package main

import (
	"bufio"
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
	wmDisplayChange = uint32(0x007E)
	wmTimer         = uint32(0x0113)

	htTransparent = ^uintptr(0)
	maNoActivate  = uintptr(3)

	swMinimize       = 6
	swRestore        = 9
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

	vkLButton  = uintptr(0x01)
	vkRButton  = uintptr(0x02)
	vkMButton  = uintptr(0x04)
	vkXButton1 = uintptr(0x05)
	vkXButton2 = uintptr(0x06)
	vkShift    = uintptr(0x10)
	vkMenu     = uintptr(0x12)
	vkTab      = uintptr(0x09)

	processQueryLimitedInformation = uint32(0x1000)
	dwmwaCloaked                   = uint32(14)
	dwmwaExtendedFrameBounds       = uint32(9)

	eventSystemMinimizeStart = uint32(0x0016)
	eventSystemMinimizeEnd   = uint32(0x0017)
	objidWindow              = int32(0)
	childidSelf              = int32(0)
	wineventOutofcontext     = uint32(0x0000)

	qdcDatabaseCurrent         = uint32(0x00000004)
	displayConfigTopologyClone = uint32(0x00000002)
	errorInsufficientBuffer    = uint32(122)
	displayConfigQueryAttempts = 3
	displayTopologyRetryMS     = uint64(1000)
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	dwmapi   = syscall.NewLazyDLL("dwmapi.dll")

	procRegisterClassExW              = user32.NewProc("RegisterClassExW")
	procEnumWindows                   = user32.NewProc("EnumWindows")
	procGetCursorPos                  = user32.NewProc("GetCursorPos")
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
	procSetWinEventHook               = user32.NewProc("SetWinEventHook")
	procUnhookWinEvent                = user32.NewProc("UnhookWinEvent")
	procGetDisplayConfigBufferSizes   = user32.NewProc("GetDisplayConfigBufferSizes")
	procQueryDisplayConfig            = user32.NewProc("QueryDisplayConfig")

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
	procDwmFlush              = dwmapi.NewProc("DwmFlush")
)

type point struct{ X, Y int32 }
type size struct{ CX, CY int32 }
type rect struct{ Left, Top, Right, Bottom int32 }
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

// Only the sizes and alignment of these CCD structures matter here: Halo asks
// QueryDisplayConfig for the topology identifier and does not inspect paths or
// modes. The fields mirror DISPLAYCONFIG_PATH_INFO and DISPLAYCONFIG_MODE_INFO.
type displayConfigLUID struct {
	LowPart  uint32
	HighPart int32
}

type displayConfigPathSourceInfo struct {
	AdapterID   displayConfigLUID
	ID          uint32
	ModeInfoIdx uint32
	StatusFlags uint32
}

type displayConfigPathTargetInfo struct {
	AdapterID        displayConfigLUID
	ID               uint32
	ModeInfoIdx      uint32
	OutputTechnology uint32
	Rotation         uint32
	Scaling          uint32
	RefreshRate      [2]uint32
	ScanLineOrdering uint32
	TargetAvailable  int32
	StatusFlags      uint32
}

type displayConfigPathInfo struct {
	SourceInfo displayConfigPathSourceInfo
	TargetInfo displayConfigPathTargetInfo
}

type displayConfigModeInfo struct {
	InfoType  uint32
	ID        uint32
	AdapterID displayConfigLUID
	Mode      [6]uint64
}

type config struct {
	name           string
	logo           image.Image
	color          color.NRGBA
	borderWidth    int
	borderStyle    string
	fontFamily     string
	fontWeight     int
	shadow         bool
	pill           bool
	pillOpacity    int
	pillMargin     int
	borderSegment  int
	logPath        string
	allowAnyWindow bool
	targetOverride uintptr
	waitForVSCode  time.Duration
	focusHandshake bool
	windowMode     string
	startupWarning string
}

type application struct {
	cfg               config
	target            uintptr
	overlay           uintptr
	targetRect        rect
	minimizeHook      uintptr
	minimizeState     minimizePhase
	minimizeReplayAt  uint64
	renderedRect      rect
	visible           bool
	visibilityReason  string
	activationVisible bool
	wasFocused        bool
	manualVisible     bool
	altTabVisible     bool
	taskbarHover      bool
	taskbarHit        bool
	thumbnailHit      bool
	cursorPos         point
	shiftDown         bool
	lastShiftRelease  uint64
	lastManualInput   uint32
	displayTopology   uint32
	topologyKnown     bool
	topologyDirty     bool
	topologyRetryAt   uint64
	logger            *log.Logger
	logFile           *os.File
}

var (
	activeApp                *application
	minimizeWinEventCallback = syscall.NewCallback(minimizeWinEventProc)
)

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

	focused := foregroundWindow() == target
	app := &application{
		cfg:               cfg,
		target:            target,
		activationVisible: true,
		wasFocused:        focused,
		logger:            logger,
		logFile:           logFile,
	}
	activeApp = app
	if err := app.refreshDisplayTopology(); err != nil {
		logger.Printf("initial display topology query failed: %v", err)
	}
	if err := app.createOverlay(); err != nil {
		fatalf("create overlay: %v", err)
	}
	defer app.close()
	if err := app.updateGeometryAndBitmap(); err != nil {
		logger.Printf("initial activation render error: %v", err)
	} else if err := app.show(); err != nil {
		logger.Printf("initial activation show error: %v", err)
	} else {
		app.visibilityReason = "activation"
	}
	if err := app.installMinimizeHook(); err != nil {
		fatalf("watch minimize events: %v", err)
	}

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
	flag.BoolVar(&cfg.pill, "pill", true, "draw a contrast pill behind the workspace name")
	flag.IntVar(&cfg.pillOpacity, "pill-opacity", 100, "pill opacity percentage, 0 to 100")
	flag.IntVar(&cfg.pillMargin, "pill-margin", 50, "minimal left and right margin of the pill and name, in pixels")
	flag.IntVar(&cfg.borderSegment, "border-segment", 50, "length of the alternating black border segments, 0 for a continuous border")
	flag.StringVar(&cfg.logPath, "log", filepath.Join(os.TempDir(), "workspace-halo-companion.log"), "companion log file")
	flag.BoolVar(&cfg.allowAnyWindow, "allow-any-window", false, "allow binding to a non-Code.exe foreground window for diagnostics")
	flag.StringVar(&hwndValue, "hwnd", "", "optional target HWND in decimal or 0x-prefixed hexadecimal")
	flag.DurationVar(&cfg.waitForVSCode, "wait-for-vscode", time.Minute, "time to wait for a stable VS Code window to become foreground")
	flag.BoolVar(&cfg.focusHandshake, "focus-handshake", false, "require the launching extension to confirm the foreground window over stdin")
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
	if cfg.pillOpacity < 0 || cfg.pillOpacity > 100 {
		return cfg, errors.New("--pill-opacity must be between 0 and 100")
	}
	if cfg.pillMargin < 0 {
		return cfg, errors.New("--pill-margin must be zero or positive")
	}
	if cfg.borderSegment < 0 {
		return cfg, errors.New("--border-segment must be zero or positive")
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
	if cfg.focusHandshake {
		target, err := uniqueWorkspaceCodeWindow(cfg.name)
		if err == nil {
			logger.Printf("identified unique workspace window without focus hwnd=0x%X", target)
			fmt.Fprintln(os.Stdout, "workspace-halo-bound startup")
			return target, nil
		}
		logger.Printf("startup window identification unavailable: %v", err)
		fmt.Fprintln(os.Stdout, "workspace-halo-ready")
		return acquireTargetByFocusHandshake(os.Stdin, os.Stdout, foregroundCodeWindow, logger)
	}

	deadline := time.Now().Add(cfg.waitForVSCode)
	logger.Printf("waiting up to %s; focus the stable VS Code window to identify", cfg.waitForVSCode)
	for {
		target, err := foregroundCodeWindow()
		if err == nil {
			return target, nil
		}
		if cfg.waitForVSCode <= 0 || time.Now().After(deadline) {
			return 0, errors.New("no stable VS Code window became foreground before the wait expired")
		}
		time.Sleep(100 * time.Millisecond)
	}
}

type foregroundCodeWindowFunc func() (uintptr, error)

// acquireTargetByFocusHandshake prevents a host launched by one VS Code
// extension window from adopting another VS Code window during process startup.
// A focus message tells the host when the launching extension believes its own
// window is focused. The host proposes the foreground HWND, then attaches only
// if a matching confirmation arrives while that exact HWND is still foreground.
func acquireTargetByFocusHandshake(
	input io.Reader,
	output io.Writer,
	foreground foregroundCodeWindowFunc,
	logger *log.Logger,
) (uintptr, error) {
	scanner := bufio.NewScanner(input)
	var candidate uintptr
	var candidateToken string
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 {
			logger.Printf("ignored malformed focus-handshake command %q", scanner.Text())
			continue
		}
		command, token := fields[0], fields[1]
		switch command {
		case "focus":
			target, err := foreground()
			if err != nil {
				candidate, candidateToken = 0, ""
				logger.Printf("focus proposal %s rejected: %v", token, err)
				fmt.Fprintf(output, "workspace-halo-rejected %s\n", token)
				continue
			}
			candidate, candidateToken = target, token
			logger.Printf("focus proposal %s candidate hwnd=0x%X", token, target)
			fmt.Fprintf(output, "workspace-halo-candidate %s\n", token)
		case "blur":
			candidate, candidateToken = 0, ""
		case "confirm":
			if candidate == 0 || token != candidateToken {
				continue
			}
			target, err := foreground()
			if err != nil || target != candidate {
				logger.Printf("focus proposal %s became stale before confirmation", token)
				candidate, candidateToken = 0, ""
				fmt.Fprintf(output, "workspace-halo-rejected %s\n", token)
				continue
			}
			fmt.Fprintf(output, "workspace-halo-bound %s\n", token)
			return candidate, nil
		default:
			logger.Printf("ignored unknown focus-handshake command %q", command)
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("read focus handshake: %w", err)
	}
	return 0, errors.New("focus handshake closed before a window was confirmed")
}

func foregroundCodeWindow() (uintptr, error) {
	target := foregroundWindow()
	if target == 0 {
		return 0, errors.New("no foreground window is available")
	}
	path, err := processImagePath(target)
	if err != nil {
		return 0, err
	}
	if !strings.EqualFold(filepath.Base(path), "Code.exe") {
		return 0, fmt.Errorf("foreground window belongs to %q, not stable VS Code (Code.exe)", path)
	}
	return target, nil
}

type workspaceWindowSearch struct {
	name    string
	matches []uintptr
}

var activeWorkspaceWindowSearch *workspaceWindowSearch

var workspaceCodeWindowEnumProc = syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
	search := activeWorkspaceWindowSearch
	if search == nil {
		return 0
	}
	if visible, _, _ := procIsWindowVisible.Call(hwnd); visible == 0 {
		return 1
	}
	path, err := processImagePath(hwnd)
	if err != nil || !strings.EqualFold(filepath.Base(path), "Code.exe") {
		return 1
	}
	if windowTitleMatchesWorkspace(windowTitle(hwnd), search.name) {
		search.matches = append(search.matches, hwnd)
	}
	return 1
})

// uniqueWorkspaceCodeWindow identifies the launching VS Code window without
// requiring focus when its workspace name appears as an exact title segment.
// Ambiguous and customized titles fail closed into the focus handshake.
func uniqueWorkspaceCodeWindow(name string) (uintptr, error) {
	search := &workspaceWindowSearch{name: name}
	activeWorkspaceWindowSearch = search
	procEnumWindows.Call(workspaceCodeWindowEnumProc, 0)
	activeWorkspaceWindowSearch = nil
	if len(search.matches) != 1 {
		return 0, fmt.Errorf("workspace title %q matched %d stable VS Code windows", name, len(search.matches))
	}
	return search.matches[0], nil
}

func windowTitleMatchesWorkspace(title, workspaceName string) bool {
	for _, segment := range []string{workspaceName, workspaceName + " (Workspace)"} {
		suffix := segment + " - Visual Studio Code"
		if title == suffix || strings.HasSuffix(title, " - "+suffix) {
			return true
		}
	}
	return false
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
	case wmDisplayChange:
		if activeApp != nil {
			// Query on the next timer tick, after Windows has finished
			// applying every path involved in the Win+P transition.
			activeApp.topologyDirty = true
			activeApp.topologyRetryAt = 0
		}
		return 0
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
	if a.minimizeHook != 0 {
		procUnhookWinEvent.Call(a.minimizeHook)
		a.minimizeHook = 0
	}
	if a.overlay != 0 {
		procKillTimer.Call(a.overlay, 1)
		procDestroyWindow.Call(a.overlay)
	}
}

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

func (a *application) tick() error {
	if ok, _, _ := procIsWindow.Call(a.target); ok == 0 {
		a.logger.Printf("target window closed")
		procPostMessageW.Call(a.overlay, uintptr(wmClose), 0, 0)
		return nil
	}
	now := getTickCount64()
	a.replayPendingMinimize(now)
	if a.topologyDirty && now >= a.topologyRetryAt {
		if err := a.refreshDisplayTopology(); err != nil {
			// Suppress ambient occlusion while topology is unsettled. A later
			// tick retries, so a transient CCD race does not disable it forever.
			a.topologyRetryAt = now + displayTopologyRetryMS
			a.logger.Printf("display topology refresh failed: %v", err)
		} else {
			a.topologyDirty = false
			a.topologyRetryAt = 0
		}
	}

	focused := foregroundWindow() == a.target
	pointerDown := false
	if a.activationVisible && focused {
		pointerDown = pointerButtonDown()
	}
	if shouldDismissActivation(a.activationVisible, focused, a.wasFocused, pointerDown) {
		a.activationVisible = false
	}
	a.wasFocused = focused
	a.pollDoubleShift(focused)
	a.pollAltTab()
	a.pollTaskbarHover()
	if a.manualVisible {
		if !focused || lastInputTime() != a.lastManualInput {
			a.manualVisible = false
		}
	}

	minimized, _, _ := procIsIconic.Call(a.target)
	visible, _, _ := procIsWindowVisible.Call(a.target)
	occluded := false
	if shouldCheckOcclusion(
		focused,
		minimized != 0,
		visible != 0,
		a.topologyDirty,
		a.displayTopology,
	) {
		occluded = a.hasOccludingWindow()
	}

	desired, reason := visibilityState(
		a.manualVisible,
		a.activationVisible,
		minimized != 0 || a.minimizeState != minimizeIdle,
		focused,
		a.altTabVisible,
		a.taskbarHover,
		occluded,
	)
	if desired {
		if err := a.updateGeometryAndBitmap(); err != nil {
			return err
		}
		if err := a.show(); err != nil {
			return err
		}
	} else {
		if err := a.hide(); err != nil {
			return err
		}
	}
	if reason != a.visibilityReason {
		a.logger.Printf("visibility=%s focused=%t minimized=%t", reason, focused, minimized != 0)
		a.visibilityReason = reason
	}
	return nil
}

func shouldCheckOcclusion(focused, minimized, visible, topologyPending bool, topology uint32) bool {
	return !focused &&
		!minimized &&
		visible &&
		!topologyPending &&
		topology != displayConfigTopologyClone
}

func (a *application) refreshDisplayTopology() error {
	topology, err := currentDisplayTopology()
	if err != nil {
		return err
	}
	changed := !a.topologyKnown || topology != a.displayTopology
	a.displayTopology = topology
	a.topologyKnown = true
	if changed {
		a.logger.Printf("display topology=%s", displayTopologyName(topology))
	}
	return nil
}

func displayTopologyName(topology uint32) string {
	switch topology {
	case 0x00000001:
		return "internal"
	case displayConfigTopologyClone:
		return "clone"
	case 0x00000004:
		return "extend"
	case 0x00000008:
		return "external"
	default:
		return fmt.Sprintf("unknown(0x%X)", topology)
	}
}

// currentDisplayTopology reads the topology selected by Win+P from Windows'
// persisted display database. Buffer sizing and querying can race with a
// topology transition, so retry the documented insufficient-buffer result.
func currentDisplayTopology() (uint32, error) {
	for attempt := 0; attempt < displayConfigQueryAttempts; attempt++ {
		var pathCount, modeCount uint32
		result, _, _ := procGetDisplayConfigBufferSizes.Call(
			uintptr(qdcDatabaseCurrent),
			uintptr(unsafe.Pointer(&pathCount)),
			uintptr(unsafe.Pointer(&modeCount)),
		)
		if code := uint32(result); code != 0 {
			return 0, fmt.Errorf("GetDisplayConfigBufferSizes: %w", syscall.Errno(code))
		}
		// The pointers are required even if the persistence database has not
		// materialized mode records for the current set of connected displays.
		paths := make([]displayConfigPathInfo, max(pathCount, 1))
		modes := make([]displayConfigModeInfo, max(modeCount, 1))
		var topology uint32
		result, _, _ = procQueryDisplayConfig.Call(
			uintptr(qdcDatabaseCurrent),
			uintptr(unsafe.Pointer(&pathCount)),
			uintptr(unsafe.Pointer(&paths[0])),
			uintptr(unsafe.Pointer(&modeCount)),
			uintptr(unsafe.Pointer(&modes[0])),
			uintptr(unsafe.Pointer(&topology)),
		)
		code := uint32(result)
		if code == 0 {
			return topology, nil
		}
		if code != errorInsufficientBuffer {
			return 0, fmt.Errorf("QueryDisplayConfig: %w", syscall.Errno(code))
		}
	}
	return 0, errors.New("display configuration kept changing while it was queried")
}

func shouldDismissActivation(activationVisible, focused, wasFocused, pointerDown bool) bool {
	return activationVisible && focused && (!wasFocused || pointerDown)
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

// visibilityState limits rendering to explicit user and shell triggers, the
// initial activation, a minimized target, and actual overlap. Focus suppresses
// only ambient overlap after activation has been acknowledged.
func visibilityState(manual, activation, minimized, focused, altTab, taskbarHover, occluded bool) (bool, string) {
	switch {
	case manual:
		return true, "double-shift"
	case activation:
		return true, "activation"
	case altTab:
		return true, "alt-tab"
	case taskbarHover:
		return true, "taskbar-hover"
	case minimized:
		return true, "minimized"
	case focused:
		return false, "focused"
	case occluded:
		return true, "occluded"
	default:
		return false, "hidden"
	}
}

// isTaskbarClass recognizes the taskbar windows themselves.
func isTaskbarClass(class string) bool {
	return class == "Shell_TrayWnd" || class == "Shell_SecondaryTrayWnd"
}

// isTaskbarThumbnailClass recognizes Explorer's taskbar thumbnail flyouts.
// Windows 11 can host them in XAML islands, including Windows App SDK islands.
// Task View's MultitaskingViewFrame is intentionally excluded.
func isTaskbarThumbnailClass(class string) bool {
	return class == "TaskListThumbnailWnd" ||
		class == "Microsoft.UI.Content.PopupWindowSiteBridge" ||
		strings.HasPrefix(class, "XamlExplorerHostIslandWindow")
}

func pointInRect(p point, r rect) bool {
	return p.X >= r.Left && p.X < r.Right && p.Y >= r.Top && p.Y < r.Bottom
}

// taskbarEnumProc visits top-level windows until it finds a taskbar or an
// Explorer-hosted taskbar thumbnail flyout under the cursor. Shell flyouts
// live in z-bands that a sibling walk from an application window cannot see,
// so detection goes through EnumWindows.
var taskbarEnumProc = syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
	app := activeApp
	if app == nil {
		return 0
	}
	if isVisible, _, _ := procIsWindowVisible.Call(hwnd); isVisible == 0 {
		return 1
	}
	class := windowClass(hwnd)
	if isTaskbarClass(class) {
		if r, err := windowRect(hwnd); err == nil && pointInRect(app.cursorPos, r) {
			app.taskbarHit = true
			return 0
		}
		return 1
	}
	if !isTaskbarThumbnailClass(class) || isCloaked(hwnd) {
		return 1
	}
	r, err := windowRect(hwnd)
	if err != nil || r.width() <= 2 || r.height() <= 2 || !pointInRect(app.cursorPos, r) {
		return 1
	}
	path, err := processImagePath(hwnd)
	if err == nil && strings.EqualFold(filepath.Base(path), "explorer.exe") {
		app.thumbnailHit = true
		return 0
	}
	return 1
})

func taskbarHoverState(taskbarHit, thumbnailHit, topologyPending bool, topology uint32) bool {
	return taskbarHit ||
		(thumbnailHit && !topologyPending && topology != displayConfigTopologyClone)
}

// pollTaskbarHover reports whether the pointer is over a taskbar or, outside
// Duplicate mode, one of its thumbnail previews.
func (a *application) pollTaskbarHover() {
	a.taskbarHit = false
	a.thumbnailHit = false
	a.cursorPos = point{X: -2147483648, Y: -2147483648}
	if result, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&a.cursorPos))); result == 0 {
		a.cursorPos = point{X: -2147483648, Y: -2147483648}
	}
	procEnumWindows.Call(taskbarEnumProc, 0)
	hover := taskbarHoverState(a.taskbarHit, a.thumbnailHit, a.topologyDirty, a.displayTopology)
	if hover && !a.taskbarHover {
		a.logger.Printf("taskbar hover")
	}
	a.taskbarHover = hover
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
	triggered, nextRelease := doubleShiftRelease(a.lastShiftRelease, now)
	a.lastShiftRelease = nextRelease
	if triggered {
		a.manualVisible = true
		a.lastManualInput = lastInputTime()
		a.logger.Printf("double-shift gesture")
	}
}

func doubleShiftRelease(lastRelease, now uint64) (triggered bool, nextRelease uint64) {
	if lastRelease != 0 && now-lastRelease <= 400 {
		return true, 0
	}
	return false, now
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

func flushDwm() error {
	result, _, _ := procDwmFlush.Call()
	if int32(result) < 0 {
		return fmt.Errorf("DwmFlush: HRESULT 0x%08X", uint32(result))
	}
	return nil
}

func (a *application) show() error {
	if !a.visible {
		procShowWindow.Call(a.overlay, swShowNoActivate)
		a.visible = true
	}
	a.placeDirectlyAboveTarget()
	return nil
}

func (a *application) hide() error {
	if a.visible {
		procShowWindow.Call(a.overlay, swHide)
		a.visible = false
	}
	return nil
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

// visibleFrameRect excludes the invisible resize borders included by
// GetWindowRect. Those borders can cross an adjacent monitor boundary and
// otherwise make two visually separate maximized windows appear to overlap.
func visibleFrameRect(hwnd uintptr) (rect, error) {
	var r rect
	result, _, _ := procDwmGetWindowAttribute.Call(
		hwnd,
		uintptr(dwmwaExtendedFrameBounds),
		uintptr(unsafe.Pointer(&r)),
		unsafe.Sizeof(r),
	)
	if int32(result) >= 0 && r.width() > 0 && r.height() > 0 {
		return r, nil
	}
	return windowRect(hwnd)
}

func (a *application) hasOccludingWindow() bool {
	targetRect, err := visibleFrameRect(a.target)
	if err != nil {
		return false
	}
	for candidate, _, _ := procGetWindow.Call(a.target, uintptr(gwHwndPrev)); candidate != 0; candidate, _, _ = procGetWindow.Call(candidate, uintptr(gwHwndPrev)) {
		if candidate == a.overlay || isIgnoredOccludingWindow(candidate) {
			continue
		}
		isVisible, _, _ := procIsWindowVisible.Call(candidate)
		isMinimized, _, _ := procIsIconic.Call(candidate)
		if isVisible == 0 || isMinimized != 0 || isCloaked(candidate) {
			continue
		}
		candidateRect, err := visibleFrameRect(candidate)
		if err == nil && rectanglesIntersect(targetRect, candidateRect) {
			return true
		}
	}
	return false
}

func isIgnoredOccludingWindow(hwnd uintptr) bool {
	return isIgnoredOccludingClass(windowClass(hwnd))
}

func isIgnoredOccludingClass(class string) bool {
	switch class {
	case "WorkspaceHaloOverlay", "Progman", "WorkerW", "Shell_TrayWnd", "Shell_SecondaryTrayWnd", "tooltips_class32":
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
	drawBorder(canvas, a.cfg.color, a.cfg.borderWidth, a.cfg.borderStyle, a.cfg.borderSegment)

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

// drawBorder paints the halo border. With a positive segment length the
// border is continuous and alternates runs of the halo color and of opaque
// black along the perimeter: the black runs are painted, never transparent,
// so the border always reads as one unbroken frame. A zero segment keeps
// the plain motif rendering.
func drawBorder(dst *image.NRGBA, c color.NRGBA, width int, style string, segment int) {
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	width = min(width, max(1, min(w, h)/2))
	shape := style
	if segment > 0 {
		shape = "solid"
	}
	black := color.NRGBA{A: 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !borderPixel(x, y, w, h, width, shape) {
				continue
			}
			if segment > 0 && segmentIsBlack(x, y, w, h, width, segment) {
				dst.SetNRGBA(x, y, black)
				continue
			}
			dst.SetNRGBA(x, y, c)
		}
	}
}

// segmentIsBlack alternates along the border direction: the horizontal
// bands follow x and the vertical bands follow y, like the dashed motif.
func segmentIsBlack(x, y, w, h, width, segment int) bool {
	coordinate := x
	if y >= width && y < h-width {
		coordinate = y
	}
	return (coordinate/segment)%2 == 1
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
	maxWidth := max(1, w-2*max(cfg.pillMargin, cfg.borderWidth+16))
	fontSize := findFontSize(hdc, face, text, textLength, cfg.fontWeight, maxWidth, maxHeight, cfg.pill)
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

	if cfg.pill {
		drawNamePill(pixels, w, h, x, y, int(measured.CX), int(measured.CY), cfg)
	}

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

// relativeLuminance is the WCAG relative luminance of an sRGB color, the
// basis of the contrast-ratio computation.
func relativeLuminance(c color.NRGBA) float64 {
	linear := func(channel byte) float64 {
		v := float64(channel) / 255
		if v <= 0.03928 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*linear(c.R) + 0.7152*linear(c.G) + 0.0722*linear(c.B)
}

// contrastRatio is the WCAG contrast ratio between two colors, from 1 to 21.
func contrastRatio(a, b color.NRGBA) float64 {
	la, lb := relativeLuminance(a), relativeLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// pillBackgroundColor computes the pill color from the text color: the WCAG
// break-even (L+0.05)^2 = 1.05*0.05 picks the higher-contrast pole, white
// under a dark text and black under a light one, and one eighth of the text
// color tints that pole so the pill visibly carries the name's hue. When
// the tint would drop the contrast under 3:1, the WCAG minimum for large
// text (and the halo name is always large), the pure pole wins: the pole
// always reaches at least 4.58:1.
func pillBackgroundColor(text color.NRGBA) color.NRGBA {
	pole := color.NRGBA{A: 255}
	if relativeLuminance(text) <= 0.1791287847 {
		pole = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	}
	tinted := color.NRGBA{
		R: byte((7*int(pole.R) + int(text.R)) / 8),
		G: byte((7*int(pole.G) + int(text.G)) / 8),
		B: byte((7*int(pole.B) + int(text.B)) / 8),
		A: 255,
	}
	if contrastRatio(text, tinted) < 3 {
		return pole
	}
	return tinted
}

// pillPixel reports whether the pixel belongs to the pill: a core rectangle
// with a half-disc cap of radius (bottom-top)/2 at each end.
func pillPixel(x, y, left, top, right, bottom int) bool {
	if y < top || y >= bottom || x < left || x >= right {
		return false
	}
	radius := (bottom - top) / 2
	if x >= left+radius && x < right-radius {
		return true
	}
	centerX := left + radius
	if x >= right-radius {
		centerX = right - radius
	}
	dx, dy := x-centerX, y-(top+radius)
	return dx*dx+dy*dy <= radius*radius
}

// bayer4 is the 4x4 ordered-dither matrix. The color-key transparency of
// the child overlay is binary, so an opacity percentage is rendered as the
// matching share of pill pixels, spread evenly by the matrix.
var bayer4 = [4][4]int{
	{0, 8, 2, 10},
	{12, 4, 14, 6},
	{3, 11, 1, 9},
	{15, 7, 13, 5},
}

func ditherKeep(x, y, opacity int) bool {
	return bayer4[y&3][x&3]*100 < opacity*16
}

// pillBounds hugs the pill to the ink of the name. The GDI text cell is
// taller than the visible glyphs: internal leading sits above them, about a
// seventh of the cell for common UI fonts, so the pill trims it and keeps
// only a sliver below the descent. The horizontal pad stays small; the
// rounded caps provide the visual breathing room at the ends.
func pillBounds(textX, textY, textW, textH int) (left, top, right, bottom int) {
	top = textY + textH/7
	bottom = textY + textH + textH/30
	padX := textH / 3
	left = textX - padX
	right = textX + textW + padX
	return left, top, right, bottom
}

// drawNamePill fills the contrast pill behind the workspace name directly
// into the BGRA buffer, before GDI draws the letters, so their
// anti-aliasing blends against the pill color.
func drawNamePill(pixels []byte, w, h, textX, textY, textW, textH int, cfg config) {
	background := pillBackgroundColor(cfg.color)
	left, top, right, bottom := pillBounds(textX, textY, textW, textH)
	// The font fitting already reserves the margins; this clamp only guards
	// the integer rounding of the cap overhang.
	left = max(left, cfg.pillMargin)
	right = min(right, w-cfg.pillMargin)
	for y := max(0, top); y < min(h, bottom); y++ {
		for x := max(0, left); x < min(w, right); x++ {
			if !pillPixel(x, y, left, top, right, bottom) || !ditherKeep(x, y, cfg.pillOpacity) {
				continue
			}
			i := (y*w + x) * 4
			pixels[i] = background.B
			pixels[i+1] = background.G
			pixels[i+2] = background.R
			pixels[i+3] = 255
		}
	}
}

// findFontSize fits the name into maxWidth and maxHeight; with the pill on,
// the fitted width includes the pill cap overhang of a third of the text
// height on each side, so the whole pill respects the margins.
func findFontSize(hdc uintptr, face *uint16, text []uint16, textLength, weight, maxWidth, maxHeight int, pill bool) int {
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
		width := int(measured.CX)
		if pill {
			width += 2 * (int(measured.CY) / 3)
		}
		if width <= maxWidth && int(measured.CY) <= maxHeight {
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

func pointerButtonDown() bool {
	for _, button := range [...]uintptr{vkLButton, vkRButton, vkMButton, vkXButton1, vkXButton2} {
		if asyncKeyDown(button) {
			return true
		}
	}
	return false
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
