package macos

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/bighaidao/clover-go/internal/domain"
)

// ReadFinderFavorites reads favorite directories from macOS Finder sidebar (sfl3).
// If sfl3 is unavailable, returns standard system favorite directories.
func ReadFinderFavorites() ([]domain.FinderFavoriteItem, error) {
	userHome, _ := os.UserHomeDir()
	// 允许通过环境变量显式指定用户主目录（沙盒/测试环境注入）。
	if envHome := os.Getenv("CLOVER_USER_HOME"); envHome != "" {
		userHome = envHome
	} else if strings.Contains(userHome, ".agy_home") {
		// Use real login user directory when running in sandbox home
		if u := os.Getenv("USER"); u != "" {
			realHome := "/Users/" + u
			if _, err := os.Stat(realHome); err == nil {
				userHome = realHome
			}
		}
	}

	sflPath := filepath.Join(userHome, "Library/Application Support/com.apple.sharedfilelist/com.apple.LSSharedFileList.FavoriteItems.sfl3")
	items, err := parseSFL3Favorites(sflPath)
	if err != nil || len(items) == 0 {
		items = fallbackStandardFavorites(userHome)
	}

	seen := make(map[string]bool)
	var result []domain.FinderFavoriteItem
	for _, it := range items {
		clean := domain.NormalizePath(it.Path)
		if clean == "" || clean == "/" || seen[clean] {
			continue
		}
		if strings.HasSuffix(clean, ".cannedSearch") {
			continue
		}
		seen[clean] = true
		_, statErr := os.Stat(clean)
		it.Path = clean
		it.Exists = (statErr == nil)
		it.Selected = it.Exists
		result = append(result, it)
	}

	return result, nil
}

func parseSFL3Favorites(sflPath string) ([]domain.FinderFavoriteItem, error) {
	if _, err := os.Stat(sflPath); err != nil {
		return nil, err
	}

	out, err := runHelper("finder-favs", sflPath)
	if err != nil {
		return nil, err
	}

	var favs []domain.FinderFavoriteItem
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 {
			favs = append(favs, domain.FinderFavoriteItem{
				Name: parts[0],
				Path: parts[1],
			})
		}
	}
	return favs, nil
}

func fallbackStandardFavorites(userHome string) []domain.FinderFavoriteItem {
	candidates := []struct {
		name string
		rel  string
	}{
		{"桌面", "Desktop"},
		{"下载", "Downloads"},
		{"文稿", "Documents"},
		{"图片", "Pictures"},
		{"影视", "Movies"},
		{"音乐", "Music"},
		{"应用程序", "/Applications"},
	}

	var list []domain.FinderFavoriteItem
	for _, c := range candidates {
		p := c.rel
		if !strings.HasPrefix(p, "/") {
			p = filepath.Join(userHome, p)
		}
		list = append(list, domain.FinderFavoriteItem{
			Name: c.name,
			Path: p,
		})
	}
	return list
}
