package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/007Style/piKillView/pkg/engine"
)

const AppTitle = "piKillView v1.1 — From the minds of IBM Bob & Daneyand"

// AppComponents holds all major UI widgets so main.go can wire callbacks.
type AppComponents struct {
	Controls    *ControlsPanel
	Stats       *StatsPanel
	DigitView   *DigitView
	Header      *HeaderWidget
	Trivia      *TriviaWidget
	SessionLog  *SessionLog
	Toast       *ToastOverlay
	SearchBar   *SearchBar
	CurrentTheme fyne.Theme
}

// NewMainWindow constructs and returns the main application window.
// It also populates comp with all components so the caller can wire them.
func NewMainWindow(a fyne.App, eng *engine.Engine, comp *AppComponents) fyne.Window {
	w := a.NewWindow(AppTitle)
	w.Resize(fyne.NewSize(1200, 750))
	w.SetMaster()

	currentTheme := NewDarkTheme()
	comp.CurrentTheme = currentTheme

	// Build components.
	header := NewHeaderWidget()
	trivia := NewTriviaWidget()
	sessionLog := NewSessionLog()
	toast := NewToastOverlay()
	digitView := NewDigitView()
	stats := NewStatsPanel()
	controls := NewControlsPanel(w)
	search := NewSearchBar()

	comp.Header = header
	comp.Trivia = trivia
	comp.SessionLog = sessionLog
	comp.Toast = toast
	comp.DigitView = digitView
	comp.Stats = stats
	comp.Controls = controls
	comp.SearchBar = search

	// ── Search result callback ──
	search.OnResult = func(matchStart int64, length int) {
		digitView.SetHighlight(matchStart, int64(length))
		digitView.AppendDigits("") // trigger refresh
	}

	// ── About button ──
	aboutBtn := widget.NewButton("π About", func() {
		ShowAboutDialog(w)
	})

	// ── Theme toggle button ──
	themeBtn := widget.NewButton("☾", func() {
		ToggleTheme(a, &comp.CurrentTheme)
	})

	// ── Status bar ──
	logVisible := true
	logToggle := widget.NewButton("▾ Log", func() {
		if logVisible {
			sessionLog.Collapse()
			logVisible = false
		} else {
			sessionLog.Expand()
			logVisible = true
		}
	})
	hintLabel := widget.NewLabel("Space: Start/Stop  |  ⌘F: Search  |  ⌘S: Snapshot")
	hintLabel.TextStyle = fyne.TextStyle{Monospace: true}

	statusBar := container.NewBorder(nil, nil, logToggle,
		container.NewHBox(aboutBtn, themeBtn), hintLabel)

	// ── Bottom strip: trivia + status bar + session log ──
	bottom := container.NewVBox(
		trivia,
		statusBar,
		sessionLog.Widget(),
	)

	// ── Center: digit view stacked with toast overlay ──
	center := container.NewStack(digitView.Widget(), toast.Widget())

	// ── Left: controls in scroll ──
	leftScroll := container.NewVScroll(controls.Widget())
	leftScroll.SetMinSize(fyne.NewSize(300, 0))

	// ── Right: stats in scroll ──
	rightScroll := container.NewVScroll(stats.Widget())
	rightScroll.SetMinSize(fyne.NewSize(240, 0))

	// ── Center with search bar on top ──
	centerWithSearch := container.NewBorder(search.Widget(), nil, nil, nil, center)

	// ── Root layout ──
	root := container.NewBorder(
		header,
		bottom,
		leftScroll,
		rightScroll,
		centerWithSearch,
	)
	w.SetContent(root)

	// ── Keyboard shortcuts ──
	// Space: start/stop (handled via custom key handler — Fyne doesn't have a Space shortcut)
	w.Canvas().SetOnTypedKey(func(ke *fyne.KeyEvent) {
		switch ke.Name {
		case fyne.KeySpace:
			if controls.startBtn.Disabled() {
				if controls.OnStop != nil {
					controls.OnStop()
				}
			} else {
				if controls.OnStart != nil {
					controls.OnStart(controls.buildConfig())
				}
			}
		case fyne.KeyEscape:
			search.Hide()
		}
	})

	// Cmd+F: show search
	w.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyF,
		Modifier: fyne.KeyModifierSuper,
	}, func(s fyne.Shortcut) {
		search.Show()
	})

	// Cmd+S: snapshot
	w.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyS,
		Modifier: fyne.KeyModifierSuper,
	}, func(s fyne.Shortcut) {
		if controls.OnSnapshot != nil {
			controls.OnSnapshot()
		}
	})

	// ── Close handler ──
	w.SetOnClosed(func() {
		eng.Stop()
		header.Stop()
		trivia.Stop()
		stats.CPUDials.Stop()
	})

	return w
}
