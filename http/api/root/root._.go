package root

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/root/activities"
	"tty.fail/mrus/korda/http/api/root/admin"
	"tty.fail/mrus/korda/http/api/root/albums"
	"tty.fail/mrus/korda/http/api/root/apikeys"
	"tty.fail/mrus/korda/http/api/root/assets"
	"tty.fail/mrus/korda/http/api/root/auth"
	"tty.fail/mrus/korda/http/api/root/memories"
	"tty.fail/mrus/korda/http/api/root/partners"
	"tty.fail/mrus/korda/http/api/root/search"
	"tty.fail/mrus/korda/http/api/root/server"
	"tty.fail/mrus/korda/http/api/root/sessions"
	"tty.fail/mrus/korda/http/api/root/sharedlinks"
	"tty.fail/mrus/korda/http/api/root/stacks"
	"tty.fail/mrus/korda/http/api/root/sync"
	"tty.fail/mrus/korda/http/api/root/tags"
	"tty.fail/mrus/korda/http/api/root/trash"
	"tty.fail/mrus/korda/http/api/root/users"
	"tty.fail/mrus/korda/http/api/root/view"
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

	sharedLinksRoute, err := sharedlinks.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, sharedLinksRoute)

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

	assetsRoute, err := assets.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, assetsRoute)

	activitiesRoute, err := activities.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, activitiesRoute)

	albumsRoute, err := albums.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, albumsRoute)

	memoriesRoute, err := memories.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, memoriesRoute)

	partnersRoute, err := partners.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, partnersRoute)

	stacksRoute, err := stacks.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, stacksRoute)

	tagsRoute, err := tags.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, tagsRoute)

	searchRoute, err := search.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, searchRoute)

	viewRoute, err := view.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, viewRoute)

	trashRoute, err := trash.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, trashRoute)

	syncRoute, err := sync.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, syncRoute)

	return r, nil
}
