package auth

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
)

func (r *Route) Login(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "login")

	var in logicauth.LoginInput
	if err := req.Body(&in); err != nil {
		return err
	}

	result, err := logicauth.Login(req.Context(), r.Runtime, &in, req.LoginDetails())
	if err != nil {
		return err
	}

	req.SetAuthCookies(result.Token)
	return req.Respond(fiber.StatusCreated, view.NewLogin(result))
}

func (r *Route) SignUpAdmin(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "signUpAdmin")
	if err := req.RequireSetup(); err != nil {
		return err
	}

	var in logicauth.SignUpInput
	if err := req.Body(&in); err != nil {
		return err
	}

	u, err := logicauth.SignUpAdmin(req.Context(), r.Runtime, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusCreated, view.NewUserAdmin(u))
}

func (r *Route) ValidateAccessToken(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "validateAccessToken")
	if err := req.Require(logicauth.Requirement{NoPermission: true}); err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.ValidateAccessToken{AuthStatus: true})
}

func (r *Route) Logout(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "logout")
	if err := req.Require(logicauth.Requirement{}); err != nil {
		return err
	}

	result, err := logicauth.Logout(req.Context(), r.Runtime, req.Auth)
	if err != nil {
		return err
	}

	req.ClearAuthCookies()
	return req.Respond(fiber.StatusOK, view.NewLogout(result))
}

func (r *Route) GetAuthStatus(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getAuthStatus")
	if err := req.Require(logicauth.Requirement{}); err != nil {
		return err
	}

	status, err := logicauth.GetStatus(req.Context(), r.Runtime, req.Auth)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewAuthStatus(status))
}
