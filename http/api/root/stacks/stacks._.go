package stacks

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logicstacks "tty.fail/mrus/korda/logic/root/stacks"
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
	r.Path = route.For("APIStacks").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Get("", r.SearchStacks).Name("searchStacks")
		base.Post("", r.CreateStack).Name("createStack")
		base.Delete("", r.DeleteStacks).Name("deleteStacks")
		base.Get("/:id", r.GetStack).Name("getStack")
		base.Put("/:id", r.UpdateStack).Name("updateStack")
		base.Patch("/:id", r.UpdateStackV3).Name("updateStackV3")
		base.Delete("/:id", r.DeleteStack).Name("deleteStack")
		base.Delete("/:id/assets/:assetId", r.RemoveAssetFromStack).Name("removeAssetFromStack")
	}, r.Path+".")

	return r, nil
}

func (r *Route) respond(req *request.Request, status int, d *logicstacks.Details) error {
	out, err := view.NewStack(req.Context(), r.Runtime, d, req.Auth.User.ID)
	if err != nil {
		return err
	}

	return req.Respond(status, out)
}

func (r *Route) SearchStacks(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "searchStacks")
	if err := req.Require(logicauth.Requirement{Permission: permission.StackRead}); err != nil {
		return err
	}

	var in logicstacks.SearchInput
	if err := req.Query(&in); err != nil {
		return err
	}

	list, err := logicstacks.Search(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	out, err := view.NewStacks(req.Context(), r.Runtime, list, req.Auth.User.ID)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, out)
}

func (r *Route) CreateStack(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "createStack")
	if err := req.Require(logicauth.Requirement{Permission: permission.StackCreate}); err != nil {
		return err
	}

	var in logicstacks.CreateInput
	if err := req.Body(&in); err != nil {
		return err
	}

	d, err := logicstacks.Create(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return r.respond(req, fiber.StatusCreated, d)
}

func (r *Route) DeleteStacks(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "deleteStacks")
	if err := req.Require(logicauth.Requirement{Permission: permission.StackDelete}); err != nil {
		return err
	}

	var in logicstacks.BulkIDsInput
	if err := req.Body(&in); err != nil {
		return err
	}

	if err := logicstacks.DeleteAll(req.Context(), r.Runtime, req.Auth, &in); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) GetStack(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getStack")
	if err := req.Require(logicauth.Requirement{Permission: permission.StackRead}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	d, err := logicstacks.Get(req.Context(), r.Runtime, req.Auth, id)
	if err != nil {
		return err
	}

	return r.respond(req, fiber.StatusOK, d)
}

func (r *Route) UpdateStack(c fiber.Ctx) error {
	return r.update(c, "updateStack")
}

func (r *Route) UpdateStackV3(c fiber.Ctx) error {
	return r.update(c, "updateStackV3")
}

func (r *Route) update(c fiber.Ctx, operation string) error {
	req := request.New(r.Runtime, c, operation)
	if err := req.Require(logicauth.Requirement{Permission: permission.StackUpdate}); err != nil {
		return err
	}

	var in logicstacks.UpdateInput
	if err := req.Body(&in); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	d, err := logicstacks.Update(req.Context(), r.Runtime, req.Auth, id, &in)
	if err != nil {
		return err
	}

	return r.respond(req, fiber.StatusOK, d)
}

func (r *Route) DeleteStack(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "deleteStack")
	if err := req.Require(logicauth.Requirement{Permission: permission.StackDelete}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	if err = logicstacks.Delete(req.Context(), r.Runtime, req.Auth, id); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) RemoveAssetFromStack(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "removeAssetFromStack")
	if err := req.Require(logicauth.Requirement{Permission: permission.StackUpdate}); err != nil {
		return err
	}

	var in logicstacks.RemoveAssetInput
	if err := req.Params(&in, "id", "assetId"); err != nil {
		return err
	}

	if err := logicstacks.RemoveAsset(req.Context(), r.Runtime, req.Auth, &in); err != nil {
		return err
	}

	return req.NoContent()
}
