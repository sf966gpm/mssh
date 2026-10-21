package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/sf966gpm/mssh/app/internal/sshconfig"
)

type sshAlias struct {
	name          string
	hostName      string
	user          string
	port          string
	identity      string
	hostEditable  bool
	identityCount int
	editable      bool
}

type sshConfigFile struct {
	path      string
	label     string
	aliases   []sshAlias
	expanded  bool
	deletable bool
}

type sshAliasRow struct {
	configIndex int
	aliasIndex  int
	isConfig    bool
	isDraft     bool
}

type sshOverview struct {
	window       fyne.Window
	sshDir       string
	setStatus    func(string)
	sshService   *sshconfig.Service
	configs      []sshConfigFile
	rows         []sshAliasRow
	activeConfig int
	draftAlias   string
	loadErr      string
	list         *widget.List
	details      *fyne.Container
	selected     string
	creating     bool
	hostName     *widget.Entry
	alias        *widget.Entry
	user         *widget.Entry
	port         *widget.Entry
	identity     *widget.Entry
	save         *widget.Button
	remove       *widget.Button
}

func newOverview(window fyne.Window, sshDir string, setStatus func(string)) *sshOverview {
	service := sshconfig.NewService(sshDir)
	overview := &sshOverview{window: window, sshDir: sshDir, setStatus: setStatus, sshService: service}
	overview.refresh()
	return overview
}

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
			return container.NewHBox(widget.NewIcon(theme.FileIcon()), widget.NewButton("", nil), layout.NewSpacer(), widget.NewButtonWithIcon("", theme.ContentAddIcon(), nil))
		},
		func(id widget.ListItemID, object fyne.CanvasObject) {
			row := object.(*fyne.Container)
			item := o.rows[id]
			icon := row.Objects[0].(*widget.Icon)
			label := row.Objects[1].(*widget.Button)
			button := row.Objects[3].(*widget.Button)
			if item.isConfig {
				config := o.configs[item.configIndex]
				icon.SetResource(theme.FolderIcon())
				disclosure := "▸ "
				if config.expanded {
					disclosure = "▾ "
				}
				label.SetText(disclosure + config.label)
				label.Importance = widget.LowImportance
				label.Show()
				label.Enable()
				label.OnTapped = func() { o.toggleConfig(item.configIndex) }
				button.Show()
				button.OnTapped = func() { o.addAliasToConfig(item.configIndex) }
			} else {
				icon.SetResource(theme.ComputerIcon())
				name := o.draftAlias
				if !item.isDraft {
					name = o.configs[item.configIndex].aliases[item.aliasIndex].name
				}
				label.SetText("    " + name)
				label.Importance = widget.LowImportance
				label.Show()
				if item.isDraft {
					label.Disable()
				} else {
					label.Enable()
					label.OnTapped = func() { o.showAlias(item.configIndex, item.aliasIndex) }
				}
				button.Hide()
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
	detailPane := container.NewPadded(o.details)
	split := container.NewHSplit(listPane, detailPane)
	split.SetOffset(0.3)
	return split
}

func (o *sshOverview) refresh() {
	activePath := o.sshService.MainConfigPath()
	if o.activeConfig >= 0 && o.activeConfig < len(o.configs) {
		activePath = o.configs[o.activeConfig].path
	}
	o.creating = false
	o.selected = ""
	o.draftAlias = ""
	if o.alias != nil {
		o.alias.OnChanged = nil
		o.alias.Disable()
		o.hostName.Disable()
		o.user.Disable()
		o.port.Disable()
		o.identity.Disable()
		o.save.Disable()
		o.save.SetText("Сохранить")
		o.remove.Disable()
	}
	snapshot, err := o.sshService.Load()
	if err != nil {
		o.loadErr = err.Error()
		o.showLoadError(err)
		return
	}
	o.loadErr = snapshot.Warning
	o.configs = make([]sshConfigFile, 0, len(snapshot.Configs))
	o.activeConfig = 0
	for _, sourceConfig := range snapshot.Configs {
		config := sshConfigFile{
			path: sourceConfig.Path, label: sshConfigLabel(sourceConfig.Path, o.sshDir),
			expanded: true, deletable: sourceConfig.Deletable,
		}
		for _, sourceAlias := range sourceConfig.Aliases {
			config.aliases = append(config.aliases, sshAlias{
				name: sourceAlias.Name, hostName: sourceAlias.HostName, user: sourceAlias.User,
				port: sourceAlias.Port, identity: sourceAlias.Identity,
				hostEditable: sourceAlias.HostEditable, identityCount: sourceAlias.IdentityCount,
				editable: sourceAlias.Editable,
			})
		}
		o.configs = append(o.configs, config)
	}
	for index := range o.configs {
		if o.configs[index].path == activePath {
			o.activeConfig = index
			break
		}
	}
	aliasCount := 0
	for configIndex := range o.configs {
		aliasCount += len(o.configs[configIndex].aliases)
	}
	o.rebuildRows()
	if o.list != nil {
		o.list.UnselectAll()
		o.list.Refresh()
	}
	if o.details != nil {
		o.details.Objects = []fyne.CanvasObject{widget.NewLabel("Выберите конфиг или подключение слева")}
		o.details.Refresh()
	}
	status := fmt.Sprintf("Найдено конфигов: %d; алиасов: %d", len(o.configs), aliasCount)
	if o.loadErr != "" {
		status = o.loadErr
	}
	o.setStatus(status)
}

func (o *sshOverview) showLoadError(err error) {
	o.configs = nil
	o.rows = nil
	if o.list != nil {
		o.list.Refresh()
	}
	if o.details != nil {
		message := "SSH config не найден: " + o.sshService.MainConfigPath()
		if !errors.Is(err, os.ErrNotExist) {
			message = err.Error()
		}
		o.details.Objects = []fyne.CanvasObject{widget.NewLabel(message)}
		o.details.Refresh()
	}
	o.setStatus("Не удалось загрузить SSH config")
}

func (o *sshOverview) rebuildRows() {
	o.rows = make([]sshAliasRow, 0)
	for configIndex, config := range o.configs {
		o.rows = append(o.rows, sshAliasRow{configIndex: configIndex, aliasIndex: -1, isConfig: true})
		if config.expanded {
			for aliasIndex := range config.aliases {
				o.rows = append(o.rows, sshAliasRow{configIndex: configIndex, aliasIndex: aliasIndex})
			}
			if o.creating && configIndex == o.activeConfig && o.draftAlias != "" {
				o.rows = append(o.rows, sshAliasRow{configIndex: configIndex, aliasIndex: -1, isDraft: true})
			}
		}
	}
	if o.list != nil {
		o.list.Refresh()
	}
}

func sshConfigLabel(path, sshDir string) string {
	relative, err := filepath.Rel(sshDir, path)
	if err != nil || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || relative == ".." {
		return path
	}
	return "~/.ssh/" + filepath.ToSlash(relative)
}

func (o *sshOverview) showConfig(configIndex int) {
	if configIndex < 0 || configIndex >= len(o.configs) {
		return
	}
	o.activeConfig = configIndex
	o.selected = ""
	o.creating = false
	o.alias.OnChanged = nil
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
			o.refresh()
		}
		return
	}
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

