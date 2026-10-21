package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/sf966gpm/mssh/app/internal/sshconfig"
)

func (o *sshOverview) validationForField(name string) fyne.StringValidator {
	switch name {
	case "Host":
		return sshconfig.ValidateAlias
	case "HostName":
		return sshconfig.ValidateHostName
	case "User":
		return sshconfig.ValidateUser
	case "Port":
		return sshconfig.ValidatePort
	case "IdentityFile":
		return sshconfig.ValidateIdentityFile
	default:
		return nil
	}
}

func (o *sshOverview) validationBorder(entry *widget.Entry) *canvas.Rectangle {
	if o.validationBorders == nil {
		o.validationBorders = make(map[*widget.Entry]*canvas.Rectangle)
	}
	if border := o.validationBorders[entry]; border != nil {
		return border
	}
	border := canvas.NewRectangle(color.Transparent)
	border.StrokeWidth = 2
	border.CornerRadius = theme.Size(theme.SizeNameInputRadius)
	border.Hide()
	o.validationBorders[entry] = border
	return border
}

func (o *sshOverview) validateEntry(entry *widget.Entry) error {
	if entry == nil || entry.Validator == nil {
		return nil
	}
	err := entry.Validator(entry.Text)
	entry.SetValidationError(err)
	if border := o.validationBorders[entry]; border != nil {
		if err != nil && !entry.Disabled() {
			border.StrokeColor = theme.Color(theme.ColorNameError)
			border.Show()
		} else {
			border.Hide()
		}
		border.Refresh()
	}
	entry.Refresh()
	return err
}

func (o *sshOverview) fieldInput(name string, entry *widget.Entry) fyne.CanvasObject {
	entry.Validator = o.validationForField(name)
	entry.AlwaysShowValidationError = true
	border := o.validationBorder(entry)
	o.validateEntry(entry)
	return container.NewStack(entry, border)
}
