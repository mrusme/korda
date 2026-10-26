package sharedlinks

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logicsharedlinks "tty.fail/mrus/korda/logic/root/sharedlinks"
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
	r.Path = route.For("APISharedLinks").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Get("", r.GetAllSharedLinks).Name("getAllSharedLinks")
		base.Post("", r.CreateSharedLink).Name("createSharedLink")
		base.Post("/login", r.SharedLinkLogin).Name("sharedLinkLogin")
		base.Get("/me", r.GetMySharedLink).Name("getMySharedLink")
		base.Get("/:id", r.GetSharedLinkByID).Name("getSharedLinkById")
		base.Patch("/:id", r.UpdateSharedLink).Name("updateSharedLink")
		base.Delete("/:id", r.RemoveSharedLink).Name("removeSharedLink")
		base.Put("/:id/assets", r.AddSharedLinkAssets).Name("addSharedLinkAssets")
		base.Delete("/:id/assets", r.RemoveSharedLinkAssets).Name("removeSharedLinkAssets")
	}, r.Path+".")

	return r, nil
}

func (r *Route) GetAllSharedLinks(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getAllSharedLinks")
	if err := req.Require(logicauth.Requirement{Permission: permission.SharedLinkRead}); err != nil {
		return err
	}

	var in logicsharedlinks.SearchInput
	if err := req.Query(&in); err != nil {
		return err
	}

	list, err := logicsharedlinks.All(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewSharedLinks(list))
}

func (r *Route) CreateSharedLink(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "createSharedLink")
	if err := req.Require(logicauth.Requirement{Permission: permission.SharedLinkCreate}); err != nil {
		return err
	}

	var in logicsharedlinks.CreateInput
	if err := req.Body(&in); err != nil {
		return err
	}

	d, err := logicsharedlinks.Create(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusCreated, view.NewSharedLink(d))
}

func (r *Route) SharedLinkLogin(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "sharedLinkLogin")
	if err := req.Require(logicauth.Requirement{SharedLink: true, NoSharedLinkPassword: true}); err != nil {
		return err
	}

	var in logicsharedlinks.LoginInput
	if err := req.Body(&in); err != nil {
		return err
	}

	d, token, err := logicsharedlinks.Login(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}
	req.AddSharedLinkToken(token)

	return req.Respond(fiber.StatusCreated, view.NewSharedLink(d))
}

func (r *Route) GetMySharedLink(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getMySharedLink")
	if err := req.Require(logicauth.Requirement{SharedLink: true}); err != nil {
		return err
	}

	d, err := logicsharedlinks.Mine(req.Context(), r.Runtime, req.Auth)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewSharedLink(d))
}

func (r *Route) GetSharedLinkByID(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getSharedLinkById")
	if err := req.Require(logicauth.Requirement{Permission: permission.SharedLinkRead}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	d, err := logicsharedlinks.Get(req.Context(), r.Runtime, req.Auth, id)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewSharedLink(d))
}

func (r *Route) UpdateSharedLink(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "updateSharedLink")
	if err := req.Require(logicauth.Requirement{Permission: permission.SharedLinkUpdate}); err != nil {
		return err
	}

	var in logicsharedlinks.UpdateInput
	if err := req.Body(&in); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	d, err := logicsharedlinks.Update(req.Context(), r.Runtime, req.Auth, id, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewSharedLink(d))
}

func (r *Route) RemoveSharedLink(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "removeSharedLink")
	if err := req.Require(logicauth.Requirement{Permission: permission.SharedLinkDelete}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	if err = logicsharedlinks.Remove(req.Context(), r.Runtime, req.Auth, id); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) assets(c fiber.Ctx, operation string, apply func(*request.Request, *logicsharedlinks.AssetIDsInput) ([]logicsharedlinks.Result, error)) error {
	req := request.New(r.Runtime, c, operation)
	if err := req.Require(logicauth.Requirement{Permission: permission.SharedLinkUpdate}); err != nil {
		return err
	}

	var in logicsharedlinks.AssetIDsInput
	if err := req.Body(&in); err != nil {
		return err
	}

	results, err := apply(req, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewAssetIDResults(results))
}

func (r *Route) AddSharedLinkAssets(c fiber.Ctx) error {
	return r.assets(c, "addSharedLinkAssets", func(req *request.Request, in *logicsharedlinks.AssetIDsInput) ([]logicsharedlinks.Result, error) {
		id, err := req.UUID("id")
		if err != nil {
			return nil, err
		}

		return logicsharedlinks.AddAssets(req.Context(), r.Runtime, req.Auth, id, in)
	})
}

func (r *Route) RemoveSharedLinkAssets(c fiber.Ctx) error {
	return r.assets(c, "removeSharedLinkAssets", func(req *request.Request, in *logicsharedlinks.AssetIDsInput) ([]logicsharedlinks.Result, error) {
		id, err := req.UUID("id")
		if err != nil {
			return nil, err
		}

		return logicsharedlinks.RemoveAssets(req.Context(), r.Runtime, req.Auth, id, in)
	})
}
