package main

import (
	"fmt"
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

	// Track fired milestones.
	firedMilestones := make(map[int64]bool)

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

		// Fan-out: DigitCh
		go func() {
			lastTime := time.Now()
			var lastLen int
			for digits := range eng.DigitCh {
				d := digits
				fyne.Do(func() {
					digitView.AppendDigits(d)
					stats.Histogram.AddDigits(d)
					search.SetBuffer(digitView.Buffer())
				})
				// Sample sparkline ~every 1s.
				now := time.Now()
				if dt := now.Sub(lastTime).Seconds(); dt >= 1.0 {
					dps := float64(len(d)+lastLen) / dt
					fyne.Do(func() { stats.Sparkline.Sample(dps) })
					lastLen = 0
					lastTime = now
				} else {
					lastLen += len(d)
				}
			}
		}()

		// Fan-out: CountCh
		go func() {
			var prevCount int64
			prevTime := time.Now()
			for count := range eng.CountCh {
				c := count
				firedLocal := firedMilestones
				fyne.Do(func() {
					stats.UpdateCount(c)
					for _, m := range milestones {
						if !firedLocal[m] && c >= m {
							firedLocal[m] = true
							msg := fmt.Sprintf("🎉 %s Digits!", ui.FormatInt(m))
							toast.Show(msg)
							sessionLog.Append("Milestone: " + msg)
						}
					}
				})
				// Update speed gauge outside fyne.Do (pure math).
				now := time.Now()
				dt := now.Sub(prevTime).Seconds()
				if dt > 0 {
					dps := float64(c-prevCount) / dt
					fyne.Do(func() { stats.UpdateDPS(dps) })
				}
				prevCount = c
				prevTime = now
			}
			// Engine stopped — update UI.
			fyne.Do(func() {
				controls.SetRunning(false)
				stats.StopTimer()
			})
		}()

		// Fan-out: WorkerStatCh
		go func() {
			for stat := range eng.WorkerStatCh {
				s := stat
				fyne.Do(func() { stats.UpdateWorker(s) })
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

	w.ShowAndRun()
}
