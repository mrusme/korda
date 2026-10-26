package systemconfig

import "testing"

func TestOriginMatchesWHATWG(t *testing.T) {
	for in, want := range map[string]string{
		"":                                     "",
		"https://photos.example.com":           "https://photos.example.com",
		"https://Photos.Example.COM:443/x?y#z": "https://photos.example.com",
		"HTTP://photos.example.com:80":         "http://photos.example.com",
		"http://photos.example.com:8080/":      "http://photos.example.com:8080",
		"https://[::1]:443":                    "https://[::1]",
		"https://[::1]:8443/":                  "https://[::1]:8443",
		"https://ann:secret@example.com:443/":  "https://ann:secret@example.com",
		"https://ann@example.com/":             "https://example.com",
		"custom://example.com/":                "null",
	} {
		if got := Origin(in); got != want {
			t.Errorf("Origin(%q) = %q, want %q", in, got, want)
		}
	}
}
