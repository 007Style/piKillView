package ui

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"math/rand"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// First 1000 digits of π after the decimal point (no leading "3").
const piDigits1000 = "1415926535897932384626433832795028841971693993751058209749445923078164062862089986280348253421170679821480865132823066470938446095505822317253594081284811174502841027019385211055596446229489549303819644288109756659334461284756482337867831652712019091456485669234603486104543266482133936072602491412737245870066063155881748815209209628292540917153643678925903600113305305488204665213841469519415116094330572703657595919530921861173819326117931051185480744623799627495673518857527248912279381830119491298336733624406566430860213949463952247371907021798609437027705392171762931767523846748184676694051320005681271452635608277857713427577896091736371787214684409012249534301465495853710507922796892589235420199561121290219608640344181598136297747713099605187072113499999372978049951059731732816096318595024459455346908302642522308253344685035261931188171010003137838752886587533208381420617177669147303598253490428755468731159562863882353787593751957781857780532171226806613001927876611195909216420199"

// rainColumn is one falling column of digits in the header rain animation.
type rainColumn struct {
	x      float64 // horizontal position in pixels
	y      float64 // current head position in pixels
	speed  float64 // pixels per frame
	offset int     // index into piDigits1000 for character cycling
	length int     // trail length in characters
}

// HeaderWidget displays the animated digit-rain + pulsing π symbol.
type HeaderWidget struct {
	widget.BaseWidget

	mu      sync.Mutex
	cols    []*rainColumn
	initW   float32 // width at last column init
	raster  *canvas.Raster
	stop    chan struct{}
	piSize  float32 // current pulsed size of the π label
	piAlpha uint8   // current pulsed alpha of the π label
}

// NewHeaderWidget creates and starts the animated header.
func NewHeaderWidget() *HeaderWidget {
	h := &HeaderWidget{
		stop:    make(chan struct{}),
		piSize:  72,
		piAlpha: 200,
	}
	h.ExtendBaseWidget(h)
	return h
}

// MinSize satisfies fyne.Widget.
func (h *HeaderWidget) MinSize() fyne.Size {
	return fyne.NewSize(0, 130)
}

// CreateRenderer satisfies fyne.Widget.
func (h *HeaderWidget) CreateRenderer() fyne.WidgetRenderer {
	h.raster = canvas.NewRaster(h.draw)
	h.raster.SetMinSize(fyne.NewSize(0, 130))

	go h.animate()
	return widget.NewSimpleRenderer(h.raster)
}

// Stop halts the animation goroutine.
func (h *HeaderWidget) Stop() {
	select {
	case <-h.stop:
	default:
		close(h.stop)
	}
}

// initColumns creates or re-creates rain columns to match the current width.
func (h *HeaderWidget) initColumns(w float32) {
	n := int(w/18) + 1
	if n < 5 {
		n = 5
	}
	cols := make([]*rainColumn, n)
	for i := range cols {
		cols[i] = &rainColumn{
			x:      float64(i) * (float64(w) / float64(n)),
			y:      -float64(rand.Intn(130)),
			speed:  1.5 + rand.Float64()*3.0,
			offset: rand.Intn(len(piDigits1000)),
			length: 6 + rand.Intn(14),
		}
	}
	h.mu.Lock()
	h.cols = cols
	h.initW = w
	h.mu.Unlock()
}

// animate runs the 30fps ticker that advances columns and pulses the π symbol.
func (h *HeaderWidget) animate() {
	ticker := time.NewTicker(33 * time.Millisecond)
	defer ticker.Stop()
	start := time.Now()
	for {
		select {
		case <-h.stop:
			return
		case <-ticker.C:
			elapsed := time.Since(start).Seconds()

			// Pulse π symbol: sine wave over 2s period.
			t := float32(math.Sin(elapsed * math.Pi)) // 0.5 Hz = 2s
			h.mu.Lock()
			h.piSize = 60 + 20*t
			// alpha oscillates 160–255
			h.piAlpha = uint8(160 + 95*float64(t+1)/2)

			// Advance rain columns.
			for _, c := range h.cols {
				c.y += c.speed
			}
			h.mu.Unlock()

			r := h.raster
			fyne.Do(func() { canvas.Refresh(r) })
		}
	}
}

