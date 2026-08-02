//go:build windows

package main

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"io"
	"log"
	"math"
	"strings"
	"testing"
	"unsafe"
)

func TestFocusHandshakeRejectsWindowSwitchBeforeConfirmation(t *testing.T) {
	foregrounds := []uintptr{0xA, 0xB, 0xB, 0xB}
	probe := func() (uintptr, error) {
		if len(foregrounds) == 0 {
			t.Fatal("foreground probe called too many times")
		}
		target := foregrounds[0]
		foregrounds = foregrounds[1:]
		return target, nil
	}
	var output bytes.Buffer
	target, err := acquireTargetByFocusHandshake(
		strings.NewReader("focus 1\nconfirm 1\nfocus 2\nconfirm 2\n"),
		&output,
		probe,
		log.New(io.Discard, "", 0),
	)
	if err != nil {
		t.Fatalf("focus handshake: %v", err)
	}
	if target != 0xB {
		t.Fatalf("bound target = 0x%X, want the freshly confirmed 0xB", target)
	}
	transcript := output.String()
	for _, message := range []string{
		"workspace-halo-candidate 1",
		"workspace-halo-rejected 1",
		"workspace-halo-candidate 2",
		"workspace-halo-bound 2",
	} {
		if !strings.Contains(transcript, message) {
			t.Errorf("protocol transcript %q does not contain %q", transcript, message)
		}
	}
}

func TestFocusHandshakeIgnoresConfirmationAfterBlur(t *testing.T) {
	foregrounds := []uintptr{0xA, 0xB, 0xB}
	probe := func() (uintptr, error) {
		target := foregrounds[0]
		foregrounds = foregrounds[1:]
		return target, nil
	}
	var output bytes.Buffer
	target, err := acquireTargetByFocusHandshake(
		strings.NewReader("focus 1\nblur 2\nconfirm 1\nfocus 3\nconfirm 3\n"),
		&output,
		probe,
		log.New(io.Discard, "", 0),
	)
	if err != nil {
		t.Fatalf("focus handshake: %v", err)
	}
	if target != 0xB {
		t.Fatalf("bound target = 0x%X, want 0xB after the fresh focus token", target)
	}
	if strings.Contains(output.String(), "workspace-halo-bound 1") {
		t.Fatalf("stale token was bound: %q", output.String())
	}
}

func TestFocusHandshakeFailsClosedWhenForegroundIsNotVSCode(t *testing.T) {
	calls := 0
	probe := func() (uintptr, error) {
		calls++
		if calls == 1 {
			return 0, errors.New("foreground is not Code.exe")
		}
		return 0xA, nil
	}
	var output bytes.Buffer
	target, err := acquireTargetByFocusHandshake(
		strings.NewReader("focus 1\nfocus 2\nconfirm 2\n"),
		&output,
		probe,
		log.New(io.Discard, "", 0),
	)
	if err != nil {
		t.Fatalf("focus handshake: %v", err)
	}
	if target != 0xA {
		t.Fatalf("bound target = 0x%X, want 0xA", target)
	}
	if !strings.Contains(output.String(), "workspace-halo-rejected 1") {
		t.Fatalf("invalid foreground was not rejected: %q", output.String())
	}
}

func TestWindowTitleMatchesWorkspace(t *testing.T) {
	tests := []struct {
		title string
		name  string
		want  bool
	}{
		{"README.md - workspace-halo - Visual Studio Code", "workspace-halo", true},
		{"README.md - workspace-halo (Workspace) - Visual Studio Code", "workspace-halo", true},
		{"workspace-halo - Visual Studio Code", "workspace-halo", true},
		{"README.md - other - Visual Studio Code", "workspace-halo", false},
		{"workspace-halo - other - Visual Studio Code", "workspace-halo", false},
		{"README.md - workspace-halo-tools - Visual Studio Code", "workspace-halo", false},
	}
	for _, test := range tests {
		if got := windowTitleMatchesWorkspace(test.title, test.name); got != test.want {
			t.Errorf("windowTitleMatchesWorkspace(%q, %q) = %t, want %t", test.title, test.name, got, test.want)
		}
	}
}

