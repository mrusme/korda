package tag

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Tag struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Value     string
	Color     sql.Null[string]
	ParentID  uuid.NullUUID
	CreatedAt time.Time
	UpdatedAt time.Time
	UpdateID  uuid.UUID
}
