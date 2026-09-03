package ui

import (
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// LogEntry holds one timestamped event.
type LogEntry struct {
	Timestamp time.Time
	Message   string
}

// formatted returns the display string for this entry.
func (e LogEntry) formatted() string {
	return fmt.Sprintf("[%s] %s", e.Timestamp.Format("15:04:05.000"), e.Message)
}

// SessionLog is an append-only, collapsible event log widget.
type SessionLog struct {
	mu      sync.Mutex
	entries []LogEntry
	list    *widget.List
	scroll  *container.Scroll
}

// NewSessionLog creates an empty session log.
func NewSessionLog() *SessionLog {
	sl := &SessionLog{}
	sl.list = widget.NewList(
		func() int {
			sl.mu.Lock()
			defer sl.mu.Unlock()
			return len(sl.entries)
		},
		func() fyne.CanvasObject {
			lbl := widget.NewLabel("")
			lbl.TextStyle = fyne.TextStyle{Monospace: true}
			return lbl
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			sl.mu.Lock()
			defer sl.mu.Unlock()
			if id < len(sl.entries) {
				obj.(*widget.Label).SetText(sl.entries[id].formatted())
			}
		},
	)
	sl.scroll = container.NewVScroll(sl.list)
	sl.scroll.SetMinSize(fyne.NewSize(0, 150))
	return sl
}

// Append adds a timestamped message to the log.
func (sl *SessionLog) Append(msg string) {
	sl.mu.Lock()
	sl.entries = append(sl.entries, LogEntry{
		Timestamp: time.Now(),
		Message:   msg,
	})
	sl.mu.Unlock()
	sl.list.Refresh()
	// Scroll to bottom.
	sl.scroll.ScrollToBottom()
}

// Widget returns the scrollable list container.
func (sl *SessionLog) Widget() fyne.CanvasObject {
	return sl.scroll
}

// Collapse hides the log (sets min height to 0).
func (sl *SessionLog) Collapse() {
	sl.scroll.SetMinSize(fyne.NewSize(0, 0))
	sl.scroll.Hide()
}

// Expand shows the log at full height.
func (sl *SessionLog) Expand() {
	sl.scroll.SetMinSize(fyne.NewSize(0, 150))
	sl.scroll.Show()
}
