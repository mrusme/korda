package memory

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/models/memory"
	"tty.fail/mrus/korda/services/repositories/query"
	"tty.fail/mrus/korda/services/repositories/syncstate"
	"xn--gckvb8fzb.com/glides/services/database"
)

const columns = `id, created_at, updated_at, deleted_at, owner_id, type, data, is_saved, memory_at, seen_at, show_at,
	hide_at, update_id`

const assetColumns = `a.id, a.owner_id, a.type, a.original_path, a.original_file_name, a.checksum, a.checksum_algorithm,
	a.file_created_at, a.file_modified_at, a.local_date_time, a.is_favorite, a.duration, a.live_photo_video_id,
	a.thumbhash, a.visibility, a.status, a.deleted_at, a.stack_id, a.library_id, a.is_offline, a.is_external,
	a.is_edited, a.duplicate_id, a.width, a.height, a.created_at, a.updated_at, a.update_id`

type Repository struct {
	db *database.Database
}

type Filter struct {
	ID         uuid.NullUUID
	Type       *memory.Type
	For        *time.Time
	IsTrashed  bool
	IsSaved    *bool
	IsUpcoming *bool
	Now        time.Time
	Size       *int64
	Page       *int64
	Order      memory.Order
}

type Create struct {
	OwnerID  uuid.UUID
	Type     memory.Type
	Data     string
	IsSaved  bool
	MemoryAt time.Time
	SeenAt   sql.Null[time.Time]
	ShowAt   sql.Null[time.Time]
	HideAt   sql.Null[time.Time]
	AssetIDs []uuid.UUID
}

type Update struct {
	IsSaved  *bool
	SeenAt   *time.Time
	MemoryAt *time.Time
}

type MemberAsset struct {
	MemoryID uuid.UUID
	asset.Asset
}

type YearAssets struct {
	Year   int
	Assets []uuid.UUID
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

func (f Filter) where(ownerID uuid.UUID, args *[]any) string {
	*args = append(*args, ownerID)
	where := []string{`owner_id = $` + strconv.Itoa(len(*args))}
	if f.IsTrashed {
		where = append(where, `deleted_at IS NOT NULL`)
	} else {
		where = append(where, `deleted_at IS NULL`)
	}
	add := func(clause string, v any) {
		*args = append(*args, v)
		where = append(where, strings.ReplaceAll(clause, "?", "$"+strconv.Itoa(len(*args))))
	}
	if f.IsSaved != nil {
		add(`is_saved = ?`, *f.IsSaved)
	}
	if f.Type != nil {
		add(`type = ?`, string(*f.Type))
	}
	if f.For != nil {
		add(`(show_at IS NULL OR show_at <= ?)`, *f.For)
		add(`(hide_at IS NULL OR hide_at >= ?)`, *f.For)
	}
	if f.IsUpcoming != nil {
		if *f.IsUpcoming {
			add(`show_at > ?`, f.Now)
		} else {
			add(`(show_at IS NULL OR show_at <= ?)`, f.Now)
		}
	}

	return strings.Join(where, " AND ")
}

func (repo *Repository) Search(ctx context.Context, ownerID uuid.UUID, f Filter) ([]memory.Memory, error) {
	args := []any{}
	where := f.where(ownerID, &args)
	if f.ID.Valid {
		args = append(args, f.ID.UUID)
		where += ` AND id = $` + strconv.Itoa(len(args))
	}

	order := ` ORDER BY show_at IS NULL, show_at DESC, memory_at DESC, rowid`
	switch f.Order {
	case memory.OrderRandom:
		order = ` ORDER BY random()`
	case memory.OrderAsc:
		order = ` ORDER BY show_at IS NULL, show_at ASC, memory_at ASC, rowid`
	}
	if f.Size != nil {
		order += ` LIMIT ` + strconv.FormatInt(*f.Size, 10)
		if f.Page != nil {
			order += ` OFFSET ` + strconv.FormatInt((*f.Page-1)**f.Size, 10)
		}
	}

	return database.Rows[memory.Memory](ctx, repo.db, `SELECT `+columns+` FROM memories WHERE `+where+order, args...)
}

func (repo *Repository) Count(ctx context.Context, ownerID uuid.UUID, f Filter) (int64, error) {
	args := []any{}
	where := f.where(ownerID, &args)

	return database.Value[int64](ctx, repo.db, `SELECT count(*) FROM memories WHERE `+where, args...)
}

func (repo *Repository) Get(ctx context.Context, id uuid.UUID) (*memory.Memory, error) {
	return database.Row[memory.Memory](ctx, repo.db, `SELECT `+columns+` FROM memories WHERE id = $1 AND deleted_at IS NULL`, id)
}

func (repo *Repository) Assets(ctx context.Context, memoryIDs []uuid.UUID) (map[uuid.UUID][]asset.Asset, error) {
	out := map[uuid.UUID][]asset.Asset{}
	for chunk := range slices.Chunk(memoryIDs, query.ChunkSize) {
		args := []any{}
		marks := query.Placeholders(&args, chunk)
		rows, err := database.Rows[MemberAsset](ctx, repo.db, `SELECT ma.memory_id, `+assetColumns+`
			FROM memory_assets ma JOIN assets a ON a.id = ma.asset_id
			WHERE ma.memory_id IN (`+marks+`) AND a.visibility = 'timeline' AND a.deleted_at IS NULL
			ORDER BY a.file_created_at, ma.rowid`, args...)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			out[r.MemoryID] = append(out[r.MemoryID], r.Asset)
		}
	}

	return out, nil
}

