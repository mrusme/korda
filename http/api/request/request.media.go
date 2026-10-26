package request

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/errs"
)

const CachePrivate = "private, max-age=86400, no-transform, stale-while-revalidate=2592000, stale-if-error=2592000"

type fileBody struct {
	*io.SectionReader
	f *os.File
}

func (b fileBody) Close() error {
	return b.f.Close()
}

func encodeURIComponent(s string) string {
	const unreserved = "-_.!~*'()"

	var b strings.Builder
	for _, c := range []byte(s) {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', strings.IndexByte(unreserved, c) >= 0:
			b.WriteByte(c)
		default:
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}

	return b.String()
}

func ETag(size int64, modified time.Time) string {
	return `W/"` + strconv.FormatInt(size, 16) + `-` + strconv.FormatInt(modified.UnixMilli(), 16) + `"`
}

func (req *Request) fresh(etag string, modified time.Time) bool {
	c := req.c
	noneMatch := c.Get(fiber.HeaderIfNoneMatch)
	since := c.Get(fiber.HeaderIfModifiedSince)
	if noneMatch == "" && since == "" {
		return false
	}
	if strings.Contains(c.Get(fiber.HeaderCacheControl), "no-cache") {
		return false
	}

	if noneMatch != "" && noneMatch != "*" {
		matched := false
		for tag := range strings.SplitSeq(noneMatch, ",") {
			tag = strings.TrimPrefix(strings.TrimSpace(tag), "W/")
			if tag == strings.TrimPrefix(etag, "W/") {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	if since != "" {
		t, err := http.ParseTime(since)
		if err != nil || modified.Truncate(time.Second).After(t) {
			return false
		}
	}

	return true
}

func parseRange(header string, size int64) (start int64, end int64, ok bool, satisfiable bool) {
	spec, found := strings.CutPrefix(header, "bytes=")
	if !found || strings.Contains(spec, ",") {
		return 0, 0, false, true
	}

	first, last, found := strings.Cut(strings.TrimSpace(spec), "-")
	if !found {
		return 0, 0, false, true
	}

	switch {
	case first == "":
		n, err := strconv.ParseInt(last, 10, 64)
		if err != nil {
			return 0, 0, false, true
		}
		start, end = max(size-n, 0), size-1
	default:
		var err error
		if start, err = strconv.ParseInt(first, 10, 64); err != nil {
			return 0, 0, false, true
		}
		end = size - 1
		if last != "" {
			if end, err = strconv.ParseInt(last, 10, 64); err != nil {
				return 0, 0, false, true
			}
			end = min(end, size-1)
		}
	}

	if start > end || start >= size {
		return 0, 0, true, false
	}

	return start, end, true, true
}

func (req *Request) SendMedia(abs string, contentType string, fileName string) error {
	f, err := os.Open(abs)
	if err != nil {
		return errs.ErrNotFound
	}

	st, err := f.Stat()
	if err != nil || st.IsDir() {
		f.Close()
		return errs.ErrNotFound
	}

	size := st.Size()
	modified := st.ModTime()
	etag := ETag(size, modified)

	c := req.c
	c.Set(fiber.HeaderCacheControl, CachePrivate)
	c.Set(fiber.HeaderContentType, contentType)
	if fileName != "" {
		c.Set(fiber.HeaderContentDisposition, "inline; filename*=UTF-8''"+encodeURIComponent(fileName))
	}
	c.Set(fiber.HeaderAcceptRanges, "bytes")
	c.Set(fiber.HeaderLastModified, modified.UTC().Format(http.TimeFormat))
	c.Set(fiber.HeaderETag, etag)

	if req.fresh(etag, modified) {
		f.Close()
		return c.SendStatus(fiber.StatusNotModified)
	}

	start, end := int64(0), size-1
	status := fiber.StatusOK
	if header := c.Get(fiber.HeaderRange); header != "" && size > 0 {
		ifRange := c.Get(fiber.HeaderIfRange)
		if ifRange == "" || ifRange == etag || ifRange == modified.UTC().Format(http.TimeFormat) {
			s, e, ranged, satisfiable := parseRange(header, size)
			switch {
			case !satisfiable:
				f.Close()
				c.Set(fiber.HeaderContentRange, "bytes */"+strconv.FormatInt(size, 10))
				c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")
				return c.Status(fiber.StatusRequestedRangeNotSatisfiable).SendString("Range Not Satisfiable")
			case ranged:
				start, end = s, e
				status = fiber.StatusPartialContent
				c.Set(fiber.HeaderContentRange, fmt.Sprintf("bytes %d-%d/%d", start, end, size))
			}
		}
	}

	length := end - start + 1
	if size == 0 {
		length = 0
	}
	c.Status(status)

	return c.SendStream(fileBody{SectionReader: io.NewSectionReader(f, start, length), f: f}, int(length))
}

type queryParam struct {
	key   string
	value string
}

func parseQuery(raw string) []queryParam {
	var out []queryParam
	for part := range strings.SplitSeq(raw, "&") {
		if part == "" {
			continue
		}
		key, value, _ := strings.Cut(part, "=")
		k, err := url.QueryUnescape(key)
		if err != nil {
			k = key
		}
		v, err := url.QueryUnescape(value)
		if err != nil {
			v = value
		}
		out = append(out, queryParam{k, v})
	}

	return out
}

func formEncode(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		switch {
		case c == ' ':
			b.WriteByte('+')
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '*', c == '-', c == '.', c == '_':
			b.WriteByte(c)
		default:
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}

	return b.String()
}

func encodeQuery(params []queryParam) string {
	parts := make([]string, len(params))
	for i, p := range params {
		parts[i] = formEncode(p.key) + "=" + formEncode(p.value)
	}

	return strings.Join(parts, "&")
}

func (req *Request) rawQuery() string {
	return string(req.c.RequestCtx().URI().QueryString())
}

func (req *Request) QueryWithout(key string) string {
	var out []queryParam
	for _, p := range parseQuery(req.rawQuery()) {
		if p.key != key {
			out = append(out, p)
		}
	}

	return encodeQuery(out)
}

func (req *Request) QueryWith(key string, value string) string {
	params := parseQuery(req.rawQuery())
	out := make([]queryParam, 0, len(params)+1)
	set := false
	for _, p := range params {
		if p.key != key {
			out = append(out, p)
			continue
		}
		if !set {
			out = append(out, queryParam{key, value})
			set = true
		}
	}
	if !set {
		out = append(out, queryParam{key, value})
	}

	return encodeQuery(out)
}

func (req *Request) Redirect(location string) error {
	return req.RedirectStatus(fiber.StatusFound, location)
}

func (req *Request) RedirectStatus(status int, location string) error {
	c := req.c
	c.Set(fiber.HeaderLocation, location)
	c.Status(status)
	text := http.StatusText(status)

	if html := c.Accepts(fiber.MIMETextPlain, fiber.MIMETextHTML); html == fiber.MIMETextHTML {
		c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")
		return c.SendString("<p>" + text + ". Redirecting to " + htmlEscape(location) + "</p>")
	}

	c.Set(fiber.HeaderContentType, "text/plain; charset=utf-8")
	return c.SendString(text + ". Redirecting to " + location)
}

func (req *Request) Cookie(name string) string {
	return req.c.Cookies(name)
}

func htmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;").Replace(s)
}

var ErrNotAFile = errors.New("not a regular file")

func (req *Request) SendArchive(name *string, write func(w io.Writer) error) error {
	c := req.c
	c.Status(fiber.StatusOK)
	c.Set(fiber.HeaderContentType, fiber.MIMEOctetStream)
	if name != nil {
		disposition := ""
		if *name != "" {
			disposition = "attachment; filename*=UTF-8''" + encodeURIComponent(*name) + ".zip"
		}
		c.Set(fiber.HeaderContentDisposition, disposition)
	}

	return c.SendStreamWriter(func(w *bufio.Writer) {
		if err := write(w); err != nil {
			req.rt.Warn("download", "archive", "error", err)
		}
		_ = w.Flush()
	})
}
