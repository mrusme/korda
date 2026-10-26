package theme

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/logic/root/systemconfig"
	"xn--gckvb8fzb.com/glides/http/route"
	"xn--gckvb8fzb.com/glides/runtime"
)

type Route struct {
	route.RouteController
}

func New(rt *runtime.Runtime, router fiber.Router) (*Route, error) {
	r := new(Route)

	r.Runtime = rt
	r.Router = router
	r.Path = route.For("CustomCSS").Pathname()
	r.Env = route.NewEnv()

	r.Router.Get("/"+r.Path, r.CustomCSS).Name("getCustomCss")

	return r, nil
}

func (r *Route) CustomCSS(c fiber.Ctx) error {
	cfg, err := systemconfig.Effective(r.Runtime)
	if err != nil {
		return err
	}

	c.Set(fiber.HeaderContentType, "text/css; charset=utf-8")
	return c.SendString(cfg.Theme.CustomCSS)
}
