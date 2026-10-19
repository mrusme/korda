package user

type Status string

const (
	StatusActive   Status = "active"
	StatusRemoving Status = "removing"
	StatusDeleted  Status = "deleted"
)
