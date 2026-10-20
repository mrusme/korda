package asset

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/helpers/checksum"
	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/models/search"
	"xn--gckvb8fzb.com/glides/services/database"
)

type builder struct {
	joins []string
	where []string
	args  []any
	exif  bool
}

func (b *builder) arg(v any) string {
	b.args = append(b.args, v)
	return "$" + strconv.Itoa(len(b.args))
}

func (b *builder) list(values []any) string {
	marks := make([]string, 0, len(values))
	for _, v := range values {
		marks = append(marks, b.arg(v))
	}

	return "(" + strings.Join(marks, ", ") + ")"
}

func (b *builder) innerExif() {
	if !b.exif {
		b.joins = append(b.joins, `INNER JOIN asset_exif e ON e.asset_id = a.id`)
		b.exif = true
	}
}

func (b *builder) sql() string {
	query := strings.Join(b.joins, " ")
	if len(b.where) > 0 {
		query += " WHERE " + strings.Join(b.where, " AND ")
	}

	return query
}

func anyList[T any](values []T) []any {
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = v
	}

	return out
}

func (b *builder) contains(column string, value string) string {
	return `korda_fold(` + column + `) LIKE '%' || korda_fold(` + b.arg(value) + `) || '%' ESCAPE '\'`
}

type Flat struct {
	UserIDs          []uuid.UUID
	Visibility       string
	AlbumIDs         []uuid.UUID
	TagIDs           []uuid.UUID
	NoTags           bool
	PersonIDs        []uuid.UUID
	CreatedBefore    *time.Time
	CreatedAfter     *time.Time
	UpdatedBefore    *time.Time
	UpdatedAfter     *time.Time
	TrashedBefore    *time.Time
	TrashedAfter     *time.Time
	TakenBefore      *time.Time
	TakenAfter       *time.Time
	City             optional.Value[string]
	State            optional.Value[string]
	Country          optional.Value[string]
	Make             optional.Value[string]
	Model            optional.Value[string]
	LensModel        optional.Value[string]
	Rating           optional.Value[int64]
	Checksum         []byte
	ID               *uuid.UUID
	LibraryID        *uuid.UUID
	EncodedVideoPath string
	OriginalPath     string
	OriginalFileName string
	Description      string
	OCR              string
	Type             *asset.Type
	IsFavorite       *bool
	IsOffline        *bool
	IsEncoded        *bool
	IsMotion         *bool
	IsNotInAlbum     bool
	WithStacked      *bool
	WithDeleted      bool
	WithExif         bool
}

