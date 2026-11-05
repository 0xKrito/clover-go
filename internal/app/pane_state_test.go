package app

import (
	"testing"

	"github.com/bighaidao/clover-go/internal/domain"
	"github.com/egoist/mygo/ui"
)

func TestClickSelectRow(t *testing.T) {
	p := NewPaneState("p1", "/tmp/box")
	p.DisplayEntries = []domain.FileInfo{
		{Path: "/tmp/box/a.txt"},
		{Path: "/tmp/box/b.txt"},
		{Path: "/tmp/box/c.txt"},
	}

	// 1. 普通单击：单选该行
	p.ClickSelectRow(1, 0)
	if p.SelectedIdx != 1 || !p.Chosen.Has("/tmp/box/b.txt") || p.Chosen.Len() != 1 {
		t.Errorf("plain click: idx=%d len=%d, want idx=1 len=1 with b.txt", p.SelectedIdx, p.Chosen.Len())
	}

	// 2. Cmd 单击：追加选中（toggle on）
	p.ClickSelectRow(2, ui.Cmd)
	if !p.Chosen.Has("/tmp/box/c.txt") || !p.Chosen.Has("/tmp/box/b.txt") {
		t.Errorf("cmd click should keep b.txt and add c.txt, len=%d", p.Chosen.Len())
	}

	// 3. Cmd 再次单击：取消选中（toggle off）
	p.ClickSelectRow(2, ui.Cmd)
	if p.Chosen.Has("/tmp/box/c.txt") {
		t.Errorf("second cmd click should remove c.txt")
	}

	// 4. Shift 单击：从锚点扩展范围（含两端）
	p.ClickSelectRow(0, ui.Shift)
	if p.Chosen.Len() != 3 || !p.Chosen.Has("/tmp/box/a.txt") || !p.Chosen.Has("/tmp/box/c.txt") {
		t.Errorf("shift click should select range [0,2] from anchor idx=2, len=%d", p.Chosen.Len())
	}

	// 5. 普通单击收拢为单选
	p.ClickSelectRow(2, 0)
	if p.SelectedIdx != 2 || p.Chosen.Len() != 1 || !p.Chosen.Has("/tmp/box/c.txt") {
		t.Errorf("plain click should collapse to c.txt only, len=%d", p.Chosen.Len())
	}

	// 6. 越界索引：不 panic 且状态不变
	p.ClickSelectRow(99, 0)
	if p.SelectedIdx != 2 || p.Chosen.Len() != 1 {
		t.Errorf("out-of-range click must not change state")
	}
}
