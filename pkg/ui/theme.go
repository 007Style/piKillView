package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// DigitColorNames maps digit 0–9 to a Fyne ThemeColorName so RichTextSegments
// can reference them via ColorName.
var DigitColorNames = [10]fyne.ThemeColorName{
	"digit0", "digit1", "digit2", "digit3", "digit4",
	"digit5", "digit6", "digit7", "digit8", "digit9",
}

// ─── Dark Theme ──────────────────────────────────────────────────────────────

type piKillDarkTheme struct{}

var _ fyne.Theme = (*piKillDarkTheme)(nil)

func (t *piKillDarkTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	// Digit custom colors.
	for i, cn := range DigitColorNames {
		if name == cn {
			return DigitColors[i]
		}
	}
	switch name {
	case theme.ColorNameBackground:
		return color.RGBA{R: 0x0d, G: 0x11, B: 0x17, A: 0xFF}
	case theme.ColorNameButton:
		return color.RGBA{R: 0x16, G: 0x1b, B: 0x22, A: 0xFF}
	case theme.ColorNameDisabledButton:
		return color.RGBA{R: 0x21, G: 0x26, B: 0x2d, A: 0xFF}
	case theme.ColorNameDisabled:
		return color.RGBA{R: 0x8b, G: 0x94, B: 0x9e, A: 0x88}
	case theme.ColorNameError:
		return color.RGBA{R: 0xFF, G: 0x55, B: 0x55, A: 0xFF}
	case theme.ColorNameFocus:
		return color.RGBA{R: 0x00, G: 0xFF, B: 0x41, A: 0x88}
	case theme.ColorNameForeground:
		return color.RGBA{R: 0xe6, G: 0xed, B: 0xf3, A: 0xFF}
	case theme.ColorNameHover:
		return color.RGBA{R: 0x30, G: 0x36, B: 0x3d, A: 0xFF}
	case theme.ColorNameInputBackground:
		return color.RGBA{R: 0x0d, G: 0x11, B: 0x17, A: 0xFF}
	case theme.ColorNameInputBorder:
		return color.RGBA{R: 0x30, G: 0x36, B: 0x3d, A: 0xFF}
	case theme.ColorNameMenuBackground:
		return color.RGBA{R: 0x16, G: 0x1b, B: 0x22, A: 0xFF}
	case theme.ColorNameOverlayBackground:
		return color.RGBA{R: 0x16, G: 0x1b, B: 0x22, A: 0xDD}
	case theme.ColorNamePlaceHolder:
		return color.RGBA{R: 0x8b, G: 0x94, B: 0x9e, A: 0xFF}
	case theme.ColorNamePressed:
		return color.RGBA{R: 0x00, G: 0xFF, B: 0x41, A: 0x44}
	case theme.ColorNamePrimary:
		return color.RGBA{R: 0x00, G: 0xFF, B: 0x41, A: 0xFF} // matrix green
	case theme.ColorNameScrollBar:
		return color.RGBA{R: 0x8b, G: 0x94, B: 0x9e, A: 0x88}
	case theme.ColorNameSeparator:
		return color.RGBA{R: 0x30, G: 0x36, B: 0x3d, A: 0xFF}
	case theme.ColorNameShadow:
		return color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x88}
	case theme.ColorNameSuccess:
		return color.RGBA{R: 0x00, G: 0xFF, B: 0x41, A: 0xFF}
	case theme.ColorNameWarning:
		return color.RGBA{R: 0xFF, G: 0xD7, B: 0x00, A: 0xFF}
	case theme.ColorNameHeaderBackground:
		return color.RGBA{R: 0x16, G: 0x1b, B: 0x22, A: 0xFF}
	case theme.ColorNameSelection:
		return color.RGBA{R: 0x00, G: 0xFF, B: 0x41, A: 0x44}
	}
	return theme.DefaultTheme().Color(name, theme.VariantDark)
}

func (t *piKillDarkTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *piKillDarkTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *piKillDarkTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

// ─── Light Theme ─────────────────────────────────────────────────────────────

type piKillLightTheme struct{}

var _ fyne.Theme = (*piKillLightTheme)(nil)

func (t *piKillLightTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	// Digit custom colors (same for light theme).
	for i, cn := range DigitColorNames {
		if name == cn {
			return DigitColors[i]
		}
	}
	switch name {
	case theme.ColorNameBackground:
		return color.RGBA{R: 0xf0, G: 0xf0, B: 0xf0, A: 0xFF}
	case theme.ColorNameButton:
		return color.RGBA{R: 0xe0, G: 0xe0, B: 0xe0, A: 0xFF}
	case theme.ColorNameForeground:
		return color.RGBA{R: 0x1f, G: 0x23, B: 0x28, A: 0xFF}
	case theme.ColorNamePrimary:
		return color.RGBA{R: 0x00, G: 0x88, B: 0x33, A: 0xFF}
	case theme.ColorNameFocus:
		return color.RGBA{R: 0x00, G: 0x88, B: 0x33, A: 0x88}
	case theme.ColorNameHover:
		return color.RGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xFF}
	case theme.ColorNameInputBackground:
		return color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	case theme.ColorNamePlaceHolder:
		return color.RGBA{R: 0x57, G: 0x60, B: 0x6a, A: 0xFF}
	case theme.ColorNameSuccess:
		return color.RGBA{R: 0x00, G: 0x88, B: 0x33, A: 0xFF}
	case theme.ColorNameWarning:
		return color.RGBA{R: 0xFF, G: 0x88, B: 0x00, A: 0xFF}
	}
	return theme.DefaultTheme().Color(name, theme.VariantLight)
}

func (t *piKillLightTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *piKillLightTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *piKillLightTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

// ─── Public constructors ──────────────────────────────────────────────────────

// NewDarkTheme returns the piKillView dark theme.
func NewDarkTheme() fyne.Theme { return &piKillDarkTheme{} }

// NewLightTheme returns the piKillView light theme.
func NewLightTheme() fyne.Theme { return &piKillLightTheme{} }

// ToggleTheme switches the app between dark and light theme and updates
// *current to the newly applied theme.
func ToggleTheme(a fyne.App, current *fyne.Theme) {
	if _, ok := (*current).(*piKillDarkTheme); ok {
		*current = NewLightTheme()
	} else {
		*current = NewDarkTheme()
	}
	a.Settings().SetTheme(*current)
}
