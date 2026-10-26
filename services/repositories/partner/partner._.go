package partner

import (
	"context"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/partner"
	"tty.fail/mrus/korda/services/repositories/query"
	"xn--gckvb8fzb.com/glides/services/database"
)

const columns = `p.shared_by_id, p.shared_with_id, p.in_timeline, p.create_id, p.created_at, p.updated_at, p.update_id`

const returning = `shared_by_id, shared_with_id, in_timeline, create_id, created_at, updated_at, update_id`

const active = `FROM partners p
	JOIN users b ON b.id = p.shared_by_id AND b.deleted_at IS NULL
	JOIN users w ON w.id = p.shared_with_id AND w.deleted_at IS NULL`

type Repository struct {
	db *database.Database
}

func New(db *database.Database) (*Repository, error) {
	repo := new(Repository)
	repo.db = db

	return repo, nil
}

func (repo *Repository) Startup() error {
	return nil
}

func (repo *Repository) Shutdown() error {
	return nil
}

func (repo *Repository) Get(ctx context.Context, sharedByID uuid.UUID, sharedWithID uuid.UUID) (*partner.Partner, error) {
	return database.Row[partner.Partner](ctx, repo.db, `SELECT `+columns+` `+active+`
		WHERE p.shared_by_id = $1 AND p.shared_with_id = $2`,
		sharedByID, sharedWithID,
	)
}

func (repo *Repository) All(ctx context.Context, userID uuid.UUID) ([]partner.Partner, error) {
	return database.Rows[partner.Partner](ctx, repo.db, `SELECT `+columns+` `+active+`
		WHERE p.shared_with_id = $1 OR p.shared_by_id = $1
		ORDER BY p.update_id`,
		userID,
	)
}

func (repo *Repository) Create(ctx context.Context, sharedByID uuid.UUID, sharedWithID uuid.UUID, now time.Time) (*partner.Partner, error) {
	return database.Row[partner.Partner](ctx, repo.db, `INSERT INTO partners (shared_by_id, shared_with_id, in_timeline,
		create_id, created_at, updated_at, update_id)
		VALUES ($1, $2, 0, $3, $4, $4, $5)
		RETURNING `+returning,
		sharedByID, sharedWithID, uuid.Must(uuid.NewV7()), now, uuid.Must(uuid.NewV7()),
	)
}

func (repo *Repository) Update(ctx context.Context, sharedByID uuid.UUID, sharedWithID uuid.UUID, inTimeline bool, now time.Time) (*partner.Partner, error) {
	return database.Row[partner.Partner](ctx, repo.db, `UPDATE partners SET in_timeline = $1, updated_at = $2, update_id = $3
		WHERE shared_by_id = $4 AND shared_with_id = $5
		RETURNING `+returning,
		inTimeline, now, uuid.Must(uuid.NewV7()), sharedByID, sharedWithID,
	)
}

func (repo *Repository) Remove(ctx context.Context, sharedByID uuid.UUID, sharedWithID uuid.UUID, now time.Time) error {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.End(ctx)

	res, err := database.Exec(ctx, tx, `DELETE FROM partners WHERE shared_by_id = $1 AND shared_with_id = $2`,
		sharedByID, sharedWithID)
	if err != nil {
		return err
	}
	if res.RowsAffected() > 0 {
		if _, err = database.Exec(ctx, tx, `INSERT INTO partner_audit (id, shared_by_id, shared_with_id, deleted_at)
			VALUES ($1, $2, $3, $4)`,
			uuid.Must(uuid.NewV7()), sharedByID, sharedWithID, now,
		); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (repo *Repository) SharingWith(ctx context.Context, userID uuid.UUID, sharedByIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(sharedByIDs) == 0 {
		return nil, nil
	}

	args := []any{userID}
	in := query.Placeholders(&args, sharedByIDs)

	return database.Values[uuid.UUID](ctx, repo.db, `SELECT shared_by_id FROM partners
		WHERE shared_with_id = $1 AND shared_by_id IN (`+in+`)`, args...)
}

func (repo *Repository) InTimeline(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return database.Values[uuid.UUID](ctx, repo.db, `SELECT p.shared_by_id `+active+`
		WHERE p.shared_with_id = $1 AND p.in_timeline = 1
		ORDER BY p.update_id`, userID)
}

func (repo *Repository) SharingWithMe(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return database.Values[uuid.UUID](ctx, repo.db, `SELECT p.shared_by_id `+active+`
		WHERE p.shared_with_id = $1 ORDER BY p.update_id`, userID)
}
