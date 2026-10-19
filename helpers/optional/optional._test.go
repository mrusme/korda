package optional

import (
	"encoding/json"
	"testing"
)

type probe struct {
	Label Value[string] `json:"label"`
	Quota Value[int64]  `json:"quota"`
}

func TestValueTellsAbsentFromNullFromSet(t *testing.T) {
	var p probe
	if err := json.Unmarshal([]byte(`{"label":null,"quota":5}`), &p); err != nil {
		t.Fatal(err)
	}

	if !p.Label.Set || !p.Label.Null || p.Label.Valid() {
		t.Errorf("label = %+v, want set to null", p.Label)
	}
	if !p.Quota.Valid() || p.Quota.V != 5 {
		t.Errorf("quota = %+v, want 5", p.Quota)
	}

	var absent probe
	if err := json.Unmarshal([]byte(`{}`), &absent); err != nil || absent.Label.Set || absent.Quota.Set {
		t.Errorf("absent = %+v, %v", absent, err)
	}
}
