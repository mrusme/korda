package asset

import (
	"cmp"
	"context"
	"database/sql"
	"slices"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/services/repositories/query"
	"xn--gckvb8fzb.com/glides/services/database"
)

type Download struct {
	Position         int64
	ID               uuid.UUID
	LivePhotoVideoID uuid.NullUUID
	OriginalPath     string
	Size             sql.Null[int64]
}

const downloadColumns = `SELECT e.rowid AS position, a.id, a.live_photo_video_id, a.original_path, e.file_size_in_byte AS size
	FROM assets a JOIN asset_exif e ON e.asset_id = a.id`

func (repo *Repository) DownloadIDs(ctx context.Context, ids []uuid.UUID) ([]Download, error) {
	out := []Download{}
	for chunk := range slices.Chunk(ids, query.ChunkSize) {
		var args []any
		in := query.Placeholders(&args, chunk)
		found, err := database.Rows[Download](ctx, repo.db, downloadColumns+`
			WHERE a.deleted_at IS NULL AND a.id IN (`+in+`)`, args...)
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}
	slices.SortFunc(out, func(x, y Download) int { return cmp.Compare(x.Position, y.Position) })

	return out, nil
}

func (repo *Repository) DownloadAlbum(ctx context.Context, albumID uuid.UUID) ([]Download, error) {
	return database.Rows[Download](ctx, repo.db, downloadColumns+`
		JOIN album_assets x ON x.asset_id = a.id
		WHERE a.deleted_at IS NULL AND x.album_id = $1 ORDER BY x.rowid`, albumID)
}

func (repo *Repository) DownloadUser(ctx context.Context, userID uuid.UUID, elevated bool) ([]Download, error) {
	return database.Rows[Download](ctx, repo.db, downloadColumns+`
		WHERE a.deleted_at IS NULL AND a.owner_id = $1 AND a.visibility != 'hidden'
			AND ($2 OR a.visibility != 'locked') ORDER BY e.rowid`, userID, elevated)
}

type Original struct {
	ID               uuid.UUID
	OriginalFileName string
	OriginalPath     string
}

func (repo *Repository) Originals(ctx context.Context, ids []uuid.UUID) ([]Original, error) {
	out := []Original{}
	for chunk := range slices.Chunk(ids, query.ChunkSize) {
		var args []any
		in := query.Placeholders(&args, chunk)
		found, err := database.Rows[Original](ctx, repo.db, `SELECT id, original_file_name, original_path FROM assets
			WHERE id IN (`+in+`)`, args...)
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}

	return out, nil
}
