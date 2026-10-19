package search

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logicsearch "tty.fail/mrus/korda/logic/root/search"
	"tty.fail/mrus/korda/models/permission"
	"tty.fail/mrus/korda/models/search"
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
	r.Path = route.For("APISearch").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Post("/metadata", r.SearchAssets).Name("searchAssets")
		base.Post("/statistics", r.SearchAssetStatistics).Name("searchAssetStatistics")
		base.Post("/random", r.SearchRandom).Name("searchRandom")
		base.Post("/large-assets", r.SearchLargeAssets).Name("searchLargeAssets")
		base.Post("/smart", r.SearchSmart).Name("searchSmart")
		base.Get("/explore", r.GetExploreData).Name("getExploreData")
		base.Get("/person", r.SearchPerson).Name("searchPerson")
		base.Get("/places", r.SearchPlaces).Name("searchPlaces")
		base.Get("/cities", r.GetAssetsByCity).Name("getAssetsByCity")
		base.Get("/suggestions", r.GetSearchSuggestions).Name("getSearchSuggestions")
	}, r.Path+".")

	return r, nil
}

func (r *Route) SearchAssets(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "searchAssets")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetRead}); err != nil {
		return err
	}

	var in search.Metadata
	if err := req.Body(&in); err != nil {
		return err
	}

	page, err := logicsearch.Metadata(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	items, err := view.NewAssetList(req.Context(), r.Runtime, page.Items, req.Auth.User.ID, page.WithExif, false)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewSearchResponse(items, page.NextPage, page.NextCursor))
}

func (r *Route) SearchAssetStatistics(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "searchAssetStatistics")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetStatistics}); err != nil {
		return err
	}

	var in search.Statistics
	if err := req.Body(&in); err != nil {
		return err
	}

	total, err := logicsearch.Statistics(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.SearchTotal{Total: total})
}

func (r *Route) SearchRandom(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "searchRandom")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetRead}); err != nil {
		return err
	}

	var in search.Random
	if err := req.Body(&in); err != nil {
		return err
	}

	list, err := logicsearch.Random(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	items, err := view.NewAssetList(req.Context(), r.Runtime, list, req.Auth.User.ID, in.WithExif != nil && *in.WithExif, false)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, items)
}

func (r *Route) SearchLargeAssets(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "searchLargeAssets")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetRead}); err != nil {
		return err
	}

	var in search.Large
	if err := req.Query(&in); err != nil {
		return err
	}

	list, err := logicsearch.Large(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	items, err := view.NewAssetList(req.Context(), r.Runtime, list, req.Auth.User.ID, true, false)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, items)
}

func (r *Route) SearchSmart(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "searchSmart")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetRead}); err != nil {
		return err
	}

	var in search.Smart
	if err := req.Body(&in); err != nil {
		return err
	}

	return logicsearch.Smart(req.Auth, &in)
}

func (r *Route) GetExploreData(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getExploreData")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetRead}); err != nil {
		return err
	}

	recent, err := logicsearch.Recent(req.Context(), r.Runtime, req.Auth)
	if err != nil {
		return err
	}

	items, err := view.NewAssetList(req.Context(), r.Runtime, recent, req.Auth.User.ID, true, true)
	if err != nil {
		return err
	}

	recentItems := make([]view.ExploreItem, 0, len(items))
	for i := range items {
		recentItems = append(recentItems, view.ExploreItem{Value: items[i].CreatedAt.Format("2006-01-02T15:04:05.000Z"), Data: items[i]})
	}

	return req.Respond(fiber.StatusOK, []view.Explore{
		{FieldName: "exifInfo.city", Items: []view.ExploreItem{}},
		{FieldName: "createdAt", Items: recentItems},
	})
}

func (r *Route) SearchPerson(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "searchPerson")
	if err := req.Require(logicauth.Requirement{Permission: permission.PersonRead}); err != nil {
		return err
	}

	var in search.People
	if err := req.Query(&in); err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, []any{})
}

func (r *Route) SearchPlaces(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "searchPlaces")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetRead}); err != nil {
		return err
	}

	var in search.Places
	if err := req.Query(&in); err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, []any{})
}

func (r *Route) GetAssetsByCity(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getAssetsByCity")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetRead}); err != nil {
		return err
	}

	list, err := logicsearch.Cities(req.Context(), r.Runtime, req.Auth)
	if err != nil {
		return err
	}

	items, err := view.NewAssetList(req.Context(), r.Runtime, list, uuid.Nil, true, false)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, items)
}

func (r *Route) GetSearchSuggestions(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getSearchSuggestions")
	if err := req.Require(logicauth.Requirement{Permission: permission.AssetRead}); err != nil {
		return err
	}

	var in search.Suggestions
	if err := req.Query(&in); err != nil {
		return err
	}

	values, err := logicsearch.Suggestions(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, values)
}
