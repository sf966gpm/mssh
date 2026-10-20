package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

const maxEditableFileSize = 2 << 20

type explorer struct {
	window       fyne.Window
	currentDir   string
	entries      []os.DirEntry
	selectedFile string
	dirty        bool
	loadingText  bool

	pathLabel   *widget.Label
	fileLabel   *widget.Label
	statusLabel *widget.Label
	fileList    *widget.List
	editor      *widget.Entry
	saveButton  *widget.Button
}

func NewWindow(application fyne.App) fyne.Window {
	window := application.NewWindow("MSSH")
	explorer := newExplorer(window)

	window.SetContent(explorer.content())
	window.Resize(fyne.NewSize(1200, 800))
	window.SetFixedSize(true)

	return window
}

func newExplorer(window fyne.Window) *explorer {
	e := &explorer{
		window:      window,
		entries:     []os.DirEntry{},
		pathLabel:   widget.NewLabel(""),
		fileLabel:   widget.NewLabel("Выберите файл"),
		statusLabel: widget.NewLabel(""),
		editor:      widget.NewMultiLineEntry(),
	}
	e.editor.SetPlaceHolder("Выберите текстовый файл слева")
	e.editor.SetMinRowsVisible(20)
	e.editor.Wrapping = fyne.TextWrapOff
	e.editor.TextStyle = fyne.TextStyle{Monospace: true}
	e.editor.Disable()
	e.editor.OnChanged = e.onEditorChanged

	e.saveButton = widget.NewButton("Сохранить", e.saveFile)
	e.saveButton.Disable()
	e.fileList = widget.NewList(
		func() int { return len(e.entries) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		e.updateListItem,
	)
	e.fileList.OnSelected = e.openEntry

	e.loadDirectory(startDirectory())

	return e
}

func (e *explorer) content() fyne.CanvasObject {
	upButton := widget.NewButton("Вверх", e.openParentDirectory)
	leftHeader := container.NewBorder(nil, nil, nil, upButton, e.pathLabel)
	leftPane := container.NewBorder(leftHeader, nil, nil, nil, e.fileList)

	rightHeader := container.NewBorder(nil, nil, nil, e.saveButton, e.fileLabel)
	rightPane := container.NewBorder(rightHeader, nil, nil, nil, container.NewScroll(e.editor))

	split := container.NewHSplit(leftPane, rightPane)
	split.SetOffset(0.5)

	return container.NewBorder(nil, e.statusLabel, nil, nil, split)
}

func startDirectory() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}

	sshDirectory := filepath.Join(home, ".ssh")
	info, err := os.Stat(sshDirectory)
	if err == nil && info.IsDir() {
		return sshDirectory
	}

	return home
}

func (e *explorer) updateListItem(id widget.ListItemID, object fyne.CanvasObject) {
	entry := e.entries[id]
	name := entry.Name()
	if entry.IsDir() {
		name += string(filepath.Separator)
	} else if entry.Type()&os.ModeSymlink != 0 {
		name += " @"
	}

	object.(*widget.Label).SetText(name)
}

func (e *explorer) openEntry(id widget.ListItemID) {
	if id < 0 || id >= len(e.entries) {
		return
	}

	path := filepath.Join(e.currentDir, e.entries[id].Name())
	info, err := os.Stat(path)
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

	content, err := readTextFile(path, info.Size())
	if err != nil {
		e.statusLabel.SetText(err.Error())
		return
	}

	e.withDiscardedChanges(func() {
		e.showFile(path, content)
	})
}

func readTextFile(path string, size int64) (string, error) {
	if size > maxEditableFileSize {
		return "", fmt.Errorf("файл %s больше 2 МиБ и не открыт", path)
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("не удалось прочитать %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s не является обычным файлом", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("не удалось прочитать %s: %w", path, err)
	}
	if !utf8.Valid(data) {
		return "", fmt.Errorf("%s не похож на текстовый файл", path)
	}

	return string(data), nil
}

func (e *explorer) showFile(path, content string) {
	e.selectedFile = path
	e.fileLabel.SetText(path)
	e.editor.Enable()
	e.loadingText = true
	e.editor.SetText(content)
	e.loadingText = false
	e.dirty = false
	e.saveButton.Disable()
	e.statusLabel.SetText("Файл открыт")
}

func (e *explorer) onEditorChanged(string) {
	if e.loadingText || e.selectedFile == "" {
		return
	}

	e.dirty = true
	e.saveButton.Enable()
	e.statusLabel.SetText("Есть несохранённые изменения")
}

func (e *explorer) saveFile() {
	if e.selectedFile == "" {
		return
	}
	if len(e.editor.Text) > maxEditableFileSize {
		e.statusLabel.SetText("Текст больше 2 МиБ, сохранить его нельзя")
		return
	}

	info, err := os.Stat(e.selectedFile)
	if err != nil {
		e.statusLabel.SetText(fmt.Sprintf("Не удалось сохранить файл: %v", err))
		return
	}
	if !info.Mode().IsRegular() {
		e.statusLabel.SetText("Выбранный путь больше не является обычным файлом")
		return
	}

	if err := os.WriteFile(e.selectedFile, []byte(e.editor.Text), info.Mode().Perm()); err != nil {
		e.statusLabel.SetText(fmt.Sprintf("Не удалось сохранить файл: %v", err))
		return
	}

	e.dirty = false
	e.saveButton.Disable()
	e.statusLabel.SetText("Файл сохранён")
}

func (e *explorer) openParentDirectory() {
	parent := filepath.Dir(e.currentDir)
	if parent == e.currentDir {
		return
	}

	e.withDiscardedChanges(func() {
		e.loadDirectory(parent)
	})
}

func (e *explorer) loadDirectory(path string) {
	entries, err := os.ReadDir(path)
	if err != nil {
		e.statusLabel.SetText(fmt.Sprintf("Не удалось открыть папку %s: %v", path, err))
		return
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})

	e.currentDir = path
	e.entries = entries
	e.pathLabel.SetText(path)
	e.clearFile()
	e.fileList.Refresh()
	e.statusLabel.SetText(fmt.Sprintf("Элементов: %d", len(entries)))
}

func (e *explorer) clearFile() {
	e.selectedFile = ""
	e.fileLabel.SetText("Выберите текстовый файл")
	e.loadingText = true
	e.editor.SetText("")
	e.loadingText = false
	e.editor.Disable()
	e.dirty = false
	e.saveButton.Disable()
}

func (e *explorer) withDiscardedChanges(next func()) {
	if !e.dirty {
		next()
		return
	}

	dialog.ShowConfirm(
		"Несохранённые изменения",
		"Отбросить изменения и продолжить?",
		func(discard bool) {
			if !discard {
				return
			}

			next()
		},
		e.window,
	)
}
