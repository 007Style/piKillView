package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"runtime"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	gopsutilcpu "github.com/shirou/gopsutil/v3/cpu"
	"golang.org/x/image/font/basicfont"
)

const maxDials = 12

// CPUDialsWidget draws per-core arc gauges.
type CPUDialsWidget struct {
	widget.BaseWidget

	mu      sync.Mutex
	percents []float64
	numCPU  int
	raster  *canvas.Raster
	stopCh  chan struct{}
}

// NewCPUDialsWidget creates and starts the CPU dials widget.
func NewCPUDialsWidget() *CPUDialsWidget {
	n := runtime.NumCPU()
	c := &CPUDialsWidget{
		numCPU:   n,
		percents: make([]float64, n),
		stopCh:   make(chan struct{}),
	}
	c.ExtendBaseWidget(c)
	return c
}

// MinSize satisfies fyne.Widget.
func (c *CPUDialsWidget) MinSize() fyne.Size {
	return fyne.NewSize(200, 160)
}

// CreateRenderer satisfies fyne.Widget.
func (c *CPUDialsWidget) CreateRenderer() fyne.WidgetRenderer {
	c.raster = canvas.NewRaster(c.draw)
	c.raster.SetMinSize(c.MinSize())
	go c.poll()
	return widget.NewSimpleRenderer(c.raster)
}

// Stop halts the polling goroutine.
func (c *CPUDialsWidget) Stop() {
	select {
	case <-c.stopCh:
	default:
		close(c.stopCh)
	}
}

func (c *CPUDialsWidget) poll() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			pcts, err := gopsutilcpu.Percent(0, true)
			if err != nil || len(pcts) == 0 {
				continue
			}
			c.mu.Lock()
			c.percents = pcts
			c.numCPU = len(pcts)
			c.mu.Unlock()
			r := c.raster
			fyne.Do(func() { canvas.Refresh(r) })
		}
	}
}

// lerpColor linearly interpolates between two colors.
func lerpColor(a, b color.RGBA, t float64) color.RGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return color.RGBA{
		R: uint8(float64(a.R) + t*float64(b.R-a.R)),
		G: uint8(float64(a.G) + t*float64(b.G-a.G)),
		B: uint8(float64(a.B) + t*float64(b.B-a.B)),
		A: 0xFF,
	}
}

func dialColor(pct float64) color.RGBA {
	green := color.RGBA{R: 0x00, G: 0xC8, B: 0x53, A: 0xFF}
	yellow := color.RGBA{R: 0xFF, G: 0xD6, B: 0x00, A: 0xFF}
	red := color.RGBA{R: 0xD5, G: 0x00, B: 0x00, A: 0xFF}
	t := pct / 100.0
	if t <= 0.6 {
		return lerpColor(green, yellow, t/0.6)
	}
	return lerpColor(yellow, red, (t-0.6)/0.4)
}

func (c *CPUDialsWidget) draw(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	bg := color.RGBA{R: 0x16, G: 0x1b, B: 0x22, A: 0xFF}
	draw.Draw(img, img.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)

	c.mu.Lock()
	percents := make([]float64, len(c.percents))
	copy(percents, c.percents)
	c.mu.Unlock()

	n := len(percents)
	showAggregate := n > maxDials
	numDials := n
	if showAggregate {
		numDials = maxDials + 1
	}

	// Compute grid layout.
	cols := 4
	rows := (numDials + cols - 1) / cols
	if rows < 1 {
		rows = 1
	}
	cellW := w / cols
	cellH := h / rows
	if cellH < 1 {
		cellH = 1
	}

	drawDial := func(px, py int, pct float64, label string) {
		r := cellW/2 - 4
		if cellH/2-4 < r {
			r = cellH/2 - 4
		}
		if r < 4 {
			return
		}
		cx := px + cellW/2
		cy := py + cellH/2 + r/4

		col := dialColor(pct)
		track := color.RGBA{R: 0x30, G: 0x36, B: 0x3d, A: 0xFF}

		sweepAngle := math.Pi * pct / 100.0 // 0 to π

		// Draw track arc (full semicircle).
		for i := 0; i < 360; i++ {
			angle := math.Pi + float64(i)*math.Pi/180.0 // left to right over top
			px2 := cx + int(float64(r)*math.Cos(angle))
			py2 := cy + int(float64(r)*math.Sin(angle))
			setPixelSafe(img, px2, py2, w, h, track)
			// inner ring
			px2 = cx + int(float64(r-2)*math.Cos(angle))
			py2 = cy + int(float64(r-2)*math.Sin(angle))
			setPixelSafe(img, px2, py2, w, h, track)
		}

		// Draw filled arc.
		for i := 0; i < 180; i++ {
			if float64(i)*math.Pi/180.0 > sweepAngle {
				break
			}
			angle := math.Pi + float64(i)*math.Pi/180.0
			for rad := r - 3; rad <= r; rad++ {
				px2 := cx + int(float64(rad)*math.Cos(angle))
				py2 := cy + int(float64(rad)*math.Sin(angle))
				setPixelSafe(img, px2, py2, w, h, col)
			}
		}

		// Draw label.
		lbl := fmt.Sprintf("%s %.0f%%", label, pct)
		tw := bitmapTextWidth(basicfont.Face7x13, lbl)
		drawBitmapText(img, basicfont.Face7x13, lbl, cx-tw/2, py+cellH-3,
			color.RGBA{R: 0x8b, G: 0x94, B: 0x9e, A: 0xFF})
	}

	shown := n
	if showAggregate {
		shown = maxDials
	}
	for i := 0; i < shown; i++ {
		col := i % cols
		row := i / cols
		px := col * cellW
		py := row * cellH
		drawDial(px, py, percents[i], fmt.Sprintf("C%d", i))
	}

	if showAggregate {
		// Aggregate: average of all.
		var sum float64
		for _, v := range percents {
			sum += v
		}
		avg := sum / float64(len(percents))
		idx := maxDials
		col := idx % cols
		row := idx / cols
		drawDial(col*cellW, row*cellH, avg, "AVG")
	}

	return img
}

func setPixelSafe(img *image.RGBA, x, y, w, h int, col color.RGBA) {
	if x >= 0 && y >= 0 && x < w && y < h {
		img.SetRGBA(x, y, col)
	}
}
