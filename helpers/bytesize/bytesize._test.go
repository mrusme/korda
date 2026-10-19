package bytesize

import "testing"

func TestHumanReadableMatchesImmich(t *testing.T) {
	tests := map[int64]string{
		0:           "0 B",
		1023:        "1023 B",
		1024:        "1.0 KiB",
		1280:        "1.3 KiB",
		13269012480: "12.4 GiB",
		40799043584: "38.0 GiB",
		26261172224: "24.5 GiB",
		4 * GiB:     "4.0 GiB",
	}

	for bytes, want := range tests {
		if got := HumanReadable(bytes); got != want {
			t.Errorf("HumanReadable(%d) = %q, want %q", bytes, got, want)
		}
	}
}

func TestToFixedRoundsHalfUpLikeJavaScript(t *testing.T) {
	tests := []struct {
		x      float64
		digits int
		want   string
	}{
		{1.25, 1, "1.3"},
		{2.5, 0, "3"},
		{0.125, 2, "0.13"},
		{1.005, 2, "1.00"},
		{64.36636, 2, "64.37"},
	}

	for _, tt := range tests {
		if got := ToFixed(tt.x, tt.digits); got != tt.want {
			t.Errorf("ToFixed(%v, %d) = %q, want %q", tt.x, tt.digits, got, tt.want)
		}
	}
}

func TestPercentageIsTheNumberOfToFixedTwo(t *testing.T) {
	if got := Percentage(26261172224, 40799043584); got != 64.37 {
		t.Errorf("Percentage = %v, want 64.37 as Immich reported for the same disk", got)
	}
	if got := Percentage(3, 10); got != 30 {
		t.Errorf("Percentage(3, 10) = %v, want 30", got)
	}
}
