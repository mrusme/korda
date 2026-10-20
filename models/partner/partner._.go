package partner

import (
	"time"

	"github.com/google/uuid"
)

type Direction string

const (
	DirectionSharedBy   Direction = "shared-by"
	DirectionSharedWith Direction = "shared-with"
)

func (Direction) Enum() []string {
	return []string{string(DirectionSharedBy), string(DirectionSharedWith)}
}

type Partner struct {
	SharedByID   uuid.UUID
	SharedWithID uuid.UUID
	InTimeline   bool
	CreateID     uuid.UUID
	CreatedAt    time.Time
	UpdatedAt    time.Time
	UpdateID     uuid.UUID
}
