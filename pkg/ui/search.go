package ui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// SearchEngine implements Boyer-Moore-Horspool string search.
type SearchEngine struct{}

// Search finds all occurrences of pattern in buf using Boyer-Moore-Horspool.
// Returns a slice of byte offsets where pattern starts.
func (se *SearchEngine) Search(pattern []byte, buf []byte) []int {
	n := len(buf)
	m := len(pattern)
	if m == 0 || n < m {
		return nil
	}

	// Build bad character shift table.
	var skip [256]int
	for i := range skip {
		skip[i] = m
	}
	for i := 0; i < m-1; i++ {
		skip[pattern[i]] = m - 1 - i
	}

	var matches []int
	i := m - 1
	for i < n {
		j := m - 1
		k := i
		for j >= 0 && buf[k] == pattern[j] {
			j--
			k--
		}
		if j < 0 {
			matches = append(matches, k+1)
		}
		i += skip[buf[i]]
	}
	return matches
}

// SearchBar embeds a search entry and navigation buttons.
type SearchBar struct {
	engine       *SearchEngine
	entry        *widget.Entry
	statusLabel  *widget.Label
	cont         *fyne.Container

	buf          []byte
	matches      []int
	currentMatch int

	OnResult func(matchStart int64, length int)
}

// NewSearchBar creates a search bar (hidden by default).
func NewSearchBar() *SearchBar {
	sb := &SearchBar{
		engine:      &SearchEngine{},
		statusLabel: widget.NewLabel(""),
	}

	sb.entry = widget.NewEntry()
	sb.entry.PlaceHolder = "Search digits…"
	sb.entry.OnSubmitted = func(_ string) { sb.doSearch() }

	findBtn := widget.NewButton("Find", func() { sb.doSearch() })
	nextBtn := widget.NewButton("Next", func() { sb.nextMatch() })
	prevBtn := widget.NewButton("Prev", func() { sb.prevMatch() })
	closeBtn := widget.NewButton("✕", func() { sb.Hide() })

	birthdayBtn := widget.NewButton("🎂 Birthday", func() {
		sb.entry.SetText(time.Now().Format("01022006"))
	})

	sb.cont = container.NewBorder(
		nil, nil, nil,
		container.NewHBox(birthdayBtn, findBtn, prevBtn, nextBtn, closeBtn),
		container.NewHBox(sb.entry, sb.statusLabel),
	)
	sb.cont.Hide()
	return sb
}

// Widget returns the search bar container.
func (sb *SearchBar) Widget() fyne.CanvasObject { return sb.cont }

// Show makes the search bar visible and focuses the entry.
func (sb *SearchBar) Show() {
	sb.cont.Show()
	sb.entry.FocusGained()
}

// Hide hides the search bar.
func (sb *SearchBar) Hide() {
	sb.cont.Hide()
}

// SetBuffer updates the corpus used for searching.
func (sb *SearchBar) SetBuffer(buf []byte) {
	sb.buf = buf
}

// doSearch runs the search in a background goroutine.
func (sb *SearchBar) doSearch() {
	pattern := []byte(sb.entry.Text)
	if len(pattern) == 0 {
		return
	}
	buf := sb.buf
	go func() {
		matches := sb.engine.Search(pattern, buf)
		fyne.Do(func() {
			sb.matches = matches
			sb.currentMatch = 0
			sb.updateStatus(len(pattern))
		})
	}()
}

func (sb *SearchBar) updateStatus(patLen int) {
	if len(sb.matches) == 0 {
		sb.statusLabel.SetText("No matches")
		return
	}
	sb.statusLabel.SetText(fmt.Sprintf("%d/%d", sb.currentMatch+1, len(sb.matches)))
	if sb.OnResult != nil && sb.currentMatch < len(sb.matches) {
		sb.OnResult(int64(sb.matches[sb.currentMatch]), patLen)
	}
}

func (sb *SearchBar) nextMatch() {
	if len(sb.matches) == 0 {
		return
	}
	sb.currentMatch = (sb.currentMatch + 1) % len(sb.matches)
	sb.updateStatus(len(sb.entry.Text))
}

func (sb *SearchBar) prevMatch() {
	if len(sb.matches) == 0 {
		return
	}
	sb.currentMatch = (sb.currentMatch - 1 + len(sb.matches)) % len(sb.matches)
	sb.updateStatus(len(sb.entry.Text))
}
