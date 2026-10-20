package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	msshTheme "github.com/sf966gpm/mssh/app/internal/ui/theme"
)

const maxEditableLines = 10_000

type explorer struct {
	application  fyne.App
	window       fyne.Window
	rootDir      string
	currentDir   string
	entries      []os.DirEntry
	selectedFile string
	editable     bool
	dirty        bool
	loadingText  bool
	editing      bool
	savedText    string

	pathLabel    *widget.Label
	fileLabel    *widget.Label
	statusLabel  *widget.Label
	fileList     *widget.List
	fileViewer   *widget.Label
	editor       *widget.Entry
	viewerGutter *lineGutter
	lineGutter   *lineGutter
	editorScroll *container.Scroll
	fileScroll   *container.Scroll
	themeSelect  *widget.Select
	settings     *widget.Button
	editButton   *widget.Button
	saveButton   *widget.Button
	cancelButton *widget.Button
	fileBody     *fyne.Container
}

func newExplorer(application fyne.App, window fyne.Window, mode msshTheme.Mode) *explorer {
	rootDir := startDirectory()
	if absolutePath, err := filepath.Abs(rootDir); err == nil {
		rootDir = absolutePath
	}
	if resolvedPath, err := filepath.EvalSymlinks(rootDir); err == nil {
		rootDir = resolvedPath
	}

	e := &explorer{
		window:       window,
		application:  application,
		rootDir:      rootDir,
		currentDir:   rootDir,
		entries:      []os.DirEntry{},
		pathLabel:    widget.NewLabel(""),
		fileLabel:    widget.NewLabel("Выберите файл"),
		statusLabel:  widget.NewLabel(""),
		fileViewer:   widget.NewLabel("Выберите текстовый файл слева"),
		editor:       widget.NewMultiLineEntry(),
		viewerGutter: newLineGutter(),
		lineGutter:   newLineGutter(),
	}
	e.pathLabel.SetText(rootDir)
	e.fileViewer.Wrapping = fyne.TextWrapOff
	e.fileViewer.TextStyle = fyne.TextStyle{Monospace: true}
	e.fileViewer.Selectable = true
	e.editor.Wrapping = fyne.TextWrapOff
	// The outer scroll container owns scrolling so the line gutter stays aligned.
	e.editor.Scroll = fyne.ScrollNone
	e.editor.TextStyle = fyne.TextStyle{Monospace: true}
	e.editor.SetMinRowsVisible(1)
	e.editor.OnChanged = e.onEditorChanged
	e.editorScroll = container.NewScroll(container.NewBorder(nil, nil, e.lineGutter, nil, e.editor))
	e.editorScroll.OnScrolled = func(position fyne.Position) {
		e.lineGutter.SetViewport(position.Y, e.editorScroll.Size().Height)
	}
	e.editorScroll.Hide()
	e.themeSelect = widget.NewSelect([]string{"Система", "Светлая", "Тёмная"}, e.setThemeMode)
	e.themeSelect.SetSelected(themeModeLabel(mode))
	e.settings = widget.NewButtonWithIcon("", theme.SettingsIcon(), e.showSettings)
	e.editButton = widget.NewButton("Редактировать", e.beginEditing)
	e.editButton.Hide()
	e.saveButton = widget.NewButton("Сохранить", e.saveFile)
	e.saveButton.Disable()
	e.saveButton.Hide()
	e.cancelButton = widget.NewButton("Отмена", e.cancelEditing)
	e.cancelButton.Hide()
	e.fileScroll = container.NewScroll(container.NewBorder(nil, nil, e.viewerGutter, nil, e.fileViewer))
	e.fileScroll.OnScrolled = func(position fyne.Position) {
		e.viewerGutter.SetViewport(position.Y, e.fileScroll.Size().Height)
	}
	e.fileBody = container.NewStack(e.fileScroll, e.editorScroll)
	e.fileList = widget.NewList(
		func() int { return e.entryCount() },
		func() fyne.CanvasObject {
			return container.NewHBox(widget.NewIcon(theme.FileIcon()), widget.NewLabel(""))
		},
		e.updateListItem,
	)
	e.fileList.OnSelected = e.openEntry
	e.loadDirectory(rootDir)
	return e
}

func (e *explorer) setThemeMode(label string) {
	mode := msshTheme.ModeSystem
	switch label {
	case "Светлая":
		mode = msshTheme.ModeLight
	case "Тёмная":
		mode = msshTheme.ModeDark
	}
	e.application.Preferences().SetString("theme_mode", string(mode))
	e.application.Settings().SetTheme(msshTheme.NewForMode(mode))
}

