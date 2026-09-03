# piKillView — Plan

> Graphical π computation viewer. Dark-themed, animated, multi-threaded, resumable.
> From the minds of IBM Bob & Daneyand.

---

## Top-Level Overview

**Goal:** Build `piKillView` — a standalone GUI application that wraps the piKill computation engine
in a beautiful, animated, dark-themed interface built with [Fyne](https://fyne.io) (Go-native GUI
framework). Single binary per platform, zero installer. Ships as v1.0 with all planned features
including everything originally scoped for v1.1.

**Approach:** New Go module `github.com/007Style/piKillView`. The Chudnovsky engine is copied from
piKill (not imported) so piKillView is fully self-contained. The engine lives in `pkg/engine`, the
GUI in `pkg/ui`. Cross-compilation is handled by GitHub Actions (native runners per platform),
triggered on tag push, auto-creates the GitHub release with all 5 binaries attached.

**Tech stack:**
- GUI: `fyne.io/fyne/v2` — dark theme, cross-platform, CGO, single binary
- CPU monitoring: `github.com/shirou/gopsutil/v3/cpu`
- Image/animation: Go stdlib `image/draw`, `image/color`
- Search: Boyer-Moore-Horspool in a background goroutine
- Sparkline / charts: custom `canvas.Raster` widgets
- Cross-compile: GitHub Actions matrix (macos-arm64, macos-amd64, linux-amd64, linux-arm64, windows-amd64)

**Platforms:** macOS arm64/amd64 · Linux amd64/arm64 · Windows amd64

---

## Directory Structure

```
piKillView/
├── main.go
├── go.mod
├── go.sum
├── Makefile                        ← local build (current platform) + fyne-cross notes
├── README.md
├── .github/
│   └── workflows/
│       └── release.yml             ← GitHub Actions cross-compile + release
├── assets/
│   └── icon.png                    ← π symbol app icon
└── pkg/
    ├── engine/
    │   ├── engine.go               ← Engine struct, Start/Stop/Resume, channels
    │   ├── chudnovsky.go           ← Chudnovsky algorithm (from piKill)
    │   ├── worker.go               ← Work-stealing pool, per-worker stats
    │   └── engine_test.go
    └── ui/
        ├── app.go                  ← Root window, layout, theme toggle
        ├── header.go               ← Digit rain canvas + pulsing π animation
        ├── controls.go             ← Thread slider, Start/Stop, file picker, Resume
        ├── digitview.go            ← Ring buffer viewer, color-coded, range display
        ├── stats.go                ← Digit counter, elapsed, round, speed gauge, workers
        ├── histogram.go            ← Live 0–9 digit frequency bar chart
        ├── sparkline.go            ← 60s rolling digits/sec sparkline
        ├── cpudials.go             ← Per-core CPU arc gauges
        ├── search.go               ← Boyer-Moore-Horspool search + highlight
        ├── trivia.go               ← π trivia scrolling ticker
        ├── sessionlog.go           ← Append-only session event log
        ├── toasts.go               ← Milestone toast notifications
        └── theme.go                ← Dark/light theme definitions + toggle
```

---

## Data Flow

```
engine.go
  └── chudnovsky.go  (work-stealing workers)
        ├── WorkerStatCh  chan WorkerStat   → ui/stats.go    (pulse dots, terms/worker)
        ├── DigitCh       chan string        → ui/digitview.go (ring buffer append)
        │                                   → ui/histogram.go (frequency update)
        │                                   → ui/sparkline.go (throughput sample)
        │                                   → ui/search.go   (background scan)
        └── CountCh       chan int64         → ui/stats.go    (digit counter, milestones)

gopsutil ticker (500ms) → ui/cpudials.go
time.Ticker  (100ms)    → ui/stats.go    (elapsed timer)
time.Ticker  (1s)       → ui/sparkline.go (digits/sec sample)
time.Ticker  (33ms)     → ui/header.go   (rain animation)
trivia slice            → ui/trivia.go   (scrolling ticker)
```

---

## Sub-Tasks

---

### Sub-Task 1 — Project Scaffold + GitHub Actions

**Intent:** Create the Go module, directory structure, dependencies, local Makefile, and the
GitHub Actions release workflow that cross-compiles all 5 platforms and attaches binaries to the
release on tag push.

