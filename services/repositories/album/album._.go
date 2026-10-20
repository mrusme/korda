package album

import (
	"xn--gckvb8fzb.com/glides/services/database"
)

const owned = `SELECT album_id FROM album_users WHERE user_id = $1 AND role = 'owner'`

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
