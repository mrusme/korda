package geomap

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logicgeomap "tty.fail/mrus/korda/logic/root/geomap"
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
	r.Path = route.For("APIMap").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Get("/markers", r.GetMapMarkers).Name("getMapMarkers")
		base.Get("/reverse-geocode", r.ReverseGeocode).Name("reverseGeocode")
	}, r.Path+".")

	return r, nil
}

func (r *Route) GetMapMarkers(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getMapMarkers")
	if err := req.Require(logicauth.Requirement{Permission: permission.MapRead}); err != nil {
		return err
	}

	var in logicgeomap.MarkersInput
	if err := req.Query(&in); err != nil {
		return err
	}

	list, err := logicgeomap.Markers(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewAssetMapMarkers(list))
}

func (r *Route) ReverseGeocode(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "reverseGeocode")
	if err := req.Require(logicauth.Requirement{Permission: permission.MapSearch}); err != nil {
		return err
	}

	var in logicgeomap.ReverseInput
	if err := req.Query(&in); err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, []view.ReverseGeocode{{}})
}
