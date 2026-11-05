package domain

import (
	"reflect"
	"testing"
)

func TestParsePageRanges(t *testing.T) {
	tests := []struct {
		name      string
		expr      string
		pageCount int
		expected  []int
		wantErr   bool
	}{
		{
			name:      "empty means all",
			expr:      "",
			pageCount: 5,
			expected:  []int{1, 2, 3, 4, 5},
			wantErr:   false,
		},
		{
			name:      "all keyword",
			expr:      "all",
			pageCount: 3,
			expected:  []int{1, 2, 3},
			wantErr:   false,
		},
		{
			name:      "single pages and ranges",
			expr:      "1-3, 5, 8-10",
			pageCount: 10,
			expected:  []int{1, 2, 3, 5, 8, 9, 10},
			wantErr:   false,
		},
		{
			name:      "deduplication and reordering",
			expr:      "5, 2, 1-3, 2",
			pageCount: 6,
			expected:  []int{1, 2, 3, 5},
			wantErr:   false,
		},
		{
			name:      "reversed range 4-2",
			expr:      "4-2",
			pageCount: 5,
			expected:  []int{2, 3, 4},
			wantErr:   false,
		},
		{
			name:      "out of bounds high",
			expr:      "1-12",
			pageCount: 10,
			wantErr:   true,
		},
		{
			name:      "out of bounds zero",
			expr:      "0-2",
			pageCount: 5,
			wantErr:   true,
		},
		{
			name:      "invalid chars",
			expr:      "abc",
			pageCount: 5,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePageRanges(tt.expr, tt.pageCount)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParsePageRanges() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("ParsePageRanges() = %v, want %v", got, tt.expected)
			}
		})
	}
}
