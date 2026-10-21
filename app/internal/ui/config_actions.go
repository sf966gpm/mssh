package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"path/filepath"
	"strings"
)

func (o *sshOverview) showConfig(configIndex int) {
	if configIndex < 0 || configIndex >= len(o.configs) {
		return
	}
	o.activeConfig = configIndex
	o.selected = ""
	o.creating = false
	o.activeDraft = 0
	o.clearDraftInputCallbacks()
	config := o.configs[configIndex]
	rows := []fyne.CanvasObject{
		widget.NewLabelWithStyle(config.label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel(fmt.Sprintf("Подключений: %d", len(config.aliases))),
		widget.NewButtonWithIcon("Добавить подключение", theme.ContentAddIcon(), func() { o.addAliasToConfig(configIndex) }),
	}
	if filepath.Clean(config.path) != filepath.Clean(o.sshService.MainConfigPath()) {
		if config.deletable {
			rows = append(rows, widget.NewButtonWithIcon("Удалить конфиг", theme.DeleteIcon(), func() { o.confirmDeleteConfig(configIndex) }))
		} else {
			rows = append(rows, widget.NewLabel("Удаление доступно только для файлов внутри ~/.ssh"))
		}
	}
	o.details.Objects = []fyne.CanvasObject{container.NewVBox(rows...)}
	o.details.Refresh()
}

func (o *sshOverview) confirmDeleteConfig(configIndex int) {
	if configIndex < 0 || configIndex >= len(o.configs) {
		return
	}
	config := o.configs[configIndex]
	if filepath.Clean(config.path) == filepath.Clean(o.sshService.MainConfigPath()) || !config.deletable {
		return
	}
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Введите путь конфига")
	confirm := widget.NewButton("Удалить конфиг", nil)
	confirm.Importance = widget.DangerImportance
	confirm.Disable()
	var popup dialog.Dialog
	entry.OnChanged = func(value string) {
		if value == config.label {
			confirm.Enable()
		} else {
			confirm.Disable()
		}
	}
	confirm.OnTapped = func() {
		popup.Hide()
		o.deleteConfig(config.path)
	}
	popup = dialog.NewCustom("Подтвердите удаление конфига", "Отмена", container.NewVBox(
		widget.NewLabel("Для удаления введите путь конфига полностью:"),
		widget.NewLabelWithStyle(config.label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		entry,
		confirm,
	), o.window)
	popup.Resize(fyne.NewSize(460, 230))
	popup.Show()
}

func (o *sshOverview) deleteConfig(configPath string) {
	result, err := o.sshService.DeleteConfig(configPath)
	if err != nil {
		o.setStatus(err.Error())
		if result.Deleted {
			o.discardDraftForConfig(configPath)
			o.refresh()
		}
		return
	}
	o.discardDraftForConfig(configPath)
	o.refresh()
	message := "Конфиг удалён"
	if result.RemovedInclude {
		message += "; строка Include убрана"
	} else {
		message += "; файл исключён из Include по маске"
	}
	if result.RemovedDirs > 0 {
		message += fmt.Sprintf("; пустых папок удалено: %d", result.RemovedDirs)
	}
	o.setStatus(message)
}

func (o *sshOverview) discardDraftForConfig(configPath string) {
	remainingDrafts := make([]sshAliasDraft, 0, len(o.drafts))
	for _, draft := range o.drafts {
		if filepath.Clean(draft.configPath) == filepath.Clean(configPath) {
			if draft.id == o.activeDraft {
				o.creating = false
				o.activeDraft = 0
				o.clearDraftInputCallbacks()
			}
			continue
		}
		remainingDrafts = append(remainingDrafts, draft)
	}
	o.drafts = remainingDrafts
	for key := range o.edits {
		if filepath.Clean(key.configPath) == filepath.Clean(configPath) {
			delete(o.edits, key)
		}
	}
}

func (o *sshOverview) addConfig() {
	pathEntry := widget.NewEntry()
	pathEntry.SetPlaceHolder("например, /test/work.config")
	create := widget.NewButton("Создать конфиг", nil)
	create.Disable()
	var popup dialog.Dialog
	pathEntry.OnChanged = func(value string) {
		if strings.TrimSpace(value) == "" {
			create.Disable()
		} else {
			create.Enable()
		}
	}
	create.OnTapped = func() {
		popup.Hide()
		o.createConfig(strings.TrimSpace(pathEntry.Text))
	}
	popup = dialog.NewCustom("Новый SSH config", "Отмена", container.NewVBox(
		widget.NewLabel("Путь внутри ~/.ssh, например /test/work.config"), pathEntry, create,
	), o.window)
	popup.Resize(fyne.NewSize(440, 180))
	popup.Show()
}

func (o *sshOverview) createConfig(relativePath string) {
	configPath, err := o.sshService.CreateConfig(relativePath)
	if err != nil {
		o.setStatus(err.Error())
		return
	}
	o.refresh()
	for index := range o.configs {
		if o.configs[index].path == configPath {
			o.configs[index].expanded = true
			o.showConfig(index)
			break
		}
	}
	o.rebuildRows()
	o.setStatus("Конфиг создан и подключён через Include: " + sshConfigLabel(configPath, o.sshDir))
}