func TestParseColor(t *testing.T) {
	tests := []struct {
		input string
		want  color.NRGBA
	}{
		{"#ff2d55", color.NRGBA{R: 0xff, G: 0x2d, B: 0x55, A: 0xff}},
		{"#0af", color.NRGBA{R: 0x00, G: 0xaa, B: 0xff, A: 0xff}},
	}
	for _, test := range tests {
		got, err := parseColor(test.input)
		if err != nil {
			t.Fatalf("parseColor(%q): %v", test.input, err)
		}
		if got != test.want {
			t.Errorf("parseColor(%q) = %#v, want %#v", test.input, got, test.want)
		}
	}
	if _, err := parseColor("peacock"); err == nil {
		t.Error("parseColor accepted a non-hex value")
	}
}

func TestBorderMotifs(t *testing.T) {
	if !borderPixel(0, 10, 100, 100, 12, "solid") {
		t.Error("solid border omitted an edge pixel")
	}
	if borderPixel(50, 50, 100, 100, 12, "solid") {
		t.Error("solid border filled the center")
	}
	if borderPixel(0, 16, 100, 100, 4, "dashed") == borderPixel(0, 24, 100, 100, 4, "dashed") {
		t.Error("dashed border did not alternate")
	}
	if borderPixel(5, 50, 100, 100, 12, "double") {
		t.Error("double border filled its intended gap")
	}
}

func TestLogoUpscalesToOneThirdHeight(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 3, 6))
	for y := 0; y < 6; y++ {
		for x := 0; x < 3; x++ {
			source.SetNRGBA(x, y, color.NRGBA{R: 1, G: 2, B: 3, A: 255})
		}
	}
	destination := image.NewNRGBA(image.Rect(0, 0, 300, 300))
	drawScaledLogo(destination, source, 12)
	if got := destination.NRGBAAt(230, 180); got.A != 255 {
		t.Fatalf("expected upscaled logo at bottom-right origin, got %#v", got)
	}
	if got := destination.NRGBAAt(229, 180); got.A != 0 {
		t.Fatalf("logo exceeded its calculated width: %#v", got)
	}
	if got := destination.NRGBAAt(230, 179); got.A != 0 {
		t.Fatalf("logo exceeded its calculated height: %#v", got)
	}
}

func TestVisibilityStatePrecedence(t *testing.T) {
	tests := []struct {
		manual, activation, minimized, focused, altTab, taskbarHover, occluded bool
		wantVisible                                                            bool
		wantReason                                                             string
	}{
		{true, true, true, true, true, true, true, true, "double-shift"},
		{false, true, true, true, true, true, true, true, "activation"},
		{false, false, true, true, true, true, true, true, "alt-tab"},
		{false, false, true, true, false, true, true, true, "taskbar-hover"},
		{false, false, true, false, false, false, true, true, "minimized"},
		{false, false, false, true, false, false, true, false, "focused"},
		{false, false, false, false, false, false, true, true, "occluded"},
		{false, false, false, false, false, false, false, false, "hidden"},
	}
	for _, test := range tests {
		visible, reason := visibilityState(
			test.manual,
			test.activation,
			test.minimized,
			test.focused,
			test.altTab,
			test.taskbarHover,
			test.occluded,
		)
		if visible != test.wantVisible || reason != test.wantReason {
			t.Errorf(
				"visibilityState(%t, %t, %t, %t, %t, %t, %t) = (%t, %q), want (%t, %q)",
				test.manual, test.activation, test.minimized, test.focused,
				test.altTab, test.taskbarHover, test.occluded,
				visible, reason, test.wantVisible, test.wantReason,
			)
		}
	}
}

