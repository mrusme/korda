package people

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/errs"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
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
	r.Path = route.For("APIPeople").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Get("", r.GetAllPeople).Name("getAllPeople")
		base.Post("", r.CreatePerson).Name("createPerson")
		base.Put("", r.UpdatePeople).Name("updatePeople")
		base.Delete("", r.DeletePeople).Name("deletePeople")
		base.Post("/merge", r.MergePeople).Name("mergePeople")
		base.Get("/users", r.GetUsersForPeople).Name("getUsersForPeople")
		base.Put("/users", r.UpsertPeopleUsers).Name("upsertPeopleUsers")
		base.Delete("/users", r.RemoveUsersFromPeople).Name("removeUsersFromPeople")
		base.Get("/:id", r.GetPerson).Name("getPerson")
		base.Put("/:id", r.UpdatePerson).Name("updatePerson")
		base.Patch("/:id", r.UpdatePersonV3).Name("updatePersonV3")
		base.Delete("/:id", r.DeletePerson).Name("deletePerson")
		base.Post("/:id/merge", r.MergePersonLegacy).Name("mergePersonLegacy")
		base.Put("/:id/reassign", r.ReassignFaces).Name("reassignFaces")
		base.Get("/:id/statistics", r.GetPersonStatistics).Name("getPersonStatistics")
		base.Get("/:id/thumbnail", r.GetPersonThumbnail).Name("getPersonThumbnail")
	}, r.Path+".")

	return r, nil
}

func (r *Route) start(c fiber.Ctx, operation string, p permission.Permission) (*request.Request, error) {
	req := request.New(r.Runtime, c, operation)
	return req, req.Require(logicauth.Requirement{Permission: p})
}

func (r *Route) GetAllPeople(c fiber.Ctx) error {
	req, err := r.start(c, "getAllPeople", permission.PersonRead)
	if err != nil {
		return err
	}

	var in logicpeople.SearchInput
	if err = req.Query(&in); err != nil {
		return err
	}
	if err = logicpeople.Search(&in); err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewPeople())
}

func (r *Route) CreatePerson(c fiber.Ctx) error {
	req, err := r.start(c, "createPerson", permission.PersonCreate)
	if err != nil {
		return err
	}

	var in logicpeople.CreateInput
	if err = req.Body(&in); err != nil {
		return err
	}

	return logicpeople.Create(&in)
}

func (r *Route) UpdatePeople(c fiber.Ctx) error {
	req, err := r.start(c, "updatePeople", permission.PersonUpdate)
	if err != nil {
		return err
	}

	var in logicpeople.UpdateAllInput
	if err = req.Body(&in); err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewPersonResults(logicpeople.UpdateAll(req.Context(), r.Runtime, req.Auth, &in)))
}

func (r *Route) DeletePeople(c fiber.Ctx) error {
	req, err := r.start(c, "deletePeople", permission.PersonDelete)
	if err != nil {
		return err
	}

	var in logicpeople.DeleteAllInput
	if err = req.Body(&in); err != nil {
		return err
	}
	if err = logicpeople.DeleteAll(req.Context(), r.Runtime, req.Auth, in.IDs, in.UserID); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) MergePeople(c fiber.Ctx) error {
	req, err := r.start(c, "mergePeople", permission.PersonMerge)
	if err != nil {
		return err
	}

	var in logicpeople.MergeInput
	if err = req.Body(&in); err != nil {
		return err
	}

	results, err := logicpeople.Merge(in.IDs)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewPersonResults(results))
}

func (r *Route) GetUsersForPeople(c fiber.Ctx) error {
	req, err := r.start(c, "getUsersForPeople", permission.PersonRead)
	if err != nil {
		return err
	}

	var in logicpeople.UsersSearchInput
	if err = req.Query(&in); err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, []struct{}{})
}

