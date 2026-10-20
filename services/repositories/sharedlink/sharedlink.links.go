package sharedlink

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/models/sharedlink"
	"tty.fail/mrus/korda/services/repositories/query"
	"xn--gckvb8fzb.com/glides/services/database"
)

const assetColumns = `a.id, a.owner_id, a.type, a.original_path, a.original_file_name, a.checksum, a.checksum_algorithm,
	a.file_created_at, a.file_modified_at, a.local_date_time, a.is_favorite, a.duration, a.live_photo_video_id,
	a.thumbhash, a.visibility, a.status, a.deleted_at, a.stack_id, a.library_id, a.is_offline, a.is_external,
	a.is_edited, a.duplicate_id, a.width, a.height, a.created_at, a.updated_at, a.update_id`

const hasExif = `EXISTS (SELECT 1 FROM asset_exif e WHERE e.asset_id = a.id)`

var ErrEmptyList = errors.New(`syntax error at or near ")"`)

type Filter struct {
	ID      uuid.NullUUID
	AlbumID uuid.NullUUID
}

type Update struct {
	Description   optional.Value[string]
	Password      optional.Value[string]
	ExpiresAt     optional.Value[time.Time]
	AllowUpload   *bool
	AllowDownload *bool
	ShowExif      *bool
	Slug          sql.Null[string]
}

func (repo *Repository) Get(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*sharedlink.SharedLink, error) {
	return database.Row[sharedlink.SharedLink](ctx, repo.db, `SELECT `+prefixed+` `+authenticating+`
		AND sl.id = $1 AND sl.user_id = $2`, id, userID)
}

func (repo *Repository) All(ctx context.Context, userID uuid.UUID, f Filter) ([]sharedlink.SharedLink, error) {
	args := []any{userID}
	where := ""
	if f.AlbumID.Valid {
		args = append(args, f.AlbumID.UUID)
		where += ` AND sl.album_id = $` + strconv.Itoa(len(args))
	}
	if f.ID.Valid {
		args = append(args, f.ID.UUID)
		where += ` AND sl.id = $` + strconv.Itoa(len(args))
	}

	return database.Rows[sharedlink.SharedLink](ctx, repo.db, `SELECT `+prefixed+` `+authenticating+`
		AND sl.user_id = $1`+where+` ORDER BY sl.created_at DESC`, args...)
}

func (repo *Repository) Create(ctx context.Context, l *sharedlink.SharedLink, assetIDs []uuid.UUID) error {
	if _, err := database.Exec(ctx, repo.db, `INSERT INTO shared_links (`+columns+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		l.ID, l.Description, l.UserID, l.Key, l.Type, l.CreatedAt, l.ExpiresAt, l.AllowUpload, l.AlbumID,
		l.AllowDownload, l.ShowExif, l.Password, l.Slug,
	); err != nil {
		return err
	}

	return repo.InsertAssets(ctx, l.ID, assetIDs)
}

func (repo *Repository) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, u Update) error {
	sets := []string{}
	args := []any{}
	set := func(column string, v any) {
		args = append(args, v)
		sets = append(sets, column+" = $"+strconv.Itoa(len(args)))
	}

	set("user_id", userID)
	if u.Description.Set {
		set("description", sql.Null[string]{V: u.Description.V, Valid: !u.Description.Null})
	}
	if u.Password.Set {
		set("password", sql.Null[string]{V: u.Password.V, Valid: !u.Password.Null})
	}
	if u.ExpiresAt.Set {
		set("expires_at", sql.Null[time.Time]{V: u.ExpiresAt.V.UTC(), Valid: !u.ExpiresAt.Null})
	}
	if u.AllowUpload != nil {
		set("allow_upload", *u.AllowUpload)
	}
	if u.AllowDownload != nil {
		set("allow_download", *u.AllowDownload)
	}
	if u.ShowExif != nil {
		set("show_exif", *u.ShowExif)
	}
	set("slug", u.Slug)
	args = append(args, id)

	_, err := database.Exec(ctx, repo.db, `UPDATE shared_links SET `+strings.Join(sets, ", ")+`
		WHERE id = $`+strconv.Itoa(len(args)), args...)

	return err
}

func (repo *Repository) Remove(ctx context.Context, id uuid.UUID) error {
	_, err := database.Exec(ctx, repo.db, `DELETE FROM shared_links WHERE id = $1`, id)
	return err
}