func TestDoubleShiftReleaseTriggersWithinFourHundredMilliseconds(t *testing.T) {
	triggered, next := doubleShiftRelease(1_000, 1_400)
	if !triggered || next != 0 {
		t.Fatalf("second release = (%t, %d), want (true, 0)", triggered, next)
	}

	triggered, next = doubleShiftRelease(1_000, 1_401)
	if triggered || next != 1_401 {
		t.Fatalf("late second release = (%t, %d), want (false, 1401)", triggered, next)
	}
}

func TestActivationHaloDismissal(t *testing.T) {
	tests := []struct {
		name                                         string
		activation, focused, wasFocused, pointerDown bool
		want                                         bool
	}{
		{"focused click", true, true, true, true, true},
		{"focus gained", true, true, false, false, true},
		{"initial focused frame", true, true, true, false, false},
		{"unfocused click", true, false, false, true, false},
		{"already dismissed", false, true, false, true, false},
	}
	for _, test := range tests {
		if got := shouldDismissActivation(test.activation, test.focused, test.wasFocused, test.pointerDown); got != test.want {
			t.Errorf("%s: shouldDismissActivation() = %t, want %t", test.name, got, test.want)
		}
	}
}

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

func TestHaloOverlayNeverCountsAsOcclusion(t *testing.T) {
	if !isIgnoredOccludingClass("WorkspaceHaloOverlay") {
		t.Fatal("another workspace halo can trigger an occlusion loop")
	}
	if isIgnoredOccludingClass("Chrome_WidgetWin_1") {
		t.Fatal("a VS Code window was incorrectly ignored as an occluder")
	}
}

func TestDuplicateDisplayTopologySuppressesAmbientOcclusion(t *testing.T) {
	if shouldCheckOcclusion(false, false, true, false, displayConfigTopologyClone) {
		t.Fatal("Duplicate display mode can trigger the overlapping-window z-order loop")
	}
	if !shouldCheckOcclusion(false, false, true, false, 0x00000004) {
		t.Fatal("Extend display mode incorrectly suppresses ambient occlusion")
	}
	if shouldCheckOcclusion(false, false, true, true, 0x00000004) {
		t.Fatal("ambient occlusion ran while display topology was changing")
	}
}

func TestDisplayConfigInteropStructureSizes(t *testing.T) {
	if got := unsafe.Sizeof(displayConfigPathInfo{}); got != 68 {
		t.Fatalf("DISPLAYCONFIG_PATH_INFO size = %d, want 68", got)
	}
	if got := unsafe.Sizeof(displayConfigModeInfo{}); got != 64 {
		t.Fatalf("DISPLAYCONFIG_MODE_INFO size = %d, want 64", got)
	}
	if got := unsafe.Alignof(displayConfigModeInfo{}); got != 8 {
		t.Fatalf("DISPLAYCONFIG_MODE_INFO alignment = %d, want 8", got)
	}
}

func TestTaskbarClassRecognition(t *testing.T) {
	for _, class := range []string{"Shell_TrayWnd", "Shell_SecondaryTrayWnd"} {
		if !isTaskbarClass(class) {
			t.Errorf("taskbar class %q is not recognized", class)
		}
	}
	if isTaskbarClass("Chrome_WidgetWin_1") {
		t.Error("an application window class was treated as a taskbar")
	}
}

func TestTaskbarThumbnailClassRecognition(t *testing.T) {
	for _, class := range []string{
		"TaskListThumbnailWnd",
		"XamlExplorerHostIslandWindow",
		"XamlExplorerHostIslandWindow_WASDK",
		"Microsoft.UI.Content.PopupWindowSiteBridge",
	} {
		if !isTaskbarThumbnailClass(class) {
			t.Errorf("taskbar thumbnail class %q is not recognized", class)
		}
	}
	for _, class := range []string{
		"MultitaskingViewFrame",
		"Chrome_WidgetWin_1",
		"Shell_TrayWnd",
		"TopLevelWindowForOverflowXamlIsland",
	} {
		if isTaskbarThumbnailClass(class) {
			t.Errorf("class %q was wrongly treated as a taskbar thumbnail", class)
		}
	}
}

