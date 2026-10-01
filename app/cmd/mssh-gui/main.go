package main

import (
	"fyne.io/fyne/v2/app"
	"github.com/sf966gpm/mssh/internal/gui"
	"github.com/sf966gpm/mssh/internal/sshconfig"
)

func main() {
	application := app.New()
	path, err := sshconfig.ConfigPath()
	if err != nil {
		window := gui.NewErrorWindow(application, err)
		window.ShowAndRun()
		return
	}

	config, err := sshconfig.ParseFileExpanded(path)
	if err != nil {
		window := gui.NewErrorWindow(application, err)
		window.ShowAndRun()
		return
	}

	window := gui.NewWindow(application, config)
	window.ShowAndRun()
}
