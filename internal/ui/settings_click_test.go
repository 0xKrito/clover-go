package ui

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestSettingsModal_ClickResponse 复现设置弹窗点击无响应问题：
// 依次点击标签页、开关行、关闭按钮，断言内部状态随之变化。
func TestSettingsModal_ClickResponse(t *testing.T) {
	coord := createTestCoordinator(t)
	appState := NewAppState(coord)

	tt := ui.NewTester(func(c *ui.Context) {
		appState.View(c)
	}, 900, 600)
	tt.Frame()

	coord.OpenSettings()
	tt.Frame()

	// 1. 点击"外观"标签 → SettingsActiveTab 应切换为 appearance
	if err := tt.Click("外观"); err != nil {
		t.Fatalf("click 外观 tab: %v", err)
	}
	tt.Frame()
	if coord.SettingsActiveTab != "appearance" {
		t.Fatalf("tab not switched after click: got %q, want appearance", coord.SettingsActiveTab)
	}

	// 2. 点击"通用"标签回到通用页
	if err := tt.Click("通用"); err != nil {
		t.Fatalf("click 通用 tab: %v", err)
	}
	tt.Frame()
	if coord.SettingsActiveTab != "general" {
		t.Fatalf("tab not switched back: got %q", coord.SettingsActiveTab)
	}

	// 3. 点击"显示书签栏"行的开启/关闭按钮 → ShowBookmark 翻转。
	// 多个开关按钮文本同名（开启/关闭），Find 会命中第一行，
	// 因此用"显示书签栏"标签所在行的 y 坐标 + 按钮列的 x 坐标精确定位。
	before := coord.Prefs.Get().ShowBookmark
	btnText := "关闭" // 当前关闭，按钮显示"关闭"，点击后开启
	if before {
		btnText = "开启"
	}
	rowRect, ok := tt.Find("显示书签栏")
	if !ok {
		t.Fatalf("label 显示书签栏 not found")
	}
	btnRect, ok := tt.Find(btnText)
	if !ok {
		t.Fatalf("toggle button %q not found", btnText)
	}
	tt.ClickAt(btnRect.X+btnRect.W/2, rowRect.Y+rowRect.H/2)
	tt.Frame()
	after := coord.Prefs.Get().ShowBookmark
	if after == before {
		t.Fatalf("ShowBookmark not toggled: before=%v after=%v", before, after)
	}

	// 4. 点击"完成"按钮 → 弹窗应关闭
	if err := tt.Click("完成"); err != nil {
		t.Fatalf("click 完成: %v", err)
	}
	tt.Frame()
	if coord.ShowSettingsModal {
		t.Fatalf("settings modal still open after 完成 click")
	}
}

func TestToolbar_SettingsButtonClicked(t *testing.T) {
	coord := createTestCoordinator(t)
	appState := NewAppState(coord)

	tt := ui.NewTester(func(c *ui.Context) {
		appState.View(c)
	}, 900, 600)
	tt.Frame()

	if coord.ShowSettingsModal {
		t.Fatalf("expected ShowSettingsModal=false initially")
	}

	// 1. Click toolbar settings button
	if err := tt.Click("偏好设置 (⌘,)"); err != nil {
		t.Fatalf("failed to click settings button: %v", err)
	}
	tt.Frame()

	if !coord.ShowSettingsModal {
		t.Fatalf("expected ShowSettingsModal=true after clicking settings button")
	}

	// 2. Press Escape inside modal overlay
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if coord.ShowSettingsModal {
		t.Fatalf("expected ShowSettingsModal=false after pressing Escape")
	}

	// 3. Open settings via Cmd+, shortcut
	tt.Key(ui.Cmd, ui.KeyComma)
	tt.Frame()
	if !coord.ShowSettingsModal {
		t.Fatalf("expected ShowSettingsModal=true after Cmd+, shortcut")
	}

	// 4. Close via "完成" button
	if err := tt.Click("完成"); err != nil {
		t.Fatalf("failed to click 完成: %v", err)
	}
	tt.Frame()
	if coord.ShowSettingsModal {
		t.Fatalf("expected ShowSettingsModal=false after clicking 完成")
	}
}
