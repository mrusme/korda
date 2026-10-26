package request

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	jsYear     = regexp.MustCompile(`^([+-]\d{6}|\d{4})$`)
	jsISODate  = regexp.MustCompile(`^([+-]\d{6}|\d{4})[-/](\d{1,2})(?:[-/](\d{1,2}))?(?:[T ](\d{1,2}):(\d{2})(?::(\d{2})(?:\.\d+)?)?)?(Z|[+-]\d{2}:?\d{2})?$`)
	jsMonthDay = regexp.MustCompile(`(?i)^(?:[a-z]{3,9},? )?(jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)[a-z]* (\d{1,2}),? (\d{4})$`)
	jsMonths   = []string{"jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"}
)

func jsDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	atoi := func(v string) int {
		n, _ := strconv.Atoi(strings.TrimPrefix(v, "+"))
		return n
	}

	switch {
	case jsYear.MatchString(s):
		return time.Date(atoi(s), 1, 1, 0, 0, 0, 0, time.UTC), true

	case jsISODate.MatchString(s):
		m := jsISODate.FindStringSubmatch(s)
		month, day := atoi(m[2]), 1
		if m[3] != "" {
			day = atoi(m[3])
		}
		if month < 1 || month > 12 || day < 1 || day > 31 {
			return time.Time{}, false
		}
		hour, minute, second := atoi(m[4]), atoi(m[5]), atoi(m[6])
		if hour > 24 || minute > 59 || second > 59 {
			return time.Time{}, false
		}
		t := time.Date(atoi(m[1]), time.Month(month), day, hour, minute, second, 0, time.UTC)
		if zone := m[7]; zone != "" && zone != "Z" {
			offset := strings.ReplaceAll(zone[1:], ":", "")
			seconds := (atoi(offset[:2])*60 + atoi(offset[2:])) * 60
			if zone[0] == '+' {
				seconds = -seconds
			}
			t = t.Add(time.Duration(seconds) * time.Second)
		}
		return t, true

	case jsMonthDay.MatchString(s):
		m := jsMonthDay.FindStringSubmatch(s)
		month := 1
		for i, name := range jsMonths {
			if strings.EqualFold(m[1], name) {
				month = i + 1
			}
		}
		return time.Date(atoi(m[3]), time.Month(month), atoi(m[2]), 0, 0, 0, 0, time.UTC), true
	}

	return time.Time{}, false
}

func birthDate(raw any) string {
	s, _ := raw.(string)
	if s == "" {
		return ""
	}
	if t, ok := jsDate(s); ok && !t.After(time.Now()) {
		return ""
	}

	return "Birth date cannot be in the future"
}

func uniqueIDs(raw any) string {
	list, _ := raw.([]any)
	seen := make(map[any]bool, len(list))
	for _, item := range list {
		if seen[item] {
			return "Items must be unique"
		}
		seen[item] = true
	}

	return ""
}
