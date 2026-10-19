package root

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/root/admin"
	"tty.fail/mrus/korda/http/api/root/apikeys"
	"tty.fail/mrus/korda/http/api/root/auth"
	"tty.fail/mrus/korda/http/api/root/server"
	"tty.fail/mrus/korda/http/api/root/sessions"
	"tty.fail/mrus/korda/http/api/root/users"
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
	r.Path = route.For("API").AsURL()
	r.Env = route.NewEnv()

	serverRoute, err := server.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, serverRoute)

	authRoute, err := auth.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, authRoute)

	sessionsRoute, err := sessions.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, sessionsRoute)

	usersRoute, err := users.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, usersRoute)

	adminRoute, err := admin.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, adminRoute)

	apikeysRoute, err := apikeys.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, apikeysRoute)

	return r, nil
}
