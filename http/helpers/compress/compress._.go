package compress

import (
	"bytes"
	"regexp"
	"slices"
	"strings"

	"github.com/gofiber/fiber/v3"
)

const (
	threshold   = 1024
	defaultType = "text/plain; charset=utf-8"
	varyField   = "Accept-Encoding"
)

var noTransform = regexp.MustCompile(`(?i)(?:^|,)\s*?no-transform\s*?(?:,|$)`)

func New() fiber.Handler {
	return func(c fiber.Ctx) error {
		if err := c.Next(); err != nil {
			if err = c.App().ErrorHandler(c, err); err != nil {
				return err
			}
		}
		apply(c)

		return nil
	}
}

func Vary(header string, field string) string {
	if header == "" {
		return field
	}

	values := []string{}
	for v := range strings.SplitSeq(strings.ToLower(header), ",") {
		values = append(values, strings.Trim(v, " "))
	}
	switch {
	case slices.Contains(values, "*"):
		return "*"
	case slices.Contains(values, strings.ToLower(field)):
		return header
	}

	return header + ", " + field
}

func apply(c fiber.Ctx) {
	resp := c.Response()
	stream := resp.IsBodyStream()
	size := resp.Header.ContentLength()
	if !stream {
		size = len(resp.Body())
	}

	contentType := string(resp.Header.ContentType())
	if !stream && size == 0 && contentType == defaultType {
		return
	}
	if !Compressible(contentType) || noTransform.Match(resp.Header.Peek(fiber.HeaderCacheControl)) {
		return
	}
	resp.Header.Set(fiber.HeaderVary, Vary(string(resp.Header.Peek(fiber.HeaderVary)), varyField))

	if size >= 0 && size < threshold {
		return
	}
	if encoding := string(resp.Header.Peek(fiber.HeaderContentEncoding)); encoding != "" && encoding != identityMethod {
		return
	}
	if c.Method() == fiber.MethodHead {
		return
	}
	encoding := Negotiate(string(c.Request().Header.Peek(fiber.HeaderAcceptEncoding)))
	if encoding == "" || encoding == identityMethod {
		return
	}

	body, ok := resp.BodyStream().(*Body)
	switch {
	case stream && !ok:
		return
	case !stream:
		body = Bytes(bytes.Clone(resp.Body()))
		resp.SetBodyStream(body, -1)
	}
	body.SetEncoding(encoding)
	resp.Header.Set(fiber.HeaderContentEncoding, encoding)
	resp.Header.SetContentLength(-1)
}
