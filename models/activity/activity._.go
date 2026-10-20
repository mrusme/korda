package activity

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Type string

const (
	TypeComment Type = "comment"
	TypeLike    Type = "like"
)

func (Type) Enum() []string {
	return []string{string(TypeComment), string(TypeLike)}
}

type Activity struct {
	ID        uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	AlbumID   uuid.UUID
	UserID    uuid.UUID
	AssetID   uuid.NullUUID
	Comment   sql.Null[string]
	IsLiked   bool
	UpdateID  uuid.UUID
}
