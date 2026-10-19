package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/models/user"
	"xn--gckvb8fzb.com/glides/services/database"
	"xn--gckvb8fzb.com/glides/services/database/sqlite"
)

func fileRuntimeConfig(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	cfg := testConfig(dir, harnessOptions{publicUsers: true}) + "\n[Database]\nPath = " + `"` + filepath.Join(dir, "korda.db") + `"` + "\n"
	path := filepath.Join(dir, "korda.toml")
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}

	return "file://" + path
}

func usersOf(t *testing.T, cfgstr string) []user.User {
	t.Helper()

	rt, err := newRuntime(cfgstr, sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err = prepare(rt, nil); err != nil {
		t.Fatal(err)
	}
	if err = database.From(rt).Startup(); err != nil {
		t.Fatal(err)
	}
	defer database.From(rt).Shutdown()

	list, err := gh.Repositories(rt).User.List(context.Background(), nullUUID(), false)
	if err != nil {
		t.Fatal(err)
	}

	return list
}

func runCreateUser(t *testing.T, cfgstr string, input string) error {
	t.Helper()

	rt, err := newRuntime(cfgstr, sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}

	return createUser(rt, input)
}

func TestCreateUser(t *testing.T) {
	cfgstr := fileRuntimeConfig(t)

	if err := runCreateUser(t, cfgstr, `{"email":"vera@example.com","name":"Vera","password":"vera-password"}`); !errors.Is(err, errs.ErrFirstUserMustBeAdmin) {
		t.Errorf("a first user without admin = %v", err)
	}
	if err := runCreateUser(t, cfgstr, `{"email":"vera@example.com"`); !errors.Is(err, errs.ErrCreateUserInvalid) {
		t.Errorf("broken JSON = %v", err)
	}
	if err := runCreateUser(t, cfgstr, `{"email":"vera@example.com","name":"Vera"}`); !errors.Is(err, errs.ErrCreateUserInvalid) {
		t.Errorf("no password = %v", err)
	}

	if err := runCreateUser(t, cfgstr, `{"email":"admin@example.com","name":"Admin","password":"admin-password","admin":true}`); err != nil {
		t.Fatal(err)
	}
	if err := runCreateUser(t, cfgstr, `{"email":"second@example.com","name":"Second","password":"second-password","admin":true}`); err != nil {
		t.Fatal(err)
	}
	if err := runCreateUser(t, cfgstr, `{"email":"vera@example.com","name":"Vera","password":"vera-password"}`); err != nil {
		t.Fatal(err)
	}
	if err := runCreateUser(t, cfgstr, `{"email":"vera@example.com","name":"Vera","password":"vera-password"}`); !errors.Is(err, errs.ErrEmailNotAvailable) {
		t.Errorf("a second vera = %v", err)
	}

	list := usersOf(t, cfgstr)
	if len(list) != 3 {
		t.Fatalf("users = %d, want 3", len(list))
	}

	byEmail := map[string]user.User{}
	for _, u := range list {
		byEmail[u.Email] = u
	}
	if u := byEmail["admin@example.com"]; !u.IsAdmin || u.StorageLabel.V != "admin" {
		t.Errorf("the first admin = %+v", u)
	}
	if u := byEmail["second@example.com"]; !u.IsAdmin || u.StorageLabel.Valid {
		t.Errorf("the second admin = %+v", u)
	}
	if u := byEmail["vera@example.com"]; u.IsAdmin || !u.HasPassword() {
		t.Errorf("vera = %+v", u)
	}
}

func TestMigrate(t *testing.T) {
	cfgstr := fileRuntimeConfig(t)

	rt, err := newRuntime(cfgstr, sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err = migrate(rt); err != nil {
		t.Fatal(err)
	}

	if list := usersOf(t, cfgstr); len(list) != 0 {
		t.Errorf("a migrated database has %d users", len(list))
	}
}
