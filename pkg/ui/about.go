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
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const (
	AppVersion = "v1.1"
	AppTagline = "From the minds of IBM Bob & Daneyand"
)

// aboutRainCol is one falling column for the About dialog animation.
type aboutRainCol struct {
	x, y, speed float64
	offset      int
	length      int
	hue         float64 // 0=green, 1=cyan, 2=blue — slowly drifts
}

// aboutCanvas is the full-window animated canvas inside the About dialog.
type aboutCanvas struct {
	widget.BaseWidget

	mu      sync.Mutex
	cols    []*aboutRainCol
	initW   int
	raster  *canvas.Raster
	stop    chan struct{}
	frame   float64 // total elapsed seconds
}

func newAboutCanvas() *aboutCanvas {
	ac := &aboutCanvas{stop: make(chan struct{})}
	ac.ExtendBaseWidget(ac)
	return ac
}

func (ac *aboutCanvas) MinSize() fyne.Size { return fyne.NewSize(560, 360) }

func (ac *aboutCanvas) CreateRenderer() fyne.WidgetRenderer {
	ac.raster = canvas.NewRaster(ac.draw)
	ac.raster.SetMinSize(ac.MinSize())
	go ac.animate()
	return widget.NewSimpleRenderer(ac.raster)
}

func (ac *aboutCanvas) stopAnim() {
	select {
	case <-ac.stop:
	default:
		close(ac.stop)
	}
}

func (ac *aboutCanvas) initCols(w int) {
	n := w/16 + 1
	if n < 8 {
		n = 8
	}
	cols := make([]*aboutRainCol, n)
	for i := range cols {
		cols[i] = &aboutRainCol{
			x:      float64(i) * (float64(w) / float64(n)),
			y:      -float64(rand.Intn(360)),
			speed:  1.0 + rand.Float64()*2.5,
			offset: rand.Intn(len(piDigits1000)),
			length: 8 + rand.Intn(16),
			hue:    rand.Float64(),
		}
	}
	ac.mu.Lock()
	ac.cols = cols
	ac.initW = w
	ac.mu.Unlock()
}

func (ac *aboutCanvas) animate() {
	ticker := time.NewTicker(33 * time.Millisecond)
	defer ticker.Stop()
	start := time.Now()
	for {
		select {
		case <-ac.stop:
			return
		case <-ticker.C:
			elapsed := time.Since(start).Seconds()
			ac.mu.Lock()
			ac.frame = elapsed
			for _, c := range ac.cols {
				c.y += c.speed
				// Slowly drift hue for color variety.
				c.hue += 0.002
				if c.hue > 3 {
					c.hue = 0
				}
			}
			ac.mu.Unlock()
			r := ac.raster
			fyne.Do(func() { canvas.Refresh(r) })
		}
	}
}

// hueToColor maps a hue value (0–3) to a rain digit color.
func hueToColor(hue float64, alpha uint8) color.RGBA {
	h := hue - math.Floor(hue/3)*3
	switch {
	case h < 1: // green
		t := h
		return color.RGBA{G: uint8(200 + 55*t), A: alpha}
	case h < 2: // green→cyan
		t := h - 1
		return color.RGBA{R: 0, G: uint8(180 + 75*t), B: uint8(200 * t), A: alpha}
	default: // cyan→blue
		t := h - 2
		return color.RGBA{R: 0, G: uint8(255 * (1 - t)), B: uint8(200 + 55*t), A: alpha}
	}
}

