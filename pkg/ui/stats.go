package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/007Style/piKillView/pkg/engine"
	"golang.org/x/image/font/basicfont"
)

// WorkerRow holds display state for one worker.
type WorkerRow struct {
	ID     int
	Terms  int64
	Active bool
}

// StatsPanel is the right-side stats display.
type StatsPanel struct {
	mu sync.Mutex

	// Digit counter
	countLabel *widget.Label
	count      atomic.Int64

	// Timer
	timerLabel  *widget.Label
	timerStart  time.Time
	timerActive bool
	timerStop   chan struct{}
	elapsed     time.Duration

	// Round + ETA
	roundLabel *widget.Label
	etaLabel   *widget.Label
	roundN     int
	roundDigs  int64

	// Speed gauge (canvas.Raster arc dial)
	speedRaster *canvas.Raster
	currentDPS  float64
	maxDPS      float64

	// Memory
	memLabel *widget.Label

	// Workers
	workers     []WorkerRow
	workerList  *widget.List

	// Sparkline
	Sparkline *SparklineWidget

	// CPU dials
	CPUDials *CPUDialsWidget

	// Histogram
	Histogram *HistogramWidget
}

// NewStatsPanel constructs the stats panel.
func NewStatsPanel() *StatsPanel {
	sp := &StatsPanel{
		timerStop: make(chan struct{}, 1),
		Sparkline: NewSparklineWidget(),
		CPUDials:  NewCPUDialsWidget(),
		Histogram: NewHistogramWidget(),
	}

	sp.countLabel = widget.NewLabel("0")
	sp.countLabel.TextStyle = fyne.TextStyle{Bold: true}

	sp.timerLabel = widget.NewLabel("00:00:00.000")
	sp.timerLabel.TextStyle = fyne.TextStyle{Monospace: true}

	sp.roundLabel = widget.NewLabel("Round 0")
	sp.etaLabel = widget.NewLabel("")

	sp.speedRaster = canvas.NewRaster(sp.drawSpeedDial)
	sp.speedRaster.SetMinSize(fyne.NewSize(160, 80))

	sp.memLabel = widget.NewLabel("Heap: 0 MB")

	sp.workerList = widget.NewList(
		func() int {
			sp.mu.Lock()
			defer sp.mu.Unlock()
			return len(sp.workers)
		},
		func() fyne.CanvasObject {
			dot := canvas.NewCircle(color.RGBA{G: 0xFF, A: 0xFF})
			dotMin := fyne.NewSize(10, 10)
			_ = dotMin
			lbl := widget.NewLabel("")
			lbl.TextStyle = fyne.TextStyle{Monospace: true}
			dotContainer := container.New(&fixedSizeLayout{size: fyne.NewSize(10, 10)}, dot)
			return container.NewHBox(dotContainer, lbl)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			sp.mu.Lock()
			defer sp.mu.Unlock()
			if id >= len(sp.workers) {
				return
			}
			row := sp.workers[id]
			hbox := obj.(*fyne.Container)
			dotCont := hbox.Objects[0].(*fyne.Container)
			dot := dotCont.Objects[0].(*canvas.Circle)
			lbl := hbox.Objects[1].(*widget.Label)
			if row.Active {
				dot.FillColor = color.RGBA{R: 0x00, G: 0xFF, B: 0x64, A: 0xFF}
			} else {
				dot.FillColor = color.RGBA{R: 0x3c, G: 0x3c, B: 0x3c, A: 0xFF}
			}
			lbl.SetText(fmt.Sprintf("W%02d  %s terms", row.ID, formatStatInt(row.Terms)))
			canvas.Refresh(dot)
		},
	)

	go sp.memTicker()
	return sp
}

// UpdateCount sets the live digit count label.
func (sp *StatsPanel) UpdateCount(n int64) {
	sp.count.Store(n)
	sp.countLabel.SetText(formatStatInt(n))
}

// UpdateWorker updates the worker list for the given worker stat.
// Called from the 30fps flush ticker — safe to call Refresh() here since
// it's already coalesced (at most once per frame per batch of workers).
func (sp *StatsPanel) UpdateWorker(stat engine.WorkerStat) {
	sp.mu.Lock()
	// Grow slice if needed.
	for len(sp.workers) <= stat.WorkerID {
		sp.workers = append(sp.workers, WorkerRow{ID: len(sp.workers)})
	}
	sp.workers[stat.WorkerID].Terms = stat.TermsComputed
	sp.workers[stat.WorkerID].Active = stat.Active
	sp.mu.Unlock()
	// Refresh is cheap now — the flush ticker already batches worker updates
	// so this is called at most once per frame regardless of worker count.
	sp.workerList.Refresh()
}

// StartTimer begins the elapsed timer.
func (sp *StatsPanel) StartTimer() {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	if sp.timerActive {
		return
	}
	sp.timerStart = time.Now()
	sp.timerActive = true
	go sp.timerLoop()
}

// StopTimer pauses the elapsed timer.
func (sp *StatsPanel) StopTimer() {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	if !sp.timerActive {
		return
	}
	sp.timerActive = false
	sp.elapsed += time.Since(sp.timerStart)
	select {
	case sp.timerStop <- struct{}{}:
	default:
	}
}

// ResetTimer resets elapsed time to zero.
func (sp *StatsPanel) ResetTimer() {
	sp.mu.Lock()
	sp.elapsed = 0
	sp.timerActive = false
	sp.mu.Unlock()
	sp.timerLabel.SetText("00:00:00.000")
}