func startDirectory() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".ssh")
}

func (e *explorer) updateListItem(id widget.ListItemID, object fyne.CanvasObject) {
	if e.hasParentEntry() {
		if id == 0 {
			row := object.(*fyne.Container)
			row.Objects[0].(*widget.Icon).SetResource(theme.MoveUpIcon())
			row.Objects[1].(*widget.Label).SetText("..")
			return
		}
		id--
	}

	entry := e.entries[id]
	name := entry.Name()
	icon := theme.FileIcon()
	if entry.IsDir() {
		name += string(filepath.Separator)
		icon = theme.FolderIcon()
	} else if entry.Type()&os.ModeSymlink != 0 {
		name += " @"
	}

	row := object.(*fyne.Container)
	row.Objects[0].(*widget.Icon).SetResource(icon)
	row.Objects[1].(*widget.Label).SetText(name)
}

func (e *explorer) openEntry(id widget.ListItemID) {
	if e.hasParentEntry() {
		if id == 0 {
			e.openParentDirectory()
			return
		}
		id--
	}
	if id < 0 || id >= len(e.entries) {
		return
	}

	entry := e.entries[id]
	if entry.Type()&os.ModeSymlink != 0 {
		e.statusLabel.SetText("Символические ссылки не открываются")
		return
	}
	path := filepath.Join(e.currentDir, entry.Name())
	if !e.isInsideRoot(path) {
		e.statusLabel.SetText("Нельзя выйти за пределы начальной папки")
		return
	}
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		e.statusLabel.SetText(fmt.Sprintf("Не удалось открыть %s: %v", path, err))
		return
	}
	if !e.isInsideRoot(resolvedPath) {
		e.statusLabel.SetText("Нельзя выйти за пределы начальной папки")
		return
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		e.statusLabel.SetText(fmt.Sprintf("Не удалось открыть %s: %v", path, err))
		return
	}
	if info.IsDir() {
		e.withDiscardedChanges(func() {
			e.loadDirectory(path)
		})
		return
	}

	content, err := readTextFile(resolvedPath)
	if err != nil {
		e.statusLabel.SetText(err.Error())
		return
	}
	e.withDiscardedChanges(func() {
		e.showFile(resolvedPath, content)
	})
}

func (e *explorer) loadDirectory(path string) {
	if !e.isInsideRoot(path) {
		e.statusLabel.SetText("Нельзя выйти за пределы начальной папки")
		return
	}
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		e.statusLabel.SetText(fmt.Sprintf("Не удалось открыть папку %s: %v", path, err))
		return
	}
	if !e.isInsideRoot(resolvedPath) {
		e.statusLabel.SetText("Нельзя выйти за пределы начальной папки")
		return
	}

	entries, err := os.ReadDir(resolvedPath)
	if err != nil {
		e.statusLabel.SetText(fmt.Sprintf("Не удалось открыть папку %s: %v", resolvedPath, err))
		return
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})

	e.currentDir = resolvedPath
	e.entries = entries
	e.pathLabel.SetText(resolvedPath)
	e.clearFile()
	e.fileList.UnselectAll()
	e.fileList.Refresh()
	e.statusLabel.SetText(fmt.Sprintf("Элементов: %d", len(entries)))
}

func (e *explorer) clearFile() {
	e.selectedFile = ""
	e.editable = false
	e.finishEditing()
	e.fileLabel.SetText("Выберите файл")
	e.fileViewer.SetText("Выберите текстовый файл слева")
	e.viewerGutter.SetLines(1)
	e.fileScroll.ScrollToTop()
}

func (e *explorer) hasParentEntry() bool {
	return filepath.Clean(e.currentDir) != filepath.Clean(e.rootDir)
}

func (e *explorer) entryCount() int {
	count := len(e.entries)
	if e.hasParentEntry() {
		count++
	}
	return count
}

func (e *explorer) openParentDirectory() {
	if !e.hasParentEntry() {
		return
	}
	e.withDiscardedChanges(func() {
		e.loadDirectory(filepath.Dir(e.currentDir))
	})
}

func (e *explorer) isInsideRoot(path string) bool {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	relativePath, err := filepath.Rel(e.rootDir, absolutePath)
	if err != nil {
		return false
	}
	return relativePath == "." || (relativePath != ".." && !strings.HasPrefix(relativePath, ".."+string(filepath.Separator)))
}