func TestTaskbarThumbnailHoverRetainsHalosOutsideDuplicateMode(t *testing.T) {
	const extendTopology = uint32(0x00000004)
	if !taskbarHoverState(true, false, true, false, extendTopology) {
		t.Fatal("thumbnail hover did not retain halos in Extend mode")
	}
	if taskbarHoverState(true, false, true, false, displayConfigTopologyClone) {
		t.Fatal("thumbnail hover bypassed Duplicate-mode protection")
	}
	if taskbarHoverState(true, false, true, true, extendTopology) {
		t.Fatal("thumbnail hover ran while display topology was changing")
	}
	if !taskbarHoverState(false, true, false, true, displayConfigTopologyClone) {
		t.Fatal("direct taskbar hover was suppressed during a topology transition")
	}
	if taskbarHoverState(false, false, true, false, extendTopology) {
		t.Fatal("an independently visible shell flyout started taskbar hover")
	}
	if taskbarHoverState(true, false, false, false, extendTopology) {
		t.Fatal("taskbar hover remained active after the thumbnail flyout closed")
	}
}

func TestTaskbarThumbnailChildHitRetainsOnlyAnExistingTaskbarHover(t *testing.T) {
	taskbar := rect{Left: 0, Top: 1040, Right: 1920, Bottom: 1080}
	flyout := rect{Left: 620, Top: 720, Right: 1300, Bottom: 1040}
	cursor := point{X: 900, Y: 900}
	if !taskbarThumbnailChildHit(true, true, cursor, taskbar, flyout) {
		t.Fatal("a visible taskbar child under the cursor did not retain hover")
	}
	if taskbarThumbnailChildHit(false, true, cursor, taskbar, flyout) {
		t.Fatal("a taskbar child started hover without a direct taskbar hit")
	}
	if taskbarThumbnailChildHit(true, false, cursor, taskbar, flyout) {
		t.Fatal("a hidden taskbar child retained hover")
	}
	if taskbarThumbnailChildHit(true, true, point{X: 900, Y: 1060}, taskbar, flyout) {
		t.Fatal("a cursor still inside the taskbar was treated as flyout hover")
	}
	if taskbarThumbnailChildHit(true, true, point{X: 200, Y: 900}, taskbar, flyout) {
		t.Fatal("a taskbar child away from the cursor retained hover")
	}
	if taskbarThumbnailChildHit(true, true, cursor, taskbar, rect{Left: 900, Top: 900, Right: 902, Bottom: 902}) {
		t.Fatal("an unlaid-out child retained hover")
	}
}

func TestTaskbarThumbnailBandRetainsCompositionOnlyPreviewHover(t *testing.T) {
	taskbar := rect{Left: 0, Top: 912, Right: 1536, Bottom: 960}
	if !taskbarThumbnailBandHit(true, point{X: 700, Y: 700}, taskbar) {
		t.Fatal("the Windows 11 preview band did not retain taskbar hover")
	}
	if taskbarThumbnailBandHit(false, point{X: 700, Y: 700}, taskbar) {
		t.Fatal("the preview band started hover without a direct taskbar hit")
	}
	if taskbarThumbnailBandHit(true, point{X: 700, Y: 930}, taskbar) {
		t.Fatal("a direct taskbar point was treated as preview-band retention")
	}
	if taskbarThumbnailBandHit(true, point{X: 700, Y: 527}, taskbar) {
		t.Fatal("a point above the preview band retained taskbar hover")
	}
	if taskbarThumbnailBandHit(true, point{X: 1536, Y: 700}, taskbar) {
		t.Fatal("a point beyond the taskbar width retained taskbar hover")
	}
	if taskbarThumbnailBandHit(true, point{X: 700, Y: 700}, rect{}) {
		t.Fatal("an unknown taskbar rectangle retained hover")
	}
}

