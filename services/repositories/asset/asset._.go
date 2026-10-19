package asset

import (
	"strings"

	"xn--gckvb8fzb.com/glides/services/database"
)

const columns = `id, owner_id, type, original_path, original_file_name, checksum, checksum_algorithm,
	file_created_at, file_modified_at, local_date_time, is_favorite, duration, live_photo_video_id,
	thumbhash, visibility, status, deleted_at, stack_id, library_id, is_offline, is_external,
	is_edited, duplicate_id, width, height, created_at, updated_at, update_id`

func prefixed(alias string) string {
	parts := strings.Split(columns, ",")
	for i, p := range parts {
		parts[i] = alias + "." + strings.TrimSpace(p)
	}

	return strings.Join(parts, ", ")
}

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
