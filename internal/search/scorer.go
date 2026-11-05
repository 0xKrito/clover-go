package search

import (
	"sort"
	"strings"
	"unicode"

	"github.com/bighaidao/clover-go/internal/domain"
)

// SearchResult pairs a FileInfo with its computed relevance score.
type SearchResult struct {
	Item  domain.FileInfo
	Score int
}

// ScoreItem evaluates a query against a file name.
func ScoreItem(query, target string) (int, bool) {
	cleanQuery := strings.TrimSpace(strings.ToLower(query))
	cleanTarget := strings.ToLower(target)

	if cleanQuery == "" {
		return 0, true
	}

	tokens := strings.Fields(cleanQuery)
	totalScore := 0

	for _, token := range tokens {
		tokenScore, matched := scoreToken(token, cleanTarget, target)
		if !matched {
			return 0, false
		}
		totalScore += tokenScore
	}

	// Length penalty: shorter matches are generally more relevant
	lenDiff := len(cleanTarget) - len(cleanQuery)
	if lenDiff > 0 && lenDiff < 100 {
		totalScore += (100 - lenDiff) / 2
	}

	return totalScore, true
}

func scoreToken(token, lowerTarget, origTarget string) (int, bool) {
	if lowerTarget == token {
		return 1000, true // Exact match
	}

	if strings.HasPrefix(lowerTarget, token) {
		return 500, true // Prefix match
	}

	if idx := strings.Index(lowerTarget, token); idx >= 0 {
		// Check if it's at a word boundary (e.g. after _, -, space, or dot)
		if idx > 0 {
			prev := rune(origTarget[idx-1])
			if prev == '_' || prev == '-' || prev == '.' || prev == ' ' || unicode.IsSpace(prev) {
				return 350, true
			}
			// CamelCase boundary
			if unicode.IsLower(prev) && unicode.IsUpper(rune(origTarget[idx])) {
				return 300, true
			}
		}
		return 150, true // Normal substring match
	}

	return 0, false
}

// SortSearchResults sorts results by score descending, then by name naturally.
func SortSearchResults(results []SearchResult) {
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return domain.NaturalCompare(results[i].Item.Name, results[j].Item.Name) < 0
	})
}