func (f Flat) build(b *builder) {
	withDeleted := f.WithDeleted || f.TrashedAfter != nil || f.TrashedBefore != nil || (f.IsOffline != nil && *f.IsOffline)

	switch f.Visibility {
	case "":
	case "not-locked":
		b.where = append(b.where, `a.visibility != 'locked'`)
	default:
		b.where = append(b.where, `a.visibility = `+b.arg(f.Visibility))
	}

	if len(f.PersonIDs) > 0 || f.OCR != "" {
		b.where = append(b.where, `0 = 1`)
	}
	if len(f.AlbumIDs) > 0 {
		b.where = append(b.where, inAllAlbums(b, f.AlbumIDs))
	} else if f.IsNotInAlbum {
		b.where = append(b.where, `NOT EXISTS (SELECT 1 FROM album_assets x WHERE x.asset_id = a.id)`)
	}
	if len(f.TagIDs) > 0 {
		b.where = append(b.where, `a.id IN (`+tagClosure(b, f.TagIDs)+`
			SELECT ta.asset_id FROM tag_assets ta JOIN closure c ON c.descendant = ta.tag_id
			GROUP BY ta.asset_id HAVING count(DISTINCT c.ancestor) >= `+strconv.Itoa(len(f.TagIDs))+`)`)
	}
	if f.NoTags {
		b.where = append(b.where, `NOT EXISTS (SELECT 1 FROM tag_assets x WHERE x.asset_id = a.id)`)
	}

	for _, c := range []struct {
		column string
		op     string
		value  *time.Time
	}{
		{"a.created_at", "<=", f.CreatedBefore}, {"a.created_at", ">=", f.CreatedAfter},
		{"a.updated_at", "<=", f.UpdatedBefore}, {"a.updated_at", ">=", f.UpdatedAfter},
		{"a.deleted_at", "<=", f.TrashedBefore}, {"a.deleted_at", ">=", f.TrashedAfter},
		{"a.file_created_at", "<=", f.TakenBefore}, {"a.file_created_at", ">=", f.TakenAfter},
	} {
		if c.value != nil {
			b.where = append(b.where, c.column+" "+c.op+" "+b.arg(c.value.UTC()))
		}
	}

	for _, c := range []struct {
		column string
		value  optional.Value[string]
	}{
		{"e.city", f.City}, {"e.state", f.State}, {"e.country", f.Country},
		{"e.make", f.Make}, {"e.model", f.Model}, {"e.lens_model", f.LensModel},
	} {
		if !c.value.Set {
			continue
		}
		b.innerExif()
		if c.value.Null {
			b.where = append(b.where, c.column+" IS NULL")
		} else {
			b.where = append(b.where, c.column+" = "+b.arg(c.value.V))
		}
	}
	if f.Rating.Set {
		b.innerExif()
		if f.Rating.Null {
			b.where = append(b.where, "e.rating IS NULL")
		} else {
			b.where = append(b.where, "e.rating = "+b.arg(f.Rating.V))
		}
	}

	if len(f.Checksum) > 0 {
		b.where = append(b.where, `a.checksum = `+b.arg(f.Checksum))
	}
	if f.ID != nil {
		b.where = append(b.where, `a.id = `+b.arg(*f.ID))
	}
	if f.LibraryID != nil {
		b.where = append(b.where, `a.library_id = `+b.arg(*f.LibraryID))
	}
	if f.UserIDs != nil {
		b.where = append(b.where, `a.owner_id IN `+b.list(anyList(f.UserIDs)))
	}
	if f.EncodedVideoPath != "" {
		b.joins = append(b.joins, `INNER JOIN asset_files vf ON vf.asset_id = a.id AND vf.type = 'encoded_video' AND vf.is_edited = 0`)
		b.where = append(b.where, `vf.path = `+b.arg(f.EncodedVideoPath))
	}
	if f.OriginalPath != "" {
		b.where = append(b.where, b.contains("a.original_path", f.OriginalPath))
	}
	if f.OriginalFileName != "" {
		b.where = append(b.where, b.contains("a.original_file_name", f.OriginalFileName))
	}
	if f.Description != "" {
		b.innerExif()
		b.where = append(b.where, b.contains("e.description", f.Description))
	}
	if f.Type != nil {
		b.where = append(b.where, `a.type = `+b.arg(*f.Type))
	}
	if f.IsFavorite != nil {
		b.where = append(b.where, `a.is_favorite = `+b.arg(*f.IsFavorite))
	}
	if f.IsOffline != nil {
		b.where = append(b.where, `a.is_offline = `+b.arg(*f.IsOffline))
	}
	if f.IsEncoded != nil {
		exists := `EXISTS (SELECT 1 FROM asset_files ef WHERE ef.asset_id = a.id AND ef.type = 'encoded_video')`
		if !*f.IsEncoded {
			exists = "NOT " + exists
		}
		b.where = append(b.where, exists)
	}
	if f.IsMotion != nil {
		if *f.IsMotion {
			b.where = append(b.where, `a.live_photo_video_id IS NOT NULL`)
		} else {
			b.where = append(b.where, `a.live_photo_video_id IS NULL`)
		}
	}
	if f.WithStacked != nil && !*f.WithStacked {
		b.where = append(b.where, `a.stack_id IS NULL`)
	}
	if f.WithExif {
		b.innerExif()
	}
	if !withDeleted {
		b.where = append(b.where, `a.deleted_at IS NULL`)
	}
}

