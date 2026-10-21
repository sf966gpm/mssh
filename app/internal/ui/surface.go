package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type roundedSurface struct {
	widget.BaseWidget
	content fyne.CanvasObject
}

func newRoundedSurface(content fyne.CanvasObject) fyne.CanvasObject {
	surface := &roundedSurface{content: container.NewPadded(content)}
	surface.ExtendBaseWidget(surface)
	return surface
}

func (s *roundedSurface) CreateRenderer() fyne.WidgetRenderer {
	background := canvas.NewRectangle(theme.Color(theme.ColorNameHeaderBackground))
	background.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	background.StrokeWidth = 1
	background.CornerRadius = theme.Size(theme.SizeNameCardRadius)
	return &roundedSurfaceRenderer{
		surface:    s,
		background: background,
		content:    s.content,
	}
}

type roundedSurfaceRenderer struct {
	surface    *roundedSurface
	background *canvas.Rectangle
	content    fyne.CanvasObject
}

func (r *roundedSurfaceRenderer) Destroy() {}

func (r *roundedSurfaceRenderer) Layout(size fyne.Size) {
	r.background.Resize(size)
	r.content.Resize(size)
}

func (r *roundedSurfaceRenderer) MinSize() fyne.Size {
	return r.content.MinSize()
}

func (r *roundedSurfaceRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.background, r.content}
}

func (r *roundedSurfaceRenderer) Refresh() {
	variant := fyne.CurrentApp().Settings().ThemeVariant()
	currentTheme := r.surface.Theme()
	r.background.FillColor = currentTheme.Color(theme.ColorNameHeaderBackground, variant)
	r.background.StrokeColor = currentTheme.Color(theme.ColorNameInputBorder, variant)
	r.background.StrokeWidth = 1
	r.background.CornerRadius = currentTheme.Size(theme.SizeNameCardRadius)
	r.background.Refresh()
	r.content.Refresh()
}
