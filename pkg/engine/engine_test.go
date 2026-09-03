package engine

import (
	"strings"
	"testing"
)

// known first 50 decimal digits of π after the decimal point.
const knownPi50 = "14159265358979323846264338327950288419716939937510"

func computeFirst100(t *testing.T, workers int) string {
	t.Helper()
	e := NewEngine()
	e.Start(Config{
		ThreadMult: func() int {
			// workers==1 → single-threaded (ThreadMult=0)
			if workers <= 1 {
				return 0
			}
			return 1 // 1×CPU, but we'll have at least 4 goroutines on any machine
		}(),
		DigitLimit: 100,
	})
	var sb strings.Builder
	for chunk := range e.DigitCh {
		sb.WriteString(chunk)
		if int64(sb.Len()) >= 100 {
			break
		}
	}
	e.Stop()
	// Drain remaining channels so Stop() goroutine can finish cleanly.
	for range e.CountCh {
	}
	for range e.WorkerStatCh {
	}
	return sb.String()
}

// TestFirstDigits verifies the first 50 decimal digits with single-threaded compute.
func TestFirstDigits(t *testing.T) {
	digits := computeFirst100(t, 1)
	if len(digits) < 50 {
		t.Fatalf("got only %d digits, want at least 50", len(digits))
	}
	got := digits[:50]
	if got != knownPi50 {
		t.Errorf("first 50 digits mismatch\n got: %s\nwant: %s", got, knownPi50)
	}
}

// TestFirstDigitsMultiWorker verifies the first 50 decimal digits with multiple workers.
func TestFirstDigitsMultiWorker(t *testing.T) {
	digits := computeFirst100(t, 4)
	if len(digits) < 50 {
		t.Fatalf("got only %d digits, want at least 50", len(digits))
	}
	got := digits[:50]
	if got != knownPi50 {
		t.Errorf("first 50 digits mismatch\n got: %s\nwant: %s", got, knownPi50)
	}
}