func (repo *Repository) SearchFlat(ctx context.Context, f Flat, desc bool, limit int, offset int) ([]asset.Asset, error) {
	b := &builder{}
	f.build(b)

	dir := "ASC"
	if desc {
		dir = "DESC"
	}

	return database.Rows[asset.Asset](ctx, repo.db, `SELECT `+prefixed("a")+` FROM assets a `+b.sql()+`
		ORDER BY a.file_created_at `+dir+`, a.id `+dir+` LIMIT `+strconv.Itoa(limit)+` OFFSET `+strconv.Itoa(offset),
		b.args...,
	)
}

func (repo *Repository) CountFlat(ctx context.Context, f Flat) (int64, error) {
	b := &builder{}
	f.build(b)

	return database.Value[int64](ctx, repo.db, `SELECT COUNT(*) FROM assets a `+b.sql(), b.args...)
}

func (repo *Repository) RandomFlat(ctx context.Context, f Flat, size int) ([]asset.Asset, error) {
	b := &builder{}
	f.build(b)

	return database.Rows[asset.Asset](ctx, repo.db, `SELECT `+prefixed("a")+` FROM assets a `+b.sql()+`
		ORDER BY random() LIMIT `+strconv.Itoa(size),
		b.args...,
	)
}

func (repo *Repository) LargeFlat(ctx context.Context, f Flat, size int, minFileSize int64) ([]asset.Asset, error) {
	b := &builder{}
	f.build(b)
	b.innerExif()
	if f.Visibility != string(asset.VisibilityHidden) {
		b.where = append(b.where, `a.visibility != 'hidden'`)
	}
	b.where = append(b.where, `e.file_size_in_byte > `+b.arg(minFileSize))

	return database.Rows[asset.Asset](ctx, repo.db, `SELECT `+prefixed("a")+` FROM assets a `+b.sql()+`
		ORDER BY e.file_size_in_byte DESC LIMIT `+strconv.Itoa(size),
		b.args...,
	)
}

type Scope struct {
	UserIDs []uuid.UUID
	OwnerID uuid.UUID
}

func comparison[T any](b *builder, preds *[]string, column string, eq optional.Value[T], ne optional.Value[T],
	lt, lte, gt, gte *T, in []T, notIn []T,
) {
	if eq.Set {
		if eq.Null {
			*preds = append(*preds, column+" IS NULL")
		} else {
			*preds = append(*preds, column+" = "+b.arg(eq.V))
		}
	}
	if ne.Set {
		if ne.Null {
			*preds = append(*preds, column+" IS NOT NULL")
		} else {
			*preds = append(*preds, column+" != "+b.arg(ne.V))
		}
	}
	for _, c := range []struct {
		op    string
		value *T
	}{{"<", lt}, {"<=", lte}, {">", gt}, {">=", gte}} {
		if c.value != nil {
			*preds = append(*preds, column+" "+c.op+" "+b.arg(*c.value))
		}
	}
	if in != nil {
		*preds = append(*preds, column+" IN "+b.list(anyList(in)))
	}
	if notIn != nil {
		*preds = append(*preds, column+" NOT IN "+b.list(anyList(notIn)))
	}
}

func ptrOpt[T any](p *T) optional.Value[T] {
	if p == nil {
		return optional.Value[T]{}
	}

	return optional.Of(*p)
}

func utcOpt(v optional.Value[time.Time]) optional.Value[time.Time] {
	if v.Valid() {
		v.V = v.V.UTC()
	}

	return v
}

func utc(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()

	return &u
}

func stringFilter(b *builder, preds *[]string, column string, f *search.StringFilterNullable) {
	if f == nil {
		return
	}
	comparison[string](b, preds, column, f.Eq, f.Ne, nil, nil, nil, nil, f.In, f.NotIn)
}

