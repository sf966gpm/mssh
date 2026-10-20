package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"

	"github.com/sf966gpm/mssh/app/internal/ui/theme"
)

func NewWindow(application fyne.App) fyne.Window {
	window := application.NewWindow("MSSH")
	mode := theme.Mode(application.Preferences().StringWithFallback("theme_mode", string(theme.ModeSystem)))
	if mode != theme.ModeLight && mode != theme.ModeDark {
		mode = theme.ModeSystem
	}
	application.Settings().SetTheme(theme.NewForMode(mode))

	explorer := newExplorer(application, window, mode)
	window.SetContent(explorer.content())
	window.Resize(fyne.NewSize(1200, 800))
	return window
}

func (e *explorer) content() fyne.CanvasObject {
	leftPane := container.NewBorder(e.pathLabel, nil, nil, nil, e.fileList)
	fileActions := container.NewHBox(e.editButton, e.saveButton, e.cancelButton)
	rightHeader := container.NewBorder(nil, nil, nil, fileActions, e.fileLabel)
	rightPane := container.NewBorder(rightHeader, nil, nil, nil, e.fileBody)
	split := container.NewHSplit(leftPane, rightPane)
	split.SetOffset(0.2)
	header := container.NewHBox(e.settings, layout.NewSpacer())
	main := container.NewBorder(header, e.statusLabel, nil, nil, split)
	return main
}
