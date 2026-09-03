//go:build ignore

// Run with: go run assets/icon.go
// Generates assets/icon.png — a white π symbol on a #1a1a2e background.

package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

func main() {
	const size = 256

	bg := color.RGBA{R: 0x1a, G: 0x1a, B: 0x2e, A: 0xff}
	fg := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}

	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Fill background
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.SetRGBA(x, y, bg)
		}
	}

	// Draw a hand-crafted π symbol using thick lines.
	// The π character consists of:
	//   1. A horizontal top bar
	//   2. Two vertical legs hanging down from the bar
	// Coordinates are in the 256×256 space.

	thick := 14.0 // line thickness (radius for rounded caps)

	// Top bar: horizontal line
	barY := 90.0
	barX1 := 56.0
	barX2 := 200.0

	// Left leg: from (barX1+20, barY) down to (barX1+20, 186)
	legLX := barX1 + 26
	legLY1 := barY
	legLY2 := 186.0

	// Right leg: from (barX2-20, barY) down to (barX2-20, 186)
	legRX := barX2 - 26
	legRY1 := barY
	legRY2 := 186.0

	// Helper: distance from point (px,py) to segment (ax,ay)-(bx,by)
	distToSeg := func(px, py, ax, ay, bx, by float64) float64 {
		dx, dy := bx-ax, by-ay
		if dx == 0 && dy == 0 {
			return math.Hypot(px-ax, py-ay)
		}
		t := ((px-ax)*dx + (py-ay)*dy) / (dx*dx + dy*dy)
		if t < 0 {
			t = 0
		}
		if t > 1 {
			t = 1
		}
		return math.Hypot(px-ax-t*dx, py-ay-t*dy)
	}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5

			// Anti-aliased coverage: minimum distance to any segment
			d := math.Min(
				distToSeg(px, py, barX1, barY, barX2, barY),   // top bar
				math.Min(
					distToSeg(px, py, legLX, legLY1, legLX, legLY2), // left leg
					distToSeg(px, py, legRX, legRY1, legRX, legRY2), // right leg
				),
			)

			if d <= thick-1 {
				img.SetRGBA(x, y, fg)
			} else if d <= thick+1 {
				// Anti-alias edge
				alpha := uint8((thick + 1 - d) / 2 * 255)
				img.SetRGBA(x, y, color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: alpha})
			}
		}
	}

	f, err := os.Create("assets/icon.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}