func (ac *aboutCanvas) draw(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	// Deep dark background.
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{R: 0x06, G: 0x08, B: 0x10, A: 0xFF}}, image.Point{}, draw.Src)

	ac.mu.Lock()
	cols := ac.cols
	frame := ac.frame
	if len(cols) == 0 || ac.initW == 0 {
		ac.mu.Unlock()
		ac.initCols(w)
		ac.mu.Lock()
		cols = ac.cols
	}
	ac.mu.Unlock()

	charH := 14
	charW := 7

	for _, c := range cols {
		headY := int(c.y)
		for row := 0; row <= c.length; row++ {
			py := headY - row*charH
			if py < -charH || py > h+charH {
				continue
			}
			charIdx := (c.offset + row) % len(piDigits1000)
			ch := rune(piDigits1000[charIdx])

			var col color.RGBA
			if row == 0 {
				// Bright head — pure white flash.
				col = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
			} else {
				fade := math.Exp(-float64(row) * 0.28)
				alpha := uint8(fade * 220)
				col = hueToColor(c.hue+float64(row)*0.1, alpha)
			}
			d := &font.Drawer{
				Dst:  img,
				Src:  image.NewUniform(col),
				Face: basicfont.Face7x13,
				Dot:  fixed.P(int(c.x), py+charH-2),
			}
			d.DrawString(string(ch))
			_ = charW
		}

		if c.y > float64(h)+float64(c.length*charH) {
			c.y = -float64(c.length * charH)
			c.offset = rand.Intn(len(piDigits1000))
			c.speed = 1.0 + rand.Float64()*2.5
		}
	}

	// ── Central π glyph — large, pulsing, multi-color glow ──
	cx := w / 2
	cy := h/2 - 40

	pulse := 0.5 + 0.5*math.Sin(frame*math.Pi) // 0–1 over 2s
	baseScale := 10
	scaleF := float64(baseScale) + 2*pulse

	// Draw three concentric π layers for a glow effect.
	glowColors := []color.RGBA{
		{R: 0x00, G: 0x88, B: 0x44, A: uint8(40 + 30*pulse)},  // outer glow
		{R: 0x00, G: 0xCC, B: 0x66, A: uint8(100 + 60*pulse)}, // mid
		{R: 0x00, G: 0xFF, B: 0x88, A: uint8(200 + 55*pulse)}, // bright core
	}
	glowScales := []float64{scaleF + 3, scaleF + 1.5, scaleF}
	for i, gc := range glowColors {
		drawScaledChar(img, cx, cy, 'π', gc, glowScales[i])
	}

	// ── Version number ──
	versionStr := "piKillView " + AppVersion
	drawCenteredBitmapText(img, versionStr, cx, cy+int(scaleF)*9+8,
		color.RGBA{R: 0x00, G: 0xFF, B: 0x88, A: 0xFF}, 2)

	// ── Tagline ──
	taglineCol := color.RGBA{R: 0xCC, G: 0xDD, B: 0xFF, A: uint8(180 + 75*pulse)}
	drawCenteredBitmapText(img, AppTagline, cx, cy+int(scaleF)*9+34, taglineCol, 1)

	// ── Subtitle ──
	subtitleCol := color.RGBA{R: 0x66, G: 0x88, B: 0xAA, A: 0xCC}
	drawCenteredBitmapText(img, "Computing π forever — because why stop?", cx, cy+int(scaleF)*9+52, subtitleCol, 1)

	// ── Animated digit ring around π — digits orbit the symbol ──
	ringR := float64(int(scaleF)*5 + 30)
	nRingDigits := 16
	for i := 0; i < nRingDigits; i++ {
		angle := frame*0.8 + float64(i)*2*math.Pi/float64(nRingDigits)
		rx := cx + int(ringR*math.Cos(angle))
		ry := cy + int(ringR*math.Sin(angle))
		charIdx := (i * 7) % len(piDigits1000)
		ch := rune(piDigits1000[charIdx])
		// Brightness pulses per digit with phase offset.
		brightness := 0.4 + 0.6*math.Abs(math.Sin(frame*2+float64(i)*0.4))
		rc := color.RGBA{
			R: uint8(0 * brightness),
			G: uint8(200 * brightness),
			B: uint8(120 * brightness),
			A: uint8(180 * brightness),
		}
		d := &font.Drawer{
			Dst:  img,
			Src:  image.NewUniform(rc),
			Face: basicfont.Face7x13,
			Dot:  fixed.P(rx-3, ry+6),
		}
		d.DrawString(string(ch))
	}

	return img
}

