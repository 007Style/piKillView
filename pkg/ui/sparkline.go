package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/image/font/basicfont"
)

const sparklinePoints = 60

// SparklineWidget renders a rolling 60-point digits/sec area chart.
type SparklineWidget struct {
	widget.BaseWidget

	mu     sync.Mutex
	ring   [sparklinePoints]float64
	head   int // next write position
	count  int // number of valid samples
	raster *canvas.Raster
}

// NewSparklineWidget creates a new sparkline widget.
func NewSparklineWidget() *SparklineWidget {
	s := &SparklineWidget{}
	s.ExtendBaseWidget(s)
	return s
}

// MinSize satisfies fyne.Widget.
func (s *SparklineWidget) MinSize() fyne.Size {
	return fyne.NewSize(200, 80)
}

// CreateRenderer satisfies fyne.Widget.
func (s *SparklineWidget) CreateRenderer() fyne.WidgetRenderer {
	s.raster = canvas.NewRaster(s.draw)
	s.raster.SetMinSize(s.MinSize())
	return widget.NewSimpleRenderer(s.raster)
}

// Sample adds a digits/sec measurement to the ring buffer.
func (s *SparklineWidget) Sample(dps float64) {
	s.mu.Lock()
	s.ring[s.head] = dps
	s.head = (s.head + 1) % sparklinePoints
	if s.count < sparklinePoints {
		s.count++
	}
	s.mu.Unlock()
	if s.raster != nil {
		canvas.Refresh(s.raster)
	}
}

// Reset clears all samples.
func (s *SparklineWidget) Reset() {
	s.mu.Lock()
	s.ring = [sparklinePoints]float64{}
	s.head = 0
	s.count = 0
	s.mu.Unlock()
	if s.raster != nil {
		canvas.Refresh(s.raster)
	}
}

func (s *SparklineWidget) draw(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	bg := color.RGBA{R: 0x16, G: 0x1b, B: 0x22, A: 0xFF}
	draw.Draw(img, img.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)

	s.mu.Lock()
	count := s.count
	head := s.head
	var ring [sparklinePoints]float64
	copy(ring[:], s.ring[:])
	s.mu.Unlock()

	if count == 0 {
		return img
	}

	// Build ordered slice of samples (oldest → newest).
	pts := make([]float64, count)
	start := (head - count + sparklinePoints) % sparklinePoints
	for i := 0; i < count; i++ {
		pts[i] = ring[(start+i)%sparklinePoints]
	}

	// Find max for scaling.
	maxVal := 1.0
	for _, v := range pts {
		if v > maxVal {
			maxVal = v
		}
	}

	lineCol := color.RGBA{R: 0x00, G: 0xFF, B: 0x41, A: 0xFF}
	fillCol := color.RGBA{R: 0x00, G: 0xFF, B: 0x41, A: 0x40}
	const botPad = 14
	chartH := h - botPad

	// Draw filled area and line.
	for i := 0; i < len(pts); i++ {
		x := i * (w - 1) / (sparklinePoints - 1)
		if len(pts) > 1 {
			x = i * (w - 1) / (len(pts) - 1)
		}
		y := chartH - int(pts[i]/maxVal*float64(chartH-2))
		if y < 0 {
			y = 0
		}
		// Fill column from y to bottom.
		for fy := y; fy < chartH; fy++ {
			if x >= 0 && x < w && fy >= 0 && fy < h {
				img.SetRGBA(x, fy, fillCol)
			}
		}
		// Line pixel.
		if x >= 0 && x < w && y >= 0 && y < h {
			img.SetRGBA(x, y, lineCol)
		}
	}

	// Latest value label.
	latest := pts[len(pts)-1]
	label := fmt.Sprintf("%.1f d/s", latest)
	drawBitmapText(img, basicfont.Face7x13, label, 2, h-2, color.RGBA{R: 0x8b, G: 0x94, B: 0x9e, A: 0xFF})

	return img
}
