package assets

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/errs"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	"tty.fail/mrus/korda/logic/helpers/access"
	logicassets "tty.fail/mrus/korda/logic/root/assets"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	"tty.fail/mrus/korda/models/permission"
)

const (
	hlsPlaylistType = "application/vnd.apple.mpegurl"
	headerHlsPos    = "x-immich-hls-pos"
	headerHlsMsn    = "x-immich-hls-msn"
)

type hlsSession struct {
	ID        string `json:"id" v:"uuid"`
	SessionID string `json:"sessionId" v:"uuid"`
}

type hlsVariant struct {
	hlsSession
	VariantIndex int64 `json:"variantIndex" v:"coerce,min=0"`
}

type hlsSegment struct {
	hlsVariant
	Filename string `json:"filename" v:"regex=hlsSegment"`
}

type hlsPlaylistHeaders struct {
	Position *float64 `json:"x-immich-hls-pos" v:"coerce,min=0"`
}

type hlsSegmentHeaders struct {
	Segment *int64 `json:"x-immich-hls-msn" v:"coerce,min=0"`
}

func (r *Route) RunAssetJobs(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "runAssetJobs")
	if err := req.Require(logicauth.Requirement{Permission: permission.JobCreate}); err != nil {
		return err
	}

	var in logicassets.JobsInput
	if err := req.Body(&in); err != nil {
		return err
	}

	if err := logicassets.RunJobs(req.Context(), r.Runtime, req.Auth, &in); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) CopyAsset(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "copyAsset")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetCopy}); err != nil {
		return err
	}

	var in logicassets.CopyInput
	if err := req.Body(&in); err != nil {
		return err
	}

	if err := logicassets.Copy(req.Context(), r.Runtime, req.Auth, &in); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) GetAssetOcr(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getAssetOcr")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetRead}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	if err = access.Require(req.Context(), r.Runtime, req.Auth, permission.AssetRead, id); err != nil {
		return err
	}
	exif, err := logicassets.Exif(req.Context(), r.Runtime, id)
	if err != nil {
		return err
	}
	if exif == nil {
		return errs.ErrAssetNotFound
	}

	return req.Respond(fiber.StatusOK, []view.AssetOcr{})
}

func (r *Route) GetAssetEdits(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getAssetEdits")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetEditGet}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	if err = access.Require(req.Context(), r.Runtime, req.Auth, permission.AssetRead, id); err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewAssetEdits(id))
}

func (r *Route) GetMainPlaylist(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getMainPlaylist")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetView}); err != nil {
		return err
	}
	c.Set(fiber.HeaderCacheControl, "no-cache")

	id, err := req.UUID("id")
	if err != nil {
		return err
	}
	if err = access.Require(req.Context(), r.Runtime, req.Auth, permission.AssetView, id); err != nil {
		return err
	}

	return errs.ErrRealtimeTranscoding
}

func (r *Route) GetMediaPlaylist(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getMediaPlaylist")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetView}); err != nil {
		return err
	}
	c.Set(fiber.HeaderCacheControl, "no-cache")

	var params hlsVariant
	if err := req.Params(&params, "id", "sessionId", "variantIndex"); err != nil {
		return err
	}
	var headers hlsPlaylistHeaders
	if err := req.Headers(&headers, headerHlsPos); err != nil {
		return err
	}
	if err := access.Require(req.Context(), r.Runtime, req.Auth, permission.AssetView, parseID(params.ID)); err != nil {
		return err
	}

	return errs.ErrStreamNotReady
}

func (r *Route) GetSegment(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getSegment")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetView}); err != nil {
		return err
	}

	var params hlsSegment
	if err := req.Params(&params, "id", "sessionId", "variantIndex", "filename"); err != nil {
		return err
	}
	var headers hlsSegmentHeaders
	if err := req.Headers(&headers, headerHlsMsn); err != nil {
		return err
	}

	return errs.ErrNotFound
}

func (r *Route) EndSession(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "endSession")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetView}); err != nil {
		return err
	}

	var params hlsSession
	if err := req.Params(&params, "id", "sessionId"); err != nil {
		return err
	}
	if err := access.Require(req.Context(), r.Runtime, req.Auth, permission.AssetView, parseID(params.ID)); err != nil {
		return err
	}

	return req.NoContent()
}
