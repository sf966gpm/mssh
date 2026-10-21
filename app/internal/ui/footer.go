package ui

import (
	"fyne.io/fyne/v2"
)

func (e *appShell) footerContent() fyne.CanvasObject {
	return newRoundedSurface(e.statusLabel)
}
