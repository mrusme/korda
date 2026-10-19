package assets

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicassets "tty.fail/mrus/korda/logic/root/assets"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	"tty.fail/mrus/korda/models/permission"
)

func (r *Route) GetAssetMetadata(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getAssetMetadata")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetRead}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	list, err := logicassets.ListMetadata(req.Context(), r.Runtime, req.Auth, id)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewAssetMetadataList(list))
}

func (r *Route) UpdateAssetMetadata(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "updateAssetMetadata")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetUpdate}); err != nil {
		return err
	}

	var in logicassets.MetadataUpsertInput
	if err := req.Body(&in); err != nil {
		return err
	}
	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	list, err := logicassets.UpsertMetadata(req.Context(), r.Runtime, req.Auth, id, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewAssetMetadataList(list))
}

func (r *Route) metadataRoute(req *request.Request) (uuid.UUID, string, error) {
	var in logicassets.MetadataRouteInput
	if err := req.Params(&in, "id", "key"); err != nil {
		return uuid.Nil, "", err
	}

	return uuid.MustParse(in.ID), in.Key, nil
}

func (r *Route) GetAssetMetadataByKey(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getAssetMetadataByKey")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetRead}); err != nil {
		return err
	}

	id, key, err := r.metadataRoute(req)
	if err != nil {
		return err
	}

	item, err := logicassets.GetMetadata(req.Context(), r.Runtime, req.Auth, id, key)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewAssetMetadata(item))
}

func (r *Route) DeleteAssetMetadata(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "deleteAssetMetadata")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetUpdate}); err != nil {
		return err
	}

	id, key, err := r.metadataRoute(req)
	if err != nil {
		return err
	}

	if err = logicassets.DeleteMetadata(req.Context(), r.Runtime, req.Auth, id, key); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) UpdateBulkAssetMetadata(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "updateBulkAssetMetadata")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetUpdate}); err != nil {
		return err
	}

	var in logicassets.MetadataBulkUpsertInput
	if err := req.Body(&in); err != nil {
		return err
	}

	list, err := logicassets.UpsertBulkMetadata(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewAssetMetadataBulkList(list))
}

func (r *Route) DeleteBulkAssetMetadata(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "deleteBulkAssetMetadata")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetUpdate}); err != nil {
		return err
	}

	var in logicassets.MetadataBulkDeleteInput
	if err := req.Body(&in); err != nil {
		return err
	}

	if err := logicassets.DeleteBulkMetadata(req.Context(), r.Runtime, req.Auth, &in); err != nil {
		return err
	}

	return req.NoContent()
}
