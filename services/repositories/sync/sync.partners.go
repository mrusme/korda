package sync

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
	"xn--gckvb8fzb.com/glides/services/database"
)

type Partner struct {
	SharedByID   uuid.UUID
	SharedWithID uuid.UUID
	InTimeline   bool
	UpdateID     string
}

type PartnerDelete struct {
	ID           string
	SharedByID   uuid.UUID
	SharedWithID uuid.UUID
}

type Membership struct {
	ID       uuid.UUID
	CreateID string
}

type PartnerAsset Asset

type Stack struct {
	ID             uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
	PrimaryAssetID uuid.UUID
	OwnerID        uuid.UUID
	UpdateID       string
}

type StackDelete struct {
	ID      string
	StackID uuid.UUID
}

const partnerAssetColumns = `a.id, a.owner_id, a.original_file_name, a.thumbhash, a.checksum, a.file_created_at,
	a.file_modified_at, a.local_date_time, a.created_at, a.type, a.deleted_at, a.visibility, a.duration,
	a.live_photo_video_id, a.stack_id, a.library_id, a.width, a.height, a.is_edited, 0 AS is_favorite, a.update_id`

const sharedWith = `SELECT shared_by_id FROM partners WHERE shared_with_id = $`

func (repo *Repository) Partners(ctx context.Context, userID uuid.UUID, b Bounds) ([]Partner, error) {
	args := []any{}
	where := b.where("update_id", &args)
	args = append(args, userID)
	n := strconv.Itoa(len(args))

	return database.Rows[Partner](ctx, repo.db, `SELECT shared_by_id, shared_with_id, in_timeline, update_id
		FROM partners WHERE `+where+` AND (shared_by_id = $`+n+` OR shared_with_id = $`+n+`)`+b.tail("update_id"),
		args...,
	)
}

func (repo *Repository) PartnerDeletes(ctx context.Context, userID uuid.UUID, b Bounds) ([]PartnerDelete, error) {
	args := []any{}
	where := b.where("id", &args)
	args = append(args, userID)
	n := strconv.Itoa(len(args))

	return database.Rows[PartnerDelete](ctx, repo.db, `SELECT id, shared_by_id, shared_with_id FROM partner_audit
		WHERE `+where+` AND (shared_by_id = $`+n+` OR shared_with_id = $`+n+`)`+b.tail("id"),
		args...,
	)
}

func (repo *Repository) PartnersCreatedAfter(ctx context.Context, userID uuid.UUID, after *string, before string) ([]Membership, error) {
	args := []any{userID, before}
	query := `SELECT shared_by_id AS id, create_id FROM partners WHERE shared_with_id = $1 AND create_id < $2`
	if after != nil {
		args = append(args, *after)
		query += ` AND create_id >= $3`
	}

	return database.Rows[Membership](ctx, repo.db, query+` ORDER BY create_id`, args...)
}

func (repo *Repository) PartnerAssets(ctx context.Context, userID uuid.UUID, b Bounds) ([]PartnerAsset, error) {
	args := []any{}
	where := b.where("a.update_id", &args)
	args = append(args, userID)

	return database.Rows[PartnerAsset](ctx, repo.db, `SELECT `+partnerAssetColumns+` FROM assets a
		WHERE `+where+` AND a.owner_id IN (`+sharedWith+strconv.Itoa(len(args))+`)`+b.tail("a.update_id"),
		args...,
	)
}

func (repo *Repository) PartnerAssetsBackfill(ctx context.Context, ownerID uuid.UUID, b Backfill) ([]PartnerAsset, error) {
	args := []any{}
	where := b.where("a.update_id", &args)
	args = append(args, ownerID)

	return database.Rows[PartnerAsset](ctx, repo.db, `SELECT `+partnerAssetColumns+` FROM assets a
		WHERE `+where+` AND a.owner_id = $`+strconv.Itoa(len(args))+b.tail("a.update_id"),
		args...,
	)
}

