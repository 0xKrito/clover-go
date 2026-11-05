package app

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/bighaidao/clover-go/internal/domain"
)

// ClosedTabSnapshot holds metadata for closed tab restoration (⇧⌘T).
type ClosedTabSnapshot struct {
	Path     string    `json:"path"`
	Title    string    `json:"title"`
	ClosedAt time.Time `json:"closed_at"`
}

// TabState represents a single tab session, supporting single or dual pane.
type TabState struct {
	ID        string
	Title     string
	Subtitle  string
	IsPinned  bool
	IsDual    bool
	ActiveIdx int // 0 for Left/Primary, 1 for Right/Secondary

	PaneState            // Embedded Left/Primary Pane for full backwards compatibility
	RightPane *PaneState // Secondary/Right Pane when IsDual is true
}

// ActivePane returns the currently active pane in this tab.
func (t *TabState) ActivePane() *PaneState {
	if t.IsDual && t.ActiveIdx == 1 && t.RightPane != nil {
		return t.RightPane
	}
	return &t.PaneState
}

// OppositePane returns the opposite pane when dual-pane mode is active, or nil.
func (t *TabState) OppositePane() *PaneState {
	if !t.IsDual || t.RightPane == nil {
		return nil
	}
	if t.ActiveIdx == 1 {
		return &t.PaneState
	}
	return t.RightPane
}

// Left returns the left/primary pane.
func (t *TabState) Left() *PaneState {
	return &t.PaneState
}

// Right returns the right/secondary pane.
func (t *TabState) Right() *PaneState {
	return t.RightPane
}

// ToggleDualPane toggles dual pane mode.
func (t *TabState) ToggleDualPane() {
	t.IsDual = !t.IsDual
	if t.IsDual {
		if t.RightPane == nil {
			t.RightPane = NewPaneState(t.ID+"_right", t.PaneState.Path)
		}
		t.ActiveIdx = 1
	} else {
		t.ActiveIdx = 0
	}
}

// SwitchActivePane switches active pane between left and right.
func (t *TabState) SwitchActivePane() {
	if !t.IsDual {
		return
	}
	if t.ActiveIdx == 0 {
		t.ActiveIdx = 1
	} else {
		t.ActiveIdx = 0
	}
}

// TabManager orchestrates tab lifecycle, history, and restoration.
type TabManager struct {
	Tabs        []*TabState
	ActiveIndex int
	ClosedStack []ClosedTabSnapshot
	// DefaultSort is the user-preferred default ordering applied to newly
	// created panes (tabs and dual-pane right panes). Zero value keeps the
	// name/asc fallback of InitPaneState.
	DefaultSort domain.SortOptions
	nextID      int
}

// NewTabManager creates a TabManager with an initial root tab.
func NewTabManager(initialPath string) *TabManager {
	tm := &TabManager{
		// Fallback ordering when no preference was wired in (tests, early
		// construction); NewCoordinator overrides it with the user's choice.
		DefaultSort: domain.SortOptions{Field: domain.SortByName, Direction: domain.SortAsc},
		ActiveIndex: 0,
		ClosedStack: make([]ClosedTabSnapshot, 0, 20),
	}
	tm.NewTab(initialPath)
	return tm
}

// NewTab adds and activates a new tab at the given path.
func (tm *TabManager) NewTab(path string) *TabState {
	tm.nextID++
	norm := domain.NormalizePath(path)
	title := filepath.Base(norm)
	if norm == "/" {
		title = "Macintosh HD"
	}

	tab := &TabState{
		ID:        fmt.Sprintf("tab_%d", tm.nextID),
		Title:     title,
		ActiveIdx: 0,
	}
	InitPaneState(&tab.PaneState, tab.ID+"_left", norm)
	tab.PaneState.Sort = tm.DefaultSort

	tm.Tabs = append(tm.Tabs, tab)
	tm.ActiveIndex = len(tm.Tabs) - 1
	tm.Disambiguate()
	return tab
}

