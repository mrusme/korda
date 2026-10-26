package download

import (
	"io"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/helpers/ziparchive"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logicdownload "tty.fail/mrus/korda/logic/root/download"
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
	r.Path = route.For("APIDownload").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Post("/info", r.GetDownloadInfo).Name("getDownloadInfo")
		base.Post("/archive", r.DownloadArchive).Name("downloadArchive")
	}, r.Path+".")

	return r, nil
}

func (r *Route) GetDownloadInfo(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getDownloadInfo")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetDownload, SharedLink: true}); err != nil {
		return err
	}

	var in logicdownload.InfoInput
	if err := req.Body(&in); err != nil {
		return err
	}

	info, err := logicdownload.GetInfo(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusCreated, view.NewDownloadInfo(info))
}

func (r *Route) DownloadArchive(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "downloadArchive")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetDownload, SharedLink: true}); err != nil {
		return err
	}

	var in logicdownload.ArchiveInput
	if err := req.Body(&in); err != nil {
		return err
	}

	entries, err := logicdownload.Entries(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.SendArchive(in.ArchiveName, func(w io.Writer) error { return ziparchive.Write(w, entries) })
}
