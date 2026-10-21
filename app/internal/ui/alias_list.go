package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (o *sshOverview) aliasContent() fyne.CanvasObject {
	o.alias = widget.NewEntry()
	o.hostName = widget.NewEntry()
	o.user = widget.NewEntry()
	o.port = widget.NewEntry()
	o.identity = widget.NewEntry()
	o.save = widget.NewButton("Сохранить", o.saveAlias)
	o.remove = widget.NewButton("Удалить", o.confirmDelete)
	o.save.Disable()
	o.remove.Disable()
	o.list = widget.NewList(
		func() int { return len(o.rows) },
		func() fyne.CanvasObject {
			return container.NewBorder(
				nil,
				nil,
				widget.NewIcon(theme.FileIcon()),
				widget.NewButtonWithIcon("", theme.ContentAddIcon(), nil),
				widget.NewLabel(""),
			)
		},
		func(id widget.ListItemID, object fyne.CanvasObject) {
			row := object.(*fyne.Container)
			item := o.rows[id]
			label := row.Objects[0].(*widget.Label)
			icon := row.Objects[1].(*widget.Icon)
			button := row.Objects[2].(*widget.Button)
			if item.isDraft {
				draft := o.draftByID(item.draftID)
				if draft == nil {
					return
				}
				name := strings.TrimSpace(draft.values.Name)
				if name == "" {
					name = "Новое подключение"
				}
				label.SetText("    " + name + " · не сохранено")
				label.Importance = widget.LowImportance
				label.Show()
				icon.SetResource(theme.ComputerIcon())
				button.SetIcon(theme.CancelIcon())
				button.SetText("")
				button.Importance = widget.LowImportance
				button.Show()
				button.OnTapped = func() { o.discardDraft(item.draftID); o.rebuildRows() }
				row.Refresh()
				return
			}
			label.Truncation = fyne.TextTruncateEllipsis
			label.Importance = widget.MediumImportance
			if item.isConfig {
				config := o.configs[item.configIndex]
				icon.SetResource(theme.FolderIcon())
				disclosureIcon := theme.NavigateNextIcon()
				if config.expanded {
					disclosureIcon = theme.MenuDropDownIcon()
				}
				label.SetText(config.label)
				label.Show()
				button.Show()
				button.Importance = widget.MediumImportance
				button.SetText("")
				button.SetIcon(disclosureIcon)
				button.OnTapped = func() { o.toggleConfig(item.configIndex) }
				row.Refresh()
			} else {
				icon.SetResource(theme.ComputerIcon())
				name := o.configs[item.configIndex].aliases[item.aliasIndex].name
				if item.modified {
					key := aliasEditKey{
						configPath: o.configs[item.configIndex].path,
						aliasName:  name,
					}
					name = o.edits[key].values.Name
					if strings.TrimSpace(name) == "" {
						name = o.configs[item.configIndex].aliases[item.aliasIndex].name
					}
					label.SetText("    " + name + " · не сохранено")
					label.Importance = widget.LowImportance
					label.Show()
					button.SetIcon(theme.CancelIcon())
					button.SetText("")
					button.Importance = widget.LowImportance
					button.Show()
					button.OnTapped = func() { o.cancelAliasEdit(item.configIndex, item.aliasIndex) }
					row.Refresh()
					return
				}
				label.SetText("    " + name)
				label.Show()
				button.Hide()
				button.Importance = widget.MediumImportance
				button.SetText("")
				row.Refresh()
			}
		},
	)
	o.list.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= widget.ListItemID(len(o.rows)) {
			return
		}
		item := o.rows[id]
		if item.isConfig {
			o.showConfig(item.configIndex)
			return
		}
		if item.isDraft {
			if !o.creating || o.activeDraft != item.draftID {
				o.resumeDraft(item.draftID)
			}
			return
		}
		o.showAlias(item.configIndex, item.aliasIndex)
	}
	o.details = container.NewVBox(widget.NewLabel("Выберите алиас слева"))
	if o.loadErr != "" {
		o.details.Objects = []fyne.CanvasObject{widget.NewLabel(o.loadErr)}
	}
	add := widget.NewButtonWithIcon("Добавить конфиг", theme.ContentAddIcon(), o.addConfig)
	listPane := container.NewBorder(
		container.NewVBox(widget.NewLabelWithStyle("Конфиги SSH", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), add),
		nil,
		nil,
		nil,
		o.list,
	)
	listPanel := container.NewPadded(newRoundedSurface(listPane))
	detailPanel := container.NewPadded(newRoundedSurface(o.details))
	split := container.NewHSplit(listPanel, detailPanel)
	split.SetOffset(0.3)
	return split
}

func (o *sshOverview) toggleConfig(configIndex int) {
	if configIndex < 0 || configIndex >= len(o.configs) {
		return
	}
	o.configs[configIndex].expanded = !o.configs[configIndex].expanded
	o.rebuildRows()
	o.showConfig(configIndex)
}

func (o *sshOverview) selectDraftRow(draftID uint64) {
	for rowIndex, row := range o.rows {
		if row.draftID == draftID && row.isDraft {
			o.list.Select(widget.ListItemID(rowIndex))
			return
		}
	}
}

func (o *sshOverview) draftByID(draftID uint64) *sshAliasDraft {
	for index := range o.drafts {
		if o.drafts[index].id == draftID {
			return &o.drafts[index]
		}
	}
	return nil
}
