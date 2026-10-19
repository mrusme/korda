package auth

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	"tty.fail/mrus/korda/models/permission"
)

func (r *Route) ChangePassword(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "changePassword")
	if err := req.Require(logicauth.Requirement{Permission: permission.AuthChangePassword}); err != nil {
		return err
	}

	var in logicauth.ChangePasswordInput
	if err := req.Body(&in); err != nil {
		return err
	}

	u, err := logicauth.ChangePassword(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewUserAdmin(u))
}
