package metadata

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Date struct {
	Tag    string
	Wall   time.Time
	Offset *int
	Raw    string
}

func (d Date) Instant(zone *int) time.Time {
	switch {
	case d.Offset != nil:
		return d.Wall.Add(-time.Duration(*d.Offset) * time.Minute)
	case zone != nil:
		return d.Wall.Add(-time.Duration(*zone) * time.Minute)
	}

	return d.Wall
}

var (
	exifDate = regexp.MustCompile(`^(\d{4})[:-](\d{1,2})[:-](\d{1,2})[ T](\d{1,2}):(\d{1,2})(?::(\d{1,2})(?:[.,](\d+))?)?\s*(Z|[+-]\d{2}(?::?\d{2})?)?$`)

	likelyOffsets = []int{
		-720, -660, -600, -570, -540, -480, -420, -360, -300, -240, -210, -180, -120, -60, 0,
		60, 120, 180, 210, 240, 270, 300, 330, 345, 360, 390, 420, 480, 525, 540, 570, 600, 630,
		660, 720, 765, 780, 840,
	}
)

func ParseOffset(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "Z" {
		return 0, true
	}
	if len(s) < 3 || (s[0] != '+' && s[0] != '-') {
		return 0, false
	}

	digits := strings.ReplaceAll(s[1:], ":", "")
	if len(digits) != 2 && len(digits) != 4 {
		return 0, false
	}
	hours, err := strconv.Atoi(digits[:2])
	if err != nil || hours > 14 {
		return 0, false
	}
	minutes := 0
	if len(digits) == 4 {
		if minutes, err = strconv.Atoi(digits[2:]); err != nil || minutes > 59 {
			return 0, false
		}
	}

	total := hours*60 + minutes
	if s[0] == '-' {
		total = -total
	}

	return total, true
}

func ParseDate(tag string, raw string) (Date, bool) {
	s := strings.TrimSpace(raw)
	m := exifDate.FindStringSubmatch(s)
	if m == nil {
		return Date{}, false
	}

	atoi := func(v string) int {
		n, _ := strconv.Atoi(v)
		return n
	}
	year, month, day := atoi(m[1]), atoi(m[2]), atoi(m[3])
	hour, minute, second := atoi(m[4]), atoi(m[5]), atoi(m[6])
	if year == 0 || month < 1 || month > 12 || day < 1 || day > 31 || hour > 23 || minute > 59 || second > 59 {
		return Date{}, false
	}

	nanos := 0
	if m[7] != "" {
		fraction := (m[7] + "000")[:3]
		nanos = atoi(fraction) * int(time.Millisecond)
	}

	wall := time.Date(year, time.Month(month), day, hour, minute, second, nanos, time.UTC)
	if wall.Day() != day {
		return Date{}, false
	}

	d := Date{Tag: tag, Wall: wall, Raw: s}
	if m[8] != "" {
		if offset, ok := ParseOffset(m[8]); ok {
			d.Offset = &offset
		}
	}

	return d, true
}

func ZoneName(offset int) string {
	if offset == 0 {
		return "UTC"
	}

	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	name := "UTC" + sign + strconv.Itoa(offset/60)
	if minutes := offset % 60; minutes != 0 {
		name += ":" + strconv.Itoa(minutes/10) + strconv.Itoa(minutes%10)
	}

	return name
}

func ZoneOffset(name string) (int, bool) {
	switch name {
	case "UTC", "UTC+0", "UTC-0", "Z", "Etc/UTC", "GMT":
		return 0, true
	}
	rest, ok := strings.CutPrefix(name, "UTC")
	if !ok || rest == "" {
		return 0, false
	}

	hours, minutes, _ := strings.Cut(rest[1:], ":")
	h, err := strconv.Atoi(hours)
	if err != nil {
		return 0, false
	}
	m := 0
	if minutes != "" {
		if m, err = strconv.Atoi(minutes); err != nil {
			return 0, false
		}
	}

	total := h*60 + m
	if rest[0] == '-' {
		total = -total
	}

	return total, true
}

func inferLikelyOffset(deltaMinutes float64) (int, bool) {
	best := likelyOffsets[0]
	for _, o := range likelyOffsets {
		if math.Abs(float64(o)-deltaMinutes) < math.Abs(float64(best)-deltaMinutes) {
			best = o
		}
	}
	if math.Abs(float64(best)-deltaMinutes) < 30 {
		return best, true
	}

	return 0, false
}