func (r *Route) UpsertPeopleUsers(c fiber.Ctx) error {
	req, err := r.start(c, "upsertPeopleUsers", permission.PersonUpdate)
	if err != nil {
		return err
	}

	var in logicpeople.UsersUpsertInput
	if err = req.Body(&in); err != nil {
		return err
	}
	if err = logicpeople.UpsertUsers(req.Context(), r.Runtime, req.Auth, &in); err != nil {
		return err
	}

	return req.Empty(fiber.StatusOK)
}

func (r *Route) RemoveUsersFromPeople(c fiber.Ctx) error {
	req, err := r.start(c, "removeUsersFromPeople", permission.PersonUpdate)
	if err != nil {
		return err
	}

	var in logicpeople.UsersDeleteInput
	if err = req.Body(&in); err != nil {
		return err
	}
	if err = logicpeople.RemoveUsers(req.Context(), r.Runtime, req.Auth, in); err != nil {
		return err
	}

	return req.Empty(fiber.StatusOK)
}

func (r *Route) id(req *request.Request, c fiber.Ctx) (string, error) {
	if _, err := req.UUID("id"); err != nil {
		return "", err
	}

	return c.Params("id"), nil
}

func (r *Route) GetPerson(c fiber.Ctx) error {
	return r.read(c, "getPerson", permission.PersonRead)
}

func (r *Route) GetPersonStatistics(c fiber.Ctx) error {
	return r.read(c, "getPersonStatistics", permission.PersonStatistics)
}

func (r *Route) read(c fiber.Ctx, operation string, p permission.Permission) error {
	req, err := r.start(c, operation, p)
	if err != nil {
		return err
	}

	id, err := r.id(req, c)
	if err != nil {
		return err
	}

	return logicpeople.Read(req.Context(), r.Runtime, req.Auth, id)
}

func (r *Route) GetPersonThumbnail(c fiber.Ctx) error {
	req, err := r.start(c, "getPersonThumbnail", permission.PersonRead)
	if err != nil {
		return err
	}

	if _, err = r.id(req, c); err != nil {
		return err
	}

	return errs.ErrNotFound
}

func (r *Route) UpdatePerson(c fiber.Ctx) error {
	return r.update(c, "updatePerson")
}

func (r *Route) UpdatePersonV3(c fiber.Ctx) error {
	return r.update(c, "updatePersonV3")
}

func (r *Route) update(c fiber.Ctx, operation string) error {
	req, err := r.start(c, operation, permission.PersonUpdate)
	if err != nil {
		return err
	}

	var in logicpeople.UpdateInput
	if err = req.Body(&in); err != nil {
		return err
	}
	id, err := r.id(req, c)
	if err != nil {
		return err
	}

	return logicpeople.Update(req.Context(), r.Runtime, req.Auth, id, &in)
}

func (r *Route) DeletePerson(c fiber.Ctx) error {
	req, err := r.start(c, "deletePerson", permission.PersonDelete)
	if err != nil {
		return err
	}

	var in logicpeople.DeleteInput
	if err = req.BodyOrEmpty(&in); err != nil {
		return err
	}
	id, err := r.id(req, c)
	if err != nil {
		return err
	}
	if err = logicpeople.DeleteAll(req.Context(), r.Runtime, req.Auth, []string{id}, in.UserID); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) MergePersonLegacy(c fiber.Ctx) error {
	req, err := r.start(c, "mergePersonLegacy", permission.PersonMerge)
	if err != nil {
		return err
	}

	var in logicpeople.MergeInput
	if err = req.Body(&in); err != nil {
		return err
	}
	id, err := r.id(req, c)
	if err != nil {
		return err
	}

	results, err := logicpeople.Merge(append([]string{id}, in.IDs...))
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewPersonResults(results))
}

func (r *Route) ReassignFaces(c fiber.Ctx) error {
	req, err := r.start(c, "reassignFaces", permission.PersonReassign)
	if err != nil {
		return err
	}

	var in logicpeople.ReassignInput
	if err = req.Body(&in); err != nil {
		return err
	}
	id, err := r.id(req, c)
	if err != nil {
		return err
	}

	return logicpeople.Reassign(req.Context(), r.Runtime, req.Auth, id, &in)
}
