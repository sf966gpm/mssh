package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/sf966gpm/mssh/app/internal/ui/theme"
)

func (e *explorer) showFile(path, content string) {
	e.selectedFile = path
	e.editable = filepath.Base(path) == "config"
	e.fileLabel.SetText(path)
	e.fileViewer.SetText(content)
	e.viewerGutter.SetLines(editorLineCount(content))
	e.fileScroll.ScrollToTop()
	e.savedText = content
	e.finishEditing()
	e.statusLabel.SetText("Файл открыт только для чтения")
}

func (e *explorer) beginEditing() {
	if !e.editable || e.selectedFile == "" {
		return
	}
	if editorLineCount(e.fileViewer.Text) > maxEditableLines {
		e.statusLabel.SetText(fmt.Sprintf("Редактирование недоступно: больше %d строк", maxEditableLines))
		return
	}

	e.loadingText = true
	e.editor.SetText(e.fileViewer.Text)
	e.loadingText = false
	e.savedText = e.fileViewer.Text
	e.updateEditorLines()
	e.editorScroll.ScrollToTop()
	e.editing = true
	e.dirty = false
	e.fileScroll.Hide()
	e.editorScroll.Show()
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
	e.dirty = e.editor.Text != e.savedText
	withinLineLimit := e.updateEditorLines()
	if !withinLineLimit {
		e.saveButton.Disable()
		e.statusLabel.SetText(fmt.Sprintf("Лимит редактора — %d строк", maxEditableLines))
		return
	}
	if e.dirty {
		e.saveButton.Enable()
		e.statusLabel.SetText("Есть несохранённые изменения")
	} else {
		e.saveButton.Disable()
		e.statusLabel.SetText("Нет несохранённых изменений")
	}
}

func (e *explorer) showSettings() {
	settings := container.NewVBox(widget.NewLabel("Тема оформления"), e.themeSelect)
	dialog.ShowCustom("Настройки", "Закрыть", settings, e.window)
}

func themeModeLabel(mode theme.Mode) string {
	switch mode {
	case theme.ModeLight:
		return "Светлая"
	case theme.ModeDark:
		return "Тёмная"
	default:
		return "Система"
	}
}

func editorLineCount(content string) int {
	return strings.Count(content, "\n") + 1
}

func (e *explorer) updateEditorLines() bool {
	lineCount := editorLineCount(e.editor.Text)
	if lineCount > maxEditableLines {
		e.editor.SetMinRowsVisible(maxEditableLines)
		e.lineGutter.SetLines(maxEditableLines)
		e.editorScroll.Refresh()
		return false
	}
	e.editor.SetMinRowsVisible(lineCount)
	e.lineGutter.SetLines(lineCount)
	e.editorScroll.Refresh()
	return true
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
	e.editorScroll.Hide()
	e.fileScroll.Show()
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
	if editorLineCount(e.editor.Text) > maxEditableLines {
		e.statusLabel.SetText(fmt.Sprintf("Лимит редактора — %d строк", maxEditableLines))
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
	e.savedText = e.editor.Text
	e.updateEditorLines()
	e.finishEditing()
	e.statusLabel.SetText("SSH config проверен и сохранён; Match exec не выполнялся")
}
