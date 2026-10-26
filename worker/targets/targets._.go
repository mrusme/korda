package targets

import (
	"context"
	"fmt"

	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	logicadminusers "tty.fail/mrus/korda/logic/root/admin/users"
	logicassets "tty.fail/mrus/korda/logic/root/assets"
	logicmemories "tty.fail/mrus/korda/logic/root/memories"
	logicnotifications "tty.fail/mrus/korda/logic/root/notifications"
	logictags "tty.fail/mrus/korda/logic/root/tags"
	"tty.fail/mrus/korda/services/jobs"
	"xn--gckvb8fzb.com/glides/models/asyncjob"
	"xn--gckvb8fzb.com/glides/runtime"
	"xn--gckvb8fzb.com/glides/services/config"
	gtargets "xn--gckvb8fzb.com/glides/worker/targets"
	"xn--gckvb8fzb.com/glides/worker/targets/handler"
)

func Check(cfg *config.Config) error {
	defs, err := cfg.Targets()
	if err != nil {
		return err
	}

	for _, def := range defs {
		if def.ID == jobs.TargetID && def.Type == config.TargetTypeHandler {
			return nil
		}
	}

	return fmt.Errorf("%w: a [[Target]] with ID %q and Type %q", errs.ErrJobsTargetMissing, jobs.TargetID, config.TargetTypeHandler)
}

func each[P any](fn func(ctx context.Context, p P) error) func(ctx context.Context, j asyncjob.AsyncJob) error {
	return func(ctx context.Context, j asyncjob.AsyncJob) error {
		payloads, err := asyncjob.Payloads[P](j)
		if err != nil {
			return err
		}

		for _, p := range payloads {
			if err = fn(ctx, p); err != nil {
				return err
			}
		}

		return nil
	}
}

func limited[P any](sem chan struct{}, fn func(ctx context.Context, p P) error) func(ctx context.Context, p P) error {
	return func(ctx context.Context, p P) error {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			return ctx.Err()
		}
		defer func() { <-sem }()

		return fn(ctx, p)
	}
}

func Install(rt *runtime.Runtime, h *handler.Handlers) {
	concurrency := 1
	if cfg, err := gh.MediaConfigFrom(rt.Config()); err == nil && cfg.Concurrency > 0 {
		concurrency = cfg.Concurrency
	}
	sem := make(chan struct{}, concurrency)

	h.Handle(jobs.DeleteUser, jobs.Run, each(func(ctx context.Context, p jobs.DeleteUserPayload) error {
		return logicadminusers.Purge(ctx, rt, p.UserID, p.Force)
	}))

	h.Handle(jobs.Metadata, jobs.Run, each(limited(sem, func(ctx context.Context, p jobs.AssetPayload) error {
		return logicassets.Extract(ctx, rt, p)
	})))

	h.Handle(jobs.Thumbnails, jobs.Run, each(limited(sem, func(ctx context.Context, p jobs.AssetPayload) error {
		return logicassets.GenerateThumbnails(ctx, rt, p)
	})))

	h.Handle(jobs.Transcode, jobs.Run, each(limited(sem, func(ctx context.Context, p jobs.AssetPayload) error {
		return logicassets.Transcode(ctx, rt, p)
	})))

	h.Handle(jobs.Edit, jobs.Run, each(limited(sem, func(ctx context.Context, p jobs.AssetPayload) error {
		return logicassets.RenderEdits(ctx, rt, p)
	})))

	h.Handle(jobs.DeleteAsset, jobs.Run, each(func(ctx context.Context, p jobs.DeleteAssetPayload) error {
		return logicassets.Purge(ctx, rt, p.AssetID, p.DeleteOnDisk)
	}))

	h.Handle(jobs.NotifyAlbumInvite, jobs.Run, each(func(ctx context.Context, p jobs.AlbumInvitePayload) error {
		return logicnotifications.AlbumInvite(ctx, rt, p.AlbumID, p.RecipientID, p.SenderName)
	}))

	h.Handle(jobs.NotifyAlbumUpdate, jobs.Run, each(func(ctx context.Context, p jobs.AlbumUpdatePayload) error {
		return logicnotifications.AlbumUpdate(ctx, rt, p.AlbumID, p.RecipientID)
	}))

	for _, jobType := range []asyncjob.JobType{jobs.ThumbnailsAll, jobs.MetadataAll, jobs.TranscodeAll} {
		h.Handle(jobType, jobs.Run, each(func(ctx context.Context, p jobs.ForcePayload) error {
			return logicassets.QueueAll(ctx, rt, jobType, p.Force != nil && *p.Force)
		}))
	}

	manual := map[asyncjob.JobType]func(ctx context.Context, rt *runtime.Runtime) error{
		jobs.UserDeleteCheck: logicadminusers.QueueDueDeletions,
		jobs.MemoryCleanup:   logicmemories.Cleanup,
		jobs.MemoryGenerate:  logicmemories.Generate,
		jobs.TagCleanup:      logictags.Cleanup,
	}
	for jobType, run := range manual {
		h.Handle(jobType, jobs.Run, each(func(ctx context.Context, _ jobs.ManualPayload) error {
			return run(ctx, rt)
		}))
	}

	h.Handle(jobs.EmptyTrash, jobs.Run, each(func(ctx context.Context, _ jobs.EmptyTrashPayload) error {
		return logicassets.QueueDeleted(ctx, rt)
	}))
}

func Register(rt *runtime.Runtime) gtargets.Register {
	return func(id string, t gtargets.ITarget) error {
		if target, ok := t.(*handler.Target); ok && id == jobs.TargetID {
			Install(rt, &target.Handlers)
		}

		return nil
	}
}