// CloseTab closes the tab at index unless it is pinned.
func (tm *TabManager) CloseTab(index int) bool {
	if index < 0 || index >= len(tm.Tabs) {
		return false
	}
	tab := tm.Tabs[index]
	if tab.IsPinned {
		return false // Pinned tabs cannot be closed accidentally
	}

	// Push snapshot to closed stack (capacity 20)
	tm.pushClosed(ClosedTabSnapshot{
		Path:     tab.Path,
		Title:    tab.Title,
		ClosedAt: time.Now(),
	})

	tm.Tabs = append(tm.Tabs, tab) // Keep slice length safe
	tm.Tabs = append(tm.Tabs[:index], tm.Tabs[index+1:len(tm.Tabs)-1]...)

	if len(tm.Tabs) == 0 {
		tm.NewTab(tab.Path) // Always maintain at least one tab
		return true
	}

	if tm.ActiveIndex >= len(tm.Tabs) {
		tm.ActiveIndex = len(tm.Tabs) - 1
	}
	tm.Disambiguate()
	return true
}

func (tm *TabManager) pushClosed(snap ClosedTabSnapshot) {
	if len(tm.ClosedStack) >= 20 {
		tm.ClosedStack = tm.ClosedStack[1:]
	}
	tm.ClosedStack = append(tm.ClosedStack, snap)
}

// RestoreLastClosedTab pops from the closed stack and reopens the tab (⇧⌘T).
func (tm *TabManager) RestoreLastClosedTab() *TabState {
	if len(tm.ClosedStack) == 0 {
		return nil
	}
	last := tm.ClosedStack[len(tm.ClosedStack)-1]
	tm.ClosedStack = tm.ClosedStack[:len(tm.ClosedStack)-1]
	return tm.NewTab(last.Path)
}

// TogglePin toggles pinned state for the tab at index.
func (tm *TabManager) TogglePin(index int) {
	if index < 0 || index >= len(tm.Tabs) {
		return
	}
	tm.Tabs[index].IsPinned = !tm.Tabs[index].IsPinned
}

// DuplicateTab clones the tab at index.
func (tm *TabManager) DuplicateTab(index int) *TabState {
	if index < 0 || index >= len(tm.Tabs) {
		return nil
	}
	src := tm.Tabs[index]
	newTab := tm.NewTab(src.Path)
	return newTab
}

// CloseOtherTabs closes all unpinned tabs except the one at keepIdx.
func (tm *TabManager) CloseOtherTabs(keepIdx int) {
	if keepIdx < 0 || keepIdx >= len(tm.Tabs) {
		return
	}
	kept := tm.Tabs[keepIdx]
	// Snapshot by pointer first: CloseTab mutates tm.Tabs, so indexes taken
	// from the pre-loop slice would drift while closing.
	var toClose []*TabState
	for i, tab := range tm.Tabs {
		if i != keepIdx && !tab.IsPinned {
			toClose = append(toClose, tab)
		}
	}
	for _, tab := range toClose {
		for idx, t := range tm.Tabs {
			if t == tab {
				tm.CloseTab(idx) // keeps ⇧⌘T restoration and pinned guard consistent
				break
			}
		}
	}
	// Re-locate the kept tab; closing may have shifted its index.
	for idx, tab := range tm.Tabs {
		if tab == kept {
			tm.ActiveIndex = idx
			break
		}
	}
	tm.Disambiguate()
}

// CloseTabsToTheRight closes all unpinned tabs situated after rightIdx.
func (tm *TabManager) CloseTabsToTheRight(rightIdx int) {
	if rightIdx < 0 || rightIdx >= len(tm.Tabs)-1 {
		return
	}
	// Close from the tail so indexes stay stable while removing.
	for i := len(tm.Tabs) - 1; i > rightIdx; i-- {
		if !tm.Tabs[i].IsPinned {
			tm.CloseTab(i)
		}
	}
	tm.Disambiguate()
}

