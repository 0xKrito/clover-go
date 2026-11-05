package macos

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// ParseCloverURL parses fclover:// URL schemes.
// Supports: fclover://open?path=/Users/... and fclover://reveal?path=...
func ParseCloverURL(rawURL string) (action, targetPath string, err error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid URL: %w", err)
	}

	if u.Scheme != "fclover" {
		return "", "", fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}

	action = strings.ToLower(u.Host)
	if action == "" {
		action = strings.TrimPrefix(u.Path, "/")
	}

	q := u.Query()
	targetPath = q.Get("path")
	if targetPath == "" && u.Path != "" && u.Host != "open" && u.Host != "reveal" {
		targetPath = filepath.Clean(u.Host + u.Path)
	}

	return action, targetPath, nil
}
