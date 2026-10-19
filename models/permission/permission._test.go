package permission

import "testing"

func TestIsGranted(t *testing.T) {
	tests := []struct {
		name      string
		requested []Permission
		current   []Permission
		want      bool
	}{
		{"all grants everything", []Permission{UserRead, SessionDelete}, []Permission{All}, true},
		{"a subset is granted", []Permission{UserRead}, []Permission{UserRead, SessionRead}, true},
		{"a missing permission is not", []Permission{UserRead, SessionDelete}, []Permission{UserRead}, false},
		{"all is only granted by all", []Permission{All}, []Permission{UserRead}, false},
		{"nothing requested is granted", nil, []Permission{UserRead}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsGranted(tt.requested, tt.current); got != tt.want {
				t.Errorf("IsGranted(%v, %v) = %v, want %v", tt.requested, tt.current, got, tt.want)
			}
		})
	}
}

func TestListRoundTripsAsJSON(t *testing.T) {
	value, err := List{UserRead, AssetUpload}.Value()
	if err != nil || value != `["user.read","asset.upload"]` {
		t.Fatalf("Value = %v, %v", value, err)
	}

	var l List
	if err = l.Scan(value); err != nil || len(l) != 2 || l[1] != AssetUpload {
		t.Errorf("Scan = %v, %v", l, err)
	}

	if value, _ = List(nil).Value(); value != "[]" {
		t.Errorf("a nil list is stored as %v, want []", value)
	}
}

func TestValuesKeepImmichsOrder(t *testing.T) {
	v := Values()
	if len(v) != 168 || v[0] != "all" || v[len(v)-1] != "adminAuth.unlinkAll" {
		t.Errorf("Values has %d entries from %q to %q", len(v), v[0], v[len(v)-1])
	}
}
