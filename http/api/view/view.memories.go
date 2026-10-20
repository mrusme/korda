package view

import (
	"encoding/json"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/helpers/jsontime"
	logicmemories "tty.fail/mrus/korda/logic/root/memories"
	"tty.fail/mrus/korda/models/memory"
)

type Memory struct {
	ID        string          `json:"id"`
	CreatedAt jsontime.Time   `json:"createdAt"`
	UpdatedAt jsontime.Time   `json:"updatedAt"`
	DeletedAt *jsontime.Time  `json:"deletedAt,omitempty"`
	MemoryAt  jsontime.Time   `json:"memoryAt"`
	SeenAt    *jsontime.Time  `json:"seenAt,omitempty"`
	ShowAt    *jsontime.Time  `json:"showAt,omitempty"`
	HideAt    *jsontime.Time  `json:"hideAt,omitempty"`
	OwnerID   string          `json:"ownerId"`
	Type      memory.Type     `json:"type"`
	Data      json.RawMessage `json:"data"`
	IsSaved   bool            `json:"isSaved"`
	Assets    []Asset         `json:"assets"`
}

type MemoryStatistics struct {
	Total int64 `json:"total"`
}

func NewMemory(d *logicmemories.Details, viewer uuid.UUID) (Memory, error) {
	data, err := normalized(d.Memory.Data)
	if err != nil {
		return Memory{}, err
	}

	assets := make([]Asset, 0, len(d.Assets))
	for i := range d.Assets {
		assets = append(assets, NewAsset(&d.Assets[i], viewer, AssetDetails{Aggregated: true}))
	}

	return Memory{
		ID:        id(d.Memory.ID),
		CreatedAt: time3(d.Memory.CreatedAt),
		UpdatedAt: time3(d.Memory.UpdatedAt),
		DeletedAt: nullTime(d.Memory.DeletedAt),
		MemoryAt:  time3(d.Memory.MemoryAt),
		SeenAt:    nullTime(d.Memory.SeenAt),
		ShowAt:    nullTime(d.Memory.ShowAt),
		HideAt:    nullTime(d.Memory.HideAt),
		OwnerID:   id(d.Memory.OwnerID),
		Type:      d.Memory.Type,
		Data:      data,
		IsSaved:   d.Memory.IsSaved,
		Assets:    assets,
	}, nil
}

func NewMemories(list []logicmemories.Details, viewer uuid.UUID) ([]Memory, error) {
	out := make([]Memory, 0, len(list))
	for i := range list {
		m, err := NewMemory(&list[i], viewer)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}

	return out, nil
}