// draw renders one frame of the digit rain into an image.RGBA.
func (h *HeaderWidget) draw(w, wh int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, wh))
	// Fill background.
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{R: 0x0d, G: 0x11, B: 0x17, A: 0xFF}}, image.Point{}, draw.Src)

	h.mu.Lock()
	cols := h.cols
	piSize := h.piSize
	piAlpha := h.piAlpha

	// Re-init columns if width changed significantly.
	if len(cols) == 0 || h.initW == 0 {
		h.mu.Unlock()
		h.initColumns(float32(w))
		h.mu.Lock()
		cols = h.cols
	}
	h.mu.Unlock()

	charH := 14 // basicfont face height
	charW := 7  // basicfont.Face7x13 is 7px wide

	for _, c := range cols {
		headY := int(c.y)
		for row := 0; row <= c.length; row++ {
			py := headY - row*charH
			if py < -charH || py > wh+charH {
				continue
			}
			charIdx := (c.offset + row) % len(piDigits1000)
			ch := rune(piDigits1000[charIdx])

			var col color.RGBA
			if row == 0 {
				// Head: bright green
				col = color.RGBA{R: 0x00, G: 0xFF, B: 0x41, A: 0xFF}
			} else {
				// Trail: exponential fade
				fade := math.Exp(-float64(row) * 0.35)
				g := uint8(fade * 180)
				col = color.RGBA{G: g, A: 0xFF}
			}
			drawChar(img, charW, charH, int(c.x), py, ch, col)
		}

		// Wrap column.
		if c.y > float64(wh)+float64(c.length*charH) {
			c.y = -float64(c.length * charH)
			c.offset = rand.Intn(len(piDigits1000))
			c.speed = 1.5 + rand.Float64()*3.0
		}
	}

	// Draw pulsing π symbol in center.
	_ = piSize // used to scale the symbol; we approximate with text
	cx := w / 2
	cy := wh / 2
	piCol := color.RGBA{R: 0x00, G: 0xFF, B: 0x41, A: piAlpha}
	drawLargePI(img, cx, cy, piCol)

	return img
}

// drawChar renders a single rune from basicfont at pixel position (x,y).
func drawChar(img *image.RGBA, charW, charH, x, py int, ch rune, col color.RGBA) {
	face := basicfont.Face7x13
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.P(x, py+charH-2),
	}
	d.DrawString(string(ch))
	_ = charW
}

// drawLargePI draws a large "π" character by repeating basicfont scaled up.
// We draw it 6×6 pixel-enlarged by rendering the character at multiple offsets.
func drawLargePI(img *image.RGBA, cx, cy int, col color.RGBA) {
	scale := 6
	face := basicfont.Face7x13
	str := "π"
	bounds, _ := font.BoundString(face, str)
	sw := (bounds.Max.X - bounds.Min.X).Ceil()
	sh := (bounds.Max.Y - bounds.Min.Y).Ceil()

	// Render into a small temp image then scale-blit.
	tmp := image.NewRGBA(image.Rect(0, 0, sw+4, sh+4))
	d := &font.Drawer{
		Dst:  tmp,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.P(2, sh+1),
	}
	d.DrawString(str)

	// Scale-blit to main image centered at (cx,cy).
	dstX0 := cx - (sw*scale)/2
	dstY0 := cy - (sh*scale)/2
	for sy := 0; sy < tmp.Bounds().Dy(); sy++ {
		for sx := 0; sx < tmp.Bounds().Dx(); sx++ {
			src := tmp.RGBAAt(sx, sy)
			if src.A == 0 {
				continue
			}
			src.A = col.A
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					px := dstX0 + sx*scale + dx
					py := dstY0 + sy*scale + dy
					if px >= 0 && py >= 0 && px < img.Bounds().Dx() && py < img.Bounds().Dy() {
						img.SetRGBA(px, py, src)
					}
				}
			}
		}
	}
}
