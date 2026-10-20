package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const maxViewFileSize = 2 << 20
const sshConfigValidationTimeout = 5 * time.Second

type explorer struct {
	window       fyne.Window
	rootDir      string
	currentDir   string
	entries      []os.DirEntry
	selectedFile string
	editable     bool
	dirty        bool
	loadingText  bool
	editing      bool

	pathLabel    *widget.Label
	fileLabel    *widget.Label
	statusLabel  *widget.Label
	fileList     *widget.List
	fileViewer   *widget.Label
	editor       *widget.Entry
	editButton   *widget.Button
	saveButton   *widget.Button
	cancelButton *widget.Button
	fileBody     *fyne.Container
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
	rootDir := startDirectory()
	if absolutePath, err := filepath.Abs(rootDir); err == nil {
		rootDir = absolutePath
	}
	if resolvedPath, err := filepath.EvalSymlinks(rootDir); err == nil {
		rootDir = resolvedPath
	}

	e := &explorer{
		window:      window,
		rootDir:     rootDir,
		currentDir:  rootDir,
		entries:     []os.DirEntry{},
		pathLabel:   widget.NewLabel(""),
		fileLabel:   widget.NewLabel("Выберите файл"),
		statusLabel: widget.NewLabel(""),
		fileViewer:  widget.NewLabel("Выберите текстовый файл слева"),
		editor:      widget.NewMultiLineEntry(),
	}
	e.fileViewer.Wrapping = fyne.TextWrapOff
	e.fileViewer.TextStyle = fyne.TextStyle{Monospace: true}
	e.editor.SetMinRowsVisible(20)
	e.editor.Wrapping = fyne.TextWrapOff
	e.editor.TextStyle = fyne.TextStyle{Monospace: true}
	e.editor.Hide()
	e.editor.OnChanged = e.onEditorChanged
	e.editButton = widget.NewButton("Редактировать", e.beginEditing)
	e.editButton.Hide()
	e.saveButton = widget.NewButton("Сохранить", e.saveFile)
	e.saveButton.Disable()
	e.saveButton.Hide()
	e.cancelButton = widget.NewButton("Отмена", e.cancelEditing)
	e.cancelButton.Hide()
	e.fileBody = container.NewStack(e.fileViewer, e.editor)
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

func (e *explorer) content() fyne.CanvasObject {
	leftPane := container.NewBorder(e.pathLabel, nil, nil, nil, e.fileList)
	fileActions := container.NewHBox(e.editButton, e.saveButton, e.cancelButton)
	rightHeader := container.NewBorder(nil, nil, nil, fileActions, e.fileLabel)
	rightPane := container.NewBorder(rightHeader, nil, nil, nil, container.NewScroll(e.fileBody))
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

	content, err := readTextFile(resolvedPath, info.Size())
	if err != nil {
		e.statusLabel.SetText(err.Error())
		return
	}
	e.withDiscardedChanges(func() {
		e.showFile(resolvedPath, content)
	})
}

func (e *explorer) showFile(path, content string) {
	e.selectedFile = path
	e.editable = filepath.Base(path) == "config"
	e.fileLabel.SetText(path)
	e.fileViewer.SetText(content)
	e.editor.SetText("")
	e.finishEditing()
	if e.editable {
		e.editButton.Show()
	} else {
		e.editButton.Hide()
	}
	e.statusLabel.SetText("Файл открыт только для чтения")
}

func (e *explorer) beginEditing() {
	if !e.editable || e.selectedFile == "" {
		return
	}

	e.loadingText = true
	e.editor.SetText(e.fileViewer.Text)
	e.loadingText = false
	e.editing = true
	e.dirty = false
	e.fileViewer.Hide()
	e.editor.Show()
	e.editButton.Hide()
	e.saveButton.Disable()
	e.saveButton.Show()
	e.cancelButton.Show()
	e.statusLabel.SetText("Редактирование SSH config")
}

func (e *explorer) onEditorChanged(string) {
	if !e.editing || e.loadingText {
		return
	}
	e.dirty = true
	e.saveButton.Enable()
	e.statusLabel.SetText("Есть несохранённые изменения")
}

func (e *explorer) cancelEditing() {
	e.withDiscardedChanges(func() {
		e.finishEditing()
		e.statusLabel.SetText("Изменения отменены")
	})
}

func (e *explorer) finishEditing() {
	e.editing = false
	e.dirty = false
	e.editor.Hide()
	e.fileViewer.Show()
	e.saveButton.Disable()
	e.saveButton.Hide()
	e.cancelButton.Hide()
	if e.editable {
		e.editButton.Show()
	} else {
		e.editButton.Hide()
	}
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
			if discard {
				next()
			}
		},
		e.window,
	)
}

