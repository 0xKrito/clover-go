package ui

import (
	"testing"
)

func TestBuildTheme(t *testing.T) {
	dark := BuildTheme(true)
	if !dark.Dark {
		t.Errorf("expected dark theme Dark=true")
	}

	light := BuildTheme(false)
	if light.Dark {
		t.Errorf("expected light theme Dark=false")
	}

	custom := BuildThemeColor(true, "#FF3B30")
	if custom.Accent.R == 0 {
		t.Errorf("expected customized red accent")
	}
}

func TestTintColors(t *testing.T) {
	th := BuildTheme(false)
	c := TintBlue.C(th)
	if c.B == 0 {
		t.Errorf("expected blue tint to have non-zero blue channel")
	}
}
