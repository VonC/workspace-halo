//go:build windows

package main

import (
	"image"
	"image/color"
	"testing"
)

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

func TestRectanglesIntersect(t *testing.T) {
	base := rect{Left: 0, Top: 0, Right: 100, Bottom: 100}
	if !rectanglesIntersect(base, rect{Left: 99, Top: 99, Right: 120, Bottom: 120}) {
		t.Error("overlap was not detected")
	}
	if rectanglesIntersect(base, rect{Left: 100, Top: 0, Right: 120, Bottom: 20}) {
		t.Error("touching edges were treated as overlap")
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
