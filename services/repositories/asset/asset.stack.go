package asset

import (
	"context"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/asset"
	"xn--gckvb8fzb.com/glides/services/database"
)

func (repo *Repository) StackMembers(ctx context.Context, stackID uuid.UUID) ([]asset.Asset, error) {
	return database.Rows[asset.Asset](ctx, repo.db, `SELECT `+prefixed("a")+` FROM assets a
		JOIN asset_exif e ON e.asset_id = a.id
		WHERE a.stack_id = $1 AND a.deleted_at IS NULL AND a.visibility IN ('archive', 'timeline')
		ORDER BY a.file_created_at, a.id`,
		stackID,
	)
}

func (repo *Repository) StackOf(ctx context.Context, id uuid.UUID) (uuid.NullUUID, uuid.NullUUID, error) {
	type row struct {
		StackID        uuid.NullUUID
		PrimaryAssetID uuid.NullUUID
	}

	r, err := database.Row[row](ctx, repo.db, `SELECT a.stack_id, s.primary_asset_id FROM assets a
		LEFT JOIN stacks s ON s.id = a.stack_id WHERE a.id = $1`, id)
	if err != nil {
		return uuid.NullUUID{}, uuid.NullUUID{}, err
	}

	return r.StackID, r.PrimaryAssetID, nil
}

type StackSummary struct {
	ID             uuid.UUID
	PrimaryAssetID uuid.UUID
	AssetCount     int64
}

func (repo *Repository) StackSummary(ctx context.Context, stackID uuid.UUID) (*StackSummary, error) {
	return database.Row[StackSummary](ctx, repo.db, `SELECT s.id, s.primary_asset_id, count(a.id) + 1 AS asset_count
		FROM stacks s
		JOIN assets a ON a.stack_id = s.id AND a.id <> s.primary_asset_id AND a.deleted_at IS NULL
			AND a.visibility = 'timeline'
		WHERE s.id = $1
		GROUP BY s.id, s.primary_asset_id`,
		stackID,
	)
}
