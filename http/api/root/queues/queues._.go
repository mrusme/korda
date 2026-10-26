package queues

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
	r.Path = route.For("APIQueues").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Get("", r.GetQueues).Name("getQueues")
		base.Get("/:name", r.GetQueue).Name("getQueue")
		base.Put("/:name", r.UpdateQueue).Name("updateQueue")
		base.Get("/:name/jobs", r.GetQueueJobs).Name("getQueueJobs")
		base.Delete("/:name/jobs", r.EmptyQueue).Name("emptyQueue")
	}, r.Path+".")

	return r, nil
}

func (r *Route) GetQueues(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getQueues")
	if err := req.Require(logicauth.Requirement{Permission: permission.QueueRead, Admin: true}); err != nil {
		return err
	}

	list, err := logicqueues.List(r.Runtime)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewQueues(list))
}

func (r *Route) GetQueue(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getQueue")
	if err := req.Require(logicauth.Requirement{Permission: permission.QueueRead, Admin: true}); err != nil {
		return err
	}

	var p logicqueues.NameParams
	if err := req.Params(&p, "name"); err != nil {
		return err
	}

	q, err := logicqueues.Get(r.Runtime, p.Name)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewQueue(q))
}

func (r *Route) UpdateQueue(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "updateQueue")
	if err := req.Require(logicauth.Requirement{Permission: permission.QueueUpdate, Admin: true}); err != nil {
		return err
	}

	var in logicqueues.UpdateInput
	if err := req.Body(&in); err != nil {
		return err
	}

	var p logicqueues.NameParams
	if err := req.Params(&p, "name"); err != nil {
		return err
	}

	q, err := logicqueues.Update(r.Runtime, p.Name, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewQueue(q))
}

func (r *Route) GetQueueJobs(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getQueueJobs")
	if err := req.Require(logicauth.Requirement{Permission: permission.QueueJobRead, Admin: true}); err != nil {
		return err
	}

	var in logicqueues.JobsInput
	if err := req.Query(&in); err != nil {
		return err
	}

	var p logicqueues.NameParams
	if err := req.Params(&p, "name"); err != nil {
		return err
	}

	tasks, err := logicqueues.Jobs(r.Runtime, p.Name, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewQueueJobs(tasks))
}

func (r *Route) EmptyQueue(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "emptyQueue")
	if err := req.Require(logicauth.Requirement{Permission: permission.QueueJobDelete, Admin: true}); err != nil {
		return err
	}

	var in logicqueues.EmptyInput
	if err := req.Body(&in); err != nil {
		return err
	}

	var p logicqueues.NameParams
	if err := req.Params(&p, "name"); err != nil {
		return err
	}

	if err := logicqueues.Empty(r.Runtime, p.Name, &in); err != nil {
		return err
	}

	return req.NoContent()
}
