package user

import "testing"

func TestColorFromEmailMatchesImmich(t *testing.T) {
	tests := map[string]AvatarColor{
		"admin@example.com": ColorGray,
		"vera@example.com":  ColorOrange,
		"juno@example.com":  ColorPink,
		"é@x":               ColorOrange,
	}

	for email, want := range tests {
		if got := ColorFromEmail(email); got != want {
			t.Errorf("ColorFromEmail(%q) = %s, want %s", email, got, want)
		}
	}
}
