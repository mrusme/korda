package main

import (
	"encoding/json"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	gorilla "github.com/gorilla/websocket"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/internal/testmedia"
)

type socketClient struct {
	t    *testing.T
	conn *gorilla.Conn
}

func (h *apiHarness) listen() string {
	h.t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		h.t.Fatal(err)
	}
	go func() { _ = h.app.Listener(ln, fiber.ListenConfig{DisableStartupMessage: true}) }()
	h.t.Cleanup(func() { _ = h.app.Shutdown() })

	return ln.Addr().String()
}

func dialSocket(t *testing.T, addr string, headers http.Header) *socketClient {
	t.Helper()

	conn, res, err := gorilla.DefaultDialer.Dial("ws://"+addr+"/api/socket.io/?EIO=4&transport=websocket", headers)
	if err != nil {
		t.Fatalf("dial = %v %v", res, err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return &socketClient{t: t, conn: conn}
}

func (c *socketClient) read() string {
	c.t.Helper()

	if err := c.conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		c.t.Fatal(err)
	}
	_, data, err := c.conn.ReadMessage()
	if err != nil {
		c.t.Fatalf("read: %v", err)
	}

	return string(data)
}

func (c *socketClient) send(frame string) {
	c.t.Helper()

	if err := c.conn.WriteMessage(gorilla.TextMessage, []byte(frame)); err != nil {
		c.t.Fatal(err)
	}
}

func (c *socketClient) connect() string {
	c.t.Helper()

	if open := c.read(); !strings.HasPrefix(open, `0{"`) || !strings.Contains(open, `"pingInterval":25000`) ||
		!strings.Contains(open, `"pingTimeout":20000`) || !strings.Contains(open, `"upgrades":[]`) {
		c.t.Fatalf("open packet = %s", open)
	}
	c.send("40")
	if reply := c.read(); !strings.HasPrefix(reply, `40{"sid":"`) {
		c.t.Fatalf("connect reply = %s", reply)
	}

	return c.read()
}

func TestWebsocketHandshake(t *testing.T) {
	h := newHarness(t)
	token := h.admin()
	addr := h.listen()

	anonymous := dialSocket(t, addr, nil)
	if got := anonymous.connect(); got != `42["error","unauthorized"]` {
		t.Errorf("anonymous = %s", got)
	}
	if got := anonymous.read(); got != "41" {
		t.Errorf("after unauthorized = %s", got)
	}

	client := dialSocket(t, addr, http.Header{"Authorization": {"Bearer " + token}})
	if got := client.connect(); got != `42["on_server_version",{"major":3,"minor":3,"patch":3,"prerelease":null}]` {
		t.Errorf("first event = %s", got)
	}

	cookie := dialSocket(t, addr, http.Header{"Cookie": {"immich_access_token=" + token}})
	if got := cookie.connect(); !strings.HasPrefix(got, `42["on_server_version"`) {
		t.Errorf("cookie = %s", got)
	}
	if got := client.read(); !strings.HasPrefix(got, `42["on_server_version"`) {
		t.Errorf("second connect of the user = %s", got)
	}
}

type emission struct {
	room  string
	event string
	args  []string
}

type socketRecorder struct {
	mu   sync.Mutex
	list []emission
}

func (h *apiHarness) recordSockets() *socketRecorder {
	h.t.Helper()

	rec := new(socketRecorder)
	gh.Websocket(h.rt).Observe(func(room string, event string, args []json.RawMessage) {
		e := emission{room: room, event: event}
		for _, a := range args {
			e.args = append(e.args, string(a))
		}
		rec.mu.Lock()
		rec.list = append(rec.list, e)
		rec.mu.Unlock()
	})

	return rec
}

func (rec *socketRecorder) take() []emission {
	rec.mu.Lock()
	defer rec.mu.Unlock()

	out := rec.list
	rec.list = nil
	return out
}

func (rec *socketRecorder) events() []string {
	var out []string
	for _, e := range rec.take() {
		out = append(out, e.event)
	}

	return out
}

func TestWebsocketEvents(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	ownerID := h.do(fiber.MethodGet, "/api/users/me", nil, owner...).str(t, "id")
	memberID := h.createUser(admin, "mira@example.com", "Mira", "mira-password")
	rec := h.recordSockets()

	exif := h.uploadID(admin, "korda-exif.jpg", fixture(t, testmedia.ExifJPEG), "2024-05-06T05:00:00.000Z", "2026-01-01T10:00:00.000Z")
	rec.take()
	h.runAllJobs()
	got := rec.take()
	if len(got) != 2 || got[0].event != "on_upload_success" || got[1].event != "AssetUploadReadyV2" || got[0].room != ownerID {
		t.Fatalf("upload events = %+v", got)
	}
	if want := `{"id":"<uuid>","createdAt":"<time>","ownerId":"<uuid>","libraryId":null,"type":"IMAGE","originalPath":"<path>","originalFileName":"korda-exif.jpg","originalMimeType":"image/jpeg","thumbhash":"<thumbhash>","fileCreatedAt":"2024-05-06T05:08:09.123Z","fileModifiedAt":"2026-01-01T10:00:00.000Z","localDateTime":"2024-05-06T07:08:09.123Z","updatedAt":"<time>","isFavorite":false,"isArchived":false,"isTrashed":false,"visibility":"timeline","duration":null,"exifInfo":{"make":"korda","model":"Fixture One","exifImageWidth":640,"exifImageHeight":480,"fileSizeInByte":15619,"orientation":"6","dateTimeOriginal":"2024-05-06T05:08:09.123+00:00","modifyDate":"2026-01-01T10:00:00+00:00","timeZone":"UTC+2","lensModel":"Fixture Lens 4.2mm f/2.8","fNumber":2.8,"focalLength":4.2,"iso":100,"exposureTime":"1/250","latitude":52.516272,"longitude":13.377722,"city":null,"state":null,"country":null,"description":"a fixture","projectionType":null,"rating":4},"livePhotoVideoId":null,"tags":[],"people":[],"checksum":"7lWy+xd67RvuCrGVWCyajV1LYuU=","isOffline":false,"hasMetadata":true,"duplicateId":null,"resized":true,"width":480,"height":640,"isEdited":false}`; normalizeAsset(got[0].args[0]) != want {
		t.Errorf("on_upload_success = %s", normalizeAsset(got[0].args[0]))
	}
	if want := `{"asset":{"id":"<uuid>","ownerId":"<uuid>","originalFileName":"korda-exif.jpg","thumbhash":"<thumbhash>","checksum":"7lWy+xd67RvuCrGVWCyajV1LYuU=","fileCreatedAt":"2024-05-06T05:08:09.123Z","fileModifiedAt":"2026-01-01T10:00:00.000Z","createdAt":"<time>","localDateTime":"2024-05-06T07:08:09.123Z","duration":null,"type":"IMAGE","deletedAt":null,"isFavorite":false,"visibility":"timeline","livePhotoVideoId":null,"stackId":null,"libraryId":null,"width":480,"height":640,"isEdited":false},"exif":{"assetId":"<uuid>","description":"a fixture","exifImageWidth":640,"exifImageHeight":480,"fileSizeInByte":15619,"orientation":"6","dateTimeOriginal":"2024-05-06T05:08:09.123Z","modifyDate":"2026-01-01T10:00:00.000Z","timeZone":"UTC+2","latitude":52.516272,"longitude":13.377722,"projectionType":null,"city":null,"state":null,"country":null,"make":"korda","model":"Fixture One","lensModel":"Fixture Lens 4.2mm f/2.8","fNumber":2.8,"focalLength":4.2,"iso":100,"exposureTime":"1/250","profileDescription":null,"rating":4,"fps":null}}`; normalizeAsset(got[1].args[0]) != want {
		t.Errorf("AssetUploadReadyV2 = %s", normalizeAsset(got[1].args[0]))
	}

	plain := h.uploadID(admin, "korda-plain.jpg", fixture(t, testmedia.PlainJPEG), "2020-02-03T04:05:06.000Z", "2020-02-04T04:05:06.000Z")
	h.runAllJobs()
	rec.take()

	h.do(fiber.MethodPut, "/api/assets/"+exif, `{"isFavorite":true}`, owner...).expect(t, 200, "")
	got = rec.take()
	if len(got) != 1 || got[0].event != "on_asset_update" || !strings.Contains(got[0].args[0], `"isFavorite":true`) ||
		!strings.Contains(got[0].args[0], `"exifInfo"`) || strings.Contains(got[0].args[0], `"tags"`) {
		t.Errorf("update = %+v", got)
	}
	h.do(fiber.MethodPut, "/api/assets", `{"ids":["`+exif+`","`+plain+`"],"isFavorite":false}`, owner...).expect(t, 204, "")
	got = rec.take()
	if len(got) != 2 || strings.Contains(got[0].args[0], `"exifInfo"`) || strings.Contains(got[1].args[0], `"tags"`) {
		t.Errorf("bulk update = %+v", got)
	}
	h.do(fiber.MethodPut, "/api/assets/"+plain, `{"description":"changed"}`, owner...).expect(t, 200, "")
	if events := rec.events(); len(events) != 0 {
		t.Errorf("events before the sidecar job = %v", events)
	}
	h.runAllJobs()
	got = rec.take()
	if len(got) != 1 || got[0].event != "on_asset_update" || !strings.Contains(got[0].args[0], `"tags":[]`) {
		t.Errorf("update after the sidecar job = %+v", got)
	}

	h.do(fiber.MethodDelete, "/api/assets", `{"ids":["`+plain+`"]}`, owner...).expect(t, 204, "")
	if got = rec.take(); len(got) != 1 || got[0].event != "on_asset_trash" || got[0].args[0] != `["`+plain+`"]` {
		t.Errorf("trash = %+v", got)
	}
	h.do(fiber.MethodPost, "/api/trash/restore/assets", `{"ids":["`+plain+`"]}`, owner...).expect(t, 200, "")
	if got = rec.take(); len(got) != 1 || got[0].event != "on_asset_restore" || got[0].args[0] != `["`+plain+`"]` {
		t.Errorf("restore = %+v", got)
	}

	stack := h.do(fiber.MethodPost, "/api/stacks", `{"assetIds":["`+exif+`","`+plain+`"]}`, owner...)
	if got = rec.take(); len(got) != 1 || got[0].event != "on_asset_stack_update" || len(got[0].args) != 0 {
		t.Errorf("stack = %+v", got)
	}
	h.do(fiber.MethodDelete, "/api/stacks/"+stack.str(t, "id"), nil, owner...).expect(t, 204, "")
	if events := rec.events(); len(events) != 1 || events[0] != "on_asset_stack_update" {
		t.Errorf("stack delete = %v", events)
	}

	album := h.do(fiber.MethodPost, "/api/albums", `{"albumName":"Holiday","albumUsers":[{"userId":"`+memberID+`","role":"editor"}]}`, owner...).str(t, "id")
	if events := rec.events(); len(events) != 0 {
		t.Errorf("album create = %v", events)
	}
	h.do(fiber.MethodPut, "/api/albums/"+album+"/assets", `{"ids":["`+exif+`"]}`, owner...).expect(t, 200, "")
	got = rec.take()
	rooms := []string{}
	for _, e := range got {
		if e.event != "on_album_update" || e.args[0] != `"`+album+`"` {
			t.Errorf("album update = %+v", e)
		}
		rooms = append(rooms, e.room)
	}
	slices.Sort(rooms)
	if want := []string{memberID, ownerID}; !slices.Equal(rooms, slices.Sorted(slices.Values(want))) {
		t.Errorf("album update rooms = %v, want %v", rooms, want)
	}
	h.runAllJobs()
	if got = rec.take(); len(got) != 2 || got[0].event != "on_notification" || got[1].event != "on_notification" ||
		got[0].room != memberID || got[1].room != memberID {
		t.Errorf("album notifications = %+v", got)
	}

	h.do(fiber.MethodDelete, "/api/assets", `{"ids":["`+plain+`"],"force":true}`, owner...).expect(t, 204, "")
	rec.take()
	h.runAllJobs()
	if got = rec.take(); len(got) != 1 || got[0].event != "on_asset_delete" || got[0].args[0] != `"`+plain+`"` {
		t.Errorf("delete = %+v", got)
	}

	mira := h.login("mira@example.com", "mira-password")
	sessionID := h.do(fiber.MethodGet, "/api/sessions", nil, bearer(mira)...).array(t)[0].(map[string]any)["id"].(string)
	h.do(fiber.MethodPost, "/api/auth/logout", nil, bearer(mira)...).expect(t, 200, "")
	time.Sleep(700 * time.Millisecond)
	if got = rec.take(); len(got) != 1 || got[0].event != "on_session_delete" || got[0].room != sessionID || got[0].args[0] != `"`+sessionID+`"` {
		t.Errorf("logout = %+v", got)
	}

	h.do(fiber.MethodDelete, "/api/admin/users/"+memberID, `{"force":true}`, owner...).expect(t, 200, "")
	h.runAllJobs()
	if got = rec.take(); len(got) != 1 || got[0].event != "on_user_delete" || got[0].room != "" || got[0].args[0] != `"`+memberID+`"` {
		t.Errorf("user delete = %+v", got)
	}
}
