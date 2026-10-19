package mediaroot

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
)

func TestNestedIsImmichsLayout(t *testing.T) {
	root, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	userID := uuid.MustParse("b45e6023-14e8-446b-bbda-ace745673946")
	if got := root.Nested(Upload, userID, "9d803ee8-069e-4c06.jpg"); got != "upload/b45e6023-14e8-446b-bbda-ace745673946/9d/80/9d803ee8-069e-4c06.jpg" {
		t.Errorf("Nested = %s", got)
	}
	if got := root.UserFolder(Profile, userID); got != "profile/b45e6023-14e8-446b-bbda-ace745673946" {
		t.Errorf("UserFolder = %s", got)
	}
}

func TestAbsKeepsEveryPathInsideTheRoot(t *testing.T) {
	root, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	for _, escape := range []string{"../etc/passwd", "/etc/passwd", "..", "", "profile/../../x"} {
		if _, err = root.Abs(escape); !errors.Is(err, errs.ErrMediaRootInvalid) {
			t.Errorf("Abs(%q) = %v, want %v", escape, err, errs.ErrMediaRootInvalid)
		}
	}

	abs, err := root.Abs("profile/a/b.jpg")
	if err != nil || abs != filepath.Join(root.Path(), "profile", "a", "b.jpg") {
		t.Errorf("Abs = %s, %v", abs, err)
	}
}

func TestEnsureRemoveAndUsage(t *testing.T) {
	root, err := New(filepath.Join(t.TempDir(), "media"))
	if err != nil {
		t.Fatal(err)
	}
	if err = root.Startup(); err != nil {
		t.Fatal(err)
	}

	rel := "profile/u/x.jpg"
	if err = root.Ensure(rel); err != nil {
		t.Fatal(err)
	}
	abs, _ := root.Abs(rel)
	if err = os.WriteFile(abs, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err = root.Remove(rel, "profile/u/missing.jpg", ""); err != nil {
		t.Errorf("Remove = %v, want a missing file to be no error", err)
	}
	if _, err = os.Stat(abs); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the file is still there: %v", err)
	}

	u, err := root.Usage()
	if err != nil || u.Total <= 0 || u.Free > u.Total || u.Available > u.Free {
		t.Errorf("Usage = %+v, %v", u, err)
	}
}
