package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"sync"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// HistogramWidget renders a live 0–9 digit frequency bar chart.
type HistogramWidget struct {
	widget.BaseWidget

	counts [10]atomic.Int64
	total  atomic.Int64
	mu     sync.Mutex
	raster *canvas.Raster
}

// NewHistogramWidget creates a new histogram widget.
func NewHistogramWidget() *HistogramWidget {
	h := &HistogramWidget{}
	h.ExtendBaseWidget(h)
	return h
}

// MinSize satisfies fyne.Widget.
func (h *HistogramWidget) MinSize() fyne.Size {
	return fyne.NewSize(200, 120)
}

// CreateRenderer satisfies fyne.Widget.
func (h *HistogramWidget) CreateRenderer() fyne.WidgetRenderer {
	h.raster = canvas.NewRaster(h.draw)
	h.raster.SetMinSize(h.MinSize())
	return widget.NewSimpleRenderer(h.raster)
}

// AddDigits increments frequency counters for each digit character in s.
func (h *HistogramWidget) AddDigits(s string) {
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			d := ch - '0'
			h.counts[d].Add(1)
			h.total.Add(1)
		}
	}
	if h.raster != nil {
		canvas.Refresh(h.raster)
	}
}

// Reset zeros all counters.
func (h *HistogramWidget) Reset() {
	for i := range h.counts {
		h.counts[i].Store(0)
	}
	h.total.Store(0)
	if h.raster != nil {
		canvas.Refresh(h.raster)
	}
}

func (h *HistogramWidget) draw(w, wh int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, wh))
	bg := color.RGBA{R: 0x16, G: 0x1b, B: 0x22, A: 0xFF}
	draw.Draw(img, img.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)

	total := h.total.Load()
	var counts [10]int64
	for i := range counts {
		counts[i] = h.counts[i].Load()
	}

	const labelH = 20 // pixels reserved at bottom for digit labels
	const topPad = 16 // pixels reserved at top for percentage labels
	barAreaH := wh - labelH - topPad
	if barAreaH < 1 {
		return img
	}

	barW := w / 10
	face := basicfont.Face7x13

	for d := 0; d < 10; d++ {
		x0 := d * barW
		x1 := x0 + barW - 2

		var pct float64
		if total > 0 {
			pct = float64(counts[d]) / float64(total)
		}

		barH := int(pct * float64(barAreaH))
		barY0 := topPad + (barAreaH - barH)
		barY1 := topPad + barAreaH

		col := DigitColors[d]

		// Draw bar.
		for y := barY0; y < barY1; y++ {
			for x := x0; x < x1; x++ {
				if x >= 0 && x < w && y >= 0 && y < wh {
					img.SetRGBA(x, y, col)
				}
			}
		}

		// Percentage label above bar.
		label := fmt.Sprintf("%.1f%%", pct*100)
		lw := bitmapTextWidth(face, label)
		lx := x0 + (barW-lw)/2
		ly := barY0 - 2
		if ly < 12 {
			ly = 12
		}
		drawBitmapText(img, face, label, lx, ly, color.RGBA{R: 0xe6, G: 0xed, B: 0xf3, A: 0xFF})

		// Digit label below bar.
		dlabel := fmt.Sprintf("%d", d)
		dlw := bitmapTextWidth(face, dlabel)
		drawBitmapText(img, face, dlabel, x0+(barW-dlw)/2, wh-5, col)
	}

	// 10% reference line.
	refY := topPad + int(0.90*float64(barAreaH))
	refCol := color.RGBA{R: 0xFF, G: 0xD7, B: 0x00, A: 0x99}
	for x := 0; x < w; x++ {
		if refY >= 0 && refY < wh {
			img.SetRGBA(x, refY, refCol)
		}
	}

	return img
}

func bitmapTextWidth(face font.Face, s string) int {
	bounds, _ := font.BoundString(face, s)
	return (bounds.Max.X - bounds.Min.X).Ceil()
}

func drawBitmapText(img *image.RGBA, face font.Face, s string, x, y int, col color.Color) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(s)
}
