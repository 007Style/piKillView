package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"
)

const appTitle = "piKillView v1.0 — From the minds of IBM Bob & Daneyand"

func main() {
	a := app.New()
	a.Settings().SetTheme(theme.DarkTheme())

	w := a.NewWindow(appTitle)
	w.Resize(fyne.NewSize(1200, 750))
	w.ShowAndRun()
}
