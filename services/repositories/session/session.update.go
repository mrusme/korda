package session

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/session"
	"xn--gckvb8fzb.com/glides/services/database"
)

func (repo *Repository) Touch(ctx context.Context, id uuid.UUID, appVersion sql.Null[string], deviceOS string, deviceType string, now time.Time) error {
	_, err := database.Exec(ctx, repo.db, `UPDATE sessions
		SET app_version = $1, device_os = $2, device_type = $3, updated_at = $4, update_id = $5
		WHERE id = $6`,
		appVersion, deviceOS, deviceType, now, uuid.Must(uuid.NewV7()), id,
	)

	return err
}

func (repo *Repository) SetPinExpiresAt(ctx context.Context, id uuid.UUID, at sql.Null[time.Time], now time.Time) error {
	_, err := database.Exec(ctx, repo.db, `UPDATE sessions
		SET pin_expires_at = $1, updated_at = $2, update_id = $3
		WHERE id = $4`,
		at, now, uuid.Must(uuid.NewV7()), id,
	)

	return err
}

func (repo *Repository) SetPendingSyncReset(ctx context.Context, id uuid.UUID, pending bool, now time.Time) (*session.Session, error) {
	return database.Row[session.Session](ctx, repo.db, `UPDATE sessions
		SET is_pending_sync_reset = $1, updated_at = $2, update_id = $3
		WHERE id = $4
		RETURNING `+columns,
		pending, now, uuid.Must(uuid.NewV7()), id,
	)
}

func (repo *Repository) LockAll(ctx context.Context, userID uuid.UUID, now time.Time) error {
	_, err := database.Exec(ctx, repo.db, `UPDATE sessions
		SET pin_expires_at = NULL, updated_at = $1, update_id = $2
		WHERE user_id = $3`,
		now, uuid.Must(uuid.NewV7()), userID,
	)

	return err
}

func (repo *Repository) ResetSync(ctx context.Context, id uuid.UUID, now time.Time) error {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.End(ctx)

	if _, err = database.Exec(ctx, tx, `UPDATE sessions
		SET is_pending_sync_reset = 0, updated_at = $1, update_id = $2
		WHERE id = $3`,
		now, uuid.Must(uuid.NewV7()), id,
	); err != nil {
		return err
	}
	if _, err = database.Exec(ctx, tx, `DELETE FROM sync_checkpoints WHERE session_id = $1`, id); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
