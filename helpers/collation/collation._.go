package collation

import (
	"slices"
	"strings"
	"sync"
	"unicode"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

var (
	mu       sync.Mutex
	collator = collate.New(language.English)
)

func ignorable(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsMark(r)
}

func strip(s string) string {
	return strings.Map(func(r rune) rune {
		if ignorable(r) {
			return -1
		}
		return r
	}, s)
}

func ignored(a string, b string) int {
	ra, rb := []rune(a), []rune(b)
	for i := 0; i < len(ra) && i < len(rb); i++ {
		x, y := ra[i], rb[i]
		if x == y {
			continue
		}
		ix, iy := ignorable(x), ignorable(y)
		switch {
		case ix && !iy:
			return -1
		case !ix && iy:
			return 1
		case ix && iy && x < y:
			return -1
		case ix && iy:
			return 1
		}
		return 0
	}

	return len(ra) - len(rb)
}

func Compare(a string, b string) int {
	mu.Lock()
	c := collator.CompareString(strip(a), strip(b))
	mu.Unlock()
	if c != 0 {
		return c
	}
	if c = ignored(a, b); c != 0 {
		return c
	}

	return strings.Compare(a, b)
}

func Sort(values []string) {
	slices.SortStableFunc(values, Compare)
}
