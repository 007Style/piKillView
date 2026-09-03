package ui

import (
	"image"
	"image/color"
	"image/draw"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

var triviaFacts = []string{
	"NASA uses only 15 digits of π to calculate interplanetary trajectories — any more is unnecessary.",
	"The current world record for π digits is over 202 trillion digits (2021, by Guinness).",
	"The Feynman Point: digits 762–767 of π are all 9s — physicist Richard Feynman joked he'd memorize π to that point.",
	"Buffon's Needle: dropping needles on a lined floor gives a probabilistic estimate of π.",
	"π appears in the Bible (1 Kings 7:23) — a circular object is described with circumference/diameter = 3.",
	"The first trillion digits of π were computed by Yasumasa Kanada in 2002.",
	"The Bailey–Borwein–Plouffe (BBP) formula allows computing any hex digit of π without knowing prior digits.",
	"Ramanujan's series for π converges at nearly 8 decimal digits per term — one of the fastest known.",
	"π is both irrational and transcendental — it cannot be the root of any polynomial with rational coefficients.",
	"The digits of π pass all known tests for statistical randomness — they appear 'normal'.",
	"In 1897, the Indiana Pi Bill almost legislated π to equal 3.2 — it failed in the state senate.",
	"The symbol π was first used by William Jones in 1706 and popularized by Euler in 1737.",
	"Pi Day is March 14 (3/14) — Albert Einstein's birthday.",
	"The Great Pyramid of Giza has a perimeter-to-height ratio of approximately 2π.",
	"If you printed 1 trillion digits of π at 12pt font, the paper would stretch 1.3 billion km — past Jupiter.",
	"A Japanese man, Akira Haraguchi, recited π to 111,700 decimal places from memory in 2006.",
	"There are no patterns in π — or if there are, none have ever been proven to exist.",
	"π in base 10 has no repeating block; every finite string of digits appears somewhere (probably).",
	"The circumference of the observable universe calculated using only 39 digits of π is accurate to an atom.",
	"The Chudnovsky brothers computed 1 billion digits of π in 1989 using a homemade supercomputer.",
	"π is defined as the ratio of a circle's circumference to its diameter — in any flat (Euclidean) geometry.",
	"The value of π is encoded in the ratio of the Earth's equatorial circumference to its polar diameter (nearly).",
	"Euler's identity e^(iπ) + 1 = 0 links π, e, i, 1, and 0 — considered the most beautiful equation in math.",
	"π appears in the Gaussian integral: ∫e^(−x²)dx from −∞ to +∞ = √π.",
	"Stirling's approximation for n! includes π: n! ≈ √(2πn)(n/e)^n.",
	"The probability that two randomly chosen integers are coprime is 6/π².",
	"π shows up in the distribution of prime numbers via the Riemann zeta function.",
	"Georg Cantor proved that almost all real numbers are transcendental — π is one of the very few we know about.",
	"The Machin formula: π/4 = 4·arctan(1/5) − arctan(1/239) — used to compute π for centuries.",
	"π in quantum mechanics: Heisenberg's uncertainty principle is ΔxΔp ≥ h/(4π).",
	"General relativity's Einstein field equations contain π in the gravitational coupling constant.",
	"In 1706, John Machin computed 100 digits of π — a record that stood for a century.",
	"A supercomputer in 2022 computed 100 trillion digits of π in 157 days.",
	"The digits of π are thought to be uniformly distributed — each digit 0–9 appears ~10% of the time.",
	"π cannot be expressed as a fraction of two integers — it's not a ratio of whole numbers.",
	"River meanders: the ratio of a river's actual length to its straight-line distance averages approximately π.",
	"The Mandelbrot set boundary has infinite fractal complexity; π emerges from iterating z² + c near the set.",
	"The Wallis product: π/2 = (2/1)·(2/3)·(4/3)·(4/5)·(6/5)·(6/7)··· — an infinite product for π.",
	"Leibniz formula: π/4 = 1 − 1/3 + 1/5 − 1/7 + ··· — simple but converges extremely slowly.",
	"You are computing π right now. Welcome to the club.",
}

const (
	triviaScrollSpeed = 60.0 // pixels per second
	triviaFPS         = 33   // ms per frame
)

// TriviaWidget is a scrolling ticker of π facts.
type TriviaWidget struct {
	widget.BaseWidget

	mu       sync.Mutex
	xOffset  float64 // current scroll offset in pixels
	curIdx   int     // index of current fact
	nextIdx  int     // index of next fact
	imgW     int     // last known image width
	paused   bool
	raster   *canvas.Raster
	stopCh   chan struct{}
	lastTick time.Time
}

// NewTriviaWidget creates and starts the scrolling trivia ticker.
func NewTriviaWidget() *TriviaWidget {
	t := &TriviaWidget{
		stopCh:  make(chan struct{}),
		nextIdx: 1 % len(triviaFacts),
	}
	t.ExtendBaseWidget(t)
	return t
}

// MinSize satisfies fyne.Widget.
func (t *TriviaWidget) MinSize() fyne.Size {
	return fyne.NewSize(0, 28)
}

// CreateRenderer satisfies fyne.Widget.
func (t *TriviaWidget) CreateRenderer() fyne.WidgetRenderer {
	t.raster = canvas.NewRaster(t.draw)
	t.raster.SetMinSize(fyne.NewSize(0, 28))
	go t.animate()
	return widget.NewSimpleRenderer(t.raster)
}

// MouseIn pauses the ticker (implements desktop.Hoverable).
func (t *TriviaWidget) MouseIn(*desktop.MouseEvent) {
	t.mu.Lock()
	t.paused = true
	t.mu.Unlock()
}

// MouseMoved satisfies desktop.Hoverable.
func (t *TriviaWidget) MouseMoved(*desktop.MouseEvent) {}

// MouseOut resumes the ticker (implements desktop.Hoverable).
func (t *TriviaWidget) MouseOut() {
	t.mu.Lock()
	t.paused = false
	t.mu.Unlock()
}

var _ desktop.Hoverable = (*TriviaWidget)(nil)

// Stop halts the animation goroutine.
func (t *TriviaWidget) Stop() {
	select {
	case <-t.stopCh:
	default:
		close(t.stopCh)
	}
}

func (t *TriviaWidget) animate() {
	ticker := time.NewTicker(triviaFPS * time.Millisecond)
	defer ticker.Stop()
	t.lastTick = time.Now()
	for {
		select {
		case <-t.stopCh:
			return
		case now := <-ticker.C:
			dt := now.Sub(t.lastTick).Seconds()
			t.lastTick = now

			t.mu.Lock()
			if !t.paused {
				t.xOffset += triviaScrollSpeed * dt
				// Advance to next fact when current has scrolled off screen
				if t.imgW > 0 {
					curW := triviaTextWidth(triviaFacts[t.curIdx])
					sep := triviaTextWidth("   ·   ")
					if t.xOffset > float64(curW+sep) {
						t.xOffset -= float64(curW + sep)
						t.curIdx = t.nextIdx
						t.nextIdx = (t.nextIdx + 1) % len(triviaFacts)
					}
				}
			}
			t.mu.Unlock()
			r := t.raster
			fyne.Do(func() { canvas.Refresh(r) })
		}
	}
}

func triviaTextWidth(s string) int {
	face := basicfont.Face7x13
	bounds, _ := font.BoundString(face, s)
	return (bounds.Max.X - bounds.Min.X).Ceil()
}

func (t *TriviaWidget) draw(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	bg := color.RGBA{R: 0x0d, G: 0x11, B: 0x17, A: 0xFF}
	draw.Draw(img, img.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)

	t.mu.Lock()
	xOff := t.xOffset
	curFact := triviaFacts[t.curIdx]
	nextFact := triviaFacts[t.nextIdx]
	t.imgW = w
	t.mu.Unlock()

	textColor := color.RGBA{R: 0x8b, G: 0x94, B: 0x9e, A: 0xFF}
	accentColor := color.RGBA{R: 0x00, G: 0xFF, B: 0x41, A: 0xFF}

	face := basicfont.Face7x13
	baseline := h/2 + 5

	sep := "   ·   "
	curW := triviaTextWidth(curFact)
	sepW := triviaTextWidth(sep)

	// Draw current fact scrolling from right to left.
	x0 := w - int(xOff)
	drawText(img, face, curFact, x0, baseline, textColor)
	// Draw separator
	drawText(img, face, sep, x0+curW, baseline, accentColor)
	// Draw next fact
	drawText(img, face, nextFact, x0+curW+sepW, baseline, textColor)

	return img
}

func drawText(img *image.RGBA, face font.Face, s string, x, y int, col color.Color) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(s)
}