**Expected Outcomes:**
- `piKillView/go.mod` with module `github.com/007Style/piKillView`, Go 1.21+
- Fyne v2, gopsutil v3 added as dependencies (`go get`)
- Stub `main.go` opens a blank dark Fyne window — `go build ./...` passes
- `Makefile` with `build` (current platform), `clean`, version `1.0`
- `.github/workflows/release.yml`:
  - Triggers on `push: tags: ['v*']`
  - Matrix: `[macos-arm64, macos-amd64, linux-amd64, linux-arm64, windows-amd64]`
  - Each job: `actions/checkout`, `actions/setup-go`, install platform deps (libgl, xorg for Linux), `go build -ldflags "-s -w"`, upload artifact
  - Final job: `gh release create $TAG dist/*` using `GITHUB_TOKEN`
- `assets/icon.png` — simple π symbol on dark background (generated programmatically)

**Todo List:**
- [ ] `mkdir -p piKillView/pkg/engine piKillView/pkg/ui piKillView/.github/workflows piKillView/assets`
- [ ] Write `go.mod`: module + go version
- [ ] `go get fyne.io/fyne/v2@latest && go get github.com/shirou/gopsutil/v3@latest`
- [ ] Write stub `main.go` (blank dark Fyne window, title "piKillView v1.0")
- [ ] Write `Makefile`
- [ ] Write `.github/workflows/release.yml` with full matrix
- [ ] Generate `assets/icon.png` programmatically in Go (draw π on dark bg, save PNG)
- [ ] Verify `go build ./...` passes locally

**Relevant Context:**
- Fyne Linux cross-compile needs: `libgl1-mesa-dev xorg-dev` (Ubuntu runner)
- Windows cross-compile needs: `gcc-mingw-w64` on Ubuntu runner with `GOOS=windows CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc`
- macOS arm64: `runs-on: macos-14` (Apple Silicon runner); amd64: `runs-on: macos-13`
- Fyne app icon: `app.SetIcon(theme.FyneLogo())` as placeholder until real icon exists

**Status:** [x] done

---

### Sub-Task 2 — Engine Package

**Intent:** Extract and refactor the Chudnovsky computation engine from piKill into `pkg/engine`
with a clean API the UI can drive. Engine emits three channels the UI reads independently.

**Expected Outcomes:**
- `Engine` struct with `Start(cfg Config)`, `Stop()`, `Resume(filePath string)` methods
- `Config` struct: `ThreadMult int` (0=single, 1–4=N×CPU), `WriteToFile bool`, `OutputDir string`
- Three output channels: `DigitCh chan string`, `CountCh chan int64`, `WorkerStatCh chan WorkerStat`
- `WorkerStat`: `WorkerID int`, `TermsComputed int64`, `Active bool`
- Resume: reads file, counts valid digit chars after `3.`, sets `startDigits` to resume from that count
- All channels closed cleanly on `Stop()`
- `engine_test.go`: first 50 digits match known π, both single and multi-worker

**Todo List:**
- [ ] Write `pkg/engine/chudnovsky.go`: export `ComputePi(prec uint, workers int, statCh chan WorkerStat) *big.Float`
- [ ] Write `pkg/engine/worker.go`: work-stealing atomic pool; emit `WorkerStat{WorkerID, TermsComputed, Active}` per term claimed and on completion
- [ ] Write `pkg/engine/engine.go`: `Config`, `Engine`, `Start`, `Stop`, `Resume`
- [ ] Resume: `bufio.Scanner` reads file, counts rune length after `3.`, sets startDigits
- [ ] Engine loop: doubling strategy (10k→20k→…), safetyMargin=20, emit chunks to `DigitCh`, running total to `CountCh`
- [ ] File writing: `bufio.NewWriterSize(f, 4MiB)`, output to `Config.OutputDir/pi-<datetime>.txt`
- [ ] `Stop()`: cancel context, drain and close all channels, flush file buffer
- [ ] Write `pkg/engine/engine_test.go`

