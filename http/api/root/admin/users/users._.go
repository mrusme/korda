package users

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
	r.Path = route.For("APIAdminUsers").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Get("", r.SearchUsersAdmin).Name("searchUsersAdmin")
		base.Post("", r.CreateUserAdmin).Name("createUserAdmin")
		base.Get("/:id", r.GetUserAdmin).Name("getUserAdmin")
		base.Put("/:id", r.UpdateUserAdmin).Name("updateUserAdmin")
		base.Patch("/:id", r.UpdateUserAdminV3).Name("updateUserAdminV3")
		base.Delete("/:id", r.DeleteUserAdmin).Name("deleteUserAdmin")
		base.Get("/:id/calendar-heatmap", r.GetUserCalendarHeatmapAdmin).Name("getUserCalendarHeatmapAdmin")
		base.Get("/:id/sessions", r.GetUserSessionsAdmin).Name("getUserSessionsAdmin")
		base.Get("/:id/statistics", r.GetUserStatisticsAdmin).Name("getUserStatisticsAdmin")
		base.Get("/:id/preferences", r.GetUserPreferencesAdmin).Name("getUserPreferencesAdmin")
		base.Put("/:id/preferences", r.UpdateUserPreferencesAdmin).Name("updateUserPreferencesAdmin")
		base.Patch("/:id/preferences", r.UpdateUserPreferencesAdminV3).Name("updateUserPreferencesAdminV3")
		base.Post("/:id/restore", r.RestoreUserAdmin).Name("restoreUserAdmin")
	}, r.Path+".")

	return r, nil
}
