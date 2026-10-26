package clustergroups

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logicclustergroups "tty.fail/mrus/korda/logic/root/clustergroups"
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
	r.Path = route.For("APIClusterGroups").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Get("/requests", r.GetClusterGroupRequests).Name("getClusterGroupRequests")
		base.Post("/requests/:id/accept", r.AcceptClusterGroupRequest).Name("acceptClusterGroupRequest")
		base.Delete("/requests/:id", r.DeleteClusterGroupRequest).Name("deleteClusterGroupRequest")
		base.Get("/:id/requests", r.GetClusterGroupRequestsForGroup).Name("getClusterGroupRequestsForGroup")
		base.Get("/:id/users", r.GetClusterGroupUsers).Name("getClusterGroupUsers")
		base.Put("/:id/requests", r.CreateClusterGroupRequest).Name("createClusterGroupRequest")
		base.Post("/:id/regenerate-people", r.ClusterGroupRegeneratePeople).Name("clusterGroupRegeneratePeople")
		base.Post("/:id/leave", r.LeaveClusterGroup).Name("leaveClusterGroup")
	}, r.Path+".")

	return r, nil
}

func (r *Route) start(c fiber.Ctx, operation string, p permission.Permission) (*request.Request, error) {
	req := request.New(r.Runtime, c, operation)
	return req, req.Require(logicauth.Requirement{Permission: p})
}

func (r *Route) GetClusterGroupRequests(c fiber.Ctx) error {
	req, err := r.start(c, "getClusterGroupRequests", permission.ClusterGroupRequestRead)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, []struct{}{})
}

func (r *Route) AcceptClusterGroupRequest(c fiber.Ctx) error {
	req, err := r.start(c, "acceptClusterGroupRequest", permission.ClusterGroupRequestCreate)
	if err != nil {
		return err
	}
	if _, err = req.UUID("id"); err != nil {
		return err
	}

	return logicclustergroups.Accept()
}

func (r *Route) DeleteClusterGroupRequest(c fiber.Ctx) error {
	req, err := r.start(c, "deleteClusterGroupRequest", permission.ClusterGroupRequestDelete)
	if err != nil {
		return err
	}
	if _, err = req.UUID("id"); err != nil {
		return err
	}

	return logicclustergroups.DeleteRequest()
}

func (r *Route) GetClusterGroupRequestsForGroup(c fiber.Ctx) error {
	req, err := r.start(c, "getClusterGroupRequestsForGroup", permission.ClusterGroupRequestRead)
	if err != nil {
		return err
	}
	id, err := req.UUID("id")
	if err != nil {
		return err
	}
	if err = logicclustergroups.Requests(req.Auth, id); err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, []struct{}{})
}

func (r *Route) GetClusterGroupUsers(c fiber.Ctx) error {
	req, err := r.start(c, "getClusterGroupUsers", permission.ClusterGroupRead)
	if err != nil {
		return err
	}
	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	users, err := logicclustergroups.Users(req.Context(), r.Runtime, req.Auth, id)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewUsers(users))
}

func (r *Route) CreateClusterGroupRequest(c fiber.Ctx) error {
	req, err := r.start(c, "createClusterGroupRequest", permission.ClusterGroupRequestCreate)
	if err != nil {
		return err
	}

	var in logicclustergroups.RequestInput
	if err = req.Body(&in); err != nil {
		return err
	}
	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	return logicclustergroups.CreateRequest(req.Context(), r.Runtime, req.Auth, id, &in)
}

func (r *Route) ClusterGroupRegeneratePeople(c fiber.Ctx) error {
	req, err := r.start(c, "clusterGroupRegeneratePeople", permission.ClusterGroupRead)
	if err != nil {
		return err
	}
	id, err := req.UUID("id")
	if err != nil {
		return err
	}
	if err = logicclustergroups.RegeneratePeople(req.Auth, id); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) LeaveClusterGroup(c fiber.Ctx) error {
	req, err := r.start(c, "leaveClusterGroup", permission.ClusterGroupLeave)
	if err != nil {
		return err
	}
	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	return logicclustergroups.Leave(req.Auth, id)
}
