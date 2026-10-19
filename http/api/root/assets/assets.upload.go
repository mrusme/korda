package assets

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/errs"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicassets "tty.fail/mrus/korda/logic/root/assets"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	"tty.fail/mrus/korda/models/permission"
)

const HeaderChecksum = "x-immich-checksum"

func (r *Route) UploadAsset(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "uploadAsset")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetUpload}); err != nil {
		return err
	}

	id, duplicate, err := logicassets.DuplicateByChecksum(req.Context(), r.Runtime, req.Auth, c.Get(HeaderChecksum))
	if err != nil {
		return err
	}
	if duplicate {
		req.Drain()
		return req.Respond(fiber.StatusOK, view.NewAssetDuplicate(id))
	}

	fields := request.NewObject()
	var file, sidecar *logicassets.StoredFile
	discard := func() { logicassets.Discard(r.Runtime, file, sidecar) }

	err = req.Parts(func(p request.Part) error {
		if p.Filename == "" {
			value, err := p.Value()
			if err != nil {
				return err
			}
			request.AppendField(fields, p.Field, value)
			return nil
		}

		if (p.Field == logicassets.FieldAssetData && file != nil) ||
			(p.Field == logicassets.FieldSidecarData && sidecar != nil) {
			return fmt.Errorf("%w - %s", errs.ErrUnexpectedFileField, p.Field)
		}

		name := p.Filename
		if value, ok := fields.Get("filename"); ok {
			if s, ok := value.(string); ok && s != "" {
				name = s
			}
		}
		if err := logicassets.AcceptFile(p.Field, name); err != nil {
			return err
		}

		stored, err := logicassets.Store(r.Runtime, req.Auth.User.ID, p.Field, name, p.Filename, p.Reader)
		if err != nil {
			return err
		}
		if p.Field == logicassets.FieldAssetData {
			file = &stored
		} else {
			sidecar = &stored
		}
		return nil
	})
	if err == request.ErrNotMultipart {
		return request.Validation(request.MissingBody())
	}
	if err != nil {
		discard()
		return err
	}

	var in logicassets.UploadInput
	if err = request.Bind(fields, &in); err != nil {
		discard()
		return err
	}
	if file == nil {
		discard()
		if sidecar != nil {
			return errs.ErrAssetDataEmpty
		}
		return errs.ErrFileRequired
	}

	res, err := logicassets.Upload(req.Context(), r.Runtime, req.Auth, &in, *file, sidecar)
	if err != nil {
		return err
	}
	if res.Duplicate {
		return req.Respond(fiber.StatusOK, view.NewAssetDuplicate(res.ID))
	}

	return req.Respond(fiber.StatusCreated, view.NewAssetUploaded(res.ID))
}

func (r *Route) CheckBulkUpload(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "checkBulkUpload")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetUpload}); err != nil {
		return err
	}

	var in logicassets.BulkUploadCheckInput
	if err := req.Body(&in); err != nil {
		return err
	}

	results, err := logicassets.CheckBulkUpload(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewBulkUploadCheck(results))
}
