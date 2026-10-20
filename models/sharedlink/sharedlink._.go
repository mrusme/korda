package sharedlink

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Type string

const (
	TypeAlbum      Type = "ALBUM"
	TypeIndividual Type = "INDIVIDUAL"
)

func (Type) Enum() []string {
	return []string{string(TypeAlbum), string(TypeIndividual)}
}

type SharedLink struct {
	ID            uuid.UUID
	Description   sql.Null[string]
	UserID        uuid.UUID
	Key           []byte
	Type          Type
	CreatedAt     time.Time
	ExpiresAt     sql.Null[time.Time]
	AllowUpload   bool
	AlbumID       uuid.NullUUID
	AllowDownload bool
	ShowExif      bool
	Password      sql.Null[string]
	Slug          sql.Null[string]
}