// ActiveTab returns the currently focused tab.
func (tm *TabManager) ActiveTab() *TabState {
	if len(tm.Tabs) == 0 {
		return nil
	}
	if tm.ActiveIndex < 0 || tm.ActiveIndex >= len(tm.Tabs) {
		tm.ActiveIndex = 0
	}
	return tm.Tabs[tm.ActiveIndex]
}

// Navigate pushes newPath into the active pane's history and updates the current path.
func (tm *TabManager) Navigate(index int, newPath string) {
	if index < 0 || index >= len(tm.Tabs) {
		return
	}
	tab := tm.Tabs[index]
	tm.NavigatePane(index, tab.ActivePane(), newPath)
}

// NavigatePane pushes newPath into the specified pane's history. Tab-level
// Path/Title only track the left/primary pane, so right-pane navigation must
// not overwrite them (dual-pane isolation).
func (tm *TabManager) NavigatePane(index int, pane *PaneState, newPath string) {
	if index < 0 || index >= len(tm.Tabs) || pane == nil {
		return
	}
	tab := tm.Tabs[index]
	pane.Navigate(newPath)
	if pane == &tab.PaneState {
		tab.Path = tab.PaneState.Path
		tab.Title = filepath.Base(tab.Path)
		if tab.Path == "/" {
			tab.Title = "Macintosh HD"
		}
	}
	tm.Disambiguate()
}

// CanGoBack reports whether the active pane can go backward in history.
func (tm *TabManager) CanGoBack(index int) bool {
	if index < 0 || index >= len(tm.Tabs) {
		return false
	}
	return tm.Tabs[index].ActivePane().CanGoBack()
}

// GoBack navigates backward in history on the active pane.
func (tm *TabManager) GoBack(index int) string {
	if !tm.CanGoBack(index) {
		return ""
	}
	tab := tm.Tabs[index]
	activePane := tab.ActivePane()
	res := activePane.GoBack()
	if !tab.IsDual || tab.ActiveIdx == 0 {
		tab.Path = activePane.Path
		tab.Title = filepath.Base(tab.Path)
	}
	tm.Disambiguate()
	return res
}

// CanGoForward reports whether the active pane can go forward in history.
func (tm *TabManager) CanGoForward(index int) bool {
	if index < 0 || index >= len(tm.Tabs) {
		return false
	}
	return tm.Tabs[index].ActivePane().CanGoForward()
}

// GoForward navigates forward in history on the active pane.
func (tm *TabManager) GoForward(index int) string {
	if !tm.CanGoForward(index) {
		return ""
	}
	tab := tm.Tabs[index]
	activePane := tab.ActivePane()
	res := activePane.GoForward()
	if !tab.IsDual || tab.ActiveIdx == 0 {
		tab.Path = activePane.Path
		tab.Title = filepath.Base(tab.Path)
	}
	tm.Disambiguate()
	return res
}

// CanGoUp reports whether the active pane has a valid parent directory.
func (tm *TabManager) CanGoUp(index int) bool {
	if index < 0 || index >= len(tm.Tabs) {
		return false
	}
	return tm.Tabs[index].ActivePane().CanGoUp()
}

// GoUp navigates to the parent directory on the active pane.
func (tm *TabManager) GoUp(index int) string {
	if !tm.CanGoUp(index) {
		return ""
	}
	tab := tm.Tabs[index]
	parent := filepath.Dir(tab.ActivePane().Path)
	tm.Navigate(index, parent)
	return parent
}

// Disambiguate runs domain-level leaf grouping and applies subtitles.
func (tm *TabManager) Disambiguate() {
	summaries := make([]domain.TabSummary, len(tm.Tabs))
	for i, t := range tm.Tabs {
		summaries[i] = domain.TabSummary{ID: t.ID, Path: t.Path}
	}
	subtitles := domain.DisambiguateTabs(summaries)
	for _, t := range tm.Tabs {
		if sub, ok := subtitles[t.ID]; ok && sub != t.Title {
			t.Subtitle = sub
		} else {
			t.Subtitle = ""
		}
	}
}
