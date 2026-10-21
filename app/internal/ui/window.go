package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/sf966gpm/mssh/app/internal/ui/theme"
)

func NewWindow(application fyne.App) fyne.Window {
	window := application.NewWindow("MSSH")
	mode := theme.Mode(application.Preferences().StringWithFallback("theme_mode", string(theme.ModeSystem)))
	if mode != theme.ModeLight && mode != theme.ModeDark {
		mode = theme.ModeSystem
	}
	application.Settings().SetTheme(theme.NewForMode(mode))

	shell := newAppShell(application, window, mode)
	window.SetContent(shell.content())
	window.Resize(fyne.NewSize(1200, 800))
	return window
}

func (e *appShell) content() fyne.CanvasObject {
	settingsBar := container.NewHBox(e.settings, widget.NewLabel("MSSH"), layout.NewSpacer())
	overview := newOverview(e.window, e.rootDir, e.statusLabel.SetText)
	return container.NewBorder(settingsBar, e.statusLabel, nil, nil, overview.aliasContent())
}
