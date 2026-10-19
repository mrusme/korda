package search

import (
	"slices"
	"strings"
	"time"

	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/helpers/refine"
	"tty.fail/mrus/korda/models/asset"
)

func atLeastOne(in refine.Fields, keys ...string) []refine.Issue {
	if slices.ContainsFunc(keys, in.Has) {
		return nil
	}

	return []refine.Issue{refine.Root("At least one of the following fields is required: " + strings.Join(keys, ", "))}
}

type IDFilter struct {
	Eq *string `json:"eq" v:"uuid"`
	Ne *string `json:"ne" v:"uuid"`
}

func (IDFilter) Refine(in refine.Fields) []refine.Issue { return atLeastOne(in, "eq", "ne") }

type IDFilterNullable struct {
	Eq optional.Value[string] `json:"eq" v:"nullable,uuid"`
	Ne optional.Value[string] `json:"ne" v:"nullable,uuid"`
}

func (IDFilterNullable) Refine(in refine.Fields) []refine.Issue { return atLeastOne(in, "eq", "ne") }

type IDsFilter struct {
	Any  []string `json:"any" v:"optional,min=1" vi:"uuid"`
	All  []string `json:"all" v:"optional,min=1" vi:"uuid"`
	None []string `json:"none" v:"optional,min=1" vi:"uuid"`
}

func (IDsFilter) Refine(in refine.Fields) []refine.Issue { return atLeastOne(in, "any", "all", "none") }

type StringFilter struct {
	Eq    *string  `json:"eq"`
	Ne    *string  `json:"ne"`
	In    []string `json:"in" v:"optional,min=1"`
	NotIn []string `json:"notIn" v:"optional,min=1"`
}

func (StringFilter) Refine(in refine.Fields) []refine.Issue {
	return atLeastOne(in, "eq", "ne", "in", "notIn")
}

type StringFilterNullable struct {
	Eq    optional.Value[string] `json:"eq" v:"nullable"`
	Ne    optional.Value[string] `json:"ne" v:"nullable"`
	In    []string               `json:"in" v:"optional,min=1"`
	NotIn []string               `json:"notIn" v:"optional,min=1"`
}

func (StringFilterNullable) Refine(in refine.Fields) []refine.Issue {
	return atLeastOne(in, "eq", "ne", "in", "notIn")
}

type StringPatternFilter struct {
	Eq         optional.Value[string] `json:"eq" v:"nullable"`
	Ne         optional.Value[string] `json:"ne" v:"nullable"`
	In         []string               `json:"in" v:"optional,min=1"`
	NotIn      []string               `json:"notIn" v:"optional,min=1"`
	Like       *string                `json:"like" v:"min=1"`
	NotLike    *string                `json:"notLike" v:"min=1"`
	StartsWith *string                `json:"startsWith" v:"min=1"`
	EndsWith   *string                `json:"endsWith" v:"min=1"`
}

func (StringPatternFilter) Refine(in refine.Fields) []refine.Issue {
	return atLeastOne(in, "eq", "ne", "in", "notIn", "like", "notLike", "startsWith", "endsWith")
}

type NumberFilter struct {
	Eq    *float64  `json:"eq"`
	Ne    *float64  `json:"ne"`
	Lt    *float64  `json:"lt"`
	Lte   *float64  `json:"lte"`
	Gt    *float64  `json:"gt"`
	Gte   *float64  `json:"gte"`
	In    []float64 `json:"in" v:"optional,min=1"`
	NotIn []float64 `json:"notIn" v:"optional,min=1"`
}

func (NumberFilter) Refine(in refine.Fields) []refine.Issue {
	return atLeastOne(in, "eq", "ne", "lt", "lte", "gt", "gte", "in", "notIn")
}

type NumberFilterNullable struct {
	Eq    optional.Value[float64] `json:"eq" v:"nullable"`
	Ne    optional.Value[float64] `json:"ne" v:"nullable"`
	Lt    *float64                `json:"lt"`
	Lte   *float64                `json:"lte"`
	Gt    *float64                `json:"gt"`
	Gte   *float64                `json:"gte"`
	In    []float64               `json:"in" v:"optional,min=1"`
	NotIn []float64               `json:"notIn" v:"optional,min=1"`
}

func (NumberFilterNullable) Refine(in refine.Fields) []refine.Issue {
	return atLeastOne(in, "eq", "ne", "lt", "lte", "gt", "gte", "in", "notIn")
}

type DateFilter struct {
	Eq  *time.Time `json:"eq" v:"datetime"`
	Ne  *time.Time `json:"ne" v:"datetime"`
	Gt  *time.Time `json:"gt" v:"datetime"`
	Gte *time.Time `json:"gte" v:"datetime"`
	Lt  *time.Time `json:"lt" v:"datetime"`
	Lte *time.Time `json:"lte" v:"datetime"`
}

func (DateFilter) Refine(in refine.Fields) []refine.Issue {
	return atLeastOne(in, "eq", "ne", "gt", "gte", "lt", "lte")
}

type DateFilterNullable struct {
	Eq  optional.Value[time.Time] `json:"eq" v:"nullable,datetime"`
	Ne  optional.Value[time.Time] `json:"ne" v:"nullable,datetime"`
	Gt  *time.Time                `json:"gt" v:"datetime"`
	Gte *time.Time                `json:"gte" v:"datetime"`
	Lt  *time.Time                `json:"lt" v:"datetime"`
	Lte *time.Time                `json:"lte" v:"datetime"`
}

