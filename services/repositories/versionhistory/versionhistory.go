package versionhistory

import (
	"context"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/versionhistory"
	"xn--gckvb8fzb.com/glides/services/database"
)

const columns = `id, created_at, version`

func (repo *Repository) List(ctx context.Context) ([]versionhistory.VersionHistory, error) {
	return database.Rows[versionhistory.VersionHistory](ctx, repo.db, `SELECT `+columns+` FROM version_history
		ORDER BY created_at DESC`,
	)
}

func (repo *Repository) Latest(ctx context.Context) (*versionhistory.VersionHistory, error) {
	return database.Row[versionhistory.VersionHistory](ctx, repo.db, `SELECT `+columns+` FROM version_history
		ORDER BY created_at DESC LIMIT 1`,
	)
}

func (repo *Repository) Create(ctx context.Context, version string, now time.Time) error {
	_, err := database.Exec(ctx, repo.db, `INSERT INTO version_history (`+columns+`) VALUES ($1, $2, $3)`,
		uuid.New(), now, version,
	)

	return err
}
