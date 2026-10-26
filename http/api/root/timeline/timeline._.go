package timeline

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logictimeline "tty.fail/mrus/korda/logic/root/timeline"
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
	r.Path = route.For("APITimeline").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Get("/buckets", r.GetTimeBuckets).Name("getTimeBuckets")
		base.Get("/bucket", r.GetTimeBucket).Name("getTimeBucket")
	}, r.Path+".")

	return r, nil
}

func (r *Route) GetTimeBuckets(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getTimeBuckets")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetRead, SharedLink: true}); err != nil {
		return err
	}

	var in logictimeline.BucketsInput
	if err := req.Query(&in); err != nil {
		return err
	}

	list, err := logictimeline.Buckets(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewTimeBuckets(list))
}

func (r *Route) GetTimeBucket(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getTimeBucket")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetRead, SharedLink: true}); err != nil {
		return err
	}

	var in logictimeline.BucketInput
	if err := req.Query(&in); err != nil {
		return err
	}

	b, err := logictimeline.Get(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.RespondJSON(fiber.StatusOK, view.TimeBucketBody(b))
}
