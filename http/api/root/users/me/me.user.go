package me

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logicusers "tty.fail/mrus/korda/logic/root/users"
	"tty.fail/mrus/korda/models/permission"
)

func (r *Route) GetMyUser(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getMyUser")
	if err := req.Require(logicauth.Requirement{Permission: permission.UserRead}); err != nil {
		return err
	}

	u, err := logicusers.Get(req.Context(), r.Runtime, req.Auth.User.ID, false)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewUserAdmin(u))
}

func (r *Route) UpdateMyUser(c fiber.Ctx) error {
	return r.updateMyUser(c, "updateMyUser")
}

func (r *Route) UpdateMyUserV3(c fiber.Ctx) error {
	return r.updateMyUser(c, "updateMyUserV3")
}

func (r *Route) updateMyUser(c fiber.Ctx, operation string) error {
	req := request.New(r.Runtime, c, operation)
	if err := req.Require(logicauth.Requirement{Permission: permission.UserUpdate}); err != nil {
		return err
	}

	var in logicusers.UpdateMeInput
	if err := req.Body(&in); err != nil {
		return err
	}

	u, err := logicusers.UpdateMe(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewUserAdmin(u))
}

func (r *Route) GetMyCalendarHeatmap(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getMyCalendarHeatmap")
	if err := req.Require(logicauth.Requirement{Permission: permission.UserRead}); err != nil {
		return err
	}

	in, err := BindHeatmap(req)
	if err != nil {
		return err
	}

	heatmap, err := logicusers.CalendarHeatmap(req.Context(), r.Runtime, req.Auth.User.ID, in, time.Now())
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewCalendarHeatmap(heatmap))
}

func BindHeatmap(req *request.Request) (*logicusers.HeatmapInput, error) {
	var in logicusers.HeatmapInput
	if err := req.Query(&in); err != nil {
		return nil, err
	}
	if !in.Ordered() {
		return nil, request.Validation(request.Custom([]any{"from"}, "from must be before to"))
	}

	return &in, nil
}
