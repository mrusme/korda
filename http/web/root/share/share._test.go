package share

import "testing"

func TestClock(t *testing.T) {
	for ms, want := range map[int64]string{0: "0:00", 5_400: "0:05", 65_000: "1:05", 3_725_000: "1:02:05"} {
		if got := clock(ms); got != want {
			t.Errorf("clock(%d) = %s, want %s", ms, got, want)
		}
	}
}

func TestCount(t *testing.T) {
	for _, c := range []struct {
		photos int
		videos int
		want   string
	}{
		{0, 0, "No photos or videos"},
		{1, 0, "1 photo"},
		{3, 0, "3 photos"},
		{0, 1, "1 video"},
		{2, 1, "2 photos and 1 video"},
		{1, 4, "1 photo and 4 videos"},
	} {
		if got := count(c.photos, c.videos); got != c.want {
			t.Errorf("count(%d, %d) = %s, want %s", c.photos, c.videos, got, c.want)
		}
	}
}
