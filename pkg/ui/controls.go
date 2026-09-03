package ui

import (
	"fmt"
	"runtime"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/007Style/piKillView/pkg/engine"
)

// ControlsPanel is the left-side control sidebar.
type ControlsPanel struct {
	win fyne.Window

	// Config state.
	threadMult  int
	writeToFile bool
	outputDir   string
	digitLimit  int64
	autoSaveInt time.Duration

	// Widgets.
	threadLabel *widget.Label
	slider      *widget.Slider
	startBtn    *widget.Button
	stopBtn     *widget.Button
	fileCheck   *widget.Check
	dirEntry    *widget.Entry
	dirRow      *fyne.Container
	resumeBtn   *widget.Button
	limitCheck  *widget.Check
	limitEntry  *widget.Entry
	autoCheck   *widget.Check
	autoSelect  *widget.Select
	snapshotBtn *widget.Button

	// Callbacks.
	OnStart    func(cfg engine.Config)
	OnStop     func()
	OnResume   func(path string)
	OnSnapshot func()
}

// NewControlsPanel creates the controls panel.
func NewControlsPanel(win fyne.Window) *ControlsPanel {
	cp := &ControlsPanel{win: win}

	// ── Thread slider ──
	cp.threadLabel = widget.NewLabel("Single · 1 worker")
	slider := widget.NewSlider(0, 4)
	slider.Step = 1
	slider.Value = 0
	slider.OnChanged = func(v float64) {
		cp.threadMult = int(v)
		cp.threadLabel.SetText(cp.workerLabel())
	}
	cp.slider = slider

	// ── Start / Stop ──
	cp.startBtn = widget.NewButton("▶  Start", func() {
		if cp.OnStart != nil {
			cp.OnStart(cp.buildConfig())
		}
	})
	cp.startBtn.Importance = widget.HighImportance

	cp.stopBtn = widget.NewButton("■  Stop", func() {
		if cp.OnStop != nil {
			cp.OnStop()
		}
	})
	cp.stopBtn.Importance = widget.DangerImportance
	cp.stopBtn.Disable()

	// ── File output ──
	cp.dirEntry = widget.NewEntry()
	cp.dirEntry.PlaceHolder = "Output directory…"
	cp.dirEntry.OnChanged = func(s string) { cp.outputDir = s }

	folderBtn := widget.NewButton("📁", func() {
		dialog.NewFolderOpen(func(lu fyne.ListableURI, err error) {
			if err != nil || lu == nil {
				return
			}
			cp.dirEntry.SetText(lu.Path())
			cp.outputDir = lu.Path()
		}, cp.win).Show()
	})

	cp.dirRow = container.NewBorder(nil, nil, nil, folderBtn, cp.dirEntry)
	cp.dirRow.Hide()

	cp.fileCheck = widget.NewCheck("Write output to file", func(b bool) {
		cp.writeToFile = b
		if b {
			cp.dirRow.Show()
		} else {
			cp.dirRow.Hide()
		}
	})

	// ── Resume ──
	cp.resumeBtn = widget.NewButton("Resume from file…", func() {
		dialog.NewFileOpen(func(urc fyne.URIReadCloser, err error) {
			if err != nil || urc == nil {
				return
			}
			urc.Close()
			if cp.OnResume != nil {
				cp.OnResume(urc.URI().Path())
			}
		}, cp.win).Show()
	})

	// ── Precision lock ──
	cp.limitEntry = widget.NewEntry()
	cp.limitEntry.PlaceHolder = "Target digit count"
	cp.limitEntry.OnChanged = func(s string) {
		v, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			cp.digitLimit = v
		} else {
			cp.digitLimit = 0
		}
	}
	cp.limitEntry.Hide()

	cp.limitCheck = widget.NewCheck("Precision lock", func(b bool) {
		if b {
			cp.limitEntry.Show()
		} else {
			cp.limitEntry.Hide()
			cp.digitLimit = 0
		}
	})

	// ── Auto-save ──
	cp.autoSelect = widget.NewSelect([]string{"10s", "30s", "1m", "5m"}, func(s string) {
		switch s {
		case "10s":
			cp.autoSaveInt = 10 * time.Second
		case "30s":
			cp.autoSaveInt = 30 * time.Second
		case "1m":
			cp.autoSaveInt = time.Minute
		case "5m":
			cp.autoSaveInt = 5 * time.Minute
		}
	})
	cp.autoSelect.SetSelected("30s")
	cp.autoSaveInt = 30 * time.Second
	cp.autoSelect.Hide()

	cp.autoCheck = widget.NewCheck("Auto-save interval", func(b bool) {
		if b {
			cp.autoSelect.Show()
		} else {
			cp.autoSelect.Hide()
			cp.autoSaveInt = 0
		}
	})

	// ── Snapshot ──
	cp.snapshotBtn = widget.NewButton("💾  Save snapshot", func() {
		if cp.OnSnapshot != nil {
			cp.OnSnapshot()
		}
	})

	return cp
}

// workerLabel returns the human-readable worker count description.
func (cp *ControlsPanel) workerLabel() string {
	ncpu := runtime.NumCPU()
	switch cp.threadMult {
	case 0:
		return "Single · 1 worker"
	case 1:
		return fmt.Sprintf("1× · %d workers", ncpu)
	case 2:
		return fmt.Sprintf("2× · %d workers", ncpu*2)
	case 3:
		return fmt.Sprintf("3× · %d workers", ncpu*3)
	case 4:
		return fmt.Sprintf("4× · %d workers", ncpu*4)
	default:
		return "Single · 1 worker"
	}
}

// BuildConfig assembles an engine.Config from current UI state.
// Exported so main.go can call it from OnResume.
func (cp *ControlsPanel) BuildConfig() engine.Config { return cp.buildConfig() }

// buildConfig assembles an engine.Config from current UI state.
func (cp *ControlsPanel) buildConfig() engine.Config {
	cfg := engine.Config{
		ThreadMult:       cp.threadMult,
		WriteToFile:      cp.writeToFile,
		OutputDir:        cp.outputDir,
		DigitLimit:       cp.digitLimit,
		AutoSaveInterval: cp.autoSaveInt,
	}
	if !cp.writeToFile {
		cfg.AutoSaveInterval = 0
	}
	return cfg
}

// SetRunning toggles buttons enabled/disabled based on running state.
func (cp *ControlsPanel) SetRunning(running bool) {
	if running {
		cp.startBtn.Disable()
		cp.stopBtn.Enable()
		cp.resumeBtn.Disable()
	} else {
		cp.startBtn.Enable()
		cp.stopBtn.Disable()
		cp.resumeBtn.Enable()
	}
}

// Widget returns the controls panel container.
func (cp *ControlsPanel) Widget() fyne.CanvasObject {
	return container.NewVBox(
		widget.NewLabel("Threads"),
		cp.slider,
		cp.threadLabel,
		widget.NewSeparator(),
		container.NewGridWithColumns(2, cp.startBtn, cp.stopBtn),
		widget.NewSeparator(),
		cp.fileCheck,
		cp.dirRow,
		cp.resumeBtn,
		widget.NewSeparator(),
		cp.limitCheck,
		cp.limitEntry,
		widget.NewSeparator(),
		cp.autoCheck,
		cp.autoSelect,
		widget.NewSeparator(),
		cp.snapshotBtn,
	)
}