func (repo *Repository) PartnerAssetDeletes(ctx context.Context, userID uuid.UUID, b Bounds) ([]AssetDelete, error) {
	args := []any{}
	where := b.where("id", &args)
	args = append(args, userID)

	return database.Rows[AssetDelete](ctx, repo.db, `SELECT id, asset_id FROM asset_audit
		WHERE `+where+` AND owner_id IN (`+sharedWith+strconv.Itoa(len(args))+`)`+b.tail("id"),
		args...,
	)
}

func (repo *Repository) PartnerAssetExifs(ctx context.Context, userID uuid.UUID, b Bounds) ([]AssetExif, error) {
	args := []any{}
	where := b.where("e.update_id", &args)
	args = append(args, userID)

	return database.Rows[AssetExif](ctx, repo.db, `SELECT `+exifColumns+` FROM asset_exif e
		WHERE `+where+` AND e.asset_id IN (SELECT id FROM assets WHERE owner_id IN (`+sharedWith+
		strconv.Itoa(len(args))+`))`+b.tail("e.update_id"),
		args...,
	)
}

func (repo *Repository) PartnerAssetExifsBackfill(ctx context.Context, ownerID uuid.UUID, b Backfill) ([]AssetExif, error) {
	args := []any{}
	where := b.where("e.update_id", &args)
	args = append(args, ownerID)

	return database.Rows[AssetExif](ctx, repo.db, `SELECT `+exifColumns+` FROM asset_exif e
		JOIN assets a ON a.id = e.asset_id
		WHERE `+where+` AND a.owner_id = $`+strconv.Itoa(len(args))+b.tail("e.update_id"),
		args...,
	)
}

const stackColumns = `s.id, s.created_at, s.updated_at, s.primary_asset_id, s.owner_id, s.update_id`

func (repo *Repository) Stacks(ctx context.Context, ownerID uuid.UUID, b Bounds) ([]Stack, error) {
	args := []any{}
	where := b.where("s.update_id", &args)
	args = append(args, ownerID)

	return database.Rows[Stack](ctx, repo.db, `SELECT `+stackColumns+` FROM stacks s
		WHERE `+where+` AND s.owner_id = $`+strconv.Itoa(len(args))+b.tail("s.update_id"),
		args...,
	)
}

func (repo *Repository) StackDeletes(ctx context.Context, ownerID uuid.UUID, b Bounds) ([]StackDelete, error) {
	args := []any{}
	where := b.where("id", &args)
	args = append(args, ownerID)

	return database.Rows[StackDelete](ctx, repo.db, `SELECT id, stack_id FROM stack_audit
		WHERE `+where+` AND user_id = $`+strconv.Itoa(len(args))+b.tail("id"),
		args...,
	)
}

func (repo *Repository) PartnerStacks(ctx context.Context, userID uuid.UUID, b Bounds) ([]Stack, error) {
	args := []any{}
	where := b.where("s.update_id", &args)
	args = append(args, userID)

	return database.Rows[Stack](ctx, repo.db, `SELECT `+stackColumns+` FROM stacks s
		WHERE `+where+` AND s.owner_id IN (`+sharedWith+strconv.Itoa(len(args))+`)`+b.tail("s.update_id"),
		args...,
	)
}

func (repo *Repository) PartnerStacksBackfill(ctx context.Context, ownerID uuid.UUID, b Backfill) ([]Stack, error) {
	args := []any{}
	where := b.where("s.update_id", &args)
	args = append(args, ownerID)

	return database.Rows[Stack](ctx, repo.db, `SELECT `+stackColumns+` FROM stacks s
		WHERE `+where+` AND s.owner_id = $`+strconv.Itoa(len(args))+b.tail("s.update_id"),
		args...,
	)
}

func (repo *Repository) PartnerStackDeletes(ctx context.Context, userID uuid.UUID, b Bounds) ([]StackDelete, error) {
	args := []any{}
	where := b.where("id", &args)
	args = append(args, userID)

	return database.Rows[StackDelete](ctx, repo.db, `SELECT id, stack_id FROM stack_audit
		WHERE `+where+` AND user_id IN (`+sharedWith+strconv.Itoa(len(args))+`)`+b.tail("id"),
		args...,
	)
}
