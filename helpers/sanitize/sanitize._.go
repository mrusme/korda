package sanitize

import (
	"regexp"
	"unicode/utf8"
)

var (
	illegal         = regexp.MustCompile(`[/?<>\\:*|"]`)
	control         = regexp.MustCompile(`[\x00-\x1f\x{80}-\x{9f}]`)
	reserved        = regexp.MustCompile(`^\.+$`)
	windowsReserved = regexp.MustCompile(`(?i)^(con|prn|aux|nul|com[0-9]|lpt[0-9])(\..*)?$`)
	windowsTrailing = regexp.MustCompile(`[. ]+$`)
)

const maxBytes = 255

func Filename(input string) string {
	s := illegal.ReplaceAllString(input, "")
	s = control.ReplaceAllString(s, "")
	s = reserved.ReplaceAllString(s, "")
	s = windowsReserved.ReplaceAllString(s, "")
	s = windowsTrailing.ReplaceAllString(s, "")

	return truncate(s, maxBytes)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}

	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}

	return s[:n]
}
