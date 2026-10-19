package sessions

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logicsessions "tty.fail/mrus/korda/logic/root/sessions"
	"tty.fail/mrus/korda/models/permission"
)

func current(req *request.Request) uuid.UUID {
	if req.Auth == nil || req.Auth.Session == nil {
		return uuid.Nil
	}

	return req.Auth.Session.ID
}

func (r *Route) CreateSession(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "createSession")
	if err := req.Require(logicauth.Requirement{Permission: permission.SessionCreate}); err != nil {
		return err
	}

	var in logicsessions.CreateInput
	if err := req.Body(&in); err != nil {
		return err
	}

	s, token, err := logicsessions.Create(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusCreated, view.NewSessionCreate(s, token))
}

func (r *Route) GetSessions(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getSessions")
	if err := req.Require(logicauth.Requirement{Permission: permission.SessionRead}); err != nil {
		return err
	}

	list, err := logicsessions.List(req.Context(), r.Runtime, req.Auth.User.ID)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewSessions(list, current(req)))
}

func (r *Route) DeleteAllSessions(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "deleteAllSessions")
	if err := req.Require(logicauth.Requirement{Permission: permission.SessionDelete}); err != nil {
		return err
	}

	if err := logicsessions.DeleteAll(req.Context(), r.Runtime, req.Auth); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) UpdateSession(c fiber.Ctx) error {
	return r.update(c, "updateSession")
}

func (r *Route) UpdateSessionV3(c fiber.Ctx) error {
	return r.update(c, "updateSessionV3")
}

func (r *Route) update(c fiber.Ctx, operation string) error {
	req := request.New(r.Runtime, c, operation)
	if err := req.Require(logicauth.Requirement{Permission: permission.SessionUpdate}); err != nil {
		return err
	}

	var in logicsessions.UpdateInput
	if err := req.Body(&in); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	s, err := logicsessions.Update(req.Context(), r.Runtime, req.Auth, id, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewSession(s, uuid.Nil))
}

func (r *Route) DeleteSession(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "deleteSession")
	if err := req.Require(logicauth.Requirement{Permission: permission.SessionDelete}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	if err = logicsessions.Delete(req.Context(), r.Runtime, req.Auth, id); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) LockSession(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "lockSession")
	if err := req.Require(logicauth.Requirement{Permission: permission.SessionLock}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	if err = logicsessions.Lock(req.Context(), r.Runtime, req.Auth, id); err != nil {
		return err
	}

	return req.NoContent()
}
