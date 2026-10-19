package assets

import (
	"encoding/json"
	"time"

	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/helpers/refine"
	"tty.fail/mrus/korda/models/asset"
)

type MetadataItemInput struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

type UploadInput struct {
	FileCreatedAt    time.Time           `json:"fileCreatedAt" v:"datetime"`
	FileModifiedAt   time.Time           `json:"fileModifiedAt" v:"datetime"`
	Duration         *int64              `json:"duration" v:"coerce,min=0"`
	Filename         *string             `json:"filename"`
	IsFavorite       *bool               `json:"isFavorite"`
	Visibility       *asset.Visibility   `json:"visibility"`
	LivePhotoVideoID *string             `json:"livePhotoVideoId" v:"uuid"`
	Metadata         []MetadataItemInput `json:"metadata" v:"json,optional"`
}

type BulkUploadCheckItem struct {
	ID       string `json:"id"`
	Checksum string `json:"checksum"`
}

type BulkUploadCheckInput struct {
	Assets []BulkUploadCheckItem `json:"assets"`
}

type updateBase struct {
	IsFavorite       *bool                 `json:"isFavorite"`
	Visibility       *asset.Visibility     `json:"visibility"`
	DateTimeOriginal *string               `json:"dateTimeOriginal"`
	Latitude         *float64              `json:"latitude" v:"min=-90,max=90"`
	Longitude        *float64              `json:"longitude" v:"min=-180,max=180"`
	Rating           optional.Value[int64] `json:"rating" v:"nullable,min=-1,max=5,refine=rating"`
	Description      *string               `json:"description"`
}

func coordinates(in refine.Fields) []refine.Issue {
	if in.Has("latitude") != in.Has("longitude") {
		return []refine.Issue{refine.Root("Latitude and longitude must be provided together")}
	}

	return nil
}

type UpdateInput struct {
	updateBase
	LivePhotoVideoID optional.Value[string] `json:"livePhotoVideoId" v:"nullable,uuid"`
}

func (UpdateInput) Refine(in refine.Fields) []refine.Issue {
	return coordinates(in)
}

type BulkUpdateInput struct {
	updateBase
	IDs              []string               `json:"ids" vi:"uuid"`
	DuplicateID      optional.Value[string] `json:"duplicateId" v:"nullable"`
	DateTimeRelative *int64                 `json:"dateTimeRelative"`
	TimeZone         *string                `json:"timeZone"`
}

func (BulkUpdateInput) Refine(in refine.Fields) []refine.Issue {
	return coordinates(in)
}

func (BulkUpdateInput) Pipe(in refine.Fields) []refine.Issue {
	if in.Has("dateTimeRelative") && in.Has("dateTimeOriginal") {
		return []refine.Issue{refine.Root("dateTimeRelative cannot exist alongside dateTimeOriginal")}
	}

	return nil
}

type BulkIDsInput struct {
	IDs []string `json:"ids" vi:"uuid"`
}

type BulkDeleteInput struct {
	BulkIDsInput
	Force *bool `json:"force"`
}

type JobName string

const (
	JobRefreshFaces        JobName = "refresh-faces"
	JobRefreshMetadata     JobName = "refresh-metadata"
	JobRegenerateThumbnail JobName = "regenerate-thumbnail"
	JobTranscodeVideo      JobName = "transcode-video"
)

func (JobName) Enum() []string {
	return []string{string(JobRefreshFaces), string(JobRefreshMetadata), string(JobRegenerateThumbnail), string(JobTranscodeVideo)}
}

type JobsInput struct {
	AssetIDs []string `json:"assetIds" vi:"uuid"`
	Name     JobName  `json:"name"`
}

type MetadataUpsertInput struct {
	Items []MetadataItemInput `json:"items"`
}

type MetadataBulkUpsertItem struct {
	AssetID string          `json:"assetId" v:"uuid"`
	Key     string          `json:"key"`
	Value   json.RawMessage `json:"value"`
}

type MetadataBulkUpsertInput struct {
	Items []MetadataBulkUpsertItem `json:"items"`
}

type MetadataBulkDeleteItem struct {
	AssetID string `json:"assetId" v:"uuid"`
	Key     string `json:"key"`
}

type MetadataBulkDeleteInput struct {
	Items []MetadataBulkDeleteItem `json:"items"`
}

type CopyInput struct {
	SourceID    string `json:"sourceId" v:"uuid"`
	TargetID    string `json:"targetId" v:"uuid"`
	SharedLinks *bool  `json:"sharedLinks"`
	Albums      *bool  `json:"albums"`
	Sidecar     *bool  `json:"sidecar"`
	Stack       *bool  `json:"stack"`
	Favorite    *bool  `json:"favorite"`
}

type StatisticsInput struct {
	Visibility *asset.Visibility `json:"visibility"`
	IsFavorite *bool             `json:"isFavorite"`
	IsTrashed  *bool             `json:"isTrashed"`
}

type MediaSize string

const (
	SizeOriginal  MediaSize = "original"
	SizeFullsize  MediaSize = "fullsize"
	SizePreview   MediaSize = "preview"
	SizeThumbnail MediaSize = "thumbnail"
)

func (MediaSize) Enum() []string {
	return []string{string(SizeOriginal), string(SizeFullsize), string(SizePreview), string(SizeThumbnail)}
}

type MediaOptionsInput struct {
	Size   *MediaSize `json:"size"`
	Edited *bool      `json:"edited"`
}

type DownloadOriginalInput struct {
	Edited *bool `json:"edited"`
}
