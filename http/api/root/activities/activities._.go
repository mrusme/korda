package activities

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicactivities "tty.fail/mrus/korda/logic/root/activities"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
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
	r.Path = route.For("APIActivities").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Get("", r.GetActivities).Name("getActivities")
		base.Post("", r.CreateActivity).Name("createActivity")
		base.Get("/statistics", r.GetActivityStatistics).Name("getActivityStatistics")
		base.Delete("/:id", r.DeleteActivity).Name("deleteActivity")
	}, r.Path+".")

	return r, nil
}

func (r *Route) GetActivities(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getActivities")
	if err := req.Require(logicauth.Requirement{Permission: permission.ActivityRead}); err != nil {
		return err
	}

	var in logicactivities.SearchInput
	if err := req.Query(&in); err != nil {
		return err
	}

	list, err := logicactivities.Search(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewActivities(list))
}

func (r *Route) CreateActivity(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "createActivity")
	if err := req.Require(logicauth.Requirement{Permission: permission.ActivityCreate}); err != nil {
		return err
	}

	var in logicactivities.CreateInput
	if err := req.Body(&in); err != nil {
		return err
	}

	row, duplicate, err := logicactivities.Create(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	status := fiber.StatusCreated
	if duplicate {
		status = fiber.StatusOK
	}

	return req.Respond(status, view.NewActivity(row))
}

func (r *Route) GetActivityStatistics(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getActivityStatistics")
	if err := req.Require(logicauth.Requirement{Permission: permission.ActivityStatistics}); err != nil {
		return err
	}

	var in logicactivities.StatisticsInput
	if err := req.Query(&in); err != nil {
		return err
	}

	s, err := logicactivities.Statistics(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.ActivityStatistics{Comments: s.Comments, Likes: s.Likes})
}

func (r *Route) DeleteActivity(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "deleteActivity")
	if err := req.Require(logicauth.Requirement{Permission: permission.ActivityDelete}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	if err = logicactivities.Delete(req.Context(), r.Runtime, req.Auth, id); err != nil {
		return err
	}

	return req.NoContent()
}
