package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/internal/testmedia"
)

var (
	sharingStamp = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})`)
	sharingKey   = regexp.MustCompile(`"key":"[A-Za-z0-9_-]+"`)
	sharingPath  = regexp.MustCompile(`"originalPath":"[^"]*"`)
	sharingHash  = regexp.MustCompile(`"thumbhash":"[^"]*"`)
)

func sharingNormalize(body string, days ...string) string {
	body = sharingStamp.ReplaceAllStringFunc(body, func(stamp string) string {
		for _, day := range days {
			if strings.HasPrefix(stamp, day) {
				if strings.HasSuffix(stamp, "Z") {
					return "<now Z>"
				}
				return "<now +00:00>"
			}
		}
		return stamp
	})
	body = sharingKey.ReplaceAllString(body, `"key":"<key>"`)
	body = sharingPath.ReplaceAllString(body, `"originalPath":"<path>"`)
	body = sharingHash.ReplaceAllString(body, `"thumbhash":"<thumbhash>"`)
	body = syncGeocoded.ReplaceAllString(body, `"$1":"<geo>"`)
	body = usagePattern.ReplaceAllString(body, `"quotaUsageInBytes":<usage>`)

	return uuidPattern.ReplaceAllString(body, "<uuid>")
}

func sharingStream(t *testing.T, body []byte, users map[string]bool, days ...string) string {
	t.Helper()

	var groups [][]string
	last := ""
	for raw := range bytes.SplitSeq(bytes.TrimSuffix(body, []byte("\n")), []byte("\n")) {
		var l struct {
			Type string         `json:"type"`
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(raw, &l); err != nil {
			t.Fatalf("line %q: %v", raw, err)
		}
		if l.Type == "UserDeleteV1" || (l.Type == "UserV1" && !users[fmt.Sprint(l.Data["id"])]) {
			continue
		}

		if l.Type != last {
			groups = append(groups, nil)
			last = l.Type
		}
		groups[len(groups)-1] = append(groups[len(groups)-1], sharingNormalize(string(raw), days...))
	}

	var out []string
	for _, g := range groups {
		slices.Sort(g)
		out = append(out, g...)
	}

	return strings.Join(out, "\n")
}

func unorderedAssets(t *testing.T, body []byte, days ...string) string {
	t.Helper()

	var link map[string]any
	if err := json.Unmarshal([]byte(sharingNormalize(string(body), days...)), &link); err != nil {
		return string(body)
	}
	assets, _ := link["assets"].([]any)
	slices.SortFunc(assets, func(a, b any) int {
		return strings.Compare(fmt.Sprint(a.(map[string]any)["originalFileName"]), fmt.Sprint(b.(map[string]any)["originalFileName"]))
	})
	encoded, err := json.Marshal(link)
	if err != nil {
		t.Fatal(err)
	}

	return string(encoded)
}

func TestConformanceSharing(t *testing.T) {
	base := strings.TrimSuffix(os.Getenv(envReferenceURL), "/")
	email := os.Getenv(envReferenceEmail)
	password := os.Getenv(envReferencePassword)
	if base == "" || email == "" || password == "" {
		t.Skipf("set %s, %s and %s to compare korda with an Immich", envReferenceURL, envReferenceEmail, envReferencePassword)
	}

	days := []string{time.Now().UTC().Format(time.DateOnly), time.Now().UTC().Add(time.Hour).Format(time.DateOnly)}

	immich := immichSide(t, base)
	admin := immich.send(fiber.MethodPost, "/api/auth/login", fmt.Sprintf(`{"email":%q,"password":%q}`, email, password))
	if admin.Status != fiber.StatusCreated {
		t.Fatalf("login to %s = %d %s", base, admin.Status, admin.Body)
	}
	adminToken := admin.str(t, "accessToken")
	defer immich.send(fiber.MethodPost, "/api/auth/logout", "", bearer(adminToken)...)

	h := newHarness(t)
	korda := kordaSide(h)
	kordaAdmin := h.admin()

	suffix := make([]byte, 4)
	if _, err := rand.Read(suffix); err != nil {
		t.Fatal(err)
	}
	userPassword := "conformance-password"
	for _, name := range []string{"owner", "member"} {
		userEmail := "conformance-" + name + "-" + hex.EncodeToString(suffix) + "@example.com"
		display := strings.ToUpper(name[:1]) + name[1:]
		created := immich.send(fiber.MethodPost, "/api/admin/users",
			fmt.Sprintf(`{"email":%q,"password":%q,"name":%q}`, userEmail, userPassword, display), bearer(adminToken)...)
		if created.Status != fiber.StatusCreated {
			t.Fatalf("create Immich user = %d %s", created.Status, created.Body)
		}
		id := created.str(t, "id")
		defer immich.send(fiber.MethodDelete, "/api/admin/users/"+id, `{"force":true}`, bearer(adminToken)...)
		immich.vars[name] = id
		immich.tokens[name] = immich.send(fiber.MethodPost, "/api/auth/login",
			fmt.Sprintf(`{"email":%q,"password":%q}`, userEmail, userPassword)).str(t, "accessToken")

		korda.vars[name] = h.createUser(kordaAdmin, userEmail, display, userPassword)
		korda.tokens[name] = h.login(userEmail, userPassword)
	}

	files := []struct {
		name  string
		build func() ([]byte, error)
		date  string
	}{
		{"exif", testmedia.ExifJPEG, "2024-05-06T05:00:00.000Z"},
		{"plain", testmedia.PlainJPEG, "2020-02-03T04:05:06.000Z"},
		{"png", testmedia.PlainPNG, "2021-01-02T03:04:05.000Z"},
	}
	names := map[string]string{"exif": "korda-exif.jpg", "plain": "korda-plain.jpg", "png": "korda.png"}
	for _, f := range files {
		content := fixture(t, f.build)
		fields := []string{"fileCreatedAt", f.date, "fileModifiedAt", f.date}
		immich.vars[f.name] = immichUpload(t, base, immich.tokens["owner"], names[f.name], content, fields...).str(t, "id")
		korda.vars[f.name] = h.upload(korda.tokens["owner"], names[f.name], content, fields...).str(t, "id")
	}
	h.runAllJobs()
	deadline := time.Now().Add(60 * time.Second)
	for _, f := range files {
		for {
			info := immich.send(fiber.MethodGet, "/api/assets/"+immich.vars[f.name], "", bearer(immich.tokens["owner"])...)
			if !strings.Contains(string(info.Body), `"thumbhash":null`) && strings.Contains(string(info.Body), `"dateTimeOriginal":"`) {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("Immich did not process %s: %s", f.name, info.Body)
			}
			time.Sleep(500 * time.Millisecond)
		}
	}

	users := func(s *side) map[string]bool {
		return map[string]bool{s.vars["owner"]: true, s.vars["member"]: true}
	}
	stream := func(s *side, user string) apiResponse {
		time.Sleep(3 * time.Millisecond)
		return s.send(fiber.MethodPost, "/api/sync/stream", syncBody(), bearer(s.tokens[user])...)
	}
	idle := func() {
		t.Helper()
		calm := 0
		for deadline := time.Now().Add(60 * time.Second); calm < 3; time.Sleep(300 * time.Millisecond) {
			if time.Now().After(deadline) {
				t.Fatal("Immich queues did not drain")
			}
			res := immich.send(fiber.MethodGet, "/api/queues", "", bearer(adminToken)...)
			var queues []struct {
				Statistics map[string]int `json:"statistics"`
			}
			if err := json.Unmarshal(res.Body, &queues); err != nil {
				t.Fatalf("queues = %d %s", res.Status, res.Body)
			}
			busy := false
			for _, q := range queues {
				busy = busy || q.Statistics["active"]+q.Statistics["waiting"] > 0
			}
			if busy {
				calm = 0
				continue
			}
			calm++
		}
		h.runAllJobs()
	}
	compareStreams := func(name string, user string) (apiResponse, apiResponse) {
		t.Helper()
		idle()
		want, got := stream(immich, user), stream(korda, user)
		if want.Status != fiber.StatusOK || got.Status != fiber.StatusOK {
			t.Fatalf("%s: korda %d %s, Immich %d %s", name, got.Status, got.Body, want.Status, want.Body)
		}
		a, b := sharingStream(t, got.Body, users(korda), days...), sharingStream(t, want.Body, users(immich), days...)
		if a != b {
			t.Errorf("%s differs\nkorda\n%s\nImmich\n%s", name, a, b)
		}
		return want, got
	}
	ack := func(user string, want apiResponse, got apiResponse) {
		t.Helper()
		for _, pair := range [][2]any{{immich, want}, {korda, got}} {
			s, res := pair[0].(*side), pair[1].(apiResponse)
			if r := s.send(fiber.MethodPost, "/api/sync/ack", syncAckRuns(res.Body), bearer(s.tokens[user])...); r.Status != fiber.StatusNoContent {
				t.Fatalf("%s ack = %d %s", s.name, r.Status, r.Body)
			}
		}
	}

	run := func(steps []step) {
		t.Helper()
		for _, st := range steps {
			t.Run(st.name, func(t *testing.T) {
				want := immich.run(t, st)
				got := korda.run(t, st)
				if want.Status != got.Status {
					t.Errorf("status = %d, Immich %d\nkorda  %s\nImmich %s", got.Status, want.Status, got.Body, want.Body)
					return
				}
				var a, b string
				switch st.mode {
				case compareShape:
					a, _ = shapeOf(got.Body)
					b, _ = shapeOf(want.Body)
				case compareUnordered:
					a, b = unorderedAssets(t, got.Body, days...), unorderedAssets(t, want.Body, days...)
				default:
					a, b = sharingNormalize(string(got.Body), days...), sharingNormalize(string(want.Body), days...)
				}
				if a != b {
					t.Errorf("body differs\nkorda  %s\nImmich %s", a, b)
				}
			})
		}
	}

	run([]step{
		{name: "later album", method: "POST", path: "/api/albums", body: `{"albumName":"Later","assetIds":["{png}"]}`, auth: "owner", capture: "later"},
		{name: "early album", method: "POST", path: "/api/albums", body: `{"albumName":"Early","assetIds":["{plain}"],"albumUsers":[{"userId":"{member}","role":"viewer"}]}`, auth: "owner", capture: "early"},
	})

	want, got := compareStreams("member before sharing", "member")
	ack("member", want, got)
	want, got = compareStreams("owner before sharing", "owner")
	ack("owner", want, got)

	run([]step{
		{name: "album create invalid", method: "POST", path: "/api/albums", body: `{"albumName":5,"albumUsers":[{"userId":"x","role":"boss"}]}`, auth: "owner"},
		{name: "album create", method: "POST", path: "/api/albums", body: `{"albumName":"Holiday","description":"Summer","assetIds":["{exif}","{plain}"]}`, auth: "owner", capture: "album"},
		{name: "late member", method: "PUT", path: "/api/albums/{later.id}/users", body: `{"albumUsers":[{"userId":"{member}","role":"viewer"}]}`, auth: "owner"},
		{name: "album add member", method: "PUT", path: "/api/albums/{album.id}/users", body: `{"albumUsers":[{"userId":"{member}","role":"editor"}]}`, auth: "owner"},
		{name: "album add member again", method: "PUT", path: "/api/albums/{album.id}/users", body: `{"albumUsers":[{"userId":"{member}","role":"viewer"}]}`, auth: "owner"},
		{name: "album add owner", method: "PUT", path: "/api/albums/{album.id}/users", body: `{"albumUsers":[{"userId":"{owner}","role":"editor"}]}`, auth: "owner"},
		{name: "albums of owner", method: "GET", path: "/api/albums", auth: "owner"},
		{name: "albums shared with member", method: "GET", path: "/api/albums?shared=true", auth: "member"},
		{name: "album as member", method: "GET", path: "/api/albums/{album.id}", auth: "member"},
		{name: "album statistics owner", method: "GET", path: "/api/albums/statistics", auth: "owner"},
		{name: "album statistics member", method: "GET", path: "/api/albums/statistics", auth: "member"},
		{name: "album add assets", method: "PUT", path: "/api/albums/{album.id}/assets", body: `{"ids":["{png}","{exif}"]}`, auth: "owner"},
		{name: "album add assets as member", method: "PUT", path: "/api/albums/{album.id}/assets", body: `{"ids":["{png}"]}`, auth: "member"},
		{name: "album update", method: "PATCH", path: "/api/albums/{album.id}", body: `{"albumName":"Holiday 2026","order":"asc"}`, auth: "owner"},
		{name: "album update as member", method: "PATCH", path: "/api/albums/{album.id}", body: `{"albumName":"Mine"}`, auth: "member"},
		{name: "album map markers", method: "GET", path: "/api/albums/{album.id}/map-markers", auth: "member"},
		{name: "comment", method: "POST", path: "/api/activities", body: `{"albumId":"{album.id}","type":"comment","comment":"Nice"}`, auth: "member", capture: "comment"},
		{name: "like", method: "POST", path: "/api/activities", body: `{"albumId":"{album.id}","assetId":"{exif}","type":"like"}`, auth: "member"},
		{name: "like again", method: "POST", path: "/api/activities", body: `{"albumId":"{album.id}","assetId":"{exif}","type":"like"}`, auth: "member"},
		{name: "comment invalid", method: "POST", path: "/api/activities", body: `{"albumId":"{album.id}","type":"comment"}`, auth: "member"},
		{name: "activities", method: "GET", path: "/api/activities?albumId={album.id}", auth: "owner"},
		{name: "activity statistics", method: "GET", path: "/api/activities/statistics?albumId={album.id}&assetId={exif}", auth: "owner"},
		{name: "partner create", method: "POST", path: "/api/partners", body: `{"sharedWithId":"{member}"}`, auth: "owner"},
		{name: "partner create again", method: "POST", path: "/api/partners", body: `{"sharedWithId":"{member}"}`, auth: "owner"},
		{name: "partners shared by", method: "GET", path: "/api/partners?direction=shared-by", auth: "owner"},
		{name: "partners shared with", method: "GET", path: "/api/partners?direction=shared-with", auth: "member"},
		{name: "partner timeline", method: "PUT", path: "/api/partners/{owner}", body: `{"inTimeline":true}`, auth: "member"},
		{name: "partner asset", method: "GET", path: "/api/assets/{exif}", auth: "member"},
		{name: "stack create", method: "POST", path: "/api/stacks", body: `{"assetIds":["{plain}","{exif}"]}`, auth: "owner", capture: "stack"},
		{name: "stack too small", method: "POST", path: "/api/stacks", body: `{"assetIds":["{png}"]}`, auth: "owner"},
		{name: "stacks", method: "GET", path: "/api/stacks", auth: "owner"},
		{name: "stack get", method: "GET", path: "/api/stacks/{stack.id}", auth: "owner"},
		{name: "stacked asset", method: "GET", path: "/api/assets/{exif}", auth: "owner"},
		{name: "tag create", method: "POST", path: "/api/tags", body: `{"name":"Travel","color":"ABC"}`, auth: "owner", capture: "tag"},
		{name: "tag create again", method: "POST", path: "/api/tags", body: `{"name":"Travel"}`, auth: "owner"},
		{name: "tag create slash", method: "POST", path: "/api/tags", body: `{"name":"a/b"}`, auth: "owner"},
		{name: "tags upsert", method: "PUT", path: "/api/tags", body: `{"tags":["Travel/Japan","Travel 2026","travel"]}`, auth: "owner"},
		{name: "tags", method: "GET", path: "/api/tags", auth: "owner"},
		{name: "tag assets", method: "PUT", path: "/api/tags/{tag.id}/assets", body: `{"ids":["{exif}","{exif}","00000000-0000-4000-8000-000000000000"]}`, auth: "owner"},
		{name: "tagged asset", method: "GET", path: "/api/assets/{exif}", auth: "owner"},
		{name: "link create invalid", method: "POST", path: "/api/shared-links", body: `{"type":"ALBUM","assetIds":["{exif}"]}`, auth: "owner"},
		{name: "link individual", method: "POST", path: "/api/shared-links", body: `{"type":"INDIVIDUAL","assetIds":["{exif}","{plain}"],"description":""}`, auth: "owner", capture: "ind", mode: compareUnordered},
		{name: "link album", method: "POST", path: "/api/shared-links", body: `{"type":"ALBUM","albumId":"{album.id}","password":"secret","showMetadata":false,"expiresAt":"2030-01-01T10:00:00+02:00"}`, auth: "owner", capture: "alb"},
		{name: "links", method: "GET", path: "/api/shared-links", auth: "owner"},
		{name: "link get", method: "GET", path: "/api/shared-links/{alb.id}", auth: "owner"},
		{name: "link me individual", method: "GET", path: "/api/shared-links/me?key={ind.key}"},
		{name: "link me without password", method: "GET", path: "/api/shared-links/me?key={alb.key}"},
		{name: "link login wrong", method: "POST", path: "/api/shared-links/login?key={alb.key}", body: `{"password":"nope"}`},
		{name: "link login", method: "POST", path: "/api/shared-links/login?key={alb.key}", body: `{"password":"secret"}`},
		{name: "link asset stripped", method: "GET", path: "/api/assets/{exif}?key={alb.key}"},
		{name: "link asset", method: "GET", path: "/api/assets/{exif}?key={ind.key}"},
		{name: "link album info", method: "GET", path: "/api/albums/{album.id}?key={alb.key}"},
		{name: "link markers", method: "GET", path: "/api/albums/{album.id}/map-markers?key={alb.key}"},
		{name: "link forbidden", method: "GET", path: "/api/albums?key={ind.key}"},
		{name: "link add assets", method: "PUT", path: "/api/shared-links/{ind.id}/assets", body: `{"assetIds":["{exif}","{png}"]}`, auth: "owner"},
		{name: "link remove assets", method: "DELETE", path: "/api/shared-links/{ind.id}/assets", body: `{"assetIds":["{png}","{png}"]}`, auth: "owner"},
		{name: "link update", method: "PATCH", path: "/api/shared-links/{ind.id}", body: `{"description":"","allowUpload":false}`, auth: "owner", mode: compareUnordered},
		{name: "memory create invalid", method: "POST", path: "/api/memories", body: `{"type":"birthday","data":{"year":1},"memoryAt":"x"}`, auth: "owner"},
		{name: "memory create", method: "POST", path: "/api/memories", body: `{"type":"on_this_day","data":{"year":2024},"memoryAt":"2024-05-06T00:00:00.000Z","assetIds":["{exif}","{png}"]}`, auth: "owner", capture: "memory"},
		{name: "memories", method: "GET", path: "/api/memories", auth: "owner"},
		{name: "memory statistics", method: "GET", path: "/api/memories/statistics", auth: "owner"},
		{name: "memory update empty", method: "PUT", path: "/api/memories/{memory.id}", body: `{}`, auth: "owner"},
		{name: "memory update", method: "PUT", path: "/api/memories/{memory.id}", body: `{"isSaved":true}`, auth: "owner"},
		{name: "memory as member", method: "GET", path: "/api/memories/{memory.id}", auth: "member"},
		{name: "second album", method: "POST", path: "/api/albums", body: `{"albumName":"Second","albumUsers":[{"userId":"{member}","role":"viewer"}],"assetIds":["{exif}"]}`, auth: "owner", capture: "second"},
		{name: "second album as viewer", method: "PUT", path: "/api/albums/{second.id}/assets", body: `{"ids":["{png}"]}`, auth: "member"},
		{name: "albums of member", method: "GET", path: "/api/albums", auth: "member"},
	})

	want, got = compareStreams("member after sharing", "member")
	for _, kind := range []string{"AlbumAssetBackfillV2", "AlbumUserBackfillV1", "AlbumToAssetBackfillV1", "AlbumAssetExifBackfillV1"} {
		if !strings.Contains(string(want.Body), `"`+kind+`"`) || !strings.Contains(string(got.Body), `"`+kind+`"`) {
			t.Errorf("member stream after sharing has no %s line", kind)
		}
	}
	compareStreams("owner after sharing", "owner")

	run([]step{
		{name: "member leaves", method: "DELETE", path: "/api/albums/{album.id}/user/me", auth: "member"},
		{name: "stack delete", method: "DELETE", path: "/api/stacks/{stack.id}", auth: "owner"},
		{name: "partner remove", method: "DELETE", path: "/api/partners/{member}", auth: "owner"},
		{name: "memory delete", method: "DELETE", path: "/api/memories/{memory.id}", auth: "owner"},
		{name: "tag delete", method: "DELETE", path: "/api/tags/{tag.id}", auth: "owner"},
		{name: "link delete", method: "DELETE", path: "/api/shared-links/{ind.id}", auth: "owner"},
		{name: "second album delete as member", method: "DELETE", path: "/api/albums/{second.id}", auth: "member"},
		{name: "second album delete", method: "DELETE", path: "/api/albums/{second.id}", auth: "owner"},
		{name: "asset force delete", method: "DELETE", path: "/api/assets", body: `{"ids":["{png}"],"force":true}`, auth: "owner"},
	})

	compareStreams("member after removals", "member")
	want, got = compareStreams("owner after removals", "owner")
	for _, kind := range []string{"AlbumDeleteV1", "AssetDeleteV1", "AlbumToAssetDeleteV1", "StackDeleteV1", "MemoryDeleteV1", "PartnerDeleteV1"} {
		if !strings.Contains(string(want.Body), `"`+kind+`"`) || !strings.Contains(string(got.Body), `"`+kind+`"`) {
			t.Errorf("owner stream after removals has no %s line", kind)
		}
	}
}
