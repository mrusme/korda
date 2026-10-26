package people

import (
	"slices"

	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/helpers/refine"
)

type Role string

const (
	RoleRead  Role = "read"
	RoleWrite Role = "write"
	RoleAdmin Role = "admin"
)

func (Role) Enum() []string {
	return []string{string(RoleRead), string(RoleWrite), string(RoleAdmin)}
}

type Direction string

func (Direction) Enum() []string {
	return []string{"shared-by", "shared-with"}
}

type UpsertType string

func (UpsertType) Enum() []string {
	return []string{"everyone"}
}

type SearchInput struct {
	WithHidden      *bool   `json:"withHidden"`
	ClosestPersonID *string `json:"closestPersonId" v:"uuid"`
	ClosestAssetID  *string `json:"closestAssetId" v:"uuid"`
	Page            *int64  `json:"page" v:"coerce,min=1"`
	Size            *int64  `json:"size" v:"coerce,min=1,max=1000"`
	SharedByID      *string `json:"sharedById" v:"anyuuid"`
	SharedWithID    *string `json:"sharedWithId" v:"anyuuid"`
	IsFavorite      *bool   `json:"isFavorite"`
	IsHidden        *bool   `json:"isHidden"`
	Name            *string `json:"name"`
}

type CreateInput struct {
	Name       *string                `json:"name"`
	BirthDate  optional.Value[string] `json:"birthDate" v:"nullable,refine=birthDate"`
	IsHidden   *bool                  `json:"isHidden"`
	IsFavorite *bool                  `json:"isFavorite"`
	Color      optional.Value[string] `json:"color" v:"nullable,regex=hexColor"`
}

type UpdateInput struct {
	CreateInput
	FeatureFaceAssetID *string `json:"featureFaceAssetId" v:"uuid"`
	UserID             *string `json:"userId" v:"anyuuid"`
}

var updateFields = []string{"name", "birthDate", "isHidden", "isFavorite", "color", "featureFaceAssetId"}

func (UpdateInput) Refine(in refine.Fields) []refine.Issue {
	if slices.ContainsFunc(updateFields, in.Has) {
		return nil
	}

	return []refine.Issue{refine.Root("At least one of the following fields is required: name, birthDate, isHidden, isFavorite, color, featureFaceAssetId")}
}

type UpdateItem struct {
	CreateInput
	FeatureFaceAssetID *string `json:"featureFaceAssetId" v:"uuid"`
	UserID             *string `json:"userId" v:"anyuuid"`
	ID                 string  `json:"id" v:"uuid"`
}

type UpdateAllInput struct {
	People []UpdateItem `json:"people"`
}

type MergeInput struct {
	IDs []string `json:"ids" vi:"uuid"`
}

type DeleteInput struct {
	UserID *string `json:"userId"`
}

type DeleteAllInput struct {
	IDs    []string `json:"ids" vi:"uuid"`
	UserID *string  `json:"userId"`
}

type FaceUpdateItem struct {
	PersonID string  `json:"personId" v:"uuid"`
	AssetID  string  `json:"assetId" v:"uuid"`
	UserID   *string `json:"userId" v:"uuid"`
}

type ReassignInput struct {
	Data []FaceUpdateItem `json:"data"`
}

type FaceInput struct {
	ID string `json:"id" v:"uuid"`
}

type FaceCreateInput struct {
	FaceUpdateItem
	ImageWidth  int64 `json:"imageWidth"`
	ImageHeight int64 `json:"imageHeight"`
	X           int64 `json:"x"`
	Y           int64 `json:"y"`
	Width       int64 `json:"width"`
	Height      int64 `json:"height"`
}

type FaceDeleteInput struct {
	Force bool `json:"force"`
}

type UsersSearchInput struct {
	PersonID     *string    `json:"personId" v:"anyuuid"`
	Direction    *Direction `json:"direction"`
	SharedByID   *string    `json:"sharedById" v:"anyuuid"`
	SharedWithID *string    `json:"sharedWithId" v:"anyuuid"`
	Role         *Role      `json:"role"`
}

type UsersUpsertInput struct {
	PersonIDs     []string    `json:"personIds" v:"optional,refine=uniqueIds" vi:"anyuuid"`
	Type          *UpsertType `json:"type"`
	SharedWithIDs []string    `json:"sharedWithIds" v:"refine=uniqueIds" vi:"anyuuid"`
	Role          Role        `json:"role"`
}

func (UsersUpsertInput) Refine(in refine.Fields) []refine.Issue {
	personIDs, hasPersonIDs := in.Get("personIds")
	kind, hasType := in.Get("type")
	if (hasPersonIDs && personIDs != nil) || (hasType && kind != nil) {
		return nil
	}

	return []refine.Issue{refine.At([]any{"personIds"}, "Either personIds or type must be provided")}
}

type UserDeleteItem struct {
	PersonID     string  `json:"personId" v:"anyuuid"`
	SharedWithID string  `json:"sharedWithId" v:"anyuuid"`
	SharedByID   *string `json:"sharedById" v:"anyuuid"`
}

type UsersDeleteInput []UserDeleteItem
