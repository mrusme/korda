package session

import (
	"xn--gckvb8fzb.com/glides/services/database"
)

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

const columns = `id, token, user_id, parent_id, device_type, device_os, app_version,
	expires_at, pin_expires_at, is_pending_sync_reset, created_at, updated_at, update_id`

const activeColumns = `s.id AS id, s.token AS token, s.user_id AS user_id, s.parent_id AS parent_id,
	s.device_type AS device_type, s.device_os AS device_os, s.app_version AS app_version,
	s.expires_at AS expires_at, s.pin_expires_at AS pin_expires_at,
	s.is_pending_sync_reset AS is_pending_sync_reset, s.created_at AS created_at,
	s.updated_at AS updated_at, s.update_id AS update_id`
