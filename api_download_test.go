package main

import (
	"archive/zip"
	"bytes"
	"strconv"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/internal/testmedia"
	"xn--gckvb8fzb.com/glides/services/database"
)

func TestDownloadInfo(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	exif := h.uploadID(admin, "korda-exif.jpg", fixture(t, testmedia.ExifJPEG), "2024-05-06T05:00:00.000Z", "2024-05-06T05:00:00.000Z")
	plain := h.uploadID(admin, "korda-plain.jpg", fixture(t, testmedia.PlainJPEG), "2023-01-02T03:04:05.000Z", "2023-01-02T03:04:05.000Z")
	h.runAllJobs()
	sizes := map[string]int64{}
	for _, id := range []string{exif, plain} {
		sizes[id] = int64(h.do(fiber.MethodGet, "/api/assets/"+id, nil, owner...).object(t)["exifInfo"].(map[string]any)["fileSizeInByte"].(float64))
	}
	total := sizes[exif] + sizes[plain]
	itoa := func(n int64) string { return strconv.FormatInt(n, 10) }

	h.do(fiber.MethodPost, "/api/download/info", `{"assetIds":["`+plain+`","`+exif+`"]}`, owner...).
		expect(t, 201, `{"totalSize":`+itoa(total)+`,"archives":[{"size":`+itoa(total)+`,"assetIds":["`+exif+`","`+plain+`"]}]}`)
	h.do(fiber.MethodPost, "/api/download/info", `{"assetIds":["`+exif+`","`+plain+`"],"archiveSize":1}`, owner...).
		expect(t, 201, `{"totalSize":`+itoa(total)+`,"archives":[{"size":`+itoa(sizes[exif])+`,"assetIds":["`+exif+`"]},{"size":`+itoa(sizes[plain])+`,"assetIds":["`+plain+`"]}]}`)
	h.do(fiber.MethodPost, "/api/download/info", `{"assetIds":[]}`, owner...).expect(t, 201, `{"totalSize":0,"archives":[]}`)
	h.do(fiber.MethodPost, "/api/download/info", `{}`, owner...).expect(t, 400, `{"message":"assetIds, albumId, or userId is required"}`)
	h.do(fiber.MethodPost, "/api/download/info", `{"archiveSize":0}`, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"origin":"number","code":"too_small","minimum":1,"inclusive":true,"path":["archiveSize"],"message":"Too small: expected number to be >=1"}]}`)
	h.do(fiber.MethodPost, "/api/download/info", `{"archiveSize":1.5}`, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"expected":"int","format":"safeint","code":"invalid_type","path":["archiveSize"],"message":"Invalid input: expected int, received number"}]}`)

	adminID := h.do(fiber.MethodGet, "/api/users/me", nil, owner...).str(t, "id")
	h.do(fiber.MethodPost, "/api/download/info", `{"userId":"`+adminID+`"}`, owner...).
		expect(t, 201, `{"totalSize":`+itoa(total)+`,"archives":[{"size":`+itoa(total)+`,"assetIds":["`+exif+`","`+plain+`"]}]}`)
	if _, err := database.Exec(h.ctx, database.From(h.rt), `UPDATE assets SET visibility = 'locked' WHERE id = $1`, plain); err != nil {
		t.Fatal(err)
	}
	h.do(fiber.MethodPost, "/api/download/info", `{"userId":"`+adminID+`"}`, owner...).
		expect(t, 201, `{"totalSize":`+itoa(sizes[exif])+`,"archives":[{"size":`+itoa(sizes[exif])+`,"assetIds":["`+exif+`"]}]}`)

	h.createUser(admin, "mira@example.com", "Mira", "mira-password")
	member := bearer(h.login("mira@example.com", "mira-password"))
	h.do(fiber.MethodPost, "/api/download/info", `{"userId":"`+adminID+`"}`, member...).
		expect(t, 400, `{"message":"Not found or no timeline.download access"}`)
	h.do(fiber.MethodPost, "/api/download/info", `{"assetIds":["`+exif+`"]}`, member...).
		expect(t, 400, `{"message":"Not found or no asset.download access"}`)
}

