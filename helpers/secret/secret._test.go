package secret

import (
	"encoding/hex"
	"regexp"
	"testing"
)

func TestTokenIsBase64WithoutNonWordCharacters(t *testing.T) {
	word := regexp.MustCompile(`^\w+$`)
	seen := map[string]bool{}

	for range 200 {
		token := Token()
		if !word.MatchString(token) || len(token) < 30 || len(token) > 43 {
			t.Fatalf("token %q has a character outside \\w or a length outside 30 to 43", token)
		}
		if seen[token] {
			t.Fatalf("token %q came twice", token)
		}
		seen[token] = true
	}
}

func TestHashIsSHA256(t *testing.T) {
	if got := hex.EncodeToString(Hash("abc")); got != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
		t.Errorf("Hash(abc) = %s", got)
	}
}
