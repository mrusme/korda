package apikeys

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicapikeys "tty.fail/mrus/korda/logic/root/apikeys"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	"tty.fail/mrus/korda/models/permission"
)

func (r *Route) CreateApiKey(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "createApiKey")
	if err := req.Require(logicauth.Requirement{Permission: permission.APIKeyCreate}); err != nil {
		return err
	}

	var in logicapikeys.CreateInput
	if err := req.Body(&in); err != nil {
		return err
	}

	k, secret, err := logicapikeys.Create(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusCreated, view.NewAPIKeyCreate(k, secret))
}

func (r *Route) GetApiKeys(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getApiKeys")
	if err := req.Require(logicauth.Requirement{Permission: permission.APIKeyRead}); err != nil {
		return err
	}

	list, err := logicapikeys.List(req.Context(), r.Runtime, req.Auth)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewAPIKeys(list))
}

func (r *Route) GetMyApiKey(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getMyApiKey")
	if err := req.Require(logicauth.Requirement{NoPermission: true}); err != nil {
		return err
	}

	k, err := logicapikeys.Mine(req.Context(), r.Runtime, req.Auth)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewAPIKey(k))
}

func (r *Route) GetApiKey(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getApiKey")
	if err := req.Require(logicauth.Requirement{Permission: permission.APIKeyRead}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	k, err := logicapikeys.Get(req.Context(), r.Runtime, req.Auth, id)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewAPIKey(k))
}

func (r *Route) UpdateApiKey(c fiber.Ctx) error {
	return r.update(c, "updateApiKey")
}

func (r *Route) UpdateApiKeyV3(c fiber.Ctx) error {
	return r.update(c, "updateApiKeyV3")
}

func (r *Route) update(c fiber.Ctx, operation string) error {
	req := request.New(r.Runtime, c, operation)
	if err := req.Require(logicauth.Requirement{Permission: permission.APIKeyUpdate}); err != nil {
		return err
	}

	var in logicapikeys.UpdateInput
	if err := req.Body(&in); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	k, err := logicapikeys.Update(req.Context(), r.Runtime, req.Auth, id, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewAPIKey(k))
}

func (r *Route) RotateApiKey(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "rotateApiKey")
	if err := req.Require(logicauth.Requirement{Permission: permission.APIKeyRotate}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	k, secret, err := logicapikeys.Rotate(req.Context(), r.Runtime, req.Auth, id)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusCreated, view.NewAPIKeyCreate(k, secret))
}

func (r *Route) DeleteApiKey(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "deleteApiKey")
	if err := req.Require(logicauth.Requirement{Permission: permission.APIKeyDelete}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	if err = logicapikeys.Delete(req.Context(), r.Runtime, req.Auth, id); err != nil {
		return err
	}

	return req.NoContent()
}
