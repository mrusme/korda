package admin

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/root/admin/notifications"
	"tty.fail/mrus/korda/http/api/root/admin/users"
	"tty.fail/mrus/korda/http/api/root/maintenance"
	"tty.fail/mrus/korda/http/api/root/oauth"
	"tty.fail/mrus/korda/http/api/root/systemconfig"
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

		var authRoute *oauth.Route
		if authRoute, err = oauth.NewAdmin(r.Runtime, base); err != nil {
			return
		}
		r.Routes = append(r.Routes, authRoute)

		var configRoute *systemconfig.Route
		if configRoute, err = systemconfig.NewAdmin(r.Runtime, base); err != nil {
			return
		}
		r.Routes = append(r.Routes, configRoute)

		var notificationsRoute *notifications.Route
		if notificationsRoute, err = notifications.New(r.Runtime, base); err != nil {
			return
		}
		r.Routes = append(r.Routes, notificationsRoute)

		for _, mount := range []func(*runtime.Runtime, fiber.Router) (*maintenance.Route, error){
			maintenance.NewMaintenance, maintenance.NewIntegrity, maintenance.NewBackups,
		} {
			var maintenanceRoute *maintenance.Route
			if maintenanceRoute, err = mount(r.Runtime, base); err != nil {
				return
			}
			r.Routes = append(r.Routes, maintenanceRoute)
		}
	}, r.Path+".")
	if err != nil {
		return nil, err
	}

	return r, nil
}
