//go:build tzf

package metadata

import (
	"testing"
	"time"
)

func TestGPSZoneNamesTheIANAZone(t *testing.T) {
	for _, tc := range []struct {
		lat, lon float64
		want     string
	}{
		{52.516272, 13.377722, "Europe/Berlin"},
		{40.7484, -73.9857, "America/New_York"},
		{35.6895, 139.6917, "Asia/Tokyo"},
		{-33.8688, 151.2093, "Australia/Sydney"},
	} {
		if got, ok := gpsZone(tc.lat, tc.lon); !ok || got != tc.want {
			t.Errorf("gpsZone(%v, %v) = %q, %v, want %q", tc.lat, tc.lon, got, ok, tc.want)
		}
	}
}

func TestInstantReadsAZonelessDateInTheGPSZone(t *testing.T) {
	tags := newTags()
	lat, lon := 52.516272, 13.377722
	tags.Latitude, tags.Longitude = &lat, &lon
	tags.inferZone()
	if tags.TZ != "Europe/Berlin" || tags.ZoneSource != "gps" {
		t.Fatalf("zone = %q from %q", tags.TZ, tags.ZoneSource)
	}

	for raw, want := range map[string]time.Time{
		"2024:07:01 12:00:00":       time.Date(2024, 7, 1, 10, 0, 0, 0, time.UTC),
		"2024:01:15 12:00:00":       time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
		"2024:07:01 12:00:00+05:00": time.Date(2024, 7, 1, 7, 0, 0, 0, time.UTC),
	} {
		d, ok := ParseDate("DateTimeOriginal", raw)
		if !ok {
			t.Fatalf("ParseDate(%q) failed", raw)
		}
		if got := tags.Instant(d).UTC(); !got.Equal(want) {
			t.Errorf("Instant(%q) = %v, want %v", raw, got, want)
		}
	}
}

func TestOffsetTagsComeBeforeTheGPSZone(t *testing.T) {
	tags := newTags()
	lat, lon := 52.516272, 13.377722
	offset := -300
	tags.Latitude, tags.Longitude, tags.Zone = &lat, &lon, &offset
	tags.inferZone()
	if tags.TZ != "" || tags.Zone == nil || *tags.Zone != -300 {
		t.Errorf("zone = %q, %v", tags.TZ, tags.Zone)
	}
}