**Relevant Context:**
- piKill constants: `startDigits=10_000`, `safetyMargin=20`, `pipelineDepth=4`, `writeBufferSize=4MiB`
- piKill work-stealing: `var nextTerm atomic.Int64` + `sync.WaitGroup`
- Use `context.WithCancel` for clean stop propagation into worker goroutines
- Module: `github.com/007Style/piKillView/pkg/engine`

**Status:** [x] done

---

### Sub-Task 3 — Theme + App Shell

**Intent:** Build the root Fyne window with dark/light theme toggle, full layout skeleton, and all
named placeholder containers that later sub-tasks will populate.

**Expected Outcomes:**
- App launches with dark theme, title `piKillView v1.0 — From the minds of IBM Bob & Daneyand`
- Window resizable, minimum size 1200×750
- Layout regions (all named, all populated with colored placeholder):
  - **Header** (top, ~130px): animation canvas
  - **Controls** (left sidebar, ~300px): thread/start/stop/file controls
  - **Digit View** (center, grows): ring buffer display
  - **Stats** (right sidebar, ~220px): counters, worker dots, speed, CPU dials
  - **Trivia ticker** (bottom strip, ~32px): scrolling π facts
  - **Status bar** (very bottom, ~24px): session log toggle, keyboard shortcut hints
- Dark/light toggle button in top-right corner
- App icon set from `assets/icon.png`
- Window close handler: `engine.Stop()` before exit

**Todo List:**
- [ ] Write `pkg/ui/theme.go`: `DarkTheme()` and `LightTheme()` wrappers, `ToggleTheme(app)` function
- [ ] Write `pkg/ui/app.go`: `NewMainWindow(app, engine)` returns `fyne.Window`
- [ ] Define layout with `container.NewBorder` + `container.NewHSplit` + `container.NewVSplit`
- [ ] Create stub containers for each region (colored `canvas.Rectangle` backgrounds)
- [ ] Wire theme toggle button (`widget.Button` with moon/sun icon)
- [ ] Set window `OnClosed` callback
- [ ] Keyboard shortcut registration (Space=start/stop, Cmd+F=search, Cmd+S=snapshot) via `fyne.KeyboardShortcut`

**Relevant Context:**
- `app.Settings().SetTheme(myTheme)` — called from toggle
- `container.NewBorder(top, bottom, left, right, center)` for outer frame
- Fyne keyboard shortcuts: `canvas.AddShortcut(shortcut, handler)`

**Status:** [ ] pending

---

### Sub-Task 4 — Animated Header

**Intent:** Build the header widget with a digit-rain background canvas and a pulsing π symbol.

**Expected Outcomes:**
- Digit rain: columns of π digits falling at varying speeds/opacities, green-tinted on dark bg
- Rain uses the first 1000 known digits of π as its character set (cycles through them)
- π symbol: large (~80pt), centered, pulses in size and alpha via sine wave, ~30fps
- Animation runs on a background ticker goroutine; pauses cleanly when widget hidden
- Rain columns respawn at top when they reach bottom, random speed/brightness per column

**Todo List:**
- [ ] Write `pkg/ui/header.go`: `HeaderWidget` as `widget.BaseWidget` with `CreateRenderer()`
- [ ] `canvas.NewRaster(drawFn)`: `drawFn` renders rain into `image.RGBA` each frame
- [ ] `rainColumn` struct: `x, y float64`, `speed float64`, `chars []rune`, `alpha float64`, `headAlpha float64`
- [ ] Initialize N columns on `MinSize()` callback (N = width/18)
- [ ] Per-frame update: advance each column's `y`; wrap to top when `y > height`; vary `headAlpha=1.0`, trailing chars fade exponentially
- [ ] π pulse: `fyne.NewAnimation(2*time.Second, func(t float32))` driving a `canvas.Text` size and color
- [ ] `time.NewTicker(33ms)` calls `canvas.Refresh(raster)` each frame
- [ ] `Stop()` method cancels ticker goroutine

**Relevant Context:**
- Known π digits constant: embed first 1000 digits as `const piDigits = "14159265358979..."` (no leading 3)
- `image.RGBA` pixel math for anti-aliased text: use `golang.org/x/image/font` + `golang.org/x/image/font/basicfont`
- Rain head pixel: bright green `#00FF41`; trail: `#003B00` fading to black
- The `golang.org/x/image` package is a required dep for bitmap font rendering

