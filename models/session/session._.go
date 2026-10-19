package session

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID                 uuid.UUID
	Token              []byte
	UserID             uuid.UUID
	ParentID           uuid.NullUUID
	DeviceType         string
	DeviceOS           string
	AppVersion         sql.Null[string]
	ExpiresAt          sql.Null[time.Time]
	PinExpiresAt       sql.Null[time.Time]
	IsPendingSyncReset bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
	UpdateID           uuid.UUID
}
