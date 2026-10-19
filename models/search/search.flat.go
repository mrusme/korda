package search

import (
	"strings"
	"time"

	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/helpers/refine"
	"tty.fail/mrus/korda/models/asset"
)

type Base struct {
	LibraryID     optional.Value[string]   `json:"libraryId" v:"nullable,uuid"`
	Type          *asset.Type              `json:"type"`
	IsEncoded     *bool                    `json:"isEncoded" v:"boolean"`
	IsFavorite    *bool                    `json:"isFavorite" v:"boolean"`
	IsMotion      *bool                    `json:"isMotion" v:"boolean"`
	IsOffline     *bool                    `json:"isOffline" v:"boolean"`
	Visibility    *asset.Visibility        `json:"visibility"`
	CreatedBefore *time.Time               `json:"createdBefore" v:"datetime"`
	CreatedAfter  *time.Time               `json:"createdAfter" v:"datetime"`
	UpdatedBefore *time.Time               `json:"updatedBefore" v:"datetime"`
	UpdatedAfter  *time.Time               `json:"updatedAfter" v:"datetime"`
	TrashedBefore *time.Time               `json:"trashedBefore" v:"datetime"`
	TrashedAfter  *time.Time               `json:"trashedAfter" v:"datetime"`
	TakenBefore   *time.Time               `json:"takenBefore" v:"datetime"`
	TakenAfter    *time.Time               `json:"takenAfter" v:"datetime"`
	City          optional.Value[string]   `json:"city" v:"nullable"`
	State         optional.Value[string]   `json:"state" v:"nullable"`
	Country       optional.Value[string]   `json:"country" v:"nullable"`
	Make          optional.Value[string]   `json:"make" v:"nullable"`
	Model         optional.Value[string]   `json:"model" v:"nullable"`
	LensModel     optional.Value[string]   `json:"lensModel" v:"nullable"`
	IsNotInAlbum  *bool                    `json:"isNotInAlbum" v:"boolean"`
	PersonIDs     []string                 `json:"personIds" v:"optional" vi:"uuid"`
	TagIDs        optional.Value[[]string] `json:"tagIds" v:"nullable" vi:"uuid"`
	AlbumIDs      []string                 `json:"albumIds" v:"optional" vi:"uuid"`
	Rating        optional.Value[int64]    `json:"rating" v:"nullable,min=1,max=5"`
	Ocr           *string                  `json:"ocr"`
}

var baseDeprecated = []string{
	"libraryId", "type", "isEncoded", "isFavorite", "isMotion", "isOffline", "visibility", "createdBefore",
	"createdAfter", "updatedBefore", "updatedAfter", "trashedBefore", "trashedAfter", "takenBefore", "takenAfter",
	"city", "state", "country", "make", "model", "lensModel", "isNotInAlbum", "personIds", "tagIds", "albumIds",
	"rating", "ocr",
}

func exclusive(in refine.Fields, deprecated []string) []refine.Issue {
	var shape []string
	for _, name := range []string{"filter", "orderBy", "cursor"} {
		if in.Has(name) {
			shape = append(shape, name)
		}
	}
	if len(shape) == 0 {
		return nil
	}

	var issues []refine.Issue
	for _, field := range deprecated {
		if in.Has(field) {
			issues = append(issues, refine.At([]any{field},
				"Deprecated field "+field+" cannot be combined with "+strings.Join(shape, "/")))
		}
	}

	return issues
}

type WithResults struct {
	WithDeleted *bool  `json:"withDeleted" v:"boolean"`
	WithExif    *bool  `json:"withExif" v:"boolean"`
	Size        *int64 `json:"size" v:"min=1,max=1000"`
}

type Random struct {
	Base
	WithResults
	WithStacked *bool   `json:"withStacked" v:"boolean"`
	WithPeople  *bool   `json:"withPeople" v:"boolean"`
	Filter      *Filter `json:"filter"`
}

func (Random) Refine(in refine.Fields) []refine.Issue {
	return exclusive(in, append(append([]string{}, baseDeprecated...), "withDeleted"))
}

type Metadata struct {
	Base
	WithResults
	WithStacked      *bool      `json:"withStacked" v:"boolean"`
	WithPeople       *bool      `json:"withPeople" v:"boolean"`
	Filter           *Filter    `json:"filter"`
	ID               *string    `json:"id" v:"uuid"`
	Description      *string    `json:"description" v:"trim"`
	Checksum         *string    `json:"checksum"`
	OriginalFileName *string    `json:"originalFileName" v:"trim"`
	OriginalPath     *string    `json:"originalPath"`
	PreviewPath      *string    `json:"previewPath"`
	ThumbnailPath    *string    `json:"thumbnailPath"`
	EncodedVideoPath *string    `json:"encodedVideoPath"`
	Order            *Direction `json:"order"`
	Page             *int64     `json:"page" v:"min=1"`
	OrderBy          *Order     `json:"orderBy"`
	Cursor           *string    `json:"cursor" v:"min=1"`
}

func (Metadata) Refine(in refine.Fields) []refine.Issue {
	return exclusive(in, append(append([]string{}, baseDeprecated...),
		"withDeleted", "id", "description", "checksum", "originalFileName", "originalPath", "previewPath",
		"thumbnailPath", "encodedVideoPath", "order", "page"))
}

type Statistics struct {
	Base
	Description *string `json:"description" v:"trim"`
	Filter      *Filter `json:"filter"`
}

func (Statistics) Refine(in refine.Fields) []refine.Issue {
	return exclusive(in, append(append([]string{}, baseDeprecated...), "description"))
}

type Smart struct {
	Base
	WithDeleted  *bool   `json:"withDeleted" v:"boolean"`
	WithExif     *bool   `json:"withExif" v:"boolean"`
	Size         *int64  `json:"size" v:"min=1,max=1000"`
	Query        *string `json:"query" v:"trim"`
	QueryAssetID *string `json:"queryAssetId" v:"uuid"`
	Language     *string `json:"language"`
	Page         *int64  `json:"page" v:"min=1"`
	Filter       *Filter `json:"filter"`
}

func (Smart) Refine(in refine.Fields) []refine.Issue {
	return exclusive(in, append(append([]string{}, baseDeprecated...), "withDeleted", "page"))
}

type Large struct {
	Base
	WithDeleted *bool  `json:"withDeleted" v:"boolean"`
	WithExif    *bool  `json:"withExif" v:"boolean"`
	Size        *int64 `json:"size" v:"coerce,min=1,max=1000"`
	MinFileSize *int64 `json:"minFileSize" v:"coerce,min=0"`
}

type SuggestionType string

const (
	SuggestCountry   SuggestionType = "country"
	SuggestState     SuggestionType = "state"
	SuggestCity      SuggestionType = "city"
	SuggestMake      SuggestionType = "camera-make"
	SuggestModel     SuggestionType = "camera-model"
	SuggestLensModel SuggestionType = "camera-lens-model"
)

func (SuggestionType) Enum() []string {
	return []string{string(SuggestCountry), string(SuggestState), string(SuggestCity), string(SuggestMake),
		string(SuggestModel), string(SuggestLensModel)}
}

type Suggestions struct {
	Type        SuggestionType `json:"type"`
	Country     *string        `json:"country"`
	State       *string        `json:"state"`
	Make        *string        `json:"make"`
	Model       *string        `json:"model"`
	LensModel   *string        `json:"lensModel"`
	IncludeNull *bool          `json:"includeNull"`
}

type Places struct {
	Name string `json:"name"`
}

type People struct {
	Name       string `json:"name"`
	WithHidden *bool  `json:"withHidden"`
}
