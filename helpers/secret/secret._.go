package secret

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

func Random(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)

	return b
}

func Token() string {
	encoded := base64.StdEncoding.EncodeToString(Random(32))

	return strings.Map(func(r rune) rune {
		if r == '+' || r == '/' || r == '=' {
			return -1
		}
		return r
	}, encoded)
}

func Hash(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
