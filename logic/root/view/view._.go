package view

import (
	"context"
	"errors"
	"sort"
	"strings"

	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/models/asset"
	"tty.fail/mrus/korda/models/auth"
	"xn--gckvb8fzb.com/glides/runtime"
)

var ErrPathMissing = errors.New("path is required")

func UniquePaths(ctx context.Context, rt *runtime.Runtime, a *auth.Auth) ([]string, error) {
	dirs, err := gh.Repositories(rt).Asset.UniqueOriginalDirectories(ctx, a.User.ID)
	if err != nil {
		return nil, err
	}

	sort.Strings(dirs)
	out := make([]string, 0, len(dirs))
	for _, d := range dirs {
		out = append(out, strings.TrimSuffix(d, "/"))
	}

	return out, nil
}

func ByPath(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, path []string) ([]asset.Asset, error) {
	if len(path) != 1 {
		return nil, ErrPathMissing
	}

	return gh.Repositories(rt).Asset.ByOriginalDirectory(ctx, a.User.ID, strings.TrimSuffix(path[0], "/"))
}
