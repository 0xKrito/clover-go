package app

import (
	"path"
	"path/filepath"

	"github.com/bighaidao/clover-go/internal/domain"
	"github.com/bighaidao/clover-go/internal/fs"
	"github.com/egoist/mygo/ui"
)

// PaneState represents directory browsing state within a single pane.
type PaneState struct {
	ID   string
	Path string
	// PathDraft is the editable buffer for the path bar; committing it
	// navigates, discarding it (Esc) leaves the real Path untouched.
	PathDraft      string
	History        []string
	HistoryIndex   int
	Entries        []domain.FileInfo
	DisplayEntries []domain.FileInfo
	// IsSearchMode marks the listing as recursive search results.
	IsSearchMode    bool
	SelectedIdx     int
	Chosen          ui.Selection[string]
	Sort            domain.SortOptions
	FilterQuery     string
	IsLoading       bool
	LoadErr         string // Empty if directory loaded successfully, otherwise records read error
	LoadGen         uint64
	VolumeFreeBytes uint64
	TableState      ui.ListState
	GridState       ui.GridState
}

// InitPaneState initializes the fields and bindings of a PaneState.
func InitPaneState(p *PaneState, id, path string) {
	norm := domain.NormalizePath(path)
	p.ID = id
	p.Path = norm
	p.History = []string{norm}
	p.HistoryIndex = 0
	p.SelectedIdx = -1
	p.Sort = domain.SortOptions{
		Field:     domain.SortByName,
		Direction: domain.SortAsc,
	}

	p.TableState.Selected = &p.SelectedIdx
	p.TableState.Selection = &p.Chosen
	p.TableState.DragSelect = false
	p.TableState.Key = func(i int) any {
		if i >= 0 && i < len(p.DisplayEntries) {
			return p.DisplayEntries[i].Path
		}
		return i
	}
	// Label enables MyGo's native type-to-select: typing the first letters of
	// a file name while the list has focus chooses that row (Finder behavior).
	p.TableState.Label = func(i int) string {
		if i >= 0 && i < len(p.DisplayEntries) {
			return p.DisplayEntries[i].Name
		}
		return ""
	}

	p.GridState.Selected = &p.SelectedIdx
	p.GridState.Selection = &p.Chosen
	p.GridState.Key = func(i int) any {
		if i >= 0 && i < len(p.DisplayEntries) {
			return p.DisplayEntries[i].Path
		}
		return i
	}
	p.GridState.Label = func(i int) string {
		if i >= 0 && i < len(p.DisplayEntries) {
			return p.DisplayEntries[i].Name
		}
		return ""
	}
}

// NewPaneState instantiates an initialized PaneState.
func NewPaneState(id, path string) *PaneState {
	p := &PaneState{}
	InitPaneState(p, id, path)
	return p
}

// ClickSelectRow applies a pointer click on row i with modifier keys,
// mirroring Finder semantics: plain click selects one, Cmd toggles,
// Shift extends a range from the current anchor.
//
// It exists because the row-level drag source (BindFileDragSource) makes
// the row itself the press target in MyGo (flagDraggable), so the List
// row container never receives the press: the hit layer inside the row
// calls this instead of relying on the List's built-in click choosing.
func (p *PaneState) ClickSelectRow(i int, mods ui.Modifiers) {
	if i < 0 || i >= len(p.DisplayEntries) {
		return
	}
	path := p.DisplayEntries[i].Path
	switch {
	case mods&ui.Cmd != 0: // Cmd-click toggles the row, keeping others
		if p.Chosen.Has(path) {
			p.Chosen.Remove(path)
		} else {
			p.Chosen.Add(path)
		}
		p.SelectedIdx = i
	case mods&ui.Shift != 0: // Shift-click extends the range from the anchor
		from, to := p.SelectedIdx, i
		if from > to {
			from, to = to, from
		}
		if from < 0 {
			from = 0
		}
		if to >= len(p.DisplayEntries) {
			to = len(p.DisplayEntries) - 1
		}
		p.Chosen.Clear()
		for j := from; j <= to; j++ {
			p.Chosen.Add(p.DisplayEntries[j].Path)
		}
		p.SelectedIdx = i
	default: // Plain click selects exactly this row
		p.Chosen.Clear()
		p.Chosen.Add(path)
		p.SelectedIdx = i
	}
}

// Navigate pushes newPath into the pane's history and clears stale selection:
// selection never survives a directory change (Finder semantics).
func (p *PaneState) Navigate(newPath string) {
	norm := domain.NormalizePath(newPath)
	if p.Path == norm {
		return
	}
	if p.HistoryIndex+1 < len(p.History) {
		p.History = p.History[:p.HistoryIndex+1]
	}
	p.History = append(p.History, norm)
	p.HistoryIndex = len(p.History) - 1
	p.Path = norm
	p.SelectedIdx = -1
	p.Chosen.Clear()
}

// CanGoBack reports whether the pane can go backward.
func (p *PaneState) CanGoBack() bool {
	return p.HistoryIndex > 0
}

// GoBack navigates backward in history.
func (p *PaneState) GoBack() string {
	if !p.CanGoBack() {
		return ""
	}
	p.HistoryIndex--
	p.Path = p.History[p.HistoryIndex]
	p.SelectedIdx = -1
	p.Chosen.Clear()
	return p.Path
}

// CanGoForward reports whether the pane can go forward.
func (p *PaneState) CanGoForward() bool {
	return p.HistoryIndex+1 < len(p.History)
}

// GoForward navigates forward in history.
func (p *PaneState) GoForward() string {
	if !p.CanGoForward() {
		return ""
	}
	p.HistoryIndex++
	p.Path = p.History[p.HistoryIndex]
	p.SelectedIdx = -1
	p.Chosen.Clear()
	return p.Path
}

// CanGoUp reports whether the pane has a valid parent directory.
func (p *PaneState) CanGoUp() bool {
	if fs.IsArchiveVirtualPath(p.Path) {
		return true
	}
	norm := domain.NormalizePath(p.Path)
	return norm != "/" && norm != "."
}

// GoUp navigates to the parent directory or exits the archive level.
func (p *PaneState) GoUp() string {
	if !p.CanGoUp() {
		return ""
	}
	if fs.IsArchiveVirtualPath(p.Path) {
		archivePath, internalPath, ok := fs.ParseArchiveVirtualPath(p.Path)
		if ok {
			if internalPath == "" {
				parent := filepath.Dir(archivePath)
				p.Navigate(parent)
				return parent
			}
			parentInternal := path.Dir(internalPath)
			if parentInternal == "." || parentInternal == "/" {
				parentInternal = ""
			}
			parent := fs.FormatArchiveVirtualPath(archivePath, parentInternal)
			p.Navigate(parent)
			return parent
		}
	}
	parent := filepath.Dir(p.Path)
	p.Navigate(parent)
	return parent
}
