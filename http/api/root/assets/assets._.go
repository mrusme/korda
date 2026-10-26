package assets

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
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
	r.Path = route.For("APIAssets").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Post("", r.UploadAsset).Name("uploadAsset")
		base.Put("", r.UpdateAssets).Name("updateAssets")
		base.Patch("", r.UpdateAssetsV3).Name("updateAssetsV3")
		base.Delete("", r.DeleteAssets).Name("deleteAssets")
		base.Post("/bulk-upload-check", r.CheckBulkUpload).Name("checkBulkUpload")
		base.Get("/statistics", r.GetAssetStatistics).Name("getAssetStatistics")
		base.Post("/jobs", r.RunAssetJobs).Name("runAssetJobs")
		base.Put("/metadata", r.UpdateBulkAssetMetadata).Name("updateBulkAssetMetadata")
		base.Delete("/metadata", r.DeleteBulkAssetMetadata).Name("deleteBulkAssetMetadata")
		base.Put("/copy", r.CopyAsset).Name("copyAsset")
		base.Get("/:id", r.GetAssetInfo).Name("getAssetInfo")
		base.Put("/:id", r.UpdateAsset).Name("updateAsset")
		base.Patch("/:id", r.UpdateAssetV3).Name("updateAssetV3")
		base.Get("/:id/metadata", r.GetAssetMetadata).Name("getAssetMetadata")
		base.Put("/:id/metadata", r.UpdateAssetMetadata).Name("updateAssetMetadata")
		base.Get("/:id/metadata/:key", r.GetAssetMetadataByKey).Name("getAssetMetadataByKey")
		base.Delete("/:id/metadata/:key", r.DeleteAssetMetadata).Name("deleteAssetMetadata")
		base.Get("/:id/ocr", r.GetAssetOcr).Name("getAssetOcr")
		base.Get("/:id/edits", r.GetAssetEdits).Name("getAssetEdits")
		base.Put("/:id/edits", r.EditAsset).Name("editAsset")
		base.Delete("/:id/edits", r.RemoveAssetEdits).Name("removeAssetEdits")
		base.Get("/:id/thumbnail", r.ViewAsset).Name("viewAsset")
		base.Get("/:id/original", r.DownloadAsset).Name("downloadAsset")
		base.Get("/:id/video/playback", r.PlayAssetVideo).Name("playAssetVideo")
		base.Get("/:id/video/stream/main.m3u8", r.GetMainPlaylist).Name("getMainPlaylist")
		base.Get("/:id/video/stream/:sessionId/:variantIndex/playlist.m3u8", r.GetMediaPlaylist).Name("getMediaPlaylist")
		base.Get("/:id/video/stream/:sessionId/:variantIndex/:filename", r.GetSegment).Name("getSegment")
		base.Delete("/:id/video/stream/:sessionId", r.EndSession).Name("endSession")
	}, r.Path+".")

	return r, nil
}

func parseID(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}
