package faces

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logicpeople "tty.fail/mrus/korda/logic/root/people"
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
	r.Path = route.For("APIFaces").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Post("", r.CreateFace).Name("createFace")
		base.Get("", r.GetFaces).Name("getFaces")
		base.Put("/:id", r.ReassignFacesByID).Name("reassignFacesById")
		base.Delete("/:id", r.DeleteFace).Name("deleteFace")
	}, r.Path+".")

	return r, nil
}

func (r *Route) start(c fiber.Ctx, operation string, p permission.Permission) (*request.Request, error) {
	req := request.New(r.Runtime, c, operation)
	return req, req.Require(logicauth.Requirement{Permission: p})
}

func (r *Route) CreateFace(c fiber.Ctx) error {
	req, err := r.start(c, "createFace", permission.FaceCreate)
	if err != nil {
		return err
	}

	var in logicpeople.FaceCreateInput
	if err = req.Body(&in); err != nil {
		return err
	}

	return logicpeople.CreateFace(req.Context(), r.Runtime, req.Auth, &in)
}

func (r *Route) GetFaces(c fiber.Ctx) error {
	req, err := r.start(c, "getFaces", permission.FaceRead)
	if err != nil {
		return err
	}

	var in logicpeople.FaceInput
	if err = req.Query(&in); err != nil {
		return err
	}
	if err = logicpeople.Faces(req.Context(), r.Runtime, req.Auth, &in); err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, []struct{}{})
}

func (r *Route) ReassignFacesByID(c fiber.Ctx) error {
	req, err := r.start(c, "reassignFacesById", permission.FaceUpdate)
	if err != nil {
		return err
	}

	var in logicpeople.FaceInput
	if err = req.Body(&in); err != nil {
		return err
	}
	if _, err = req.UUID("id"); err != nil {
		return err
	}

	return logicpeople.Reassign(req.Context(), r.Runtime, req.Auth, c.Params("id"), nil)
}

func (r *Route) DeleteFace(c fiber.Ctx) error {
	req, err := r.start(c, "deleteFace", permission.FaceDelete)
	if err != nil {
		return err
	}

	var in logicpeople.FaceDeleteInput
	if err = req.Body(&in); err != nil {
		return err
	}
	if _, err = req.UUID("id"); err != nil {
		return err
	}

	return logicpeople.DeleteFace(req.Context(), r.Runtime, req.Auth, c.Params("id"))
}
