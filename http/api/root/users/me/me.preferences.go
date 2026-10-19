package me

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logicusers "tty.fail/mrus/korda/logic/root/users"
	"tty.fail/mrus/korda/models/permission"
	"tty.fail/mrus/korda/models/preferences"
)

func (r *Route) GetMyPreferences(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getMyPreferences")
	if err := req.Require(logicauth.Requirement{Permission: permission.UserPreferenceRead}); err != nil {
		return err
	}

	prefs, err := logicusers.Preferences(req.Context(), r.Runtime, req.Auth.User.ID)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, prefs)
}

func (r *Route) UpdateMyPreferences(c fiber.Ctx) error {
	return r.updateMyPreferences(c, "updateMyPreferences")
}

func (r *Route) UpdateMyPreferencesV3(c fiber.Ctx) error {
	return r.updateMyPreferences(c, "updateMyPreferencesV3")
}

func (r *Route) updateMyPreferences(c fiber.Ctx, operation string) error {
	req := request.New(r.Runtime, c, operation)
	if err := req.Require(logicauth.Requirement{Permission: permission.UserPreferenceUpdate}); err != nil {
		return err
	}

	var in preferences.Update
	if err := req.Body(&in); err != nil {
		return err
	}

	prefs, err := logicusers.UpdatePreferences(req.Context(), r.Runtime, req.Auth.User.ID, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, prefs)
}
