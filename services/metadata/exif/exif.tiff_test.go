package exif

import "testing"

func TestQuotient(t *testing.T) {
	cases := []struct {
		num, den int64
		want     float64
	}{
		{12499026, 2097152, 5.960000038},
		{42, 10, 4.2},
		{1, 3, 0.3333333333},
		{-2, 3, -0.6666666667},
		{1234567890123, 1, 1234567890000},
	}
	for _, c := range cases {
		if got := Quotient(c.num, c.den); got != c.want {
			t.Errorf("Quotient(%d, %d) = %v, want %v", c.num, c.den, got, c.want)
		}
	}
}
