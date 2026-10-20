package tags

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logictags "tty.fail/mrus/korda/logic/root/tags"
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
	r.Path = route.For("APITags").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Post("", r.CreateTag).Name("createTag")
		base.Get("", r.GetAllTags).Name("getAllTags")
		base.Put("", r.UpsertTags).Name("upsertTags")
		base.Put("/assets", r.BulkTagAssets).Name("bulkTagAssets")
		base.Get("/:id", r.GetTagById).Name("getTagById")
		base.Put("/:id", r.UpdateTag).Name("updateTag")
		base.Patch("/:id", r.UpdateTagV3).Name("updateTagV3")
		base.Delete("/:id", r.DeleteTag).Name("deleteTag")
		base.Put("/:id/assets", r.TagAssets).Name("tagAssets")
		base.Delete("/:id/assets", r.UntagAssets).Name("untagAssets")
	}, r.Path+".")

	return r, nil
}

func (r *Route) CreateTag(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "createTag")
	if err := req.Require(logicauth.Requirement{Permission: permission.TagCreate}); err != nil {
		return err
	}

	var in logictags.CreateInput
	if err := req.Body(&in); err != nil {
		return err
	}

	t, err := logictags.Create(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusCreated, view.NewTag(t, false))
}

func (r *Route) GetAllTags(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getAllTags")
	if err := req.Require(logicauth.Requirement{Permission: permission.TagRead}); err != nil {
		return err
	}

	list, err := logictags.All(req.Context(), r.Runtime, req.Auth)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewTags(list))
}

func (r *Route) UpsertTags(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "upsertTags")
	if err := req.Require(logicauth.Requirement{Permission: permission.TagCreate}); err != nil {
		return err
	}

	var in logictags.UpsertInput
	if err := req.Body(&in); err != nil {
		return err
	}

	list, err := logictags.Upsert(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewTags(list))
}

func (r *Route) BulkTagAssets(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "bulkTagAssets")
	if err := req.Require(logicauth.Requirement{Permission: permission.TagAsset}); err != nil {
		return err
	}

	var in logictags.BulkAssetsInput
	if err := req.Body(&in); err != nil {
		return err
	}

	count, err := logictags.BulkTag(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.TagBulkAssets{Count: count})
}

func (r *Route) GetTagById(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getTagById")
	if err := req.Require(logicauth.Requirement{Permission: permission.TagRead}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	t, err := logictags.Get(req.Context(), r.Runtime, req.Auth, id)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewTag(t, false))
}

func (r *Route) UpdateTag(c fiber.Ctx) error {
	return r.update(c, "updateTag")
}

func (r *Route) UpdateTagV3(c fiber.Ctx) error {
	return r.update(c, "updateTagV3")
}

func (r *Route) update(c fiber.Ctx, operation string) error {
	req := request.New(r.Runtime, c, operation)
	if err := req.Require(logicauth.Requirement{Permission: permission.TagUpdate}); err != nil {
		return err
	}

	var in logictags.UpdateInput
	if err := req.Body(&in); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	t, err := logictags.Update(req.Context(), r.Runtime, req.Auth, id, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewTag(t, false))
}

func (r *Route) DeleteTag(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "deleteTag")
	if err := req.Require(logicauth.Requirement{Permission: permission.TagDelete}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	if err = logictags.Delete(req.Context(), r.Runtime, req.Auth, id); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) TagAssets(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "tagAssets")
	if err := req.Require(logicauth.Requirement{Permission: permission.TagAsset}); err != nil {
		return err
	}

	var in logictags.BulkIDsInput
	if err := req.Body(&in); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	results, err := logictags.AddAssets(req.Context(), r.Runtime, req.Auth, id, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewBulkIDs(results))
}

func (r *Route) UntagAssets(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "untagAssets")
	if err := req.Require(logicauth.Requirement{Permission: permission.TagAsset}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	var in logictags.BulkIDsInput
	if err = req.Body(&in); err != nil {
		return err
	}

	results, err := logictags.RemoveAssets(req.Context(), r.Runtime, req.Auth, id, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewBulkIDs(results))
}