// drawScaledChar renders a rune pixel-scaled and centered at (cx,cy).
func drawScaledChar(img *image.RGBA, cx, cy int, ch rune, col color.RGBA, scale float64) {
	sc := int(scale)
	if sc < 1 {
		sc = 1
	}
	face := basicfont.Face7x13
	str := string(ch)
	bounds, _ := font.BoundString(face, str)
	sw := (bounds.Max.X - bounds.Min.X).Ceil()
	sh := (bounds.Max.Y - bounds.Min.Y).Ceil()

	tmp := image.NewRGBA(image.Rect(0, 0, sw+4, sh+4))
	d := &font.Drawer{
		Dst:  tmp,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.P(2, sh+1),
	}
	d.DrawString(str)

	dstX0 := cx - (sw*sc)/2
	dstY0 := cy - (sh*sc)/2
	for sy := 0; sy < tmp.Bounds().Dy(); sy++ {
		for sx := 0; sx < tmp.Bounds().Dx(); sx++ {
			src := tmp.RGBAAt(sx, sy)
			if src.A == 0 {
				continue
			}
			src.R = col.R
			src.G = col.G
			src.B = col.B
			src.A = col.A
			for dy := 0; dy < sc; dy++ {
				for dx := 0; dx < sc; dx++ {
					px := dstX0 + sx*sc + dx
					py := dstY0 + sy*sc + dy
					if px >= 0 && py >= 0 && px < img.Bounds().Dx() && py < img.Bounds().Dy() {
						// Alpha-blend over existing pixel for glow layers.
						existing := img.RGBAAt(px, py)
						blended := alphaBlend(existing, src)
						img.SetRGBA(px, py, blended)
					}
				}
			}
		}
	}
}

// alphaBlend composites src over dst.
func alphaBlend(dst, src color.RGBA) color.RGBA {
	a := float64(src.A) / 255.0
	ia := 1.0 - a
	return color.RGBA{
		R: uint8(float64(src.R)*a + float64(dst.R)*ia),
		G: uint8(float64(src.G)*a + float64(dst.G)*ia),
		B: uint8(float64(src.B)*a + float64(dst.B)*ia),
		A: 0xFF,
	}
}

// drawCenteredBitmapText renders text centered at (cx, y) with pixel scale.
func drawCenteredBitmapText(img *image.RGBA, text string, cx, y int, col color.RGBA, scale int) {
	if scale < 1 {
		scale = 1
	}
	face := basicfont.Face7x13
	tw := bitmapTextWidth(face, text) * scale
	x := cx - tw/2
	if scale == 1 {
		drawBitmapText(img, face, text, x, y, col)
		return
	}
	// Render to temp, then scale-blit.
	bounds, _ := font.BoundString(face, text)
	sw := (bounds.Max.X - bounds.Min.X).Ceil() + 4
	sh := (bounds.Max.Y - bounds.Min.Y).Ceil() + 4
	tmp := image.NewRGBA(image.Rect(0, 0, sw, sh))
	d := &font.Drawer{
		Dst:  tmp,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.P(2, sh-2),
	}
	d.DrawString(text)
	dstX0 := cx - (sw*scale)/2
	dstY0 := y - (sh*scale)/2
	for sy := 0; sy < tmp.Bounds().Dy(); sy++ {
		for sx := 0; sx < tmp.Bounds().Dx(); sx++ {
			src := tmp.RGBAAt(sx, sy)
			if src.A == 0 {
				continue
			}
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					px := dstX0 + sx*scale + dx
					py := dstY0 + sy*scale + dy
					if px >= 0 && py >= 0 && px < img.Bounds().Dx() && py < img.Bounds().Dy() {
						img.SetRGBA(px, py, col)
					}
				}
			}
		}
	}
}

// ShowAboutDialog displays the animated About dialog.
func ShowAboutDialog(win fyne.Window) {
	ac := newAboutCanvas()

	closeBtn := widget.NewButton("Close", nil)
	closeBtn.Importance = widget.HighImportance

	content := container.NewBorder(
		nil,
		container.NewCenter(closeBtn),
		nil, nil,
		ac,
	)

	d := dialog.NewCustomWithoutButtons("About piKillView", content, win)
	d.Resize(fyne.NewSize(580, 420))

	closeBtn.OnTapped = func() {
		ac.stopAnim()
		d.Hide()
	}

	d.SetOnClosed(func() { ac.stopAnim() })
	d.Show()
}
