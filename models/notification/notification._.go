package notification

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Level string

const (
	LevelSuccess Level = "success"
	LevelError   Level = "error"
	LevelWarning Level = "warning"
	LevelInfo    Level = "info"
)

func (Level) Enum() []string {
	return []string{string(LevelSuccess), string(LevelError), string(LevelWarning), string(LevelInfo)}
}

type Type string

const (
	TypeJobFailed           Type = "JobFailed"
	TypeBackupFailed        Type = "BackupFailed"
	TypeSystemMessage       Type = "SystemMessage"
	TypeAlbumInvite         Type = "AlbumInvite"
	TypeAlbumUpdate         Type = "AlbumUpdate"
	TypeClusterGroupRequest Type = "ClusterGroupRequest"
	TypeCustom              Type = "Custom"
)

func (Type) Enum() []string {
	return []string{string(TypeJobFailed), string(TypeBackupFailed), string(TypeSystemMessage), string(TypeAlbumInvite),
		string(TypeAlbumUpdate), string(TypeClusterGroupRequest), string(TypeCustom)}
}

type Notification struct {
	ID          uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   sql.Null[time.Time]
	UpdateID    uuid.UUID
	UserID      uuid.NullUUID
	Level       Level
	Type        Type
	Data        sql.Null[string]
	Title       string
	Description sql.Null[string]
	ReadAt      sql.Null[time.Time]
}
