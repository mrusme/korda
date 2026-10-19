package targets

import (
	"context"
	"fmt"

	"tty.fail/mrus/korda/errs"
	logicadminusers "tty.fail/mrus/korda/logic/root/admin/users"
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

func Install(rt *runtime.Runtime, h *handler.Handlers) {
	h.Handle(jobs.DeleteUser, jobs.Run, func(ctx context.Context, j asyncjob.AsyncJob) error {
		payloads, err := asyncjob.Payloads[jobs.DeleteUserPayload](j)
		if err != nil {
			return err
		}

		for _, p := range payloads {
			if err = logicadminusers.Purge(ctx, rt, p.UserID, p.Force); err != nil {
				return err
			}
		}

		return nil
	})
}

func Register(rt *runtime.Runtime) gtargets.Register {
	return func(id string, t gtargets.ITarget) error {
		if target, ok := t.(*handler.Target); ok && id == jobs.TargetID {
			Install(rt, &target.Handlers)
		}

		return nil
	}
}
