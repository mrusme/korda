package partners

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logicpartners "tty.fail/mrus/korda/logic/root/partners"
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
	r.Path = route.For("APIPartners").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Get("", r.GetPartners).Name("getPartners")
		base.Post("", r.CreatePartner).Name("createPartner")
		base.Delete("/:id", r.RemovePartner).Name("removePartner")
		base.Post("/:id", r.CreatePartnerDeprecated).Name("createPartnerDeprecated")
		base.Put("/:id", r.UpdatePartner).Name("updatePartner")
	}, r.Path+".")

	return r, nil
}

func (r *Route) GetPartners(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getPartners")
	if err := req.Require(logicauth.Requirement{Permission: permission.PartnerRead}); err != nil {
		return err
	}

	var in logicpartners.SearchInput
	if err := req.Query(&in); err != nil {
		return err
	}

	list, err := logicpartners.Search(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewPartners(list))
}

func (r *Route) CreatePartner(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "createPartner")
	if err := req.Require(logicauth.Requirement{Permission: permission.PartnerCreate}); err != nil {
		return err
	}

	var in logicpartners.CreateInput
	if err := req.Body(&in); err != nil {
		return err
	}

	shared, err := logicpartners.Create(req.Context(), r.Runtime, req.Auth, uuid.MustParse(in.SharedWithID))
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusCreated, view.NewPartner(shared))
}

func (r *Route) CreatePartnerDeprecated(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "createPartnerDeprecated")
	if err := req.Require(logicauth.Requirement{Permission: permission.PartnerCreate}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	shared, err := logicpartners.Create(req.Context(), r.Runtime, req.Auth, id)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusCreated, view.NewPartner(shared))
}

func (r *Route) UpdatePartner(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "updatePartner")
	if err := req.Require(logicauth.Requirement{Permission: permission.PartnerUpdate}); err != nil {
		return err
	}

	var in logicpartners.UpdateInput
	if err := req.Body(&in); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	shared, err := logicpartners.Update(req.Context(), r.Runtime, req.Auth, id, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewPartner(shared))
}

func (r *Route) RemovePartner(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "removePartner")
	if err := req.Require(logicauth.Requirement{Permission: permission.PartnerDelete}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	if err = logicpartners.Remove(req.Context(), r.Runtime, req.Auth, id); err != nil {
		return err
	}

	return req.NoContent()
}