**Status:** [ ] pending

---

### Sub-Task 5 — Controls Panel

**Intent:** Left sidebar: thread slider (5 positions), Start/Stop, file output toggle + dir picker,
Resume from file, precision lock, auto-save interval.

**Expected Outcomes:**
- Thread slider: 0–4 snapping, label shows "Single · 1 worker", "1× · 12 workers", etc. (runtime CPU count)
- Start (green) / Stop (red) buttons; mutually enabled/disabled
- File output: `widget.Check` reveals dir path entry + folder picker when checked
- Resume button: file picker → calls `engine.Resume(path)`; only active when file output checked
- Precision lock: `widget.Check` + `widget.Entry` for digit target; engine respects it
- Auto-save: `widget.Check` + `widget.Select` (every 10s / 30s / 60s / 5min)
- Export snapshot: `widget.Button` "Save snapshot" → writes current ring buffer to a file

**Todo List:**
- [ ] Write `pkg/ui/controls.go`: `ControlsPanel` struct
- [ ] Thread slider: `widget.NewSlider(0, 4)`, `OnChanged` updates label + stores in config
- [ ] Start/Stop buttons with color theming via `widget.NewButton` + button importance
- [ ] File output check + dir entry + `dialog.NewFolderOpen`
- [ ] Resume button + `dialog.NewFileOpen` with `.txt` filter; parse file, call `engine.Resume`
- [ ] Precision lock: check + entry (numeric validation); passed to engine `Config.DigitLimit int64`
- [ ] Auto-save: check + select; passed to engine `Config.AutoSaveInterval time.Duration`
- [ ] Snapshot: write `digitView.Buffer()` to user-chosen file via `dialog.NewFileSave`
- [ ] `SetRunning(bool)` method to toggle button states from engine callbacks

**Relevant Context:**
- `dialog.NewFolderOpen(func(fyne.ListableURI, error), fyne.Window)`
- `dialog.NewFileOpen(func(fyne.URIReadCloser, error), fyne.Window)`
- `dialog.NewFileSave(func(fyne.URIWriteCloser, error), fyne.Window)`
- Numeric entry validation: `widget.Entry` with `Validator: validation.NewRegexp("^[0-9]*$", "digits only")`

**Status:** [ ] pending

---

### Sub-Task 6 — Digit Display (Ring Buffer Viewer)

**Intent:** Central read-only π digit display. Color-coded by digit value. 10MB ring buffer.
Scroll range display. Selectable for copy. Background search with highlight.

**Expected Outcomes:**
- Monospace display, each digit colored (0=red, 1=orange, 2=yellow, 3=lime, 4=cyan, 5=blue, 6=indigo, 7=violet, 8=pink, 9=white)
- Buffer capped at 10MB; `bufferOffset int64` tracks absolute start digit
- Scroll position shows: "Viewing digits 3,241,000 – 3,251,000" in status strip below
- Text selectable, not editable; Cmd+C copies selection
- Auto-scrolls to bottom; pauses when user scrolls up; resumes when scrolled back to bottom
- `Reset()` for Stop/Resume
- `Buffer() []byte` for snapshot export
- Search highlights: matched regions rendered in bright yellow background

