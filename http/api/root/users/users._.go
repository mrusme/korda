package users

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/root/users/me"
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
	r.Path = route.For("APIUsers").Pathname()
	r.Env = route.NewEnv()

	var err error
	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Get("", r.SearchUsers).Name("searchUsers")

		var meRoute *me.Route
		if meRoute, err = me.New(r.Runtime, base); err != nil {
			return
		}
		r.Routes = append(r.Routes, meRoute)

		base.Post("/profile-image", r.CreateProfileImage).Name("createProfileImage")
		base.Delete("/profile-image", r.DeleteProfileImage).Name("deleteProfileImage")
		base.Get("/:id", r.GetUser).Name("getUser")
		base.Get("/:id/profile-image", r.GetProfileImage).Name("getProfileImage")
	}, r.Path+".")
	if err != nil {
		return nil, err
	}

	return r, nil
}