func (DateFilterNullable) Refine(in refine.Fields) []refine.Issue {
	return atLeastOne(in, "eq", "ne", "gt", "gte", "lt", "lte")
}

type BoolFilter struct {
	Eq bool `json:"eq"`
}

type TypeFilter struct {
	Eq    *asset.Type  `json:"eq"`
	Ne    *asset.Type  `json:"ne"`
	In    []asset.Type `json:"in" v:"optional,min=1"`
	NotIn []asset.Type `json:"notIn" v:"optional,min=1"`
}

func (TypeFilter) Refine(in refine.Fields) []refine.Issue {
	return atLeastOne(in, "eq", "ne", "in", "notIn")
}

type VisibilityFilter struct {
	Eq    *asset.Visibility  `json:"eq"`
	Ne    *asset.Visibility  `json:"ne"`
	In    []asset.Visibility `json:"in" v:"optional,min=1"`
	NotIn []asset.Visibility `json:"notIn" v:"optional,min=1"`
}

func (VisibilityFilter) Refine(in refine.Fields) []refine.Issue {
	return atLeastOne(in, "eq", "ne", "in", "notIn")
}

type SimilarityFilter struct {
	Matches string `json:"matches" v:"min=1"`
}

type Conditions struct {
	ID               *IDFilter             `json:"id"`
	LibraryID        *IDFilterNullable     `json:"libraryId"`
	Type             *TypeFilter           `json:"type"`
	Visibility       *VisibilityFilter     `json:"visibility"`
	IsFavorite       *BoolFilter           `json:"isFavorite"`
	IsMotion         *BoolFilter           `json:"isMotion"`
	IsOffline        *BoolFilter           `json:"isOffline"`
	IsEncoded        *BoolFilter           `json:"isEncoded"`
	HasAlbums        *BoolFilter           `json:"hasAlbums"`
	HasPeople        *BoolFilter           `json:"hasPeople"`
	HasTags          *BoolFilter           `json:"hasTags"`
	City             *StringFilterNullable `json:"city"`
	State            *StringFilterNullable `json:"state"`
	Country          *StringFilterNullable `json:"country"`
	Make             *StringFilterNullable `json:"make"`
	Model            *StringFilterNullable `json:"model"`
	LensModel        *StringFilterNullable `json:"lensModel"`
	Description      *StringPatternFilter  `json:"description"`
	OriginalFileName *StringPatternFilter  `json:"originalFileName"`
	OriginalPath     *StringPatternFilter  `json:"originalPath"`
	Ocr              *SimilarityFilter     `json:"ocr"`
	Rating           *NumberFilterNullable `json:"rating"`
	FileSizeInBytes  *NumberFilter         `json:"fileSizeInBytes"`
	TakenAt          *DateFilter           `json:"takenAt"`
	CreatedAt        *DateFilter           `json:"createdAt"`
	UpdatedAt        *DateFilter           `json:"updatedAt"`
	TrashedAt        *DateFilterNullable   `json:"trashedAt"`
	PersonIDs        *IDsFilter            `json:"personIds"`
	TagIDs           *IDsFilter            `json:"tagIds"`
	AlbumIDs         *IDsFilter            `json:"albumIds"`
	Checksum         *StringFilter         `json:"checksum"`
	EncodedVideoPath *StringFilter         `json:"encodedVideoPath"`
}

var conditionKeys = []string{
	"id", "libraryId", "type", "visibility", "isFavorite", "isMotion", "isOffline", "isEncoded", "hasAlbums",
	"hasPeople", "hasTags", "city", "state", "country", "make", "model", "lensModel", "description",
	"originalFileName", "originalPath", "ocr", "rating", "fileSizeInBytes", "takenAt", "createdAt", "updatedAt",
	"trashedAt", "personIds", "tagIds", "albumIds", "checksum", "encodedVideoPath",
}

type Branch struct {
	refine.Strict
	Conditions
}

func (Branch) Refine(in refine.Fields) []refine.Issue {
	if slices.ContainsFunc(conditionKeys, in.Has) {
		return nil
	}

	return []refine.Issue{refine.Root("At least one filter condition is required")}
}

type Filter struct {
	refine.Strict
	Conditions
	Or []Branch `json:"or" v:"optional,min=1"`
}

func (c *Conditions) AlbumConfined() bool {
	return c.AlbumIDs != nil && (c.AlbumIDs.Any != nil || c.AlbumIDs.All != nil)
}

func (f *Filter) FullyAlbumConfined() bool {
	if f.AlbumConfined() {
		return true
	}
	if len(f.Or) == 0 {
		return false
	}
	for i := range f.Or {
		if !f.Or[i].AlbumConfined() {
			return false
		}
	}

	return true
}

type OrderField string

const (
	OrderFileCreatedAt   OrderField = "fileCreatedAt"
	OrderLocalDateTime   OrderField = "localDateTime"
	OrderFileSizeInBytes OrderField = "fileSizeInBytes"
	OrderRating          OrderField = "rating"
)

func (OrderField) Enum() []string {
	return []string{string(OrderFileCreatedAt), string(OrderLocalDateTime), string(OrderFileSizeInBytes), string(OrderRating)}
}

type Direction string

const (
	Asc  Direction = "asc"
	Desc Direction = "desc"
)

func (Direction) Enum() []string {
	return []string{string(Asc), string(Desc)}
}

type Order struct {
	Field     *OrderField `json:"field"`
	Direction *Direction  `json:"direction"`
}
