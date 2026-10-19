package view

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	apiview "tty.fail/mrus/korda/http/api/view"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logicview "tty.fail/mrus/korda/logic/root/view"
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
	r.Path = route.For("APIView").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Get("/folder/unique-paths", r.GetUniqueOriginalPaths).Name("getUniqueOriginalPaths")
		base.Get("/folder", r.GetAssetsByOriginalPath).Name("getAssetsByOriginalPath")
	}, r.Path+".")

	return r, nil
}

func (r *Route) GetUniqueOriginalPaths(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getUniqueOriginalPaths")
	if err := req.Require(logicauth.Requirement{Permission: permission.FolderRead}); err != nil {
		return err
	}

	paths, err := logicview.UniquePaths(req.Context(), r.Runtime, req.Auth)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, paths)
}

func (r *Route) GetAssetsByOriginalPath(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getAssetsByOriginalPath")
	if err := req.Require(logicauth.Requirement{Permission: permission.FolderRead}); err != nil {
		return err
	}

	var paths []string
	for key, value := range c.RequestCtx().QueryArgs().All() {
		if string(key) == "path" {
			paths = append(paths, string(value))
		}
	}

	list, err := logicview.ByPath(req.Context(), r.Runtime, req.Auth, paths)
	if err != nil {
		return err
	}

	out, err := apiview.NewAssetsWithExif(req.Context(), r.Runtime, list, req.Auth.User.ID)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, out)
}
