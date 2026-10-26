package duplicates

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicassets "tty.fail/mrus/korda/logic/root/assets"
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
	r.Path = route.For("APIDuplicates").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Get("", r.GetAssetDuplicates).Name("getAssetDuplicates")
		base.Delete("", r.DeleteDuplicates).Name("deleteDuplicates")
		base.Post("/resolve", r.ResolveDuplicates).Name("resolveDuplicates")
		base.Delete("/:id", r.DeleteDuplicate).Name("deleteDuplicate")
	}, r.Path+".")

	return r, nil
}

func (r *Route) GetAssetDuplicates(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getAssetDuplicates")
	if err := req.Require(logicauth.Requirement{Permission: permission.DuplicateRead}); err != nil {
		return err
	}

	groups, err := logicassets.Duplicates(req.Context(), r.Runtime, req.Auth)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewDuplicates(groups, req.Auth.User.ID))
}

func (r *Route) DeleteDuplicates(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "deleteDuplicates")
	if err := req.Require(logicauth.Requirement{Permission: permission.DuplicateDelete}); err != nil {
		return err
	}

	var in logicassets.BulkIDsInput
	if err := req.Body(&in); err != nil {
		return err
	}

	ids := make([]uuid.UUID, 0, len(in.IDs))
	for _, s := range in.IDs {
		ids = append(ids, uuid.MustParse(s))
	}
	if err := logicassets.DeleteDuplicates(req.Context(), r.Runtime, req.Auth, ids); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) DeleteDuplicate(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "deleteDuplicate")
	if err := req.Require(logicauth.Requirement{Permission: permission.DuplicateDelete}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}
	if err = logicassets.DeleteDuplicates(req.Context(), r.Runtime, req.Auth, []uuid.UUID{id}); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) ResolveDuplicates(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "resolveDuplicates")
	if err := req.Require(logicauth.Requirement{Permission: permission.DuplicateDelete}); err != nil {
		return err
	}

	var in logicassets.ResolveInput
	if err := req.Body(&in); err != nil {
		return err
	}

	results, err := logicassets.ResolveDuplicates(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewResolveResults(results))
}
