package macos

import (
	"testing"
)

func TestReadFinderFavorites(t *testing.T) {
	favs, err := ReadFinderFavorites()
	if err != nil {
		t.Fatalf("ReadFinderFavorites failed: %v", err)
	}
	if len(favs) == 0 {
		t.Fatalf("expected at least 1 favorite directory, got 0")
	}

	hasValidPath := false
	for _, it := range favs {
		if it.Path != "" && it.Name != "" {
			hasValidPath = true
			break
		}
	}
	if !hasValidPath {
		t.Errorf("no valid favorite items found: %+v", favs)
	}
}
