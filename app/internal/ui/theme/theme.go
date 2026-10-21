// Package theme holds MSSH's Fyne theme and color overrides.
package theme

import (
	"image/color"

	"fyne.io/fyne/v2"
	fyneTheme "fyne.io/fyne/v2/theme"
)

// Palette contains optional Fyne color overrides for light and dark variants.
// Missing colors use the corresponding Fyne default.
type Palette struct {
	Light map[fyne.ThemeColorName]color.Color
	Dark  map[fyne.ThemeColorName]color.Color
}

// Mode is the theme variant MSSH should use.
type Mode string

const (
	ModeSystem        Mode = "system"
	ModeLight         Mode = "light"
	ModeDark          Mode = "dark"
	ModeJetBrainsDark Mode = "jetbrains-dark"
)

// Theme applies MSSH color overrides and delegates other styling to Fyne.
type Theme struct {
	base    fyne.Theme
	palette Palette
	variant *fyne.ThemeVariant
}

// New returns the MSSH theme using the system theme variant.
func New() *Theme {
	return NewForMode(ModeSystem)
}

// NewForMode returns the MSSH theme using the requested variant.
func NewForMode(mode Mode) *Theme {
	var variant *fyne.ThemeVariant
	palette := DefaultPalette()
	switch mode {
	case ModeLight:
		light := fyneTheme.VariantLight
		variant = &light
	case ModeDark:
		dark := fyneTheme.VariantDark
		variant = &dark
	case ModeJetBrainsDark:
		dark := fyneTheme.VariantDark
		variant = &dark
		palette = JetBrainsDarkPalette()
	}

	return &Theme{
		base:    fyneTheme.DefaultTheme(),
		palette: palette,
		variant: variant,
	}
}

// JetBrainsDarkPalette returns a Darcula-inspired palette for the dark variant.
func JetBrainsDarkPalette() Palette {
	return Palette{
		Dark: map[fyne.ThemeColorName]color.Color{
			fyneTheme.ColorNameBackground:          color.NRGBA{R: 0x2b, G: 0x2b, B: 0x2b, A: 0xff},
			fyneTheme.ColorNameForeground:          color.NRGBA{R: 0xa9, G: 0xb7, B: 0xc6, A: 0xff},
			fyneTheme.ColorNamePrimary:             color.NRGBA{R: 0x58, G: 0x9d, B: 0xf6, A: 0xff},
			fyneTheme.ColorNameFocus:               color.NRGBA{R: 0x58, G: 0x9d, B: 0xf6, A: 0xff},
			fyneTheme.ColorNameHyperlink:           color.NRGBA{R: 0x58, G: 0x9d, B: 0xf6, A: 0xff},
			fyneTheme.ColorNameSelection:           color.NRGBA{R: 0x21, G: 0x42, B: 0x83, A: 0xff},
			fyneTheme.ColorNameHover:               color.NRGBA{R: 0x32, G: 0x32, B: 0x32, A: 0xff},
			fyneTheme.ColorNamePressed:             color.NRGBA{R: 0x36, G: 0x58, B: 0x80, A: 0xff},
			fyneTheme.ColorNameButton:              color.NRGBA{R: 0x4c, G: 0x50, B: 0x52, A: 0xff},
			fyneTheme.ColorNameDisabledButton:      color.NRGBA{R: 0x3b, G: 0x3d, B: 0x3f, A: 0xff},
			fyneTheme.ColorNameDisabled:            color.NRGBA{R: 0x78, G: 0x78, B: 0x78, A: 0xff},
			fyneTheme.ColorNameInputBackground:     color.NRGBA{R: 0x3c, G: 0x3f, B: 0x41, A: 0xff},
			fyneTheme.ColorNameInputBorder:         color.NRGBA{R: 0x55, G: 0x59, B: 0x5c, A: 0xff},
			fyneTheme.ColorNameHeaderBackground:    color.NRGBA{R: 0x31, G: 0x33, B: 0x35, A: 0xff},
			fyneTheme.ColorNameMenuBackground:      color.NRGBA{R: 0x31, G: 0x33, B: 0x35, A: 0xff},
			fyneTheme.ColorNameOverlayBackground:   color.NRGBA{R: 0x31, G: 0x33, B: 0x35, A: 0xff},
			fyneTheme.ColorNameSeparator:           color.NRGBA{R: 0x45, G: 0x49, B: 0x4a, A: 0xff},
			fyneTheme.ColorNamePlaceHolder:         color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff},
			fyneTheme.ColorNameShadow:              color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x66},
			fyneTheme.ColorNameScrollBar:           color.NRGBA{R: 0x5a, G: 0x5d, B: 0x5e, A: 0xff},
			fyneTheme.ColorNameScrollBarBackground: color.NRGBA{R: 0x2b, G: 0x2b, B: 0x2b, A: 0xff},
		},
	}
}

