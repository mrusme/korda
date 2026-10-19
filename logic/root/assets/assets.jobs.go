package assets

import (
	"context"

	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/models/auth"
	"tty.fail/mrus/korda/models/permission"
	"tty.fail/mrus/korda/services/jobs"
	"xn--gckvb8fzb.com/glides/runtime"
)

func RunJobs(ctx context.Context, rt *runtime.Runtime, a *auth.Auth, in *JobsInput) error {
	ids := parseIDs(in.AssetIDs)
	if err := require(ctx, rt, a, permission.AssetUpdate, ids...); err != nil {
		return err
	}

	queue := gh.Jobs(rt)
	for _, id := range ids {
		var err error
		switch in.Name {
		case JobRefreshMetadata:
			err = queue.Metadata(jobs.AssetPayload{AssetID: id})
		case JobRegenerateThumbnail:
			err = queue.Thumbnails(jobs.AssetPayload{AssetID: id})
		case JobTranscodeVideo:
			err = queue.Transcode(jobs.AssetPayload{AssetID: id})
		}
		if err != nil {
			return err
		}
	}

	return nil
}
