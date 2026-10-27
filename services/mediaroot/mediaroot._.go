package mediaroot

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
)

type Folder string

const (
	Upload       Folder = "upload"
	Profile      Folder = "profile"
	Thumbnails   Folder = "thumbs"
	EncodedVideo Folder = "encoded-video"
	Backups      Folder = "backups"
)

type MediaRoot struct {
	path string
}

type Usage struct {
	Total     int64
	Free      int64
	Available int64
}

func New(path string) (*MediaRoot, error) {
	if path == "" {
		return nil, errs.ErrMediaRootInvalid
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errs.ErrMediaRootInvalid, err)
	}

	return &MediaRoot{path: abs}, nil
}

func (m *MediaRoot) Startup() error {
	return os.MkdirAll(m.path, 0o750)
}

func (m *MediaRoot) Shutdown() error {
	return nil
}

func (m *MediaRoot) Path() string {
	return m.path
}

func (m *MediaRoot) Abs(relative string) (string, error) {
	cleaned := filepath.Clean(filepath.FromSlash(relative))
	if cleaned == "." || filepath.IsAbs(cleaned) || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) || cleaned == ".." {
		return "", fmt.Errorf("%w: %q", errs.ErrMediaRootInvalid, relative)
	}

	return filepath.Join(m.path, cleaned), nil
}

func (m *MediaRoot) UserFolder(folder Folder, userID uuid.UUID) string {
	return filepath.ToSlash(filepath.Join(string(folder), userID.String()))
}

func (m *MediaRoot) Nested(folder Folder, userID uuid.UUID, name string) string {
	parts := []string{string(folder), userID.String()}
	if len(name) >= 4 {
		parts = append(parts, name[0:2], name[2:4])
	}

	return filepath.ToSlash(filepath.Join(append(parts, name)...))
}

func (m *MediaRoot) Ensure(relative string) error {
	abs, err := m.Abs(relative)
	if err != nil {
		return err
	}

	return os.MkdirAll(filepath.Dir(abs), 0o750)
}

func (m *MediaRoot) Remove(relatives ...string) error {
	var failures []error
	for _, relative := range relatives {
		if relative == "" {
			continue
		}

		abs, err := m.Abs(relative)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		if err = os.Remove(abs); err != nil && !errors.Is(err, fs.ErrNotExist) {
			failures = append(failures, err)
		}
	}

	return errors.Join(failures...)
}

func (m *MediaRoot) RemoveAll(relative string) error {
	abs, err := m.Abs(relative)
	if err != nil {
		return err
	}

	return os.RemoveAll(abs)
}

func (m *MediaRoot) Usage() (Usage, error) {
	return diskUsage(m.path)
}
