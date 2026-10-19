package sessions

import (
	"github.com/gofiber/fiber/v3"
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
	r.Path = route.For("APISessions").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Post("", r.CreateSession).Name("createSession")
		base.Get("", r.GetSessions).Name("getSessions")
		base.Delete("", r.DeleteAllSessions).Name("deleteAllSessions")
		base.Put("/:id", r.UpdateSession).Name("updateSession")
		base.Patch("/:id", r.UpdateSessionV3).Name("updateSessionV3")
		base.Delete("/:id", r.DeleteSession).Name("deleteSession")
		base.Post("/:id/lock", r.LockSession).Name("lockSession")
	}, r.Path+".")

	return r, nil
}
