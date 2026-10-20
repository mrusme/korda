package memories

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logicmemories "tty.fail/mrus/korda/logic/root/memories"
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
	r.Path = route.For("APIMemories").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Get("", r.SearchMemories).Name("searchMemories")
		base.Post("", r.CreateMemory).Name("createMemory")
		base.Get("/statistics", r.MemoriesStatistics).Name("memoriesStatistics")
		base.Get("/:id", r.GetMemory).Name("getMemory")
		base.Put("/:id", r.UpdateMemory).Name("updateMemory")
		base.Patch("/:id", r.UpdateMemoryV3).Name("updateMemoryV3")
		base.Delete("/:id", r.DeleteMemory).Name("deleteMemory")
		base.Put("/:id/assets", r.AddMemoryAssets).Name("addMemoryAssets")
		base.Delete("/:id/assets", r.RemoveMemoryAssets).Name("removeMemoryAssets")
	}, r.Path+".")

	return r, nil
}

func (r *Route) respond(req *request.Request, status int, d *logicmemories.Details) error {
	out, err := view.NewMemory(d, req.Auth.User.ID)
	if err != nil {
		return err
	}

	return req.Respond(status, out)
}

func (r *Route) SearchMemories(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "searchMemories")
	if err := req.Require(logicauth.Requirement{Permission: permission.MemoryRead}); err != nil {
		return err
	}

	var in logicmemories.SearchInput
	if err := req.Query(&in); err != nil {
		return err
	}

	list, err := logicmemories.Search(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	out, err := view.NewMemories(list, req.Auth.User.ID)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, out)
}

func (r *Route) CreateMemory(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "createMemory")
	if err := req.Require(logicauth.Requirement{Permission: permission.MemoryCreate}); err != nil {
		return err
	}

	var in logicmemories.CreateInput
	if err := req.Body(&in); err != nil {
		return err
	}

	d, err := logicmemories.Create(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return r.respond(req, fiber.StatusCreated, d)
}

func (r *Route) MemoriesStatistics(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "memoriesStatistics")
	if err := req.Require(logicauth.Requirement{Permission: permission.MemoryStatistics}); err != nil {
		return err
	}

	var in logicmemories.SearchInput
	if err := req.Query(&in); err != nil {
		return err
	}

	total, err := logicmemories.Statistics(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.MemoryStatistics{Total: total})
}

func (r *Route) GetMemory(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getMemory")
	if err := req.Require(logicauth.Requirement{Permission: permission.MemoryRead}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	d, err := logicmemories.Get(req.Context(), r.Runtime, req.Auth, id)
	if err != nil {
		return err
	}

	return r.respond(req, fiber.StatusOK, d)
}

func (r *Route) UpdateMemory(c fiber.Ctx) error {
	return r.update(c, "updateMemory")
}

func (r *Route) UpdateMemoryV3(c fiber.Ctx) error {
	return r.update(c, "updateMemoryV3")
}

func (r *Route) update(c fiber.Ctx, operation string) error {
	req := request.New(r.Runtime, c, operation)
	if err := req.Require(logicauth.Requirement{Permission: permission.MemoryUpdate}); err != nil {
		return err
	}

	var in logicmemories.UpdateInput
	if err := req.Body(&in); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	d, err := logicmemories.Update(req.Context(), r.Runtime, req.Auth, id, &in)
	if err != nil {
		return err
	}

	return r.respond(req, fiber.StatusOK, d)
}

func (r *Route) DeleteMemory(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "deleteMemory")
	if err := req.Require(logicauth.Requirement{Permission: permission.MemoryDelete}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	if err = logicmemories.Delete(req.Context(), r.Runtime, req.Auth, id); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) AddMemoryAssets(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "addMemoryAssets")
	if err := req.Require(logicauth.Requirement{Permission: permission.MemoryAssetCreate}); err != nil {
		return err
	}

	var in logicmemories.BulkIDsInput
	if err := req.Body(&in); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	results, err := logicmemories.AddAssets(req.Context(), r.Runtime, req.Auth, id, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewBulkIDs(results))
}

func (r *Route) RemoveMemoryAssets(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "removeMemoryAssets")
	if err := req.Require(logicauth.Requirement{Permission: permission.MemoryAssetDelete}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	var in logicmemories.BulkIDsInput
	if err = req.Body(&in); err != nil {
		return err
	}

	results, err := logicmemories.RemoveAssets(req.Context(), r.Runtime, req.Auth, id, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewBulkIDs(results))
}
