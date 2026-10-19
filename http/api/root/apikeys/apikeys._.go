package apikeys

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
	r.Path = route.For("APIAPIKeys").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Post("", r.CreateApiKey).Name("createApiKey")
		base.Get("", r.GetApiKeys).Name("getApiKeys")
		base.Get("/me", r.GetMyApiKey).Name("getMyApiKey")
		base.Get("/:id", r.GetApiKey).Name("getApiKey")
		base.Put("/:id", r.UpdateApiKey).Name("updateApiKey")
		base.Patch("/:id", r.UpdateApiKeyV3).Name("updateApiKeyV3")
		base.Post("/:id/rotate", r.RotateApiKey).Name("rotateApiKey")
		base.Delete("/:id", r.DeleteApiKey).Name("deleteApiKey")
	}, r.Path+".")

	return r, nil
}
