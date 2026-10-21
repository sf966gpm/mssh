package ui

import (
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	fyneTheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	msshTheme "github.com/sf966gpm/mssh/app/internal/ui/theme"
)

type appShell struct {
	application fyne.App
	window      fyne.Window
	rootDir     string
	statusLabel *widget.Label
	themeSelect *widget.Select
	settings    *widget.Button
}

func newAppShell(application fyne.App, window fyne.Window, mode msshTheme.Mode) *appShell {
	rootDir := startDirectory()
	if absolutePath, err := filepath.Abs(rootDir); err == nil {
		rootDir = absolutePath
	}
	if resolvedPath, err := filepath.EvalSymlinks(rootDir); err == nil {
		rootDir = resolvedPath
	}

	e := &appShell{
		application: application,
		window:      window,
		rootDir:     rootDir,
		statusLabel: widget.NewLabel(""),
	}
	e.themeSelect = widget.NewSelect([]string{"Система", "Светлая", "Тёмная", "JetBrains Dark"}, e.setThemeMode)
	e.themeSelect.SetSelected(themeModeLabel(mode))
	e.settings = widget.NewButtonWithIcon("", fyneTheme.SettingsIcon(), e.showSettings)
	return e
}

func (e *appShell) setThemeMode(label string) {
	mode := themeModeFromLabel(label)
	e.application.Preferences().SetString("theme_mode", string(mode))
	e.application.Settings().SetTheme(msshTheme.NewForMode(mode))
}

func (e *appShell) showSettings() {
	settings := container.NewVBox(widget.NewLabel("Тема оформления"), e.themeSelect)
	dialog.ShowCustom("Настройки", "Закрыть", settings, e.window)
}

func themeModeFromLabel(label string) msshTheme.Mode {
	switch label {
	case "Светлая":
		return msshTheme.ModeLight
	case "Тёмная":
		return msshTheme.ModeDark
	case "JetBrains Dark":
		return msshTheme.ModeJetBrainsDark
	default:
		return msshTheme.ModeSystem
	}
}

func startDirectory() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".ssh")
}

func themeModeLabel(mode msshTheme.Mode) string {
	switch mode {
	case msshTheme.ModeLight:
		return "Светлая"
	case msshTheme.ModeDark:
		return "Тёмная"
	case msshTheme.ModeJetBrainsDark:
		return "JetBrains Dark"
	default:
		return "Система"
	}
}
