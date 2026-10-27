package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/gofiber/fiber/v3"
)

type encodedResponse struct {
	status   int
	encoding string
	vary     string
	length   bool
	chunked  bool
	body     []byte
}

func decodeBody(t *testing.T, encoding string, raw []byte) []byte {
	t.Helper()

	var r io.Reader
	var err error
	switch encoding {
	case "":
		return raw
	case "gzip":
		r, err = gzip.NewReader(bytes.NewReader(raw))
	case "deflate":
		r, err = zlib.NewReader(bytes.NewReader(raw))
	case "br":
		r = brotli.NewReader(bytes.NewReader(raw))
	default:
		t.Fatalf("unknown Content-Encoding %q", encoding)
	}
	if err != nil {
		t.Fatalf("%s: %v", encoding, err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("%s: %v", encoding, err)
	}

	return out
}

func fetchEncoded(t *testing.T, client *http.Client, base string, method string, path string, body string, headers ...string) encodedResponse {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, base+path, reader)
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	encoding := resp.Header.Get(fiber.HeaderContentEncoding)
	return encodedResponse{
		status:   resp.StatusCode,
		encoding: encoding,
		vary:     resp.Header.Get(fiber.HeaderVary),
		length:   resp.ContentLength >= 0,
		chunked:  slices.Contains(resp.TransferEncoding, "chunked"),
		body:     decodeBody(t, encoding, raw),
	}
}

func rawChunks(t *testing.T, addr string, path string, token string, body string) [][]byte {
	t.Helper()

	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err = conn.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err = fmt.Fprintf(conn, "POST %s HTTP/1.1\r\nHost: %s\r\nAuthorization: Bearer %s\r\nContent-Type: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s",
		path, addr, token, len(body), body); err != nil {
		t.Fatal(err)
	}

	r := bufio.NewReader(conn)
	chunked := false
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if strings.EqualFold(strings.TrimSpace(line), "Transfer-Encoding: chunked") {
			chunked = true
		}
		if line == "\r\n" {
			break
		}
	}
	if !chunked {
		t.Fatalf("%s: response is not chunked", addr)
	}

	var chunks [][]byte
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		size, err := strconv.ParseInt(strings.TrimSpace(line), 16, 64)
		if err != nil {
			t.Fatalf("chunk size %q: %v", line, err)
		}
		if size == 0 {
			return chunks
		}
		chunk := make([]byte, size)
		if _, err = io.ReadFull(r, chunk); err != nil {
			t.Fatal(err)
		}
		if _, err = r.Discard(2); err != nil {
			t.Fatal(err)
		}
		chunks = append(chunks, chunk)
	}
}

