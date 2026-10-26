package share

import (
	"bytes"
	"errors"
	"html/template"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/errs"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/helpers/deadline"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logicsharedlinks "tty.fail/mrus/korda/logic/root/sharedlinks"
	"tty.fail/mrus/korda/views"
	"xn--gckvb8fzb.com/glides/http/route"
	"xn--gckvb8fzb.com/glides/runtime"
)

const (
	formLimit     = 64 << 10
	titleNotFound = "Link not found"
	titlePassword = "Password required"
	wrongPassword = "Invalid password"
)

type Route struct {
	route.RouteController
	page       *template.Template
	stylesheet string
}

type credential struct {
	creds logicauth.Credentials
	query url.Values
}

func New(rt *runtime.Runtime, router fiber.Router, limiter fiber.Handler, timeout time.Duration) (*Route, error) {
	page, err := template.ParseFS(views.FS, "share/share.html")
	if err != nil {
		return nil, err
	}

	r := new(Route)

	r.Runtime = rt
	r.Router = router
	r.Path = route.For("Share").Pathname()
	r.Env = route.NewEnv()
	r.page = page
	r.stylesheet = "/" + route.For("Static").AsURL() + "/share/share.css"

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Get("/:key", deadline.New(timeout), r.ByKey).Name("byKey")
		base.Post("/:key", limiter, deadline.New(timeout), r.LoginByKey).Name("loginByKey")
	}, r.Path+".")

	slug := route.For("ShareSlug").Pathname()
	r.Router.Route("/"+slug, func(base fiber.Router) {
		base.Get("/:slug", deadline.New(timeout), r.BySlug).Name("bySlug")
		base.Post("/:slug", limiter, deadline.New(timeout), r.LoginBySlug).Name("loginBySlug")
	}, slug+".")

	return r, nil
}

func byKey(c fiber.Ctx) credential {
	key := c.Params("key")

	return credential{creds: logicauth.Credentials{ShareKey: key}, query: url.Values{"key": {key}}}
}

func bySlug(c fiber.Ctx) credential {
	slug := c.Params("slug")
	if unescaped, err := url.PathUnescape(slug); err == nil {
		slug = unescaped
	}

	return credential{creds: logicauth.Credentials{ShareSlug: slug}, query: url.Values{"slug": {slug}}}
}

func invalid(err error) bool {
	return errors.Is(err, errs.ErrInvalidShareKey) || errors.Is(err, errs.ErrInvalidShareSlug)
}

func (r *Route) ByKey(c fiber.Ctx) error {
	return r.show(c, byKey(c))
}

func (r *Route) BySlug(c fiber.Ctx) error {
	return r.show(c, bySlug(c))
}

func (r *Route) LoginByKey(c fiber.Ctx) error {
	return r.login(c, byKey(c))
}

func (r *Route) LoginBySlug(c fiber.Ctx) error {
	return r.login(c, bySlug(c))
}

func (r *Route) show(c fiber.Ctx, cred credential) error {
	cred.creds.SharedLinkTokens = request.SharedLinkTokens(c)
	a, err := logicauth.Authenticate(c.Context(), r.Runtime, cred.creds, logicauth.Requirement{SharedLink: true}, c.Path())
	switch {
	case errors.Is(err, errs.ErrSharedLinkPasswordMissing):
		return r.render(c, fiber.StatusUnauthorized, &page{Title: titlePassword, Password: true})
	case invalid(err):
		return r.render(c, fiber.StatusNotFound, &page{Title: titleNotFound, NotFound: true})
	case err != nil:
		return err
	}

	p, err := gallery(c.Context(), r.Runtime, a, cred.query, origin(r.Runtime, c))
	if err != nil {
		return err
	}

	return r.render(c, fiber.StatusOK, p)
}

func (r *Route) login(c fiber.Ctx, cred credential) error {
	requirement := logicauth.Requirement{SharedLink: true, NoSharedLinkPassword: true}
	a, err := logicauth.Authenticate(c.Context(), r.Runtime, cred.creds, requirement, c.Path())
	if invalid(err) {
		return r.render(c, fiber.StatusNotFound, &page{Title: titleNotFound, NotFound: true})
	}
	if err != nil {
		return err
	}

	body, err := request.New(r.Runtime, c, "sharePageLogin").Read(formLimit)
	if errors.Is(err, errs.ErrPayloadTooLarge) {
		return fiber.ErrRequestEntityTooLarge
	}
	if err != nil {
		return err
	}
	form, _ := url.ParseQuery(string(body))

	_, token, err := logicsharedlinks.Login(c.Context(), r.Runtime, a, &logicsharedlinks.LoginInput{Password: form.Get("password")})
	switch {
	case errors.Is(err, errs.ErrSharedLinkPassword):
		return r.render(c, fiber.StatusUnauthorized, &page{Title: titlePassword, Password: true, Error: wrongPassword})
	case errors.Is(err, errs.ErrSharedLinkNoPassword):
	case err != nil:
		return err
	default:
		request.AddSharedLinkToken(c, token)
	}

	return c.Redirect().Status(fiber.StatusSeeOther).To(c.OriginalURL())
}

func (r *Route) render(c fiber.Ctx, status int, p *page) error {
	p.Stylesheet = r.stylesheet

	var buf bytes.Buffer
	if err := r.page.Execute(&buf, p); err != nil {
		return err
	}

	c.Status(status)
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.Send(buf.Bytes())
}
