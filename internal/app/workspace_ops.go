package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/bighaidao/clover-go/internal/config"
)

// SaveCurrentWorkspace captures a snapshot of all open tabs and their dual-pane states.
func (c *Coordinator) SaveCurrentWorkspace(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("workspace name cannot be empty")
	}

	var snapshots []config.WorkspacePaneSnapshot
	for _, t := range c.Tabs.Tabs {
		snap := config.WorkspacePaneSnapshot{
			LeftPath:  t.Path,
			IsDual:    t.IsDual,
			ActiveIdx: t.ActiveIdx,
		}
		if t.IsDual && t.RightPane != nil {
			snap.RightPath = t.RightPane.Path
		}
		snapshots = append(snapshots, snap)
	}

	ws := config.NamedWorkspace{
		ID:        fmt.Sprintf("ws-%d", time.Now().UnixNano()),
		Name:      trimmed,
		Tabs:      snapshots,
		ActiveTab: c.Tabs.ActiveIndex,
	}

	if err := c.Workspaces.AddOrUpdateWorkspace(ws); err != nil {
		return err
	}
	c.ActiveWorkspaceID = ws.ID
	c.Invalidate()
	return nil
}

// RestoreWorkspace restores tabs and panes layout from a saved workspace.
func (c *Coordinator) RestoreWorkspace(ws config.NamedWorkspace) {
	if len(ws.Tabs) == 0 && len(ws.Paths) == 0 {
		return
	}

	c.Tabs.Tabs = nil
	c.Tabs.ActiveIndex = 0

	if len(ws.Tabs) > 0 {
		for _, snap := range ws.Tabs {
			tab := c.Tabs.NewTab(snap.LeftPath)
			if snap.IsDual && snap.RightPath != "" {
				tab.IsDual = true
				tab.RightPane = NewPaneState(tab.ID+"_right", snap.RightPath)
				tab.RightPane.Sort = c.Tabs.DefaultSort
				tab.ActiveIdx = snap.ActiveIdx
			}
		}
	} else {
		for _, p := range ws.Paths {
			c.Tabs.NewTab(p)
		}
	}

	if ws.ActiveTab >= 0 && ws.ActiveTab < len(c.Tabs.Tabs) {
		c.Tabs.ActiveIndex = ws.ActiveTab
	}
	c.ActiveWorkspaceID = ws.ID
	c.LoadCurrentTab(nil)
}

// DeleteWorkspace removes a saved workspace by ID or name.
func (c *Coordinator) DeleteWorkspace(idOrName string) error {
	err := c.Workspaces.RemoveWorkspace(idOrName)
	if c.ActiveWorkspaceID == idOrName {
		c.ActiveWorkspaceID = ""
	}
	c.Invalidate()
	return err
}
