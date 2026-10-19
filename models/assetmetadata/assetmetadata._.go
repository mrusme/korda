package assetmetadata

import (
	"time"

	"github.com/google/uuid"
)

const KeyMobileApp = "mobile-app"

type Item struct {
	AssetID   uuid.UUID
	Key       string
	Value     string
	UpdatedAt time.Time
	UpdateID  uuid.UUID
}
