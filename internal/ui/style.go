package ui

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// Tint defines Apple HIG 9 semantic tint colors.
type Tint int

const (
	TintBlue Tint = iota
	TintGreen
	TintRed
	TintOrange
	TintTeal
	TintPurple
	TintIndigo
	TintPink
	TintGray
)

var tintHex = [...][2]string{
	TintBlue:   {"#007AFF", "#0A84FF"},
	TintGreen:  {"#34C759", "#30D158"},
	TintRed:    {"#FF3B30", "#FF453A"},
	TintOrange: {"#FF9500", "#FF9F0A"},
	TintTeal:   {"#59ADC4", "#70D7EC"},
	TintPurple: {"#AF52DE", "#BF5AF2"},
	TintIndigo: {"#5856D6", "#5E5CE6"},
	TintPink:   {"#FF2D55", "#FF375F"},
	TintGray:   {"#8E8E93", "#98989D"},
}

// C returns the appropriate tint Color based on theme mode.
func (k Tint) C(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex(tintHex[k][1])
	}
	return ui.Hex(tintHex[k][0])
}

// Palette provides semantic background and surface tones.
type Palette struct {
	Card    ui.Color
	Sidebar ui.Color
	Inset   ui.Color
	Line    ui.Color
	Hover   ui.Color
	// Stripe is the faint alternating row tint (zebra striping).
	Stripe ui.Color
}

var palettes = [2]Palette{
	// Light
	{
		Card:    ui.Hex("#F5F5F7"),
		Sidebar: ui.Hex("#EDEDF0"),
		Inset:   ui.Hex("#FFFFFF"),
		Line:    ui.RGBA(0, 0, 0, 0.08),
		Hover:   ui.RGBA(0, 0, 0, 0.05),
		Stripe:  ui.RGBA(0, 0, 0, 0.022),
	},
	// Dark
	{
		Card:    ui.Hex("#232325"),
		Sidebar: ui.Hex("#28282A"),
		Inset:   ui.Hex("#303033"),
		Line:    ui.RGBA(255, 255, 255, 0.08),
		Hover:   ui.RGBA(255, 255, 255, 0.06),
		Stripe:  ui.RGBA(255, 255, 255, 0.025),
	},
}

// Pal returns current surface palette for theme t.
func Pal(t *ui.Theme) Palette {
	if t.Dark {
		return palettes[1]
	}
	return palettes[0]
}

// BuildTheme constructs an Apple HIG base theme.
func BuildTheme(dark bool) *ui.Theme {
	if dark {
		t := ui.DarkTheme()
		t.Background = ui.Hex("#1E1E20")
		t.Surface = ui.Hex("#252528")
		t.SurfaceHover = ui.Hex("#333336")
		t.SurfacePressed = ui.Hex("#444448")
		t.Border = ui.Hex("#38383C")
		t.Text = ui.Hex("#F5F5F7")
		t.TextMuted = ui.Hex("#98989D")
		t.Accent = ui.Hex("#0A84FF")
		t.AccentHover = ui.Hex("#2E97FF")
		t.AccentPressed = ui.Hex("#006CE6")
		t.InactiveAccent = ui.Hex("#555558")
		t.Danger = ui.Hex("#FF453A")
		t.Warning = ui.Hex("#FF9F0A")
		t.Success = ui.Hex("#30D158")
		t.Selection = ui.RGBA(10, 132, 255, 0.35)
		t.Focus = ui.RGBA(10, 132, 255, 0.6)
		t.Radius = 6
		return t
	}

	t := ui.LightTheme()
	t.Background = ui.Hex("#FFFFFF")
	t.Surface = ui.Hex("#F5F5F7")
	t.SurfaceHover = ui.Hex("#E8E8EB")
	t.SurfacePressed = ui.Hex("#DCDCE0")
	t.Border = ui.Hex("#E0E0E5")
	t.Text = ui.Hex("#1D1D1F")
	t.TextMuted = ui.Hex("#6E6E73")
	t.Accent = ui.Hex("#007AFF")
	t.AccentHover = ui.Hex("#1A88FF")
	t.AccentPressed = ui.Hex("#0062CC")
	t.InactiveAccent = ui.Hex("#8E8E93")
	t.Danger = ui.Hex("#E5342A")
	t.Warning = ui.Hex("#D97E00")
	t.Success = ui.Hex("#24A148")
	t.Selection = ui.RGBA(0, 122, 255, 0.22)
	t.Focus = ui.RGBA(0, 122, 255, 0.5)
	t.Radius = 6
	return t
}

// BuildThemeColor builds a theme with a customized accent color.
func BuildThemeColor(dark bool, hexColor string) *ui.Theme {
	t := BuildTheme(dark)
	if hexColor == "" {
		return t
	}
	acc := ui.Hex(hexColor)
	t.Accent = acc
	t.AccentHover = acc.Mix(ui.RGB(255, 255, 255), 0.1)
	t.AccentPressed = acc.Mix(ui.RGB(0, 0, 0), 0.15)
	if dark {
		t.InactiveAccent = acc.Mix(ui.Hex("#555558"), 0.65)
	} else {
		t.InactiveAccent = acc.Mix(ui.Hex("#8E8E93"), 0.65)
	}
	return t
}

// ApplyAppearance sets the OS level window appearance mode.
func ApplyAppearance(appearance string) {
	switch appearance {
	case "dark":
		mygo.Theme.SetSource(mygo.ThemeDark)
	case "light":
		mygo.Theme.SetSource(mygo.ThemeLight)
	default:
		mygo.Theme.SetSource(mygo.ThemeSystem)
	}
}
