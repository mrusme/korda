package helpers

import (
	"tty.fail/mrus/korda/services/jobs"
	"tty.fail/mrus/korda/services/media"
	"tty.fail/mrus/korda/services/mediaroot"
	"tty.fail/mrus/korda/services/repositories"
	"xn--gckvb8fzb.com/glides/runtime"
)

func Repositories(rt *runtime.Runtime) *repositories.Repositories {
	if srv, _ := rt.GetService("repositories"); srv != nil {
		return srv.(*repositories.Repositories)
	}
	return nil
}

func MediaRoot(rt *runtime.Runtime) *mediaroot.MediaRoot {
	if srv, _ := rt.GetService("mediaroot"); srv != nil {
		return srv.(*mediaroot.MediaRoot)
	}
	return nil
}

func Media(rt *runtime.Runtime) *media.Media {
	if srv, _ := rt.GetService("media"); srv != nil {
		return srv.(*media.Media)
	}
	return nil
}

func Jobs(rt *runtime.Runtime) *jobs.Jobs {
	if srv, _ := rt.GetService("jobs"); srv != nil {
		return srv.(*jobs.Jobs)
	}
	return nil
}
