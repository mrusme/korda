package user

import (
	"context"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/services/repositories/syncstate"
	"xn--gckvb8fzb.com/glides/services/database"
)

func (repo *Repository) Delete(ctx context.Context, id uuid.UUID, now time.Time) (err error) {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.End(ctx)

	owned := `SELECT album_id FROM album_users WHERE user_id = $1 AND role = 'owner'`
	members := `SELECT album_id AS a, user_id AS b FROM album_users WHERE album_id IN (` + owned + `)`
	for _, table := range []string{"album_audit", "album_user_audit"} {
		if err = syncstate.Audit(ctx, tx, table, "album_id, user_id", members, now, id); err != nil {
			return err
		}
	}
	if _, err = database.Exec(ctx, tx, `DELETE FROM albums WHERE id IN (`+owned+`)`, id); err != nil {
		return err
	}

	memberships := `SELECT album_id AS a, user_id AS b FROM album_users WHERE user_id = $1`
	for _, table := range []string{"album_audit", "album_user_audit"} {
		if err = syncstate.Audit(ctx, tx, table, "album_id, user_id", memberships, now, id); err != nil {
			return err
		}
	}
	if err = syncstate.Audit(ctx, tx, "album_asset_audit", "album_id, asset_id", `SELECT aa.album_id AS a,
		aa.asset_id AS b FROM album_assets aa JOIN assets a ON a.id = aa.asset_id WHERE a.owner_id = $1`, now, id); err != nil {
		return err
	}
	thumbnails, err := database.Values[uuid.UUID](ctx, tx, `SELECT id FROM albums
		WHERE album_thumbnail_asset_id IN (SELECT id FROM assets WHERE owner_id = $1)`, id)
	if err != nil {
		return err
	}
	if err = syncstate.Bump(ctx, tx, "albums", "album_thumbnail_asset_id = NULL", thumbnails, now); err != nil {
		return err
	}
	if _, err = database.Exec(ctx, tx, `DELETE FROM users WHERE id = $1`, id); err != nil {
		return err
	}
	if _, err = database.Exec(ctx, tx, `INSERT INTO user_audit (id, user_id, deleted_at)
		VALUES ($1, $2, $3)`,
		uuid.Must(uuid.NewV7()), id, now,
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
