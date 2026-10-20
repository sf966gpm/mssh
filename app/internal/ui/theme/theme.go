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
	ModeSystem Mode = "system"
	ModeLight  Mode = "light"
	ModeDark   Mode = "dark"
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
	switch mode {
	case ModeLight:
		light := fyneTheme.VariantLight
		variant = &light
	case ModeDark:
		dark := fyneTheme.VariantDark
		variant = &dark
	}

	return &Theme{
		base:    fyneTheme.DefaultTheme(),
		palette: DefaultPalette(),
		variant: variant,
	}
}

// DefaultPalette returns MSSH's blue accent colors for light and dark variants.
func DefaultPalette() Palette {
	return Palette{
		Light: map[fyne.ThemeColorName]color.Color{
			fyneTheme.ColorNamePrimary:   color.NRGBA{R: 0x25, G: 0x63, B: 0xeb, A: 0xff},
			fyneTheme.ColorNameFocus:     color.NRGBA{R: 0x25, G: 0x63, B: 0xeb, A: 0xff},
			fyneTheme.ColorNameHyperlink: color.NRGBA{R: 0x25, G: 0x63, B: 0xeb, A: 0xff},
			fyneTheme.ColorNameSelection: color.NRGBA{R: 0x25, G: 0x63, B: 0xeb, A: 0x40},
			fyneTheme.ColorNameDisabled:  color.NRGBA{R: 0x4b, G: 0x55, B: 0x63, A: 0xff},
			fyneTheme.ColorNameDisabledButton: color.NRGBA{
				R: 0xe5,
				G: 0xe7,
				B: 0xeb,
				A: 0xff,
			},
		},
		Dark: map[fyne.ThemeColorName]color.Color{
			fyneTheme.ColorNamePrimary:   color.NRGBA{R: 0x60, G: 0xa5, B: 0xfa, A: 0xff},
			fyneTheme.ColorNameFocus:     color.NRGBA{R: 0x60, G: 0xa5, B: 0xfa, A: 0xff},
			fyneTheme.ColorNameHyperlink: color.NRGBA{R: 0x60, G: 0xa5, B: 0xfa, A: 0xff},
			fyneTheme.ColorNameSelection: color.NRGBA{R: 0x60, G: 0xa5, B: 0xfa, A: 0x66},
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
	return t.base.Size(name)
}
