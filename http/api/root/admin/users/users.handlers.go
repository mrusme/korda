package users

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/root/users/me"
	"tty.fail/mrus/korda/http/api/view"
	logicadminusers "tty.fail/mrus/korda/logic/root/admin/users"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logicusers "tty.fail/mrus/korda/logic/root/users"
	"tty.fail/mrus/korda/models/permission"
	"tty.fail/mrus/korda/models/preferences"
)

func (r *Route) SearchUsersAdmin(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "searchUsersAdmin")
	if err := req.Require(logicauth.Requirement{Permission: permission.AdminUserRead, Admin: true}); err != nil {
		return err
	}

	var in logicadminusers.SearchInput
	if err := req.Query(&in); err != nil {
		return err
	}

	list, err := logicadminusers.Search(req.Context(), r.Runtime, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewUserAdmins(list))
}

func (r *Route) CreateUserAdmin(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "createUserAdmin")
	if err := req.Require(logicauth.Requirement{Permission: permission.AdminUserCreate, Admin: true}); err != nil {
		return err
	}

	var in logicadminusers.CreateInput
	if err := req.Body(&in); err != nil {
		return err
	}

	u, err := logicadminusers.Create(req.Context(), r.Runtime, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusCreated, view.NewUserAdmin(u))
}

func (r *Route) GetUserAdmin(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getUserAdmin")
	if err := req.Require(logicauth.Requirement{Permission: permission.AdminUserRead, Admin: true}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	u, err := logicusers.Get(req.Context(), r.Runtime, id, true)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewUserAdmin(u))
}

func (r *Route) UpdateUserAdmin(c fiber.Ctx) error {
	return r.update(c, "updateUserAdmin")
}

func (r *Route) UpdateUserAdminV3(c fiber.Ctx) error {
	return r.update(c, "updateUserAdminV3")
}

func (r *Route) update(c fiber.Ctx, operation string) error {
	req := request.New(r.Runtime, c, operation)
	if err := req.Require(logicauth.Requirement{Permission: permission.AdminUserUpdate, Admin: true}); err != nil {
		return err
	}

	var in logicadminusers.UpdateInput
	if err := req.Body(&in); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	u, err := logicadminusers.Update(req.Context(), r.Runtime, req.Auth, id, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewUserAdmin(u))
}

func (r *Route) DeleteUserAdmin(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "deleteUserAdmin")
	if err := req.Require(logicauth.Requirement{Permission: permission.AdminUserDelete, Admin: true}); err != nil {
		return err
	}

	var in logicadminusers.DeleteInput
	if err := req.Body(&in); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	u, err := logicadminusers.Delete(req.Context(), r.Runtime, req.Auth, id, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewUserAdmin(u))
}

func (r *Route) RestoreUserAdmin(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "restoreUserAdmin")
	if err := req.Require(logicauth.Requirement{Permission: permission.AdminUserDelete, Admin: true}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	u, err := logicadminusers.Restore(req.Context(), r.Runtime, id)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewUserAdmin(u))
}

func (r *Route) GetUserCalendarHeatmapAdmin(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getUserCalendarHeatmapAdmin")
	if err := req.Require(logicauth.Requirement{Permission: permission.AdminUserRead, Admin: true}); err != nil {
		return err
	}

	in, err := me.BindHeatmap(req)
	if err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	if _, err = logicusers.Get(req.Context(), r.Runtime, id, false); err != nil {
		return err
	}

	heatmap, err := logicusers.CalendarHeatmap(req.Context(), r.Runtime, id, in, time.Now())
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewCalendarHeatmap(heatmap))
}

func (r *Route) GetUserSessionsAdmin(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getUserSessionsAdmin")
	if err := req.Require(logicauth.Requirement{Permission: permission.AdminSessionRead, Admin: true}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	list, err := logicadminusers.Sessions(req.Context(), r.Runtime, id)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewSessions(list, uuid.Nil))
}

func (r *Route) GetUserStatisticsAdmin(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getUserStatisticsAdmin")
	if err := req.Require(logicauth.Requirement{Permission: permission.AdminUserRead, Admin: true}); err != nil {
		return err
	}

	var in logicadminusers.StatisticsInput
	if err := req.Query(&in); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	stats, err := logicadminusers.Statistics(req.Context(), r.Runtime, id, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.AssetStats{Images: stats.Images, Videos: stats.Videos, Total: stats.Total})
}

func (r *Route) GetUserPreferencesAdmin(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getUserPreferencesAdmin")
	if err := req.Require(logicauth.Requirement{Permission: permission.AdminUserRead, Admin: true}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	prefs, err := logicadminusers.Preferences(req.Context(), r.Runtime, id)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, prefs)
}

func (r *Route) UpdateUserPreferencesAdmin(c fiber.Ctx) error {
	return r.updatePreferences(c, "updateUserPreferencesAdmin")
}

func (r *Route) UpdateUserPreferencesAdminV3(c fiber.Ctx) error {
	return r.updatePreferences(c, "updateUserPreferencesAdminV3")
}

func (r *Route) updatePreferences(c fiber.Ctx, operation string) error {
	req := request.New(r.Runtime, c, operation)
	if err := req.Require(logicauth.Requirement{Permission: permission.AdminUserUpdate, Admin: true}); err != nil {
		return err
	}

	var in preferences.Update
	if err := req.Body(&in); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	prefs, err := logicadminusers.UpdatePreferences(req.Context(), r.Runtime, id, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, prefs)
}
