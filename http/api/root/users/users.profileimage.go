package users

import (
	"io"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logicusers "tty.fail/mrus/korda/logic/root/users"
	"tty.fail/mrus/korda/models/permission"
	"tty.fail/mrus/korda/models/user"
)

func (r *Route) CreateProfileImage(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "createProfileImage")
	if err := req.Require(logicauth.Requirement{Permission: permission.UserProfileImageUpdate}); err != nil {
		return err
	}

	var updated *user.User
	err := req.File("file", logicusers.AcceptProfileImage, func(filename string, src io.Reader) (err error) {
		updated, err = logicusers.CreateProfileImage(req.Context(), r.Runtime, req.Auth, src)
		return err
	})
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusCreated, view.NewProfileImage(updated))
}

func (r *Route) DeleteProfileImage(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "deleteProfileImage")
	if err := req.Require(logicauth.Requirement{Permission: permission.UserProfileImageDelete}); err != nil {
		return err
	}

	if err := logicusers.DeleteProfileImage(req.Context(), r.Runtime, req.Auth); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) GetProfileImage(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getProfileImage")
	if err := req.Require(logicauth.Requirement{Permission: permission.UserProfileImageRead}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	path, err := logicusers.ProfileImage(req.Context(), r.Runtime, id)
	if err != nil {
		return err
	}

	return req.SendFile(path)
}
