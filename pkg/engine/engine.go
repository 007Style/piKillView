package engine

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Config controls how the Engine runs.
type Config struct {
	ThreadMult       int           // 0=single, 1–4 = N×CPU workers
	WriteToFile      bool          // write digits to a file in OutputDir
	OutputDir        string        // directory for pi-<datetime>.txt; "" = current dir
	DigitLimit       int64         // stop after this many digits (0 = unlimited)
	AutoSaveInterval time.Duration // flush file buffer every interval (0 = disabled)
}

// Engine drives the π computation and exposes three output channels the UI
// reads independently.
type Engine struct {
	// Output channels — created fresh in Start(), closed in Stop().
	DigitCh      chan string
	CountCh      chan int64
	WorkerStatCh chan WorkerStat

	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mu      sync.Mutex
	running bool
}

// NewEngine allocates an Engine with nil channels (not yet started).
func NewEngine() *Engine {
	return &Engine{}
}

// Start launches the computation pipeline.  Calling Start while already
// running is a no-op.
func (e *Engine) Start(cfg Config) {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return
	}
	e.running = true

	// Fresh channels every start.
	e.DigitCh = make(chan string, 8)
	e.CountCh = make(chan int64, 16)
	e.WorkerStatCh = make(chan WorkerStat, 64)

	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	e.mu.Unlock()

	e.wg.Add(1)
	go e.loop(ctx, cfg)
}

// Stop cancels the computation, waits for it to finish, then closes all
// channels.  Safe to call multiple times.
func (e *Engine) Stop() {
	e.mu.Lock()
	if !e.running {
		e.mu.Unlock()
		return
	}
	cancel := e.cancel
	e.mu.Unlock()

	cancel()
	e.wg.Wait()

	e.mu.Lock()
	e.running = false
	e.mu.Unlock()
}

// Resume reads a previously written pi file, counts the valid digit characters
// after "3.", and returns that count so the caller can pass it as startDigits.
func (e *Engine) Resume(filePath string) (int64, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	var count int64
	seenDot := false
	for sc.Scan() {
		line := sc.Text()
		if !seenDot {
			idx := strings.IndexByte(line, '.')
			if idx < 0 {
				continue
			}
			line = line[idx+1:]
			seenDot = true
		}
		for _, ch := range line {
			if ch >= '0' && ch <= '9' {
				count++
			}
		}
	}
	if err := sc.Err(); err != nil {
		return count, err
	}
	return count, nil
}

// loop is the main compute goroutine.
func (e *Engine) loop(ctx context.Context, cfg Config) {
	defer e.wg.Done()
	defer close(e.DigitCh)
	defer close(e.CountCh)
	defer close(e.WorkerStatCh)

	workers := 1
	if cfg.ThreadMult > 0 {
		workers = runtime.NumCPU() * cfg.ThreadMult
	}

	// Open output file if requested.
	var (
		outFile   *os.File
		outBuf    *bufio.Writer
		autoFlush <-chan time.Time
	)
	if cfg.WriteToFile {
		dir := cfg.OutputDir
		if dir == "" {
			dir = "."
		}
		ts := time.Now().Format("2006-01-02T15-04-05")
		fname := filepath.Join(dir, fmt.Sprintf("pi-%s.txt", ts))
		var err error
		outFile, err = os.OpenFile(fname, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err == nil {
			outBuf = bufio.NewWriterSize(outFile, writeBufferSize)
			fmt.Fprint(outBuf, "3.")
		}
		if cfg.AutoSaveInterval > 0 && outBuf != nil {
			t := time.NewTicker(cfg.AutoSaveInterval)
			defer t.Stop()
			autoFlush = t.C
		}
	}

	flushFile := func() {
		if outBuf != nil {
			outBuf.Flush()
		}
	}
	defer func() {
		flushFile()
		if outFile != nil {
			outFile.Close()
		}
	}()

	send := func(ch interface{}, val interface{}) bool {
		select {
		case <-ctx.Done():
			return false
		default:
		}
		switch c := ch.(type) {
		case chan string:
			select {
			case c <- val.(string):
			case <-ctx.Done():
				return false
			}
		case chan int64:
			select {
			case c <- val.(int64):
			case <-ctx.Done():
				return false
			}
		}
		return true
	}
	_ = send // used below via inline select for clarity

	prevDigits := ""
	digits := startDigits
	var totalDigits int64

	for {
		// Check auto-flush ticker.
		select {
		case <-ctx.Done():
			return
		case <-autoFlush:
			flushFile()
		default:
		}

		prec := uint(float64(digits)*3.32193) + 64
		piStr := piDigitsText(prec, workers, ctx, e.WorkerStatCh)
		if piStr == "" {
			// Cancelled inside compute.
			return
		}
		fracPart := afterDot(piStr)

		newLen := len(fracPart) - safetyMargin
		if newLen <= len(prevDigits) {
			digits *= 2
			continue
		}
		newDigits := fracPart[len(prevDigits):newLen]
		if len(newDigits) == 0 {
			digits *= 2
			continue
		}

		// Write to file.
		if outBuf != nil {
			fmt.Fprint(outBuf, newDigits)
		}

		// Emit to UI channels.
		select {
		case e.DigitCh <- newDigits:
		case <-ctx.Done():
			return
		}

		totalDigits += int64(len(newDigits))

		select {
		case e.CountCh <- totalDigits:
		case <-ctx.Done():
			return
		}

		// Respect digit limit.
		if cfg.DigitLimit > 0 && totalDigits >= cfg.DigitLimit {
			return
		}

		prevDigits = fracPart[:newLen]
		digits *= 2
	}
}
