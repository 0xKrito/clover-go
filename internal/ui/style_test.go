package ui

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestBuildTheme(t *testing.T) {
	dark := BuildTheme(true)
	if !dark.Dark {
		t.Errorf("expected dark theme Dark=true")
	}
	if dark.InactiveAccent == (ui.Color{}) {
		t.Errorf("expected dark theme InactiveAccent to be set")
	}

	light := BuildTheme(false)
	if light.Dark {
		t.Errorf("expected light theme Dark=false")
	}
	if light.InactiveAccent == (ui.Color{}) {
		t.Errorf("expected light theme InactiveAccent to be set")
	}

	custom := BuildThemeColor(true, "#FF3B30")
	if custom.Accent.R == 0 {
		t.Errorf("expected customized red accent")
	}
	if custom.InactiveAccent == (ui.Color{}) {
		t.Errorf("expected custom theme InactiveAccent to be set")
	}
}

func TestTintColors(t *testing.T) {
	th := BuildTheme(false)
	c := TintBlue.C(th)
	if c.B == 0 {
		t.Errorf("expected blue tint to have non-zero blue channel")
	}
}