func insertAssets(ctx context.Context, ex database.Executor, id uuid.UUID, assetIDs []uuid.UUID, now time.Time) error {
	for _, assetID := range assetIDs {
		if _, err := database.Exec(ctx, ex, `INSERT INTO memory_assets (memory_id, asset_id, created_at, updated_at, update_id)
			VALUES ($1, $2, $3, $3, $4)`, id, assetID, now, uuid.Must(uuid.NewV7())); err != nil {
			return err
		}
	}

	return nil
}

func (repo *Repository) Create(ctx context.Context, c Create, now time.Time) (uuid.UUID, error) {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.End(ctx)

	id := uuid.New()
	if _, err = database.Exec(ctx, tx, `INSERT INTO memories (`+columns+`)
		VALUES ($1, $2, $2, NULL, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		id, now, c.OwnerID, string(c.Type), c.Data, c.IsSaved, c.MemoryAt, c.SeenAt, c.ShowAt, c.HideAt,
		uuid.Must(uuid.NewV7()),
	); err != nil {
		return uuid.Nil, err
	}
	if err = insertAssets(ctx, tx, id, c.AssetIDs, now); err != nil {
		return uuid.Nil, err
	}

	return id, tx.Commit(ctx)
}

func (repo *Repository) Update(ctx context.Context, id uuid.UUID, u Update, now time.Time) error {
	args := []any{now, uuid.Must(uuid.NewV7())}
	set := `updated_at = $1, update_id = $2`
	if u.IsSaved != nil {
		args = append(args, *u.IsSaved)
		set += `, is_saved = $` + strconv.Itoa(len(args))
	}
	if u.SeenAt != nil {
		args = append(args, *u.SeenAt)
		set += `, seen_at = $` + strconv.Itoa(len(args))
	}
	if u.MemoryAt != nil {
		args = append(args, *u.MemoryAt)
		set += `, memory_at = $` + strconv.Itoa(len(args))
	}
	args = append(args, id)
	_, err := database.Exec(ctx, repo.db, `UPDATE memories SET `+set+` WHERE id = $`+strconv.Itoa(len(args)), args...)

	return err
}

func remove(ctx context.Context, ex database.Executor, ids []uuid.UUID, now time.Time) error {
	for _, id := range ids {
		if err := syncstate.Audit(ctx, ex, "memory_audit", "memory_id, user_id", `SELECT id AS a, owner_id AS b
			FROM memories WHERE id = $1`, now, id); err != nil {
			return err
		}
		if _, err := database.Exec(ctx, ex, `DELETE FROM memories WHERE id = $1`, id); err != nil {
			return err
		}
	}

	return nil
}

func (repo *Repository) Delete(ctx context.Context, id uuid.UUID, now time.Time) error {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.End(ctx)

	if err = remove(ctx, tx, []uuid.UUID{id}, now); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (repo *Repository) AssetIDs(ctx context.Context, id uuid.UUID, assetIDs []uuid.UUID) ([]uuid.UUID, error) {
	out := []uuid.UUID{}
	for chunk := range slices.Chunk(assetIDs, query.ChunkSize) {
		args := []any{id}
		marks := query.Placeholders(&args, chunk)
		found, err := database.Values[uuid.UUID](ctx, repo.db, `SELECT asset_id FROM memory_assets
			WHERE memory_id = $1 AND asset_id IN (`+marks+`)`, args...)
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}

	return out, nil
}

func (repo *Repository) AddAssets(ctx context.Context, id uuid.UUID, assetIDs []uuid.UUID, now time.Time) error {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.End(ctx)

	if err = insertAssets(ctx, tx, id, assetIDs, now); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func unlink(ctx context.Context, ex database.Executor, id uuid.UUID, assetID uuid.UUID, now time.Time) error {
	if err := syncstate.Audit(ctx, ex, "memory_asset_audit", "memory_id, asset_id", `SELECT memory_id AS a, asset_id AS b
		FROM memory_assets WHERE memory_id = $1 AND asset_id = $2`, now, id, assetID); err != nil {
		return err
	}
	_, err := database.Exec(ctx, ex, `DELETE FROM memory_assets WHERE memory_id = $1 AND asset_id = $2`, id, assetID)

	return err
}

func (repo *Repository) RemoveAssets(ctx context.Context, id uuid.UUID, assetIDs []uuid.UUID, now time.Time) error {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.End(ctx)

	for _, assetID := range assetIDs {
		if err = unlink(ctx, tx, id, assetID, now); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (repo *Repository) OwnedBy(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	out := []uuid.UUID{}
	for chunk := range slices.Chunk(ids, query.ChunkSize) {
		args := []any{userID}
		marks := query.Placeholders(&args, chunk)
		found, err := database.Values[uuid.UUID](ctx, repo.db, `SELECT id FROM memories
			WHERE owner_id = $1 AND id IN (`+marks+`)`, args...)
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}

	return out, nil
}

func (repo *Repository) Cleanup(ctx context.Context, before time.Time, now time.Time) error {
	tx, err := repo.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.End(ctx)

	type link struct {
		MemoryID uuid.UUID
		AssetID  uuid.UUID
	}
	links, err := database.Rows[link](ctx, tx, `SELECT ma.memory_id, ma.asset_id FROM memory_assets ma
		JOIN assets a ON a.id = ma.asset_id WHERE a.visibility <> 'timeline'`)
	if err != nil {
		return err
	}
	for _, l := range links {
		if err = unlink(ctx, tx, l.MemoryID, l.AssetID, now); err != nil {
			return err
		}
	}

	expired, err := database.Values[uuid.UUID](ctx, tx, `SELECT id FROM memories WHERE created_at < $1 AND is_saved = 0`, before)
	if err != nil {
		return err
	}
	if err = remove(ctx, tx, expired, now); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (repo *Repository) OnThisDay(ctx context.Context, ownerID uuid.UUID, month time.Month, day int, beforeYear int) ([]YearAssets, error) {
	first, err := database.Value[sql.Null[string]](ctx, repo.db, `SELECT min(date(local_date_time)) FROM assets`)
	if err != nil || !first.Valid {
		return nil, err
	}
	start, err := strconv.Atoi(first.V[:4])
	if err != nil {
		return nil, err
	}

	out := []YearAssets{}
	for year := beforeYear - 1; year >= start; year-- {
		date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		if date.Month() != month {
			return nil, ErrInvalidDate
		}
		ids, err := database.Values[uuid.UUID](ctx, repo.db, `SELECT a.id FROM assets a
			JOIN asset_job_status js ON js.asset_id = a.id
			WHERE date(a.local_date_time) = $1 AND a.owner_id = $2 AND a.visibility = 'timeline'
				AND EXISTS (SELECT 1 FROM asset_files f WHERE f.asset_id = a.id AND f.type = 'preview')
				AND a.deleted_at IS NULL
			LIMIT 20`, date.Format(time.DateOnly), ownerID)
		if err != nil {
			return nil, err
		}
		if len(ids) > 0 {
			out = append(out, YearAssets{Year: year, Assets: ids})
		}
	}

	return out, nil
}

var ErrInvalidDate = errors.New("date field value out of range")
