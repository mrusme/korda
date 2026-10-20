package collation

import (
	"slices"
	"testing"
)

func TestSortMatchesGlibcEnUS(t *testing.T) {
	for _, want := range [][]string{
		{"Trave", "travel", "Travel", "TRAVEL", "Travel 2026", "Travel-2026", "Travel/2026", "Travel2026", "Travel/Japan", "Travels"},
		{"1x", "a", "A", "ä", "Ä", "b", "B", "%x", "_x", "x"},
	} {
		got := slices.Clone(want)
		slices.Reverse(got)
		Sort(got)
		if !slices.Equal(got, want) {
			t.Errorf("Sort = %q, want %q", got, want)
		}
	}
}
