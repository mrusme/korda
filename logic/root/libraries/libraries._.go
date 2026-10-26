package libraries

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"xn--gckvb8fzb.com/glides/runtime"
)

type CreateInput struct {
	OwnerID           string   `json:"ownerId" v:"uuid"`
	Name              *string  `json:"name" v:"min=1"`
	ImportPaths       []string `json:"importPaths" v:"optional,max=128,refine=nonEmptyItems|uniqueItems"`
	ExclusionPatterns []string `json:"exclusionPatterns" v:"optional,max=128,refine=nonEmptyItems|uniqueItems"`
}

type UpdateInput struct {
	Name              *string  `json:"name" v:"min=1"`
	ImportPaths       []string `json:"importPaths" v:"optional,max=128,refine=nonEmptyItems|uniqueItems"`
	ExclusionPatterns []string `json:"exclusionPatterns" v:"optional,max=128,refine=nonEmptyItems|uniqueItems"`
}

type ValidateInput struct {
	ImportPaths       []string `json:"importPaths" v:"optional,max=128,refine=nonEmptyItems|uniqueItems"`
	ExclusionPatterns []string `json:"exclusionPatterns" v:"optional,max=128,refine=nonEmptyItems|uniqueItems"`
}

type PathResult struct {
	ImportPath string
	IsValid    bool
	Message    string
}

func Validate(rt *runtime.Runtime, in *ValidateInput) []PathResult {
	out := make([]PathResult, 0, len(in.ImportPaths))
	for _, p := range in.ImportPaths {
		out = append(out, validatePath(rt, p))
	}

	return out
}

func inside(path string, root string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	return strings.HasPrefix(strings.TrimSuffix(abs, "/")+"/", strings.TrimSuffix(root, "/")+"/")
}

func validatePath(rt *runtime.Runtime, p string) PathResult {
	r := PathResult{ImportPath: p}
	if inside(p, gh.MediaRoot(rt).Path()) {
		r.Message = "Cannot use media upload folder for external libraries"
		return r
	}
	if !filepath.IsAbs(p) {
		abs, _ := filepath.Abs(p)
		r.Message = "Import path must be absolute, try " + abs
		return r
	}

	st, err := os.Stat(p)
	switch {
	case errors.Is(err, os.ErrNotExist):
		r.Message = "Path does not exist (ENOENT)"
		return r
	case err != nil:
		r.Message = err.Error()
		return r
	case !st.IsDir():
		r.Message = "Not a directory"
		return r
	}

	if unix.Access(p, unix.R_OK) != nil {
		r.Message = "Lacking read permission for folder"
		return r
	}

	r.IsValid = true
	return r
}

func Create(in *CreateInput) error {
	return errs.UnsupportedLibraries
}
