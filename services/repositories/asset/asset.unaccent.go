package asset

import (
	"database/sql/driver"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
	"modernc.org/sqlite"
)

var unaccentLetters = strings.NewReplacer(
	"ß", "ss", "æ", "ae", "Æ", "AE", "ø", "o", "Ø", "O", "đ", "d", "Đ", "D", "ł", "l", "Ł", "L",
	"œ", "oe", "Œ", "OE", "þ", "th", "Þ", "TH", "ı", "i", "ð", "d", "Ð", "D",
)

func Fold(s string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(unaccentLetters.Replace(s)) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}

	return b.String()
}

func init() {
	sqlite.MustRegisterDeterministicScalarFunction("korda_fold", 1, func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
		switch v := args[0].(type) {
		case string:
			return Fold(v), nil
		case []byte:
			return Fold(string(v)), nil
		}

		return args[0], nil
	})
}