func patternFilter(b *builder, preds *[]string, column string, f *search.StringPatternFilter) {
	if f == nil {
		return
	}
	comparison[string](b, preds, column, f.Eq, f.Ne, nil, nil, nil, nil, f.In, f.NotIn)
	folded := `korda_fold(` + column + `)`
	if f.Like != nil {
		*preds = append(*preds, folded+` LIKE '%' || korda_fold(`+b.arg(*f.Like)+`) || '%' ESCAPE '\'`)
	}
	if f.NotLike != nil {
		*preds = append(*preds, folded+` NOT LIKE '%' || korda_fold(`+b.arg(*f.NotLike)+`) || '%' ESCAPE '\'`)
	}
	if f.StartsWith != nil {
		*preds = append(*preds, folded+` LIKE korda_fold(`+b.arg(*f.StartsWith)+`) || '%' ESCAPE '\'`)
	}
	if f.EndsWith != nil {
		*preds = append(*preds, folded+` LIKE '%' || korda_fold(`+b.arg(*f.EndsWith)+`) ESCAPE '\'`)
	}
}

func exists(preds *[]string, f *search.BoolFilter, subquery string) {
	if f == nil {
		return
	}
	if f.Eq {
		*preds = append(*preds, "EXISTS ("+subquery+")")
	} else {
		*preds = append(*preds, "NOT EXISTS ("+subquery+")")
	}
}

func nothing(preds *[]string, f *search.BoolFilter) {
	if f != nil && f.Eq {
		*preds = append(*preds, "0 = 1")
	}
}

func membership(preds *[]string, f *search.IDsFilter) {
	if f != nil && (f.Any != nil || f.All != nil) {
		*preds = append(*preds, "0 = 1")
	}
}

func uuids(ids []string) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(ids))
	seen := map[uuid.UUID]bool{}
	for _, id := range ids {
		if parsed, err := uuid.Parse(id); err == nil && !seen[parsed] {
			seen[parsed] = true
			out = append(out, parsed)
		}
	}

	return out
}

func inAllAlbums(b *builder, ids []uuid.UUID) string {
	return `a.id IN (SELECT x.asset_id FROM album_assets x WHERE x.album_id IN ` + b.list(anyList(ids)) + `
		GROUP BY x.asset_id HAVING count(DISTINCT x.album_id) = ` + strconv.Itoa(len(ids)) + `)`
}

func tagClosure(b *builder, ids []uuid.UUID) string {
	return `WITH RECURSIVE closure(ancestor, descendant) AS (
		SELECT id, id FROM tags WHERE id IN ` + b.list(anyList(ids)) + `
		UNION SELECT c.ancestor, t.id FROM tags t JOIN closure c ON t.parent_id = c.descendant
	)`
}

func albumMembership(b *builder, preds *[]string, f *search.IDsFilter) {
	if f == nil {
		return
	}
	anyOf := func(ids []uuid.UUID) string {
		return `EXISTS (SELECT 1 FROM album_assets x WHERE x.asset_id = a.id AND x.album_id IN ` + b.list(anyList(ids)) + `)`
	}
	if f.Any != nil {
		*preds = append(*preds, anyOf(uuids(f.Any)))
	}
	if f.All != nil {
		ids := uuids(f.All)
		if len(ids) == 1 {
			*preds = append(*preds, anyOf(ids))
		} else {
			*preds = append(*preds, inAllAlbums(b, ids))
		}
	}
	if f.None != nil {
		*preds = append(*preds, "NOT "+anyOf(uuids(f.None)))
	}
}

func tagMembership(b *builder, preds *[]string, f *search.IDsFilter) {
	if f == nil {
		return
	}
	matching := func(ids []uuid.UUID, having string) string {
		return `a.id IN (` + tagClosure(b, ids) + `
			SELECT ta.asset_id FROM tag_assets ta JOIN closure c ON c.descendant = ta.tag_id` + having + `)`
	}
	if f.Any != nil {
		*preds = append(*preds, matching(uuids(f.Any), ""))
	}
	if f.All != nil {
		ids := uuids(f.All)
		having := ""
		if len(ids) > 1 {
			having = ` GROUP BY ta.asset_id HAVING count(DISTINCT c.ancestor) = ` + strconv.Itoa(len(ids))
		}
		*preds = append(*preds, matching(ids, having))
	}
	if f.None != nil {
		*preds = append(*preds, "NOT "+matching(uuids(f.None), ""))
	}
}