func TestPointInRect(t *testing.T) {
	r := rect{Left: 0, Top: 912, Right: 1536, Bottom: 960}
	if !pointInRect(point{X: 700, Y: 940}, r) {
		t.Error("a point inside the taskbar was not detected")
	}
	if pointInRect(point{X: 700, Y: 911}, r) {
		t.Error("a point above the taskbar was treated as inside")
	}
	if pointInRect(point{X: 1536, Y: 940}, r) {
		t.Error("the exclusive right edge was treated as inside")
	}
	if !pointInRect(point{X: 0, Y: 912}, r) {
		t.Error("the inclusive top-left corner was not detected")
	}
}

func TestRectanglesIntersect(t *testing.T) {
	base := rect{Left: 0, Top: 0, Right: 100, Bottom: 100}
	if !rectanglesIntersect(base, rect{Left: 99, Top: 99, Right: 120, Bottom: 120}) {
		t.Error("overlap was not detected")
	}
	if rectanglesIntersect(base, rect{Left: 100, Top: 0, Right: 120, Bottom: 20}) {
		t.Error("touching edges were treated as overlap")
	}
}

func TestVisibleFrameBoundsPreventAdjacentMonitorFalseOverlap(t *testing.T) {
	// Captured from two maximized VS Code windows on vertically adjacent
	// monitors. GetWindowRect includes resize borders that cross the boundary.
	rawUpper := rect{Left: 955, Top: -1214, Right: 2893, Bottom: 4}
	rawLower := rect{Left: -7, Top: -7, Right: 1543, Bottom: 919}
	if !rectanglesIntersect(rawUpper, rawLower) {
		t.Fatal("fixture no longer demonstrates the invisible-border overlap")
	}

	visibleUpper := rect{Left: 964, Top: -1205, Right: 2884, Bottom: -5}
	visibleLower := rect{Left: 0, Top: 0, Right: 1920, Bottom: 1140}
	if rectanglesIntersect(visibleUpper, visibleLower) {
		t.Error("DWM visible frame bounds on adjacent monitors were treated as overlapping")
	}
}
func TestPillBackgroundColorTintsTheContrastPole(t *testing.T) {
	tests := []struct {
		name      string
		text      color.NRGBA
		lightPill bool
	}{
		// #ff2d55 luminance is 0.238, above the 0.179 break-even: the
		// pill sits on the dark side; #2e7d5b (0.16) gets a light pill.
		{"default red", color.NRGBA{R: 0xff, G: 0x2d, B: 0x55, A: 255}, false},
		{"peacock green", color.NRGBA{R: 0x0b, G: 0xa9, B: 0x33, A: 255}, false},
		{"dark green", color.NRGBA{R: 0x2e, G: 0x7d, B: 0x5b, A: 255}, true},
		{"navy", color.NRGBA{R: 0x00, G: 0x00, B: 0x8b, A: 255}, true},
	}
	for _, test := range tests {
		pill := pillBackgroundColor(test.text)
		if ratio := contrastRatio(test.text, pill); ratio < 3 {
			t.Errorf("%s: contrast %.2f is below the 3:1 large-text minimum", test.name, ratio)
		}
		if light := relativeLuminance(pill) > relativeLuminance(test.text); light != test.lightPill {
			t.Errorf("%s: the pill sits on the wrong side of the text luminance", test.name)
		}
		if pill.R == pill.G && pill.G == pill.B {
			t.Errorf("%s: pill %#v carries no trace of the text hue", test.name, pill)
		}
	}
	// Achromatic texts have no hue to carry; contrast still holds.
	for _, text := range []color.NRGBA{
		{R: 0, G: 0, B: 0, A: 255},
		{R: 255, G: 255, B: 255, A: 255},
	} {
		if ratio := contrastRatio(text, pillBackgroundColor(text)); ratio < 3 {
			t.Errorf("achromatic text %#v: contrast %.2f is below 3:1", text, ratio)
		}
	}
}

