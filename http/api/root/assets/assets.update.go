package assets

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	logicassets "tty.fail/mrus/korda/logic/root/assets"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	"tty.fail/mrus/korda/models/permission"
)

func (r *Route) UpdateAsset(c fiber.Ctx) error {
	return r.updateAsset(c, "updateAsset")
}

func (r *Route) UpdateAssetV3(c fiber.Ctx) error {
	return r.updateAsset(c, "updateAssetV3")
}

func (r *Route) updateAsset(c fiber.Ctx, operation string) error {
	req := request.New(r.Runtime, c, operation)
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetUpdate}); err != nil {
		return err
	}

	var in logicassets.UpdateInput
	if err := req.Body(&in); err != nil {
		return err
	}
	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	d, err := logicassets.Update(req.Context(), r.Runtime, req.Auth, id, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, Detailed(req, d))
}

func (r *Route) UpdateAssets(c fiber.Ctx) error {
	return r.updateAssets(c, "updateAssets")
}

func (r *Route) UpdateAssetsV3(c fiber.Ctx) error {
	return r.updateAssets(c, "updateAssetsV3")
}

func (r *Route) updateAssets(c fiber.Ctx, operation string) error {
	req := request.New(r.Runtime, c, operation)
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetUpdate}); err != nil {
		return err
	}

	var in logicassets.BulkUpdateInput
	if err := req.Body(&in); err != nil {
		return err
	}

	if err := logicassets.UpdateAll(req.Context(), r.Runtime, req.Auth, &in); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) DeleteAssets(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "deleteAssets")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetDelete}); err != nil {
		return err
	}

	var in logicassets.BulkDeleteInput
	if err := req.Body(&in); err != nil {
		return err
	}

	if err := logicassets.DeleteAll(req.Context(), r.Runtime, req.Auth, &in); err != nil {
		return err
	}

	return req.NoContent()
}
