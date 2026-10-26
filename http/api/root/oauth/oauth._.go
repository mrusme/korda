package oauth

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/problem"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logicoauth "tty.fail/mrus/korda/logic/root/oauth"
	"tty.fail/mrus/korda/models/permission"
	"xn--gckvb8fzb.com/glides/http/route"
	"xn--gckvb8fzb.com/glides/runtime"
)

const (
	cookieState    = "immich_oauth_state"
	cookieVerifier = "immich_oauth_code_verifier"
)

type Route struct {
	route.RouteController
}

func New(rt *runtime.Runtime, router fiber.Router) (*Route, error) {
	r := new(Route)

	r.Runtime = rt
	r.Router = router
	r.Path = route.For("APIOAuth").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Get("/mobile-redirect", r.RedirectOAuthToMobile).Name("redirectOAuthToMobile")
		base.Post("/authorize", r.StartOAuth).Name("startOAuth")
		base.Post("/callback", r.FinishOAuth).Name("finishOAuth")
		base.Post("/link", r.LinkOAuthAccount).Name("linkOAuthAccount")
		base.Post("/unlink", r.UnlinkOAuthAccount).Name("unlinkOAuthAccount")
		base.Post("/backchannel-logout", r.LogoutOAuth).Name("logoutOAuth")
	}, r.Path+".")

	return r, nil
}

func NewAdmin(rt *runtime.Runtime, router fiber.Router) (*Route, error) {
	r := new(Route)

	r.Runtime = rt
	r.Router = router
	r.Path = route.For("APIAdminAuth").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Post("/unlink-all", r.UnlinkAllOAuthAccountsAdmin).Name("unlinkAllOAuthAccountsAdmin")
	}, r.Path+".")

	return r, nil
}

func (r *Route) RedirectOAuthToMobile(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "redirectOAuthToMobile")
	return req.RedirectStatus(fiber.StatusTemporaryRedirect, logicoauth.MobileLocation(string(c.Request().Header.RequestURI())))
}

func (r *Route) StartOAuth(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "startOAuth")

	var in logicoauth.AuthorizeInput
	if err := req.Body(&in); err != nil {
		return err
	}

	return logicoauth.Authorize(&in)
}

func (r *Route) FinishOAuth(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "finishOAuth")

	var in logicoauth.CallbackInput
	if err := req.Body(&in); err != nil {
		return err
	}

	return logicoauth.Callback(&in)
}

func (r *Route) LinkOAuthAccount(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "linkOAuthAccount")
	if err := req.Require(logicauth.Requirement{}); err != nil {
		return err
	}

	var in logicoauth.CallbackInput
	if err := req.Body(&in); err != nil {
		return err
	}

	err := logicoauth.Link(&in, req.Cookie(cookieState), req.Cookie(cookieVerifier))
	if discovery, ok := errors.AsType[logicoauth.DiscoveryError](err); ok {
		return &problem.Error{Status: fiber.StatusInternalServerError, Message: discovery.Error()}
	}

	return err
}

func (r *Route) UnlinkOAuthAccount(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "unlinkOAuthAccount")
	if err := req.Require(logicauth.Requirement{}); err != nil {
		return err
	}

	u, err := logicoauth.Unlink(req.Context(), r.Runtime, req.Auth)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewUserAdmin(u))
}

func (r *Route) LogoutOAuth(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "logoutOAuth")

	var in logicoauth.LogoutInput
	if err := req.Body(&in); err != nil {
		return err
	}

	return logicoauth.Logout(&in)
}

func (r *Route) UnlinkAllOAuthAccountsAdmin(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "unlinkAllOAuthAccountsAdmin")
	if err := req.Require(logicauth.Requirement{Permission: permission.AdminAuthUnlinkAll, Admin: true}); err != nil {
		return err
	}
	if err := logicoauth.UnlinkAll(req.Context(), r.Runtime); err != nil {
		return err
	}

	return req.NoContent()
}
