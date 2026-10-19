package me

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
	r.Path = route.For("APIUsersMe").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Get("", r.GetMyUser).Name("getMyUser")
		base.Put("", r.UpdateMyUser).Name("updateMyUser")
		base.Patch("", r.UpdateMyUserV3).Name("updateMyUserV3")
		base.Get("/calendar-heatmap", r.GetMyCalendarHeatmap).Name("getMyCalendarHeatmap")
		base.Get("/preferences", r.GetMyPreferences).Name("getMyPreferences")
		base.Put("/preferences", r.UpdateMyPreferences).Name("updateMyPreferences")
		base.Patch("/preferences", r.UpdateMyPreferencesV3).Name("updateMyPreferencesV3")
		base.Get("/license", r.GetUserLicense).Name("getUserLicense")
		base.Put("/license", r.SetUserLicense).Name("setUserLicense")
		base.Delete("/license", r.DeleteUserLicense).Name("deleteUserLicense")
		base.Get("/onboarding", r.GetUserOnboarding).Name("getUserOnboarding")
		base.Put("/onboarding", r.SetUserOnboarding).Name("setUserOnboarding")
		base.Delete("/onboarding", r.DeleteUserOnboarding).Name("deleteUserOnboarding")
	}, r.Path+".")

	return r, nil
}
