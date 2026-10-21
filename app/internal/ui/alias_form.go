package ui

import (
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/sf966gpm/mssh/app/internal/sshconfig"
)

func (o *sshOverview) showAlias(configIndex, index int) {
	if configIndex < 0 || configIndex >= len(o.configs) || index < 0 || index >= len(o.configs[configIndex].aliases) {
		return
	}
	o.activeConfig = configIndex
	o.clearDraftInputCallbacks()
	alias := o.configs[configIndex].aliases[index]
	key := aliasEditKey{configPath: o.configs[configIndex].path, aliasName: alias.name}
	original := aliasValues(alias)
	values := original
	if edit, ok := o.edits[key]; ok {
		original = edit.original
		values = edit.values
	}
	o.selected = alias.name
	o.creating = false
	o.activeDraft = 0
	o.save.SetText("Сохранить")
	o.alias.SetText(values.Name)
	o.hostName.SetText(values.HostName)
	o.user.SetText(values.User)
	o.port.SetText(values.Port)
	o.identity.SetText(values.Identity)
	if alias.hostEditable {
		o.alias.Enable()
	} else {
		o.alias.Disable()
	}
	o.hostName.Enable()
	o.user.Enable()
	o.port.Enable()
	o.identity.Enable()
	o.identityPicker.Enable()
	o.save.Enable()
	o.remove.Enable()
	if !alias.editable {
		o.hostName.Disable()
		o.user.Disable()
		o.port.Disable()
		o.identity.Disable()
		o.identityPicker.Disable()
	}
	if !alias.hostEditable {
		o.save.Disable()
		o.remove.Disable()
	}
	if alias.hostEditable {
		o.alias.OnChanged = func(value string) {
			o.validateEntry(o.alias)
			values.Name = value
			o.updateAliasEdit(key, original, values)
		}
	}
	if alias.editable {
		o.hostName.OnChanged = func(value string) {
			o.validateEntry(o.hostName)
			values.HostName = value
			o.updateAliasEdit(key, original, values)
		}
		o.user.OnChanged = func(value string) {
			o.validateEntry(o.user)
			values.User = value
			o.updateAliasEdit(key, original, values)
		}
		o.port.OnChanged = func(value string) {
			o.validateEntry(o.port)
			values.Port = value
			o.updateAliasEdit(key, original, values)
		}
		o.identity.OnChanged = func(value string) {
			o.validateEntry(o.identity)
			values.Identity = value
			o.updateAliasEdit(key, original, values)
		}
	}
	lines := []fyne.CanvasObject{
		o.formRow("Host", o.alias),
		o.formRow("HostName", o.hostName),
		o.formRow("User", o.user),
		o.formRow("Port", o.port),
		o.identityFormRow(),
	}
	lines = append(lines, widget.NewSeparator(), container.NewHBox(layout.NewSpacer(), o.remove, o.save))
	o.details.Objects = []fyne.CanvasObject{container.NewVBox(lines...)}
	o.details.Refresh()
}