**Todo List:**
- [ ] Write `pkg/ui/digitview.go`: `DigitView` struct
- [ ] Ring buffer: `buf []byte` max 10MB; `bufferOffset int64`; `AppendDigits(s string)`
- [ ] Color map: `var digitColor = [10]color.RGBA{...}` — one color per digit 0–9
- [ ] Use `widget.RichText` with `[]widget.RichTextSegment` — one `widget.TextSegment` per digit with its color; batch updates for performance (group same-colored runs into segments)
- [ ] Range label: `widget.Label` below viewer, updated on scroll and append
- [ ] Auto-scroll: `widget.List` scroll position tracking or manual `Scrolled()` override
- [ ] `AppendDigits`: trim buffer from front if over 10MB, update `bufferOffset`, rebuild `RichText` segments for new chunk only (append, don't rebuild all)
- [ ] Search highlight: `SetHighlight(start, length int64)` recolors matched region yellow

**Relevant Context:**
- `widget.RichText` accepts `[]widget.RichTextSegment`; `widget.TextSegment{Style: widget.RichTextStyle{ColorName: ...}, Text: "..."}`
- Rebuilding all 10M segments on every append is too slow — only append new segments, trim from front by index
- `fyne.Theme` color names vs `color.RGBA` — use `color.RGBA` directly in `RichTextStyle{Color: c}`
- 10MB = ~10M digits; at ~50 digits per segment (same color run) = ~200K segments max — acceptable

**Status:** [ ] pending

---

### Sub-Task 7 — Stats Panel

**Intent:** Right sidebar: digit counter, elapsed timer, round indicator, estimated time to next
round, speed gauge, memory usage, per-worker pulse dots, digits/sec sparkline.

**Expected Outcomes:**
- Digit counter: large bold, comma-formatted, updates from `CountCh`
- Elapsed timer: `HH:MM:SS.mmm`, 100ms tick
- Round indicator: "Round 7 — 640,000 digits" label
- Estimated next round: "Next round in ~42s" (based on last round duration)
- Speed gauge: needle arc dial showing current digits/sec (0 to max seen)
- Memory usage: `runtime.ReadMemStats().HeapAlloc` formatted as MB, updates every 2s
- Per-worker rows: worker ID + green/grey pulse dot + terms count; scrollable `widget.List` for up to 48 workers
- Digits/sec sparkline: 60-point rolling chart, 1s samples, custom `canvas.Raster`

**Todo List:**
- [ ] Write `pkg/ui/stats.go`: `StatsPanel` struct, `Update*` methods
- [ ] Digit counter + elapsed + round + eta: four `widget.Label`s, goroutines feeding from channels/tickers
- [ ] Speed gauge: custom `canvas.NewRaster` arc dial, same pattern as CPU dials
- [ ] Memory: `time.NewTicker(2s)` + `runtime.ReadMemStats`
- [ ] Worker rows: `widget.List` with `Length()` and `UpdateItem()` callbacks; `[]WorkerRow` backing slice
- [ ] `WorkerRow`: `id int`, `terms int64`, `active bool`; goroutine reads `WorkerStatCh`, updates slice, calls `List.Refresh()`
- [ ] Write `pkg/ui/sparkline.go`: `SparklineWidget` — `canvas.NewRaster`, 60-float64 ring, draws filled area chart
- [ ] Wire ETA: record `roundStart time.Time` each new round; on completion compute `lastRoundDur`; display `lastRoundDur * 2` as estimate (each round doubles)

**Relevant Context:**
- `runtime.MemStats.HeapAlloc` is current heap in bytes
- `canvas.NewCircle` for worker pulse dots; toggle `FillColor` between `color.RGBA{0,255,100,255}` and `color.RGBA{60,60,60,255}`
- `fyne.Do(func())` required for all UI updates from goroutines (Fyne v2.5+)
- Worker count up to 48 — `widget.List` is virtualised, handles this fine

**Status:** [ ] pending

---

### Sub-Task 8 — Digit Frequency Histogram

**Intent:** Live bar chart showing how often each digit 0–9 appears in computed π. Should converge
toward uniform 10% distribution — watching it balance is mesmerizing.

**Expected Outcomes:**
- 10 vertical bars (0–9), each labeled with digit and percentage
- Bars colored with the same color map as the digit display (0=red … 9=white)
- Updates on each `DigitCh` receive (incremental counter, not full recount)
- Percentage label above each bar; "Expected: 10.0%" reference line drawn across all bars
- Renders as custom `canvas.Raster` for smooth animation
- Placed in the stats panel below the sparkline

**Todo List:**
- [ ] Write `pkg/ui/histogram.go`: `HistogramWidget` as `widget.BaseWidget`
- [ ] State: `counts [10]int64`, `total int64`; `AddDigits(s string)` method increments counters
- [ ] `CreateRenderer()` returns renderer with `canvas.NewRaster(drawFn)`
- [ ] `drawFn`: compute bar heights from percentages, draw colored filled rects, draw 10% reference line, draw labels
- [ ] `Refresh()` called from `AppendDigits` after counter update
- [ ] `Reset()` zeros all counters

**Relevant Context:**
- Same `digitColor` map as `digitview.go` — define it in a shared `pkg/ui/colors.go` file
- Bar chart math: `barHeight = int(float64(imgH-labelSpace) * float64(counts[d]) / float64(total))`
- Reference line at `y = imgH - labelSpace - int(0.10 * float64(imgH-labelSpace))`

**Status:** [ ] pending

---

### Sub-Task 9 — CPU Dials

**Intent:** Arc gauge per logical CPU showing live utilisation. Color transitions green→yellow→red.

**Expected Outcomes:**
- One arc dial per logical CPU; if `NumCPU > 12`, show 12 dials + one aggregate
- Each dial: semicircular arc (0°–180°), fill sweeps from left, labeled "CPU N" + "XX%"
- Color: green (0–60%), interpolates to yellow (60–80%), red (80–100%)
- 500ms refresh via `gopsutil/v3/cpu.Percent(0, true)`
- All dials in a wrapping grid in the stats panel

**Todo List:**
- [ ] Write `pkg/ui/cpudials.go`: `CPUDialsWidget`
- [ ] Single `canvas.NewRaster` drawing all dials into one `image.RGBA`
- [ ] `gopsutil` goroutine: `time.NewTicker(500ms)`, updates `[]float64` slice, calls `Refresh()`
- [ ] Arc drawing: pixel math for arc sweep angle `= percent/100 * π radians`, center at dial midpoint
- [ ] Color lerp: `lerpColor(green, yellow, t)` for 0–0.8, `lerpColor(yellow, red, t)` for 0.8–1.0 where t=percent/100
- [ ] Aggregate dial: average of all CPUs, shown last if `NumCPU > 12`

**Relevant Context:**
- Arc pixel math: `x = cx + r*cos(angle)`, `y = cy + r*sin(angle)`; fill by checking if pixel angle < sweep
- `gopsutil/v3/cpu` — `Percent(interval, percpu bool) ([]float64, error)`: pass `interval=0` for non-blocking (returns since last call)
- Colors: green `#00C853`, yellow `#FFD600`, red `#D50000`

**Status:** [ ] pending

---

### Sub-Task 10 — Search + Find My Birthday

**Intent:** Ctrl+F / Cmd+F opens a search bar. Boyer-Moore-Horspool search runs in a background
goroutine. Matches highlighted in the digit view. Includes "Find my birthday" shortcut (MMDDYYYY).

**Expected Outcomes:**
- Search bar appears at top of digit view on Cmd+F; dismisses on Escape
- Input: any digit string; "Find" button triggers search
- Search runs in background goroutine, doesn't block UI
- First match highlighted in digit view (scrolls to match); "N matches found" label
- "Next" / "Previous" buttons cycle through matches
- "Find my birthday" preset: `widget.Entry` pre-populated with today as `MMDDYYYY`; user edits before searching
- Search only covers current ring buffer (not the full computed output — that could be on disk)

**Todo List:**
- [ ] Write `pkg/ui/search.go`: `SearchBar` widget + `SearchEngine` struct
- [ ] `SearchBar`: `widget.Entry` + Find/Next/Prev/Close buttons; shown/hidden via `search.Show()/Hide()`
- [ ] `SearchEngine.Search(pattern string, buf []byte, resultCh chan []int)`: Boyer-Moore-Horspool over `buf`
- [ ] On result: call `digitView.SetHighlight(matchStart, len(pattern))`, scroll to match
- [ ] "Find my birthday": `widget.Button` pre-fills entry with `time.Now().Format("01022006")`
- [ ] Result cycling: `currentMatch int`, Next/Prev increment/decrement, re-highlight + scroll
- [ ] Wire Cmd+F shortcut from app shell to `search.Show()`

**Relevant Context:**
- Boyer-Moore-Horspool: O(n/m) average case; for 10MB buffer and 8-char pattern, completes in <10ms
- Run in `go func()` with `resultCh chan []int`; UI reads from channel via `fyne.Do`
- `digitView.Buffer()` returns a copy of the current ring buffer for search to operate on

**Status:** [ ] pending

---

### Sub-Task 11 — Toasts + Milestones

**Intent:** When digit count crosses milestone values, display a celebratory toast overlay.

**Expected Outcomes:**
- Milestones: 1K, 10K, 100K, 1M, 10M, 100M, 1B digits
- Toast: semi-transparent overlay label slides in from top, holds 2.5s, fades out
- Message format: "🎉 1,000,000 Digits!" in large bold text
- Does not interrupt computation or block UI
- Only fires once per milestone per session (not on resume past a milestone)

**Todo List:**
- [ ] Write `pkg/ui/toasts.go`: `ToastOverlay` widget
- [ ] `ToastOverlay` sits as top-most layer in the center container (zero size when hidden)
- [ ] `Show(message string)`: sets text, animates alpha 0→1 (200ms), holds, animates 1→0 (300ms), hides
- [ ] Milestone checker: in stats goroutine reading `CountCh`, check if new count crosses any milestone; call `toast.Show()`
- [ ] Track fired milestones in `map[int64]bool` to avoid repeat
- [ ] `fyne.NewAnimation` for fade in/out

**Relevant Context:**
- `fyne.NewAnimation(duration, func(float32))` — `float32` is 0.0→1.0 progress
- `canvas.Text.Color` alpha channel for fade
- Overlay: `container.NewStack` with digit view + toast on top

**Status:** [ ] pending

---

### Sub-Task 12 — π Trivia Ticker

**Intent:** Scrolling ticker along the bottom of the window cycling through π facts.

**Expected Outcomes:**
- ~40 π trivia facts hardcoded as a string slice
- Ticker scrolls right-to-left continuously, ~80px/sec
- New fact queued as previous one exits the left edge
- Subtle separator between facts (· or —)
- Pauses on mouse hover (polite)

**Todo List:**
- [ ] Write `pkg/ui/trivia.go`: `TriviaWidget` as `widget.BaseWidget` with `canvas.NewRaster`
- [ ] Hardcode `var triviaFacts []string` of ~40 facts
- [ ] State: `xOffset float64` (scroll position), current + next fact string
- [ ] `time.NewTicker(33ms)`: advance `xOffset` by `speed * dt`; when current fact fully exits, advance to next
- [ ] `drawFn`: render current + next fact at their respective x positions using `basicfont` bitmap rendering
- [ ] Mouse hover: `widget.BaseWidget` `MouseIn()`/`MouseOut()` pause/resume ticker
- [ ] Trivia facts to include: NASA 15 digits, 202T record, Buffon's needle, π in the Bible, Feynman point, first trillion digit date, etc.

**Relevant Context:**
- `golang.org/x/image/font/basicfont` for bitmap text in raster canvas
- Same ticker goroutine pattern as header rain
- Ticker strip height: ~32px, full window width

**Status:** [ ] pending

---

### Sub-Task 13 — Session Log

**Intent:** Collapsible panel logging key events with timestamps.

**Expected Outcomes:**
- Events logged: Started, Stopped, Resumed from file, Milestone reached, File write errors, Snapshot saved
- Format: `[14:32:01.423] Started — 12 workers — output: pi-2026-09-03T14-32-01.txt`
- `widget.List` backed by `[]LogEntry` slice, newest at bottom
- Collapsible: a chevron button in the status bar toggles the log panel height
- Log entries copy-able (select + Cmd+C)

**Todo List:**
- [ ] Write `pkg/ui/sessionlog.go`: `SessionLog` struct
- [ ] `LogEntry`: `Timestamp time.Time`, `Message string`
- [ ] `Append(msg string)`: prepend timestamp, append to slice, call `List.Refresh()`
- [ ] `widget.List` with monospace font for entries
- [ ] Collapse/expand: animate height between 0 and 150px via `fyne.NewAnimation`
- [ ] Expose `Log` function globally (or via dependency injection) so engine and all UI components can log events

**Relevant Context:**
- `widget.List` is virtualised — handles thousands of entries efficiently
- Monospace: `fyne.TextStyle{Monospace: true}` in list item renderer
- Log should be written to in-memory only (not persisted to disk unless user exports)

**Status:** [ ] pending

---

### Sub-Task 14 — Wire Everything + Final Polish

**Intent:** Connect all UI components to the engine. Verify no goroutine leaks. End-to-end test.
Add all keyboard shortcuts. Final visual polish pass.

**Expected Outcomes:**
- Start: engine launches, all panels animate, timer runs, workers pulse
- Stop: everything halts cleanly, timer stops, worker dots grey out
- File output: digits written to chosen dir; auto-save fires if configured
- Precision lock: engine stops at target digit count, shows "Target reached" toast
- Resume: file parsed, digit view starts at correct absolute offset, session log records event
- Search: Cmd+F opens bar, finds match, highlights, Next/Prev cycles
- Theme toggle: flips all panels seamlessly
- Window close: engine stopped, file flushed, no hanging goroutines
- Milestone toasts fire at correct counts
- Trivia ticker scrolls, pauses on hover
- All keyboard shortcuts work: Space, Cmd+F, Cmd+S, Escape (close search)

**Todo List:**
- [ ] Write final `main.go`: construct all components, wire callbacks, start app
- [ ] Fan-out goroutine: reads `DigitCh` → `digitView.AppendDigits` + `histogram.AddDigits` + `sparkline.Sample` + `search engine buffer update`
- [ ] Fan-out goroutine: reads `CountCh` → `stats.UpdateCount` + milestone checker + precision lock check
- [ ] Fan-out goroutine: reads `WorkerStatCh` → `stats.UpdateWorker`
- [ ] All UI updates wrapped in `fyne.Do(func())`
- [ ] Verify engine `Stop()` closes all channels → all goroutines exit
- [ ] Visual polish: padding, spacing, font sizes, consistent color use across all panels
- [ ] Smoke test: run 30 seconds with `-tttt`, verify digit count > 0, file written, stop clean

**Relevant Context:**
- `fyne.Do` (v2.5+) is the safe way to update UI from goroutines
- Channel fan-out: read once from source channel, write to multiple internal channels or call methods directly under `fyne.Do`
- `context.WithCancel` passed to engine; cancel on window close

**Status:** [ ] pending

---

### Sub-Task 15 — README + Release

**Intent:** Write the full README with development story, all features documented, screenshots
section. Create GitHub repo, push, tag v1.0, GitHub Actions auto-builds and attaches all 5 binaries.

**Expected Outcomes:**
- `README.md`: tagline, feature list, screenshot placeholders, usage, architecture, building from source, v1.1 roadmap
- GitHub repo `007Style/piKillView` created and pushed
- `v1.0` tag pushed → GitHub Actions matrix runs → 5 binaries attached to release
- Release notes include full feature list and download table

**Todo List:**
- [ ] Write `README.md`
- [ ] `gh repo create piKillView --public --source=. --remote=origin`
- [ ] `git tag v1.0 && git push origin main --tags`
- [ ] Monitor GitHub Actions run via `gh run watch`
- [ ] Verify release artifacts at `https://github.com/007Style/piKillView/releases/tag/v1.0`

**Relevant Context:**
- README should reference piKill as the spiritual predecessor
- Screenshot section: use placeholder text "*(screenshot)*" — actual screenshots added after first run
- GitHub Actions workflow file already written in Sub-Task 1

**Status:** [ ] pending

---

## Open Decisions

| Question | Decision |
|---|---|
| GUI framework | Fyne v2 |
| Cross-compile | GitHub Actions (native runners per platform) |
| Resume precision | Best effort — count digits in file, restart at that precision |
| Display buffer | 10MB ring buffer, trim oldest, track absolute offset |
| Digit cursor | Range display "Viewing digits X–Y" (not per-character) |
| Header animation | Digit rain + pulsing π at 30fps |
| CPU monitoring | gopsutil v3, arc dials, 500ms |
| Search algorithm | Boyer-Moore-Horspool in background goroutine |
| v1.1 features | All included in this build (single release) |
| File output dir | User-specified via folder picker |
| Auto-save | Optional, user-configurable interval |
| Precision lock | Optional digit target, engine stops when reached |
| Color coding | 10 distinct colors, one per digit 0–9, shared color map |
| Milestones | 1K · 10K · 100K · 1M · 10M · 100M · 1B |
| Trivia facts | ~40 hardcoded π facts, scrolling ticker |
| Session log | In-memory, collapsible, exportable |
| Snapshot export | Writes current ring buffer to user-chosen file |
| Memory indicator | `runtime.ReadMemStats().HeapAlloc`, 2s refresh |
| "Find birthday" | Pre-fills search with `MMDDYYYY` of today |
