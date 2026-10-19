package admin

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/root/admin/users"
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
	r.Path = route.For("APIAdmin").Pathname()
	r.Env = route.NewEnv()

	var err error
	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		var usersRoute *users.Route
		if usersRoute, err = users.New(r.Runtime, base); err != nil {
			return
		}
		r.Routes = append(r.Routes, usersRoute)
	}, r.Path+".")
	if err != nil {
		return nil, err
	}

	return r, nil
}
