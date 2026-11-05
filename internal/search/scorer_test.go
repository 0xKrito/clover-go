package search

import (
	"testing"

	"github.com/bighaidao/clover-go/internal/domain"
)

func TestScoreItem(t *testing.T) {
	// 1. Exact match vs prefix vs substring
	sExact, ok1 := ScoreItem("main", "main")
	sPrefix, ok2 := ScoreItem("main", "main.go")
	sSub, ok3 := ScoreItem("main", "domain.go")

	if !ok1 || !ok2 || !ok3 {
		t.Fatalf("expected all to match")
	}

	if sExact <= sPrefix {
		t.Errorf("exact score %d should be greater than prefix score %d", sExact, sPrefix)
	}
	if sPrefix <= sSub {
		t.Errorf("prefix score %d should be greater than substring score %d", sPrefix, sSub)
	}

	// 2. Multi-token match
	score, ok := ScoreItem("user model", "user_model_test.go")
	if !ok || score <= 0 {
		t.Errorf("expected multi-token match, got ok=%v, score=%d", ok, score)
	}

	// Partial miss should fail
	_, okMiss := ScoreItem("user controller", "user_model_test.go")
	if okMiss {
		t.Errorf("expected miss when one token is absent")
	}

	// 3. Word boundary bonus
	sBoundary, _ := ScoreItem("test", "my_test.go")
	sNormal, _ := ScoreItem("test", "mytest.go")
	if sBoundary <= sNormal {
		t.Errorf("boundary score %d should be greater than normal substring %d", sBoundary, sNormal)
	}
}

func TestSortSearchResults(t *testing.T) {
	results := []SearchResult{
		{Item: domain.FileInfo{Name: "b_sub.go"}, Score: 100},
		{Item: domain.FileInfo{Name: "a_exact.go"}, Score: 500},
		{Item: domain.FileInfo{Name: "a_sub.go"}, Score: 100},
	}

	SortSearchResults(results)

	if results[0].Item.Name != "a_exact.go" {
		t.Errorf("expected highest score first, got %s", results[0].Item.Name)
	}
	if results[1].Item.Name != "a_sub.go" {
		t.Errorf("expected tie-break by natural name asc, got %s", results[1].Item.Name)
	}
	if results[2].Item.Name != "b_sub.go" {
		t.Errorf("expected b_sub.go last, got %s", results[2].Item.Name)
	}
}
