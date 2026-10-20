package api

import (
	"crypto/rand"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/http/api/problem"
	"tty.fail/mrus/korda/http/api/root"
	"tty.fail/mrus/korda/http/helpers/deadline"
	"tty.fail/mrus/korda/http/helpers/requestlog"
	"tty.fail/mrus/korda/http/routes"
	"xn--gckvb8fzb.com/glides/http/route"
	"xn--gckvb8fzb.com/glides/runtime"
)

const (
	CorrelationHeader = "X-Correlation-ID"
	correlationChars  = "abcdefghijklmnopqrstuvwxyz0123456789"
	limitWindow       = time.Minute
)

type API struct {
	rt      *runtime.Runtime
	cfg     gh.APIConfig
	timeout time.Duration
	app     *fiber.App
	r       route.IRouteController
}

func New(rt *runtime.Runtime, cfg gh.APIConfig, timeout time.Duration) (*API, error) {
	srv := new(API)
	srv.rt = rt
	srv.cfg = cfg
	srv.timeout = timeout

	srv.app = fiber.New(fiber.Config{
		AppName:       "korda-api",
		StrictRouting: false,
		CaseSensitive: false,
		ErrorHandler: func(c fiber.Ctx, err error) error {
			return problem.Respond(c, srv.rt, err)
		},
	})

	return srv, nil
}

func (srv *API) App() *fiber.App {
	return srv.app
}

func correlationID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = correlationChars[int(b[i])%len(correlationChars)]
	}

	return string(b)
}

func (srv *API) loadMiddlewares() {
	srv.app.Use(requestid.New(requestid.Config{
		Header:    CorrelationHeader,
		Generator: correlationID,
	}))
	srv.app.Use(requestlog.New(srv.rt.Logger(), requestlog.Config{
		DefaultLevel:     slog.LevelDebug,
		ClientErrorLevel: slog.LevelInfo,
		ServerErrorLevel: slog.LevelError,
	}))
	srv.app.Use(recover.New())
	srv.app.Use(deadline.New(srv.timeout))

	if len(srv.cfg.CORSOrigins) > 0 {
		srv.app.Use(cors.New(cors.Config{
			AllowOrigins: srv.cfg.CORSOrigins,
			AllowMethods: []string{
				fiber.MethodGet,
				fiber.MethodPost,
				fiber.MethodPut,
				fiber.MethodPatch,
				fiber.MethodDelete,
				fiber.MethodOptions,
			},
			AllowHeaders: []string{
				fiber.HeaderAuthorization,
				fiber.HeaderContentType,
				"x-api-key",
				"x-immich-user-token",
				"x-immich-session-token",
				"x-immich-share-key",
				"x-immich-share-slug",
			},
			ExposeHeaders: []string{CorrelationHeader},
		}))
	}

	srv.app.Post(routes.APIPath("APIAuthLogin"), srv.loginLimiter())
}

func (srv *API) loginLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Expiration:     limitWindow,
		DisableHeaders: true,
		Next: func(c fiber.Ctx) bool {
			return srv.cfg.RateLimitAuth <= 0
		},
		MaxFunc: func(c fiber.Ctx) int {
			return srv.cfg.RateLimitAuth
		},
		KeyGenerator: func(c fiber.Ctx) string {
			return "login:" + c.IP()
		},
		LimitReached: func(c fiber.Ctx) error {
			return problem.Respond(c, srv.rt, errs.ErrTooManyRequests)
		},
	})
}

func (srv *API) Startup() (err error) {
	srv.rt.Debug("status", "exec")

	srv.loadMiddlewares()

	if srv.r, err = root.New(srv.rt, srv.app); err != nil {
		srv.rt.Error("status", "error", "error", err)
		return err
	}

	srv.rt.Info("status", "ok")

	return nil
}
