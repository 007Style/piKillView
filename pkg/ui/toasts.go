package ui

import (
	"image/color"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

// ToastOverlay shows transient milestone messages over the digit view.
type ToastOverlay struct {
	mu   sync.Mutex
	bg   *canvas.Rectangle
	text *canvas.Text
	cont *fyne.Container
}

// NewToastOverlay creates a hidden toast overlay.
func NewToastOverlay() *ToastOverlay {
	bg := canvas.NewRectangle(color.RGBA{R: 0x00, G: 0xFF, B: 0x41, A: 0xDD})
	bg.CornerRadius = 8

	txt := canvas.NewText("", color.RGBA{R: 0x0d, G: 0x11, B: 0x17, A: 0xFF})
	txt.TextSize = 20
	txt.TextStyle = fyne.TextStyle{Bold: true}
	txt.Alignment = fyne.TextAlignCenter

	c := container.NewStack(bg, txt)
	c.Hide()

	return &ToastOverlay{bg: bg, text: txt, cont: c}
}

// Widget returns the overlay container to be placed in a Stack.
func (t *ToastOverlay) Widget() fyne.CanvasObject { return t.cont }

// Show displays msg with fade-in → hold → fade-out sequence.
func (t *ToastOverlay) Show(msg string) {
	t.mu.Lock()
	t.text.Text = msg
	t.mu.Unlock()

	fyne.Do(func() {
		t.text.Text = msg
		t.cont.Show()
		canvas.Refresh(t.cont)
	})

	go func() {
		// Fade in 200 ms
		fadeDuration := 200 * time.Millisecond
		steps := 20
		for i := 0; i <= steps; i++ {
			alpha := uint8(float64(i) / float64(steps) * 0xDD)
			fyne.Do(func() {
				t.bg.FillColor = color.RGBA{R: 0x00, G: 0xFF, B: 0x41, A: alpha}
				canvas.Refresh(t.bg)
			})
			time.Sleep(fadeDuration / time.Duration(steps))
		}

		// Hold 2500 ms
		time.Sleep(2500 * time.Millisecond)

		// Fade out 300 ms
		outDuration := 300 * time.Millisecond
		for i := steps; i >= 0; i-- {
			alpha := uint8(float64(i) / float64(steps) * 0xDD)
			fyne.Do(func() {
				t.bg.FillColor = color.RGBA{R: 0x00, G: 0xFF, B: 0x41, A: alpha}
				canvas.Refresh(t.bg)
			})
			time.Sleep(outDuration / time.Duration(steps))
		}

		fyne.Do(func() {
			t.cont.Hide()
		})
	}()
}
