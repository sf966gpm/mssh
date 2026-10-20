package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const initialVisibleLineCount = 100

type lineGutter struct {
	widget.BaseWidget
	lineCount      int
	firstVisible   int
	visibleCount   int
	scrollOffset   float32
	viewportHeight float32
}

func newLineGutter() *lineGutter {
	gutter := &lineGutter{
		lineCount:    1,
		visibleCount: initialVisibleLineCount,
	}
	gutter.ExtendBaseWidget(gutter)
	return gutter
}

func (g *lineGutter) SetLines(count int) {
	if count < 1 {
		count = 1
	}
	g.lineCount = count
	g.updateVisibleRange()
	g.Refresh()
}

func (g *lineGutter) SetViewport(offset, height float32) {
	g.scrollOffset = offset
	g.viewportHeight = height
	g.updateVisibleRange()
	g.Refresh()
}

func (g *lineGutter) updateVisibleRange() {
	lineHeight := g.lineHeight()
	first := 0
	lineOffset := g.lineOffset()
	if g.scrollOffset > lineOffset {
		first = int((g.scrollOffset - lineOffset) / lineHeight)
	}
	if first > 0 {
		first--
	}
	if first >= g.lineCount {
		first = g.lineCount - 1
	}
	visible := initialVisibleLineCount
	if g.viewportHeight > 0 {
		visible = int(g.viewportHeight/lineHeight) + 3
	}
	if visible < 1 {
		visible = 1
	}
	if first+visible > g.lineCount {
		visible = g.lineCount - first
	}
	g.firstVisible = first
	g.visibleCount = visible
}

func (g *lineGutter) MinSize() fyne.Size {
	textSize := g.textSize()
	digits := len(fmt.Sprint(g.lineCount))
	labelWidth := fyne.MeasureText(strings.Repeat("8", digits), textSize, fyne.TextStyle{Monospace: true}).Width
	width := labelWidth + 14
	height := g.lineOffset() + g.lineHeight()*float32(g.lineCount) + g.lineOffset()
	return fyne.NewSize(width, height)
}

func (g *lineGutter) CreateRenderer() fyne.WidgetRenderer {
	renderer := &lineGutterRenderer{
		gutter:     g,
		background: canvas.NewRectangle(theme.InputBackgroundColor()),
		separator:  canvas.NewRectangle(theme.DisabledColor()),
	}
	renderer.Refresh()
	return renderer
}

func (g *lineGutter) textSize() float32 {
	return fyne.CurrentApp().Settings().Theme().Size(theme.SizeNameText)
}

func (g *lineGutter) lineHeight() float32 {
	return fyne.MeasureText("M", g.textSize(), fyne.TextStyle{Monospace: true}).Height
}

func (g *lineGutter) lineOffset() float32 {
	return fyne.CurrentApp().Settings().Theme().Size(theme.SizeNameInnerPadding)
}

type lineGutterRenderer struct {
	gutter     *lineGutter
	background *canvas.Rectangle
	separator  *canvas.Rectangle
	numbers    []*canvas.Text
}

func (r *lineGutterRenderer) Destroy() {}

func (r *lineGutterRenderer) Layout(size fyne.Size) {
	lineHeight := r.gutter.lineHeight()
	textSize := r.gutter.textSize()
	digits := len(fmt.Sprint(r.gutter.lineCount))
	labelWidth := fyne.MeasureText(strings.Repeat("8", digits), textSize, fyne.TextStyle{Monospace: true}).Width
	r.background.Resize(size)
	r.separator.Resize(fyne.NewSize(1, size.Height))
	r.separator.Move(fyne.NewPos(size.Width-1, 0))
	for index, number := range r.numbers {
		row := r.gutter.firstVisible + index
		number.Resize(fyne.NewSize(labelWidth, lineHeight))
		y := r.gutter.lineOffset() + float32(row)*lineHeight
		number.Move(fyne.NewPos(size.Width-labelWidth-8, y))
	}
}

func (r *lineGutterRenderer) MinSize() fyne.Size {
	return r.gutter.MinSize()
}

func (r *lineGutterRenderer) Objects() []fyne.CanvasObject {
	objects := make([]fyne.CanvasObject, 0, len(r.numbers)+2)
	objects = append(objects, r.background, r.separator)
	for _, number := range r.numbers {
		objects = append(objects, number)
	}
	return objects
}

func (r *lineGutterRenderer) Refresh() {
	g := r.gutter
	for len(r.numbers) < g.visibleCount {
		number := canvas.NewText("", theme.ForegroundColor())
		number.TextStyle = fyne.TextStyle{Monospace: true}
		number.Alignment = fyne.TextAlignTrailing
		r.numbers = append(r.numbers, number)
	}
	r.background.FillColor = theme.InputBackgroundColor()
	r.separator.FillColor = theme.DisabledColor()
	r.background.Refresh()
	r.separator.Refresh()
	for index, number := range r.numbers {
		row := g.firstVisible + index
		if index >= g.visibleCount || row >= g.lineCount {
			number.Hide()
			continue
		}
		number.Show()
		number.Text = fmt.Sprintf("%*d", len(fmt.Sprint(g.lineCount)), row+1)
		number.TextSize = g.textSize()
		number.Color = theme.ForegroundColor()
		number.Refresh()
	}
	r.Layout(g.Size())
}
