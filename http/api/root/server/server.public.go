package server

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicserver "tty.fail/mrus/korda/logic/root/server"
)

func (r *Route) PingServer(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "pingServer")

	return req.Respond(fiber.StatusOK, view.Ping{Res: "pong"})
}

func (r *Route) GetServerVersion(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getServerVersion")

	return req.Respond(fiber.StatusOK, view.NewVersion())
}

func (r *Route) GetVersionHistory(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getVersionHistory")

	history, err := logicserver.VersionHistory(req.Context(), r.Runtime)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewVersionHistory(history))
}

func (r *Route) GetServerFeatures(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getServerFeatures")

	return req.Respond(fiber.StatusOK, view.NewFeatures(logicserver.GetFeatures()))
}

func (r *Route) GetServerConfig(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getServerConfig")

	config, err := logicserver.GetConfig(req.Context(), r.Runtime)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewConfig(config))
}

func (r *Route) GetSupportedMediaTypes(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getSupportedMediaTypes")

	return req.Respond(fiber.StatusOK, view.NewMediaTypes(logicserver.GetMediaTypes()))
}
