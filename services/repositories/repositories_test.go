package repositories

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/migrations"
	"tty.fail/mrus/korda/models/apikey"
	"tty.fail/mrus/korda/models/permission"
	"tty.fail/mrus/korda/models/session"
	"tty.fail/mrus/korda/models/user"
	"tty.fail/mrus/korda/models/usermetadata"
	repouser "tty.fail/mrus/korda/services/repositories/user"
	"xn--gckvb8fzb.com/glides/services/database"
	"xn--gckvb8fzb.com/glides/services/database/sqlite"
)

func open(t *testing.T) (*Repositories, *database.Database) {
	t.Helper()

	db, err := sqlite.New(nil, sqlite.Options{
		InMemory:       true,
		Migrations:     &migrations.FS,
		MigrationsPath: migrations.Path,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Startup(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Shutdown() })

	repos, err := New(db)
	if err != nil {
		t.Fatal(err)
	}

	return repos, db
}

func newUser(email string, admin bool, now time.Time) *user.User {
	return &user.User{
		ID:                   uuid.New(),
		Email:                email,
		Password:             sql.Null[string]{V: "hash", Valid: true},
		Name:                 "Name",
		IsAdmin:              admin,
		ShouldChangePassword: true,
		ProfileChangedAt:     now,
		Status:               user.StatusActive,
		ClusterGroupID:       uuid.New(),
		CreatedAt:            now,
		UpdatedAt:            now,
		UpdateID:             uuid.Must(uuid.NewV7()),
	}
}

func TestUserRoundTripAndUniqueEmail(t *testing.T) {
	repos, _ := open(t)
	ctx := t.Context()
	now := time.Date(2026, 10, 20, 8, 0, 0, 123_000_000, time.UTC)

	created, err := repos.User.Create(ctx, newUser("admin@example.com", true, now))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !created.CreatedAt.Equal(now) || created.CreatedAt.Location() != time.UTC || created.AvatarColor.Valid || created.QuotaSizeInBytes.Valid {
		t.Errorf("created = %+v", created)
	}

	if _, err = repos.User.Create(ctx, newUser("admin@example.com", false, now)); !errors.Is(err, errs.ErrUniqueViolationOn) {
		t.Errorf("a second user with the email = %v, want %v", err, errs.ErrUniqueViolationOn)
	}

	if has, err := repos.User.HasAdmin(ctx); err != nil || !has {
		t.Errorf("HasAdmin = %v, %v", has, err)
	}

	color := user.ColorRed
	updated, err := repos.User.Update(ctx, created.ID, repouser.Update{
		Name:         optional.Of("Admin"),
		AvatarColor:  optional.Of(color),
		StorageLabel: optional.Value[string]{Set: true, Null: true},
	}, now.Add(time.Second))
	if err != nil || updated.Name != "Admin" || updated.AvatarColor.V != color || updated.StorageLabel.Valid || !updated.UpdatedAt.After(now) {
		t.Errorf("Update = %+v, %v", updated, err)
	}
	if updated.UpdateID == created.UpdateID {
		t.Error("Update kept the update id")
	}

	if _, err = repos.User.Get(ctx, uuid.New(), false); !errors.Is(err, errs.ErrNoRows) {
		t.Errorf("Get of a missing user = %v", err)
	}
}

func TestUserListOrderDeletionAndAudit(t *testing.T) {
	repos, db := open(t)
	ctx := t.Context()
	now := time.Now().UTC()

	first, _ := repos.User.Create(ctx, newUser("a@example.com", true, now))
	second, _ := repos.User.Create(ctx, newUser("b@example.com", false, now.Add(time.Millisecond)))

	list, err := repos.User.List(ctx, uuid.NullUUID{}, false)
	if err != nil || len(list) != 2 || list[0].ID != second.ID {
		t.Fatalf("List = %v, %v, want the newest first", list, err)
	}

	if _, err = repos.User.Update(ctx, second.ID, repouser.Update{
		Status:    optional.Of(user.StatusDeleted),
		DeletedAt: optional.Of(now),
	}, now); err != nil {
		t.Fatal(err)
	}
	if list, _ = repos.User.List(ctx, uuid.NullUUID{}, false); len(list) != 1 {
		t.Errorf("List without deleted = %d users", len(list))
	}
	if list, _ = repos.User.List(ctx, uuid.NullUUID{UUID: second.ID, Valid: true}, true); len(list) != 1 || list[0].Status != user.StatusDeleted {
		t.Errorf("List of the deleted user = %v", list)
	}
	if _, err = repos.User.Update(ctx, second.ID, repouser.Update{Name: optional.Of("x")}, now); !errors.Is(err, errs.ErrNoRows) {
		t.Errorf("Update of a deleted user = %v, want no rows", err)
	}

	restored, err := repos.User.Restore(ctx, second.ID, now)
	if err != nil || restored.DeletedAt.Valid || restored.Status != user.StatusActive {
		t.Errorf("Restore = %+v, %v", restored, err)
	}

	if err = repos.User.Delete(ctx, first.ID, now); err != nil {
		t.Fatal(err)
	}
	audits, err := database.Value[int64](ctx, db, `SELECT count(*) FROM user_audit WHERE user_id = $1`, first.ID)
	if err != nil || audits != 1 {
		t.Errorf("user_audit rows = %d, %v", audits, err)
	}
}

func TestSessionsTokenLookupExpiryAndCascade(t *testing.T) {
	repos, _ := open(t)
	ctx := t.Context()
	now := time.Now().UTC()

	owner, _ := repos.User.Create(ctx, newUser("s@example.com", true, now))
	s := &session.Session{
		ID: uuid.New(), Token: []byte("hash-1"), UserID: owner.ID,
		CreatedAt: now, UpdatedAt: now, UpdateID: uuid.Must(uuid.NewV7()),
	}
	if _, err := repos.Session.Create(ctx, s); err != nil {
		t.Fatal(err)
	}
	expired := *s
	expired.ID, expired.Token = uuid.New(), []byte("hash-2")
	expired.ExpiresAt = sql.Null[time.Time]{V: now.Add(-time.Second), Valid: true}
	if _, err := repos.Session.Create(ctx, &expired); err != nil {
		t.Fatal(err)
	}

	got, err := repos.Session.GetByToken(ctx, []byte("hash-1"), now)
	if err != nil || got.ID != s.ID || got.ParentID.Valid {
		t.Errorf("GetByToken = %+v, %v", got, err)
	}
	if _, err = repos.Session.GetByToken(ctx, []byte("hash-2"), now); !errors.Is(err, errs.ErrNoRows) {
		t.Errorf("an expired session = %v, want no rows", err)
	}
	if list, _ := repos.Session.ListByUser(ctx, owner.ID, now); len(list) != 1 {
		t.Errorf("ListByUser = %d sessions, want the one that is not expired", len(list))
	}

	if err = repos.User.Delete(ctx, owner.ID, now); err != nil {
		t.Fatal(err)
	}
	if _, err = repos.Session.Get(ctx, s.ID); !errors.Is(err, errs.ErrNoRows) {
		t.Errorf("a session outlived its user: %v", err)
	}
}

func TestAPIKeysAndMetadata(t *testing.T) {
	repos, db := open(t)
	ctx := context.Background()
	now := time.Now().UTC()

	owner, _ := repos.User.Create(ctx, newUser("k@example.com", true, now))
	key, err := repos.APIKey.Create(ctx, &apikey.APIKey{
		ID: uuid.New(), Name: "k", Key: []byte("secret-hash"), UserID: owner.ID,
		Permissions: permission.List{permission.UserRead},
		CreatedAt:   now, UpdatedAt: now, UpdateID: uuid.Must(uuid.NewV7()),
	})
	if err != nil || len(key.Permissions) != 1 || key.Permissions[0] != permission.UserRead {
		t.Fatalf("Create = %+v, %v", key, err)
	}

	name := "renamed"
	updated, err := repos.APIKey.Update(ctx, owner.ID, key.ID, &name, nil, nil, now.Add(time.Second))
	if err != nil || updated.Name != name || len(updated.Permissions) != 1 || string(updated.Key) != "secret-hash" {
		t.Errorf("Update of the name = %+v, %v", updated, err)
	}
	if _, err = repos.APIKey.Get(ctx, uuid.New(), key.ID); !errors.Is(err, errs.ErrNoRows) {
		t.Errorf("a key of another user = %v", err)
	}

	if err = repos.UserMetadata.Upsert(ctx, owner.ID, usermetadata.Onboarding, `{"isOnboarded":true}`, now); err != nil {
		t.Fatal(err)
	}
	if err = repos.UserMetadata.Upsert(ctx, owner.ID, usermetadata.Onboarding, `{"isOnboarded":false}`, now); err != nil {
		t.Fatal(err)
	}
	item, err := repos.UserMetadata.Get(ctx, owner.ID, usermetadata.Onboarding)
	if err != nil || item.Value != `{"isOnboarded":false}` {
		t.Errorf("Get = %+v, %v", item, err)
	}
	if err = repos.UserMetadata.Delete(ctx, owner.ID, usermetadata.Onboarding, now); err != nil {
		t.Fatal(err)
	}
	if err = repos.UserMetadata.Delete(ctx, owner.ID, usermetadata.Onboarding, now); err != nil {
		t.Fatal(err)
	}
	audits, _ := database.Value[int64](ctx, db, `SELECT count(*) FROM user_metadata_audit`)
	if audits != 1 {
		t.Errorf("user_metadata_audit rows = %d, want 1 for the one delete that removed a row", audits)
	}
}
