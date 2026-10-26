package assetfile

import (
	"time"

	"github.com/google/uuid"
)

type Type string

const (
	TypeFullSize     Type = "fullsize"
	TypePreview      Type = "preview"
	TypeThumbnail    Type = "thumbnail"
	TypeSidecar      Type = "sidecar"
	TypeEncodedVideo Type = "encoded_video"
)

func (Type) Enum() []string {
	return []string{string(TypeFullSize), string(TypePreview), string(TypeThumbnail), string(TypeSidecar), string(TypeEncodedVideo)}
}

type File struct {
	ID            uuid.UUID
	AssetID       uuid.UUID
	Type          Type
	Path          string
	IsEdited      bool
	IsProgressive bool
	IsTransparent bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
	UpdateID      uuid.UUID
}
