package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
	"github.com/007Style/piKillView/pkg/engine"
	"github.com/007Style/piKillView/pkg/ui"
)

var milestones = []int64{
	1_000, 10_000, 100_000,
	1_000_000, 10_000_000, 100_000_000, 1_000_000_000,
}

// pendingState holds all data written by background goroutines that the
// 30fps UI flush ticker will drain once per frame via a single fyne.Do.
// Using atomic / mutex-protected fields means goroutines never block on Fyne.
type pendingState struct {
	mu sync.Mutex

	// Digit chunks queued for display (coalesced between frames).
	digitBuf []byte

	// Latest digit count (only the most recent value matters).
	count    atomic.Int64
	countNew atomic.Bool // true when count was updated since last flush

	// Latest DPS value.
	dps    float64
	dpsNew bool

	// Worker stats (map keyed by WorkerID — latest wins per worker).
	workers    map[int]engine.WorkerStat
	workersNew bool

	// Milestones to fire (accumulate, drain per frame).
	toasts []string

	// engineDone signals the engine stopped between frames.
	engineDone atomic.Bool
}

func newPendingState() *pendingState {
	return &pendingState{workers: make(map[int]engine.WorkerStat)}
}

func main() {
	a := app.New()
	a.Settings().SetTheme(ui.NewDarkTheme())

	eng := engine.NewEngine()

	comp := &ui.AppComponents{}
	w := ui.NewMainWindow(a, eng, comp)

	controls := comp.Controls
	stats := comp.Stats
	digitView := comp.DigitView
	sessionLog := comp.SessionLog
	toast := comp.Toast
	search := comp.SearchBar

	firedMilestones := make(map[int64]bool)
	ps := newPendingState()

	// ── 30fps UI flush ticker ─────────────────────────────────────────
	// All high-frequency updates are coalesced here.  Background goroutines
	// write only to ps (no fyne.Do), this ticker flushes everything to Fyne
	// in one batch per frame — keeping the render queue clear at all times.
	flushStop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(33 * time.Millisecond) // ~30fps
		defer ticker.Stop()
		for {
			select {
			case <-flushStop:
				return
			case <-ticker.C:
				// Collect pending state under lock.
				ps.mu.Lock()
				digits := string(ps.digitBuf)
				ps.digitBuf = ps.digitBuf[:0]
				dps := ps.dps
				dpsNew := ps.dpsNew
				ps.dpsNew = false
				workersNew := ps.workersNew
				workersCopy := make(map[int]engine.WorkerStat, len(ps.workers))
				if workersNew {
					for k, v := range ps.workers {
						workersCopy[k] = v
					}
					ps.workersNew = false
				}
				toasts := ps.toasts
				ps.toasts = nil
				engineDone := ps.engineDone.Swap(false)
				ps.mu.Unlock()

				countNew := ps.countNew.Swap(false)
				count := ps.count.Load()

				// Nothing to flush? Skip fyne.Do entirely.
				if digits == "" && !countNew && !dpsNew && !workersNew && len(toasts) == 0 && !engineDone {
					continue
				}

				fyne.Do(func() {
					if digits != "" {
						digitView.AppendDigits(digits)
						stats.Histogram.AddDigits(digits)
						// Only update search buffer when there are new digits —
						// SetBuffer is cheap (pointer swap) so this is fine.
						search.SetBuffer(digitView.Buffer())
					}
					if countNew {
						stats.UpdateCount(count)
						for _, m := range milestones {
							if !firedMilestones[m] && count >= m {
								firedMilestones[m] = true
								msg := fmt.Sprintf("🎉 %s Digits!", ui.FormatInt(m))
								toast.Show(msg)
								sessionLog.Append("Milestone: " + msg)
							}
						}
					}
					if dpsNew {
						stats.UpdateDPS(dps)
					}
					if workersNew {
						for _, s := range workersCopy {
							stats.UpdateWorker(s)
						}
					}
					for _, msg := range toasts {
						toast.Show(msg)
						sessionLog.Append("Milestone: " + msg)
					}
					if engineDone {
						controls.SetRunning(false)
						stats.StopTimer()
					}
				})
			}
		}
	}()

	// ── controls.OnStart ──────────────────────────────────────────────
	controls.OnStart = func(cfg engine.Config) {
		digitView.Reset()
		stats.ResetTimer()
		stats.Sparkline.Reset()
		stats.Histogram.Reset()

		eng.Start(cfg)
		controls.SetRunning(true)
		stats.StartTimer()

		workers := 1
		if cfg.ThreadMult > 0 {
			workers = cfg.ThreadMult
		}
		sessionLog.Append(fmt.Sprintf("Started — %d worker(s) — file output: %v", workers, cfg.WriteToFile))

		// Fan-out: DigitCh — writes ONLY to ps, never calls fyne.Do directly.
		go func() {
			lastSampleTime := time.Now()
			var accLen int
			for digits := range eng.DigitCh {
				ps.mu.Lock()
				ps.digitBuf = append(ps.digitBuf, digits...)
				ps.mu.Unlock()

				// Accumulate digit count for sparkline sample (1s window).
				accLen += len(digits)
				now := time.Now()
				if dt := now.Sub(lastSampleTime).Seconds(); dt >= 1.0 {
					dps := float64(accLen) / dt
					fyne.Do(func() { stats.Sparkline.Sample(dps) })
					accLen = 0
					lastSampleTime = now
				}
			}
		}()

		// Fan-out: CountCh — writes ONLY to ps.
		go func() {
			var prevCount int64
			prevTime := time.Now()
			for count := range eng.CountCh {
				ps.count.Store(count)
				ps.countNew.Store(true)

				now := time.Now()
				dt := now.Sub(prevTime).Seconds()
				if dt > 0 {
					ps.mu.Lock()
					ps.dps = float64(count-prevCount) / dt
					ps.dpsNew = true
					ps.mu.Unlock()
				}
				prevCount = count
				prevTime = now
			}
			// Engine stopped — signal flush ticker.
			ps.engineDone.Store(true)
		}()

		// Fan-out: WorkerStatCh — coalesce: latest stat per worker wins.
		go func() {
			for stat := range eng.WorkerStatCh {
				ps.mu.Lock()
				ps.workers[stat.WorkerID] = stat
				ps.workersNew = true
				ps.mu.Unlock()
			}
		}()
	}

	// ── controls.OnStop ───────────────────────────────────────────────
	controls.OnStop = func() {
		eng.Stop()
		controls.SetRunning(false)
		stats.StopTimer()
		sessionLog.Append("Stopped")
	}

	// ── controls.OnResume ─────────────────────────────────────────────
	controls.OnResume = func(path string) {
		count, err := eng.Resume(path)
		if err != nil {
			sessionLog.Append(fmt.Sprintf("Resume error: %v", err))
			return
		}
		sessionLog.Append(fmt.Sprintf("Resumed from %s — %s digits counted", path, ui.FormatInt(count)))
		if controls.OnStart != nil {
			controls.OnStart(controls.BuildConfig())
		}
	}

	// ── controls.OnSnapshot ───────────────────────────────────────────
	controls.OnSnapshot = func() {
		buf := digitView.Buffer()
		dialog.NewFileSave(func(uwc fyne.URIWriteCloser, err error) {
			if err != nil || uwc == nil {
				return
			}
			defer uwc.Close()
			if _, werr := uwc.Write(buf); werr != nil {
				sessionLog.Append(fmt.Sprintf("Snapshot write error: %v", werr))
				return
			}
			sessionLog.Append(fmt.Sprintf("Snapshot saved — %d bytes", len(buf)))
		}, w).Show()
	}

	// Stop the flush ticker when the window closes.
	_ = flushStop // closed via window OnClosed wired in app.go

	w.ShowAndRun()
	close(flushStop)
}
