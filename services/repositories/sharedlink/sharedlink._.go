package sharedlink

import (
	"context"
	"slices"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/sharedlink"
	"tty.fail/mrus/korda/services/repositories/query"
	"xn--gckvb8fzb.com/glides/services/database"
)

const columns = `id, description, user_id, key, type, created_at, expires_at, allow_upload, album_id, allow_download,
	show_exif, password, slug`

const prefixed = `sl.id, sl.description, sl.user_id, sl.key, sl.type, sl.created_at, sl.expires_at, sl.allow_upload,
	sl.album_id, sl.allow_download, sl.show_exif, sl.password, sl.slug`

const authenticating = `FROM shared_links sl LEFT JOIN albums al ON al.id = sl.album_id
	WHERE al.deleted_at IS NULL AND (sl.type = 'INDIVIDUAL' OR al.id IS NOT NULL)`

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

func (repo *Repository) ByKey(ctx context.Context, key []byte) (*sharedlink.SharedLink, error) {
	return database.Row[sharedlink.SharedLink](ctx, repo.db, `SELECT `+prefixed+` `+authenticating+` AND sl.key = $1`, key)
}

func (repo *Repository) BySlug(ctx context.Context, slug string) (*sharedlink.SharedLink, error) {
	return database.Row[sharedlink.SharedLink](ctx, repo.db, `SELECT `+prefixed+` `+authenticating+` AND sl.slug = $1`, slug)
}

type accessRow struct {
	AssetID        uuid.NullUUID
	AssetLive      uuid.NullUUID
	AlbumAssetID   uuid.NullUUID
	AlbumAssetLive uuid.NullUUID
}

func (repo *Repository) AssetAccess(ctx context.Context, linkID uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	out := []uuid.UUID{}
	for chunk := range slices.Chunk(ids, query.ChunkSize) {
		wanted := make(map[uuid.UUID]bool, len(chunk))
		for _, id := range chunk {
			wanted[id] = true
		}

		args := []any{linkID}
		marks := query.Placeholders(&args, chunk)
		rows, err := database.Rows[accessRow](ctx, repo.db, `SELECT a.id AS asset_id, a.live_photo_video_id AS asset_live,
			b.id AS album_asset_id, b.live_photo_video_id AS album_asset_live
			FROM shared_links sl
			LEFT JOIN albums al ON al.id = sl.album_id AND al.deleted_at IS NULL
			LEFT JOIN shared_link_assets sla ON sla.shared_link_id = sl.id
			LEFT JOIN assets a ON a.id = sla.asset_id AND a.deleted_at IS NULL
			LEFT JOIN album_assets aa ON aa.album_id = al.id
			LEFT JOIN assets b ON b.id = aa.asset_id AND b.deleted_at IS NULL
			WHERE sl.id = $1 AND (a.id IN (`+marks+`) OR a.live_photo_video_id IN (`+marks+`)
				OR b.id IN (`+marks+`) OR b.live_photo_video_id IN (`+marks+`))`,
			args...,
		)
		if err != nil {
			return nil, err
		}

		seen := map[uuid.UUID]bool{}
		for _, r := range rows {
			for _, v := range []uuid.NullUUID{r.AssetID, r.AssetLive, r.AlbumAssetID, r.AlbumAssetLive} {
				if v.Valid && wanted[v.UUID] && !seen[v.UUID] {
					seen[v.UUID] = true
					out = append(out, v.UUID)
				}
			}
		}
	}

	return out, nil
}

func (repo *Repository) AlbumAccess(ctx context.Context, linkID uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	args := []any{linkID}
	marks := query.Placeholders(&args, ids)

	return database.Values[uuid.UUID](ctx, repo.db, `SELECT album_id FROM shared_links
		WHERE id = $1 AND album_id IS NOT NULL AND album_id IN (`+marks+`)`, args...)
}
