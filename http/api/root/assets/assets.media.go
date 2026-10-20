package assets

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	logicassets "tty.fail/mrus/korda/logic/root/assets"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	"tty.fail/mrus/korda/models/permission"
)

func (r *Route) ViewAsset(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "viewAsset")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetView, SharedLink: true}); err != nil {
		return err
	}

	var in logicassets.MediaOptionsInput
	if err := req.Query(&in); err != nil {
		return err
	}
	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	if in.Size != nil && *in.Size == logicassets.SizeOriginal {
		return req.Redirect("original?" + req.QueryWithout("size"))
	}

	media, err := logicassets.Thumbnail(req.Context(), r.Runtime, req.Auth, id, &in)
	if err != nil {
		return err
	}

	switch media.Redirect {
	case logicassets.SizeOriginal:
		return req.Redirect("original?" + req.QueryWithout("size"))
	case logicassets.SizePreview:
		path, _, _ := strings.Cut(string(c.RequestCtx().RequestURI()), "?")
		return req.Redirect(path + "?" + req.QueryWith("size", string(logicassets.SizePreview)))
	}

	return req.SendMedia(media.Path, media.ContentType, media.FileName)
}

func (r *Route) DownloadAsset(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "downloadAsset")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetDownload, SharedLink: true}); err != nil {
		return err
	}

	var in logicassets.DownloadOriginalInput
	if err := req.Query(&in); err != nil {
		return err
	}
	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	media, err := logicassets.Original(req.Context(), r.Runtime, req.Auth, id, &in)
	if err != nil {
		return err
	}

	return req.SendMedia(media.Path, media.ContentType, media.FileName)
}

func (r *Route) PlayAssetVideo(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "playAssetVideo")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetView, SharedLink: true}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	media, err := logicassets.Playback(req.Context(), r.Runtime, req.Auth, id)
	if err != nil {
		return err
	}

	return req.SendMedia(media.Path, media.ContentType, "")
}
