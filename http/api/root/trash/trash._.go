package trash

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logictrash "tty.fail/mrus/korda/logic/root/trash"
	"tty.fail/mrus/korda/models/permission"
	"xn--gckvb8fzb.com/glides/http/route"
	"xn--gckvb8fzb.com/glides/runtime"
)

type Route struct {
	route.RouteController
}

type count struct {
	Count int64 `json:"count"`
}

func New(rt *runtime.Runtime, router fiber.Router) (*Route, error) {
	r := new(Route)

	r.Runtime = rt
	r.Router = router
	r.Path = route.For("APITrash").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Post("/empty", r.EmptyTrash).Name("emptyTrash")
		base.Post("/restore", r.RestoreTrash).Name("restoreTrash")
		base.Post("/restore/assets", r.RestoreAssets).Name("restoreAssets")
	}, r.Path+".")

	return r, nil
}

func (r *Route) EmptyTrash(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "emptyTrash")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetDelete}); err != nil {
		return err
	}

	n, err := logictrash.Empty(req.Context(), r.Runtime, req.Auth)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, count{n})
}

func (r *Route) RestoreTrash(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "restoreTrash")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetDelete}); err != nil {
		return err
	}

	n, err := logictrash.Restore(req.Context(), r.Runtime, req.Auth)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, count{n})
}

func (r *Route) RestoreAssets(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "restoreAssets")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetDelete}); err != nil {
		return err
	}

	var in logictrash.BulkIDsInput
	if err := req.Body(&in); err != nil {
		return err
	}

	n, err := logictrash.RestoreAssets(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, count{n})
}
