package album

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/models/album"
	"tty.fail/mrus/korda/services/repositories/syncstate"
	"xn--gckvb8fzb.com/glides/services/database"
)

type NewMember struct {
	UserID uuid.UUID
	Role   album.Role
}

type Create struct {
	Name        string
	Description sql.Null[string]
	Thumbnail   uuid.NullUUID
	Order       album.Order
	Members     []NewMember
	AssetIDs    []uuid.UUID
}

type Update struct {
	Name              *string
	Description       optional.Value[string]
	Thumbnail         uuid.NullUUID
	IsActivityEnabled *bool
	Order             *album.Order
}

type Pair struct {
	AlbumID uuid.UUID
	AssetID uuid.UUID
}

func insertUser(ctx context.Context, ex database.Executor, albumID uuid.UUID, m NewMember, now time.Time) error {
	_, err := database.Exec(ctx, ex, `INSERT INTO album_users (album_id, user_id, role, create_id, created_at, updated_at,
		update_id) VALUES ($1, $2, $3, $4, $5, $5, $6)`,
		albumID, m.UserID, m.Role, uuid.Must(uuid.NewV7()), now, uuid.Must(uuid.NewV7()),
	)

	return err
}

func insertAsset(ctx context.Context, ex database.Executor, albumID uuid.UUID, assetID uuid.UUID, now time.Time) error {
	_, err := database.Exec(ctx, ex, `INSERT INTO album_assets (album_id, asset_id, created_at, updated_at, update_id)
		VALUES ($1, $2, $3, $3, $4) ON CONFLICT DO NOTHING`,
		albumID, assetID, now, uuid.Must(uuid.NewV7()),
	)

	return err
}

func (repo *Repository) Create(ctx context.Context, c Create, now time.Time) (*album.Album, error) {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.End(ctx)

	created, err := database.Row[album.Album](ctx, tx, `INSERT INTO albums (`+columns+`)
		VALUES ($1, $2, $3, $4, 1, $5, $6, $6, NULL, $7)
		RETURNING `+columns,
		uuid.New(), c.Name, c.Description, c.Thumbnail, c.Order, now, uuid.Must(uuid.NewV7()),
	)
	if err != nil {
		return nil, err
	}
	for _, m := range c.Members {
		if err = insertUser(ctx, tx, created.ID, m, now); err != nil {
			return nil, err
		}
	}
	for _, id := range c.AssetIDs {
		if err = insertAsset(ctx, tx, created.ID, id, now); err != nil {
			return nil, err
		}
	}

	return created, tx.Commit(ctx)
}

func (repo *Repository) Update(ctx context.Context, id uuid.UUID, u Update, now time.Time) (*album.Album, error) {
	args := []any{now, uuid.Must(uuid.NewV7())}
	set := []string{"updated_at = $1", "update_id = $2"}
	add := func(column string, value any) {
		args = append(args, value)
		set = append(set, column+" = $"+strconv.Itoa(len(args)))
	}
	if u.Name != nil {
		add("album_name", *u.Name)
	}
	if u.Description.Set {
		var v any
		if !u.Description.Null {
			v = u.Description.V
		}
		add("description", v)
	}
	if u.Thumbnail.Valid {
		add("album_thumbnail_asset_id", u.Thumbnail.UUID)
	}
	if u.IsActivityEnabled != nil {
		add("is_activity_enabled", *u.IsActivityEnabled)
	}
	if u.Order != nil {
		add(`"order"`, *u.Order)
	}
	args = append(args, id)

	return database.Row[album.Album](ctx, repo.db, `UPDATE albums SET `+strings.Join(set, ", ")+`
		WHERE id = $`+strconv.Itoa(len(args))+` RETURNING `+columns, args...)
}

func (repo *Repository) Touch(ctx context.Context, id uuid.UUID, thumbnail uuid.NullUUID, now time.Time) error {
	_, err := database.Exec(ctx, repo.db, `UPDATE albums SET album_thumbnail_asset_id = $1, updated_at = $2, update_id = $3
		WHERE id = $4`, thumbnail, now, uuid.Must(uuid.NewV7()), id)

	return err
}