func (sp *StatsPanel) timerLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-sp.timerStop:
			return
		case <-ticker.C:
			sp.mu.Lock()
			if !sp.timerActive {
				sp.mu.Unlock()
				return
			}
			total := sp.elapsed + time.Since(sp.timerStart)
			sp.mu.Unlock()
			h := int(total.Hours())
			m := int(total.Minutes()) % 60
			s := int(total.Seconds()) % 60
			ms := int(total.Milliseconds()) % 1000
			text := fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, ms)
			fyne.Do(func() { sp.timerLabel.SetText(text) })
		}
	}
}

// SetRound updates the round indicator.
func (sp *StatsPanel) SetRound(n int, digits int64) {
	sp.mu.Lock()
	sp.roundN = n
	sp.roundDigs = digits
	sp.mu.Unlock()
	sp.roundLabel.SetText(fmt.Sprintf("Round %d — %s digits", n, formatStatInt(digits)))
}

// SetETA updates the ETA label.
func (sp *StatsPanel) SetETA(d time.Duration) {
	sp.etaLabel.SetText(fmt.Sprintf("Next round in ~%ds", int(d.Seconds())))
}

// UpdateDPS updates the speed gauge with a new digits/sec reading.
func (sp *StatsPanel) UpdateDPS(dps float64) {
	sp.mu.Lock()
	sp.currentDPS = dps
	if dps > sp.maxDPS {
		sp.maxDPS = dps
	}
	sp.mu.Unlock()
	canvas.Refresh(sp.speedRaster)
}

func (sp *StatsPanel) drawSpeedDial(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	bg := color.RGBA{R: 0x16, G: 0x1b, B: 0x22, A: 0xFF}
	draw.Draw(img, img.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)

	sp.mu.Lock()
	cur := sp.currentDPS
	maxV := sp.maxDPS
	sp.mu.Unlock()

	if maxV < 1 {
		maxV = 1
	}

	cx, cy := w/2, h/2+10
	r := w/2 - 8
	if h/2-8 < r {
		r = h/2 - 8
	}
	if r < 8 {
		return img
	}

	trackCol := color.RGBA{R: 0x30, G: 0x36, B: 0x3d, A: 0xFF}
	needleCol := color.RGBA{R: 0x00, G: 0xFF, B: 0x41, A: 0xFF}

	// Draw semicircle track.
	for deg := 0; deg <= 180; deg++ {
		angle := math.Pi + float64(deg)*math.Pi/180.0
		for rad := r - 2; rad <= r; rad++ {
			px := cx + int(float64(rad)*math.Cos(angle))
			py := cy + int(float64(rad)*math.Sin(angle))
			setPixelSafe(img, px, py, w, h, trackCol)
		}
	}

	// Draw needle.
	fraction := cur / maxV
	if fraction > 1 {
		fraction = 1
	}
	needleAngle := math.Pi + fraction*math.Pi
	nx := cx + int(float64(r-4)*math.Cos(needleAngle))
	ny := cy + int(float64(r-4)*math.Sin(needleAngle))
	drawLine(img, cx, cy, nx, ny, w, h, needleCol)

	// Labels.
	label := fmt.Sprintf("%.0f d/s", cur)
	tw := bitmapTextWidth(basicfont.Face7x13, label)
	drawBitmapText(img, basicfont.Face7x13, label, cx-tw/2, cy+r/2,
		color.RGBA{R: 0xe6, G: 0xed, B: 0xf3, A: 0xFF})

	return img
}

// drawLine draws a simple Bresenham line.
func drawLine(img *image.RGBA, x0, y0, x1, y1, w, h int, col color.RGBA) {
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx - dy
	for {
		setPixelSafe(img, x0, y0, w, h, col)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func (sp *StatsPanel) memTicker() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		mb := float64(ms.HeapAlloc) / 1024 / 1024
		fyne.Do(func() {
			sp.memLabel.SetText(fmt.Sprintf("Heap: %.1f MB", mb))
		})
	}
}

// Widget returns the complete stats panel as a scrollable container.
func (sp *StatsPanel) Widget() fyne.CanvasObject {
	workerScroll := container.NewVScroll(sp.workerList)
	workerScroll.SetMinSize(fyne.NewSize(0, 120))

	return container.NewVBox(
		widget.NewSeparator(),
		widget.NewLabel("Digits computed"),
		sp.countLabel,
		widget.NewSeparator(),
		sp.timerLabel,
		sp.roundLabel,
		sp.etaLabel,
		widget.NewSeparator(),
		widget.NewLabel("Speed"),
		sp.speedRaster,
		sp.Sparkline,
		widget.NewSeparator(),
		sp.memLabel,
		widget.NewSeparator(),
		widget.NewLabel("Workers"),
		workerScroll,
		widget.NewSeparator(),
		widget.NewLabel("Digit Frequency"),
		sp.Histogram,
		widget.NewSeparator(),
		widget.NewLabel("CPU"),
		sp.CPUDials,
	)
}

// formatStatInt is like engine's formatInt but local.
func formatStatInt(n int64) string {
	if n < 0 {
		return fmt.Sprintf("%d", n)
	}
	s := fmt.Sprintf("%d", n)
	out := make([]byte, 0, len(s)+len(s)/3)
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(ch))
	}
	return string(out)
}

// FormatInt is the exported version of formatStatInt for use by main.go.
func FormatInt(n int64) string { return formatStatInt(n) }

// fixedSizeLayout is a fyne.Layout that forces all objects to a fixed size.
type fixedSizeLayout struct{ size fyne.Size }

func (f *fixedSizeLayout) Layout(objs []fyne.CanvasObject, _ fyne.Size) {
	for _, o := range objs {
		o.Resize(f.size)
		o.Move(fyne.NewPos(0, 0))
	}
}

func (f *fixedSizeLayout) MinSize(_ []fyne.CanvasObject) fyne.Size { return f.size }
