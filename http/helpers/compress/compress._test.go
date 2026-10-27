package compress

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
)

func TestNegotiateMatchesNegotiator(t *testing.T) {
	cases := []struct {
		header string
		want   string
	}{
		{"", "identity"},
		{"gzip", "gzip"},
		{"br", "br"},
		{"deflate", "deflate"},
		{"identity", "identity"},
		{"*", "br"},
		{"gzip, deflate, br", "br"},
		{"gzip, deflate", "gzip"},
		{"deflate, gzip", "gzip"},
		{"br;q=0.5, gzip", "gzip"},
		{"gzip;q=0.5, br;q=0.5", "br"},
		{"gzip;q=0", "identity"},
		{"identity;q=0", ""},
		{"*;q=0", ""},
		{"gzip;q=0, *", "br"},
		{"br;q=0.9, gzip;q=0.9, deflate;q=1", "deflate"},
		{"compress", "identity"},
		{"x-gzip", "identity"},
		{"GZIP", "gzip"},
		{"gzip;q=abc", "identity"},
		{"gzip;q=0.5abc, br;q=0.4", "gzip"},
		{"deflate;q=0.5, *;q=0.1", "deflate"},
		{"identity, gzip", "gzip"},
		{"*, identity;q=0", "br"},
		{" gzip ; q=1 , br ; q = 0.8", "br"},
		{"gzip;level=1;q=0.8, deflate", "deflate"},
		{",,gzip,,", "gzip"},
		{"gzip;q=1.5, br", "gzip"},
		{"br;q=-1, gzip", "gzip"},
	}
	for _, c := range cases {
		if got := Negotiate(c.header); got != c.want {
			t.Errorf("Negotiate(%q) = %q, want %q", c.header, got, c.want)
		}
	}
}

func TestCompressibleMatchesMimeDB(t *testing.T) {
	raw, err := os.ReadFile("testdata/compressible.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Compressible    []string `json:"compressible"`
		NotCompressible []string `json:"notCompressible"`
	}
	if err = json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}

	for _, ct := range fixture.Compressible {
		if !Compressible(ct) {
			t.Errorf("Compressible(%q) = false, want true", ct)
		}
	}
	for _, ct := range fixture.NotCompressible {
		if Compressible(ct) {
			t.Errorf("Compressible(%q) = true, want false", ct)
		}
	}
}

func TestVary(t *testing.T) {
	cases := []struct {
		header string
		want   string
	}{
		{"", "Accept-Encoding"},
		{"Origin", "Origin, Accept-Encoding"},
		{"origin, accept-encoding", "origin, accept-encoding"},
		{"*", "*"},
		{"Origin, *", "*"},
	}
	for _, c := range cases {
		if got := Vary(c.header, "Accept-Encoding"); got != c.want {
			t.Errorf("Vary(%q) = %q, want %q", c.header, got, c.want)
		}
	}
}

type writes struct {
	parts []string
}

func (w *writes) Write(p []byte) (int, error) {
	w.parts = append(w.parts, string(p))
	return len(p), nil
}

func TestBodyWritesEachPieceOnce(t *testing.T) {
	lines := []string{"{\"a\":1}\n", strings.Repeat("x", 20000) + "\n", "{\"b\":2}\n"}
	body := Stream(func(w io.Writer) error {
		for _, l := range lines {
			if _, err := w.Write([]byte(l)); err != nil {
				return err
			}
		}
		return nil
	})

	w := &writes{}
	n, err := body.WriteTo(w)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(w.parts, "|") != strings.Join(lines, "|") {
		t.Errorf("writes = %d pieces, want the %d lines as written", len(w.parts), len(lines))
	}
	if want := int64(len(strings.Join(lines, ""))); n != want {
		t.Errorf("WriteTo = %d bytes, want %d", n, want)
	}
}

func TestBodyEncodings(t *testing.T) {
	payload := []byte(strings.Repeat(`{"type":"AssetV1","data":{"id":"x"}}`+"\n", 2000))
	decoders := map[string]func(io.Reader) (io.Reader, error){
		"gzip":    func(r io.Reader) (io.Reader, error) { return gzip.NewReader(r) },
		"deflate": func(r io.Reader) (io.Reader, error) { return zlib.NewReader(r) },
		"br":      func(r io.Reader) (io.Reader, error) { return brotli.NewReader(r), nil },
	}
	for encoding, decode := range decoders {
		body := Bytes(payload)
		body.SetEncoding(encoding)

		var buf bytes.Buffer
		if _, err := body.WriteTo(&buf); err != nil {
			t.Fatalf("%s: %v", encoding, err)
		}
		r, err := decode(&buf)
		if err != nil {
			t.Fatalf("%s: %v", encoding, err)
		}
		got, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("%s: %v", encoding, err)
		}
		if !bytes.Equal(got, payload) {
			t.Errorf("%s: decoded %d bytes, want %d", encoding, len(got), len(payload))
		}

		body = Bytes(payload)
		body.SetEncoding(encoding)
		read, err := io.ReadAll(body)
		if err != nil || len(read) == 0 {
			t.Errorf("%s: Read = %d bytes, %v", encoding, len(read), err)
		}
	}
}
