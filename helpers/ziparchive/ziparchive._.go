package ziparchive

import (
	"archive/zip"
	"io"
	"os"
	"time"
)

const unixFileMode = 0o100644<<16 | 0x20

type Entry struct {
	Name string
	Path string
}

func dos(t time.Time) (uint16, uint16) {
	t = t.UTC()
	switch {
	case t.Year() < 1980:
		t = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
	case t.Year() > 2043:
		t = time.Date(2043, 12, 31, 23, 59, 58, 0, time.UTC)
	}

	date := uint16(t.Year()-1980)<<9 | uint16(t.Month())<<5 | uint16(t.Day())
	clock := uint16(t.Hour())<<11 | uint16(t.Minute())<<5 | uint16(t.Second()/2)

	return date, clock
}

func add(zw *zip.Writer, e Entry) error {
	f, err := os.Open(e.Path)
	if err != nil {
		return nil
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil
	}

	fh := &zip.FileHeader{Name: e.Name, Method: zip.Store, CreatorVersion: 3 << 8, ExternalAttrs: unixFileMode}
	//lint:ignore SA1019 Immich writes DOS times without the extended timestamp field that Modified adds
	fh.ModifiedDate, fh.ModifiedTime = dos(info.ModTime())

	w, err := zw.CreateHeader(fh)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, f)

	return err
}

func Write(out io.Writer, entries []Entry) error {
	zw := zip.NewWriter(out)
	for _, e := range entries {
		if err := add(zw, e); err != nil {
			return err
		}
	}

	return zw.Close()
}
