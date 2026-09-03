package ui

import (
	"fmt"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const maxBufBytes = 10_485_760 // 10 MB

// DigitView displays the π digit stream in a color-coded, scrollable RichText.
type DigitView struct {
	mu           sync.Mutex
	buf          []byte  // ring buffer of digit characters
	bufOffset    int64   // absolute index of buf[0]
	richText     *widget.RichText
	rangeLabel   *widget.Label
	scroll       *container.Scroll
	autoScroll   bool
	highlightStart int64
	highlightLen   int64
}

// NewDigitView creates an empty digit view.
func NewDigitView() *DigitView {
	dv := &DigitView{
		autoScroll: true,
	}
	dv.richText = widget.NewRichText()
	dv.richText.Wrapping = fyne.TextWrapBreak
	dv.rangeLabel = widget.NewLabel("Viewing digits 0 – 0")
	dv.rangeLabel.TextStyle = fyne.TextStyle{Monospace: true}

	dv.scroll = container.NewVScroll(dv.richText)
	dv.scroll.SetMinSize(fyne.NewSize(0, 200))

	return dv
}

// AppendDigits adds new digit characters to the ring buffer and updates
// the RichText with only the new segments appended.
func (dv *DigitView) AppendDigits(s string) {
	dv.mu.Lock()
	defer dv.mu.Unlock()

	// Append to buffer.
	dv.buf = append(dv.buf, []byte(s)...)

	// Trim from front if over limit.
	if len(dv.buf) > maxBufBytes {
		excess := len(dv.buf) - maxBufBytes
		dv.bufOffset += int64(excess)
		dv.buf = dv.buf[excess:]
	}

	// Build new RichText segments for s only (appended portion).
	// Group same-color runs.
	newSegs := buildSegments(s, dv.bufOffset+int64(len(dv.buf))-int64(len(s)),
		dv.highlightStart, dv.highlightLen)

	dv.richText.Segments = append(dv.richText.Segments, newSegs...)
	// Trim leading segments if buffer was trimmed above.
	dv.trimLeadingSegments()

	endIdx := dv.bufOffset + int64(len(dv.buf))
	dv.rangeLabel.SetText(fmt.Sprintf("Viewing digits %s – %s",
		formatStatInt(dv.bufOffset), formatStatInt(endIdx)))
	dv.richText.Refresh()

	if dv.autoScroll {
		dv.scroll.ScrollToBottom()
	}
}

// buildSegments converts digit string s into color-coded RichText segments.
// absStart is the absolute digit index of s[0].
func buildSegments(s string, absStart, hlStart, hlLen int64) []widget.RichTextSegment {
	if len(s) == 0 {
		return nil
	}

	hlEnd := hlStart + hlLen

	var segs []widget.RichTextSegment
	var curText []byte
	var curColorName fyne.ThemeColorName
	inHL := false

	flush := func() {
		if len(curText) == 0 {
			return
		}
		cn := curColorName
		if inHL {
			cn = "primary" // use theme primary for highlight
		}
		segs = append(segs, &widget.TextSegment{
			Style: widget.RichTextStyle{
				ColorName: cn,
				Inline:    true,
				TextStyle: fyne.TextStyle{Monospace: true},
			},
			Text: string(curText),
		})
		curText = curText[:0]
	}

	for i, ch := range s {
		absIdx := absStart + int64(i)
		isHL := hlLen > 0 && absIdx >= hlStart && absIdx < hlEnd
		d := int(ch - '0')
		if d < 0 || d > 9 {
			continue
		}
		cn := DigitColorNames[d]
		if i == 0 {
			curColorName = cn
			inHL = isHL
		}
		if cn != curColorName || isHL != inHL {
			flush()
			curColorName = cn
			inHL = isHL
		}
		curText = append(curText, byte(ch))
	}
	flush()
	return segs
}

// trimLeadingSegments removes segments whose content has been trimmed from the
// front of the buffer (best-effort approximation by segment count).
func (dv *DigitView) trimLeadingSegments() {
	// Estimate: each segment is ~50 chars on average.  Remove segments until
	// total estimated chars <= maxBufBytes.
	const avgSegChars = 50
	maxSegs := maxBufBytes / avgSegChars
	if len(dv.richText.Segments) > maxSegs {
		excess := len(dv.richText.Segments) - maxSegs
		dv.richText.Segments = dv.richText.Segments[excess:]
	}
}

// Buffer returns a copy of the current ring buffer.
func (dv *DigitView) Buffer() []byte {
	dv.mu.Lock()
	defer dv.mu.Unlock()
	out := make([]byte, len(dv.buf))
	copy(out, dv.buf)
	return out
}

// Reset clears the digit view.
func (dv *DigitView) Reset() {
	dv.mu.Lock()
	dv.buf = dv.buf[:0]
	dv.bufOffset = 0
	dv.highlightStart = 0
	dv.highlightLen = 0
	dv.richText.Segments = nil
	dv.mu.Unlock()
	dv.richText.Refresh()
	dv.rangeLabel.SetText("Viewing digits 0 – 0")
}

// SetHighlight marks a region for highlight coloring on the next refresh.
func (dv *DigitView) SetHighlight(start, length int64) {
	dv.mu.Lock()
	dv.highlightStart = start
	dv.highlightLen = length
	dv.mu.Unlock()
}

// Widget returns the digit view container with the range label below.
func (dv *DigitView) Widget() fyne.CanvasObject {
	return container.NewBorder(nil, dv.rangeLabel, nil, nil, dv.scroll)
}
