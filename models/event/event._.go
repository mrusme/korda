package event

import (
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/notification"
)

type Detail int

const (
	DetailBare Detail = iota
	DetailExif
	DetailFull
)

type AssetUploaded struct {
	AssetID uuid.UUID
}

type AssetUpdated struct {
	UserID  uuid.UUID
	AssetID uuid.UUID
	Detail  Detail
}

type AssetHidden struct {
	UserID  uuid.UUID
	AssetID uuid.UUID
}

type AssetsTrashed struct {
	UserID   uuid.UUID
	AssetIDs []uuid.UUID
}

type AssetsRestored struct {
	UserID   uuid.UUID
	AssetIDs []uuid.UUID
}

type AssetDeleted struct {
	UserID  uuid.UUID
	AssetID uuid.UUID
}

type StackChanged struct {
	UserID uuid.UUID
}

type AlbumUpdated struct {
	AlbumID      uuid.UUID
	UserIDs      []uuid.UUID
	RecipientIDs []uuid.UUID
}

type SessionDeleted struct {
	SessionID uuid.UUID
	At        time.Time
}

type SessionsRevoked struct {
	SessionIDs []uuid.UUID
}

type UserDeleted struct {
	UserID uuid.UUID
}

type UserSoftDeleted struct {
	UserID uuid.UUID
}

type APIKeyDeleted struct {
	KeyID uuid.UUID
}

type NotificationCreated struct {
	Notification *notification.Notification
}

type AssetEditReady struct {
	AssetID uuid.UUID
}
