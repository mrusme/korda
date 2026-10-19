package testmedia

import (
	"crypto/sha1"
	"encoding/base64"
	"testing"
)

func TestFixturesMatchTheReferenceUploads(t *testing.T) {
	for name, tc := range map[string]struct {
		build func() ([]byte, error)
		want  string
	}{
		"korda-exif.jpg":     {ExifJPEG, "7lWy+xd67RvuCrGVWCyajV1LYuU="},
		"korda-plain.jpg":    {PlainJPEG, "fH6Q5NN0jzG5Sav36O5+hEiU6o8="},
		"korda.png":          {PlainPNG, "RpvokZk7S0EdK8+aolHjenx6k5M="},
		"korda-mutate-1.jpg": {func() ([]byte, error) { return Mutate(10) }, "mBDPrU3k2Hi0wjYe35Z3FA3brmc="},
		"korda-mutate-2.jpg": {func() ([]byte, error) { return Mutate(90) }, "be04EOS7upt5NqjKvb7PCJ44yMA="},
		"korda-mutate-3.jpg": {func() ([]byte, error) { return Mutate(170) }, "aeUXefjEQklxVa72824+5YyrP4I="},
	} {
		data, err := tc.build()
		if err != nil {
			t.Fatal(err)
		}
		sum := sha1.Sum(data)
		if got := base64.StdEncoding.EncodeToString(sum[:]); got != tc.want {
			t.Errorf("%s checksum = %s, want %s", name, got, tc.want)
		}
	}
}
