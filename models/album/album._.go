package album

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleOwner  Role = "owner"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

func (Role) Enum() []string {
	return []string{string(RoleEditor), string(RoleOwner), string(RoleViewer)}
}

type Order string

const (
	OrderAsc  Order = "asc"
	OrderDesc Order = "desc"
)

func (Order) Enum() []string {
	return []string{string(OrderAsc), string(OrderDesc)}
}

type Album struct {
	ID                    uuid.UUID
	AlbumName             string
	Description           sql.Null[string]
	AlbumThumbnailAssetID uuid.NullUUID
	IsActivityEnabled     bool
	Order                 Order
	CreatedAt             time.Time
	UpdatedAt             time.Time
	DeletedAt             sql.Null[time.Time]
	UpdateID              uuid.UUID
}

type User struct {
	AlbumID   uuid.UUID
	UserID    uuid.UUID
	Role      Role
	CreateID  uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	UpdateID  uuid.UUID
}

type Asset struct {
	AlbumID   uuid.UUID
	AssetID   uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	UpdateID  uuid.UUID
}
