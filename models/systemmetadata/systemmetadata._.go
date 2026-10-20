package systemmetadata

type Key string

const (
	AdminOnboarding Key = "admin-onboarding"
	MemoriesState   Key = "memories-state"
)

type SystemMetadata struct {
	Key   Key
	Value string
}

type AdminOnboardingValue struct {
	IsOnboarded bool `json:"isOnboarded"`
}
