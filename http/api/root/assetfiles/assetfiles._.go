package assetfiles

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/errs"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicassetfiles "tty.fail/mrus/korda/logic/root/assetfiles"
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
	r.Path = route.For("APIAssetFiles").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Get("", r.SearchAssetFiles).Name("searchAssetFiles")
		base.Get("/:id", r.GetAssetFile).Name("getAssetFile")
		base.Delete("/:id", r.DeleteAssetFile).Name("deleteAssetFile")
		base.Get("/:id/download", r.DownloadAssetFile).Name("downloadAssetFile")
	}, r.Path+".")

	return r, nil
}

func (r *Route) SearchAssetFiles(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "searchAssetFiles")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetFileRead}); err != nil {
		return err
	}

	var in logicassetfiles.SearchInput
	if err := req.Query(&in); err != nil {
		return err
	}

	list, err := logicassetfiles.Search(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewAssetFiles(list))
}

func (r *Route) GetAssetFile(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getAssetFile")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetFileRead}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	f, err := logicassetfiles.Get(req.Context(), r.Runtime, req.Auth, id)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewAssetFile(f))
}

func (r *Route) DeleteAssetFile(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "deleteAssetFile")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetFileDelete}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	if err = logicassetfiles.Delete(req.Context(), r.Runtime, req.Auth, id); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) DownloadAssetFile(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "downloadAssetFile")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetFileDownload}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	d, err := logicassetfiles.Prepare(req.Context(), r.Runtime, req.Auth, id)
	if err != nil {
		return errs.ErrNotFound
	}

	return req.SendMedia(d.Abs, d.ContentType, d.FileName)
}
