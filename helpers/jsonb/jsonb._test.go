package jsonb

import "testing"

func TestNormalizeMatchesImmichsRoundTrip(t *testing.T) {
	tests := map[string]string{
		`{"iCloudId":"x","createdAt":"c","adjustmentTime":"a","latitude":1,"longitude":2}`: `{"iCloudId":"x","latitude":1,"createdAt":"c","longitude":2,"adjustmentTime":"a"}`,
		`{"b":1,"10":2,"2":3,"a":{"zz":1,"y":2}}`:                                          `{"2":3,"10":2,"a":{"y":2,"zz":1},"b":1}`,
		`{"a":1,"a":2}`: `{"a":2}`,
		`{"n":[1.50,1e2,1.5e-3,-0,1e21,1e-7,123456789012345678]}`: `{"n":[1.5,100,0.0015,0,1e+21,1e-7,123456789012345680]}`,
		`{"s":"a\"b\\c\u0001/<> é"}`:                              `{"s":"a\"b\\c\u0001/<>` + " " + `é"}`,
		`{"big":1e400}`:                                           `{"big":null}`,
	}

	for in, want := range tests {
		got, err := Normalize([]byte(in))
		if err != nil {
			t.Fatalf("Normalize(%s): %v", in, err)
		}
		if string(got) != want {
			t.Errorf("Normalize(%s)\n got %s\nwant %s", in, got, want)
		}
	}
}
