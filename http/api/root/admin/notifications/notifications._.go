package notifications

import (
	"net/url"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logicnotifications "tty.fail/mrus/korda/logic/root/notifications"
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
	r.Path = route.For("APIAdminNotifications").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Post("", r.CreateNotification).Name("createNotification")
		base.Post("/templates/:name", r.GetNotificationTemplateAdmin).Name("getNotificationTemplateAdmin")
		base.Post("/test-email", r.SendTestEmailAdmin).Name("sendTestEmailAdmin")
	}, r.Path+".")

	return r, nil
}

func (r *Route) CreateNotification(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "createNotification")
	if err := req.Require(logicauth.Requirement{Admin: true}); err != nil {
		return err
	}

	var in logicnotifications.CreateInput
	if err := req.Body(&in); err != nil {
		return err
	}

	n, err := logicnotifications.Create(req.Context(), r.Runtime, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusCreated, view.NewNotification(n))
}

func (r *Route) GetNotificationTemplateAdmin(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getNotificationTemplateAdmin")
	if err := req.Require(logicauth.Requirement{Admin: true}); err != nil {
		return err
	}

	var in logicnotifications.TemplateInput
	if err := req.Body(&in); err != nil {
		return err
	}

	name := c.Params("name")
	if decoded, err := url.PathUnescape(name); err == nil {
		name = decoded
	}

	return req.Respond(fiber.StatusOK, view.NewNotificationTemplate(logicnotifications.RenderTemplate(name, &in)))
}

func (r *Route) SendTestEmailAdmin(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "sendTestEmailAdmin")
	if err := req.Require(logicauth.Requirement{Admin: true}); err != nil {
		return err
	}

	var in logicnotifications.SMTPInput
	if err := req.Body(&in); err != nil {
		return err
	}

	return logicnotifications.SendTestEmail(&in)
}
