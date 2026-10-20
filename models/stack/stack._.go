package stack

import (
	"time"

	"github.com/google/uuid"
)

type Stack struct {
	ID             uuid.UUID
	OwnerID        uuid.UUID
	PrimaryAssetID uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
	UpdateID       uuid.UUID
}
