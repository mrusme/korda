package usermetadata

import (
	"time"

	"github.com/google/uuid"
)

type Key string

const (
	Preferences Key = "preferences"
	License     Key = "license"
	Onboarding  Key = "onboarding"
)

type UserMetadata struct {
	UserID    uuid.UUID
	Key       Key
	Value     string
	UpdatedAt time.Time
	UpdateID  uuid.UUID
}

type OnboardingValue struct {
	IsOnboarded bool `json:"isOnboarded"`
}
