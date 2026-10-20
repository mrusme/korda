package web

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/http/api"
	"tty.fail/mrus/korda/http/web/root/wellknown"
	"xn--gckvb8fzb.com/glides/http/route"
	"xn--gckvb8fzb.com/glides/runtime"
)

type Web struct {
	rt  *runtime.Runtime
	cfg gh.WebConfig
	api *api.API
	app *fiber.App
}

func New(rt *runtime.Runtime, cfg gh.WebConfig, apiConfig gh.APIConfig) (srv *Web, err error) {
	srv = new(Web)
	srv.rt = rt
	srv.cfg = cfg

	if srv.api, err = api.New(rt, apiConfig, time.Duration(cfg.RequestTimeout)*time.Second); err != nil {
		return nil, err
	}

	srv.app = fiber.New(fiber.Config{
		AppName:                      "korda",
		StrictRouting:                false,
		CaseSensitive:                false,
		BodyLimit:                    cfg.BodyLimit,
		Concurrency:                  cfg.Concurrency,
		StreamRequestBody:            true,
		DisablePreParseMultipartForm: true,
		ProxyHeader:                  cfg.ProxyHeader,
		EnableIPValidation:           cfg.EnableIPValidation,
		TrustProxy:                   cfg.TrustProxy,
		TrustProxyConfig: fiber.TrustProxyConfig{
			Loopback: cfg.TrustLoopback,
			Proxies:  cfg.TrustProxies,
		},
		ReduceMemoryUsage: cfg.ReduceMemoryUsage,
		ServerHeader:      cfg.ServerHeader,
	})

	return srv, nil
}

func (srv *Web) App() *fiber.App {
	return srv.app
}

func (srv *Web) Startup() (err error) {
	srv.rt.Debug("status", "exec")

	if err = srv.api.Startup(); err != nil {
		srv.rt.Error("status", "error", "error", err)
		return err
	}

	srv.app.Use(recover.New())
	srv.app.Use("/"+route.For("API").AsURL(), srv.api.App())

	if _, err = wellknown.New(srv.rt, srv.app); err != nil {
		srv.rt.Error("status", "error", "error", err)
		return err
	}

	srv.rt.Info("status", "ok")

	return nil
}

func (srv *Web) Run() error {
	listenAddr := fmt.Sprintf("%s:%d", srv.cfg.BindIP, srv.cfg.Port)
	if err := srv.app.Listen(listenAddr, fiber.ListenConfig{
		DisableStartupMessage: true,
	}); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (srv *Web) Shutdown() (err error) {
	srv.rt.Debug("status", "exec")

	if err = srv.app.ShutdownWithTimeout(time.Second * 5); err != nil {
		srv.rt.Error("status", "error", "error", err)
		return err
	}

	srv.rt.Info("status", "ok")

	return nil
}
