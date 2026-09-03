// Package engine provides the Chudnovsky π computation engine for piKillView.
package engine

import (
	"fmt"
	"strings"
)

// WorkerStat carries a per-worker progress snapshot emitted to the UI.
type WorkerStat struct {
	WorkerID      int
	TermsComputed int64
	Active        bool
}

// formatInt formats n with comma separators, e.g. 1234567 → "1,234,567".
func formatInt(n int64) string {
	s := fmt.Sprintf("%d", n)
	if n < 0 {
		s = s[1:] // strip minus for grouping, re-add later
	}
	var b strings.Builder
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(ch)
	}
	if n < 0 {
		return "-" + b.String()
	}
	return b.String()
}
