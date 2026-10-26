package asset

import (
	"context"
	"slices"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/services/repositories/query"
	"xn--gckvb8fzb.com/glides/services/database"
)

func (repo *Repository) DuplicateAccess(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	out := []uuid.UUID{}
	for chunk := range slices.Chunk(ids, query.ChunkSize) {
		args := []any{userID}
		in := query.Placeholders(&args, chunk)
		found, err := database.Values[uuid.UUID](ctx, repo.db, `SELECT DISTINCT duplicate_id FROM assets
			WHERE owner_id = $1 AND deleted_at IS NULL AND duplicate_id IN (`+in+`)`, args...)
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}

	return out, nil
}

func (repo *Repository) ClearSingletonDuplicates(ctx context.Context, ownerID uuid.UUID, now time.Time) error {
	_, err := database.Exec(ctx, repo.db, `UPDATE assets SET duplicate_id = NULL, updated_at = $2, update_id = $3
		WHERE duplicate_id IN (SELECT duplicate_id FROM assets
			WHERE owner_id = $1 AND duplicate_id IS NOT NULL AND deleted_at IS NULL AND stack_id IS NULL
			GROUP BY duplicate_id HAVING count(id) = 1)`,
		ownerID, now, uuid.Must(uuid.NewV7()))

	return err
}

const duplicateFilter = `FROM assets a JOIN asset_exif e ON e.asset_id = a.id
	WHERE a.visibility IN ('archive', 'timeline') AND a.deleted_at IS NULL AND a.stack_id IS NULL`

func (repo *Repository) Duplicates(ctx context.Context, ownerID uuid.UUID) ([]asset.Asset, error) {
	return database.Rows[asset.Asset](ctx, repo.db, `SELECT `+prefixed("a")+` `+duplicateFilter+`
		AND a.owner_id = $1 AND a.duplicate_id IS NOT NULL
		ORDER BY a.duplicate_id, a.local_date_time`, ownerID)
}

func (repo *Repository) DuplicateGroup(ctx context.Context, duplicateID uuid.UUID) ([]asset.Asset, error) {
	return database.Rows[asset.Asset](ctx, repo.db, `SELECT `+prefixed("a")+` `+duplicateFilter+`
		AND a.duplicate_id = $1 ORDER BY a.local_date_time`, duplicateID)
}

func (repo *Repository) ClearDuplicates(ctx context.Context, ownerID uuid.UUID, duplicateIDs []uuid.UUID, now time.Time) error {
	for chunk := range slices.Chunk(duplicateIDs, query.ChunkSize) {
		args := []any{ownerID, now, uuid.Must(uuid.NewV7())}
		in := query.Placeholders(&args, chunk)
		if _, err := database.Exec(ctx, repo.db, `UPDATE assets SET duplicate_id = NULL, updated_at = $2, update_id = $3
			WHERE owner_id = $1 AND duplicate_id IN (`+in+`)`, args...); err != nil {
			return err
		}
	}

	return nil
}
