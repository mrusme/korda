package versionhistory

import (
	"time"

	"github.com/google/uuid"
)

type VersionHistory struct {
	ID        uuid.UUID
	CreatedAt time.Time
	Version   string
}
