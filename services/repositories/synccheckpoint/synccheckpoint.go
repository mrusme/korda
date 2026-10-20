package synccheckpoint

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/synccheckpoint"
	"xn--gckvb8fzb.com/glides/services/database"
)

const columns = `session_id, type, ack, created_at, updated_at, update_id`

func (repo *Repository) List(ctx context.Context, sessionID uuid.UUID) ([]synccheckpoint.Checkpoint, error) {
	return database.Rows[synccheckpoint.Checkpoint](ctx, repo.db, `SELECT `+columns+` FROM sync_checkpoints
		WHERE session_id = $1
		ORDER BY update_id`,
		sessionID,
	)
}

func (repo *Repository) Upsert(ctx context.Context, sessionID uuid.UUID, checkpoints []synccheckpoint.Checkpoint, now time.Time) error {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.End(ctx)

	for _, c := range checkpoints {
		if _, err = database.Exec(ctx, tx, `INSERT INTO sync_checkpoints (`+columns+`)
			VALUES ($1, $2, $3, $4, $4, $5)
			ON CONFLICT (session_id, type) DO UPDATE SET
				ack = excluded.ack,
				updated_at = excluded.updated_at,
				update_id = excluded.update_id`,
			sessionID, c.Type, c.Ack, now, uuid.Must(uuid.NewV7()),
		); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (repo *Repository) Delete(ctx context.Context, sessionID uuid.UUID, types []string) error {
	if types == nil {
		_, err := database.Exec(ctx, repo.db, `DELETE FROM sync_checkpoints WHERE session_id = $1`, sessionID)
		return err
	}
	if len(types) == 0 {
		return nil
	}

	args := []any{sessionID}
	marks := make([]string, 0, len(types))
	for _, t := range types {
		args = append(args, t)
		marks = append(marks, "$"+strconv.Itoa(len(args)))
	}
	_, err := database.Exec(ctx, repo.db, `DELETE FROM sync_checkpoints
		WHERE session_id = $1 AND type IN (`+strings.Join(marks, ", ")+`)`,
		args...,
	)

	return err
}
