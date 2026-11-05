package domain

// FinderFavoriteItem represents a bookmarked favorite location detected from Finder sidebar or system presets.
type FinderFavoriteItem struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Selected bool   `json:"selected"`
	Exists   bool   `json:"exists"`
}