func checksums(values []string) []any {
	out := make([]any, 0, len(values))
	for _, v := range values {
		out = append(out, checksum.Decode(v))
	}

	return out
}

func branch(b *builder, c *search.Conditions) []string {
	var preds []string

	if c.ID != nil {
		comparison[string](b, &preds, "a.id", ptrOpt(c.ID.Eq), ptrOpt(c.ID.Ne), nil, nil, nil, nil, nil, nil)
	}
	if c.LibraryID != nil {
		comparison[string](b, &preds, "a.library_id", c.LibraryID.Eq, c.LibraryID.Ne, nil, nil, nil, nil, nil, nil)
	}
	if c.Type != nil {
		comparison[asset.Type](b, &preds, "a.type", ptrOpt(c.Type.Eq), ptrOpt(c.Type.Ne), nil, nil, nil, nil, c.Type.In, c.Type.NotIn)
	}
	if c.Visibility != nil {
		comparison[asset.Visibility](b, &preds, "a.visibility", ptrOpt(c.Visibility.Eq), ptrOpt(c.Visibility.Ne),
			nil, nil, nil, nil, c.Visibility.In, c.Visibility.NotIn)
	}
	if c.IsFavorite != nil {
		preds = append(preds, "a.is_favorite = "+b.arg(c.IsFavorite.Eq))
	}
	if c.IsOffline != nil {
		preds = append(preds, "a.is_offline = "+b.arg(c.IsOffline.Eq))
	}
	if c.IsMotion != nil {
		if c.IsMotion.Eq {
			preds = append(preds, "a.live_photo_video_id IS NOT NULL")
		} else {
			preds = append(preds, "a.live_photo_video_id IS NULL")
		}
	}
	exists(&preds, c.IsEncoded, `SELECT 1 FROM asset_files ef WHERE ef.asset_id = a.id AND ef.type = 'encoded_video'`)
	exists(&preds, c.HasAlbums, `SELECT 1 FROM album_assets x WHERE x.asset_id = a.id`)
	nothing(&preds, c.HasPeople)
	exists(&preds, c.HasTags, `SELECT 1 FROM tag_assets x WHERE x.asset_id = a.id`)

	stringFilter(b, &preds, "e.city", c.City)
	stringFilter(b, &preds, "e.state", c.State)
	stringFilter(b, &preds, "e.country", c.Country)
	stringFilter(b, &preds, "e.make", c.Make)
	stringFilter(b, &preds, "e.model", c.Model)
	stringFilter(b, &preds, "e.lens_model", c.LensModel)
	patternFilter(b, &preds, "e.description", c.Description)
	patternFilter(b, &preds, "a.original_file_name", c.OriginalFileName)
	patternFilter(b, &preds, "a.original_path", c.OriginalPath)
	if c.Ocr != nil {
		preds = append(preds, "0 = 1")
	}

	if r := c.Rating; r != nil {
		comparison[float64](b, &preds, "e.rating", r.Eq, r.Ne, r.Lt, r.Lte, r.Gt, r.Gte, r.In, r.NotIn)
	}
	if s := c.FileSizeInBytes; s != nil {
		comparison[float64](b, &preds, "e.file_size_in_byte", ptrOpt(s.Eq), ptrOpt(s.Ne), s.Lt, s.Lte, s.Gt, s.Gte, s.In, s.NotIn)
	}
	for _, d := range []struct {
		column string
		f      *search.DateFilter
	}{{"a.file_created_at", c.TakenAt}, {"a.created_at", c.CreatedAt}, {"a.updated_at", c.UpdatedAt}} {
		if d.f != nil {
			comparison[time.Time](b, &preds, d.column, ptrOpt(utc(d.f.Eq)), ptrOpt(utc(d.f.Ne)),
				utc(d.f.Lt), utc(d.f.Lte), utc(d.f.Gt), utc(d.f.Gte), nil, nil)
		}
	}
	if t := c.TrashedAt; t != nil {
		comparison[time.Time](b, &preds, "a.deleted_at", utcOpt(t.Eq), utcOpt(t.Ne), utc(t.Lt), utc(t.Lte), utc(t.Gt), utc(t.Gte), nil, nil)
	}

	albumMembership(b, &preds, c.AlbumIDs)
	membership(&preds, c.PersonIDs)
	tagMembership(b, &preds, c.TagIDs)

	if cs := c.Checksum; cs != nil {
		var eq, ne optional.Value[any]
		if cs.Eq != nil {
			eq = optional.Of[any](checksum.Decode(*cs.Eq))
		}
		if cs.Ne != nil {
			ne = optional.Of[any](checksum.Decode(*cs.Ne))
		}
		var in, notIn []any
		if cs.In != nil {
			in = checksums(cs.In)
		}
		if cs.NotIn != nil {
			notIn = checksums(cs.NotIn)
		}
		comparison[any](b, &preds, "a.checksum", eq, ne, nil, nil, nil, nil, in, notIn)
	}
	if v := c.EncodedVideoPath; v != nil {
		var inner []string
		comparison[string](b, &inner, "vf.path", ptrOpt(v.Eq), ptrOpt(v.Ne), nil, nil, nil, nil, v.In, v.NotIn)
		query := `SELECT 1 FROM asset_files vf WHERE vf.asset_id = a.id AND vf.type = 'encoded_video' AND vf.is_edited = 0`
		if len(inner) > 0 {
			query += " AND " + strings.Join(inner, " AND ")
		}
		preds = append(preds, "EXISTS ("+query+")")
	}

	return preds
}

