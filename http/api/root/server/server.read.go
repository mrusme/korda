package server

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logicserver "tty.fail/mrus/korda/logic/root/server"
	"tty.fail/mrus/korda/models/permission"
)

func (r *Route) GetAboutInfo(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getAboutInfo")
	if err := req.Require(logicauth.Requirement{Permission: permission.ServerAbout}); err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewAbout(logicserver.GetAbout(r.Runtime)))
}

func (r *Route) GetApkLinks(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getApkLinks")
	if err := req.Require(logicauth.Requirement{Permission: permission.ServerApkLinks}); err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewAPKLinks(logicserver.GetAPKLinks()))
}

func (r *Route) GetStorage(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getStorage")
	if err := req.Require(logicauth.Requirement{Permission: permission.ServerStorage}); err != nil {
		return err
	}

	usage, err := logicserver.GetStorage(r.Runtime)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewStorage(usage))
}

func (r *Route) GetServerStatistics(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getServerStatistics")
	if err := req.Require(logicauth.Requirement{Permission: permission.ServerStatistics, Admin: true}); err != nil {
		return err
	}

	stats, err := logicserver.GetStatistics(req.Context(), r.Runtime)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewStatistics(stats))
}
