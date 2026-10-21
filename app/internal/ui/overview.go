package ui

import (
	"errors"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/sf966gpm/mssh/app/internal/sshconfig"
	"os"
	"path/filepath"
	"strings"
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
	draftID     uint64
	isConfig    bool
	isDraft     bool
	modified    bool
}

type sshAliasDraft struct {
	id         uint64
	configPath string
	values     sshconfig.AliasValues
}

type aliasEditKey struct {
	configPath string
	aliasName  string
}

type sshAliasEdit struct {
	original sshconfig.AliasValues
	values   sshconfig.AliasValues
}

type sshOverview struct {
	window       fyne.Window
	sshDir       string
	setStatus    func(string)
	sshService   *sshconfig.Service
	configs      []sshConfigFile
	rows         []sshAliasRow
	activeConfig int
	drafts       []sshAliasDraft
	activeDraft  uint64
	nextDraftID  uint64
	edits        map[aliasEditKey]sshAliasEdit
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
	overview := &sshOverview{
		window: window, sshDir: sshDir, setStatus: setStatus,
		sshService: service, edits: make(map[aliasEditKey]sshAliasEdit),
	}
	overview.refresh()
	return overview
}

func (o *sshOverview) refresh() {
	activePath := o.sshService.MainConfigPath()
	if o.activeConfig >= 0 && o.activeConfig < len(o.configs) {
		activePath = o.configs[o.activeConfig].path
	}
	o.creating = false
	o.activeDraft = 0
	o.selected = ""
	if o.alias != nil {
		o.clearDraftInputCallbacks()
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
				alias := config.aliases[aliasIndex]
				key := aliasEditKey{configPath: config.path, aliasName: alias.name}
				_, modified := o.edits[key]
				o.rows = append(o.rows, sshAliasRow{
					configIndex: configIndex, aliasIndex: aliasIndex, modified: modified,
				})
			}
			for _, draft := range o.drafts {
				if draft.configPath == config.path {
					o.rows = append(o.rows, sshAliasRow{
						configIndex: configIndex, aliasIndex: -1, draftID: draft.id, isDraft: true,
					})
				}
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