func tree(b *builder, f *search.Filter, scope Scope, withStacked *bool) {
	b.joins = append(b.joins, `LEFT JOIN asset_exif e ON e.asset_id = a.id`)

	ownership := func() string {
		return "a.owner_id IN " + b.list(anyList(scope.UserIDs))
	}

	topConfined := f.AlbumConfined()
	anyBranchConfined := false
	for i := range f.Or {
		if f.Or[i].AlbumConfined() {
			anyBranchConfined = true
		}
	}
	perBranch := !topConfined && anyBranchConfined
	if !topConfined && !anyBranchConfined {
		b.where = append(b.where, ownership())
	}

	b.where = append(b.where, "(a.visibility != 'locked' OR a.owner_id = "+b.arg(scope.OwnerID)+")")
	if withStacked != nil && !*withStacked {
		b.where = append(b.where, "a.stack_id IS NULL")
	}

	preds := branch(b, &f.Conditions)
	if len(f.Or) > 0 {
		var ors []string
		for i := range f.Or {
			bp := branch(b, &f.Or[i].Conditions)
			if perBranch && !f.Or[i].AlbumConfined() {
				bp = append(bp, ownership())
			}
			if len(bp) == 0 {
				bp = []string{"1 = 1"}
			}
			ors = append(ors, "("+strings.Join(bp, " AND ")+")")
		}
		preds = append(preds, "("+strings.Join(ors, " OR ")+")")
	}
	b.where = append(b.where, preds...)
}

func orderBy(o *search.Order) string {
	field, direction := search.OrderFileCreatedAt, search.Desc
	if o != nil && o.Field != nil {
		field = *o.Field
	}
	if o != nil && o.Direction != nil {
		direction = *o.Direction
	}

	dir := "DESC"
	if direction == search.Asc {
		dir = "ASC"
	}

	column, nullable := "a.file_created_at", false
	switch field {
	case search.OrderLocalDateTime:
		column = "a.local_date_time"
	case search.OrderFileSizeInBytes:
		column, nullable = "e.file_size_in_byte", true
	case search.OrderRating:
		column, nullable = "e.rating", true
	}

	order := column + " " + dir
	if nullable {
		order += " NULLS LAST"
	}

	return order + ", a.id " + dir
}

