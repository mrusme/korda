package wellknown

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/problem"
	"xn--gckvb8fzb.com/glides/http/route"
	"xn--gckvb8fzb.com/glides/runtime"
)

type Route struct {
	route.RouteController
}

type endpoint struct {
	Endpoint string `json:"endpoint"`
}

type document struct {
	API endpoint `json:"api"`
}

func New(rt *runtime.Runtime, router fiber.Router) (*Route, error) {
	r := new(Route)

	r.Runtime = rt
	r.Router = router
	r.Path = route.For("WellKnown").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Get("/"+route.For("WellKnownImmich").Pathname(), r.Immich).Name("immich")
	}, r.Path+".")

	return r, nil
}

func (r *Route) Immich(c fiber.Ctx) error {
	body, err := problem.Encode(document{API: endpoint{Endpoint: "/" + route.For("API").AsURL()}})
	if err != nil {
		return err
	}

	c.Set(fiber.HeaderContentType, problem.ContentType)
	return c.Send(body)
}
