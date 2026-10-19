package server

import (
	"github.com/gofiber/fiber/v3"
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
	r.Path = route.For("APIServer").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Get("/about", r.GetAboutInfo).Name("getAboutInfo")
		base.Get("/apk-links", r.GetApkLinks).Name("getApkLinks")
		base.Get("/storage", r.GetStorage).Name("getStorage")
		base.Get("/ping", r.PingServer).Name("pingServer")
		base.Get("/version", r.GetServerVersion).Name("getServerVersion")
		base.Get("/version-history", r.GetVersionHistory).Name("getVersionHistory")
		base.Get("/features", r.GetServerFeatures).Name("getServerFeatures")
		base.Get("/config", r.GetServerConfig).Name("getServerConfig")
		base.Get("/statistics", r.GetServerStatistics).Name("getServerStatistics")
		base.Get("/media-types", r.GetSupportedMediaTypes).Name("getSupportedMediaTypes")
		base.Get("/license", r.GetServerLicense).Name("getServerLicense")
		base.Put("/license", r.SetServerLicense).Name("setServerLicense")
		base.Delete("/license", r.DeleteServerLicense).Name("deleteServerLicense")
	}, r.Path+".")

	return r, nil
}
