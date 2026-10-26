package sharedlink

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
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

func (l *SharedLink) HasPassword() bool {
	return l.Password.Valid && l.Password.V != ""
}

func (l *SharedLink) Token() string {
	sum := sha256.Sum256([]byte(l.ID.String() + "-" + l.Password.V))
	return base64.StdEncoding.EncodeToString(sum[:])
}
