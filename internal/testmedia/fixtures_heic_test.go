//go:build heic

package testmedia

import (
	"crypto/sha1"
	"encoding/base64"
	"testing"
)

func TestHEICFixtureMatchesTheReferenceUpload(t *testing.T) {
	data, err := ExifHEIC()
	if err != nil {
		t.Fatal(err)
	}
	sum := sha1.Sum(data)
	if got := base64.StdEncoding.EncodeToString(sum[:]); got != "IPZsla7EF5sbFqM2meP6XyDcELc=" {
		t.Errorf("korda.heic checksum = %s", got)
	}
}
