package main

import (
	"fyne.io/fyne/v2/app"

	"github.com/sf966gpm/mssh/app/internal/ui"
)

func main() {
	application := app.New()
	window := ui.NewWindow(application)
	window.ShowAndRun()
}
