package jobs

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logicqueues "tty.fail/mrus/korda/logic/root/queues"
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
	r.Path = route.For("APIJobs").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Get("", r.GetQueuesLegacy).Name("getQueuesLegacy")
		base.Post("", r.CreateJob).Name("createJob")
		base.Put("/:name", r.RunQueueCommandLegacy).Name("runQueueCommandLegacy")
	}, r.Path+".")

	return r, nil
}

func (r *Route) GetQueuesLegacy(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getQueuesLegacy")
	if err := req.Require(logicauth.Requirement{Permission: permission.JobRead, Admin: true}); err != nil {
		return err
	}

	list, err := logicqueues.List(r.Runtime)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewQueuesLegacy(list))
}

func (r *Route) CreateJob(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "createJob")
	if err := req.Require(logicauth.Requirement{Permission: permission.JobCreate, Admin: true}); err != nil {
		return err
	}

	var in logicqueues.CreateJobInput
	if err := req.Body(&in); err != nil {
		return err
	}

	if err := logicqueues.CreateJob(r.Runtime, &in); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) RunQueueCommandLegacy(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "runQueueCommandLegacy")
	if err := req.Require(logicauth.Requirement{Permission: permission.JobCreate, Admin: true}); err != nil {
		return err
	}

	var in logicqueues.CommandInput
	if err := req.Body(&in); err != nil {
		return err
	}

	var p logicqueues.NameParams
	if err := req.Params(&p, "name"); err != nil {
		return err
	}

	q, err := logicqueues.Command(r.Runtime, p.Name, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewQueueLegacy(q))
}