func TestConformanceCompression(t *testing.T) {
	base := strings.TrimSuffix(os.Getenv(envReferenceURL), "/")
	email := os.Getenv(envReferenceEmail)
	password := os.Getenv(envReferencePassword)
	if base == "" || email == "" || password == "" {
		t.Skipf("set %s, %s and %s to compare korda with an Immich", envReferenceURL, envReferenceEmail, envReferencePassword)
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
	immichUser := created.str(t, "id")
	immichToken := immich.send(fiber.MethodPost, "/api/auth/login", fmt.Sprintf(`{"email":%q,"password":%q}`, userEmail, userPassword)).str(t, "accessToken")

	h := newHarness(t)
	kordaAdmin := h.admin()
	kordaUser := h.createUser(kordaAdmin, userEmail, "Conformance", userPassword)
	kordaToken := h.login(userEmail, userPassword)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = h.app.Listener(ln, fiber.ListenConfig{DisableStartupMessage: true}) }()
	t.Cleanup(func() { _ = h.app.Shutdown() })
	kordaBase := "http://" + ln.Addr().String()

	reference, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}

	client := &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{DisableCompression: true}}
	tokens := map[string][2]string{"admin": {adminToken, kordaAdmin}, "user": {immichToken, kordaToken}}
	users := [2]string{immichUser, kordaUser}

	steps := []struct {
		name   string
		method string
		path   string
		body   string
		auth   string
		accept string
		stream bool
	}{
		{name: "small plain", method: fiber.MethodGet, path: "/api/server/ping"},
		{name: "small gzip", method: fiber.MethodGet, path: "/api/server/ping", accept: "gzip"},
		{name: "head gzip", method: fiber.MethodHead, path: "/api/server/ping", accept: "gzip"},
		{name: "large plain", method: fiber.MethodGet, path: "/api/system-config/storage-template-options", auth: "admin"},
		{name: "large gzip", method: fiber.MethodGet, path: "/api/system-config/storage-template-options", auth: "admin", accept: "gzip"},
		{name: "large br", method: fiber.MethodGet, path: "/api/system-config/storage-template-options", auth: "admin", accept: "br"},
		{name: "large deflate", method: fiber.MethodGet, path: "/api/system-config/storage-template-options", auth: "admin", accept: "deflate"},
		{name: "large ios", method: fiber.MethodGet, path: "/api/system-config/storage-template-options", auth: "admin", accept: "gzip, deflate, br"},
		{name: "large any", method: fiber.MethodGet, path: "/api/system-config/storage-template-options", auth: "admin", accept: "*"},
		{name: "large quality", method: fiber.MethodGet, path: "/api/system-config/storage-template-options", auth: "admin", accept: "br;q=0.5, gzip"},
		{name: "large no identity", method: fiber.MethodGet, path: "/api/system-config/storage-template-options", auth: "admin", accept: "identity;q=0"},
		{name: "large unknown", method: fiber.MethodGet, path: "/api/system-config/storage-template-options", auth: "admin", accept: "compress"},
		{name: "error gzip", method: fiber.MethodGet, path: "/api/assets/00000000-0000-4000-8000-000000000000", auth: "user", accept: "gzip"},
		{name: "no content gzip", method: fiber.MethodDelete, path: "/api/sync/ack", body: `{}`, auth: "user", accept: "gzip"},
		{name: "well-known gzip", method: fiber.MethodGet, path: "/.well-known/immich", accept: "gzip"},
		{name: "custom css gzip", method: fiber.MethodGet, path: "/custom.css", accept: "gzip"},
		{name: "stream plain", method: fiber.MethodPost, path: "/api/sync/stream", body: syncBody(), auth: "user", stream: true},
		{name: "stream gzip", method: fiber.MethodPost, path: "/api/sync/stream", body: syncBody(), auth: "user", accept: "gzip", stream: true},
		{name: "stream br", method: fiber.MethodPost, path: "/api/sync/stream", body: syncBody(), auth: "user", accept: "br", stream: true},
		{name: "stream deflate", method: fiber.MethodPost, path: "/api/sync/stream", body: syncBody(), auth: "user", accept: "deflate", stream: true},
	}
	for _, st := range steps {
		t.Run(st.name, func(t *testing.T) {
			var got [2]encodedResponse
			for i, side := range []string{base, kordaBase} {
				var headers []string
				if st.auth != "" {
					headers = append(headers, bearer(tokens[st.auth][i])...)
				}
				if st.accept != "" {
					headers = append(headers, fiber.HeaderAcceptEncoding, st.accept)
				}
				got[i] = fetchEncoded(t, client, side, st.method, st.path, st.body, headers...)
			}
			want, have := got[0], got[1]

			if have.status != want.status || have.encoding != want.encoding || have.vary != want.vary ||
				have.length != want.length || have.chunked != want.chunked {
				t.Errorf("korda %d encoding %q vary %q length %t chunked %t, Immich %d encoding %q vary %q length %t chunked %t",
					have.status, have.encoding, have.vary, have.length, have.chunked,
					want.status, want.encoding, want.vary, want.length, want.chunked)
			}
			a, b := string(have.body), string(want.body)
			if st.stream {
				a, b = normalizeStream(t, have.body, users[1]), normalizeStream(t, want.body, users[0])
			}
			if a != b {
				t.Errorf("decoded body differs\nkorda  %s\nImmich %s", a, b)
			}
		})
	}

	t.Run("stream chunks", func(t *testing.T) {
		for i, addr := range []string{reference.Host, ln.Addr().String()} {
			chunks := rawChunks(t, addr, "/api/sync/stream", [2]string{immichToken, kordaToken}[i], syncBody())
			for _, chunk := range chunks {
				if bytes.Count(chunk, []byte("\n")) != 1 || !bytes.HasSuffix(chunk, []byte("\n")) {
					t.Errorf("%s: chunk of %d bytes holds %d lines", [2]string{"Immich", "korda"}[i], len(chunk), bytes.Count(chunk, []byte("\n")))
				}
			}
			if len(chunks) == 0 {
				t.Errorf("%s: no chunks", [2]string{"Immich", "korda"}[i])
			}
		}
	})
}
