package assets

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicassets "tty.fail/mrus/korda/logic/root/assets"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	"tty.fail/mrus/korda/models/permission"
)

func Detailed(req *request.Request, d *logicassets.Details) view.Asset {
	return view.NewAsset(d.Asset, req.Auth.User.ID, view.AssetDetails{
		Owner:     d.Owner,
		Exif:      d.Exif,
		Tags:      true,
		TagList:   d.Tags,
		WithStack: true,
		Stack:     d.Stack,
	})
}

func (r *Route) GetAssetInfo(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getAssetInfo")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetRead, SharedLink: true}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	d, err := logicassets.Get(req.Context(), r.Runtime, req.Auth, id)
	if err != nil {
		return err
	}

	if link := req.Auth.SharedLink; link != nil {
		if !link.ShowExif {
			return req.Respond(fiber.StatusOK, view.NewStrippedAsset(d.Asset, false))
		}
		out := Detailed(req, d)
		out.Owner = nil
		return req.Respond(fiber.StatusOK, out)
	}

	return req.Respond(fiber.StatusOK, Detailed(req, d))
}

func (r *Route) GetAssetStatistics(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getAssetStatistics")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetStatistics}); err != nil {
		return err
	}

	var in logicassets.StatisticsInput
	if err := req.Query(&in); err != nil {
		return err
	}

	stats, err := logicassets.Statistics(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.AssetStatistics{
		Images: stats.Image,
		Videos: stats.Video,
		Total:  stats.Audio + stats.Image + stats.Video + stats.Other,
	})
}
