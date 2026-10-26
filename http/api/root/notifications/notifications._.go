package notifications

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logicnotifications "tty.fail/mrus/korda/logic/root/notifications"
	"tty.fail/mrus/korda/models/permission"
	"xn--gckvb8fzb.com/glides/http/route"
	"xn--gckvb8fzb.com/glides/runtime"
)

type Route struct {
	route.RouteController
}

func New(rt *runtime.Runtime, router fiber.Router) (*Route, error) {
	r := new(Route)

	r.Runtime = rt
	r.Router = router
	r.Path = route.For("APINotifications").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Get("", r.GetNotifications).Name("getNotifications")
		base.Put("", r.UpdateNotifications).Name("updateNotifications")
		base.Delete("", r.DeleteNotifications).Name("deleteNotifications")
		base.Get("/:id", r.GetNotification).Name("getNotification")
		base.Put("/:id", r.UpdateNotification).Name("updateNotification")
		base.Delete("/:id", r.DeleteNotification).Name("deleteNotification")
	}, r.Path+".")

	return r, nil
}

func (r *Route) GetNotifications(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getNotifications")
	if err := req.Require(logicauth.Requirement{Permission: permission.NotificationRead}); err != nil {
		return err
	}

	var in logicnotifications.SearchInput
	if err := req.Query(&in); err != nil {
		return err
	}

	list, err := logicnotifications.Search(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewNotifications(list))
}

func (r *Route) UpdateNotifications(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "updateNotifications")
	if err := req.Require(logicauth.Requirement{Permission: permission.NotificationUpdate}); err != nil {
		return err
	}

	var in logicnotifications.UpdateAllInput
	if err := req.Body(&in); err != nil {
		return err
	}

	if err := logicnotifications.UpdateAll(req.Context(), r.Runtime, req.Auth, &in); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) DeleteNotifications(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "deleteNotifications")
	if err := req.Require(logicauth.Requirement{Permission: permission.NotificationDelete}); err != nil {
		return err
	}

	var in logicnotifications.DeleteAllInput
	if err := req.Body(&in); err != nil {
		return err
	}

	if err := logicnotifications.DeleteAll(req.Context(), r.Runtime, req.Auth, &in); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) GetNotification(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getNotification")
	if err := req.Require(logicauth.Requirement{Permission: permission.NotificationRead}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	n, err := logicnotifications.Get(req.Context(), r.Runtime, req.Auth, id)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewNotification(n))
}

func (r *Route) UpdateNotification(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "updateNotification")
	if err := req.Require(logicauth.Requirement{Permission: permission.NotificationUpdate}); err != nil {
		return err
	}

	var in logicnotifications.UpdateInput
	if err := req.Body(&in); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	n, err := logicnotifications.Update(req.Context(), r.Runtime, req.Auth, id, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewNotification(n))
}

func (r *Route) DeleteNotification(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "deleteNotification")
	if err := req.Require(logicauth.Requirement{Permission: permission.NotificationDelete}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	if err = logicnotifications.Delete(req.Context(), r.Runtime, req.Auth, []uuid.UUID{id}); err != nil {
		return err
	}

	return req.NoContent()
}