func (o *sshOverview) toggleConfig(configIndex int) {
	if configIndex < 0 || configIndex >= len(o.configs) {
		return
	}
	o.configs[configIndex].expanded = !o.configs[configIndex].expanded
	o.rebuildRows()
	o.showConfig(configIndex)
}

func (o *sshOverview) showAlias(configIndex, index int) {
	if configIndex < 0 || configIndex >= len(o.configs) || index < 0 || index >= len(o.configs[configIndex].aliases) {
		return
	}
	o.activeConfig = configIndex
	o.alias.OnChanged = nil
	o.draftAlias = ""
	alias := o.configs[configIndex].aliases[index]
	o.selected = alias.name
	o.creating = false
	o.save.SetText("Сохранить")
	o.alias.SetText(alias.name)
	o.hostName.SetText(alias.hostName)
	o.user.SetText(alias.user)
	o.port.SetText(alias.port)
	o.identity.SetText(alias.identity)
	if alias.hostEditable {
		o.alias.Enable()
	} else {
		o.alias.Disable()
	}
	o.hostName.Enable()
	o.user.Enable()
	o.port.Enable()
	o.identity.Enable()
	o.save.Enable()
	o.remove.Enable()
	if !alias.editable {
		o.hostName.Disable()
		o.user.Disable()
		o.port.Disable()
		o.identity.Disable()
	}
	if !alias.hostEditable {
		o.save.Disable()
		o.remove.Disable()
	}
	lines := []fyne.CanvasObject{
		o.formRow("Host", o.alias),
		o.formRow("HostName", o.hostName),
		o.formRow("User", o.user),
		o.formRow("Port", o.port),
		o.formRow("IdentityFile", o.identity),
	}
	lines = append(lines, widget.NewSeparator(), container.NewHBox(layout.NewSpacer(), o.remove, o.save))
	o.details.Objects = []fyne.CanvasObject{container.NewVBox(lines...)}
	o.details.Refresh()
}

func (o *sshOverview) formRow(name string, entry *widget.Entry) fyne.CanvasObject {
	entryRow := container.NewBorder(nil, nil, nil, nil, entry)
	return container.NewGridWithColumns(2,
		widget.NewLabelWithStyle(name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		entryRow,
	)
}

func (o *sshOverview) addAliasToConfig(configIndex int) {
	if configIndex < 0 || configIndex >= len(o.configs) {
		return
	}
	o.activeConfig = configIndex
	o.creating = true
	o.selected = ""
	o.draftAlias = ""
	o.alias.SetText("")
	o.hostName.SetText("")
	o.user.SetText("")
	o.port.SetText("22")
	o.identity.SetText("")
	o.alias.Enable()
	o.alias.OnChanged = func(value string) {
		o.draftAlias = strings.TrimSpace(value)
		o.rebuildRows()
	}
	o.hostName.Enable()
	o.user.Enable()
	o.port.Enable()
	o.identity.Enable()
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
		o.formRow("IdentityFile", o.identity),
		widget.NewSeparator(),
		container.NewHBox(layout.NewSpacer(), o.save),
	)}
	o.details.Refresh()
	o.rebuildRows()
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

func (o *sshOverview) saveAlias() {
	if o.activeConfig < 0 || o.activeConfig >= len(o.configs) {
		return
	}
	configPath := o.configs[o.activeConfig].path
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
