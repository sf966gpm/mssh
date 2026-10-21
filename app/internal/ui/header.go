package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func (e *appShell) headerContent() fyne.CanvasObject {
	content := container.NewHBox(e.settings, widget.NewLabel("MSSH"), layout.NewSpacer())
	return newRoundedSurface(content)
}