func (e *explorer) saveFile() {
	if !e.editing || !e.dirty || !e.editable || e.selectedFile == "" {
		return
	}
	if len(e.editor.Text) > maxViewFileSize {
		e.statusLabel.SetText("Текст больше 2 МиБ, сохранить его нельзя")
		return
	}

	resolvedPath, err := filepath.EvalSymlinks(e.selectedFile)
	if err != nil || !e.isInsideRoot(resolvedPath) || filepath.Base(resolvedPath) != "config" {
		e.statusLabel.SetText("Путь к SSH config изменился, сохранение отменено")
		return
	}
	info, err := os.Lstat(e.selectedFile)
	if err != nil {
		e.statusLabel.SetText(fmt.Sprintf("Не удалось проверить файл: %v", err))
		return
	}
	if !info.Mode().IsRegular() {
		e.statusLabel.SetText("SSH config больше не является обычным файлом")
		return
	}

	content := []byte(e.editor.Text)
	if err := validateSSHConfig(e.selectedFile, content); err != nil {
		e.statusLabel.SetText(fmt.Sprintf("Ошибка синтаксиса SSH config: %v", err))
		return
	}
	if err := replaceFile(e.selectedFile, content, info.Mode().Perm()); err != nil {
		e.statusLabel.SetText(fmt.Sprintf("Не удалось сохранить SSH config: %v", err))
		return
	}

	e.fileViewer.SetText(e.editor.Text)
	e.finishEditing()
	e.statusLabel.SetText("SSH config проверен и сохранён")
}

func validateSSHConfig(path string, content []byte) error {
	tempFile, err := os.CreateTemp(filepath.Dir(path), ".mssh-config-check-*")
	if err != nil {
		return fmt.Errorf("не удалось создать временный файл проверки: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if _, err := tempFile.Write(content); err != nil {
		tempFile.Close()
		return fmt.Errorf("не удалось записать временный файл проверки: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("не удалось закрыть временный файл проверки: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), sshConfigValidationTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, "ssh", "-G", "-F", tempPath, "mssh-config-validation.invalid")
	output, err := command.CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("проверка SSH config превысила лимит времени %s", sshConfigValidationTimeout)
		}
		if message := strings.TrimSpace(string(output)); message != "" {
			return fmt.Errorf("%s", message)
		}
		return fmt.Errorf("не удалось запустить ssh -G: %w", err)
	}
	return nil
}

func replaceFile(path string, content []byte, mode os.FileMode) error {
	tempFile, err := os.CreateTemp(filepath.Dir(path), ".mssh-config-save-*")
	if err != nil {
		return fmt.Errorf("не удалось создать временный файл: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if _, err := tempFile.Write(content); err != nil {
		tempFile.Close()
		return fmt.Errorf("не удалось записать временный файл: %w", err)
	}
	if err := tempFile.Chmod(mode.Perm()); err != nil {
		tempFile.Close()
		return fmt.Errorf("не удалось установить права файла: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		tempFile.Close()
		return fmt.Errorf("не удалось синхронизировать файл: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("не удалось закрыть временный файл: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("не удалось заменить файл: %w", err)
	}
	return nil
}

func readTextFile(path string, size int64) (string, error) {
	if size > maxViewFileSize {
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
