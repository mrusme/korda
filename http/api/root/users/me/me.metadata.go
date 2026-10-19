package me

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logicusers "tty.fail/mrus/korda/logic/root/users"
	"tty.fail/mrus/korda/models/permission"
)

func (r *Route) GetUserLicense(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getUserLicense")
	if err := req.Require(logicauth.Requirement{Permission: permission.UserLicenseRead}); err != nil {
		return err
	}

	return logicusers.License(req.Context(), r.Runtime, req.Auth)
}

func (r *Route) SetUserLicense(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "setUserLicense")
	if err := req.Require(logicauth.Requirement{Permission: permission.UserLicenseUpdate}); err != nil {
		return err
	}

	var in logicusers.LicenseInput
	if err := req.Body(&in); err != nil {
		return err
	}

	return logicusers.SetLicense(req.Context(), r.Runtime, req.Auth, &in)
}

func (r *Route) DeleteUserLicense(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "deleteUserLicense")
	if err := req.Require(logicauth.Requirement{Permission: permission.UserLicenseDelete}); err != nil {
		return err
	}

	if err := logicusers.DeleteLicense(req.Context(), r.Runtime, req.Auth); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) GetUserOnboarding(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getUserOnboarding")
	if err := req.Require(logicauth.Requirement{Permission: permission.UserOnboardingRead}); err != nil {
		return err
	}

	onboarded, err := logicusers.Onboarding(req.Context(), r.Runtime, req.Auth)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.Onboarding{IsOnboarded: onboarded})
}

func (r *Route) SetUserOnboarding(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "setUserOnboarding")
	if err := req.Require(logicauth.Requirement{Permission: permission.UserOnboardingUpdate}); err != nil {
		return err
	}

	var in logicusers.OnboardingInput
	if err := req.Body(&in); err != nil {
		return err
	}

	onboarded, err := logicusers.SetOnboarding(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.Onboarding{IsOnboarded: onboarded})
}

func (r *Route) DeleteUserOnboarding(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "deleteUserOnboarding")
	if err := req.Require(logicauth.Requirement{Permission: permission.UserOnboardingDelete}); err != nil {
		return err
	}

	if err := logicusers.DeleteOnboarding(req.Context(), r.Runtime, req.Auth); err != nil {
		return err
	}

	return req.NoContent()
}
