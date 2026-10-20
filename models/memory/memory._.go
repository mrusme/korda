package memory

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Type string

const (
	TypeOnThisDay Type = "on_this_day"
	TypeBirthday  Type = "birthday"
)

func (Type) Enum() []string {
	return []string{string(TypeOnThisDay), string(TypeBirthday)}
}

type Order string

const (
	OrderAsc    Order = "asc"
	OrderDesc   Order = "desc"
	OrderRandom Order = "random"
)

func (Order) Enum() []string {
	return []string{string(OrderAsc), string(OrderDesc), string(OrderRandom)}
}

type Memory struct {
	ID        uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.Null[time.Time]
	OwnerID   uuid.UUID
	Type      Type
	Data      string
	IsSaved   bool
	MemoryAt  time.Time
	SeenAt    sql.Null[time.Time]
	ShowAt    sql.Null[time.Time]
	HideAt    sql.Null[time.Time]
	UpdateID  uuid.UUID
}
