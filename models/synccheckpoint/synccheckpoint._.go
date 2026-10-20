package synccheckpoint

import (
	"time"

	"github.com/google/uuid"
)

type Checkpoint struct {
	SessionID uuid.UUID
	Type      string
	Ack       string
	CreatedAt time.Time
	UpdatedAt time.Time
	UpdateID  uuid.UUID
}
