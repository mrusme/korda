package server

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/errs"
	"tty.fail/mrus/korda/http/api/request"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	"tty.fail/mrus/korda/logic/root/users"
	"tty.fail/mrus/korda/models/permission"
)

func (r *Route) GetServerLicense(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getServerLicense")
	if err := req.Require(logicauth.Requirement{Permission: permission.ServerLicenseRead, Admin: true}); err != nil {
		return err
	}

	return errs.ErrNotFound
}

func (r *Route) SetServerLicense(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "setServerLicense")
	if err := req.Require(logicauth.Requirement{Permission: permission.ServerLicenseUpdate, Admin: true}); err != nil {
		return err
	}

	var in users.LicenseInput
	if err := req.Body(&in); err != nil {
		return err
	}

	return errs.ErrInvalidLicenseKey
}

func (r *Route) DeleteServerLicense(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "deleteServerLicense")
	if err := req.Require(logicauth.Requirement{Permission: permission.ServerLicenseDelete, Admin: true}); err != nil {
		return err
	}

	return req.NoContent()
}