func TestPillPixelRoundsTheEnds(t *testing.T) {
	left, top, right, bottom := 0, 0, 100, 40
	if !pillPixel(50, 20, left, top, right, bottom) {
		t.Error("the pill center was not inside")
	}
	if !pillPixel(0, 20, left, top, right, bottom) {
		t.Error("the left cap apex was not inside")
	}
	if pillPixel(1, 1, left, top, right, bottom) {
		t.Error("the top-left corner was inside: the cap is not rounded")
	}
	if pillPixel(98, 38, left, top, right, bottom) {
		t.Error("the bottom-right corner was inside: the cap is not rounded")
	}
	if pillPixel(50, 40, left, top, right, bottom) {
		t.Error("a pixel below the pill was inside")
	}
}

func TestSegmentedBorderAlternatesBlack(t *testing.T) {
	c := color.NRGBA{R: 0xff, G: 0x2d, B: 0x55, A: 255}
	black := color.NRGBA{A: 255}
	canvas := image.NewNRGBA(image.Rect(0, 0, 200, 150))
	drawBorder(canvas, c, 4, "solid", 50)
	if got := canvas.NRGBAAt(10, 0); got != c {
		t.Errorf("first top segment = %#v, want the halo color", got)
	}
	if got := canvas.NRGBAAt(60, 0); got != black {
		t.Errorf("second top segment = %#v, want opaque black", got)
	}
	if got := canvas.NRGBAAt(1, 60); got != black {
		t.Errorf("second left segment = %#v, want opaque black", got)
	}
	if got := canvas.NRGBAAt(60, 0); got.A != 255 {
		t.Error("a black segment is transparent: it must be painted")
	}
	plain := image.NewNRGBA(image.Rect(0, 0, 200, 150))
	drawBorder(plain, c, 4, "solid", 0)
	if got := plain.NRGBAAt(60, 0); got != c {
		t.Errorf("segment 0 should keep a continuous border, got %#v", got)
	}
}

func TestPillBoundsHugTheInk(t *testing.T) {
	left, top, right, bottom := pillBounds(100, 100, 400, 140)
	if top != 120 {
		t.Errorf("pill top = %d, want 120: the internal leading is not trimmed", top)
	}
	if bottom != 244 {
		t.Errorf("pill bottom = %d, want 244", bottom)
	}
	if left != 54 || right != 546 {
		t.Errorf("pill sides = %d..%d, want 54..546", left, right)
	}
	if height := bottom - top; height >= 140 {
		t.Errorf("pill height %d is not tighter than the text cell 140", height)
	}
}

func TestDitherCoverageMatchesOpacity(t *testing.T) {
	coverage := func(opacity int) float64 {
		kept := 0
		for y := 0; y < 64; y++ {
			for x := 0; x < 64; x++ {
				if ditherKeep(x, y, opacity) {
					kept++
				}
			}
		}
		return float64(kept) / (64 * 64)
	}
	if got := coverage(0); got != 0 {
		t.Errorf("opacity 0 kept %.2f of the pixels, want 0", got)
	}
	if got := coverage(100); got != 1 {
		t.Errorf("opacity 100 kept %.2f of the pixels, want 1", got)
	}
	if got := coverage(80); math.Abs(got-0.8) > 0.05 {
		t.Errorf("opacity 80 kept %.2f of the pixels, want about 0.80", got)
	}
	if coverage(30) >= coverage(70) {
		t.Error("coverage does not grow with opacity")
	}
}

func TestApplyTransparentColorKeyPreservesOpaqueBlack(t *testing.T) {
	pixels := []byte{
		0, 0, 0, 0,
		0, 0, 0, 255,
	}
	applyTransparentColorKey(pixels)
	if got := pixels[:4]; got[0] != 1 || got[1] != 2 || got[2] != 3 || got[3] != 0 {
		t.Fatalf("transparent pixel = %v, want BGR color key with zero alpha", got)
	}
	if got := pixels[4:]; got[0] != 0 || got[1] != 0 || got[2] != 0 || got[3] != 255 {
		t.Fatalf("opaque black pixel changed: %v", got)
	}
}
