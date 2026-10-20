package albums

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicalbums "tty.fail/mrus/korda/logic/root/albums"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
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
	r.Path = route.For("APIAlbums").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Get("", r.GetAllAlbums).Name("getAllAlbums")
		base.Post("", r.CreateAlbum).Name("createAlbum")
		base.Get("/statistics", r.GetAlbumStatistics).Name("getAlbumStatistics")
		base.Put("/assets", r.AddAssetsToAlbums).Name("addAssetsToAlbums")
		base.Get("/:id", r.GetAlbumInfo).Name("getAlbumInfo")
		base.Patch("/:id", r.UpdateAlbumInfo).Name("updateAlbumInfo")
		base.Delete("/:id", r.DeleteAlbum).Name("deleteAlbum")
		base.Get("/:id/map-markers", r.GetAlbumMapMarkers).Name("getAlbumMapMarkers")
		base.Put("/:id/assets", r.AddAssetsToAlbum).Name("addAssetsToAlbum")
		base.Delete("/:id/assets", r.RemoveAssetFromAlbum).Name("removeAssetFromAlbum")
		base.Put("/:id/users", r.AddUsersToAlbum).Name("addUsersToAlbum")
		base.Put("/:id/user/:userId", r.UpdateAlbumUser).Name("updateAlbumUser")
		base.Delete("/:id/user/:userId", r.RemoveUserFromAlbum).Name("removeUserFromAlbum")
	}, r.Path+".")

	return r, nil
}

func (r *Route) GetAllAlbums(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getAllAlbums")
	if err := req.Require(logicauth.Requirement{Permission: permission.AlbumRead}); err != nil {
		return err
	}

	var in logicalbums.ListInput
	if err := req.Query(&in); err != nil {
		return err
	}

	list, err := logicalbums.List(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewAlbums(list))
}

func (r *Route) CreateAlbum(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "createAlbum")
	if err := req.Require(logicauth.Requirement{Permission: permission.AlbumCreate}); err != nil {
		return err
	}

	var in logicalbums.CreateInput
	if err := req.Body(&in); err != nil {
		return err
	}

	d, err := logicalbums.Create(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusCreated, view.NewAlbum(d))
}

func (r *Route) GetAlbumStatistics(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getAlbumStatistics")
	if err := req.Require(logicauth.Requirement{Permission: permission.AlbumStatistics}); err != nil {
		return err
	}

	s, err := logicalbums.Stats(req.Context(), r.Runtime, req.Auth)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.AlbumStatistics{Owned: s.Owned, Shared: s.Shared, NotShared: s.NotShared})
}

func (r *Route) AddAssetsToAlbums(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "addAssetsToAlbums")
	if err := req.Require(logicauth.Requirement{Permission: permission.AlbumAssetCreate}); err != nil {
		return err
	}

	var in logicalbums.AddToAlbumsInput
	if err := req.Body(&in); err != nil {
		return err
	}

	success, reason, err := logicalbums.AddToAlbums(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.AlbumsAddAssets{Success: success, Error: reason})
}

func (r *Route) GetAlbumInfo(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getAlbumInfo")
	if err := req.Require(logicauth.Requirement{Permission: permission.AlbumRead, SharedLink: true}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	d, err := logicalbums.Get(req.Context(), r.Runtime, req.Auth, id)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewAlbum(d))
}

func (r *Route) UpdateAlbumInfo(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "updateAlbumInfo")
	if err := req.Require(logicauth.Requirement{Permission: permission.AlbumUpdate}); err != nil {
		return err
	}

	var in logicalbums.UpdateInput
	if err := req.Body(&in); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	d, err := logicalbums.Update(req.Context(), r.Runtime, req.Auth, id, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewAlbum(d))
}

func (r *Route) DeleteAlbum(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "deleteAlbum")
	if err := req.Require(logicauth.Requirement{Permission: permission.AlbumDelete}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	if err = logicalbums.Delete(req.Context(), r.Runtime, req.Auth, id); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) GetAlbumMapMarkers(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getAlbumMapMarkers")
	if err := req.Require(logicauth.Requirement{Permission: permission.AlbumRead, SharedLink: true}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	list, err := logicalbums.MapMarkers(req.Context(), r.Runtime, req.Auth, id)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewMapMarkers(list))
}

func (r *Route) AddAssetsToAlbum(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "addAssetsToAlbum")
	if err := req.Require(logicauth.Requirement{Permission: permission.AlbumAssetCreate}); err != nil {
		return err
	}

	var in logicalbums.BulkIDsInput
	if err := req.Body(&in); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	results, err := logicalbums.AddAssets(req.Context(), r.Runtime, req.Auth, id, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewBulkIDs(results))
}

func (r *Route) RemoveAssetFromAlbum(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "removeAssetFromAlbum")
	if err := req.Require(logicauth.Requirement{Permission: permission.AlbumAssetDelete}); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	var in logicalbums.BulkIDsInput
	if err = req.Body(&in); err != nil {
		return err
	}

	results, err := logicalbums.RemoveAssets(req.Context(), r.Runtime, req.Auth, id, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewBulkIDs(results))
}

func (r *Route) AddUsersToAlbum(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "addUsersToAlbum")
	if err := req.Require(logicauth.Requirement{Permission: permission.AlbumUserCreate}); err != nil {
		return err
	}

	var in logicalbums.AddUsersInput
	if err := req.Body(&in); err != nil {
		return err
	}

	id, err := req.UUID("id")
	if err != nil {
		return err
	}

	d, err := logicalbums.AddUsers(req.Context(), r.Runtime, req.Auth, id, &in)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewAlbum(d))
}

func (r *Route) UpdateAlbumUser(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "updateAlbumUser")
	if err := req.Require(logicauth.Requirement{Permission: permission.AlbumUserUpdate}); err != nil {
		return err
	}

	var in logicalbums.UpdateUserInput
	if err := req.Body(&in); err != nil {
		return err
	}

	var params logicalbums.UserParams
	if err := req.Params(&params, "id", "userId"); err != nil {
		return err
	}

	if err := logicalbums.UpdateUser(req.Context(), r.Runtime, req.Auth, &params, in.Role); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) RemoveUserFromAlbum(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "removeUserFromAlbum")
	if err := req.Require(logicauth.Requirement{Permission: permission.AlbumUserDelete}); err != nil {
		return err
	}

	var params logicalbums.UserParams
	if err := req.Params(&params, "id", "userId"); err != nil {
		return err
	}

	if err := logicalbums.RemoveUser(req.Context(), r.Runtime, req.Auth, &params); err != nil {
		return err
	}

	return req.NoContent()
}
