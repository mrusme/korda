package asset

import (
	"context"
	"database/sql"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/asset"
	"xn--gckvb8fzb.com/glides/services/database"
)

func (repo *Repository) Get(ctx context.Context, id uuid.UUID) (*asset.Asset, error) {
	return database.Row[asset.Asset](ctx, repo.db, `SELECT `+columns+` FROM assets WHERE id = $1`, id)
}

func (repo *Repository) ByChecksum(ctx context.Context, ownerID uuid.UUID, checksum []byte) (*asset.Asset, error) {
	return database.Row[asset.Asset](ctx, repo.db, `SELECT `+columns+` FROM assets
		WHERE owner_id = $1 AND checksum = $2 AND library_id IS NULL`,
		ownerID, checksum,
	)
}

const chunkSize = 1000

func placeholders[T any](args *[]any, values []T) string {
	marks := make([]string, 0, len(values))
	for _, v := range values {
		*args = append(*args, v)
		marks = append(marks, "$"+strconv.Itoa(len(*args)))
	}

	return strings.Join(marks, ", ")
}

func (repo *Repository) ByChecksums(ctx context.Context, ownerID uuid.UUID, checksums [][]byte) ([]asset.Asset, error) {
	out := []asset.Asset{}
	for chunk := range slices.Chunk(checksums, chunkSize) {
		args := []any{ownerID}
		marks := placeholders(&args, chunk)

		found, err := database.Rows[asset.Asset](ctx, repo.db, `SELECT `+columns+` FROM assets
			WHERE owner_id = $1 AND checksum IN (`+marks+`)`,
			args...,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}

	return out, nil
}

func (repo *Repository) OwnedBy(ctx context.Context, ownerID uuid.UUID, ids []uuid.UUID, elevated bool) ([]uuid.UUID, error) {
	out := []uuid.UUID{}
	for chunk := range slices.Chunk(ids, chunkSize) {
		args := []any{ownerID}
		marks := placeholders(&args, chunk)

		query := `SELECT id FROM assets WHERE owner_id = $1 AND id IN (` + marks + `)`
		if !elevated {
			query += ` AND visibility != 'locked'`
		}

		found, err := database.Values[uuid.UUID](ctx, repo.db, query, args...)
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}

	return out, nil
}

type ForThumbnail struct {
	OriginalPath     string
	OriginalFileName string
	Path             sql.Null[string]
}

func (repo *Repository) ForThumbnail(ctx context.Context, id uuid.UUID, kind string, edited bool) (*ForThumbnail, error) {
	order := "ASC"
	if edited {
		order = "DESC"
	}

	return database.Row[ForThumbnail](ctx, repo.db, `SELECT a.original_path, a.original_file_name, f.path
		FROM assets a LEFT JOIN asset_files f ON f.asset_id = a.id AND f.type = $2
		WHERE a.id = $1 ORDER BY f.is_edited `+order+` LIMIT 1`,
		id, kind,
	)
}

type ForOriginal struct {
	OriginalPath     string
	OriginalFileName string
	EditedPath       sql.Null[string]
}

func (repo *Repository) ForOriginal(ctx context.Context, id uuid.UUID, edited bool) (*ForOriginal, error) {
	edit := `NULL`
	if edited {
		edit = `(SELECT path FROM asset_files WHERE asset_id = a.id AND is_edited = 1 AND type = 'fullsize' LIMIT 1)`
	}

	return database.Row[ForOriginal](ctx, repo.db, `SELECT a.original_path, a.original_file_name, `+edit+` AS edited_path
		FROM assets a WHERE a.id = $1`,
		id,
	)
}

type ForVideo struct {
	OriginalPath     string
	EncodedVideoPath sql.Null[string]
}

func (repo *Repository) ForVideo(ctx context.Context, id uuid.UUID) (*ForVideo, error) {
	return database.Row[ForVideo](ctx, repo.db, `SELECT a.original_path,
		(SELECT path FROM asset_files WHERE asset_id = a.id AND type = 'encoded_video' AND is_edited = 0) AS encoded_video_path
		FROM assets a WHERE a.id = $1 AND a.type = 'VIDEO'`,
		id,
	)
}
