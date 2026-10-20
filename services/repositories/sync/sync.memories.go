package sync

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/google/uuid"
	"xn--gckvb8fzb.com/glides/services/database"
)

type Memory struct {
	ID        uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.Null[time.Time]
	OwnerID   uuid.UUID
	Type      string
	Data      string
	IsSaved   bool
	MemoryAt  time.Time
	SeenAt    sql.Null[time.Time]
	ShowAt    sql.Null[time.Time]
	HideAt    sql.Null[time.Time]
	UpdateID  string
}

type MemoryDelete struct {
	ID       string
	MemoryID uuid.UUID
}

type MemoryAsset struct {
	MemoryID uuid.UUID
	AssetID  uuid.UUID
	UpdateID string
}

type MemoryAssetDelete struct {
	ID       string
	MemoryID uuid.UUID
	AssetID  uuid.UUID
}

func (repo *Repository) Memories(ctx context.Context, ownerID uuid.UUID, onThisDayOnly bool, b Bounds) ([]Memory, error) {
	args := []any{}
	where := b.where("update_id", &args)
	args = append(args, ownerID)
	filter := ``
	if onThisDayOnly {
		filter = ` AND type = 'on_this_day'`
	}

	return database.Rows[Memory](ctx, repo.db, `SELECT id, created_at, updated_at, deleted_at, owner_id, type, data,
		is_saved, memory_at, seen_at, show_at, hide_at, update_id
		FROM memories WHERE `+where+` AND owner_id = $`+strconv.Itoa(len(args))+filter+b.tail("update_id"),
		args...,
	)
}

func (repo *Repository) MemoryDeletes(ctx context.Context, ownerID uuid.UUID, b Bounds) ([]MemoryDelete, error) {
	args := []any{}
	where := b.where("id", &args)
	args = append(args, ownerID)

	return database.Rows[MemoryDelete](ctx, repo.db, `SELECT id, memory_id FROM memory_audit
		WHERE `+where+` AND user_id = $`+strconv.Itoa(len(args))+b.tail("id"),
		args...,
	)
}

func (repo *Repository) MemoryAssets(ctx context.Context, ownerID uuid.UUID, onThisDayOnly bool, b Bounds) ([]MemoryAsset, error) {
	args := []any{}
	where := b.where("update_id", &args)
	args = append(args, ownerID)
	filter := ``
	if onThisDayOnly {
		filter = ` AND type = 'on_this_day'`
	}

	return database.Rows[MemoryAsset](ctx, repo.db, `SELECT memory_id, asset_id, update_id FROM memory_assets
		WHERE `+where+` AND memory_id IN (SELECT id FROM memories WHERE owner_id = $`+strconv.Itoa(len(args))+filter+`)`+
		b.tail("update_id"),
		args...,
	)
}

func (repo *Repository) MemoryAssetDeletes(ctx context.Context, ownerID uuid.UUID, b Bounds) ([]MemoryAssetDelete, error) {
	args := []any{}
	where := b.where("id", &args)
	args = append(args, ownerID)

	return database.Rows[MemoryAssetDelete](ctx, repo.db, `SELECT id, memory_id, asset_id FROM memory_asset_audit
		WHERE `+where+` AND memory_id IN (SELECT id FROM memories WHERE owner_id = $`+strconv.Itoa(len(args))+`)`+
		b.tail("id"),
		args...,
	)
}