func (repo *Repository) InsertAssets(ctx context.Context, id uuid.UUID, assetIDs []uuid.UUID) error {
	if len(assetIDs) == 0 {
		return nil
	}

	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.End(ctx)

	for chunk := range slices.Chunk(assetIDs, query.ChunkSize) {
		args := []any{id}
		rows := make([]string, 0, len(chunk))
		for _, assetID := range chunk {
			args = append(args, assetID)
			rows = append(rows, "($"+strconv.Itoa(len(args))+", $1)")
		}
		if _, err = database.Exec(ctx, tx, `INSERT INTO shared_link_assets (asset_id, shared_link_id)
			VALUES `+strings.Join(rows, ", "), args...); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (repo *Repository) AttachAsset(ctx context.Context, id uuid.UUID, assetID uuid.UUID) error {
	_, err := database.Exec(ctx, repo.db, `INSERT INTO shared_link_assets (asset_id, shared_link_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING`, assetID, id)
	return err
}

func (repo *Repository) RemoveAssets(ctx context.Context, id uuid.UUID, assetIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(assetIDs) == 0 {
		return nil, ErrEmptyList
	}

	out := []uuid.UUID{}
	for chunk := range slices.Chunk(assetIDs, query.ChunkSize) {
		args := []any{id}
		marks := query.Placeholders(&args, chunk)
		removed, err := database.Values[uuid.UUID](ctx, repo.db, `DELETE FROM shared_link_assets
			WHERE shared_link_id = $1 AND asset_id IN (`+marks+`) RETURNING asset_id`, args...)
		if err != nil {
			return nil, err
		}
		out = append(out, removed...)
	}

	return out, nil
}

func (repo *Repository) Assets(ctx context.Context, id uuid.UUID) ([]asset.Asset, error) {
	return database.Rows[asset.Asset](ctx, repo.db, `SELECT `+assetColumns+`
		FROM shared_link_assets sla JOIN assets a ON a.id = sla.asset_id
		WHERE sla.shared_link_id = $1 AND a.deleted_at IS NULL AND `+hasExif+`
		ORDER BY a.file_created_at, sla.rowid`, id)
}

func (repo *Repository) Cover(ctx context.Context, id uuid.UUID) ([]asset.Asset, error) {
	return database.Rows[asset.Asset](ctx, repo.db, `SELECT `+assetColumns+`
		FROM shared_link_assets sla JOIN assets a ON a.id = sla.asset_id
		WHERE sla.shared_link_id = $1 AND a.deleted_at IS NULL
		ORDER BY a.file_created_at, sla.rowid LIMIT 1`, id)
}

func (repo *Repository) Linked(ctx context.Context, id uuid.UUID) ([]asset.Asset, error) {
	return database.Rows[asset.Asset](ctx, repo.db, `SELECT `+assetColumns+`
		FROM shared_link_assets sla JOIN assets a ON a.id = sla.asset_id
		WHERE sla.shared_link_id = $1 AND `+hasExif+`
		ORDER BY a.update_id`, id)
}

func (repo *Repository) AlbumAssets(ctx context.Context, albumID uuid.UUID) ([]asset.Asset, error) {
	return database.Rows[asset.Asset](ctx, repo.db, `SELECT `+assetColumns+`
		FROM album_assets aa JOIN assets a ON a.id = aa.asset_id
		WHERE aa.album_id = $1 AND a.deleted_at IS NULL AND `+hasExif+`
		ORDER BY a.file_created_at, aa.rowid`, albumID)
}

func (repo *Repository) ByID(ctx context.Context, id uuid.UUID) (*sharedlink.SharedLink, error) {
	return database.Row[sharedlink.SharedLink](ctx, repo.db, `SELECT `+columns+` FROM shared_links WHERE id = $1`, id)
}

func (repo *Repository) CopyAssets(ctx context.Context, sourceID uuid.UUID, targetID uuid.UUID) error {
	_, err := database.Exec(ctx, repo.db, `INSERT INTO shared_link_assets (asset_id, shared_link_id)
		SELECT $1, shared_link_id FROM shared_link_assets WHERE asset_id = $2 ORDER BY rowid
		ON CONFLICT DO NOTHING`, targetID, sourceID)
	return err
}
