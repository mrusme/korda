package server

import (
	"context"

	gh "tty.fail/mrus/korda/helpers"
	repouser "tty.fail/mrus/korda/services/repositories/user"
	"xn--gckvb8fzb.com/glides/runtime"
)

type Statistics struct {
	Photos      int64
	Videos      int64
	Usage       int64
	UsagePhotos int64
	UsageVideos int64
	Users       []repouser.Statistics
}

func GetStatistics(ctx context.Context, rt *runtime.Runtime) (*Statistics, error) {
	users, err := gh.Repositories(rt).User.Statistics(ctx)
	if err != nil {
		return nil, err
	}

	stats := &Statistics{Users: users}
	for _, u := range users {
		stats.Photos += u.Photos
		stats.Videos += u.Videos
		stats.Usage += u.Usage
		stats.UsagePhotos += u.UsagePhotos
		stats.UsageVideos += u.UsageVideos
	}

	return stats, nil
}
