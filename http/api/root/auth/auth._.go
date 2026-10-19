package auth

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
	r.Path = route.For("APIAuth").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Post("/login", r.Login).Name("login")
		base.Post("/admin-sign-up", r.SignUpAdmin).Name("signUpAdmin")
		base.Post("/validateToken", r.ValidateAccessToken).Name("validateAccessToken")
		base.Post("/change-password", r.ChangePassword).Name("changePassword")
		base.Post("/logout", r.Logout).Name("logout")
		base.Get("/status", r.GetAuthStatus).Name("getAuthStatus")
		base.Post("/pin-code", r.SetupPinCode).Name("setupPinCode")
		base.Put("/pin-code", r.ChangePinCode).Name("changePinCode")
		base.Delete("/pin-code", r.ResetPinCode).Name("resetPinCode")
		base.Post("/session/unlock", r.UnlockAuthSession).Name("unlockAuthSession")
		base.Post("/session/lock", r.LockAuthSession).Name("lockAuthSession")
	}, r.Path+".")

	return r, nil
}
