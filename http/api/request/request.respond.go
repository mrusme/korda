package request

import (
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/problem"
)

const (
	cookieMaxAge     = 400 * 24 * time.Hour
	sharedLinkMaxAge = 24 * time.Hour
)

func (req *Request) Respond(status int, v any) error {
	body, err := problem.Encode(v)
	if err != nil {
		return err
	}

	req.c.Status(status)
	req.c.Set(fiber.HeaderContentType, ContentType)
	return req.c.Send(body)
}

func (req *Request) RespondJSON(status int, body []byte) error {
	req.c.Status(status)
	req.c.Set(fiber.HeaderContentType, ContentType)
	return req.c.Send(body)
}

func (req *Request) Empty(status int) error {
	req.c.Status(status)
	req.c.Response().ResetBody()
	return nil
}

func (req *Request) NoContent() error {
	return req.c.SendStatus(fiber.StatusNoContent)
}

func (req *Request) SetAuthCookies(token string) {
	at := time.Now()
	for _, cookie := range []struct {
		name     string
		value    string
		httpOnly bool
	}{
		{CookieAccessToken, token, true},
		{CookieAuthType, "password", true},
		{CookieIsAuthenticated, "true", false},
	} {
		req.c.Cookie(&fiber.Cookie{
			Name:     cookie.name,
			Value:    cookie.value,
			Path:     "/",
			MaxAge:   int(cookieMaxAge / time.Second),
			Expires:  at.Add(cookieMaxAge),
			HTTPOnly: cookie.httpOnly,
			Secure:   req.c.Secure(),
			SameSite: fiber.CookieSameSiteLaxMode,
		})
	}
}

func (req *Request) ClearAuthCookies() {
	for _, name := range []string{CookieAccessToken, CookieAuthType, CookieIsAuthenticated} {
		req.c.Cookie(&fiber.Cookie{
			Name:    name,
			Value:   "",
			Path:    "/",
			Expires: time.Unix(0, 0),
		})
	}
}

func (req *Request) SharedLinkTokens() []string {
	return SharedLinkTokens(req.c)
}

func (req *Request) AddSharedLinkToken(token string) {
	AddSharedLinkToken(req.c, token)
}

func SharedLinkTokens(c fiber.Ctx) []string {
	raw := c.Cookies(CookieSharedLinkToken)
	if raw == "" {
		return []string{}
	}
	if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		raw = raw[1 : len(raw)-1]
	}
	if decoded, err := url.PathUnescape(raw); err == nil {
		raw = decoded
	}

	return strings.Split(raw, ",")
}

func AddSharedLinkToken(c fiber.Ctx, token string) {
	tokens := SharedLinkTokens(c)
	if !slices.Contains(tokens, token) {
		tokens = append(tokens, token)
	}

	c.Cookie(&fiber.Cookie{
		Name:     CookieSharedLinkToken,
		Value:    encodeURIComponent(strings.Join(tokens, ",")),
		Path:     "/",
		MaxAge:   int(sharedLinkMaxAge / time.Second),
		Expires:  time.Now().Add(sharedLinkMaxAge),
		HTTPOnly: true,
		Secure:   c.Secure(),
		SameSite: fiber.CookieSameSiteLaxMode,
	})
}
