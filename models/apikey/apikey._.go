package apikey

import (
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/permission"
)

type APIKey struct {
	ID          uuid.UUID
	Name        string
	Key         []byte
	UserID      uuid.UUID
	Permissions permission.List
	CreatedAt   time.Time
	UpdatedAt   time.Time
	UpdateID    uuid.UUID
}

const DefaultName = "API Key"
