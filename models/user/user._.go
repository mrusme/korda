package user

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                   uuid.UUID
	Email                string
	Password             sql.Null[string]
	PinCode              sql.Null[string]
	Name                 string
	IsAdmin              bool
	ShouldChangePassword bool
	AvatarColor          sql.Null[AvatarColor]
	ProfileImagePath     string
	ProfileChangedAt     time.Time
	StorageLabel         sql.Null[string]
	QuotaSizeInBytes     sql.Null[int64]
	QuotaUsageInBytes    int64
	Status               Status
	OAuthID              sql.Null[string]
	ClusterGroupID       uuid.UUID
	CreatedAt            time.Time
	UpdatedAt            time.Time
	DeletedAt            sql.Null[time.Time]
	UpdateID             uuid.UUID
}

func (u *User) Color() AvatarColor {
	if u.AvatarColor.Valid {
		return u.AvatarColor.V
	}

	return ColorFromEmail(u.Email)
}

func (u *User) HasPassword() bool {
	return u.Password.Valid && u.Password.V != ""
}

func (u *User) HasPinCode() bool {
	return u.PinCode.Valid && u.PinCode.V != ""
}
