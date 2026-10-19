package checksum

import (
	"encoding/base64"
	"encoding/hex"
	"strings"
)

func Decode(s string) []byte {
	if len(s) == 28 {
		var filtered strings.Builder
		for _, r := range s {
			switch {
			case r == '=':
				return decodeBase64(filtered.String())
			case r == '-':
				filtered.WriteByte('+')
			case r == '_':
				filtered.WriteByte('/')
			case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '+', r == '/':
				filtered.WriteRune(r)
			}
		}
		return decodeBase64(filtered.String())
	}

	out := make([]byte, 0, len(s)/2)
	for i := 0; i+1 < len(s); i += 2 {
		b, err := hex.DecodeString(s[i : i+2])
		if err != nil {
			break
		}
		out = append(out, b...)
	}

	return out
}

func decodeBase64(s string) []byte {
	if len(s)%4 == 1 {
		s = s[:len(s)-1]
	}
	out, err := base64.RawStdEncoding.DecodeString(s)
	if err != nil {
		return nil
	}

	return out
}
