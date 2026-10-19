package sanitize

import (
	"strings"
	"testing"
)

func TestFilenameMatchesSanitizeFilename(t *testing.T) {
	tests := map[string]string{
		"ju/no":        "juno",
		"admin":        "admin",
		`a<b>c:d"e|f?`: "abcdef",
		"..":           "",
		"con":          "",
		"COM1.txt":     "",
		"name. ":       "name",
		"tab\there":    "tabhere",
		"ünïcode":      "ünïcode",
	}

	for in, want := range tests {
		if got := Filename(in); got != want {
			t.Errorf("Filename(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFilenameTruncatesTo255BytesOnARuneBoundary(t *testing.T) {
	got := Filename(strings.Repeat("é", 200))
	if len(got) > 255 || len(got)%2 != 0 {
		t.Errorf("Filename kept %d bytes", len(got))
	}
}