func (o *sshOverview) formRow(name string, entry *widget.Entry) fyne.CanvasObject {
	entryRow := o.fieldInput(name, entry)
	return container.NewGridWithColumns(2,
		widget.NewLabelWithStyle(name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		entryRow,
	)
}

func (o *sshOverview) identityFormRow() fyne.CanvasObject {
	return container.NewGridWithColumns(2,
		widget.NewLabelWithStyle("IdentityFile", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewBorder(nil, nil, nil, o.identityPicker, o.fieldInput("IdentityFile", o.identity)),
	)
}

func (o *sshOverview) chooseIdentityFile() {
	fileDialog := dialog.NewFileOpen(func(file fyne.URIReadCloser, err error) {
		if err != nil {
			o.setStatus("Не удалось выбрать файл ключа: " + err.Error())
			return
		}
		if file == nil {
			return
		}
		defer file.Close()

		uri := file.URI()
		if uri.Scheme() != "file" {
			o.setStatus("Выберите файл из локальной файловой системы")
			return
		}
		if uri.Authority() != "" && uri.Authority() != "localhost" {
			o.setStatus("Не удалось определить локальный путь к файлу")
			return
		}
		o.identity.SetText(filepath.FromSlash(uri.Path()))
	}, o.window)
	location, err := storage.ListerForURI(storage.NewFileURI(o.sshDir))
	if err != nil {
		o.setStatus("Не удалось открыть папку ~/.ssh: " + err.Error())
		return
	}
	fileDialog.SetLocation(location)
	fileDialog.Resize(fyne.NewSize(1000, 700))
	fileDialog.Show()
}

func (o *sshOverview) addAliasToConfig(configIndex int) {
	if configIndex < 0 || configIndex >= len(o.configs) {
		return
	}
	o.nextDraftID++
	draft := sshAliasDraft{
		id:         o.nextDraftID,
		configPath: o.configs[configIndex].path,
		values:     sshconfig.AliasValues{Port: "22"},
	}
	o.drafts = append(o.drafts, draft)
	o.configs[configIndex].expanded = true
	o.rebuildRows()
	o.resumeDraft(draft.id)
}

func (o *sshOverview) resumeDraft(draftID uint64) {
	draftIndex := -1
	for index := range o.drafts {
		if o.drafts[index].id == draftID {
			draftIndex = index
			break
		}
	}
	if draftIndex < 0 {
		return
	}
	draft := o.drafts[draftIndex]
	configIndex := -1
	for index := range o.configs {
		if o.configs[index].path == draft.configPath {
			configIndex = index
			break
		}
	}
	if configIndex < 0 {
		o.discardDraft(draftID)
		o.rebuildRows()
		return
	}
	o.activeConfig = configIndex
	o.configs[configIndex].expanded = true
	o.creating = true
	o.activeDraft = draftID
	o.selected = ""
	o.clearDraftInputCallbacks()
	o.alias.SetText(draft.values.Name)
	o.hostName.SetText(draft.values.HostName)
	o.user.SetText(draft.values.User)
	o.port.SetText(draft.values.Port)
	o.identity.SetText(draft.values.Identity)
	o.alias.Enable()
	o.alias.OnChanged = func(value string) {
		o.validateEntry(o.alias)
		o.updateDraft(draftID, func(values *sshconfig.AliasValues) { values.Name = value })
		o.rebuildRows()
	}
	o.hostName.OnChanged = func(value string) {
		o.validateEntry(o.hostName)
		o.updateDraft(draftID, func(values *sshconfig.AliasValues) { values.HostName = value })
	}
	o.user.OnChanged = func(value string) {
		o.validateEntry(o.user)
		o.updateDraft(draftID, func(values *sshconfig.AliasValues) { values.User = value })
	}
	o.port.OnChanged = func(value string) {
		o.validateEntry(o.port)
		o.updateDraft(draftID, func(values *sshconfig.AliasValues) { values.Port = value })
	}
	o.identity.OnChanged = func(value string) {
		o.validateEntry(o.identity)
		o.updateDraft(draftID, func(values *sshconfig.AliasValues) { values.Identity = value })
	}
	o.hostName.Enable()
	o.user.Enable()
	o.port.Enable()
	o.identity.Enable()
	o.identityPicker.Enable()
	o.remove.Disable()
	o.save.SetText("Создать подключение")
	o.save.Enable()
	o.details.Objects = []fyne.CanvasObject{container.NewVBox(
		widget.NewLabelWithStyle("Новое подключение", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		o.formRow("Host", o.alias),
		o.formRow("HostName", o.hostName),
		o.formRow("User", o.user),
		o.formRow("Port", o.port),
		o.identityFormRow(),
		widget.NewSeparator(),
		container.NewHBox(layout.NewSpacer(), o.save),
	)}
	o.details.Refresh()
	o.rebuildRows()
	o.selectDraftRow(draftID)
}

func (o *sshOverview) updateDraft(draftID uint64, update func(*sshconfig.AliasValues)) {
	for index := range o.drafts {
		if o.drafts[index].id == draftID {
			update(&o.drafts[index].values)
			return
		}
	}
}

func (o *sshOverview) updateAliasEdit(key aliasEditKey, original, values sshconfig.AliasValues) {
	if values == original {
		delete(o.edits, key)
	} else {
		if o.edits == nil {
			o.edits = make(map[aliasEditKey]sshAliasEdit)
		}
		o.edits[key] = sshAliasEdit{original: original, values: values}
	}
	o.rebuildRows()
}

func aliasValues(alias sshAlias) sshconfig.AliasValues {
	return sshconfig.AliasValues{
		Name: alias.name, HostName: alias.hostName, User: alias.user,
		Port: alias.port, Identity: alias.identity,
	}
}

func (o *sshOverview) clearDraftInputCallbacks() {
	o.alias.OnChanged = nil
	o.hostName.OnChanged = nil
	o.user.OnChanged = nil
	o.port.OnChanged = nil
	o.identity.OnChanged = nil
}

func (o *sshOverview) saveAlias() {
	if o.activeConfig < 0 || o.activeConfig >= len(o.configs) {
		return
	}
	for _, field := range []*widget.Entry{o.alias, o.hostName, o.user, o.port, o.identity} {
		if err := o.validateEntry(field); err != nil {
			o.setStatus(err.Error())
			return
		}
	}
	configPath := o.configs[o.activeConfig].path
	wasCreating := o.creating
	draftID := o.activeDraft
	editKey := aliasEditKey{configPath: configPath, aliasName: o.selected}
	aliasName := strings.TrimSpace(o.alias.Text)
	values := sshconfig.AliasValues{
		Name: aliasName, HostName: strings.TrimSpace(o.hostName.Text),
		User: strings.TrimSpace(o.user.Text), Port: strings.TrimSpace(o.port.Text),
		Identity: strings.TrimSpace(o.identity.Text),
	}
	if err := o.sshService.SaveAlias(configPath, o.selected, values, o.creating); err != nil {
		o.setStatus(err.Error())
		return
	}
	o.creating = false
	if wasCreating {
		o.discardDraft(draftID)
	} else {
		delete(o.edits, editKey)
	}
	o.refresh()
	for index, alias := range o.configs[o.activeConfig].aliases {
		if alias.name == values.Name {
			for rowIndex, row := range o.rows {
				if row.configIndex == o.activeConfig && !row.isConfig && !row.isDraft && row.aliasIndex == index {
					o.list.Select(widget.ListItemID(rowIndex))
					break
				}
			}
			break
		}
	}
	o.setStatus("SSH config проверен и сохранён")
}

func (o *sshOverview) discardDraft(draftID uint64) {
	remaining := make([]sshAliasDraft, 0, len(o.drafts))
	for _, draft := range o.drafts {
		if draft.id != draftID {
			remaining = append(remaining, draft)
		}
	}
	o.drafts = remaining
	if o.activeDraft != draftID {
		return
	}
	o.activeDraft = 0
	o.creating = false
	o.clearDraftInputCallbacks()
	for _, entry := range []*widget.Entry{o.alias, o.hostName, o.user, o.port, o.identity} {
		entry.Disable()
	}
	o.save.Disable()
	o.remove.Disable()
	o.details.Objects = []fyne.CanvasObject{widget.NewLabel("Создание подключения отменено")}
	o.details.Refresh()
}

func (o *sshOverview) cancelAliasEdit(configIndex, aliasIndex int) {
	if configIndex < 0 || configIndex >= len(o.configs) || aliasIndex < 0 || aliasIndex >= len(o.configs[configIndex].aliases) {
		return
	}
	alias := o.configs[configIndex].aliases[aliasIndex]
	key := aliasEditKey{configPath: o.configs[configIndex].path, aliasName: alias.name}
	delete(o.edits, key)
	o.rebuildRows()
	if !o.creating && o.activeConfig == configIndex && o.selected == alias.name {
		o.showAlias(configIndex, aliasIndex)
	}
}

func (o *sshOverview) confirmDelete() {
	if o.selected == "" {
		return
	}
	aliasName := o.selected
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Введите " + aliasName)
	confirm := widget.NewButton("Удалить подключение", nil)
	confirm.Importance = widget.DangerImportance
	confirm.Disable()
	var confirmation dialog.Dialog
	entry.OnChanged = func(value string) {
		if value == aliasName {
			confirm.Enable()
		} else {
			confirm.Disable()
		}
	}
	confirm.OnTapped = func() {
		confirmation.Hide()
		o.deleteAlias(aliasName)
	}
	content := container.NewVBox(
		widget.NewLabel("Для удаления введите алиас подключения точно так, как он указан в Host."),
		widget.NewLabelWithStyle(aliasName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		entry,
		confirm,
	)
	confirmation = dialog.NewCustom("Подтвердите удаление", "Отмена", content, o.window)
	confirmation.Resize(fyne.NewSize(420, 220))
	confirmation.Show()
}

func (o *sshOverview) deleteAlias(aliasName string) {
	configPath := o.configs[o.activeConfig].path
	if err := o.sshService.DeleteAlias(configPath, aliasName); err != nil {
		o.setStatus(err.Error())
		return
	}
	o.selected = ""
	o.refresh()
	o.details.Objects = []fyne.CanvasObject{widget.NewLabel("Подключение удалено")}
	o.details.Refresh()
	o.setStatus("Подключение Host " + aliasName + " удалено")
}
