package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/services/metadata"
)

const envReferenceFolder = "KORDA_REFERENCE_FOLDER"

var folderVideos = []string{".mov", ".mp4", ".m4v", ".3gp", ".mkv", ".webm", ".avi"}

type folderFile struct {
	name     string
	path     string
	video    bool
	cid      string
	modified string
	motion   *folderFile
	hidden   bool
}

func readFolder(t *testing.T, root string) []*folderFile {
	t.Helper()

	var files []*folderFile
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.HasPrefix(d.Name(), ".") {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		f := &folderFile{
			name:     d.Name(),
			path:     path,
			video:    slices.Contains(folderVideos, strings.ToLower(filepath.Ext(path))),
			modified: info.ModTime().UTC().Format("2006-01-02T15:04:05.000Z"),
		}
		kind := "IMAGE"
		if f.video {
			kind = "VIDEO"
		}
		tags, err := metadata.Read(context.Background(), path, "ffprobe", kind)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		f.cid = tags.ContentIdentifier
		files = append(files, f)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })

	motions := map[string]*folderFile{}
	for _, f := range files {
		if f.video && f.cid != "" {
			motions[f.cid] = f
		}
	}
	for _, f := range files {
		if m := motions[f.cid]; !f.video && f.cid != "" && m != nil {
			f.motion, m.hidden = m, true
		}
	}

	return files
}

func flattenJSON(prefix string, v any, out map[string]string) {
	switch x := v.(type) {
	case map[string]any:
		for k, val := range x {
			flattenJSON(prefix+"."+k, val, out)
		}
	case []any:
		for i, val := range x {
			flattenJSON(fmt.Sprintf("%s[%d]", prefix, i), val, out)
		}
	default:
		b, _ := json.Marshal(x)
		out[prefix] = string(b)
	}
}

func jsonDiff(got string, want string) []string {
	var a, b any
	if json.Unmarshal([]byte(got), &a) != nil || json.Unmarshal([]byte(want), &b) != nil {
		return []string{"korda  " + got, "Immich " + want}
	}
	fa, fb := map[string]string{}, map[string]string{}
	flattenJSON("", a, fa)
	flattenJSON("", b, fb)

	keys := map[string]bool{}
	for k := range fa {
		keys[k] = true
	}
	for k := range fb {
		keys[k] = true
	}
	var diff []string
	for k := range keys {
		ga, oka := fa[k]
		gb, okb := fb[k]
		if !oka {
			ga = "<missing>"
		}
		if !okb {
			gb = "<missing>"
		}
		if ga != gb {
			diff = append(diff, fmt.Sprintf("%s: korda %s, Immich %s", k, ga, gb))
		}
	}
	sort.Strings(diff)

	return diff
}

