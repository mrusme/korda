package auth

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	"tty.fail/mrus/korda/models/permission"
)

func (r *Route) SetupPinCode(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "setupPinCode")
	if err := req.Require(logicauth.Requirement{Permission: permission.PinCodeCreate}); err != nil {
		return err
	}

	var in logicauth.PinCodeSetupInput
	if err := req.Body(&in); err != nil {
		return err
	}

	if err := logicauth.SetupPinCode(req.Context(), r.Runtime, req.Auth, &in); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) ChangePinCode(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "changePinCode")
	if err := req.Require(logicauth.Requirement{Permission: permission.PinCodeUpdate}); err != nil {
		return err
	}

	var in logicauth.PinCodeChangeInput
	if err := req.Body(&in); err != nil {
		return err
	}

	if err := logicauth.ChangePinCode(req.Context(), r.Runtime, req.Auth, &in); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) ResetPinCode(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "resetPinCode")
	if err := req.Require(logicauth.Requirement{Permission: permission.PinCodeDelete}); err != nil {
		return err
	}

	var in logicauth.PinCodeResetInput
	if err := req.Body(&in); err != nil {
		return err
	}

	if err := logicauth.ResetPinCode(req.Context(), r.Runtime, req.Auth, &in); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) UnlockAuthSession(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "unlockAuthSession")
	if err := req.Require(logicauth.Requirement{}); err != nil {
		return err
	}

	var in logicauth.SessionUnlockInput
	if err := req.Body(&in); err != nil {
		return err
	}

	if err := logicauth.UnlockSession(req.Context(), r.Runtime, req.Auth, &in); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) LockAuthSession(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "lockAuthSession")
	if err := req.Require(logicauth.Requirement{}); err != nil {
		return err
	}

	if err := logicauth.LockSession(req.Context(), r.Runtime, req.Auth); err != nil {
		return err
	}

	return req.NoContent()
}
