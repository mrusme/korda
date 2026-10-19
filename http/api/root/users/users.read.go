package users

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logicusers "tty.fail/mrus/korda/logic/root/users"
	"tty.fail/mrus/korda/models/permission"
)

func (r *Route) SearchUsers(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "searchUsers")
	if err := req.Require(logicauth.Requirement{Permission: permission.UserRead}); err != nil {
		return err
	}

	list, err := logicusers.Search(req.Context(), r.Runtime, req.Auth)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewUsers(list))
}

func (r *Route) GetUser(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getUser")
	if err := req.Require(logicauth.Requirement{Permission: permission.UserRead}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	u, err := logicusers.Get(req.Context(), r.Runtime, id, false)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewUser(u))
}