func TestConformanceFolder(t *testing.T) {
	base := strings.TrimSuffix(os.Getenv(envReferenceURL), "/")
	email := os.Getenv(envReferenceEmail)
	password := os.Getenv(envReferencePassword)
	root := os.Getenv(envReferenceFolder)
	if base == "" || email == "" || password == "" || root == "" {
		t.Skipf("set %s, %s, %s and %s to compare korda with an Immich on a folder of photos", envReferenceURL, envReferenceEmail, envReferencePassword, envReferenceFolder)
	}

	files := readFolder(t, root)
	if len(files) == 0 {
		t.Fatalf("no files in %s", root)
	}

	immich := immichSide(t, base)
	admin := immich.send(fiber.MethodPost, "/api/auth/login", fmt.Sprintf(`{"email":%q,"password":%q}`, email, password))
	if admin.Status != fiber.StatusCreated {
		t.Fatalf("login to %s = %d %s", base, admin.Status, admin.Body)
	}
	adminToken := admin.str(t, "accessToken")

	suffix := make([]byte, 4)
	if _, err := rand.Read(suffix); err != nil {
		t.Fatal(err)
	}
	userEmail := "conformance-" + hex.EncodeToString(suffix) + "@example.com"
	userPassword := "conformance-password"
	created := immich.send(fiber.MethodPost, "/api/admin/users",
		fmt.Sprintf(`{"email":%q,"password":%q,"name":"Conformance"}`, userEmail, userPassword), bearer(adminToken)...)
	if created.Status != fiber.StatusCreated {
		t.Fatalf("create Immich user = %d %s", created.Status, created.Body)
	}
	defer immich.send(fiber.MethodDelete, "/api/admin/users/"+created.str(t, "id"), `{"force":true}`, bearer(adminToken)...)
	immichToken := immich.send(fiber.MethodPost, "/api/auth/login", fmt.Sprintf(`{"email":%q,"password":%q}`, userEmail, userPassword)).str(t, "accessToken")

	h := newHarness(t)
	korda := kordaSide(h)
	kordaAdmin := h.admin()
	h.createUser(kordaAdmin, userEmail, "Conformance", userPassword)
	kordaToken := h.login(userEmail, userPassword)

	immichIDs, kordaIDs := map[*folderFile]string{}, map[*folderFile]string{}
	send := func(f *folderFile, extra ...string) {
		content, err := os.ReadFile(f.path)
		if err != nil {
			t.Fatal(err)
		}
		fields := append([]string{
			"fileCreatedAt", f.modified, "fileModifiedAt", f.modified, "isFavorite", "false", "duration", "0",
		}, extra...)
		var immichExtra, kordaExtra []string
		if f.motion != nil {
			immichExtra = []string{"livePhotoVideoId", immichIDs[f.motion]}
			kordaExtra = []string{"livePhotoVideoId", kordaIDs[f.motion]}
		}
		want := immichUpload(t, base, immichToken, f.name, content,
			append(append([]string{"deviceAssetId", "test-" + f.name, "deviceId", "korda-test"}, fields...), immichExtra...)...)
		got := h.upload(kordaToken, f.name, content, append(fields, kordaExtra...)...)
		if want.Status != got.Status || normalize(string(want.Body)) != normalize(string(got.Body)) {
			t.Errorf("upload %s = %d %s, Immich %d %s", f.name, got.Status, got.Body, want.Status, want.Body)
		}
		if want.Status >= 300 || got.Status >= 300 {
			t.Fatalf("upload %s failed", f.name)
		}
		immichIDs[f] = want.str(t, "id")
		kordaIDs[f] = got.str(t, "id")
	}
	motions := 0
	for _, f := range files {
		if f.hidden {
			continue
		}
		if f.motion != nil {
			send(f.motion, "visibility", "hidden")
			motions++
		}
		send(f)
	}
	t.Logf("uploaded %d files, %d of them live photo videos", len(files), motions)

	h.runAllJobs()
	deadline := time.Now().Add(10 * time.Minute)
	for _, f := range files {
		for {
			info := immich.send(fiber.MethodGet, "/api/assets/"+immichIDs[f], "", bearer(immichToken)...)
			body := string(info.Body)
			if info.Status == fiber.StatusOK && strings.Contains(body, `"dateTimeOriginal":"`) &&
				(f.hidden || !strings.Contains(body, `"thumbhash":null`)) {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("Immich did not process %s: %s", f.name, body)
			}
			time.Sleep(time.Second)
		}
	}

	for _, f := range files {
		t.Run(f.name, func(t *testing.T) {
			want := immich.send(fiber.MethodGet, "/api/assets/"+immichIDs[f], "", bearer(immichToken)...)
			got := korda.send(fiber.MethodGet, "/api/assets/"+kordaIDs[f], "", bearer(kordaToken)...)
			if want.Status != got.Status {
				t.Fatalf("status = %d, Immich %d\nkorda  %s\nImmich %s", got.Status, want.Status, got.Body, want.Body)
			}
			if diff := jsonDiff(normalizeConformance(string(got.Body)), normalizeConformance(string(want.Body))); len(diff) > 0 {
				t.Errorf("body differs:\n%s", strings.Join(diff, "\n"))
			}
		})
	}
}