func (repo *Repository) SearchTree(ctx context.Context, f *search.Filter, scope Scope, withStacked *bool, o *search.Order, limit int, offset int) ([]asset.Asset, error) {
	b := &builder{}
	tree(b, f, scope, withStacked)

	return database.Rows[asset.Asset](ctx, repo.db, `SELECT `+prefixed("a")+` FROM assets a `+b.sql()+`
		ORDER BY `+orderBy(o)+` LIMIT `+strconv.Itoa(limit)+` OFFSET `+strconv.Itoa(offset),
		b.args...,
	)
}

func (repo *Repository) CountTree(ctx context.Context, f *search.Filter, scope Scope) (int64, error) {
	b := &builder{}
	tree(b, f, scope, nil)

	return database.Value[int64](ctx, repo.db, `SELECT COUNT(*) FROM assets a `+b.sql(), b.args...)
}

func (repo *Repository) RandomTree(ctx context.Context, f *search.Filter, scope Scope, withStacked *bool, size int) ([]asset.Asset, error) {
	b := &builder{}
	tree(b, f, scope, withStacked)

	return database.Rows[asset.Asset](ctx, repo.db, `SELECT `+prefixed("a")+` FROM assets a `+b.sql()+`
		ORDER BY random() LIMIT `+strconv.Itoa(size),
		b.args...,
	)
}

func (repo *Repository) Suggestions(ctx context.Context, userIDs []uuid.UUID, column string, narrow map[string]string) ([]string, error) {
	b := &builder{}
	b.where = append(b.where,
		"a.owner_id IN "+b.list(anyList(userIDs)),
		"a.visibility = 'timeline'",
		"a.deleted_at IS NULL",
		"e."+column+" IS NOT NULL",
		"e."+column+" != ''",
	)
	for _, col := range []string{"country", "state", "make", "model", "lens_model"} {
		if value := narrow[col]; value != "" {
			b.where = append(b.where, "e."+col+" = "+b.arg(value))
		}
	}

	return database.Values[string](ctx, repo.db, `SELECT DISTINCT e.`+column+` FROM asset_exif e
		INNER JOIN assets a ON a.id = e.asset_id WHERE `+strings.Join(b.where, " AND "),
		b.args...,
	)
}

func (repo *Repository) RecentlyCreated(ctx context.Context, ownerID uuid.UUID, limit int) ([]asset.Asset, error) {
	return database.Rows[asset.Asset](ctx, repo.db, `SELECT `+columns+` FROM (
		SELECT `+columns+`, rowid AS physical FROM assets
		WHERE owner_id = $1 AND visibility = 'timeline' AND type = 'IMAGE' AND deleted_at IS NULL
		ORDER BY created_at DESC LIMIT `+strconv.Itoa(limit)+`) ORDER BY physical`,
		ownerID,
	)
}

func (repo *Repository) ByCity(ctx context.Context, userIDs []uuid.UUID) ([]asset.Asset, error) {
	b := &builder{}
	owners := b.list(anyList(userIDs))
	base := `FROM asset_exif e INNER JOIN assets a ON a.id = e.asset_id
		WHERE a.owner_id IN ` + owners + ` AND a.visibility = 'timeline' AND a.type = 'IMAGE' AND a.deleted_at IS NULL`

	cities, err := database.Values[string](ctx, repo.db, `SELECT DISTINCT e.city `+base+` AND e.city IS NOT NULL ORDER BY e.city`, b.args...)
	if err != nil {
		return nil, err
	}

	var out []asset.Asset
	if len(cities) == 0 {
		return database.Rows[asset.Asset](ctx, repo.db, `SELECT `+prefixed("a")+` `+base+` ORDER BY a.rowid LIMIT 1`, b.args...)
	}
	for _, city := range cities {
		args := append(append([]any{}, b.args...), city)
		found, err := database.Rows[asset.Asset](ctx, repo.db, `SELECT `+prefixed("a")+` `+base+`
			AND e.city = $`+strconv.Itoa(len(args))+` ORDER BY a.rowid LIMIT 1`, args...)
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}

	return out, nil
}
