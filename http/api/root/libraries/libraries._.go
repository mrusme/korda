package libraries

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/errs"
	"tty.fail/mrus/korda/http/api/problem"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logiclibraries "tty.fail/mrus/korda/logic/root/libraries"
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
	r.Path = route.For("APILibraries").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Get("", r.GetAllLibraries).Name("getAllLibraries")
		base.Post("", r.CreateLibrary).Name("createLibrary")
		base.Get("/:id", r.GetLibrary).Name("getLibrary")
		base.Put("/:id", r.UpdateLibrary).Name("updateLibrary")
		base.Patch("/:id", r.UpdateLibraryV3).Name("updateLibraryV3")
		base.Delete("/:id", r.DeleteLibrary).Name("deleteLibrary")
		base.Post("/:id/validate", r.Validate).Name("validate")
		base.Get("/:id/statistics", r.GetLibraryStatistics).Name("getLibraryStatistics")
		base.Post("/:id/scan", r.ScanLibrary).Name("scanLibrary")
	}, r.Path+".")

	return r, nil
}

func (r *Route) start(c fiber.Ctx, operation string, p permission.Permission) (*request.Request, error) {
	req := request.New(r.Runtime, c, operation)
	return req, req.Require(logicauth.Requirement{Permission: p, Admin: true})
}

func (r *Route) GetAllLibraries(c fiber.Ctx) error {
	req, err := r.start(c, "getAllLibraries", permission.LibraryRead)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, []struct{}{})
}

func (r *Route) CreateLibrary(c fiber.Ctx) error {
	req, err := r.start(c, "createLibrary", permission.LibraryCreate)
	if err != nil {
		return err
	}

	var in logiclibraries.CreateInput
	if err = req.Body(&in); err != nil {
		return err
	}

	return logiclibraries.Create(&in)
}

func (r *Route) missing(c fiber.Ctx, operation string, p permission.Permission, body any) error {
	req, err := r.start(c, operation, p)
	if err != nil {
		return err
	}
	if body != nil {
		if err = req.Body(body); err != nil {
			return err
		}
	}
	if _, err = req.UUID("id"); err != nil {
		return err
	}

	return errs.ErrLibraryNotFound
}

func (r *Route) GetLibrary(c fiber.Ctx) error {
	return r.missing(c, "getLibrary", permission.LibraryRead, nil)
}

func (r *Route) UpdateLibrary(c fiber.Ctx) error {
	return r.missing(c, "updateLibrary", permission.LibraryUpdate, new(logiclibraries.UpdateInput))
}

func (r *Route) UpdateLibraryV3(c fiber.Ctx) error {
	return r.missing(c, "updateLibraryV3", permission.LibraryUpdate, new(logiclibraries.UpdateInput))
}

func (r *Route) DeleteLibrary(c fiber.Ctx) error {
	return r.missing(c, "deleteLibrary", permission.LibraryDelete, nil)
}

func (r *Route) ScanLibrary(c fiber.Ctx) error {
	return r.missing(c, "scanLibrary", permission.LibraryUpdate, nil)
}

func (r *Route) GetLibraryStatistics(c fiber.Ctx) error {
	req, err := r.start(c, "getLibraryStatistics", permission.LibraryStatistics)
	if err != nil {
		return err
	}
	if _, err = req.UUID("id"); err != nil {
		return err
	}

	return &problem.Error{Status: fiber.StatusBadRequest, Message: "Library " + c.Params("id") + " not found"}
}

func (r *Route) Validate(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "validate")
	if err := req.Require(logicauth.Requirement{Admin: true}); err != nil {
		return err
	}

	var in logiclibraries.ValidateInput
	if err := req.Body(&in); err != nil {
		return err
	}
	if _, err := req.UUID("id"); err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewLibraryValidation(logiclibraries.Validate(r.Runtime, &in)))
}