// DefaultPalette returns MSSH's blue accent colors for light and dark variants.
func DefaultPalette() Palette {
	return Palette{
		Light: map[fyne.ThemeColorName]color.Color{
			fyneTheme.ColorNamePrimary:          color.NRGBA{R: 0x25, G: 0x63, B: 0xeb, A: 0xff},
			fyneTheme.ColorNameFocus:            color.NRGBA{R: 0x25, G: 0x63, B: 0xeb, A: 0xff},
			fyneTheme.ColorNameHyperlink:        color.NRGBA{R: 0x25, G: 0x63, B: 0xeb, A: 0xff},
			fyneTheme.ColorNameSelection:        color.NRGBA{R: 0x25, G: 0x63, B: 0xeb, A: 0x40},
			fyneTheme.ColorNameDisabled:         color.NRGBA{R: 0x4b, G: 0x55, B: 0x63, A: 0xff},
			fyneTheme.ColorNameHeaderBackground: color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
			fyneTheme.ColorNameInputBorder:      color.NRGBA{R: 0xd1, G: 0xd5, B: 0xdb, A: 0xff},
			fyneTheme.ColorNameDisabledButton: color.NRGBA{
				R: 0xe5,
				G: 0xe7,
				B: 0xeb,
				A: 0xff,
			},
		},
		Dark: map[fyne.ThemeColorName]color.Color{
			fyneTheme.ColorNamePrimary:          color.NRGBA{R: 0x60, G: 0xa5, B: 0xfa, A: 0xff},
			fyneTheme.ColorNameFocus:            color.NRGBA{R: 0x60, G: 0xa5, B: 0xfa, A: 0xff},
			fyneTheme.ColorNameHyperlink:        color.NRGBA{R: 0x60, G: 0xa5, B: 0xfa, A: 0xff},
			fyneTheme.ColorNameSelection:        color.NRGBA{R: 0x60, G: 0xa5, B: 0xfa, A: 0x66},
			fyneTheme.ColorNameHeaderBackground: color.NRGBA{R: 0x29, G: 0x2d, B: 0x33, A: 0xff},
			fyneTheme.ColorNameInputBorder:      color.NRGBA{R: 0x3f, G: 0x46, B: 0x50, A: 0xff},
		},
	}
}

// NewWithPalette returns the MSSH theme with the supplied color overrides.
func NewWithPalette(palette Palette) *Theme {
	return &Theme{
		base:    fyneTheme.DefaultTheme(),
		palette: palette,
	}
}

func (t *Theme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if t.variant != nil {
		variant = *t.variant
	}
	colors := t.palette.Light
	if variant == fyneTheme.VariantDark {
		colors = t.palette.Dark
	}
	if override, ok := colors[name]; ok && override != nil {
		return override
	}
	return t.base.Color(name, variant)
}

func (t *Theme) Font(style fyne.TextStyle) fyne.Resource {
	return t.base.Font(style)
}

func (t *Theme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(name)
}

func (t *Theme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case fyneTheme.SizeNameInnerWindowRadius:
		return 10
	case fyneTheme.SizeNameInputRadius:
		return 6
	case fyneTheme.SizeNameButtonRadius:
		return 7
	case fyneTheme.SizeNameSelectionRadius:
		return 5
	case fyneTheme.SizeNameScrollBarRadius:
		return 4
	case fyneTheme.SizeNameCardRadius:
		return 10
	case fyneTheme.SizeNameDialogRadius:
		return 12
	case fyneTheme.SizeNamePopupRadius:
		return 8
	case fyneTheme.SizeNameMenuRadius:
		return 6
	}
	return t.base.Size(name)
}