func (repo *Repository) AddAssets(ctx context.Context, pairs []Pair, now time.Time) error {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.End(ctx)

	for _, p := range pairs {
		if err = insertAsset(ctx, tx, p.AlbumID, p.AssetID, now); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (repo *Repository) RemoveAssets(ctx context.Context, albumID uuid.UUID, assetIDs []uuid.UUID, now time.Time) error {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.End(ctx)

	for _, id := range assetIDs {
		if err = syncstate.Audit(ctx, tx, "album_asset_audit", "album_id, asset_id", `SELECT album_id AS a, asset_id AS b
			FROM album_assets WHERE album_id = $1 AND asset_id = $2`, now, albumID, id); err != nil {
			return err
		}
		if _, err = database.Exec(ctx, tx, `DELETE FROM album_assets WHERE album_id = $1 AND asset_id = $2`,
			albumID, id); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (repo *Repository) Delete(ctx context.Context, id uuid.UUID, now time.Time) error {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.End(ctx)

	members := `SELECT album_id AS a, user_id AS b FROM album_users WHERE album_id = $1`
	if err = syncstate.Audit(ctx, tx, "album_audit", "album_id, user_id", members, now, id); err != nil {
		return err
	}
	if err = syncstate.Audit(ctx, tx, "album_user_audit", "album_id, user_id", members, now, id); err != nil {
		return err
	}
	if _, err = database.Exec(ctx, tx, `DELETE FROM albums WHERE id = $1`, id); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

const liveAssets = `SELECT aa.asset_id FROM album_assets aa
	JOIN assets a ON a.id = aa.asset_id AND a.deleted_at IS NULL
	WHERE aa.album_id = al.id`

func (repo *Repository) UpdateThumbnails(ctx context.Context, now time.Time) error {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.End(ctx)

	ids, err := database.Values[uuid.UUID](ctx, tx, `SELECT al.id FROM albums al
		WHERE (al.album_thumbnail_asset_id IS NULL AND EXISTS (`+liveAssets+`))
			OR (al.album_thumbnail_asset_id IS NOT NULL
				AND NOT EXISTS (`+liveAssets+` AND aa.asset_id = al.album_thumbnail_asset_id))`)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if _, err = database.Exec(ctx, tx, `UPDATE albums SET
			album_thumbnail_asset_id = (SELECT aa.asset_id FROM album_assets aa
				JOIN assets a ON a.id = aa.asset_id AND a.deleted_at IS NULL
				WHERE aa.album_id = $1 ORDER BY a.file_created_at DESC, aa.rowid LIMIT 1),
			updated_at = $2, update_id = $3
			WHERE id = $1`, id, now, uuid.Must(uuid.NewV7())); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (repo *Repository) AddUser(ctx context.Context, albumID uuid.UUID, m NewMember, now time.Time) error {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.End(ctx)

	if err = insertUser(ctx, tx, albumID, m, now); err != nil {
		return err
	}
	if err = syncstate.Bump(ctx, tx, "albums", "", []uuid.UUID{albumID}, now); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (repo *Repository) UpdateRole(ctx context.Context, albumID uuid.UUID, userID uuid.UUID, role album.Role, now time.Time) error {
	_, err := database.Exec(ctx, repo.db, `UPDATE album_users SET role = $1, updated_at = $2, update_id = $3
		WHERE album_id = $4 AND user_id = $5`,
		role, now, uuid.Must(uuid.NewV7()), albumID, userID,
	)

	return err
}

func (repo *Repository) RemoveUser(ctx context.Context, albumID uuid.UUID, userID uuid.UUID, now time.Time) error {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.End(ctx)

	member := `SELECT album_id AS a, user_id AS b FROM album_users WHERE album_id = $1 AND user_id = $2`
	if err = syncstate.Audit(ctx, tx, "album_audit", "album_id, user_id", member, now, albumID, userID); err != nil {
		return err
	}
	if err = syncstate.Audit(ctx, tx, "album_user_audit", "album_id, user_id", member, now, albumID, userID); err != nil {
		return err
	}
	if _, err = database.Exec(ctx, tx, `DELETE FROM album_users WHERE album_id = $1 AND user_id = $2`,
		albumID, userID); err != nil {
		return err
	}
	if _, err = database.Exec(ctx, tx, `DELETE FROM albums WHERE id = $1
		AND NOT EXISTS (SELECT 1 FROM album_users WHERE album_id = $1 AND role = 'owner')`, albumID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (repo *Repository) CopyAssets(ctx context.Context, sourceID uuid.UUID, targetID uuid.UUID, now time.Time) error {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.End(ctx)

	albums, err := database.Values[uuid.UUID](ctx, tx, `SELECT album_id FROM album_assets WHERE asset_id = $1
		ORDER BY rowid`, sourceID)
	if err != nil {
		return err
	}
	for _, albumID := range albums {
		if err = insertAsset(ctx, tx, albumID, targetID, now); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
