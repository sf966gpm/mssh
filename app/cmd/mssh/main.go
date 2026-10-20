package main

import (
	"fyne.io/fyne/v2/app"

	"github.com/sf966gpm/mssh/app/internal/ui"
)

func main() {
	application := app.NewWithID("io.github.sf966gpm.mssh")
	window := ui.NewWindow(application)
	window.ShowAndRun()
}
