package asset

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Type string

const (
	TypeImage Type = "IMAGE"
	TypeVideo Type = "VIDEO"
	TypeAudio Type = "AUDIO"
	TypeOther Type = "OTHER"
)

func (Type) Enum() []string {
	return []string{string(TypeImage), string(TypeVideo), string(TypeAudio), string(TypeOther)}
}

type Status string

const (
	StatusActive  Status = "active"
	StatusTrashed Status = "trashed"
	StatusDeleted Status = "deleted"
)

const ChecksumSHA1 = "sha1"

type Asset struct {
	ID                uuid.UUID
	OwnerID           uuid.UUID
	Type              Type
	OriginalPath      string
	OriginalFileName  string
	Checksum          []byte
	ChecksumAlgorithm string
	FileCreatedAt     time.Time
	FileModifiedAt    time.Time
	LocalDateTime     time.Time
	IsFavorite        bool
	Duration          sql.Null[int64]
	LivePhotoVideoID  uuid.NullUUID
	Thumbhash         []byte
	Visibility        Visibility
	Status            Status
	DeletedAt         sql.Null[time.Time]
	StackID           uuid.NullUUID
	LibraryID         uuid.NullUUID
	IsOffline         bool
	IsExternal        bool
	IsEdited          bool
	DuplicateID       uuid.NullUUID
	Width             sql.Null[int64]
	Height            sql.Null[int64]
	CreatedAt         time.Time
	UpdatedAt         time.Time
	UpdateID          uuid.UUID
}

func (a *Asset) IsTrashed() bool {
	return a.DeletedAt.Valid
}
