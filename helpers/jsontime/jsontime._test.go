package jsontime

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"
)

func TestMarshalIsJavaScriptsToISOString(t *testing.T) {
	local := time.FixedZone("CEST", 2*60*60)
	value := New(time.Date(2026, 10, 19, 16, 47, 22, 900_123_456, local))

	encoded, err := json.Marshal(value)
	if err != nil || string(encoded) != `"2026-10-19T14:47:22.900Z"` {
		t.Errorf("Marshal = %s, %v", encoded, err)
	}

	if encoded, _ = json.Marshal(New(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))); string(encoded) != `"2026-01-02T03:04:05.000Z"` {
		t.Errorf("whole seconds = %s, want three zero digits", encoded)
	}
}

func TestUnmarshalTakesEveryRFC3339Form(t *testing.T) {
	want := time.Date(2026, 10, 19, 14, 47, 22, 0, time.UTC)

	for _, raw := range []string{
		`"2026-10-19T14:47:22Z"`,
		`"2026-10-19T14:47:22.000Z"`,
		`"2026-10-19T16:47:22+02:00"`,
		`"2026-10-19T14:47:22.000000+00:00"`,
	} {
		var got Time
		if err := json.Unmarshal([]byte(raw), &got); err != nil || !got.Equal(want) || got.Location() != time.UTC {
			t.Errorf("Unmarshal(%s) = %v, %v", raw, got, err)
		}
	}
}

func TestNullable(t *testing.T) {
	if Nullable(sql.Null[time.Time]{}) != nil {
		t.Error("an invalid time is not nil")
	}

	encoded, _ := json.Marshal(struct {
		At *Time `json:"at"`
	}{})
	if string(encoded) != `{"at":null}` {
		t.Errorf("a nil time encodes as %s", encoded)
	}
}