func TestDownloadArchive(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	exif := h.uploadID(admin, "korda-exif.jpg", fixture(t, testmedia.ExifJPEG), "2024-05-06T05:00:00.000Z", "2024-05-06T05:00:01.000Z")
	plain := h.uploadID(admin, "Ünïcødé ñame.jpg", fixture(t, testmedia.PlainJPEG), "2023-01-02T03:04:05.000Z", "2023-01-02T03:04:05.000Z")
	h.runAllJobs()

	res := h.do(fiber.MethodPost, "/api/download/archive", `{"assetIds":["`+exif+`","`+plain+`","`+exif+`"]}`, owner...)
	if res.Status != 200 || res.Header.Get(fiber.HeaderContentType) != "application/octet-stream" ||
		res.Header.Get(fiber.HeaderContentLength) != "" || len(res.Header.Values(fiber.HeaderContentDisposition)) != 0 {
		t.Fatalf("archive = %d %v", res.Status, res.Header)
	}
	zr, err := zip.NewReader(bytes.NewReader(res.Body), int64(len(res.Body)))
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		name  string
		flags uint16
		at    time.Time
		data  []byte
	}{
		{"korda-exif.jpg", 0x0008, time.Date(2024, 5, 6, 5, 0, 0, 0, time.UTC), fixture(t, testmedia.ExifJPEG)},
		{"Ünïcødé ñame.jpg", 0x0808, time.Date(2023, 1, 2, 3, 4, 4, 0, time.UTC), fixture(t, testmedia.PlainJPEG)},
		{"korda-exif+1.jpg", 0x0008, time.Date(2024, 5, 6, 5, 0, 0, 0, time.UTC), fixture(t, testmedia.ExifJPEG)},
	}
	if len(zr.File) != len(want) {
		t.Fatalf("entries = %d", len(zr.File))
	}
	for i, f := range zr.File {
		w := want[i]
		if f.Name != w.name || f.Method != zip.Store || f.Flags != w.flags || f.ExternalAttrs != 0x81A40020 ||
			f.CreatorVersion>>8 != 3 || f.ReaderVersion != 20 || len(f.Extra) != 0 || !f.Modified.Equal(w.at) {
			t.Errorf("entry %d = %q method %d flags %#x attrs %#x creator %#x reader %d extra %d time %v",
				i, f.Name, f.Method, f.Flags, f.ExternalAttrs, f.CreatorVersion, f.ReaderVersion, len(f.Extra), f.Modified)
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		var data bytes.Buffer
		if _, err = data.ReadFrom(rc); err != nil {
			t.Fatal(err)
		}
		rc.Close()
		if !bytes.Equal(data.Bytes(), w.data) {
			t.Errorf("entry %d has %d bytes, want %d", i, data.Len(), len(w.data))
		}
	}

	form := formRequest("/api/download/archive", "assetIds="+exif+","+plain+"&archiveName=Fotos+M%C3%A4rz&edited=true")
	form.Header.Set(fiber.HeaderAuthorization, owner[1])
	if res := h.send(form); res.Status != 200 || res.Header.Get(fiber.HeaderContentDisposition) != "attachment; filename*=UTF-8''Fotos%20M%C3%A4rz.zip" {
		t.Errorf("form archive = %d %v", res.Status, res.Header)
	}
	if res := h.do(fiber.MethodPost, "/api/download/archive", `{"assetIds":["`+exif+`"],"archiveName":""}`, owner...); res.Status != 200 ||
		len(res.Header.Values(fiber.HeaderContentDisposition)) != 1 || res.Header.Get(fiber.HeaderContentDisposition) != "" {
		t.Errorf("archive with an empty name = %d %v", res.Status, res.Header)
	}

	h.do(fiber.MethodPost, "/api/download/archive", `{"assetIds":[]}`, owner...).expect(t, 500, `{"message":"Failed to download archive"}`)
	h.do(fiber.MethodPost, "/api/download/archive", `{}`, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"expected":"array","code":"invalid_type","path":["assetIds"],"message":"Invalid input: expected array, received undefined"}]}`)
	h.do(fiber.MethodPost, "/api/download/archive", `{"assetIds":"`+exif+`, `+plain+`","edited":"TRUE"}`, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"origin":"string","code":"invalid_format","format":"uuid","pattern":"/^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})$/","path":["assetIds",1],"message":"Invalid UUID"},{"expected":"boolean","code":"invalid_type","path":["edited"],"message":"Invalid input: expected boolean, received string"}]}`)
}
