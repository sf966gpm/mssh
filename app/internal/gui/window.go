package gui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/sf966gpm/mssh/internal/sshconfig"
)

func NewWindow(application fyne.App, config *sshconfig.Config) fyne.Window {
	window := application.NewWindow("MSSH")

	details := widget.NewLabel("Select a host")
	details.Wrapping = fyne.TextWrapWord
	detailsScroll := container.NewVScroll(details)

	hostList := widget.NewList(
		func() int { return len(config.Hosts) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, object fyne.CanvasObject) {
			host := config.Hosts[id]
			object.(*widget.Label).SetText(strings.Join(host.Patterns, " "))
		},
	)
	hostList.OnSelected = func(id widget.ListItemID) {
		details.SetText(formatHost(config.Hosts[id]))
	}

	left := container.NewBorder(widget.NewLabel("Hosts"), nil, nil, nil, hostList)
	right := container.NewBorder(widget.NewLabel("Details"), nil, nil, nil, detailsScroll)
	split := container.NewHSplit(left, right)
	split.SetOffset(0.32)

	status := widget.NewLabel(fmt.Sprintf("Read: %s", config.Path))
	diagnostics := widget.NewLabel(formatFooter(config))
	diagnostics.Wrapping = fyne.TextWrapWord
	diagnosticsScroll := container.NewVScroll(diagnostics)

	content := container.NewBorder(status, container.NewVBox(widget.NewSeparator(), diagnosticsScroll), nil, nil, split)
	window.SetContent(content)
	window.Resize(fyne.NewSize(1000, 650))
	if len(config.Hosts) > 0 {
		hostList.Select(0)
	}
	return window
}

func NewErrorWindow(application fyne.App, err error) fyne.Window {
	window := application.NewWindow("MSSH")
	message := widget.NewLabel("MSSH could not read the SSH configuration:\n\n" + err.Error())
	message.Wrapping = fyne.TextWrapWord
	window.SetContent(container.NewPadded(message))
	window.Resize(fyne.NewSize(600, 240))
	return window
}

func formatHost(host sshconfig.HostBlock) string {
	lines := []string{
		"Host: " + strings.Join(host.Patterns, " "),
		fmt.Sprintf("Source: %s:%d", host.Path, host.Line),
		"",
	}
	for _, field := range host.Fields {
		lines = append(lines, fmt.Sprintf("%s: %s (%s:%d)", field.Name, field.DisplayValue(), field.Path, field.Line))
	}
	return strings.Join(lines, "\n")
}

func formatFooter(config *sshconfig.Config) string {
	lines := []string{fmt.Sprintf("Included files: %d", len(config.Includes))}
	for _, include := range config.Includes {
		lines = append(lines, fmt.Sprintf("  %s:%d: %s", include.Path, include.Line, include.Pattern))
	}
	if len(config.Diagnostics) == 0 {
		lines = append(lines, "Diagnostics: none")
		return strings.Join(lines, "\n")
	}

	lines = append(lines, fmt.Sprintf("Diagnostics: %d", len(config.Diagnostics)))
	for _, diagnostic := range config.Diagnostics {
		lines = append(lines, fmt.Sprintf("  %s:%d: %s", diagnostic.File, diagnostic.Line, diagnostic.Message))
	}
	return strings.Join(lines, "\n")
}
