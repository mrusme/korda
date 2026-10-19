package user

import (
	"xn--gckvb8fzb.com/glides/services/database"
)

const columns = `id, email, password, pin_code, name, is_admin, should_change_password,
	avatar_color, profile_image_path, profile_changed_at, storage_label,
	quota_size_in_bytes, quota_usage_in_bytes, status, oauth_id, cluster_group_id,
	created_at, updated_at, deleted_at, update_id`

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
